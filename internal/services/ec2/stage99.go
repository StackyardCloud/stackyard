package ec2

import "strings"

func (s *Service) CopyFpgaImage(
	sourceFpgaImageID string,
	sourceRegion string,
	name *string,
	description *string,
	clientToken *string,
) (string, error) {
	sourceFpgaImageID = strings.TrimSpace(sourceFpgaImageID)
	sourceRegion = strings.TrimSpace(sourceRegion)
	if sourceFpgaImageID == "" || sourceRegion == "" {
		return "", ErrInvalidParameter
	}
	_ = name
	_ = description
	_ = clientToken

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.nextIDLocked("afi"), nil
}
