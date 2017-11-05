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

package ipvs

import (
	"k8s.io/apimachinery/pkg/util/sets"
	utilipset "k8s.io/kubernetes/pkg/util/ipset"
	utilversion "k8s.io/kubernetes/pkg/util/version"

	"github.com/golang/glog"
)

const (
	// We need the IPv6 support from ipset 6.x
	MinIPSetCheckVersion = "6.0"

	// KubeLoopBackIPSet is the source ip set(ip type) created by ipvs proxier.
	KubeLoopBackIPSet = "KUBE-LOOP-BACK"

	// KubeMasqAllIPSet is the source ip set(ip:port type) created by ipvs proxier.
	KubeMasqAllIPSet = "KUBE-MASQ-ALL"

	// KubeClusterCIDRIPSet is the destination ip set created by ipvs proxier.
	KubeClusterCIDRIPSet = "KUBE-CLUSTER-CIDR"

	// KubeNodePortSet is the destination ip set created by ipvs proxier.
	KubeNodePortSetTCP = "KUBE-NODE-PORT-TCP"
	KubeNodePortSetUDP = "KUBE-NODE-PORT-UDP"

	// KubeServiceAccessSet is the destination ip set created by ipvs proxier.
	KubeServiceAccessSet = "KUBE-SERVICE-ACCESS"
)

// IPSetVersioner can query the current ipset version.
type IPSetVersioner interface {
	// returns "X.Y"
	GetVersion() (string, error)
}

type IPSet struct {
	utilipset.IPSet
	activeEntries sets.String
	handle        utilipset.Interface
}

func NewIPSet(handle utilipset.Interface, name string, setType utilipset.IPSetType, isIPv6 bool) *IPSet {
	hashFamily := utilipset.ProtocolFamilyIPV4
	if isIPv6 {
		hashFamily = utilipset.ProtocolFamilyIPV6
	}
	set := &IPSet{
		activeEntries: sets.NewString(),
		handle:        handle,
	}
	set.Name = name
	set.SetType = setType
	set.HashFamily = hashFamily
	return set
}

func (set *IPSet) isEmpty() bool {
	entries, _ := set.handle.ListEntries(set.Name)
	return len(entries) == 0
}

func (set *IPSet) resetEntries() {
	set.activeEntries = sets.NewString()
}

func (set *IPSet) syncIPSetEntries() {
	appliedEntries, err := set.handle.ListEntries(set.Name)
	if err != nil {
		glog.Errorf("Failed to list ip set entries, error: %v", err)
		return
	}

	// currentIPSetEntries represents Endpoints watched from API Server.
	currentIPSetEntries := sets.NewString()
	for _, appliedEntry := range appliedEntries {
		currentIPSetEntries.Insert(appliedEntry)
	}

	if !set.activeEntries.Equal(currentIPSetEntries) {
		// Clean legacy entries
		for _, entry := range currentIPSetEntries.Difference(set.activeEntries).List() {
			if err := set.handle.DelEntry(entry, set.Name); err != nil {
				glog.Errorf("Failed to delete ip set entry: %s from ip set: %s, error: %v", entry, set.Name, err)
			} else {
				glog.V(3).Infof("Successfully delete legacy ip set entry: %s from ip set: %s", entry, set.Name)
			}
		}
		// Create active entries
		for _, entry := range set.activeEntries.Difference(currentIPSetEntries).List() {
			if err := set.handle.AddEntry(entry, set.Name, true); err != nil {
				glog.Errorf("Failed to add entry: %v to ip set: %s, error: %v", entry, set.Name, err)
			} else {
				glog.Errorf("Successfully add entry: %v to ip set: %s", entry, set.Name)
			}
		}
	}
}

func ensureIPSets(ipSets ...*IPSet) error {
	for _, set := range ipSets {
		err := set.handle.CreateSet(&set.IPSet, true)
		if err != nil {
			glog.Errorf("Failed to make sure ip set: %v exist, error: %v", set, err)
			return err
		}
	}
	return nil
}

// checkMinVersion checks if ipset current version satisfies required min version
func checkMinVersion(vstring string) bool {
	version, err := utilversion.ParseGeneric(vstring)
	if err != nil {
		glog.Errorf("vstring (%s) is not a valid version string: %v", vstring, err)
		return false
	}

	minVersion, err := utilversion.ParseGeneric(MinIPSetCheckVersion)
	if err != nil {
		glog.Errorf("MinCheckVersion (%s) is not a valid version string: %v", MinIPSetCheckVersion, err)
		return false
	}
	return !version.LessThan(minVersion)
}
