package ec2

import "strings"

func (s *Service) StartVpcEndpointServicePrivateDnsVerification(serviceID string) (bool, error) {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcEndpointServiceConfigurations[serviceID]; !ok {
		return false, ErrNotFound
	}

	return true, nil
}
