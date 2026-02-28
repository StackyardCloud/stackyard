package ec2

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type AssignedPrivateIPAddress struct {
	PrivateIPAddress string
}

type AssignPrivateIPAddressesResult struct {
	NetworkInterfaceID   string
	AssignedPrivateIPs   []AssignedPrivateIPAddress
	AssignedIPv4Prefixes []string
}

type AssignIPv6AddressesResult struct {
	NetworkInterfaceID   string
	AssignedIPv6Addrs    []string
	AssignedIPv6Prefixes []string
}

type UnassignIPv6AddressesResult struct {
	NetworkInterfaceID     string
	UnassignedIPv6Addrs    []string
	UnassignedIPv6Prefixes []string
}

func (s *Service) AssignPrivateIPAddresses(
	networkInterfaceID string,
	privateIPs []string,
	secondaryPrivateIPCount *int32,
	ipv4Prefixes []string,
	ipv4PrefixCount *int32,
	allowReassignment bool,
) (AssignPrivateIPAddressesResult, error) {
	_ = allowReassignment
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	if networkInterfaceID == "" {
		return AssignPrivateIPAddressesResult{}, ErrInvalidParameter
	}
	if len(privateIPs) == 0 && secondaryPrivateIPCount == nil && len(ipv4Prefixes) == 0 && ipv4PrefixCount == nil {
		return AssignPrivateIPAddressesResult{}, ErrInvalidParameter
	}
	if len(privateIPs) > 0 && secondaryPrivateIPCount != nil {
		return AssignPrivateIPAddressesResult{}, ErrInvalidParameter
	}
	if len(ipv4Prefixes) > 0 && ipv4PrefixCount != nil {
		return AssignPrivateIPAddressesResult{}, ErrInvalidParameter
	}
	if secondaryPrivateIPCount != nil && *secondaryPrivateIPCount < 0 {
		return AssignPrivateIPAddressesResult{}, ErrInvalidParameter
	}
	if ipv4PrefixCount != nil && *ipv4PrefixCount < 0 {
		return AssignPrivateIPAddressesResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	iface := s.networkInterfaces[networkInterfaceID]
	if iface == nil {
		return AssignPrivateIPAddressesResult{}, ErrNotFound
	}

	assignedPrivate := make([]AssignedPrivateIPAddress, 0)
	seenPrivate := networkInterfacePrivateIPSet(iface)
	for _, ip := range privateIPs {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if _, ok := seenPrivate[ip]; ok {
			continue
		}
		iface.SecondaryPrivateIPs = append(iface.SecondaryPrivateIPs, ip)
		seenPrivate[ip] = struct{}{}
		assignedPrivate = append(assignedPrivate, AssignedPrivateIPAddress{PrivateIPAddress: ip})
	}
	if secondaryPrivateIPCount != nil {
		for i := int32(0); i < *secondaryPrivateIPCount; i++ {
			ip := s.nextNetworkInterfacePrivateIPLocked(iface)
			iface.SecondaryPrivateIPs = append(iface.SecondaryPrivateIPs, ip)
			seenPrivate[ip] = struct{}{}
			assignedPrivate = append(assignedPrivate, AssignedPrivateIPAddress{PrivateIPAddress: ip})
		}
	}

	assignedPrefixes := make([]string, 0)
	seenPrefixes := toStringSet(iface.IPv4Prefixes)
	for _, prefix := range ipv4Prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			continue
		}
		if _, ok := seenPrefixes[prefix]; ok {
			continue
		}
		iface.IPv4Prefixes = append(iface.IPv4Prefixes, prefix)
		seenPrefixes[prefix] = struct{}{}
		assignedPrefixes = append(assignedPrefixes, prefix)
	}
	if ipv4PrefixCount != nil {
		for i := int32(0); i < *ipv4PrefixCount; i++ {
			prefix := s.nextNetworkInterfaceIPv4PrefixLocked(iface)
			iface.IPv4Prefixes = append(iface.IPv4Prefixes, prefix)
			seenPrefixes[prefix] = struct{}{}
			assignedPrefixes = append(assignedPrefixes, prefix)
		}
	}

	return AssignPrivateIPAddressesResult{
		NetworkInterfaceID:   iface.ID,
		AssignedPrivateIPs:   assignedPrivate,
		AssignedIPv4Prefixes: assignedPrefixes,
	}, nil
}

func (s *Service) UnassignPrivateIPAddresses(networkInterfaceID string, privateIPs, ipv4Prefixes []string) error {
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	if networkInterfaceID == "" {
		return ErrInvalidParameter
	}
	if len(privateIPs) == 0 && len(ipv4Prefixes) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	iface := s.networkInterfaces[networkInterfaceID]
	if iface == nil {
		return ErrNotFound
	}

	removePrivate := toStringSet(privateIPs)
	if len(removePrivate) > 0 {
		filtered := make([]string, 0, len(iface.SecondaryPrivateIPs))
		for _, ip := range iface.SecondaryPrivateIPs {
			if _, ok := removePrivate[ip]; ok {
				continue
			}
			filtered = append(filtered, ip)
		}
		iface.SecondaryPrivateIPs = filtered
	}

	removePrefixes := toStringSet(ipv4Prefixes)
	if len(removePrefixes) > 0 {
		filtered := make([]string, 0, len(iface.IPv4Prefixes))
		for _, prefix := range iface.IPv4Prefixes {
			if _, ok := removePrefixes[prefix]; ok {
				continue
			}
			filtered = append(filtered, prefix)
		}
		iface.IPv4Prefixes = filtered
	}

	return nil
}

func (s *Service) AssignIPv6Addresses(
	networkInterfaceID string,
	ipv6Addrs []string,
	ipv6AddrCount *int32,
	ipv6Prefixes []string,
	ipv6PrefixCount *int32,
) (AssignIPv6AddressesResult, error) {
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	if networkInterfaceID == "" {
		return AssignIPv6AddressesResult{}, ErrInvalidParameter
	}
	if len(ipv6Addrs) == 0 && ipv6AddrCount == nil && len(ipv6Prefixes) == 0 && ipv6PrefixCount == nil {
		return AssignIPv6AddressesResult{}, ErrInvalidParameter
	}
	if len(ipv6Addrs) > 0 && ipv6AddrCount != nil {
		return AssignIPv6AddressesResult{}, ErrInvalidParameter
	}
	if len(ipv6Prefixes) > 0 && ipv6PrefixCount != nil {
		return AssignIPv6AddressesResult{}, ErrInvalidParameter
	}
	if ipv6AddrCount != nil && *ipv6AddrCount < 0 {
		return AssignIPv6AddressesResult{}, ErrInvalidParameter
	}
	if ipv6PrefixCount != nil && *ipv6PrefixCount < 0 {
		return AssignIPv6AddressesResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	iface := s.networkInterfaces[networkInterfaceID]
	if iface == nil {
		return AssignIPv6AddressesResult{}, ErrNotFound
	}

	assignedAddrs := make([]string, 0)
	seenAddrs := toStringSet(iface.IPv6Addresses)
	for _, address := range ipv6Addrs {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}
		if _, ok := seenAddrs[address]; ok {
			continue
		}
		iface.IPv6Addresses = append(iface.IPv6Addresses, address)
		seenAddrs[address] = struct{}{}
		assignedAddrs = append(assignedAddrs, address)
	}
	if ipv6AddrCount != nil {
		for i := int32(0); i < *ipv6AddrCount; i++ {
			address := s.nextNetworkInterfaceIPv6AddressLocked(iface)
			iface.IPv6Addresses = append(iface.IPv6Addresses, address)
			seenAddrs[address] = struct{}{}
			assignedAddrs = append(assignedAddrs, address)
		}
	}

	assignedPrefixes := make([]string, 0)
	seenPrefixes := toStringSet(iface.IPv6Prefixes)
	for _, prefix := range ipv6Prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			continue
		}
		if _, ok := seenPrefixes[prefix]; ok {
			continue
		}
		iface.IPv6Prefixes = append(iface.IPv6Prefixes, prefix)
		seenPrefixes[prefix] = struct{}{}
		assignedPrefixes = append(assignedPrefixes, prefix)
	}
	if ipv6PrefixCount != nil {
		for i := int32(0); i < *ipv6PrefixCount; i++ {
			prefix := s.nextNetworkInterfaceIPv6PrefixLocked(iface)
			iface.IPv6Prefixes = append(iface.IPv6Prefixes, prefix)
			seenPrefixes[prefix] = struct{}{}
			assignedPrefixes = append(assignedPrefixes, prefix)
		}
	}

	return AssignIPv6AddressesResult{
		NetworkInterfaceID:   iface.ID,
		AssignedIPv6Addrs:    assignedAddrs,
		AssignedIPv6Prefixes: assignedPrefixes,
	}, nil
}

func (s *Service) UnassignIPv6Addresses(networkInterfaceID string, ipv6Addrs, ipv6Prefixes []string) (UnassignIPv6AddressesResult, error) {
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	if networkInterfaceID == "" {
		return UnassignIPv6AddressesResult{}, ErrInvalidParameter
	}
	if len(ipv6Addrs) == 0 && len(ipv6Prefixes) == 0 {
		return UnassignIPv6AddressesResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	iface := s.networkInterfaces[networkInterfaceID]
	if iface == nil {
		return UnassignIPv6AddressesResult{}, ErrNotFound
	}

	removeAddrs := toStringSet(ipv6Addrs)
	unassignedAddrs := make([]string, 0)
	if len(removeAddrs) > 0 {
		filtered := make([]string, 0, len(iface.IPv6Addresses))
		for _, address := range iface.IPv6Addresses {
			if _, ok := removeAddrs[address]; ok {
				unassignedAddrs = append(unassignedAddrs, address)
				continue
			}
			filtered = append(filtered, address)
		}
		iface.IPv6Addresses = filtered
	}

	removePrefixes := toStringSet(ipv6Prefixes)
	unassignedPrefixes := make([]string, 0)
	if len(removePrefixes) > 0 {
		filtered := make([]string, 0, len(iface.IPv6Prefixes))
		for _, prefix := range iface.IPv6Prefixes {
			if _, ok := removePrefixes[prefix]; ok {
				unassignedPrefixes = append(unassignedPrefixes, prefix)
				continue
			}
			filtered = append(filtered, prefix)
		}
		iface.IPv6Prefixes = filtered
	}

	return UnassignIPv6AddressesResult{
		NetworkInterfaceID:     iface.ID,
		UnassignedIPv6Addrs:    unassignedAddrs,
		UnassignedIPv6Prefixes: unassignedPrefixes,
	}, nil
}

func networkInterfacePrivateIPSet(iface *NetworkInterface) map[string]struct{} {
	out := map[string]struct{}{}
	if iface == nil {
		return out
	}
	if primary := strings.TrimSpace(iface.PrivateIP); primary != "" {
		out[primary] = struct{}{}
	}
	for _, ip := range iface.SecondaryPrivateIPs {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		out[ip] = struct{}{}
	}
	return out
}

func (s *Service) nextNetworkInterfacePrivateIPLocked(iface *NetworkInterface) string {
	used := networkInterfacePrivateIPSet(iface)
	for suffix := 11; suffix <= 254; suffix++ {
		candidate := "10.0.0." + strconv.Itoa(suffix)
		if _, ok := used[candidate]; ok {
			continue
		}
		return candidate
	}
	return "10.0.1.1"
}

func (s *Service) nextNetworkInterfaceIPv4PrefixLocked(iface *NetworkInterface) string {
	used := toStringSet(iface.IPv4Prefixes)
	for i := 0; i <= 255; i++ {
		candidate := fmt.Sprintf("10.0.%d.0/28", i)
		if _, ok := used[candidate]; ok {
			continue
		}
		return candidate
	}
	return "10.1.0.0/28"
}

func (s *Service) nextNetworkInterfaceIPv6AddressLocked(iface *NetworkInterface) string {
	used := toStringSet(iface.IPv6Addresses)
	for i := 1; i <= 65535; i++ {
		candidate := fmt.Sprintf("2001:db8::%x", i)
		if _, ok := used[candidate]; ok {
			continue
		}
		return candidate
	}
	return "2001:db8::ffff:1"
}

func (s *Service) nextNetworkInterfaceIPv6PrefixLocked(iface *NetworkInterface) string {
	used := toStringSet(iface.IPv6Prefixes)
	for i := 1; i <= 65535; i++ {
		candidate := fmt.Sprintf("2001:db8:%x::/80", i)
		if _, ok := used[candidate]; ok {
			continue
		}
		return candidate
	}
	return "2001:db8:ffff::/80"
}

func sortAndDedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
