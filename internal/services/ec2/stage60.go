package ec2

import (
	"sort"
	"strconv"
	"strings"
)

func (s *Service) DescribeVpcEndpointConnectionNotifications(
	connectionNotificationID *string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VpcEndpointConnectionNotification, *string, error) {
	start := 0
	if nextToken != nil {
		token := strings.TrimSpace(*nextToken)
		if token != "" {
			parsed, err := strconv.Atoi(token)
			if err != nil || parsed < 0 {
				return nil, nil, ErrInvalidParameter
			}
			start = parsed
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var idFilter string
	if connectionNotificationID != nil {
		idFilter = strings.TrimSpace(*connectionNotificationID)
	}

	normalizedFilters := map[string]map[string]struct{}{}
	for name, values := range filters {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		valueSet := map[string]struct{}{}
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			valueSet[value] = struct{}{}
		}
		if len(valueSet) > 0 {
			normalizedFilters[name] = valueSet
		}
	}

	items := make([]VpcEndpointConnectionNotification, 0, len(s.vpcEndpointConnectionNotifications))
	for _, notification := range s.vpcEndpointConnectionNotifications {
		if notification == nil {
			continue
		}
		if idFilter != "" && notification.ConnectionNotificationID != idFilter {
			continue
		}
		if !matchesVpcEndpointConnectionNotificationFilters(notification, normalizedFilters) {
			continue
		}

		items = append(items, VpcEndpointConnectionNotification{
			ConnectionNotificationID:    notification.ConnectionNotificationID,
			ConnectionNotificationARN:   notification.ConnectionNotificationARN,
			ConnectionEvents:            append([]string(nil), notification.ConnectionEvents...),
			ConnectionNotificationState: notification.ConnectionNotificationState,
			ConnectionNotificationType:  notification.ConnectionNotificationType,
			ServiceID:                   notification.ServiceID,
			ServiceRegion:               notification.ServiceRegion,
			VpcEndpointID:               notification.VpcEndpointID,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ConnectionNotificationID < items[j].ConnectionNotificationID
	})

	if start > len(items) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(items)
	if maxResults != nil {
		if *maxResults < 0 {
			return nil, nil, ErrInvalidParameter
		}
		if *maxResults > 0 {
			end = start + int(*maxResults)
			if end > len(items) {
				end = len(items)
			}
		}
	}

	out := append([]VpcEndpointConnectionNotification(nil), items[start:end]...)
	var outputToken *string
	if end < len(items) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return out, outputToken, nil
}

func matchesVpcEndpointConnectionNotificationFilters(
	notification *VpcEndpointConnectionNotification,
	filters map[string]map[string]struct{},
) bool {
	for name, values := range filters {
		switch name {
		case "connection-notification-arn":
			if _, ok := values[notification.ConnectionNotificationARN]; !ok {
				return false
			}
		case "connection-notification-id":
			if _, ok := values[notification.ConnectionNotificationID]; !ok {
				return false
			}
		case "connection-notification-state":
			if _, ok := values[notification.ConnectionNotificationState]; !ok {
				return false
			}
		case "connection-notification-type":
			if _, ok := values[notification.ConnectionNotificationType]; !ok {
				return false
			}
		case "service-id":
			if _, ok := values[notification.ServiceID]; !ok {
				return false
			}
		case "vpc-endpoint-id":
			if _, ok := values[notification.VpcEndpointID]; !ok {
				return false
			}
		}
	}
	return true
}
