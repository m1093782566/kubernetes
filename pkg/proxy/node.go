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
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/proxy/metrics"
	"reflect"
	"sync"
)

// BaseNodeInfo contains base information that defines a node.
type BaseNodeInfo struct {
	name   types.NodeName
	labels map[string]string
}

var _ Node = &BaseNodeInfo{}

func newBaseNodeInfo(name string, labels map[string]string) *BaseNodeInfo {
	return &BaseNodeInfo{
		name:   types.NodeName(name),
		labels: labels,
	}
}

// NodeName is part of proxy.Node interface.
func (info *BaseNodeInfo) NodeName() types.NodeName {
	return info.name
}

// GetTopologyValue is part of proxy.Node interface.
func (info *BaseNodeInfo) GetTopologyValue(key string) (string, bool) {
	if value, ok := info.labels[key]; ok {
		return value, true
	}
	return "", false
}

// NodeChangeTracker carries state about uncommitted changes to an arbitrary number of
// Nodes
type NodeChangeTracker struct {
	// lock protects items.
	lock sync.Mutex

	// items maps a service to is nodeChange.
	items map[types.NodeName]*nodeChange
}

// NewNodeChangeTracker initializes an NodesChangeMap
func NewNodeChangeTracker() *NodeChangeTracker {
	return &NodeChangeTracker{
		items: make(map[types.NodeName]*nodeChange),
	}
}

// Update updates given node's node change map based on the <previous, current> node pair.  It returns true
// if items changed, otherwise return false.  Update can be used to add/update/delete items of NodeChangeMap.  For example,
// Add item
//   - pass <nil, node> as the <previous, current> pair.
// Update item
//   - pass <oldNodes, node> as the <previous, current> pair.
// Delete item
//   - pass <node, nil> as the <previous, current> pair.
func (ect *NodeChangeTracker) Update(previous, current *v1.Node) bool {
	node := current
	if node == nil {
		node = previous
	}
	// previous == nil && current == nil is unexpected, we should return false directly.
	if node == nil {
		return false
	}
	metrics.NodeChangesTotal.Inc()

	ect.lock.Lock()
	defer ect.lock.Unlock()

	change, exists := ect.items[types.NodeName(node.Name)]
	if !exists {
		change = &nodeChange{}
		change.previous = ect.convertNode(previous)
		ect.items[types.NodeName(node.Name)] = change
	}
	change.current = ect.convertNode(current)
	// if change.previous equal to change.current, it means no change
	if reflect.DeepEqual(change.previous, change.current) {
		delete(ect.items, types.NodeName(node.Name))
	}

	metrics.NodeChangesPending.Set(float64(len(ect.items)))
	return len(ect.items) > 0
}

func (ect *NodeChangeTracker) convertNode(node *v1.Node) Node {
	if node == nil {
		return nil
	}
	return newBaseNodeInfo(node.Name, node.ObjectMeta.Labels)
}

// nodeChange contains all changes to node that happened since proxy rules were synced.  For a single object,
// changes are accumulated, i.e. previous is state from before applying the changes,
// current is state after applying the changes.
type nodeChange struct {
	previous Node
	current  Node
}

// NodeMap maps a node name to a Node.
type NodeMap map[types.NodeName]Node

// apply the changes to NodeMap.
func (em NodeMap) apply(changes *NodeChangeTracker) {
	if changes == nil {
		return
	}
	changes.lock.Lock()
	defer changes.lock.Unlock()
	for _, change := range changes.items {
		em.remove(change.previous)
		em.add(change.current)
	}
	changes.items = make(map[types.NodeName]*nodeChange)
	metrics.NodeChangesPending.Set(0)
}

// Add adds a node to NodeMap
func (em NodeMap) add(other Node) {
	if other != nil {
		em[other.NodeName()] = other
	}
}

// Remove removes a node in NodeMap
func (em NodeMap) remove(other Node) {
	if other != nil {
		delete(em, other.NodeName())
	}
}

// UpdateNodeMap updates NodeMap based on the given changes.
func UpdateNodeMap(nodeMap NodeMap, changes *NodeChangeTracker) {
	nodeMap.apply(changes)
}
