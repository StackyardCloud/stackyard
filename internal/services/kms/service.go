package kms

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter     = errors.New("invalid parameter")
	ErrNotFound             = errors.New("resource not found")
	ErrAlreadyExists        = errors.New("resource already exists")
	ErrDisabled             = errors.New("key disabled")
	ErrInvalidState         = errors.New("invalid key state")
	ErrInvalidCiphertext    = errors.New("invalid ciphertext")
	ErrIncorrectKey         = errors.New("incorrect key")
	ErrUnsupportedAlgorithm = errors.New("unsupported algorithm")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"

	defaultListLimit = int32(100)
	maxListLimit     = int32(1000)
)

type Key struct {
	AWSAccountID           string
	KeyID                  string
	Arn                    string
	Description            string
	Enabled                bool
	KeyState               string
	KeyUsage               string
	KeySpec                string
	CustomerMasterKeySpec  string
	Origin                 string
	KeyManager             string
	CreationDate           time.Time
	DeletionDate           *time.Time
	EncryptionAlgorithms   []string
	SigningAlgorithms      []string
	MacAlgorithms          []string
	KeyAgreementAlgorithms []string
	RotationEnabled        bool
	MultiRegion            bool
	PrimaryRegion          string
	material               []byte
	publicKey              []byte
	tags                   map[string]string
	policies               map[string]string
	rotationHistory        []KeyRotationEntry
	importedKeyExpiration  *time.Time
}

type Alias struct {
	AliasName    string
	AliasArn     string
	TargetKeyID  string
	TargetKeyArn string
	CreatedAt    time.Time
}

type KeyListEntry struct {
	KeyID  string
	KeyArn string
}

type EncryptOutput struct {
	CiphertextBlob      []byte
	KeyID               string
	EncryptionAlgorithm string
}

type DecryptOutput struct {
	Plaintext           []byte
	KeyID               string
	EncryptionAlgorithm string
}

type ReEncryptOutput struct {
	CiphertextBlob                 []byte
	KeyID                          string
	SourceKeyID                    string
	SourceEncryptionAlgorithm      string
	DestinationEncryptionAlgorithm string
}

type DataKeyOutput struct {
	CiphertextBlob []byte
	Plaintext      []byte
	KeyID          string
}

type DataKeyPairOutput struct {
	KeyID                    string
	KeyPairSpec              string
	PublicKey                []byte
	PrivateKeyPlaintext      []byte
	PrivateKeyCiphertextBlob []byte
}

type PublicKeyOutput struct {
	KeyID                  string
	PublicKey              []byte
	KeySpec                string
	KeyUsage               string
	EncryptionAlgorithms   []string
	SigningAlgorithms      []string
	KeyAgreementAlgorithms []string
}

type SignOutput struct {
	KeyID            string
	Signature        []byte
	SigningAlgorithm string
}

type VerifyOutput struct {
	KeyID            string
	SignatureValid   bool
	SigningAlgorithm string
}

type MacOutput struct {
	KeyID        string
	Mac          []byte
	MacAlgorithm string
}

type VerifyMacOutput struct {
	KeyID        string
	MacValid     bool
	MacAlgorithm string
}

type DeriveSharedSecretOutput struct {
	KeyID                 string
	SharedSecret          []byte
	KeyAgreementAlgorithm string
}

type Tag struct {
	TagKey   string
	TagValue string
}

type Grant struct {
	GrantID           string
	GrantToken        string
	KeyID             string
	Name              string
	GranteePrincipal  string
	RetiringPrincipal string
	Operations        []string
	IssuingAccount    string
	CreationDate      time.Time
}

type KeyRotationEntry struct {
	RotationDate time.Time
	RotationType string
}

type ImportParametersOutput struct {
	KeyID             string
	ImportToken       []byte
	PublicKey         []byte
	ParametersValidTo time.Time
	WrappingAlgorithm string
	WrappingKeySpec   string
}

type CustomKeyStore struct {
	CustomKeyStoreID   string
	CustomKeyStoreName string
	CloudHsmClusterID  string
	ConnectionState    string
	CustomKeyStoreType string
	XksProxyURI        string
	CreationDate       time.Time
}

type importTokenRecord struct {
	KeyID     string
	ExpiresAt time.Time
}

type Service struct {
	mu              sync.Mutex
	seq             uint64
	grantSeq        uint64
	customStoreSeq  uint64
	keys            map[string]*Key
	aliases         map[string]*Alias
	grants          map[string]*Grant
	importTokens    map[string]importTokenRecord
	customKeyStores map[string]*CustomKeyStore
}

func NewService() *Service {
	return &Service{
		keys:            map[string]*Key{},
		aliases:         map[string]*Alias{},
		grants:          map[string]*Grant{},
		importTokens:    map[string]importTokenRecord{},
		customKeyStores: map[string]*CustomKeyStore{},
	}
}

func (s *Service) CreateKey(description, keyUsage, keySpec string) (Key, error) {
	usage := strings.ToUpper(strings.TrimSpace(keyUsage))
	if usage == "" {
		usage = "ENCRYPT_DECRYPT"
	}

	spec := strings.ToUpper(strings.TrimSpace(keySpec))
	if spec == "" {
		switch usage {
		case "ENCRYPT_DECRYPT":
			spec = "SYMMETRIC_DEFAULT"
		case "SIGN_VERIFY":
			spec = "RSA_2048"
		case "GENERATE_VERIFY_MAC":
			spec = "HMAC_256"
		case "KEY_AGREEMENT":
			spec = "ECC_NIST_P256"
		default:
			return Key{}, ErrInvalidParameter
		}
	}

	if !validKeyUsage(usage) {
		return Key{}, ErrInvalidParameter
	}
	if !validKeySpecForUsage(spec, usage) {
		return Key{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextKeyIDLocked()
	arn := keyARN(id)
	now := time.Now().UTC()

	material, err := randomBytes(32)
	if err != nil {
		return Key{}, err
	}
	pub, err := randomBytes(64)
	if err != nil {
		return Key{}, err
	}

	key := &Key{
		AWSAccountID:           DefaultAccountID,
		KeyID:                  id,
		Arn:                    arn,
		Description:            strings.TrimSpace(description),
		Enabled:                true,
		KeyState:               "Enabled",
		KeyUsage:               usage,
		KeySpec:                spec,
		CustomerMasterKeySpec:  spec,
		Origin:                 "AWS_KMS",
		KeyManager:             "CUSTOMER",
		CreationDate:           now,
		EncryptionAlgorithms:   defaultEncryptionAlgorithms(spec, usage),
		SigningAlgorithms:      defaultSigningAlgorithms(spec, usage),
		MacAlgorithms:          defaultMacAlgorithms(spec, usage),
		KeyAgreementAlgorithms: defaultKeyAgreementAlgorithms(spec, usage),
		RotationEnabled:        false,
		MultiRegion:            false,
		PrimaryRegion:          DefaultRegion,
		material:               material,
		publicKey:              pub,
		tags:                   map[string]string{},
		policies:               map[string]string{"default": "{}"},
		rotationHistory:        []KeyRotationEntry{},
	}
	s.keys[id] = key
	return cloneKey(*key), nil
}

func (s *Service) DescribeKey(keyRef string) (Key, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return Key{}, err
	}
	return cloneKey(*key), nil
}

func (s *Service) ListKeys(marker string, limit int32) ([]KeyListEntry, string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.keys))
	for id := range s.keys {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	offset, err := parseMarker(marker, len(ids))
	if err != nil {
		return nil, "", false, err
	}
	pageSize := normalizeLimit(limit)
	end := offset + int(pageSize)
	if end > len(ids) {
		end = len(ids)
	}

	items := make([]KeyListEntry, 0, end-offset)
	for _, id := range ids[offset:end] {
		key := s.keys[id]
		items = append(items, KeyListEntry{KeyID: key.KeyID, KeyArn: key.Arn})
	}

	truncated := end < len(ids)
	nextMarker := ""
	if truncated {
		nextMarker = strconv.Itoa(end)
	}
	return items, nextMarker, truncated, nil
}

func (s *Service) EnableKey(keyRef string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	if key.KeyState == "PendingDeletion" {
		return ErrInvalidState
	}
	key.Enabled = true
	key.KeyState = "Enabled"
	return nil
}

func (s *Service) DisableKey(keyRef string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	if key.KeyState == "PendingDeletion" {
		return ErrInvalidState
	}
	key.Enabled = false
	key.KeyState = "Disabled"
	return nil
}

func (s *Service) ScheduleKeyDeletion(keyRef string, pendingWindowInDays int32) (string, time.Time, error) {
	window := pendingWindowInDays
	if window == 0 {
		window = 30
	}
	if window < 7 || window > 30 {
		return "", time.Time{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return "", time.Time{}, err
	}
	if key.KeyState == "PendingDeletion" {
		return "", time.Time{}, ErrInvalidState
	}

	deletionDate := time.Now().UTC().Add(time.Duration(window) * 24 * time.Hour)
	key.DeletionDate = &deletionDate
	key.Enabled = false
	key.KeyState = "PendingDeletion"
	return key.KeyID, deletionDate, nil
}

func (s *Service) CancelKeyDeletion(keyRef string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return "", err
	}
	if key.KeyState != "PendingDeletion" {
		return "", ErrInvalidState
	}

	key.DeletionDate = nil
	key.Enabled = false
	key.KeyState = "Disabled"
	return key.KeyID, nil
}

func (s *Service) UpdateKeyDescription(keyRef, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	key.Description = strings.TrimSpace(description)
	return nil
}

func (s *Service) CreateAlias(aliasName, targetKeyRef string) error {
	aliasName = strings.TrimSpace(aliasName)
	if !validAliasName(aliasName) {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.aliases[aliasName]; exists {
		return ErrAlreadyExists
	}
	key, err := s.resolveKeyLocked(targetKeyRef)
	if err != nil {
		return err
	}
	if key.KeyState == "PendingDeletion" {
		return ErrInvalidState
	}

	s.aliases[aliasName] = &Alias{
		AliasName:    aliasName,
		AliasArn:     aliasARN(aliasName),
		TargetKeyID:  key.KeyID,
		TargetKeyArn: key.Arn,
		CreatedAt:    time.Now().UTC(),
	}
	return nil
}

func (s *Service) UpdateAlias(aliasName, targetKeyRef string) error {
	aliasName = strings.TrimSpace(aliasName)
	if !validAliasName(aliasName) {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.aliases[aliasName]
	if !exists {
		return ErrNotFound
	}
	key, err := s.resolveKeyLocked(targetKeyRef)
	if err != nil {
		return err
	}
	if key.KeyState == "PendingDeletion" {
		return ErrInvalidState
	}

	entry.TargetKeyID = key.KeyID
	entry.TargetKeyArn = key.Arn
	return nil
}

func (s *Service) DeleteAlias(aliasName string) error {
	aliasName = strings.TrimSpace(aliasName)
	if !validAliasName(aliasName) {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.aliases[aliasName]; !exists {
		return ErrNotFound
	}
	delete(s.aliases, aliasName)
	return nil
}

func (s *Service) ListAliases(keyRef, marker string, limit int32) ([]Alias, string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filterKeyID := ""
	if strings.TrimSpace(keyRef) != "" {
		key, err := s.resolveKeyLocked(keyRef)
		if err != nil {
			return nil, "", false, err
		}
		filterKeyID = key.KeyID
	}

	aliasNames := make([]string, 0, len(s.aliases))
	for name := range s.aliases {
		if filterKeyID != "" && s.aliases[name].TargetKeyID != filterKeyID {
			continue
		}
		aliasNames = append(aliasNames, name)
	}
	sort.Strings(aliasNames)

	offset, err := parseMarker(marker, len(aliasNames))
	if err != nil {
		return nil, "", false, err
	}
	pageSize := normalizeLimit(limit)
	end := offset + int(pageSize)
	if end > len(aliasNames) {
		end = len(aliasNames)
	}

	items := make([]Alias, 0, end-offset)
	for _, name := range aliasNames[offset:end] {
		items = append(items, cloneAlias(*s.aliases[name]))
	}

	truncated := end < len(aliasNames)
	nextMarker := ""
	if truncated {
		nextMarker = strconv.Itoa(end)
	}
	return items, nextMarker, truncated, nil
}

func (s *Service) Encrypt(keyRef string, plaintext []byte, encryptionAlgorithm string) (EncryptOutput, error) {
	algorithm := strings.ToUpper(strings.TrimSpace(encryptionAlgorithm))

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return EncryptOutput{}, err
	}
	if err := ensureKeyUsableForEncrypt(key); err != nil {
		return EncryptOutput{}, err
	}
	if len(plaintext) == 0 {
		return EncryptOutput{}, ErrInvalidParameter
	}
	if algorithm == "" {
		algorithm = defaultEncryptAlgorithmForKey(key)
	}
	if !containsString(key.EncryptionAlgorithms, algorithm) {
		return EncryptOutput{}, ErrUnsupportedAlgorithm
	}

	ciphertext, err := buildCiphertext(key, plaintext, algorithm)
	if err != nil {
		return EncryptOutput{}, err
	}
	return EncryptOutput{CiphertextBlob: ciphertext, KeyID: key.Arn, EncryptionAlgorithm: algorithm}, nil
}

func (s *Service) Decrypt(ciphertextBlob []byte, keyRef, encryptionAlgorithm string) (DecryptOutput, error) {
	algorithmHint := strings.ToUpper(strings.TrimSpace(encryptionAlgorithm))

	s.mu.Lock()
	defer s.mu.Unlock()

	record, err := parseCiphertextRecord(ciphertextBlob, s.keys)
	if err != nil {
		return DecryptOutput{}, err
	}

	key := s.keys[record.KeyID]
	if key == nil {
		return DecryptOutput{}, ErrInvalidCiphertext
	}
	if err := ensureKeyUsableForEncrypt(key); err != nil {
		return DecryptOutput{}, err
	}
	if strings.TrimSpace(keyRef) != "" {
		requested, err := s.resolveKeyLocked(keyRef)
		if err != nil {
			return DecryptOutput{}, err
		}
		if requested.KeyID != key.KeyID {
			return DecryptOutput{}, ErrIncorrectKey
		}
	}
	if algorithmHint != "" && algorithmHint != record.Algorithm {
		return DecryptOutput{}, ErrInvalidCiphertext
	}

	return DecryptOutput{Plaintext: append([]byte(nil), record.Plaintext...), KeyID: key.Arn, EncryptionAlgorithm: record.Algorithm}, nil
}

func (s *Service) ReEncrypt(ciphertextBlob []byte, sourceKeyRef, destinationKeyRef, sourceEncryptionAlgorithm, destinationEncryptionAlgorithm string) (ReEncryptOutput, error) {
	if strings.TrimSpace(destinationKeyRef) == "" {
		return ReEncryptOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, err := parseCiphertextRecord(ciphertextBlob, s.keys)
	if err != nil {
		return ReEncryptOutput{}, err
	}
	sourceKey := s.keys[record.KeyID]
	if sourceKey == nil {
		return ReEncryptOutput{}, ErrInvalidCiphertext
	}
	if err := ensureKeyUsableForEncrypt(sourceKey); err != nil {
		return ReEncryptOutput{}, err
	}

	if strings.TrimSpace(sourceKeyRef) != "" {
		requestedSource, err := s.resolveKeyLocked(sourceKeyRef)
		if err != nil {
			return ReEncryptOutput{}, err
		}
		if requestedSource.KeyID != sourceKey.KeyID {
			return ReEncryptOutput{}, ErrIncorrectKey
		}
	}

	sourceAlgHint := strings.ToUpper(strings.TrimSpace(sourceEncryptionAlgorithm))
	if sourceAlgHint != "" && sourceAlgHint != record.Algorithm {
		return ReEncryptOutput{}, ErrInvalidCiphertext
	}

	destinationKey, err := s.resolveKeyLocked(destinationKeyRef)
	if err != nil {
		return ReEncryptOutput{}, err
	}
	if err := ensureKeyUsableForEncrypt(destinationKey); err != nil {
		return ReEncryptOutput{}, err
	}
	destinationAlg := strings.ToUpper(strings.TrimSpace(destinationEncryptionAlgorithm))
	if destinationAlg == "" {
		destinationAlg = defaultEncryptAlgorithmForKey(destinationKey)
	}
	if !containsString(destinationKey.EncryptionAlgorithms, destinationAlg) {
		return ReEncryptOutput{}, ErrUnsupportedAlgorithm
	}

	ciphertext, err := buildCiphertext(destinationKey, record.Plaintext, destinationAlg)
	if err != nil {
		return ReEncryptOutput{}, err
	}

	return ReEncryptOutput{
		CiphertextBlob:                 ciphertext,
		KeyID:                          destinationKey.Arn,
		SourceKeyID:                    sourceKey.Arn,
		SourceEncryptionAlgorithm:      record.Algorithm,
		DestinationEncryptionAlgorithm: destinationAlg,
	}, nil
}

func (s *Service) GenerateDataKey(keyRef, keySpec string, numberOfBytes int32) (DataKeyOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return DataKeyOutput{}, err
	}
	if err := ensureKeyUsableForEncrypt(key); err != nil {
		return DataKeyOutput{}, err
	}

	size, err := resolveDataKeySize(strings.ToUpper(strings.TrimSpace(keySpec)), numberOfBytes)
	if err != nil {
		return DataKeyOutput{}, err
	}
	plaintext, err := randomBytes(size)
	if err != nil {
		return DataKeyOutput{}, err
	}
	ciphertext, err := buildCiphertext(key, plaintext, defaultEncryptAlgorithmForKey(key))
	if err != nil {
		return DataKeyOutput{}, err
	}
	return DataKeyOutput{KeyID: key.Arn, Plaintext: plaintext, CiphertextBlob: ciphertext}, nil
}

func (s *Service) GenerateDataKeyWithoutPlaintext(keyRef, keySpec string, numberOfBytes int32) (DataKeyOutput, error) {
	out, err := s.GenerateDataKey(keyRef, keySpec, numberOfBytes)
	if err != nil {
		return DataKeyOutput{}, err
	}
	out.Plaintext = nil
	return out, nil
}

func (s *Service) GenerateDataKeyPair(keyRef, keyPairSpec string) (DataKeyPairOutput, error) {
	spec := strings.ToUpper(strings.TrimSpace(keyPairSpec))
	if spec == "" {
		spec = "RSA_2048"
	}
	privateSize, publicSize, err := keyPairSizes(spec)
	if err != nil {
		return DataKeyPairOutput{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return DataKeyPairOutput{}, err
	}
	if err := ensureKeyUsableForEncrypt(key); err != nil {
		return DataKeyPairOutput{}, err
	}

	privateKey, err := randomBytes(privateSize)
	if err != nil {
		return DataKeyPairOutput{}, err
	}
	publicKey, err := randomBytes(publicSize)
	if err != nil {
		return DataKeyPairOutput{}, err
	}
	ciphertext, err := buildCiphertext(key, privateKey, defaultEncryptAlgorithmForKey(key))
	if err != nil {
		return DataKeyPairOutput{}, err
	}

	return DataKeyPairOutput{
		KeyID:                    key.Arn,
		KeyPairSpec:              spec,
		PublicKey:                publicKey,
		PrivateKeyPlaintext:      privateKey,
		PrivateKeyCiphertextBlob: ciphertext,
	}, nil
}

func (s *Service) GenerateDataKeyPairWithoutPlaintext(keyRef, keyPairSpec string) (DataKeyPairOutput, error) {
	out, err := s.GenerateDataKeyPair(keyRef, keyPairSpec)
	if err != nil {
		return DataKeyPairOutput{}, err
	}
	out.PrivateKeyPlaintext = nil
	return out, nil
}

func (s *Service) GenerateRandom(numberOfBytes int32) ([]byte, error) {
	n := numberOfBytes
	if n == 0 {
		n = 32
	}
	if n < 1 || n > 1024 {
		return nil, ErrInvalidParameter
	}
	return randomBytes(int(n))
}

func (s *Service) GetPublicKey(keyRef string) (PublicKeyOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return PublicKeyOutput{}, err
	}
	if key.KeyState == "PendingDeletion" {
		return PublicKeyOutput{}, ErrInvalidState
	}
	if len(key.publicKey) == 0 {
		pub, err := randomBytes(64)
		if err != nil {
			return PublicKeyOutput{}, err
		}
		key.publicKey = pub
	}

	return PublicKeyOutput{
		KeyID:                  key.Arn,
		PublicKey:              append([]byte(nil), key.publicKey...),
		KeySpec:                key.KeySpec,
		KeyUsage:               key.KeyUsage,
		EncryptionAlgorithms:   append([]string(nil), key.EncryptionAlgorithms...),
		SigningAlgorithms:      append([]string(nil), key.SigningAlgorithms...),
		KeyAgreementAlgorithms: append([]string(nil), key.KeyAgreementAlgorithms...),
	}, nil
}

func (s *Service) Sign(keyRef string, message []byte, signingAlgorithm string) (SignOutput, error) {
	if len(message) == 0 {
		return SignOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return SignOutput{}, err
	}
	if key.KeyUsage != "SIGN_VERIFY" {
		return SignOutput{}, ErrInvalidState
	}
	if !key.Enabled {
		return SignOutput{}, ErrDisabled
	}
	if key.KeyState == "PendingDeletion" {
		return SignOutput{}, ErrInvalidState
	}

	algorithm := strings.ToUpper(strings.TrimSpace(signingAlgorithm))
	if algorithm == "" {
		if len(key.SigningAlgorithms) == 0 {
			return SignOutput{}, ErrUnsupportedAlgorithm
		}
		algorithm = key.SigningAlgorithms[0]
	}
	if !containsString(key.SigningAlgorithms, algorithm) {
		return SignOutput{}, ErrUnsupportedAlgorithm
	}

	signature := computeHMAC(key.material, algorithm, message)
	return SignOutput{KeyID: key.Arn, Signature: signature, SigningAlgorithm: algorithm}, nil
}

func (s *Service) Verify(keyRef string, message, signature []byte, signingAlgorithm string) (VerifyOutput, error) {
	if len(message) == 0 || len(signature) == 0 {
		return VerifyOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return VerifyOutput{}, err
	}
	if key.KeyUsage != "SIGN_VERIFY" {
		return VerifyOutput{}, ErrInvalidState
	}
	if !key.Enabled {
		return VerifyOutput{}, ErrDisabled
	}
	if key.KeyState == "PendingDeletion" {
		return VerifyOutput{}, ErrInvalidState
	}

	algorithm := strings.ToUpper(strings.TrimSpace(signingAlgorithm))
	if algorithm == "" {
		if len(key.SigningAlgorithms) == 0 {
			return VerifyOutput{}, ErrUnsupportedAlgorithm
		}
		algorithm = key.SigningAlgorithms[0]
	}
	if !containsString(key.SigningAlgorithms, algorithm) {
		return VerifyOutput{}, ErrUnsupportedAlgorithm
	}

	expected := computeHMAC(key.material, algorithm, message)
	return VerifyOutput{KeyID: key.Arn, SignatureValid: hmac.Equal(signature, expected), SigningAlgorithm: algorithm}, nil
}

func (s *Service) GenerateMac(keyRef string, message []byte, macAlgorithm string) (MacOutput, error) {
	if len(message) == 0 {
		return MacOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return MacOutput{}, err
	}
	if key.KeyUsage != "GENERATE_VERIFY_MAC" {
		return MacOutput{}, ErrInvalidState
	}
	if !key.Enabled {
		return MacOutput{}, ErrDisabled
	}
	if key.KeyState == "PendingDeletion" {
		return MacOutput{}, ErrInvalidState
	}

	algorithm := strings.ToUpper(strings.TrimSpace(macAlgorithm))
	if algorithm == "" {
		if len(key.MacAlgorithms) == 0 {
			return MacOutput{}, ErrUnsupportedAlgorithm
		}
		algorithm = key.MacAlgorithms[0]
	}
	if !containsString(key.MacAlgorithms, algorithm) {
		return MacOutput{}, ErrUnsupportedAlgorithm
	}

	mac := computeHMAC(key.material, algorithm, message)
	return MacOutput{KeyID: key.Arn, Mac: mac, MacAlgorithm: algorithm}, nil
}

func (s *Service) VerifyMac(keyRef string, message, mac []byte, macAlgorithm string) (VerifyMacOutput, error) {
	if len(message) == 0 || len(mac) == 0 {
		return VerifyMacOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return VerifyMacOutput{}, err
	}
	if key.KeyUsage != "GENERATE_VERIFY_MAC" {
		return VerifyMacOutput{}, ErrInvalidState
	}
	if !key.Enabled {
		return VerifyMacOutput{}, ErrDisabled
	}
	if key.KeyState == "PendingDeletion" {
		return VerifyMacOutput{}, ErrInvalidState
	}

	algorithm := strings.ToUpper(strings.TrimSpace(macAlgorithm))
	if algorithm == "" {
		if len(key.MacAlgorithms) == 0 {
			return VerifyMacOutput{}, ErrUnsupportedAlgorithm
		}
		algorithm = key.MacAlgorithms[0]
	}
	if !containsString(key.MacAlgorithms, algorithm) {
		return VerifyMacOutput{}, ErrUnsupportedAlgorithm
	}

	expected := computeHMAC(key.material, algorithm, message)
	return VerifyMacOutput{KeyID: key.Arn, MacValid: hmac.Equal(mac, expected), MacAlgorithm: algorithm}, nil
}

func (s *Service) DeriveSharedSecret(keyRef string, peerPublicKey []byte, keyAgreementAlgorithm string) (DeriveSharedSecretOutput, error) {
	if len(peerPublicKey) == 0 {
		return DeriveSharedSecretOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return DeriveSharedSecretOutput{}, err
	}
	if key.KeyUsage != "KEY_AGREEMENT" {
		return DeriveSharedSecretOutput{}, ErrInvalidState
	}
	if !key.Enabled {
		return DeriveSharedSecretOutput{}, ErrDisabled
	}
	if key.KeyState == "PendingDeletion" {
		return DeriveSharedSecretOutput{}, ErrInvalidState
	}

	algorithm := strings.ToUpper(strings.TrimSpace(keyAgreementAlgorithm))
	if algorithm == "" {
		if len(key.KeyAgreementAlgorithms) == 0 {
			return DeriveSharedSecretOutput{}, ErrUnsupportedAlgorithm
		}
		algorithm = key.KeyAgreementAlgorithms[0]
	}
	if !containsString(key.KeyAgreementAlgorithms, algorithm) {
		return DeriveSharedSecretOutput{}, ErrUnsupportedAlgorithm
	}

	h := sha256.New()
	h.Write(key.material)
	h.Write(peerPublicKey)
	secret := h.Sum(nil)

	return DeriveSharedSecretOutput{KeyID: key.Arn, SharedSecret: secret, KeyAgreementAlgorithm: algorithm}, nil
}

func (s *Service) PutKeyPolicy(keyRef, policyName, policy string) error {
	name := strings.TrimSpace(policyName)
	if name == "" {
		name = "default"
	}
	policy = strings.TrimSpace(policy)
	if policy == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	if key.policies == nil {
		key.policies = map[string]string{}
	}
	key.policies[name] = policy
	return nil
}

func (s *Service) GetKeyPolicy(keyRef, policyName string) (string, string, error) {
	name := strings.TrimSpace(policyName)
	if name == "" {
		name = "default"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return "", "", err
	}
	policy, ok := key.policies[name]
	if !ok {
		return "", "", ErrNotFound
	}
	return name, policy, nil
}

func (s *Service) ListKeyPolicies(keyRef, marker string, limit int32) ([]string, string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return nil, "", false, err
	}

	names := make([]string, 0, len(key.policies))
	for name := range key.policies {
		names = append(names, name)
	}
	sort.Strings(names)

	offset, err := parseMarker(marker, len(names))
	if err != nil {
		return nil, "", false, err
	}
	pageSize := normalizeLimit(limit)
	end := offset + int(pageSize)
	if end > len(names) {
		end = len(names)
	}

	items := append([]string(nil), names[offset:end]...)
	truncated := end < len(names)
	nextMarker := ""
	if truncated {
		nextMarker = strconv.Itoa(end)
	}
	return items, nextMarker, truncated, nil
}

func (s *Service) TagResource(keyRef string, tags []Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	if key.tags == nil {
		key.tags = map[string]string{}
	}
	for _, tag := range tags {
		k := strings.TrimSpace(tag.TagKey)
		if k == "" {
			return ErrInvalidParameter
		}
		key.tags[k] = strings.TrimSpace(tag.TagValue)
	}
	return nil
}

func (s *Service) UntagResource(keyRef string, tagKeys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	for _, raw := range tagKeys {
		k := strings.TrimSpace(raw)
		if k == "" {
			continue
		}
		delete(key.tags, k)
	}
	return nil
}

func (s *Service) ListResourceTags(keyRef, marker string, limit int32) ([]Tag, string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return nil, "", false, err
	}

	names := make([]string, 0, len(key.tags))
	for name := range key.tags {
		names = append(names, name)
	}
	sort.Strings(names)

	offset, err := parseMarker(marker, len(names))
	if err != nil {
		return nil, "", false, err
	}
	pageSize := normalizeLimit(limit)
	end := offset + int(pageSize)
	if end > len(names) {
		end = len(names)
	}

	items := make([]Tag, 0, end-offset)
	for _, name := range names[offset:end] {
		items = append(items, Tag{TagKey: name, TagValue: key.tags[name]})
	}

	truncated := end < len(names)
	nextMarker := ""
	if truncated {
		nextMarker = strconv.Itoa(end)
	}
	return items, nextMarker, truncated, nil
}

func (s *Service) CreateGrant(keyRef, name, granteePrincipal, retiringPrincipal string, operations []string) (Grant, error) {
	granteePrincipal = strings.TrimSpace(granteePrincipal)
	if granteePrincipal == "" {
		return Grant{}, ErrInvalidParameter
	}
	if len(operations) == 0 {
		return Grant{}, ErrInvalidParameter
	}

	normalizedOps := make([]string, 0, len(operations))
	for _, op := range operations {
		op = strings.TrimSpace(op)
		if op == "" {
			continue
		}
		normalizedOps = append(normalizedOps, op)
	}
	if len(normalizedOps) == 0 {
		return Grant{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return Grant{}, err
	}
	if key.KeyState == "PendingDeletion" {
		return Grant{}, ErrInvalidState
	}

	grantID := s.nextGrantIDLocked()
	grantTokenBytes, err := randomBytes(18)
	if err != nil {
		return Grant{}, err
	}
	grant := &Grant{
		GrantID:           grantID,
		GrantToken:        base64.RawStdEncoding.EncodeToString(grantTokenBytes),
		KeyID:             key.KeyID,
		Name:              strings.TrimSpace(name),
		GranteePrincipal:  granteePrincipal,
		RetiringPrincipal: strings.TrimSpace(retiringPrincipal),
		Operations:        append([]string(nil), normalizedOps...),
		IssuingAccount:    DefaultAccountID,
		CreationDate:      time.Now().UTC(),
	}
	s.grants[grantID] = grant
	return cloneGrant(*grant), nil
}

func (s *Service) ListGrants(keyRef, marker string, limit int32, granteePrincipal string) ([]Grant, string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return nil, "", false, err
	}

	filterPrincipal := strings.TrimSpace(granteePrincipal)
	grantIDs := make([]string, 0, len(s.grants))
	for id, grant := range s.grants {
		if grant.KeyID != key.KeyID {
			continue
		}
		if filterPrincipal != "" && grant.GranteePrincipal != filterPrincipal {
			continue
		}
		grantIDs = append(grantIDs, id)
	}
	sort.Strings(grantIDs)

	offset, err := parseMarker(marker, len(grantIDs))
	if err != nil {
		return nil, "", false, err
	}
	pageSize := normalizeLimit(limit)
	end := offset + int(pageSize)
	if end > len(grantIDs) {
		end = len(grantIDs)
	}

	items := make([]Grant, 0, end-offset)
	for _, id := range grantIDs[offset:end] {
		items = append(items, cloneGrant(*s.grants[id]))
	}

	truncated := end < len(grantIDs)
	nextMarker := ""
	if truncated {
		nextMarker = strconv.Itoa(end)
	}
	return items, nextMarker, truncated, nil
}

func (s *Service) ListRetirableGrants(retiringPrincipal, marker string, limit int32) ([]Grant, string, bool, error) {
	retiringPrincipal = strings.TrimSpace(retiringPrincipal)
	if retiringPrincipal == "" {
		return nil, "", false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	grantIDs := make([]string, 0, len(s.grants))
	for id, grant := range s.grants {
		if grant.RetiringPrincipal != retiringPrincipal {
			continue
		}
		grantIDs = append(grantIDs, id)
	}
	sort.Strings(grantIDs)

	offset, err := parseMarker(marker, len(grantIDs))
	if err != nil {
		return nil, "", false, err
	}
	pageSize := normalizeLimit(limit)
	end := offset + int(pageSize)
	if end > len(grantIDs) {
		end = len(grantIDs)
	}

	items := make([]Grant, 0, end-offset)
	for _, id := range grantIDs[offset:end] {
		items = append(items, cloneGrant(*s.grants[id]))
	}

	truncated := end < len(grantIDs)
	nextMarker := ""
	if truncated {
		nextMarker = strconv.Itoa(end)
	}
	return items, nextMarker, truncated, nil
}

func (s *Service) RetireGrant(grantID, keyRef string) error {
	grantID = strings.TrimSpace(grantID)
	if grantID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	grant := s.grants[grantID]
	if grant == nil {
		return ErrNotFound
	}
	if strings.TrimSpace(keyRef) != "" {
		key, err := s.resolveKeyLocked(keyRef)
		if err != nil {
			return err
		}
		if key.KeyID != grant.KeyID {
			return ErrIncorrectKey
		}
	}
	delete(s.grants, grantID)
	return nil
}

func (s *Service) RevokeGrant(keyRef, grantID string) error {
	grantID = strings.TrimSpace(grantID)
	if grantID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	grant := s.grants[grantID]
	if grant == nil {
		return ErrNotFound
	}
	if grant.KeyID != key.KeyID {
		return ErrIncorrectKey
	}
	delete(s.grants, grantID)
	return nil
}

func (s *Service) EnableKeyRotation(keyRef string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	if !isRotatableKey(key) {
		return ErrInvalidState
	}
	key.RotationEnabled = true
	return nil
}

func (s *Service) DisableKeyRotation(keyRef string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	if !isRotatableKey(key) {
		return ErrInvalidState
	}
	key.RotationEnabled = false
	return nil
}

func (s *Service) GetKeyRotationStatus(keyRef string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return false, err
	}
	if !isRotatableKey(key) {
		return false, ErrInvalidState
	}
	return key.RotationEnabled, nil
}

func (s *Service) RotateKeyOnDemand(keyRef string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return "", err
	}
	if !isRotatableKey(key) {
		return "", ErrInvalidState
	}
	if !key.Enabled {
		return "", ErrDisabled
	}
	if key.KeyState == "PendingDeletion" {
		return "", ErrInvalidState
	}

	material, err := randomBytes(32)
	if err != nil {
		return "", err
	}
	key.material = material
	key.rotationHistory = append(key.rotationHistory, KeyRotationEntry{
		RotationDate: time.Now().UTC(),
		RotationType: "ON_DEMAND",
	})
	return key.KeyID, nil
}

func (s *Service) ListKeyRotations(keyRef, marker string, limit int32) ([]KeyRotationEntry, string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return nil, "", false, err
	}

	offset, err := parseMarker(marker, len(key.rotationHistory))
	if err != nil {
		return nil, "", false, err
	}
	pageSize := normalizeLimit(limit)
	end := offset + int(pageSize)
	if end > len(key.rotationHistory) {
		end = len(key.rotationHistory)
	}

	items := make([]KeyRotationEntry, 0, end-offset)
	for _, entry := range key.rotationHistory[offset:end] {
		items = append(items, cloneRotationEntry(entry))
	}

	truncated := end < len(key.rotationHistory)
	nextMarker := ""
	if truncated {
		nextMarker = strconv.Itoa(end)
	}
	return items, nextMarker, truncated, nil
}

func (s *Service) GetParametersForImport(keyRef, wrappingAlgorithm, wrappingKeySpec string) (ImportParametersOutput, error) {
	algorithm := strings.ToUpper(strings.TrimSpace(wrappingAlgorithm))
	if algorithm == "" {
		algorithm = "RSAES_OAEP_SHA_256"
	}
	switch algorithm {
	case "RSAES_OAEP_SHA_1", "RSAES_OAEP_SHA_256", "RSA_AES_KEY_WRAP_SHA_1", "RSA_AES_KEY_WRAP_SHA_256":
	default:
		return ImportParametersOutput{}, ErrInvalidParameter
	}

	spec := strings.ToUpper(strings.TrimSpace(wrappingKeySpec))
	if spec == "" {
		spec = "RSA_2048"
	}
	switch spec {
	case "RSA_2048", "RSA_3072", "RSA_4096":
	default:
		return ImportParametersOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return ImportParametersOutput{}, err
	}

	publicKey, err := randomBytes(128)
	if err != nil {
		return ImportParametersOutput{}, err
	}
	importToken, err := randomBytes(32)
	if err != nil {
		return ImportParametersOutput{}, err
	}
	validTo := time.Now().UTC().Add(24 * time.Hour)
	s.importTokens[base64.StdEncoding.EncodeToString(importToken)] = importTokenRecord{
		KeyID:     key.KeyID,
		ExpiresAt: validTo,
	}

	return ImportParametersOutput{
		KeyID:             key.Arn,
		ImportToken:       importToken,
		PublicKey:         publicKey,
		ParametersValidTo: validTo,
		WrappingAlgorithm: algorithm,
		WrappingKeySpec:   spec,
	}, nil
}

func (s *Service) ImportKeyMaterial(keyRef string, encryptedKeyMaterial, importToken []byte, expirationModel string, validTo *time.Time) error {
	if len(encryptedKeyMaterial) == 0 || len(importToken) == 0 {
		return ErrInvalidParameter
	}

	model := strings.ToUpper(strings.TrimSpace(expirationModel))
	if model == "" {
		model = "KEY_MATERIAL_DOES_NOT_EXPIRE"
	}
	switch model {
	case "KEY_MATERIAL_DOES_NOT_EXPIRE":
	case "KEY_MATERIAL_EXPIRES":
		if validTo == nil {
			return ErrInvalidParameter
		}
	default:
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}

	tokenKey := base64.StdEncoding.EncodeToString(importToken)
	record, ok := s.importTokens[tokenKey]
	if !ok || record.KeyID != key.KeyID || time.Now().UTC().After(record.ExpiresAt) {
		return ErrInvalidParameter
	}

	hash := sha256.New()
	hash.Write(encryptedKeyMaterial)
	hash.Write(importToken)
	sum := hash.Sum(nil)

	key.material = append([]byte(nil), sum...)
	key.Origin = "EXTERNAL"
	if model == "KEY_MATERIAL_EXPIRES" {
		exp := validTo.UTC()
		key.importedKeyExpiration = &exp
	} else {
		key.importedKeyExpiration = nil
	}
	delete(s.importTokens, tokenKey)
	return nil
}

func (s *Service) DeleteImportedKeyMaterial(keyRef string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return err
	}
	if key.Origin != "EXTERNAL" {
		return ErrInvalidState
	}

	material, err := randomBytes(32)
	if err != nil {
		return err
	}
	key.material = material
	key.Origin = "AWS_KMS"
	key.importedKeyExpiration = nil
	return nil
}

func (s *Service) ReplicateKey(keyRef, replicaRegion string) (Key, error) {
	replicaRegion = strings.TrimSpace(replicaRegion)
	if replicaRegion == "" {
		return Key{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	source, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return Key{}, err
	}
	if source.KeyState == "PendingDeletion" {
		return Key{}, ErrInvalidState
	}

	id := s.nextKeyIDLocked()
	replica := cloneKey(*source)
	replica.KeyID = id
	replica.Arn = keyARNWithRegion(replicaRegion, id)
	replica.CreationDate = time.Now().UTC()
	replica.MultiRegion = true
	replica.PrimaryRegion = source.PrimaryRegion
	if strings.TrimSpace(replica.PrimaryRegion) == "" {
		replica.PrimaryRegion = keyRegion(source)
	}
	replica.DeletionDate = nil
	replica.material = append([]byte(nil), source.material...)
	replica.publicKey = append([]byte(nil), source.publicKey...)
	replica.tags = cloneStringMap(source.tags)
	replica.policies = cloneStringMap(source.policies)
	replica.rotationHistory = cloneRotationEntries(source.rotationHistory)
	s.keys[id] = &replica

	source.MultiRegion = true
	if strings.TrimSpace(source.PrimaryRegion) == "" {
		source.PrimaryRegion = keyRegion(source)
	}

	return cloneKey(replica), nil
}

func (s *Service) UpdatePrimaryRegion(keyRef, primaryRegion string) (Key, error) {
	primaryRegion = strings.TrimSpace(primaryRegion)
	if primaryRegion == "" {
		return Key{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.resolveKeyLocked(keyRef)
	if err != nil {
		return Key{}, err
	}
	if key.KeyState == "PendingDeletion" {
		return Key{}, ErrInvalidState
	}

	key.MultiRegion = true
	key.PrimaryRegion = primaryRegion
	key.Arn = keyARNWithRegion(primaryRegion, key.KeyID)
	for _, alias := range s.aliases {
		if alias.TargetKeyID == key.KeyID {
			alias.TargetKeyArn = key.Arn
		}
	}
	return cloneKey(*key), nil
}

func (s *Service) CreateCustomKeyStore(name, cloudHsmClusterID, customKeyStoreType, xksProxyURI string) (CustomKeyStore, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CustomKeyStore{}, ErrInvalidParameter
	}

	storeType := strings.ToUpper(strings.TrimSpace(customKeyStoreType))
	if storeType == "" {
		storeType = "AWS_CLOUDHSM"
	}
	switch storeType {
	case "AWS_CLOUDHSM", "EXTERNAL_KEY_STORE":
	default:
		return CustomKeyStore{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.customKeyStores {
		if strings.EqualFold(existing.CustomKeyStoreName, name) {
			return CustomKeyStore{}, ErrAlreadyExists
		}
	}

	id := s.nextCustomKeyStoreIDLocked()
	store := CustomKeyStore{
		CustomKeyStoreID:   id,
		CustomKeyStoreName: name,
		CloudHsmClusterID:  strings.TrimSpace(cloudHsmClusterID),
		ConnectionState:    "DISCONNECTED",
		CustomKeyStoreType: storeType,
		XksProxyURI:        strings.TrimSpace(xksProxyURI),
		CreationDate:       time.Now().UTC(),
	}
	s.customKeyStores[id] = &store
	return cloneCustomKeyStore(store), nil
}

func (s *Service) ConnectCustomKeyStore(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	store, err := s.resolveCustomKeyStoreLocked(ref)
	if err != nil {
		return err
	}
	store.ConnectionState = "CONNECTED"
	return nil
}

func (s *Service) DisconnectCustomKeyStore(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	store, err := s.resolveCustomKeyStoreLocked(ref)
	if err != nil {
		return err
	}
	store.ConnectionState = "DISCONNECTED"
	return nil
}

func (s *Service) UpdateCustomKeyStore(ref, name, cloudHsmClusterID, xksProxyURI string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	store, err := s.resolveCustomKeyStoreLocked(ref)
	if err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name != "" && !strings.EqualFold(name, store.CustomKeyStoreName) {
		for _, existing := range s.customKeyStores {
			if existing.CustomKeyStoreID == store.CustomKeyStoreID {
				continue
			}
			if strings.EqualFold(existing.CustomKeyStoreName, name) {
				return ErrAlreadyExists
			}
		}
		store.CustomKeyStoreName = name
	}
	if strings.TrimSpace(cloudHsmClusterID) != "" {
		store.CloudHsmClusterID = strings.TrimSpace(cloudHsmClusterID)
	}
	if strings.TrimSpace(xksProxyURI) != "" {
		store.XksProxyURI = strings.TrimSpace(xksProxyURI)
	}
	return nil
}

func (s *Service) DeleteCustomKeyStore(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	store, err := s.resolveCustomKeyStoreLocked(ref)
	if err != nil {
		return err
	}
	if store.ConnectionState == "CONNECTED" {
		return ErrInvalidState
	}
	delete(s.customKeyStores, store.CustomKeyStoreID)
	return nil
}

func (s *Service) DescribeCustomKeyStores(customKeyStoreID, customKeyStoreName, marker string, limit int32) ([]CustomKeyStore, string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filterID := strings.TrimSpace(customKeyStoreID)
	filterName := strings.TrimSpace(customKeyStoreName)

	storeIDs := make([]string, 0, len(s.customKeyStores))
	for id, store := range s.customKeyStores {
		if filterID != "" && store.CustomKeyStoreID != filterID {
			continue
		}
		if filterName != "" && !strings.EqualFold(store.CustomKeyStoreName, filterName) {
			continue
		}
		storeIDs = append(storeIDs, id)
	}
	sort.Strings(storeIDs)

	offset, err := parseMarker(marker, len(storeIDs))
	if err != nil {
		return nil, "", false, err
	}
	pageSize := normalizeLimit(limit)
	end := offset + int(pageSize)
	if end > len(storeIDs) {
		end = len(storeIDs)
	}

	items := make([]CustomKeyStore, 0, end-offset)
	for _, id := range storeIDs[offset:end] {
		items = append(items, cloneCustomKeyStore(*s.customKeyStores[id]))
	}

	truncated := end < len(storeIDs)
	nextMarker := ""
	if truncated {
		nextMarker = strconv.Itoa(end)
	}
	return items, nextMarker, truncated, nil
}

type cipherRecord struct {
	KeyID     string `json:"k"`
	Algorithm string `json:"a"`
	Plaintext string `json:"p"`
	Nonce     string `json:"n"`
}

type parsedCipherRecord struct {
	KeyID     string
	Algorithm string
	Plaintext []byte
	Nonce     string
}

func buildCiphertext(key *Key, plaintext []byte, algorithm string) ([]byte, error) {
	nonce, err := randomBytes(12)
	if err != nil {
		return nil, err
	}
	record := cipherRecord{
		KeyID:     key.KeyID,
		Algorithm: algorithm,
		Plaintext: base64.StdEncoding.EncodeToString(plaintext),
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
	}
	body, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, key.material)
	mac.Write(body)
	sum := mac.Sum(nil)

	out := make([]byte, 0, 4+16+len(body))
	out = append(out, []byte("SYK2")...)
	out = append(out, sum[:16]...)
	out = append(out, body...)
	return out, nil
}

func parseCiphertextRecord(ciphertext []byte, keys map[string]*Key) (parsedCipherRecord, error) {
	if len(ciphertext) < 20 {
		return parsedCipherRecord{}, ErrInvalidCiphertext
	}
	if string(ciphertext[:4]) != "SYK2" {
		return parsedCipherRecord{}, ErrInvalidCiphertext
	}

	sig := ciphertext[4:20]
	body := ciphertext[20:]

	var record cipherRecord
	if err := json.Unmarshal(body, &record); err != nil {
		return parsedCipherRecord{}, ErrInvalidCiphertext
	}
	if strings.TrimSpace(record.KeyID) == "" || strings.TrimSpace(record.Algorithm) == "" || strings.TrimSpace(record.Plaintext) == "" {
		return parsedCipherRecord{}, ErrInvalidCiphertext
	}

	key := keys[record.KeyID]
	if key == nil {
		return parsedCipherRecord{}, ErrInvalidCiphertext
	}
	mac := hmac.New(sha256.New, key.material)
	mac.Write(body)
	expected := mac.Sum(nil)
	if len(expected) < len(sig) || !hmac.Equal(sig, expected[:len(sig)]) {
		return parsedCipherRecord{}, ErrInvalidCiphertext
	}

	plaintext, err := base64.StdEncoding.DecodeString(record.Plaintext)
	if err != nil {
		return parsedCipherRecord{}, ErrInvalidCiphertext
	}
	return parsedCipherRecord{
		KeyID:     record.KeyID,
		Algorithm: record.Algorithm,
		Plaintext: append([]byte(nil), plaintext...),
		Nonce:     record.Nonce,
	}, nil
}

func ensureKeyUsableForEncrypt(key *Key) error {
	if key.KeyState == "PendingDeletion" {
		return ErrInvalidState
	}
	if !key.Enabled {
		return ErrDisabled
	}
	if key.KeyUsage != "ENCRYPT_DECRYPT" {
		return ErrInvalidState
	}
	return nil
}

func (s *Service) resolveKeyLocked(ref string) (*Key, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrInvalidParameter
	}

	if strings.HasPrefix(ref, "arn:aws:kms:") {
		if idx := strings.Index(ref, ":key/"); idx >= 0 {
			id := strings.TrimSpace(ref[idx+len(":key/"):])
			if id == "" {
				return nil, ErrInvalidParameter
			}
			key := s.keys[id]
			if key == nil {
				return nil, ErrNotFound
			}
			return key, nil
		}
		if idx := strings.Index(ref, ":alias/"); idx >= 0 {
			aliasName := "alias/" + strings.TrimSpace(ref[idx+len(":alias/"):])
			entry := s.aliases[aliasName]
			if entry == nil {
				return nil, ErrNotFound
			}
			key := s.keys[entry.TargetKeyID]
			if key == nil {
				return nil, ErrNotFound
			}
			return key, nil
		}
		return nil, ErrInvalidParameter
	}

	if strings.HasPrefix(ref, "alias/") {
		entry := s.aliases[ref]
		if entry == nil {
			return nil, ErrNotFound
		}
		key := s.keys[entry.TargetKeyID]
		if key == nil {
			return nil, ErrNotFound
		}
		return key, nil
	}

	key := s.keys[ref]
	if key == nil {
		return nil, ErrNotFound
	}
	return key, nil
}

func (s *Service) nextKeyIDLocked() string {
	s.seq++
	return fmt.Sprintf("00000000-0000-0000-0000-%012x", s.seq)
}

func (s *Service) nextGrantIDLocked() string {
	s.grantSeq++
	return fmt.Sprintf("%016x", s.grantSeq)
}

func (s *Service) nextCustomKeyStoreIDLocked() string {
	s.customStoreSeq++
	return fmt.Sprintf("cks-%08x", s.customStoreSeq)
}

func keyARN(keyID string) string {
	return fmt.Sprintf("arn:aws:kms:%s:%s:key/%s", DefaultRegion, DefaultAccountID, keyID)
}

func keyARNWithRegion(region, keyID string) string {
	return fmt.Sprintf("arn:aws:kms:%s:%s:key/%s", strings.TrimSpace(region), DefaultAccountID, keyID)
}

func aliasARN(aliasName string) string {
	return fmt.Sprintf("arn:aws:kms:%s:%s:%s", DefaultRegion, DefaultAccountID, aliasName)
}

func cloneKey(in Key) Key {
	out := in
	out.EncryptionAlgorithms = append([]string(nil), in.EncryptionAlgorithms...)
	out.SigningAlgorithms = append([]string(nil), in.SigningAlgorithms...)
	out.MacAlgorithms = append([]string(nil), in.MacAlgorithms...)
	out.KeyAgreementAlgorithms = append([]string(nil), in.KeyAgreementAlgorithms...)
	out.material = append([]byte(nil), in.material...)
	out.publicKey = append([]byte(nil), in.publicKey...)
	out.tags = cloneStringMap(in.tags)
	out.policies = cloneStringMap(in.policies)
	out.rotationHistory = cloneRotationEntries(in.rotationHistory)
	if in.DeletionDate != nil {
		d := *in.DeletionDate
		out.DeletionDate = &d
	}
	if in.importedKeyExpiration != nil {
		d := *in.importedKeyExpiration
		out.importedKeyExpiration = &d
	}
	return out
}

func cloneAlias(in Alias) Alias {
	return Alias{
		AliasName:    in.AliasName,
		AliasArn:     in.AliasArn,
		TargetKeyID:  in.TargetKeyID,
		TargetKeyArn: in.TargetKeyArn,
		CreatedAt:    in.CreatedAt,
	}
}

func cloneGrant(in Grant) Grant {
	return Grant{
		GrantID:           in.GrantID,
		GrantToken:        in.GrantToken,
		KeyID:             in.KeyID,
		Name:              in.Name,
		GranteePrincipal:  in.GranteePrincipal,
		RetiringPrincipal: in.RetiringPrincipal,
		Operations:        append([]string(nil), in.Operations...),
		IssuingAccount:    in.IssuingAccount,
		CreationDate:      in.CreationDate,
	}
}

func cloneRotationEntry(in KeyRotationEntry) KeyRotationEntry {
	return KeyRotationEntry{
		RotationDate: in.RotationDate,
		RotationType: in.RotationType,
	}
}

func cloneRotationEntries(items []KeyRotationEntry) []KeyRotationEntry {
	out := make([]KeyRotationEntry, 0, len(items))
	for _, item := range items {
		out = append(out, cloneRotationEntry(item))
	}
	return out
}

func cloneCustomKeyStore(in CustomKeyStore) CustomKeyStore {
	return CustomKeyStore{
		CustomKeyStoreID:   in.CustomKeyStoreID,
		CustomKeyStoreName: in.CustomKeyStoreName,
		CloudHsmClusterID:  in.CloudHsmClusterID,
		ConnectionState:    in.ConnectionState,
		CustomKeyStoreType: in.CustomKeyStoreType,
		XksProxyURI:        in.XksProxyURI,
		CreationDate:       in.CreationDate,
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func keyRegion(key *Key) string {
	if key == nil {
		return DefaultRegion
	}
	arn := strings.TrimSpace(key.Arn)
	if arn == "" {
		return DefaultRegion
	}
	parts := strings.Split(arn, ":")
	if len(parts) < 4 || strings.TrimSpace(parts[3]) == "" {
		return DefaultRegion
	}
	return parts[3]
}

func isRotatableKey(key *Key) bool {
	if key == nil {
		return false
	}
	return key.KeyUsage == "ENCRYPT_DECRYPT" && key.KeySpec == "SYMMETRIC_DEFAULT"
}

func (s *Service) resolveCustomKeyStoreLocked(ref string) (*CustomKeyStore, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrInvalidParameter
	}
	if store := s.customKeyStores[ref]; store != nil {
		return store, nil
	}
	for _, store := range s.customKeyStores {
		if strings.EqualFold(store.CustomKeyStoreName, ref) {
			return store, nil
		}
	}
	return nil, ErrNotFound
}

func validAliasName(aliasName string) bool {
	if !strings.HasPrefix(aliasName, "alias/") {
		return false
	}
	if strings.HasPrefix(aliasName, "alias/aws/") {
		return false
	}
	suffix := strings.TrimPrefix(aliasName, "alias/")
	if strings.TrimSpace(suffix) == "" {
		return false
	}
	return len(aliasName) <= 256
}

func validKeyUsage(value string) bool {
	switch value {
	case "ENCRYPT_DECRYPT", "SIGN_VERIFY", "GENERATE_VERIFY_MAC", "KEY_AGREEMENT":
		return true
	default:
		return false
	}
}

func validKeySpecForUsage(spec, usage string) bool {
	switch usage {
	case "ENCRYPT_DECRYPT":
		switch spec {
		case "SYMMETRIC_DEFAULT", "RSA_2048", "RSA_3072", "RSA_4096":
			return true
		}
	case "SIGN_VERIFY":
		switch spec {
		case "RSA_2048", "RSA_3072", "RSA_4096", "ECC_NIST_P256", "ECC_NIST_P384", "ECC_SECG_P256K1":
			return true
		}
	case "GENERATE_VERIFY_MAC":
		switch spec {
		case "HMAC_224", "HMAC_256", "HMAC_384", "HMAC_512":
			return true
		}
	case "KEY_AGREEMENT":
		switch spec {
		case "ECC_NIST_P256", "ECC_NIST_P384", "ECC_SECG_P256K1":
			return true
		}
	}
	return false
}

func defaultEncryptionAlgorithms(spec, usage string) []string {
	if usage != "ENCRYPT_DECRYPT" {
		return nil
	}
	switch spec {
	case "SYMMETRIC_DEFAULT":
		return []string{"SYMMETRIC_DEFAULT"}
	case "RSA_2048", "RSA_3072", "RSA_4096":
		return []string{"RSAES_OAEP_SHA_1", "RSAES_OAEP_SHA_256"}
	default:
		return nil
	}
}

func defaultSigningAlgorithms(spec, usage string) []string {
	if usage != "SIGN_VERIFY" {
		return nil
	}
	switch spec {
	case "RSA_2048", "RSA_3072", "RSA_4096":
		return []string{"RSASSA_PSS_SHA_256", "RSASSA_PKCS1_V1_5_SHA_256"}
	case "ECC_NIST_P256", "ECC_SECG_P256K1":
		return []string{"ECDSA_SHA_256"}
	case "ECC_NIST_P384":
		return []string{"ECDSA_SHA_384"}
	default:
		return nil
	}
}

func defaultMacAlgorithms(spec, usage string) []string {
	if usage != "GENERATE_VERIFY_MAC" {
		return nil
	}
	switch spec {
	case "HMAC_224":
		return []string{"HMAC_SHA_224"}
	case "HMAC_256":
		return []string{"HMAC_SHA_256"}
	case "HMAC_384":
		return []string{"HMAC_SHA_384"}
	case "HMAC_512":
		return []string{"HMAC_SHA_512"}
	default:
		return nil
	}
}

func defaultKeyAgreementAlgorithms(spec, usage string) []string {
	if usage != "KEY_AGREEMENT" {
		return nil
	}
	switch spec {
	case "ECC_NIST_P256", "ECC_NIST_P384", "ECC_SECG_P256K1":
		return []string{"ECDH"}
	default:
		return nil
	}
}

func defaultEncryptAlgorithmForKey(key *Key) string {
	if len(key.EncryptionAlgorithms) == 0 {
		return "SYMMETRIC_DEFAULT"
	}
	return key.EncryptionAlgorithms[0]
}

func parseMarker(marker string, total int) (int, error) {
	marker = strings.TrimSpace(marker)
	if marker == "" {
		return 0, nil
	}
	idx, err := strconv.Atoi(marker)
	if err != nil || idx < 0 || idx > total {
		return 0, ErrInvalidParameter
	}
	return idx, nil
}

func normalizeLimit(limit int32) int32 {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

func resolveDataKeySize(keySpec string, numberOfBytes int32) (int, error) {
	if numberOfBytes > 0 {
		if numberOfBytes < 1 || numberOfBytes > 1024 {
			return 0, ErrInvalidParameter
		}
		return int(numberOfBytes), nil
	}

	switch keySpec {
	case "", "AES_256":
		return 32, nil
	case "AES_128":
		return 16, nil
	default:
		return 0, ErrInvalidParameter
	}
}

func keyPairSizes(spec string) (int, int, error) {
	switch spec {
	case "RSA_2048":
		return 256, 256, nil
	case "RSA_3072":
		return 384, 384, nil
	case "RSA_4096":
		return 512, 512, nil
	case "ECC_NIST_P256", "ECC_SECG_P256K1":
		return 32, 65, nil
	case "ECC_NIST_P384":
		return 48, 97, nil
	default:
		return 0, 0, ErrInvalidParameter
	}
}

func randomBytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, ErrInvalidParameter
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}

func computeHMAC(secret []byte, algorithm string, message []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(algorithm))
	mac.Write([]byte{0})
	mac.Write(message)
	return mac.Sum(nil)
}
