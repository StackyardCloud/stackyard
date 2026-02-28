package ec2

import (
	"strings"
)

func (s *Service) MoveAddressToVpc(publicIP string) (string, string, error) {
	publicIP = strings.TrimSpace(publicIP)
	if publicIP == "" {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	address := s.resolveElasticAddressLocked("", publicIP)
	if address == nil {
		return "", "", ErrNotFound
	}
	if address.AssociationID != "" {
		return "", "", ErrConflict
	}

	if strings.EqualFold(address.Domain, "vpc") {
		return address.AllocationID, "InVpc", nil
	}
	address.Domain = "vpc"
	if address.AllocationID == "" {
		address.AllocationID = s.nextIDLocked("eipalloc")
	}
	return address.AllocationID, "InVpc", nil
}

func (s *Service) RestoreAddressToClassic(publicIP string) (string, string, error) {
	publicIP = strings.TrimSpace(publicIP)
	if publicIP == "" {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	address := s.resolveElasticAddressLocked("", publicIP)
	if address == nil {
		return "", "", ErrNotFound
	}
	if address.AssociationID != "" {
		return "", "", ErrConflict
	}

	address.Domain = "standard"
	return address.PublicIP, "InClassic", nil
}

func (s *Service) ModifyAddressAttribute(allocationID, domainName string) (AddressAttribute, error) {
	allocationID = strings.TrimSpace(allocationID)
	domainName = strings.TrimSpace(domainName)
	if allocationID == "" || domainName == "" {
		return AddressAttribute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	address := s.addresses[allocationID]
	if address == nil {
		return AddressAttribute{}, ErrNotFound
	}
	address.PtrRecord = domainName
	return cloneAddressAttributeFromElasticAddress(address), nil
}

func (s *Service) ResetAddressAttribute(allocationID, attribute string) (AddressAttribute, error) {
	allocationID = strings.TrimSpace(allocationID)
	attribute = strings.ToLower(strings.TrimSpace(attribute))
	if allocationID == "" || attribute == "" {
		return AddressAttribute{}, ErrInvalidParameter
	}
	if attribute != "domain-name" {
		return AddressAttribute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	address := s.addresses[allocationID]
	if address == nil {
		return AddressAttribute{}, ErrNotFound
	}
	address.PtrRecord = ""
	return cloneAddressAttributeFromElasticAddress(address), nil
}

func cloneAddressAttributeFromElasticAddress(address *ElasticAddress) AddressAttribute {
	ptr := strings.TrimSpace(address.PtrRecord)
	if ptr == "" {
		ptr = strings.ReplaceAll(address.PublicIP, ".", "-") + ".compute.internal"
	}
	return AddressAttribute{
		AllocationID: address.AllocationID,
		PublicIP:     address.PublicIP,
		PtrRecord:    ptr,
	}
}
