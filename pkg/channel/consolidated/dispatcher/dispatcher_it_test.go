/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dispatcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/binding/transformer"
	protocolhttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/cloudevents/sdk-go/v2/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/eventing/pkg/channel/fanout"
	"knative.dev/eventing/pkg/kncloudevents"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/tracing"
	tracingconfig "knative.dev/pkg/tracing/config"

	"knative.dev/eventing-kafka/pkg/channel/consolidated/utils"
	"knative.dev/eventing-kafka/pkg/common/config"
	"knative.dev/eventing-kafka/pkg/common/constants"
)

// This dispatcher tests the full integration of the dispatcher code with Kafka.
// This test doesn't run on the CI because unit tests script doesn't start a Kafka cluster.
// Use it in emergency situations when you can't reproduce the e2e test failures and the failure might be
// in the dispatcher code.
// Start a kafka cluster with docker: docker run --rm --net=host -e ADV_HOST=localhost -e SAMPLEDATA=0 lensesio/fast-data-dev
// Keep also the port 8080 free for the MessageReceiver
func TestDispatcher(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skipf("This test can't run in CI")
	}

	ctx := context.TODO()

	logger, err := zap.NewDevelopment(zap.AddStacktrace(zap.WarnLevel))
	if err != nil {
		t.Fatal(err)
	}

	tracer, _ := tracing.SetupPublishingWithStaticConfig(logger.Sugar(), "localhost", &tracingconfig.Config{
		Backend:        tracingconfig.Zipkin,
		Debug:          true,
		SampleRate:     1.0,
		ZipkinEndpoint: "http://localhost:9411/api/v2/spans",
	})
	tracer.Shutdown(context.Background())

	// Configure connection arguments - to be done exactly once per process
	kncloudevents.ConfigureConnectionArgs(&kncloudevents.ConnectionArgs{
		MaxIdleConns:        constants.DefaultMaxIdleConns,
		MaxIdleConnsPerHost: constants.DefaultMaxIdleConnsPerHost,
	})

	dispatcherArgs := KafkaDispatcherArgs{
		Config:    &config.EventingKafkaConfig{},
		Brokers:   []string{"localhost:9092"},
		TopicFunc: utils.TopicName,
	}

	// Create the dispatcher. At this point, if Kafka is not up, this thing fails
	dispatcher, err := NewDispatcher(context.Background(), &dispatcherArgs, func(ref types.NamespacedName) {})
	if err != nil {
		t.Skipf("no dispatcher: %v", err)
	}

	// Start the dispatcher
	go func() {
		if err := dispatcher.Start(context.Background()); err != nil {
			t.Error(err)
		}
	}()

	time.Sleep(1 * time.Second)

	// We need a channelaproxy and channelbproxy for handling correctly the Host header
	channelAProxy := httptest.NewServer(createReverseProxy(t, "channela.svc"))
	defer channelAProxy.Close()
	channelBProxy := httptest.NewServer(createReverseProxy(t, "channelb.svc"))
	defer channelBProxy.Close()

	// Start a bunch of test servers to simulate the various services
	transformationsWg := sync.WaitGroup{}
	transformationsWg.Add(1)
	transformationsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer transformationsWg.Done()
		message := protocolhttp.NewMessageFromHttpRequest(r)
		defer message.Finish(nil)

		err := protocolhttp.WriteResponseWriter(context.Background(), message, 200, w, transformer.AddExtension("transformed", "true"))
		if err != nil {
			w.WriteHeader(500)
			t.Fatal(err)
		}
	}))
	defer transformationsServer.Close()

	receiverWg := sync.WaitGroup{}
	receiverWg.Add(1)
	receiverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer receiverWg.Done()
		transformed := r.Header.Get("ce-transformed")
		if transformed != "true" {
			w.WriteHeader(500)
			t.Fatalf("Expecting ce-transformed: true, found %s", transformed)
		}
	}))
	defer receiverServer.Close()

	transformationsFailureWg := sync.WaitGroup{}
	transformationsFailureWg.Add(1)
	transformationsFailureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer transformationsFailureWg.Done()
		w.WriteHeader(500)
	}))
	defer transformationsFailureServer.Close()

	deadLetterWg := sync.WaitGroup{}
	deadLetterWg.Add(1)
	deadLetterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer deadLetterWg.Done()
		transformed := r.Header.Get("ce-transformed")
		if transformed != "" {
			w.WriteHeader(500)
			t.Fatalf("Not expecting ce-transformed, found %s", transformed)
		}
	}))
	defer deadLetterServer.Close()

	logger.Debug("Test servers",
		zap.String("transformations server", transformationsServer.URL),
		zap.String("transformations failure server", transformationsFailureServer.URL),
		zap.String("receiver server", receiverServer.URL),
		zap.String("dead letter server", deadLetterServer.URL),
	)

	// send -> channela -> sub with transformationServer and reply to channelb -> channelb -> sub with receiver -> receiver
	channelAConfig := &ChannelConfig{
		Namespace: "default",
		Name:      "channela",
		HostName:  "channela.svc",
		Subscriptions: []Subscription{
			{
				UID: "aaaa",
				Subscription: fanout.Subscription{
					Subscriber: mustParseUrl(t, transformationsServer.URL),
					Reply:      mustParseUrl(t, channelBProxy.URL),
				},
			},
			{
				UID: "cccc",
				Subscription: fanout.Subscription{
					Subscriber: mustParseUrl(t, transformationsFailureServer.URL),
					Reply:      mustParseUrl(t, channelBProxy.URL),
					DeadLetter: mustParseUrl(t, deadLetterServer.URL),
				},
			},
		},
	}
	require.NoError(t, dispatcher.RegisterChannelHost(channelAConfig))
	require.NoError(t, dispatcher.ReconcileConsumers(ctx, channelAConfig))

	channelBConfig := &ChannelConfig{
		Namespace: "default",
		Name:      "channelb",
		HostName:  "channelb.svc",
		Subscriptions: []Subscription{
			{
				UID: "bbbb",
				Subscription: fanout.Subscription{
					Subscriber: mustParseUrl(t, receiverServer.URL),
				},
			},
		},
	}
	require.NoError(t, dispatcher.RegisterChannelHost(channelBConfig))
	require.NoError(t, dispatcher.ReconcileConsumers(ctx, channelBConfig))

	time.Sleep(5 * time.Second)

	// Ok now everything should be ready to send the event
	httpsender, err := kncloudevents.NewHTTPMessageSenderWithTarget(channelAProxy.URL)
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpsender.NewCloudEventRequest(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	event := test.FullEvent()
	_ = protocolhttp.WriteRequest(context.Background(), binding.ToMessage(&event), req)

	res, err := httpsender.Send(req)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 202 {
		t.Fatalf("Expected 202, Have %d", res.StatusCode)
	}

	transformationsFailureWg.Wait()
	deadLetterWg.Wait()
	transformationsWg.Wait()
	receiverWg.Wait()

	// Try to close consumer groups
	require.NoError(t, dispatcher.CleanupChannel("channela", "default", "channela.svc"))
	require.NoError(t, dispatcher.CleanupChannel("channelb", "default", "channelb.svc"))
}

func createReverseProxy(t *testing.T, host string) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		target := mustParseUrl(t, "http://localhost:8080")
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		req.Host = host
	}
	return &httputil.ReverseProxy{Director: director}
}

func mustParseUrl(t *testing.T, str string) *url.URL {
	url, err := apis.ParseURL(str)
	if err != nil {
		t.Fatal(err)
	}
	return url.URL()
}
