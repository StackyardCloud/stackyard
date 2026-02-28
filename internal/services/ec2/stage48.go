package ec2

import "strings"

func (s *Service) ModifyVpcEndpointServicePayerResponsibility(serviceID, payerResponsibility string) (bool, error) {
	serviceID = strings.TrimSpace(serviceID)
	payerResponsibility = strings.TrimSpace(payerResponsibility)
	if serviceID == "" || payerResponsibility == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.vpcEndpointServicePayerResponsibility[serviceID]
	if !ok {
		return false, ErrNotFound
	}

	switch {
	case strings.EqualFold(payerResponsibility, "ServiceOwner"):
		s.vpcEndpointServicePayerResponsibility[serviceID] = "ServiceOwner"
		if cfg := s.vpcEndpointServiceConfigurations[serviceID]; cfg != nil {
			cfg.PayerResponsibility = "ServiceOwner"
		}
		return true, nil
	case strings.EqualFold(payerResponsibility, "EndpointOwner"):
		if strings.EqualFold(current, "ServiceOwner") {
			return false, ErrInvalidParameter
		}
		s.vpcEndpointServicePayerResponsibility[serviceID] = "EndpointOwner"
		if cfg := s.vpcEndpointServiceConfigurations[serviceID]; cfg != nil {
			cfg.PayerResponsibility = "EndpointOwner"
		}
		return true, nil
	default:
		return false, ErrInvalidParameter
	}
}
