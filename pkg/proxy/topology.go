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
)

func FilterTopologyEndpoint(currentNodeName types.NodeName, nodeMap NodeMap, topologyKeys []string, endpoints []Endpoint) []Endpoint {
	if len(topologyKeys) == 0 {
		return endpoints
	}
	currentNode, ok := nodeMap[currentNodeName]
	if !ok {
		return endpoints
	}
	filteredEndpoint := []Endpoint{}
	for _, key := range topologyKeys {
		if key == "" {
			return endpoints
		}
		topologyValue, ok := currentNode.GetTopologyValue(key)
		if !ok {
			continue
		}

		for _, ep := range endpoints {
			nodeName := ep.GetNodeName()
			if nodeName == "" {
				continue
			}
			node, ok := nodeMap[nodeName]
			if !ok {
				continue
			}
			if value, ok := node.GetTopologyValue(key); ok && value == topologyValue {
				filteredEndpoint = append(filteredEndpoint, ep)
			}
		}
		if len(filteredEndpoint) > 0 {
			break
		}
	}
	return filteredEndpoint
}
