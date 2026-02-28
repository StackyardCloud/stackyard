package ec2

import "strings"

func (s *Service) DeleteVpcEndpointServiceConfigurations(serviceIDs []string) ([]UnsuccessfulItem, error) {
	serviceIDs = dedupeTrimmedStrings(serviceIDs)
	if len(serviceIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	unsuccessful := make([]UnsuccessfulItem, 0)
	for _, serviceID := range serviceIDs {
		serviceID = strings.TrimSpace(serviceID)
		if serviceID == "" {
			continue
		}
		if _, ok := s.vpcEndpointServiceConfigurations[serviceID]; !ok {
			unsuccessful = append(unsuccessful, UnsuccessfulItem{
				ResourceID: serviceID,
				Code:       "InvalidServiceId.NotFound",
				Message:    "service configuration not found",
			})
			continue
		}
		delete(s.vpcEndpointServiceConfigurations, serviceID)
		delete(s.vpcEndpointServicePayerResponsibility, serviceID)
		delete(s.vpcEndpointServicePermissions, serviceID)
	}

	return unsuccessful, nil
}
