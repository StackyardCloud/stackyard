package ec2

import "strings"

func (s *Service) DeleteVpcEndpoints(vpcEndpointIDs []string) ([]UnsuccessfulItem, error) {
	vpcEndpointIDs = dedupeTrimmedStrings(vpcEndpointIDs)
	if len(vpcEndpointIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	unsuccessful := make([]UnsuccessfulItem, 0)
	for _, endpointID := range vpcEndpointIDs {
		endpointID = strings.TrimSpace(endpointID)
		if endpointID == "" {
			continue
		}
		if _, ok := s.vpcEndpoints[endpointID]; !ok {
			unsuccessful = append(unsuccessful, UnsuccessfulItem{
				ResourceID: endpointID,
				Code:       "InvalidVpcEndpointId.NotFound",
				Message:    "vpc endpoint not found",
			})
			continue
		}
		delete(s.vpcEndpoints, endpointID)
	}

	return unsuccessful, nil
}
