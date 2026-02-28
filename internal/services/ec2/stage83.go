package ec2

import (
	"fmt"
	"strings"
)

func (s *Service) AssociateEnclaveCertificateIamRole(certificateARN, roleARN string) (EnclaveCertificateRoleAssociation, error) {
	certificateARN = strings.TrimSpace(certificateARN)
	roleARN = strings.TrimSpace(roleARN)
	if certificateARN == "" || roleARN == "" {
		return EnclaveCertificateRoleAssociation{}, ErrInvalidParameter
	}

	association := EnclaveCertificateRoleAssociation{
		AssociatedRoleArn:       roleARN,
		CertificateS3BucketName: "stackyard-enclave-certificates",
		CertificateS3ObjectKey:  fmt.Sprintf("%s/%s", roleARN, certificateARN),
		EncryptionKmsKeyID:      fmt.Sprintf("arn:aws:kms:%s:%s:key/stackyard-enclave", DefaultRegion, DefaultAccountID),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	associationsByRole := s.enclaveCertificateRoleAssociations[certificateARN]
	if associationsByRole == nil {
		associationsByRole = map[string]EnclaveCertificateRoleAssociation{}
		s.enclaveCertificateRoleAssociations[certificateARN] = associationsByRole
	}
	associationsByRole[roleARN] = association
	return association, nil
}
