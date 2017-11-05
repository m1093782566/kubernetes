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

package ipset

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	utilexec "k8s.io/utils/exec"
)

// Interface is an injectable interface for running ipset commands.  Implementations must be goroutine-safe.
type Interface interface {
	// FlushSet deletes all entries from a named set
	FlushSet(set string) error
	// DestroySet deletes a named set
	DestroySet(set string) error
	// DestroyAllSets deletes all sets
	DestroyAllSets() error
	// CreateSet creates a new set -> EnsureSet()?
	CreateSet(set *IPSet, ignoreExistErr bool) error
	// AddEntry adds a new entry to the named set.
	AddEntry(entry string, set string, ignoreExistErr bool) error
	// DelEntry deletes one entry from the named set
	DelEntry(entry string, set string) error
	// Test test if an entry exists in the named set
	TestEntry(entry string, set string) (bool, error)
	// ListEntries lists all the entries from a named set
	ListEntries(set string) ([]string, error)
	// ListSets list all set names from kernel
	ListSets() ([]string, error)
	// GetVersion returns the "X.Y" version string for ipset.
	GetVersion() (string, error)
}

const IPSetCmd = "ipset"

// IPSet implements an Interface to an set.
type IPSet struct {
	Name       string
	SetType    IPSetType
	HashFamily string
	HashSize   int
	MaxElem    int
	PortRange  string
}

type Entry struct {
	IP       string
	Port     int
	Protocol string
	SetType  IPSetType
}

func (e *Entry) String() string {
	switch e.SetType {
	case HashIpPort:
		// Entry{192.168.1.1, udp, 53} -> 192.168.1.1,udp:53
		// Entry{192.168.1.2, tcp, 8080} -> 192.168.1.2,tcp:8080
		return e.IP + "," + e.Protocol + ":" + strconv.Itoa(e.Port)
	case HashIp:
		// Entry{192.168.1.1} -> 192.168.1.1
		return e.IP
	case BitmapPort:
		// Entry{53} -> 53
		// Entry{8080} -> 8080
		return strconv.Itoa(e.Port)
	}
	return ""
}

type runner struct {
	exec utilexec.Interface
}

// New returns a new Interface which will exec ipset.
func New(exec utilexec.Interface) Interface {
	return &runner{
		exec: exec,
	}
}

func (runner *runner) CreateSet(set *IPSet, ignoreExistErr bool) error {
	// Using default values.
	if set.HashSize == 0 {
		set.HashSize = 1024
	}
	if set.MaxElem == 0 {
		set.MaxElem = 65536
	}
	if set.HashFamily == "" {
		set.HashFamily = ProtocolFamilyIPV4
	}
	if len(set.HashFamily) != 0 && set.HashFamily != ProtocolFamilyIPV4 && set.HashFamily != ProtocolFamilyIPV6 {
		return fmt.Errorf("Currently supported protocol families are: %s and %s, %s is not supported", ProtocolFamilyIPV4, ProtocolFamilyIPV6, set.HashFamily)
	}
	// Default ipset type is "hash:ip,port"
	if len(set.SetType) == 0 {
		set.SetType = HashIpPort
	}
	// Check if setType is supported
	if !IsValidIPSetType(set.SetType) {
		return fmt.Errorf("Currently supported ipset types are: %v, %s is not supported", ValidIPSetTypes, set.SetType)
	}

	return runner.createSet(set, ignoreExistErr)
}

// If ignoreExistErr set to true, then the -exist option of ipset will be specified, ipset ignores the error
// otherwise raised when the same set (setname and create parameters are identical) already exists.
func (runner *runner) createSet(set *IPSet, ignoreExistErr bool) error {
	args := []string{
		"create", set.Name, string(set.SetType),
	}
	if set.SetType == HashIp || set.SetType == HashIpPort {
		args = append(args,
			"family", set.HashFamily,
			"hashsize", strconv.Itoa(set.HashSize),
			"maxelem", strconv.Itoa(set.MaxElem),
		)
	}
	if set.SetType == BitmapPort {
		if len(set.PortRange) == 0 {
			set.PortRange = DefaultPortRange
		}
		if !validatePortRange(set.PortRange) {
			return fmt.Errorf("invalid port range for %s type ip set: %s, expect: a-b", BitmapPort, set.PortRange)
		}
		args = append(args,
			"range", set.PortRange,
		)
	}
	if ignoreExistErr {
		args = append(args, "-exist")
	}
	_, err := runner.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating ipset %s, error: %v", set.Name, err)
	}
	return nil
}

// If the -exist option is specified, ipset ignores the error otherwise raised when
// the same set (setname and create parameters are identical) already exists.
func (runner *runner) AddEntry(entry string, set string, ignoreExistErr bool) error {
	args := []string{
		"add", set, entry,
	}
	if ignoreExistErr {
		args = append(args, "-exist")
	}
	_, err := runner.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error adding entry %s, error: %v", entry, err)
	}
	return nil
}

// Del is used to delete the specified entry from the set.
func (runner *runner) DelEntry(entry string, set string) error {
	_, err := runner.exec.Command(IPSetCmd, "del", set, entry).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error deleting entry %s: from set: %s, error: %v", entry, set, err)
	}
	return nil
}

// Test is used to check whether the specified entry is in the set or not.
func (runner *runner) TestEntry(entry string, set string) (bool, error) {
	out, err := runner.exec.Command(IPSetCmd, "test", set, entry).CombinedOutput()
	if err == nil {
		reg, e := regexp.Compile("NOT")
		if e == nil && reg.MatchString(string(out)) {
			return false, nil
		} else if e == nil {
			return true, nil
		} else {
			return false, fmt.Errorf("error testing entry: %s, error: %v", entry, e)
		}
	} else {
		return false, fmt.Errorf("error testing entry %s: %v (%s)", entry, err, out)
	}
}

func (runner *runner) FlushSet(set string) error {
	_, err := runner.exec.Command(IPSetCmd, "flush", set).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error flushing set: %s, error: %v", set, err)
	}
	return nil
}

// DestroySet is used to destroy a named set.
func (runner *runner) DestroySet(set string) error {
	_, err := runner.exec.Command(IPSetCmd, "destroy", set).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error destroying set %s:, error: %v", set, err)
	}
	return nil
}

// DestroyAllSets is used to destroy all sets.
func (runner *runner) DestroyAllSets() error {
	_, err := runner.exec.Command(IPSetCmd, "destroy").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error destroying all sets, error: %v", err)
	}
	return nil
}

func (runner *runner) ListSets() ([]string, error) {
	out, err := runner.exec.Command(IPSetCmd, "list", "-n").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error listing all sets, error: %v", err)
	}
	return strings.Split(string(out), "\n"), nil
}

//Name: foobar
//Type: hash:ip,port
//Revision: 2
//Header: family inet hashsize 1024 maxelem 65536
//Size in memory: 16592
//References: 0
//Members:
//192.168.1.2,tcp:8080
//192.168.1.1,udp:53
// ListEntries lists all the entries from a named set
func (runner *runner) ListEntries(set string) ([]string, error) {
	if len(set) == 0 {
		return nil, fmt.Errorf("set name can't be nil")
	}
	out, err := runner.exec.Command(IPSetCmd, "list", set).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error listing set: %s, error: %v", set, err)
	}
	r := regexp.MustCompile("(?m)^(.*\n)*Members:\n")
	list := r.ReplaceAllString(string(out[:]), "")
	strs := strings.Split(list, "\n")
	results := make([]string, 0)
	for i := range strs {
		if len(strs[i]) > 0 {
			results = append(results, strs[i])
		}
	}
	return results, nil
}

// GetVersion returns the version string.
func (runner *runner) GetVersion() (string, error) {
	return getIPSetVersionString(runner.exec)
}

// getIPSetVersionString runs "ipset --version" to get the version string
// in the form of "X.Y", i.e "6.19"
func getIPSetVersionString(exec utilexec.Interface) (string, error) {
	cmd := exec.Command(IPSetCmd, "--version")
	cmd.SetStdin(bytes.NewReader([]byte{}))
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	versionMatcher := regexp.MustCompile("v[0-9]+\\.[0-9]+")
	match := versionMatcher.FindStringSubmatch(string(bytes))
	if match == nil {
		return "", fmt.Errorf("no ipset version found in string: %s", bytes)
	}
	return match[0], nil
}

func validatePortRange(portRange string) bool {
	strs := strings.Split(portRange, "-")
	if len(strs) != 2 {
		return false
	}
	for i := range strs {
		_, err := strconv.Atoi(strs[i])
		if err != nil {
			return false
		}
	}
	return true
}

var _ = Interface(&runner{})
