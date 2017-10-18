/*
Copyright 2017 The Kubernetes Authors.

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

package testing

import (
	"k8s.io/kubernetes/pkg/util/ipset"
)

// no-op implementation of ipset Interface
type FakeIPSet struct {
	Lines []byte
}

func NewFake() *FakeIPSet {
	return &FakeIPSet{}
}

func (*FakeIPSet) GetVersion() (string, error) {
	return "0.0", nil
}

func (*FakeIPSet) FlushSet(set string) error {
	return nil
}

func (*FakeIPSet) DestroySet(set string) error {
	return nil
}

func (*FakeIPSet) DestroyAllSets() error {
	return nil
}

func (*FakeIPSet) CreateSet(set *ipset.IPSet, ignoreExistErr bool) error {
	return nil
}

func (*FakeIPSet) AddEntry(entry string, set string, ignoreExistErr bool) error {
	return nil
}

func (*FakeIPSet) DelEntry(entry string, set string) error {
	return nil
}

func (*FakeIPSet) TestEntry(entry string, set string) (bool, error) {
	return true, nil
}

func (*FakeIPSet) ListEntries(set string) ([]string, error) {
	return nil, nil
}

func (*FakeIPSet) ListSets() ([]string, error) {
	return nil, nil
}

var _ = ipset.Interface(&FakeIPSet{})
