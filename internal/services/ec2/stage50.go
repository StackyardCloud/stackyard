package ec2

import "strings"

type VpcEndpointConnectionNotification struct {
	ConnectionNotificationID    string
	ConnectionNotificationARN   string
	ConnectionEvents            []string
	ConnectionNotificationState string
	ConnectionNotificationType  string
	ServiceID                   string
	ServiceRegion               string
	VpcEndpointID               string
}

func (s *Service) ModifyVpcEndpointConnectionNotification(
	connectionNotificationID string,
	connectionNotificationARN *string,
	connectionEvents []string,
) (bool, error) {
	connectionNotificationID = strings.TrimSpace(connectionNotificationID)
	if connectionNotificationID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	notification := s.vpcEndpointConnectionNotifications[connectionNotificationID]
	if notification == nil {
		return false, ErrNotFound
	}

	if connectionNotificationARN != nil {
		arn := strings.TrimSpace(*connectionNotificationARN)
		if arn == "" {
			return false, ErrInvalidParameter
		}
		notification.ConnectionNotificationARN = arn
	}
	if len(connectionEvents) > 0 {
		notification.ConnectionEvents = dedupeTrimmedStrings(connectionEvents)
	}

	return true, nil
}
