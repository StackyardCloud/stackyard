package ec2

import "strings"

func (s *Service) CancelImageLaunchPermission(imageID string) (bool, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelledImageLaunchPermissions[imageID] = true
	return true, nil
}
