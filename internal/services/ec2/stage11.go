package ec2

import "strings"

func (s *Service) EnableEbsEncryptionByDefault() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ebsEncryptionByDefault = true
	return s.ebsEncryptionByDefault
}

func (s *Service) DisableEbsEncryptionByDefault() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ebsEncryptionByDefault = false
	return s.ebsEncryptionByDefault
}

func (s *Service) GetEbsEncryptionByDefault() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.ebsEncryptionByDefault
}

func (s *Service) GetEbsDefaultKmsKeyID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.ebsDefaultKMSKeyID
}

func (s *Service) ModifyEbsDefaultKmsKeyID(kmsKeyID string) (string, error) {
	kmsKeyID = strings.TrimSpace(kmsKeyID)
	if kmsKeyID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ebsDefaultKMSKeyID = kmsKeyID
	return s.ebsDefaultKMSKeyID, nil
}

func (s *Service) ResetEbsDefaultKmsKeyID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ebsDefaultKMSKeyID = "arn:aws:kms:" + DefaultRegion + ":" + DefaultAccountID + ":alias/aws/ebs"
	return s.ebsDefaultKMSKeyID
}
