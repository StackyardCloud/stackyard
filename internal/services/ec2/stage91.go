package ec2

import "strings"

func (s *Service) CancelConversionTask(conversionTaskID string, reasonMessage *string) error {
	conversionTaskID = strings.TrimSpace(conversionTaskID)
	if conversionTaskID == "" {
		return ErrInvalidParameter
	}

	var reason string
	if reasonMessage != nil {
		reason = strings.TrimSpace(*reasonMessage)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.conversionTaskStates[conversionTaskID] = "cancelled"
	if reason != "" {
		s.conversionTaskCancelReasons[conversionTaskID] = reason
	}
	return nil
}
