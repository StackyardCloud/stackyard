package ec2

import (
	"strings"
	"time"
)

type ImageBlockPublicAccessState struct {
	ImageBlockPublicAccessState string
	ManagedBy                   string
}

func (s *Service) EnableImageBlockPublicAccess(imageBlockPublicAccessState string) (ImageBlockPublicAccessState, error) {
	imageBlockPublicAccessState = strings.ToLower(strings.TrimSpace(imageBlockPublicAccessState))
	if imageBlockPublicAccessState != "block-new-sharing" {
		return ImageBlockPublicAccessState{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.imageBlockPublicAccessState.ImageBlockPublicAccessState = imageBlockPublicAccessState
	s.imageBlockPublicAccessState.ManagedBy = "account"
	return s.imageBlockPublicAccessState, nil
}

func (s *Service) DisableImageBlockPublicAccess() ImageBlockPublicAccessState {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.imageBlockPublicAccessState.ImageBlockPublicAccessState = "unblocked"
	s.imageBlockPublicAccessState.ManagedBy = "account"
	return s.imageBlockPublicAccessState
}

func (s *Service) GetImageBlockPublicAccessState() ImageBlockPublicAccessState {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.imageBlockPublicAccessState
}

func (s *Service) EnableImageDeprecation(imageID string, deprecateAt time.Time) (bool, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" || deprecateAt.IsZero() {
		return false, ErrInvalidParameter
	}
	deprecateAt = deprecateAt.UTC()
	if deprecateAt.Before(time.Now().UTC()) {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return false, ErrNotFound
	}

	image.DeprecationTime = cloneTimePointer(&deprecateAt)
	return true, nil
}

func (s *Service) DisableImageDeprecation(imageID string) (bool, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return false, ErrNotFound
	}

	image.DeprecationTime = nil
	return true, nil
}
