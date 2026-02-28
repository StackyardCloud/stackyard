package ec2

import "strings"

type UnsuccessfulItem struct {
	ResourceID string
	Code       string
	Message    string
}

func (s *Service) DeleteVpcEndpointConnectionNotifications(connectionNotificationIDs []string) ([]UnsuccessfulItem, error) {
	connectionNotificationIDs = dedupeTrimmedStrings(connectionNotificationIDs)
	if len(connectionNotificationIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	unsuccessful := make([]UnsuccessfulItem, 0)
	for _, id := range connectionNotificationIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := s.vpcEndpointConnectionNotifications[id]; !ok {
			unsuccessful = append(unsuccessful, UnsuccessfulItem{
				ResourceID: id,
				Code:       "InvalidConnectionNotificationId.NotFound",
				Message:    "connection notification not found",
			})
			continue
		}
		delete(s.vpcEndpointConnectionNotifications, id)
	}
	return unsuccessful, nil
}
