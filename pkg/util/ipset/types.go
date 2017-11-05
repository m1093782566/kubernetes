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

// IPSetType represents the ipset type
type IPSetType string

const (
	// HashIpPort represents the `hash:ip,port` type ipset
	HashIpPort IPSetType = "hash:ip,port"
	// HashIpPort represents the `hash:ip` type ipset
	HashIp     IPSetType = "hash:ip"
	// HashIpPort represents the `bitmap:port` type ipset
	BitmapPort IPSetType = "bitmap:port"
)

// DefaultPortRange defines the default bitmap:port valid port range.
const DefaultPortRange string = "0-65535"

const (
	// ProtocolFamilyIPV4 represents IPv4 protocol.
	ProtocolFamilyIPV4 = "inet"
	// ProtocolFamilyIPV6 represents IPv6 protocol.
	ProtocolFamilyIPV6 = "inet6"
	// ProtocolTCP represents TCP protocol.
	ProtocolTCP        = "tcp"
	// ProtocolUDP represents UDP protocol.
	ProtocolUDP        = "udp"
)

// ValidIPSetTypes defines the supported ip set type.
var ValidIPSetTypes = []IPSetType{
	HashIpPort,
	HashIp,
	BitmapPort,
}

// IsValidIPSetType checks if the given ipset type is valid.
func IsValidIPSetType(set IPSetType) bool {
	for _, valid := range ValidIPSetTypes {
		if set == valid {
			return true
		}
	}
	return false
}
