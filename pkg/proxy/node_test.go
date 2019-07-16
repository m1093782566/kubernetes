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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func makeTestNode(name string, labels map[string]string) *v1.Node {
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: map[string]string{},
			Labels:      labels,
		},
		Spec:   v1.NodeSpec{},
		Status: v1.NodeStatus{},
	}
	return node
}

func (fake *FakeProxier) addNode(node *v1.Node) {
	fake.nodeChanges.Update(nil, node)
}

func (fake *FakeProxier) updateNode(oldNode *v1.Node, node *v1.Node) {
	fake.nodeChanges.Update(oldNode, node)
}

func (fake *FakeProxier) deleteNode(node *v1.Node) {
	fake.nodeChanges.Update(node, nil)
}

func TestBuildNodeMapAddRemoveUpdate(t *testing.T) {
	fp := newFakeProxier()

	nodes := []*v1.Node{
		makeTestNode("testNode1", nil),
		makeTestNode("testNode2", nil),
		makeTestNode("testNode3", nil),
		makeTestNode("testNode4", nil),
	}

	for _, node := range nodes {
		fp.addNode(node)
	}

	UpdateNodeMap(fp.nodeMap, fp.nodeChanges)
	if len(fp.nodeMap) != 4 {
		t.Errorf("expected service map length 4, got %v", len(fp.nodeMap))
	}

	// Remove some stuff
	// oneNode is a modification of node[0] with added a label
	oneNode := makeTestNode("testNode1", map[string]string{"failure-domain.beta.kubernetes.io/region": "bj"})
	fp.updateNode(nodes[0], oneNode)
	fp.deleteNode(nodes[1])
	fp.deleteNode(nodes[2])
	fp.deleteNode(nodes[3])

	UpdateNodeMap(fp.nodeMap, fp.nodeChanges)
	if len(fp.nodeMap) != 1 {
		t.Errorf("expected service map length 1, got %v", fp.nodeMap)
	}
	if value, ok := fp.nodeMap["testNode1"].GetTopologyValue("failure-domain.beta.kubernetes.io/region"); (!ok) || (value != "bj") {
		t.Errorf("expected topology value 'bj', got '%s'", value)
	}
}
