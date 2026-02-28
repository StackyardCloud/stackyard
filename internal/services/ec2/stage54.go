package ec2

import "strings"

func (s *Service) CreateVpcEndpointConnectionNotification(
	serviceID, vpcEndpointID, connectionNotificationARN string,
	connectionEvents []string,
	clientToken *string,
) (VpcEndpointConnectionNotification, *string, error) {
	serviceID = strings.TrimSpace(serviceID)
	vpcEndpointID = strings.TrimSpace(vpcEndpointID)
	connectionNotificationARN = strings.TrimSpace(connectionNotificationARN)
	if connectionNotificationARN == "" {
		return VpcEndpointConnectionNotification{}, nil, ErrInvalidParameter
	}
	connectionEvents = dedupeTrimmedStrings(connectionEvents)
	if len(connectionEvents) == 0 {
		return VpcEndpointConnectionNotification{}, nil, ErrInvalidParameter
	}
	if serviceID == "" && vpcEndpointID == "" {
		return VpcEndpointConnectionNotification{}, nil, ErrInvalidParameter
	}

	var trimmedToken *string
	if clientToken != nil {
		token := strings.TrimSpace(*clientToken)
		if token != "" {
			trimmedToken = &token
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if serviceID != "" {
		if _, ok := s.vpcEndpointServiceConfigurations[serviceID]; !ok {
			return VpcEndpointConnectionNotification{}, nil, ErrNotFound
		}
	}
	if vpcEndpointID != "" {
		if _, ok := s.vpcEndpoints[vpcEndpointID]; !ok {
			return VpcEndpointConnectionNotification{}, nil, ErrNotFound
		}
	}

	notification := VpcEndpointConnectionNotification{
		ConnectionNotificationID:    s.nextIDLocked("vpce-nfn"),
		ConnectionNotificationARN:   connectionNotificationARN,
		ConnectionEvents:            connectionEvents,
		ConnectionNotificationState: "Enabled",
		ConnectionNotificationType:  "Topic",
		ServiceID:                   serviceID,
		ServiceRegion:               DefaultRegion,
		VpcEndpointID:               vpcEndpointID,
	}
	s.vpcEndpointConnectionNotifications[notification.ConnectionNotificationID] = &notification
	return notification, trimmedToken, nil
}
