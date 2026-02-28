package ec2

import "strings"

func (s *Service) CancelExportTask(exportTaskID string) error {
	exportTaskID = strings.TrimSpace(exportTaskID)
	if exportTaskID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelledExportTasks[exportTaskID] = true
	return nil
}
