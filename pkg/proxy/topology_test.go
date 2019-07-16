/*
Copyright 2019 The Kubernetes Authors.

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

package proxy

import (
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"testing"
)

func TestFilterTopologyEndpoint(t *testing.T) {
	testCases := []struct {
		nodeMap         NodeMap
		endpoints       []Endpoint
		currentNodeName types.NodeName
		topologyKeys    []string
		expected        []Endpoint
	}{
		{
			// Case[0]: no endpoint
			nodeMap: NodeMap{
				"testNode1": &BaseNodeInfo{
					name: "testNode",
					labels: map[string]string{
						"kubernetes.io/hostname":                   "10.0.0.1",
						"failure-domain.beta.kubernetes.io/zone":   "90001",
						"failure-domain.beta.kubernetes.io/region": "sg",
					},
				},
			},
			endpoints:       []Endpoint{},
			currentNodeName: "testNode",
			topologyKeys:    []string{"failure-domain.beta.kubernetes.io/region"},
			expected:        []Endpoint{},
		},
		{
			// Case[1]: no topologyKeys
			nodeMap: NodeMap{
				"testNode1": &BaseNodeInfo{
					name: "testNode1",
					labels: map[string]string{
						"kubernetes.io/hostname":                 "10.0.0.1",
						"failure-domain.beta.kubernetes.io/zone": "90001",
					},
				},
				"testNode2": &BaseNodeInfo{
					name: "testNode2",
					labels: map[string]string{
					},
				},
			},
			endpoints: []Endpoint{
				&BaseEndpointInfo{Endpoint: "1.1.1.1:11", NodeName: "testNode1"},
				&BaseEndpointInfo{Endpoint: "1.1.1.2:11", NodeName: "testNode2"},
			},
			currentNodeName: "testNode1",
			topologyKeys:    []string{},
			expected: []Endpoint{
				&BaseEndpointInfo{Endpoint: "1.1.1.1:11", NodeName: "testNode1"},
				&BaseEndpointInfo{Endpoint: "1.1.1.2:11", NodeName: "testNode2"},
			},
		},
		{
			// Case[2]: normal topology key with hard requirement
			nodeMap: NodeMap{
				"testNode1": &BaseNodeInfo{
					name: "testNode1",
					labels: map[string]string{
						"kubernetes.io/hostname":                 "10.0.0.1",
						"failure-domain.beta.kubernetes.io/zone": "90001",
					},
				},
				"testNode2": &BaseNodeInfo{
					name: "testNode2",
					labels: map[string]string{
						"kubernetes.io/hostname":                 "10.0.0.2",
						"failure-domain.beta.kubernetes.io/zone": "90002",
					},
				},
				"testNode3": &BaseNodeInfo{
					name: "testNode3",
					labels: map[string]string{
						"kubernetes.io/hostname":                 "10.0.0.3",
						"failure-domain.beta.kubernetes.io/zone": "90001",
					},
				},
			},
			endpoints: []Endpoint{
				&BaseEndpointInfo{Endpoint: "1.1.1.1:11", NodeName: "testNode1"},
				&BaseEndpointInfo{Endpoint: "1.1.1.2:11", NodeName: "testNode2"},
			},
			currentNodeName: "testNode3",
			topologyKeys:    []string{"kubernetes.io/hostname", "failure-domain.beta.kubernetes.io/zone"},
			expected: []Endpoint{
				&BaseEndpointInfo{Endpoint: "1.1.1.1:11", NodeName: "testNode1"},
			},
		},
		{
			// Case[3]: normal topology key with hard requirement (no endpoint matched)
			nodeMap: NodeMap{
				"testNode1": &BaseNodeInfo{
					name: "testNode1",
					labels: map[string]string{
						"kubernetes.io/hostname":                 "10.0.0.1",
						"failure-domain.beta.kubernetes.io/zone": "90001",
					},
				},
				"testNode2": &BaseNodeInfo{
					name: "testNode2",
					labels: map[string]string{
						"kubernetes.io/hostname":                 "10.0.0.2",
						"failure-domain.beta.kubernetes.io/zone": "90002",
					},
				},
				"testNode3": &BaseNodeInfo{
					name: "testNode3",
					labels: map[string]string{
						"kubernetes.io/hostname":                 "10.0.0.3",
						"failure-domain.beta.kubernetes.io/zone": "90001",
					},
				},
			},
			endpoints: []Endpoint{
				&BaseEndpointInfo{Endpoint: "1.1.1.1:11", NodeName: "testNode1"},
				&BaseEndpointInfo{Endpoint: "1.1.1.3:11", NodeName: "testNode3"},
			},
			currentNodeName: "testNode2",
			topologyKeys:    []string{"kubernetes.io/hostname", "failure-domain.beta.kubernetes.io/zone"},
			expected:        []Endpoint{},
		},
		{
			// Case[4]: match topology key "" with soft requirement
			nodeMap: NodeMap{
				"testNode1": &BaseNodeInfo{
					name: "testNode1",
					labels: map[string]string{
						"kubernetes.io/hostname":                   "10.0.0.1",
						"failure-domain.beta.kubernetes.io/zone":   "90001",
						"failure-domain.beta.kubernetes.io/region": "bj",
					},
				},
				"testNode2": &BaseNodeInfo{
					name: "testNode2",
					labels: map[string]string{
						"kubernetes.io/hostname":                   "10.0.0.2",
						"failure-domain.beta.kubernetes.io/zone":   "80001",
						"failure-domain.beta.kubernetes.io/region": "sh",
					},
				},
				"testNode3": &BaseNodeInfo{
					name: "testNode3",
					labels: map[string]string{
						"kubernetes.io/hostname":                   "10.0.0.3",
						"failure-domain.beta.kubernetes.io/zone":   "90002",
						"failure-domain.beta.kubernetes.io/region": "bj",
					},
				},
			},
			endpoints: []Endpoint{
				&BaseEndpointInfo{Endpoint: "1.1.1.1:11", NodeName: "testNode1"},
				&BaseEndpointInfo{Endpoint: "1.1.1.3:11", NodeName: "testNode3"},
			},
			currentNodeName: "testNode2",
			topologyKeys:    []string{"kubernetes.io/hostname", "failure-domain.beta.kubernetes.io/zone", "failure-domain.beta.kubernetes.io/region", ""},
			expected: []Endpoint{
				&BaseEndpointInfo{Endpoint: "1.1.1.1:11", NodeName: "testNode1"},
				&BaseEndpointInfo{Endpoint: "1.1.1.3:11", NodeName: "testNode3"},
			},
		},
	}
	for tci, tc := range testCases {
		filteredEndpoint := FilterTopologyEndpoint(tc.currentNodeName, tc.nodeMap, tc.topologyKeys, tc.endpoints)
		if !reflect.DeepEqual(filteredEndpoint, tc.expected) {
			t.Errorf("[%d] expected %v, got %v", tci, endpointsToStringArray(tc.expected), endpointsToStringArray(filteredEndpoint))
		}
	}
}

func endpointsToStringArray(endpoints []Endpoint) []string {
	result := []string{}
	for _, ep := range endpoints {
		result = append(result, ep.String())
	}
	return result
}
