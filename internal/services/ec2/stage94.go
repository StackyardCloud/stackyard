package ec2

import "strings"

type ImportTaskCancellationResult struct {
	ImportTaskID  string
	PreviousState string
	State         string
}

func (s *Service) CancelImportTask(importTaskID string, cancelReason *string) (ImportTaskCancellationResult, error) {
	importTaskID = strings.TrimSpace(importTaskID)
	if importTaskID == "" {
		return ImportTaskCancellationResult{}, ErrInvalidParameter
	}

	var reason string
	if cancelReason != nil {
		reason = strings.TrimSpace(*cancelReason)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	previousState := s.importTaskStates[importTaskID]
	if previousState == "" {
		previousState = "active"
	}
	currentState := "cancelled"
	s.importTaskStates[importTaskID] = currentState
	if reason != "" {
		s.importTaskCancelReasons[importTaskID] = reason
	}

	return ImportTaskCancellationResult{
		ImportTaskID:  importTaskID,
		PreviousState: previousState,
		State:         currentState,
	}, nil
}
