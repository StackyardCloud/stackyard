package ec2

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

type KeyPair struct {
	ID          string
	Name        string
	Fingerprint string
	Material    string
	Type        string
}

type IamInstanceProfileAssociation struct {
	AssociationID string
	InstanceID    string
	State         string
	Timestamp     time.Time
	ProfileName   string
	ProfileARN    string
}

func (s *Service) CreateKeyPair(keyName string) (KeyPair, error) {
	keyName = strings.TrimSpace(keyName)
	if keyName == "" {
		return KeyPair{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.keyPairs[keyName]; exists {
		return KeyPair{}, ErrAlreadyExists
	}

	material := fmt.Sprintf("-----BEGIN PRIVATE KEY-----\n%s\n-----END PRIVATE KEY-----", strings.Repeat("A", 64))
	fingerprint := fingerprintForKeyMaterial(material)
	keyPair := &KeyPair{
		ID:          s.nextIDLocked("key"),
		Name:        keyName,
		Fingerprint: fingerprint,
		Material:    material,
		Type:        "rsa",
	}
	s.keyPairs[keyName] = keyPair
	return *keyPair, nil
}

func (s *Service) ImportKeyPair(keyName, publicKeyMaterial string) (KeyPair, error) {
	keyName = strings.TrimSpace(keyName)
	publicKeyMaterial = strings.TrimSpace(publicKeyMaterial)
	if keyName == "" || publicKeyMaterial == "" {
		return KeyPair{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.keyPairs[keyName]; exists {
		return KeyPair{}, ErrAlreadyExists
	}
	keyPair := &KeyPair{
		ID:          s.nextIDLocked("key"),
		Name:        keyName,
		Fingerprint: fingerprintForKeyMaterial(publicKeyMaterial),
		Material:    "",
		Type:        "rsa",
	}
	s.keyPairs[keyName] = keyPair
	return *keyPair, nil
}

func (s *Service) DescribeKeyPairs(keyNames, keyPairIDs []string) []KeyPair {
	s.mu.Lock()
	defer s.mu.Unlock()

	nameSet := toStringSet(keyNames)
	idSet := toStringSet(keyPairIDs)
	out := make([]KeyPair, 0, len(s.keyPairs))
	for _, keyPair := range s.keyPairs {
		if len(nameSet) > 0 {
			if _, ok := nameSet[keyPair.Name]; !ok {
				continue
			}
		}
		if len(idSet) > 0 {
			if _, ok := idSet[keyPair.ID]; !ok {
				continue
			}
		}
		out = append(out, *keyPair)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeleteKeyPair(keyName, keyPairID string) error {
	keyName = strings.TrimSpace(keyName)
	keyPairID = strings.TrimSpace(keyPairID)
	if keyName == "" && keyPairID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if keyName != "" {
		if _, exists := s.keyPairs[keyName]; !exists {
			return ErrNotFound
		}
		delete(s.keyPairs, keyName)
		return nil
	}
	for name, keyPair := range s.keyPairs {
		if keyPair.ID == keyPairID {
			delete(s.keyPairs, name)
			return nil
		}
	}
	return ErrNotFound
}

func (s *Service) AssociateIamInstanceProfile(instanceID, profileName, profileARN string) (IamInstanceProfileAssociation, error) {
	instanceID = strings.TrimSpace(instanceID)
	profileName = strings.TrimSpace(profileName)
	profileARN = strings.TrimSpace(profileARN)
	if instanceID == "" {
		return IamInstanceProfileAssociation{}, ErrInvalidParameter
	}
	if profileName == "" && profileARN == "" {
		return IamInstanceProfileAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	instance := s.instances[instanceID]
	if instance == nil {
		return IamInstanceProfileAssociation{}, ErrNotFound
	}
	if instance.StateName == "terminated" {
		return IamInstanceProfileAssociation{}, ErrConflict
	}
	if assocID := s.instanceProfileByInst[instanceID]; assocID != "" {
		if assoc := s.instanceProfileAssocs[assocID]; assoc != nil && assoc.State == "associated" {
			return IamInstanceProfileAssociation{}, ErrConflict
		}
	}
	profileName, profileARN = normalizeInstanceProfile(profileName, profileARN)
	association := &IamInstanceProfileAssociation{
		AssociationID: s.nextIDLocked("iip-assoc"),
		InstanceID:    instanceID,
		State:         "associated",
		Timestamp:     time.Now().UTC(),
		ProfileName:   profileName,
		ProfileARN:    profileARN,
	}
	s.instanceProfileAssocs[association.AssociationID] = association
	s.instanceProfileByInst[instanceID] = association.AssociationID
	return *association, nil
}

func (s *Service) DescribeIamInstanceProfileAssociations(associationIDs, instanceIDs []string) []IamInstanceProfileAssociation {
	s.mu.Lock()
	defer s.mu.Unlock()

	associationIDSet := toStringSet(associationIDs)
	instanceIDSet := toStringSet(instanceIDs)
	out := make([]IamInstanceProfileAssociation, 0, len(s.instanceProfileAssocs))
	for _, association := range s.instanceProfileAssocs {
		if len(associationIDSet) > 0 {
			if _, ok := associationIDSet[association.AssociationID]; !ok {
				continue
			}
		}
		if len(instanceIDSet) > 0 {
			if _, ok := instanceIDSet[association.InstanceID]; !ok {
				continue
			}
		}
		out = append(out, *association)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AssociationID < out[j].AssociationID })
	return out
}

func (s *Service) DisassociateIamInstanceProfile(associationID string) (IamInstanceProfileAssociation, error) {
	associationID = strings.TrimSpace(associationID)
	if associationID == "" {
		return IamInstanceProfileAssociation{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	association := s.instanceProfileAssocs[associationID]
	if association == nil {
		return IamInstanceProfileAssociation{}, ErrNotFound
	}
	association.State = "disassociated"
	association.Timestamp = time.Now().UTC()
	if current := s.instanceProfileByInst[association.InstanceID]; current == associationID {
		delete(s.instanceProfileByInst, association.InstanceID)
	}
	return *association, nil
}

func (s *Service) ReplaceIamInstanceProfileAssociation(associationID, profileName, profileARN string) (IamInstanceProfileAssociation, error) {
	associationID = strings.TrimSpace(associationID)
	profileName = strings.TrimSpace(profileName)
	profileARN = strings.TrimSpace(profileARN)
	if associationID == "" || (profileName == "" && profileARN == "") {
		return IamInstanceProfileAssociation{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	association := s.instanceProfileAssocs[associationID]
	if association == nil {
		return IamInstanceProfileAssociation{}, ErrNotFound
	}
	if association.State != "associated" {
		return IamInstanceProfileAssociation{}, ErrConflict
	}
	profileName, profileARN = normalizeInstanceProfile(profileName, profileARN)
	association.ProfileName = profileName
	association.ProfileARN = profileARN
	association.Timestamp = time.Now().UTC()
	return *association, nil
}

func normalizeInstanceProfile(profileName, profileARN string) (string, string) {
	if profileName == "" && profileARN != "" {
		parts := strings.Split(profileARN, "/")
		profileName = parts[len(parts)-1]
	}
	if profileARN == "" && profileName != "" {
		profileARN = "arn:aws:iam::" + DefaultAccountID + ":instance-profile/" + profileName
	}
	return profileName, profileARN
}

func fingerprintForKeyMaterial(material string) string {
	sum := sha1.Sum([]byte(material))
	parts := make([]string, 0, len(sum))
	for _, b := range sum {
		parts = append(parts, hex.EncodeToString([]byte{b}))
	}
	return strings.Join(parts, ":")
}

func decodePublicKeyMaterial(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return string(decoded)
	}
	return value
}
