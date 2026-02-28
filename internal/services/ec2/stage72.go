package ec2

import "strings"

type SnapshotBlockPublicAccessState struct {
	State     string
	ManagedBy string
}

func (s *Service) EnableImageDeregistrationProtection(imageID string, withCooldown *bool) (string, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return "", ErrInvalidParameter
	}
	_ = withCooldown

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return "", ErrNotFound
	}

	image.DeregistrationProtection = "enabled"
	return "true", nil
}

func (s *Service) DisableImageDeregistrationProtection(imageID string) (string, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return "", ErrNotFound
	}

	image.DeregistrationProtection = "disabled"
	return "true", nil
}

func (s *Service) EnableSnapshotBlockPublicAccess(state string) (SnapshotBlockPublicAccessState, error) {
	state = strings.ToLower(strings.TrimSpace(state))
	switch state {
	case "block-all-sharing", "block-new-sharing":
	default:
		return SnapshotBlockPublicAccessState{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshotBlockPublicAccessState.State = state
	s.snapshotBlockPublicAccessState.ManagedBy = "account"
	return s.snapshotBlockPublicAccessState, nil
}

func (s *Service) DisableSnapshotBlockPublicAccess() SnapshotBlockPublicAccessState {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshotBlockPublicAccessState.State = "unblocked"
	s.snapshotBlockPublicAccessState.ManagedBy = "account"
	return s.snapshotBlockPublicAccessState
}

func (s *Service) GetSnapshotBlockPublicAccessState() SnapshotBlockPublicAccessState {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.snapshotBlockPublicAccessState
}
