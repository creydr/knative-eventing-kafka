/*
Copyright 2021 The Knative Authors

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

package messaging

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	eventingduckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	messaging "knative.dev/eventing-kafka/pkg/apis/messaging/v1beta1"
)

var deploymentConditionReady = appsv1.DeploymentCondition{
	Type:   appsv1.DeploymentAvailable,
	Status: corev1.ConditionTrue,
}

var deploymentConditionNotReady = appsv1.DeploymentCondition{
	Type:   appsv1.DeploymentAvailable,
	Status: corev1.ConditionFalse,
}

var deploymentConditionUnknown = appsv1.DeploymentCondition{
	Type:   appsv1.DeploymentAvailable,
	Status: corev1.ConditionUnknown,
}

var deploymentStatusReady = &appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{deploymentConditionReady}}
var deploymentStatusNotReady = &appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{deploymentConditionNotReady}}
var deploymentStatusUnknown = &appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{deploymentConditionUnknown}}

var ignoreAllButTypeAndStatus = cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime", "Message", "Reason", "Severity")

func TestGetConditionSet(t *testing.T) {
	RegisterConsolidatedKafkaChannelConditionSet()
	kc := &messaging.KafkaChannel{}
	if got, want := kc.GetConditionSet().GetTopLevelConditionType(), apis.ConditionReady; got != want {
		t.Errorf("GetTopLevelCondition=%v, want=%v", got, want)
	}
}

func TestInitializeConditions(t *testing.T) {
	testCases := []struct {
		name string                        // TestCase Name
		cs   *messaging.KafkaChannelStatus // Starting ConditionSet
		want *messaging.KafkaChannelStatus // Expected ConditionSet
	}{{
		name: "empty",
		cs:   &messaging.KafkaChannelStatus{},
		want: &messaging.KafkaChannelStatus{
			ChannelableStatus: eventingduckv1.ChannelableStatus{
				Status: duckv1.Status{
					Conditions: []apis.Condition{{
						Type:   messaging.KafkaChannelConditionAddressable,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionChannelServiceReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionConfigReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   KafkaChannelConditionDispatcherReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   KafkaChannelConditionEndpointsReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   KafkaChannelConditionServiceReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionTopicReady,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
	}, {
		name: "one false",
		cs: &messaging.KafkaChannelStatus{
			ChannelableStatus: eventingduckv1.ChannelableStatus{
				Status: duckv1.Status{
					Conditions: []apis.Condition{{
						Type:   KafkaChannelConditionDispatcherReady,
						Status: corev1.ConditionFalse,
					}},
				},
			},
		},
		want: &messaging.KafkaChannelStatus{
			ChannelableStatus: eventingduckv1.ChannelableStatus{
				Status: duckv1.Status{
					Conditions: []apis.Condition{{
						Type:   messaging.KafkaChannelConditionAddressable,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionChannelServiceReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionConfigReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   KafkaChannelConditionDispatcherReady,
						Status: corev1.ConditionFalse,
					}, {
						Type:   KafkaChannelConditionEndpointsReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   KafkaChannelConditionServiceReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionTopicReady,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
	}, {
		name: "one true",
		cs: &messaging.KafkaChannelStatus{
			ChannelableStatus: eventingduckv1.ChannelableStatus{
				Status: duckv1.Status{
					Conditions: []apis.Condition{{
						Type:   KafkaChannelConditionDispatcherReady,
						Status: corev1.ConditionTrue,
					}},
				},
			},
		},
		want: &messaging.KafkaChannelStatus{
			ChannelableStatus: eventingduckv1.ChannelableStatus{
				Status: duckv1.Status{
					Conditions: []apis.Condition{{
						Type:   messaging.KafkaChannelConditionAddressable,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionChannelServiceReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionConfigReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   KafkaChannelConditionDispatcherReady,
						Status: corev1.ConditionTrue,
					}, {
						Type:   KafkaChannelConditionEndpointsReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   KafkaChannelConditionServiceReady,
						Status: corev1.ConditionUnknown,
					}, {
						Type:   messaging.KafkaChannelConditionTopicReady,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
	}}

	RegisterConsolidatedKafkaChannelConditionSet()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.cs.InitializeConditions()
			if diff := cmp.Diff(testCase.want, testCase.cs, ignoreAllButTypeAndStatus); diff != "" {
				t.Errorf("unexpected conditions (-want, +got) = %v", diff)
			}
		})
	}
}

func TestChannelIsReady(t *testing.T) {
	RegisterConsolidatedKafkaChannelConditionSet()
	tests := []struct {
		name                    string
		markServiceReady        bool
		markServiceUnknown      bool
		markChannelServiceReady bool
		markConfigurationReady  bool
		setAddress              bool
		markEndpointsReady      bool
		markTopicReady          bool
		wantReady               bool
		dispatcherStatus        *appsv1.DeploymentStatus
	}{{
		name:                    "all happy",
		markServiceReady:        true,
		markChannelServiceReady: true,
		markConfigurationReady:  true,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               true,
	}, {
		name:                    "service unknown",
		markServiceReady:        false,
		markServiceUnknown:      true,
		markChannelServiceReady: false,
		markConfigurationReady:  true,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "service not ready",
		markServiceReady:        false,
		markChannelServiceReady: true,
		markConfigurationReady:  true,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "endpoints not ready",
		markServiceReady:        true,
		markChannelServiceReady: true,
		markConfigurationReady:  true,
		markEndpointsReady:      false,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "deployment unknown",
		markServiceReady:        true,
		markConfigurationReady:  true,
		markEndpointsReady:      true,
		markChannelServiceReady: false,
		dispatcherStatus:        deploymentStatusUnknown,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "deployment not ready",
		markServiceReady:        true,
		markConfigurationReady:  true,
		markEndpointsReady:      true,
		markChannelServiceReady: true,
		dispatcherStatus:        deploymentStatusNotReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "address not set",
		markServiceReady:        true,
		markConfigurationReady:  true,
		markChannelServiceReady: true,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              false,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "channel service not ready",
		markServiceReady:        true,
		markConfigurationReady:  true,
		markChannelServiceReady: false,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "topic not ready",
		markServiceReady:        true,
		markConfigurationReady:  true,
		markChannelServiceReady: true,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          false,
		wantReady:               false,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cs := &messaging.KafkaChannelStatus{}
			cs.InitializeConditions()
			if test.markServiceReady {
				MarkServiceTrue(cs)
			} else {
				if test.markServiceUnknown {
					MarkServiceUnknown(cs, "UnknownService", "testing")
				} else {
					MarkServiceFailed(cs, "NotReadyService", "testing")
				}
			}
			if test.markChannelServiceReady {
				cs.MarkChannelServiceTrue()
			} else {
				cs.MarkChannelServiceFailed("NotReadyChannelService", "testing")
			}
			if test.markConfigurationReady {
				cs.MarkConfigTrue()
			} else {
				cs.MarkConfigFailed("NotReadyConfiguration", "testing")
			}
			if test.setAddress {
				cs.SetAddress(&apis.URL{Scheme: "http", Host: "foo.bar"})
			}
			if test.markEndpointsReady {
				MarkEndpointsTrue(cs)
			} else {
				MarkEndpointsFailed(cs, "NotReadyEndpoints", "testing")
			}
			if test.dispatcherStatus != nil {
				PropagateDispatcherStatus(cs, test.dispatcherStatus)
			} else {
				MarkDispatcherFailed(cs, "NotReadyDispatcher", "testing")
			}
			if test.markTopicReady {
				cs.MarkTopicTrue()
			} else {
				cs.MarkTopicFailed("NotReadyTopic", "testing")
			}
			got := cs.IsReady()
			if test.wantReady != got {
				t.Errorf("unexpected readiness: want %v, got %v", test.wantReady, got)
			}
		})
	}
}
