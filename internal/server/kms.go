package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	kmssvc "github.com/stackyard/stackyard/internal/services/kms"
)

type kmsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleKMSJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isKMSJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "kms")
	if !ok {
		respondKMSError(w, status, code, msg)
		return true
	}

	action := parseKMSTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondKMSError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := kmsOperationByName[action]; !known {
		respondKMSError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseKMSPayload(r)
	if err != nil {
		respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	switch action {
	case "CreateKey":
		key, err := s.kms.CreateKey(
			kmsString(payload["Description"]),
			kmsString(payload["KeyUsage"]),
			kmsString(payload["KeySpec"]),
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{"KeyMetadata": kmsKeyMetadataPayload(key)})
		return true

	case "DescribeKey":
		key, err := s.kms.DescribeKey(kmsString(payload["KeyId"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{"KeyMetadata": kmsKeyMetadataPayload(key)})
		return true

	case "ListKeys":
		limit, ok := kmsInt32(payload["Limit"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		items, nextMarker, truncated, err := s.kms.ListKeys(kmsString(payload["Marker"]), limit)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Keys":      kmsKeyListPayload(items),
			"Truncated": truncated,
		}
		if strings.TrimSpace(nextMarker) != "" {
			response["NextMarker"] = nextMarker
		}
		respondKMSJSON(w, http.StatusOK, response)
		return true

	case "EnableKey":
		if err := s.kms.EnableKey(kmsString(payload["KeyId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DisableKey":
		if err := s.kms.DisableKey(kmsString(payload["KeyId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ScheduleKeyDeletion":
		pendingWindow, ok := kmsInt32(payload["PendingWindowInDays"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid PendingWindowInDays")
			return true
		}
		keyID, deletionDate, err := s.kms.ScheduleKeyDeletion(kmsString(payload["KeyId"]), pendingWindow)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{"KeyId": keyID, "DeletionDate": deletionDate})
		return true

	case "CancelKeyDeletion":
		keyID, err := s.kms.CancelKeyDeletion(kmsString(payload["KeyId"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{"KeyId": keyID})
		return true

	case "UpdateKeyDescription":
		if err := s.kms.UpdateKeyDescription(kmsString(payload["KeyId"]), kmsString(payload["Description"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "CreateAlias":
		if err := s.kms.CreateAlias(kmsString(payload["AliasName"]), kmsString(payload["TargetKeyId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UpdateAlias":
		if err := s.kms.UpdateAlias(kmsString(payload["AliasName"]), kmsString(payload["TargetKeyId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DeleteAlias":
		if err := s.kms.DeleteAlias(kmsString(payload["AliasName"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ListAliases":
		limit, ok := kmsInt32(payload["Limit"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		aliases, nextMarker, truncated, err := s.kms.ListAliases(
			kmsString(payload["KeyId"]),
			kmsString(payload["Marker"]),
			limit,
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Aliases":   kmsAliasListPayload(aliases),
			"Truncated": truncated,
		}
		if strings.TrimSpace(nextMarker) != "" {
			response["NextMarker"] = nextMarker
		}
		respondKMSJSON(w, http.StatusOK, response)
		return true

	case "Encrypt":
		plaintext, ok := kmsBlob(payload["Plaintext"])
		if !ok || len(plaintext) == 0 {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Plaintext")
			return true
		}
		output, err := s.kms.Encrypt(
			kmsString(payload["KeyId"]),
			plaintext,
			kmsString(payload["EncryptionAlgorithm"]),
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"CiphertextBlob":      output.CiphertextBlob,
			"KeyId":               output.KeyID,
			"EncryptionAlgorithm": output.EncryptionAlgorithm,
		})
		return true

	case "Decrypt":
		ciphertext, ok := kmsBlob(payload["CiphertextBlob"])
		if !ok || len(ciphertext) == 0 {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid CiphertextBlob")
			return true
		}
		output, err := s.kms.Decrypt(
			ciphertext,
			kmsString(payload["KeyId"]),
			kmsString(payload["EncryptionAlgorithm"]),
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"Plaintext":           output.Plaintext,
			"KeyId":               output.KeyID,
			"EncryptionAlgorithm": output.EncryptionAlgorithm,
		})
		return true

	case "ReEncrypt":
		ciphertext, ok := kmsBlob(payload["CiphertextBlob"])
		if !ok || len(ciphertext) == 0 {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid CiphertextBlob")
			return true
		}
		output, err := s.kms.ReEncrypt(
			ciphertext,
			kmsString(payload["SourceKeyId"]),
			kmsString(payload["DestinationKeyId"]),
			kmsString(payload["SourceEncryptionAlgorithm"]),
			kmsString(payload["DestinationEncryptionAlgorithm"]),
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"CiphertextBlob":                 output.CiphertextBlob,
			"KeyId":                          output.KeyID,
			"SourceKeyId":                    output.SourceKeyID,
			"SourceEncryptionAlgorithm":      output.SourceEncryptionAlgorithm,
			"DestinationEncryptionAlgorithm": output.DestinationEncryptionAlgorithm,
		})
		return true

	case "GenerateDataKey":
		numberOfBytes, ok := kmsInt32(payload["NumberOfBytes"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid NumberOfBytes")
			return true
		}
		output, err := s.kms.GenerateDataKey(
			kmsString(payload["KeyId"]),
			kmsString(payload["KeySpec"]),
			numberOfBytes,
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"CiphertextBlob": output.CiphertextBlob,
			"Plaintext":      output.Plaintext,
			"KeyId":          output.KeyID,
		})
		return true

	case "GenerateDataKeyWithoutPlaintext":
		numberOfBytes, ok := kmsInt32(payload["NumberOfBytes"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid NumberOfBytes")
			return true
		}
		output, err := s.kms.GenerateDataKeyWithoutPlaintext(
			kmsString(payload["KeyId"]),
			kmsString(payload["KeySpec"]),
			numberOfBytes,
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"CiphertextBlob": output.CiphertextBlob,
			"KeyId":          output.KeyID,
		})
		return true

	case "GenerateDataKeyPair":
		output, err := s.kms.GenerateDataKeyPair(
			kmsString(payload["KeyId"]),
			kmsString(payload["KeyPairSpec"]),
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyId":                    output.KeyID,
			"KeyPairSpec":              output.KeyPairSpec,
			"PublicKey":                output.PublicKey,
			"PrivateKeyPlaintext":      output.PrivateKeyPlaintext,
			"PrivateKeyCiphertextBlob": output.PrivateKeyCiphertextBlob,
		})
		return true

	case "GenerateDataKeyPairWithoutPlaintext":
		output, err := s.kms.GenerateDataKeyPairWithoutPlaintext(
			kmsString(payload["KeyId"]),
			kmsString(payload["KeyPairSpec"]),
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyId":                    output.KeyID,
			"KeyPairSpec":              output.KeyPairSpec,
			"PublicKey":                output.PublicKey,
			"PrivateKeyCiphertextBlob": output.PrivateKeyCiphertextBlob,
		})
		return true

	case "GenerateRandom":
		numberOfBytes, ok := kmsInt32(payload["NumberOfBytes"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid NumberOfBytes")
			return true
		}
		plaintext, err := s.kms.GenerateRandom(numberOfBytes)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{"Plaintext": plaintext})
		return true

	case "GetPublicKey":
		output, err := s.kms.GetPublicKey(kmsString(payload["KeyId"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"KeyId":     output.KeyID,
			"PublicKey": output.PublicKey,
			"KeySpec":   output.KeySpec,
			"KeyUsage":  output.KeyUsage,
		}
		if len(output.EncryptionAlgorithms) > 0 {
			response["EncryptionAlgorithms"] = output.EncryptionAlgorithms
		}
		if len(output.SigningAlgorithms) > 0 {
			response["SigningAlgorithms"] = output.SigningAlgorithms
		}
		if len(output.KeyAgreementAlgorithms) > 0 {
			response["KeyAgreementAlgorithms"] = output.KeyAgreementAlgorithms
		}
		respondKMSJSON(w, http.StatusOK, response)
		return true

	case "Sign":
		message, ok := kmsBlob(payload["Message"])
		if !ok || len(message) == 0 {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Message")
			return true
		}
		output, err := s.kms.Sign(kmsString(payload["KeyId"]), message, kmsString(payload["SigningAlgorithm"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyId":            output.KeyID,
			"Signature":        output.Signature,
			"SigningAlgorithm": output.SigningAlgorithm,
		})
		return true

	case "Verify":
		message, messageOK := kmsBlob(payload["Message"])
		signature, sigOK := kmsBlob(payload["Signature"])
		if !messageOK || len(message) == 0 || !sigOK || len(signature) == 0 {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Message or Signature")
			return true
		}
		output, err := s.kms.Verify(kmsString(payload["KeyId"]), message, signature, kmsString(payload["SigningAlgorithm"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyId":            output.KeyID,
			"SignatureValid":   output.SignatureValid,
			"SigningAlgorithm": output.SigningAlgorithm,
		})
		return true

	case "GenerateMac":
		message, ok := kmsBlob(payload["Message"])
		if !ok || len(message) == 0 {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Message")
			return true
		}
		output, err := s.kms.GenerateMac(kmsString(payload["KeyId"]), message, kmsString(payload["MacAlgorithm"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyId":        output.KeyID,
			"Mac":          output.Mac,
			"MacAlgorithm": output.MacAlgorithm,
		})
		return true

	case "VerifyMac":
		message, messageOK := kmsBlob(payload["Message"])
		mac, macOK := kmsBlob(payload["Mac"])
		if !messageOK || len(message) == 0 || !macOK || len(mac) == 0 {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Message or Mac")
			return true
		}
		output, err := s.kms.VerifyMac(kmsString(payload["KeyId"]), message, mac, kmsString(payload["MacAlgorithm"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyId":        output.KeyID,
			"MacValid":     output.MacValid,
			"MacAlgorithm": output.MacAlgorithm,
		})
		return true

	case "DeriveSharedSecret":
		peerPublicKey, ok := kmsBlob(payload["PublicKey"])
		if !ok || len(peerPublicKey) == 0 {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid PublicKey")
			return true
		}
		output, err := s.kms.DeriveSharedSecret(kmsString(payload["KeyId"]), peerPublicKey, kmsString(payload["KeyAgreementAlgorithm"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyId":                 output.KeyID,
			"SharedSecret":          output.SharedSecret,
			"KeyAgreementAlgorithm": output.KeyAgreementAlgorithm,
		})
		return true

	case "PutKeyPolicy":
		if err := s.kms.PutKeyPolicy(
			kmsString(payload["KeyId"]),
			kmsString(payload["PolicyName"]),
			kmsString(payload["Policy"]),
		); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetKeyPolicy":
		policyName, policy, err := s.kms.GetKeyPolicy(kmsString(payload["KeyId"]), kmsString(payload["PolicyName"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"PolicyName": policyName,
			"Policy":     policy,
		})
		return true

	case "ListKeyPolicies":
		limit, ok := kmsInt32(payload["Limit"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		names, nextMarker, truncated, err := s.kms.ListKeyPolicies(kmsString(payload["KeyId"]), kmsString(payload["Marker"]), limit)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"PolicyNames": names,
			"Truncated":   truncated,
		}
		if strings.TrimSpace(nextMarker) != "" {
			response["NextMarker"] = nextMarker
		}
		respondKMSJSON(w, http.StatusOK, response)
		return true

	case "TagResource":
		tags, ok := kmsTags(payload["Tags"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Tags")
			return true
		}
		if err := s.kms.TagResource(kmsString(payload["KeyId"]), tags); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UntagResource":
		tagKeys, ok := kmsStringSlice(payload["TagKeys"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid TagKeys")
			return true
		}
		if err := s.kms.UntagResource(kmsString(payload["KeyId"]), tagKeys); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ListResourceTags":
		limit, ok := kmsInt32(payload["Limit"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		tags, nextMarker, truncated, err := s.kms.ListResourceTags(
			kmsString(payload["KeyId"]),
			kmsString(payload["Marker"]),
			limit,
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Tags":      kmsTagListPayload(tags),
			"Truncated": truncated,
		}
		if strings.TrimSpace(nextMarker) != "" {
			response["NextMarker"] = nextMarker
		}
		respondKMSJSON(w, http.StatusOK, response)
		return true

	case "CreateGrant":
		operations, ok := kmsStringSlice(payload["Operations"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Operations")
			return true
		}
		grant, err := s.kms.CreateGrant(
			kmsString(payload["KeyId"]),
			kmsString(payload["Name"]),
			kmsString(payload["GranteePrincipal"]),
			kmsString(payload["RetiringPrincipal"]),
			operations,
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"GrantId":    grant.GrantID,
			"GrantToken": grant.GrantToken,
		})
		return true

	case "ListGrants":
		limit, ok := kmsInt32(payload["Limit"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		grants, nextMarker, truncated, err := s.kms.ListGrants(
			kmsString(payload["KeyId"]),
			kmsString(payload["Marker"]),
			limit,
			kmsString(payload["GranteePrincipal"]),
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Grants":    kmsGrantListPayload(grants),
			"Truncated": truncated,
		}
		if strings.TrimSpace(nextMarker) != "" {
			response["NextMarker"] = nextMarker
		}
		respondKMSJSON(w, http.StatusOK, response)
		return true

	case "ListRetirableGrants":
		limit, ok := kmsInt32(payload["Limit"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		grants, nextMarker, truncated, err := s.kms.ListRetirableGrants(
			kmsString(payload["RetiringPrincipal"]),
			kmsString(payload["Marker"]),
			limit,
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Grants":    kmsGrantListPayload(grants),
			"Truncated": truncated,
		}
		if strings.TrimSpace(nextMarker) != "" {
			response["NextMarker"] = nextMarker
		}
		respondKMSJSON(w, http.StatusOK, response)
		return true

	case "RetireGrant":
		if err := s.kms.RetireGrant(kmsString(payload["GrantId"]), kmsString(payload["KeyId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "RevokeGrant":
		if err := s.kms.RevokeGrant(kmsString(payload["KeyId"]), kmsString(payload["GrantId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "EnableKeyRotation":
		if err := s.kms.EnableKeyRotation(kmsString(payload["KeyId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DisableKeyRotation":
		if err := s.kms.DisableKeyRotation(kmsString(payload["KeyId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetKeyRotationStatus":
		enabled, err := s.kms.GetKeyRotationStatus(kmsString(payload["KeyId"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyRotationEnabled": enabled,
		})
		return true

	case "RotateKeyOnDemand":
		keyID, err := s.kms.RotateKeyOnDemand(kmsString(payload["KeyId"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyId": keyID,
		})
		return true

	case "ListKeyRotations":
		limit, ok := kmsInt32(payload["Limit"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		rotations, nextMarker, truncated, err := s.kms.ListKeyRotations(
			kmsString(payload["KeyId"]),
			kmsString(payload["Marker"]),
			limit,
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Rotations": kmsKeyRotationListPayload(rotations),
			"Truncated": truncated,
		}
		if strings.TrimSpace(nextMarker) != "" {
			response["NextMarker"] = nextMarker
		}
		respondKMSJSON(w, http.StatusOK, response)
		return true

	case "GetParametersForImport":
		output, err := s.kms.GetParametersForImport(
			kmsString(payload["KeyId"]),
			kmsString(payload["WrappingAlgorithm"]),
			kmsString(payload["WrappingKeySpec"]),
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyId":             output.KeyID,
			"ImportToken":       output.ImportToken,
			"PublicKey":         output.PublicKey,
			"ParametersValidTo": output.ParametersValidTo,
			"WrappingAlgorithm": output.WrappingAlgorithm,
			"WrappingKeySpec":   output.WrappingKeySpec,
		})
		return true

	case "ImportKeyMaterial":
		encryptedKeyMaterial, materialOK := kmsBlob(payload["EncryptedKeyMaterial"])
		importToken, tokenOK := kmsBlob(payload["ImportToken"])
		if !materialOK || !tokenOK || len(encryptedKeyMaterial) == 0 || len(importToken) == 0 {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid EncryptedKeyMaterial or ImportToken")
			return true
		}
		validTo, ok := kmsTimePtr(payload["ValidTo"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid ValidTo")
			return true
		}
		if err := s.kms.ImportKeyMaterial(
			kmsString(payload["KeyId"]),
			encryptedKeyMaterial,
			importToken,
			kmsString(payload["ExpirationModel"]),
			validTo,
		); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DeleteImportedKeyMaterial":
		if err := s.kms.DeleteImportedKeyMaterial(kmsString(payload["KeyId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ReplicateKey":
		replica, err := s.kms.ReplicateKey(kmsString(payload["KeyId"]), kmsString(payload["ReplicaRegion"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"ReplicaKeyMetadata": kmsKeyMetadataPayload(replica),
		})
		return true

	case "UpdatePrimaryRegion":
		key, err := s.kms.UpdatePrimaryRegion(kmsString(payload["KeyId"]), kmsString(payload["PrimaryRegion"]))
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"KeyMetadata": kmsKeyMetadataPayload(key),
		})
		return true

	case "CreateCustomKeyStore":
		store, err := s.kms.CreateCustomKeyStore(
			kmsString(payload["CustomKeyStoreName"]),
			kmsString(payload["CloudHsmClusterId"]),
			kmsString(payload["CustomKeyStoreType"]),
			kmsString(payload["XksProxyUriEndpoint"]),
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{
			"CustomKeyStoreId": store.CustomKeyStoreID,
		})
		return true

	case "ConnectCustomKeyStore":
		if err := s.kms.ConnectCustomKeyStore(kmsString(payload["CustomKeyStoreId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DisconnectCustomKeyStore":
		if err := s.kms.DisconnectCustomKeyStore(kmsString(payload["CustomKeyStoreId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UpdateCustomKeyStore":
		if err := s.kms.UpdateCustomKeyStore(
			kmsString(payload["CustomKeyStoreId"]),
			kmsString(payload["NewCustomKeyStoreName"]),
			kmsString(payload["CloudHsmClusterId"]),
			kmsString(payload["XksProxyUriEndpoint"]),
		); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DeleteCustomKeyStore":
		if err := s.kms.DeleteCustomKeyStore(kmsString(payload["CustomKeyStoreId"])); err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		respondKMSJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DescribeCustomKeyStores":
		limit, ok := kmsInt32(payload["Limit"])
		if !ok {
			respondKMSError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		stores, nextMarker, truncated, err := s.kms.DescribeCustomKeyStores(
			kmsString(payload["CustomKeyStoreId"]),
			kmsString(payload["CustomKeyStoreName"]),
			kmsString(payload["Marker"]),
			limit,
		)
		if err != nil {
			respondKMSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"CustomKeyStores": kmsCustomKeyStoreListPayload(stores),
			"Truncated":       truncated,
		}
		if strings.TrimSpace(nextMarker) != "" {
			response["NextMarker"] = nextMarker
		}
		respondKMSJSON(w, http.StatusOK, response)
		return true
	}

	respondKMSError(w, http.StatusInternalServerError, "InternalException", "known action is not routed")
	return true
}

func isKMSJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "TrentService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "TrentService")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "kms" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".kms.") || strings.HasPrefix(host, "kms.")
}

func parseKMSTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "TrentService.") {
		return strings.TrimPrefix(target, "TrentService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseKMSPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var payload map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func respondKMSJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondKMSError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondKMSJSON(w, status, kmsError{Type: code, Message: msg})
}

func respondKMSErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, kmssvc.ErrInvalidParameter):
		respondKMSError(w, http.StatusBadRequest, "ValidationException", err.Error())
	case errors.Is(err, kmssvc.ErrNotFound):
		respondKMSError(w, http.StatusBadRequest, "NotFoundException", err.Error())
	case errors.Is(err, kmssvc.ErrAlreadyExists):
		respondKMSError(w, http.StatusBadRequest, "AlreadyExistsException", err.Error())
	case errors.Is(err, kmssvc.ErrDisabled):
		respondKMSError(w, http.StatusBadRequest, "DisabledException", err.Error())
	case errors.Is(err, kmssvc.ErrInvalidState):
		respondKMSError(w, http.StatusBadRequest, "KMSInvalidStateException", err.Error())
	case errors.Is(err, kmssvc.ErrInvalidCiphertext):
		respondKMSError(w, http.StatusBadRequest, "InvalidCiphertextException", err.Error())
	case errors.Is(err, kmssvc.ErrIncorrectKey):
		respondKMSError(w, http.StatusBadRequest, "IncorrectKeyException", err.Error())
	case errors.Is(err, kmssvc.ErrUnsupportedAlgorithm):
		respondKMSError(w, http.StatusBadRequest, "ValidationException", err.Error())
	default:
		respondKMSError(w, http.StatusInternalServerError, "InternalException", err.Error())
	}
}

func kmsKeyMetadataPayload(key kmssvc.Key) map[string]any {
	payload := map[string]any{
		"AWSAccountId":          key.AWSAccountID,
		"KeyId":                 key.KeyID,
		"Arn":                   key.Arn,
		"Description":           key.Description,
		"Enabled":               key.Enabled,
		"KeyState":              key.KeyState,
		"KeyUsage":              key.KeyUsage,
		"KeySpec":               key.KeySpec,
		"CustomerMasterKeySpec": key.CustomerMasterKeySpec,
		"Origin":                key.Origin,
		"KeyManager":            key.KeyManager,
		"CreationDate":          key.CreationDate,
		"KeyRotationEnabled":    key.RotationEnabled,
		"MultiRegion":           key.MultiRegion,
	}
	if strings.TrimSpace(key.PrimaryRegion) != "" {
		payload["PrimaryRegion"] = key.PrimaryRegion
	}
	if key.DeletionDate != nil {
		payload["DeletionDate"] = *key.DeletionDate
	}
	if len(key.EncryptionAlgorithms) > 0 {
		payload["EncryptionAlgorithms"] = key.EncryptionAlgorithms
	}
	if len(key.SigningAlgorithms) > 0 {
		payload["SigningAlgorithms"] = key.SigningAlgorithms
	}
	if len(key.MacAlgorithms) > 0 {
		payload["MacAlgorithms"] = key.MacAlgorithms
	}
	if len(key.KeyAgreementAlgorithms) > 0 {
		payload["KeyAgreementAlgorithms"] = key.KeyAgreementAlgorithms
	}
	return payload
}

func kmsKeyListPayload(items []kmssvc.KeyListEntry) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"KeyId":  item.KeyID,
			"KeyArn": item.KeyArn,
		})
	}
	return out
}

func kmsAliasListPayload(items []kmssvc.Alias) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload := map[string]any{
			"AliasName": item.AliasName,
			"AliasArn":  item.AliasArn,
		}
		if strings.TrimSpace(item.TargetKeyID) != "" {
			payload["TargetKeyId"] = item.TargetKeyID
		}
		if strings.TrimSpace(item.TargetKeyArn) != "" {
			payload["TargetKeyArn"] = item.TargetKeyArn
		}
		out = append(out, payload)
	}
	return out
}

func kmsTagListPayload(items []kmssvc.Tag) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"TagKey":   item.TagKey,
			"TagValue": item.TagValue,
		})
	}
	return out
}

func kmsGrantListPayload(items []kmssvc.Grant) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload := map[string]any{
			"GrantId":          item.GrantID,
			"KeyId":            item.KeyID,
			"GranteePrincipal": item.GranteePrincipal,
			"Operations":       item.Operations,
			"IssuingAccount":   item.IssuingAccount,
			"CreationDate":     item.CreationDate,
		}
		if strings.TrimSpace(item.Name) != "" {
			payload["Name"] = item.Name
		}
		if strings.TrimSpace(item.RetiringPrincipal) != "" {
			payload["RetiringPrincipal"] = item.RetiringPrincipal
		}
		out = append(out, payload)
	}
	return out
}

func kmsKeyRotationListPayload(items []kmssvc.KeyRotationEntry) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"RotationDate": item.RotationDate,
			"RotationType": item.RotationType,
		})
	}
	return out
}

func kmsCustomKeyStoreListPayload(items []kmssvc.CustomKeyStore) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"CustomKeyStoreId":    item.CustomKeyStoreID,
			"CustomKeyStoreName":  item.CustomKeyStoreName,
			"CloudHsmClusterId":   item.CloudHsmClusterID,
			"ConnectionState":     item.ConnectionState,
			"CustomKeyStoreType":  item.CustomKeyStoreType,
			"XksProxyUriEndpoint": item.XksProxyURI,
			"CreationDate":        item.CreationDate,
		})
	}
	return out
}

func kmsString(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func kmsInt32(value any) (int32, bool) {
	switch raw := value.(type) {
	case nil:
		return 0, true
	case float64:
		return int32(raw), true
	case float32:
		return int32(raw), true
	case int:
		return int32(raw), true
	case int32:
		return raw, true
	case int64:
		return int32(raw), true
	case json.Number:
		parsed, err := raw.Int64()
		if err != nil {
			return 0, false
		}
		return int32(parsed), true
	case string:
		if strings.TrimSpace(raw) == "" {
			return 0, true
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return 0, false
		}
		return int32(parsed), true
	default:
		return 0, false
	}
}

func kmsStringSlice(value any) ([]string, bool) {
	if value == nil {
		return nil, true
	}
	switch raw := value.(type) {
	case []string:
		return append([]string(nil), raw...), true
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, strings.TrimSpace(text))
		}
		return out, true
	default:
		return nil, false
	}
}

func kmsTags(value any) ([]kmssvc.Tag, bool) {
	if value == nil {
		return nil, true
	}
	raw, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]kmssvc.Tag, 0, len(raw))
	for _, item := range raw {
		tagMap, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		tag := kmssvc.Tag{
			TagKey:   kmsString(tagMap["TagKey"]),
			TagValue: kmsString(tagMap["TagValue"]),
		}
		if strings.TrimSpace(tag.TagKey) == "" {
			return nil, false
		}
		out = append(out, tag)
	}
	return out, true
}

func kmsTimePtr(value any) (*time.Time, bool) {
	if value == nil {
		return nil, true
	}
	switch raw := value.(type) {
	case string:
		text := strings.TrimSpace(raw)
		if text == "" {
			return nil, true
		}
		parsed, err := time.Parse(time.RFC3339, text)
		if err != nil {
			return nil, false
		}
		out := parsed.UTC()
		return &out, true
	default:
		return nil, false
	}
}

func kmsBlob(value any) ([]byte, bool) {
	switch raw := value.(type) {
	case nil:
		return nil, true
	case []byte:
		return append([]byte(nil), raw...), true
	case string:
		text := strings.TrimSpace(raw)
		if text == "" {
			return []byte{}, true
		}
		decoded, err := base64.StdEncoding.DecodeString(text)
		if err == nil {
			return decoded, true
		}
		decoded, err = base64.RawStdEncoding.DecodeString(text)
		if err == nil {
			return decoded, true
		}
		return []byte(text), true
	default:
		return nil, false
	}
}
