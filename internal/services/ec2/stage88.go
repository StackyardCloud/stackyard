package ec2

import (
	"strings"
	"time"
)

func (s *Service) CancelBundleTask(bundleID string) (BundleTask, error) {
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return BundleTask{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task := s.bundleTasks[bundleID]
	if task == nil {
		return BundleTask{}, ErrNotFound
	}

	task.State = "cancelling"
	task.UpdateTime = time.Now().UTC()
	return cloneBundleTask(task), nil
}
