package ec2

import (
	"fmt"
	"strings"
)

type IpamPoolAllocation struct {
	Cidr                 string
	Description          string
	IpamPoolAllocationID string
	ResourceID           string
	ResourceOwner        string
	ResourceRegion       string
	ResourceType         string
}

func (s *Service) AllocateIpamPoolCidr(
	ipamPoolID string,
	allowedCidrs []string,
	cidr *string,
	description *string,
	disallowedCidrs []string,
	netmaskLength *int32,
	previewNextCidr *bool,
) (IpamPoolAllocation, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	if ipamPoolID == "" {
		return IpamPoolAllocation{}, ErrInvalidParameter
	}

	var requestedCIDR string
	if cidr != nil {
		requestedCIDR = strings.TrimSpace(*cidr)
	}
	if requestedCIDR == "" && netmaskLength == nil {
		return IpamPoolAllocation{}, ErrInvalidParameter
	}

	var effectiveNetmask int32 = 24
	if netmaskLength != nil {
		effectiveNetmask = *netmaskLength
		if effectiveNetmask < 0 || effectiveNetmask > 128 {
			return IpamPoolAllocation{}, ErrInvalidParameter
		}
	}

	if requestedCIDR == "" {
		if effectiveNetmask <= 32 {
			requestedCIDR = fmt.Sprintf("10.%d.0.0/%d", (int(effectiveNetmask)%200)+10, effectiveNetmask)
		} else {
			requestedCIDR = fmt.Sprintf("2001:db8:%x::/%d", effectiveNetmask, effectiveNetmask)
		}
	}

	allowed := toStringSet(dedupeTrimmedStrings(allowedCidrs))
	if len(allowed) > 0 {
		if _, ok := allowed[requestedCIDR]; !ok {
			return IpamPoolAllocation{}, ErrInvalidParameter
		}
	}
	disallowed := toStringSet(dedupeTrimmedStrings(disallowedCidrs))
	if _, denied := disallowed[requestedCIDR]; denied {
		return IpamPoolAllocation{}, ErrInvalidParameter
	}

	allocation := IpamPoolAllocation{
		Cidr:                 requestedCIDR,
		Description:          "",
		IpamPoolAllocationID: "",
		ResourceID:           ipamPoolID,
		ResourceOwner:        DefaultAccountID,
		ResourceRegion:       DefaultRegion,
		ResourceType:         "ipam-pool",
	}
	if description != nil {
		allocation.Description = strings.TrimSpace(*description)
	}

	if previewNextCidr == nil || !*previewNextCidr {
		s.mu.Lock()
		defer s.mu.Unlock()

		allocation.IpamPoolAllocationID = s.nextIDLocked("ipam-pool-alloc")
		clone := allocation
		s.ipamPoolAllocations[clone.IpamPoolAllocationID] = &clone
		return allocation, nil
	}

	return allocation, nil
}
