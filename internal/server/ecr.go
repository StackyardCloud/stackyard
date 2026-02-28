package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	ecrsvc "github.com/stackyard/stackyard/internal/services/ecr"
)

type ecrError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleECRJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isECRJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ecr")
	if !ok {
		respondECRError(w, status, code, msg)
		return true
	}

	action := parseECRTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondECRError(w, http.StatusBadRequest, "InvalidParameterException", "missing X-Amz-Target")
		return true
	}
	if _, known := ecrOperationByName[action]; !known {
		respondECRError(w, http.StatusBadRequest, "InvalidParameterException", "unknown action")
		return true
	}

	payload, err := parseECRPayload(r)
	if err != nil {
		respondECRError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
		return true
	}

	switch action {
	case "GetAuthorizationToken":
		authorizationData, err := s.ecr.GetAuthorizationToken(ecrStringSlice(payload["registryIds"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(authorizationData))
		for _, item := range authorizationData {
			items = append(items, ecrAuthorizationDataPayload(item))
		}
		respondECRJSON(w, http.StatusOK, map[string]any{"authorizationData": items})
		return true
	case "CreateRepository":
		imageScanningConfiguration := ecrMap(payload["imageScanningConfiguration"])
		var scanOnPush *bool
		if raw, ok := imageScanningConfiguration["scanOnPush"]; ok {
			value, ok := ecrBool(raw)
			if !ok {
				respondECRError(w, http.StatusBadRequest, "InvalidParameterException", "imageScanningConfiguration.scanOnPush must be a boolean")
				return true
			}
			scanOnPush = &value
		}

		encryptionConfiguration := ecrMap(payload["encryptionConfiguration"])
		tags, err := ecrTagsToMap(payload["tags"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		repository, err := s.ecr.CreateRepository(
			ecrString(payload["repositoryName"]),
			ecrString(payload["imageTagMutability"]),
			scanOnPush,
			ecrString(encryptionConfiguration["encryptionType"]),
			ecrString(encryptionConfiguration["kmsKey"]),
			tags,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{"repository": ecrRepositoryPayload(repository)})
		return true
	case "DescribeRepositories":
		maxResults, _ := ecrInt32(payload["maxResults"])
		repositories, nextToken, err := s.ecr.DescribeRepositories(
			ecrStringSlice(payload["repositoryNames"]),
			ecrString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(repositories))
		for _, repository := range repositories {
			items = append(items, ecrRepositoryPayload(repository))
		}
		response := map[string]any{"repositories": items}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "DeleteRepository":
		force, _ := ecrBool(payload["force"])
		repository, err := s.ecr.DeleteRepository(ecrString(payload["repositoryName"]), force)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{"repository": ecrRepositoryPayload(repository)})
		return true
	case "ListTagsForResource":
		tags, err := s.ecr.ListTagsForResource(ecrString(payload["resourceArn"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{"tags": ecrMapToTags(tags)})
		return true
	case "TagResource":
		tags, err := ecrTagsToMap(payload["tags"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		if err := s.ecr.TagResource(ecrString(payload["resourceArn"]), tags); err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UntagResource":
		if err := s.ecr.UntagResource(ecrString(payload["resourceArn"]), ecrStringSlice(payload["tagKeys"])); err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{})
		return true
	case "SetRepositoryPolicy":
		force, _ := ecrBool(payload["force"])
		repository, policyText, err := s.ecr.SetRepositoryPolicy(
			ecrString(payload["repositoryName"]),
			ecrString(payload["policyText"]),
			force,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":     repository.RegistryID,
			"repositoryName": repository.RepositoryName,
			"policyText":     policyText,
		})
		return true
	case "GetRepositoryPolicy":
		repository, policyText, err := s.ecr.GetRepositoryPolicy(ecrString(payload["repositoryName"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":     repository.RegistryID,
			"repositoryName": repository.RepositoryName,
			"policyText":     policyText,
		})
		return true
	case "DeleteRepositoryPolicy":
		repository, policyText, err := s.ecr.DeleteRepositoryPolicy(ecrString(payload["repositoryName"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":     repository.RegistryID,
			"repositoryName": repository.RepositoryName,
			"policyText":     policyText,
		})
		return true
	case "PutImageTagMutability":
		repository, imageTagMutability, err := s.ecr.PutImageTagMutability(
			ecrString(payload["repositoryName"]),
			ecrString(payload["imageTagMutability"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":         repository.RegistryID,
			"repositoryName":     repository.RepositoryName,
			"imageTagMutability": imageTagMutability,
		})
		return true
	case "PutImageScanningConfiguration":
		cfg := ecrMap(payload["imageScanningConfiguration"])
		scanOnPush, ok := ecrBool(cfg["scanOnPush"])
		if !ok {
			respondECRError(w, http.StatusBadRequest, "InvalidParameterException", "imageScanningConfiguration.scanOnPush must be a boolean")
			return true
		}
		repository, updatedScanOnPush, err := s.ecr.PutImageScanningConfiguration(ecrString(payload["repositoryName"]), scanOnPush)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":     repository.RegistryID,
			"repositoryName": repository.RepositoryName,
			"imageScanningConfiguration": map[string]any{
				"scanOnPush": updatedScanOnPush,
			},
		})
		return true
	case "BatchGetRepositoryScanningConfiguration":
		configs, failures, err := s.ecr.BatchGetRepositoryScanningConfiguration(ecrStringSlice(payload["repositoryNames"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		configPayload := make([]map[string]any, 0, len(configs))
		for _, cfg := range configs {
			configPayload = append(configPayload, map[string]any{
				"repositoryName": cfg.RepositoryName,
				"repositoryArn":  cfg.RepositoryArn,
				"scanFrequency":  cfg.ScanFrequency,
				"scanOnPush":     cfg.ScanOnPush,
			})
		}
		failurePayload := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failurePayload = append(failurePayload, map[string]any{
				"repositoryName": failure.RepositoryName,
				"failureCode":    failure.FailureCode,
				"failureReason":  failure.FailureReason,
			})
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"scanningConfigurations": configPayload,
			"failures":               failurePayload,
		})
		return true
	case "InitiateLayerUpload":
		uploadID, partSize, err := s.ecr.InitiateLayerUpload(ecrString(payload["repositoryName"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"uploadId": uploadID,
			"partSize": partSize,
		})
		return true
	case "UploadLayerPart":
		partFirstByte, ok := ecrInt64(payload["partFirstByte"])
		if !ok {
			respondECRError(w, http.StatusBadRequest, "InvalidParameterException", "partFirstByte must be an integer")
			return true
		}
		partLastByte, ok := ecrInt64(payload["partLastByte"])
		if !ok {
			respondECRError(w, http.StatusBadRequest, "InvalidParameterException", "partLastByte must be an integer")
			return true
		}
		layerPartBlob, ok := ecrBlob(payload["layerPartBlob"])
		if !ok {
			respondECRError(w, http.StatusBadRequest, "InvalidParameterException", "layerPartBlob must be base64-encoded")
			return true
		}
		lastByteReceived, err := s.ecr.UploadLayerPart(
			ecrString(payload["repositoryName"]),
			ecrString(payload["uploadId"]),
			partFirstByte,
			partLastByte,
			layerPartBlob,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"repositoryName":   ecrString(payload["repositoryName"]),
			"uploadId":         ecrString(payload["uploadId"]),
			"registryId":       ecrsvc.DefaultAccountID,
			"lastByteReceived": lastByteReceived,
		})
		return true
	case "CompleteLayerUpload":
		repositoryName := ecrString(payload["repositoryName"])
		uploadID := ecrString(payload["uploadId"])
		layerDigest, err := s.ecr.CompleteLayerUpload(repositoryName, uploadID, ecrStringSlice(payload["layerDigests"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":     ecrsvc.DefaultAccountID,
			"repositoryName": repositoryName,
			"uploadId":       uploadID,
			"layerDigest":    layerDigest,
		})
		return true
	case "PutImage":
		image, err := s.ecr.PutImage(
			ecrString(payload["repositoryName"]),
			ecrString(payload["imageManifest"]),
			ecrString(payload["imageManifestMediaType"]),
			ecrString(payload["imageTag"]),
			ecrString(payload["imageDigest"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{"image": ecrImagePayload(image)})
		return true
	case "BatchCheckLayerAvailability":
		layers, failures, err := s.ecr.BatchCheckLayerAvailability(
			ecrString(payload["repositoryName"]),
			ecrStringSlice(payload["layerDigests"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		layerPayload := make([]map[string]any, 0, len(layers))
		for _, layer := range layers {
			layerPayload = append(layerPayload, ecrLayerPayload(layer))
		}
		failurePayload := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failurePayload = append(failurePayload, map[string]any{
				"layerDigest":   failure.LayerDigest,
				"failureCode":   failure.FailureCode,
				"failureReason": failure.FailureReason,
			})
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"layers":   layerPayload,
			"failures": failurePayload,
		})
		return true
	case "BatchGetImage":
		imageIDs, err := ecrImageIdentifiers(payload["imageIds"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		images, failures, err := s.ecr.BatchGetImage(
			ecrString(payload["repositoryName"]),
			imageIDs,
			ecrStringSlice(payload["acceptedMediaTypes"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		imagePayload := make([]map[string]any, 0, len(images))
		for _, image := range images {
			imagePayload = append(imagePayload, ecrImagePayload(image))
		}
		failurePayload := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failurePayload = append(failurePayload, map[string]any{
				"imageId":       ecrImageIdentifierPayload(failure.ImageID),
				"failureCode":   failure.FailureCode,
				"failureReason": failure.FailureReason,
			})
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"images":   imagePayload,
			"failures": failurePayload,
		})
		return true
	case "GetDownloadUrlForLayer":
		downloadURL, layerDigest, err := s.ecr.GetDownloadURLForLayer(
			ecrString(payload["repositoryName"]),
			ecrString(payload["layerDigest"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"downloadUrl": downloadURL,
			"layerDigest": layerDigest,
		})
		return true
	case "ListImages":
		maxResults, _ := ecrInt32(payload["maxResults"])
		tagStatus := ecrString(ecrMap(payload["filter"])["tagStatus"])
		imageIDs, nextToken, err := s.ecr.ListImages(
			ecrString(payload["repositoryName"]),
			tagStatus,
			ecrString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		ids := make([]map[string]any, 0, len(imageIDs))
		for _, id := range imageIDs {
			ids = append(ids, ecrImageIdentifierPayload(id))
		}
		response := map[string]any{"imageIds": ids}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "DescribeImages":
		imageIDs, err := ecrImageIdentifiers(payload["imageIds"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		maxResults, _ := ecrInt32(payload["maxResults"])
		tagStatus := ecrString(ecrMap(payload["filter"])["tagStatus"])
		imageDetails, nextToken, err := s.ecr.DescribeImages(
			ecrString(payload["repositoryName"]),
			imageIDs,
			tagStatus,
			ecrString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		details := make([]map[string]any, 0, len(imageDetails))
		for _, detail := range imageDetails {
			details = append(details, ecrImageDetailPayload(detail))
		}
		response := map[string]any{"imageDetails": details}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "BatchDeleteImage":
		imageIDs, err := ecrImageIdentifiers(payload["imageIds"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		deletedImageIDs, failures, err := s.ecr.BatchDeleteImage(
			ecrString(payload["repositoryName"]),
			imageIDs,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		deleted := make([]map[string]any, 0, len(deletedImageIDs))
		for _, id := range deletedImageIDs {
			deleted = append(deleted, ecrImageIdentifierPayload(id))
		}
		failurePayload := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failurePayload = append(failurePayload, map[string]any{
				"imageId":       ecrImageIdentifierPayload(failure.ImageID),
				"failureCode":   failure.FailureCode,
				"failureReason": failure.FailureReason,
			})
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"imageIds": deleted,
			"failures": failurePayload,
		})
		return true
	case "PutLifecyclePolicy":
		policy, err := s.ecr.PutLifecyclePolicy(
			ecrString(payload["repositoryName"]),
			ecrString(payload["lifecyclePolicyText"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":          policy.RegistryID,
			"repositoryName":      policy.RepositoryName,
			"lifecyclePolicyText": policy.LifecyclePolicyText,
		})
		return true
	case "GetLifecyclePolicy":
		policy, err := s.ecr.GetLifecyclePolicy(ecrString(payload["repositoryName"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":          policy.RegistryID,
			"repositoryName":      policy.RepositoryName,
			"lifecyclePolicyText": policy.LifecyclePolicyText,
			"lastEvaluatedAt":     ecrTimestamp(policy.LastEvaluatedAt),
		})
		return true
	case "DeleteLifecyclePolicy":
		policy, err := s.ecr.DeleteLifecyclePolicy(ecrString(payload["repositoryName"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":          policy.RegistryID,
			"repositoryName":      policy.RepositoryName,
			"lifecyclePolicyText": policy.LifecyclePolicyText,
			"lastEvaluatedAt":     ecrTimestamp(policy.LastEvaluatedAt),
		})
		return true
	case "StartLifecyclePolicyPreview":
		policy, status, err := s.ecr.StartLifecyclePolicyPreview(
			ecrString(payload["repositoryName"]),
			ecrString(payload["lifecyclePolicyText"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":          policy.RegistryID,
			"repositoryName":      policy.RepositoryName,
			"lifecyclePolicyText": policy.LifecyclePolicyText,
			"status":              status,
		})
		return true
	case "GetLifecyclePolicyPreview":
		imageIDs, err := ecrImageIdentifiers(payload["imageIds"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		maxResults, _ := ecrInt32(payload["maxResults"])
		tagStatus := ecrString(ecrMap(payload["filter"])["tagStatus"])
		policy, status, results, summary, nextToken, err := s.ecr.GetLifecyclePolicyPreview(
			ecrString(payload["repositoryName"]),
			imageIDs,
			tagStatus,
			ecrString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		previewResults := make([]map[string]any, 0, len(results))
		for _, result := range results {
			previewResults = append(previewResults, ecrLifecyclePreviewResultPayload(result))
		}
		response := map[string]any{
			"registryId":          policy.RegistryID,
			"repositoryName":      policy.RepositoryName,
			"lifecyclePolicyText": policy.LifecyclePolicyText,
			"status":              status,
			"previewResults":      previewResults,
			"summary": map[string]any{
				"expiringImageTotalCount": summary.ExpiringImageTotalCount,
			},
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "DescribeRegistry":
		registryID, replicationConfiguration, err := s.ecr.DescribeRegistry()
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":               registryID,
			"replicationConfiguration": ecrReplicationConfigurationPayload(replicationConfiguration),
		})
		return true
	case "GetRegistryPolicy":
		registryID, policyText, err := s.ecr.GetRegistryPolicy()
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId": registryID,
			"policyText": policyText,
		})
		return true
	case "PutRegistryPolicy":
		registryID, policyText, err := s.ecr.PutRegistryPolicy(ecrString(payload["policyText"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId": registryID,
			"policyText": policyText,
		})
		return true
	case "DeleteRegistryPolicy":
		registryID, policyText, err := s.ecr.DeleteRegistryPolicy()
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId": registryID,
			"policyText": policyText,
		})
		return true
	case "GetRegistryScanningConfiguration":
		registryID, scanningConfiguration, err := s.ecr.GetRegistryScanningConfiguration()
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":            registryID,
			"scanningConfiguration": ecrRegistryScanningConfigurationPayload(scanningConfiguration),
		})
		return true
	case "PutRegistryScanningConfiguration":
		rules, err := ecrRegistryScanningRules(payload["rules"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		configuration, err := s.ecr.PutRegistryScanningConfiguration(ecrString(payload["scanType"]), rules)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryScanningConfiguration": ecrRegistryScanningConfigurationPayload(configuration),
		})
		return true
	case "PutReplicationConfiguration":
		replicationConfiguration, err := ecrReplicationConfiguration(payload["replicationConfiguration"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		updated, err := s.ecr.PutReplicationConfiguration(replicationConfiguration)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"replicationConfiguration": ecrReplicationConfigurationPayload(updated),
		})
		return true
	case "GetAccountSetting":
		setting, err := s.ecr.GetAccountSetting(ecrString(payload["name"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"name":  setting.Name,
			"value": setting.Value,
		})
		return true
	case "PutAccountSetting":
		setting, err := s.ecr.PutAccountSetting(ecrString(payload["name"]), ecrString(payload["value"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"name":  setting.Name,
			"value": setting.Value,
		})
		return true
	case "DescribeImageReplicationStatus":
		imageID, err := ecrImageIdentifier(payload["imageId"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		outImageID, statuses, repositoryName, err := s.ecr.DescribeImageReplicationStatus(
			ecrString(payload["repositoryName"]),
			imageID,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		statusPayload := make([]map[string]any, 0, len(statuses))
		for _, status := range statuses {
			statusPayload = append(statusPayload, ecrImageReplicationStatusPayload(status))
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"imageId":             ecrImageIdentifierPayload(outImageID),
			"repositoryName":      repositoryName,
			"replicationStatuses": statusPayload,
		})
		return true
	case "CreatePullThroughCacheRule":
		rule, err := s.ecr.CreatePullThroughCacheRule(
			ecrString(payload["ecrRepositoryPrefix"]),
			ecrString(payload["upstreamRegistryUrl"]),
			ecrString(payload["credentialArn"]),
			ecrString(payload["upstreamRegistry"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		response := ecrPullThroughCacheRulePayload(rule)
		delete(response, "updatedAt")
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "DescribePullThroughCacheRules":
		maxResults, _ := ecrInt32(payload["maxResults"])
		rules, nextToken, err := s.ecr.DescribePullThroughCacheRules(
			ecrStringSlice(payload["ecrRepositoryPrefixes"]),
			ecrString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(rules))
		for _, rule := range rules {
			items = append(items, ecrPullThroughCacheRulePayload(rule))
		}
		response := map[string]any{"pullThroughCacheRules": items}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "UpdatePullThroughCacheRule":
		rule, err := s.ecr.UpdatePullThroughCacheRule(
			ecrString(payload["ecrRepositoryPrefix"]),
			ecrString(payload["credentialArn"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"credentialArn":       rule.CredentialArn,
			"ecrRepositoryPrefix": rule.ECRRepositoryPrefix,
			"registryId":          rule.RegistryID,
			"updatedAt":           ecrTimestamp(rule.UpdatedAt),
		})
		return true
	case "DeletePullThroughCacheRule":
		rule, err := s.ecr.DeletePullThroughCacheRule(ecrString(payload["ecrRepositoryPrefix"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		response := ecrPullThroughCacheRulePayload(rule)
		delete(response, "updatedAt")
		delete(response, "upstreamRegistry")
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "ValidatePullThroughCacheRule":
		rule, isValid, failure, err := s.ecr.ValidatePullThroughCacheRule(ecrString(payload["ecrRepositoryPrefix"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"credentialArn":       rule.CredentialArn,
			"ecrRepositoryPrefix": rule.ECRRepositoryPrefix,
			"isValid":             isValid,
			"registryId":          rule.RegistryID,
			"upstreamRegistryUrl": rule.UpstreamRegistryURL,
		}
		if failure != "" {
			response["failure"] = failure
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "CreateRepositoryCreationTemplate":
		input, err := ecrRepositoryCreationTemplateInput(payload, true)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		template, err := s.ecr.CreateRepositoryCreationTemplate(input)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":                 ecrsvc.DefaultAccountID,
			"repositoryCreationTemplate": ecrRepositoryCreationTemplatePayload(template),
		})
		return true
	case "DescribeRepositoryCreationTemplates":
		maxResults, _ := ecrInt32(payload["maxResults"])
		templates, nextToken, err := s.ecr.DescribeRepositoryCreationTemplates(
			ecrStringSlice(payload["prefixes"]),
			ecrString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(templates))
		for _, template := range templates {
			items = append(items, ecrRepositoryCreationTemplatePayload(template))
		}
		response := map[string]any{
			"registryId":                  ecrsvc.DefaultAccountID,
			"repositoryCreationTemplates": items,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "UpdateRepositoryCreationTemplate":
		input, err := ecrRepositoryCreationTemplateInput(payload, false)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		template, err := s.ecr.UpdateRepositoryCreationTemplate(input)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":                 ecrsvc.DefaultAccountID,
			"repositoryCreationTemplate": ecrRepositoryCreationTemplatePayload(template),
		})
		return true
	case "DeleteRepositoryCreationTemplate":
		template, err := s.ecr.DeleteRepositoryCreationTemplate(ecrString(payload["prefix"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":                 ecrsvc.DefaultAccountID,
			"repositoryCreationTemplate": ecrRepositoryCreationTemplatePayload(template),
		})
		return true
	case "PutSigningConfiguration":
		configuration, err := ecrSigningConfiguration(payload["signingConfiguration"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		updated, err := s.ecr.PutSigningConfiguration(configuration)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"signingConfiguration": ecrSigningConfigurationPayload(updated),
		})
		return true
	case "GetSigningConfiguration":
		registryID, configuration, err := s.ecr.GetSigningConfiguration()
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":           registryID,
			"signingConfiguration": ecrSigningConfigurationPayload(configuration),
		})
		return true
	case "DeleteSigningConfiguration":
		registryID, configuration, err := s.ecr.DeleteSigningConfiguration()
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"registryId":           registryID,
			"signingConfiguration": ecrSigningConfigurationPayload(configuration),
		})
		return true
	case "DescribeImageSigningStatus":
		imageID, err := ecrImageIdentifier(payload["imageId"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		outImageID, repositoryName, registryID, statuses, err := s.ecr.DescribeImageSigningStatus(
			ecrString(payload["repositoryName"]),
			imageID,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(statuses))
		for _, status := range statuses {
			items = append(items, ecrImageSigningStatusPayload(status))
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"imageId":         ecrImageIdentifierPayload(outImageID),
			"registryId":      registryID,
			"repositoryName":  repositoryName,
			"signingStatuses": items,
		})
		return true
	case "StartImageScan":
		imageID, err := ecrImageIdentifier(payload["imageId"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		outImageID, imageScanStatus, registryID, repositoryName, err := s.ecr.StartImageScan(
			ecrString(payload["repositoryName"]),
			imageID,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"imageId":         ecrImageIdentifierPayload(outImageID),
			"imageScanStatus": ecrImageScanStatusPayload(imageScanStatus),
			"registryId":      registryID,
			"repositoryName":  repositoryName,
		})
		return true
	case "DescribeImageScanFindings":
		imageID, err := ecrImageIdentifier(payload["imageId"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		maxResults, _ := ecrInt32(payload["maxResults"])
		outImageID, imageScanFindings, imageScanStatus, registryID, repositoryName, nextToken, err := s.ecr.DescribeImageScanFindings(
			ecrString(payload["repositoryName"]),
			imageID,
			ecrString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"imageId":           ecrImageIdentifierPayload(outImageID),
			"imageScanFindings": ecrImageScanFindingsPayload(imageScanFindings),
			"imageScanStatus":   ecrImageScanStatusPayload(imageScanStatus),
			"registryId":        registryID,
			"repositoryName":    repositoryName,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "RegisterPullTimeUpdateExclusion":
		exclusion, err := s.ecr.RegisterPullTimeUpdateExclusion(ecrString(payload["principalArn"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"createdAt":    ecrTimestamp(exclusion.CreatedAt),
			"principalArn": exclusion.PrincipalArn,
		})
		return true
	case "DeregisterPullTimeUpdateExclusion":
		principalArn, err := s.ecr.DeregisterPullTimeUpdateExclusion(ecrString(payload["principalArn"]))
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"principalArn": principalArn,
		})
		return true
	case "ListPullTimeUpdateExclusions":
		maxResults, _ := ecrInt32(payload["maxResults"])
		exclusions, nextToken, err := s.ecr.ListPullTimeUpdateExclusions(ecrString(payload["nextToken"]), maxResults)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"pullTimeUpdateExclusions": exclusions,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "ListImageReferrers":
		subjectDigest := ecrString(ecrMap(payload["subjectId"])["imageDigest"])
		filter := ecrMap(payload["filter"])
		maxResults, _ := ecrInt32(payload["maxResults"])
		referrers, nextToken, err := s.ecr.ListImageReferrers(
			ecrString(payload["repositoryName"]),
			subjectDigest,
			ecrString(filter["artifactStatus"]),
			ecrStringSlice(filter["artifactTypes"]),
			ecrString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(referrers))
		for _, referrer := range referrers {
			items = append(items, ecrImageReferrerPayload(referrer))
		}
		response := map[string]any{
			"referrers": items,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECRJSON(w, http.StatusOK, response)
		return true
	case "UpdateImageStorageClass":
		imageID, err := ecrImageIdentifier(payload["imageId"])
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		outImageID, imageStatus, registryID, repositoryName, err := s.ecr.UpdateImageStorageClass(
			ecrString(payload["repositoryName"]),
			imageID,
			ecrString(payload["targetStorageClass"]),
		)
		if err != nil {
			respondECRErrorForErr(w, err)
			return true
		}
		respondECRJSON(w, http.StatusOK, map[string]any{
			"imageId":        ecrImageIdentifierPayload(outImageID),
			"imageStatus":    imageStatus,
			"registryId":     registryID,
			"repositoryName": repositoryName,
		})
		return true
	default:
		respondECRError(w, http.StatusNotImplemented, "NotImplementedException", action+" is not implemented")
		return true
	}
}

func respondECRJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondECRError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondECRJSON(w, status, ecrError{Type: code, Message: msg})
}

func respondECRErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ecrsvc.ErrInvalidParameter):
		respondECRError(w, http.StatusBadRequest, "InvalidParameterException", err.Error())
	case errors.Is(err, ecrsvc.ErrRepositoryAlreadyExists):
		respondECRError(w, http.StatusBadRequest, "RepositoryAlreadyExistsException", err.Error())
	case errors.Is(err, ecrsvc.ErrRepositoryNotFound):
		respondECRError(w, http.StatusBadRequest, "RepositoryNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrRepositoryPolicyNotFound):
		respondECRError(w, http.StatusBadRequest, "RepositoryPolicyNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrRegistryPolicyNotFound):
		respondECRError(w, http.StatusBadRequest, "RegistryPolicyNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrUploadNotFound):
		respondECRError(w, http.StatusBadRequest, "UploadNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrLayerNotFound):
		respondECRError(w, http.StatusBadRequest, "LayersNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrImageTagImmutable):
		respondECRError(w, http.StatusBadRequest, "ImageTagAlreadyExistsException", err.Error())
	case errors.Is(err, ecrsvc.ErrImageDigestDoesNotMatch):
		respondECRError(w, http.StatusBadRequest, "ImageDigestDoesNotMatchException", err.Error())
	case errors.Is(err, ecrsvc.ErrImageNotFound):
		respondECRError(w, http.StatusBadRequest, "ImageNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrScanNotFound):
		respondECRError(w, http.StatusBadRequest, "ScanNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrLifecyclePolicyNotFound):
		respondECRError(w, http.StatusBadRequest, "LifecyclePolicyNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrLifecyclePreviewNotFound):
		respondECRError(w, http.StatusBadRequest, "LifecyclePolicyPreviewNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrPullThroughRuleExists):
		respondECRError(w, http.StatusBadRequest, "PullThroughCacheRuleAlreadyExistsException", err.Error())
	case errors.Is(err, ecrsvc.ErrPullThroughRuleNotFound):
		respondECRError(w, http.StatusBadRequest, "PullThroughCacheRuleNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrTemplateAlreadyExists):
		respondECRError(w, http.StatusBadRequest, "TemplateAlreadyExistsException", err.Error())
	case errors.Is(err, ecrsvc.ErrTemplateNotFound):
		respondECRError(w, http.StatusBadRequest, "TemplateNotFoundException", err.Error())
	case errors.Is(err, ecrsvc.ErrSigningConfigNotFound):
		respondECRError(w, http.StatusBadRequest, "SigningConfigurationNotFoundException", err.Error())
	default:
		respondECRError(w, http.StatusInternalServerError, "ServerException", err.Error())
	}
}

func isECRJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AmazonEC2ContainerRegistry_V20150921.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "AmazonEC2ContainerRegistry")
	}
	return false
}

func parseECRTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AmazonEC2ContainerRegistry_V20150921.") {
		return strings.TrimPrefix(target, "AmazonEC2ContainerRegistry_V20150921.")
	}
	if strings.HasPrefix(target, "AmazonEC2ContainerRegistry.") {
		return strings.TrimPrefix(target, "AmazonEC2ContainerRegistry.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseECRPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return obj, nil
}

func ecrRepositoryPayload(repository ecrsvc.Repository) map[string]any {
	out := map[string]any{
		"repositoryArn":      repository.RepositoryArn,
		"registryId":         repository.RegistryID,
		"repositoryName":     repository.RepositoryName,
		"repositoryUri":      repository.RepositoryURI,
		"createdAt":          ecrTimestamp(repository.CreatedAt),
		"imageTagMutability": repository.ImageTagMutability,
		"imageScanningConfiguration": map[string]any{
			"scanOnPush": repository.ImageScanningScanOnPush,
		},
		"encryptionConfiguration": map[string]any{
			"encryptionType": repository.EncryptionType,
		},
	}
	if repository.KMSKey != "" {
		out["encryptionConfiguration"].(map[string]any)["kmsKey"] = repository.KMSKey
	}
	return out
}

func ecrImagePayload(image ecrsvc.Image) map[string]any {
	out := map[string]any{
		"registryId":             image.RegistryID,
		"repositoryName":         image.RepositoryName,
		"imageId":                ecrImageIdentifierPayload(image.ImageID),
		"imageManifest":          image.ImageManifest,
		"imageManifestMediaType": image.ImageManifestMediaType,
	}
	if image.ImageManifest == "" {
		delete(out, "imageManifest")
	}
	if image.ImageManifestMediaType == "" {
		delete(out, "imageManifestMediaType")
	}
	return out
}

func ecrImageIdentifierPayload(imageID ecrsvc.ImageIdentifier) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(imageID.ImageDigest) != "" {
		out["imageDigest"] = imageID.ImageDigest
	}
	if strings.TrimSpace(imageID.ImageTag) != "" {
		out["imageTag"] = imageID.ImageTag
	}
	return out
}

func ecrLayerPayload(layer ecrsvc.Layer) map[string]any {
	out := map[string]any{
		"layerDigest":       layer.LayerDigest,
		"layerAvailability": layer.LayerAvailability,
	}
	if layer.LayerSize > 0 {
		out["layerSize"] = layer.LayerSize
	}
	if layer.MediaType != "" {
		out["mediaType"] = layer.MediaType
	}
	return out
}

func ecrImageDetailPayload(detail ecrsvc.ImageDetail) map[string]any {
	out := map[string]any{
		"imageDigest":            detail.ImageDigest,
		"imageManifestMediaType": detail.ImageManifestMediaType,
		"imageSizeInBytes":       detail.ImageSizeInBytes,
		"imagePushedAt":          ecrTimestamp(detail.ImagePushedAt),
		"imageTags":              detail.ImageTags,
		"registryId":             detail.RegistryID,
		"repositoryName":         detail.RepositoryName,
	}
	if len(detail.ImageTags) == 0 {
		out["imageTags"] = []string{}
	}
	return out
}

func ecrLifecyclePreviewResultPayload(result ecrsvc.LifecyclePolicyPreviewResult) map[string]any {
	return map[string]any{
		"action": map[string]any{
			"type": result.ActionType,
		},
		"appliedRulePriority": result.AppliedRulePriority,
		"imageDigest":         result.ImageDigest,
		"imagePushedAt":       ecrTimestamp(result.ImagePushedAt),
		"imageTags":           result.ImageTags,
	}
}

func ecrRegistryScanningConfigurationPayload(configuration ecrsvc.RegistryScanningConfiguration) map[string]any {
	rules := make([]map[string]any, 0, len(configuration.Rules))
	for _, rule := range configuration.Rules {
		filters := make([]map[string]any, 0, len(rule.RepositoryFilters))
		for _, filter := range rule.RepositoryFilters {
			filters = append(filters, map[string]any{
				"filter":     filter.Filter,
				"filterType": filter.FilterType,
			})
		}
		rules = append(rules, map[string]any{
			"repositoryFilters": filters,
			"scanFrequency":     rule.ScanFrequency,
		})
	}
	return map[string]any{
		"scanType": configuration.ScanType,
		"rules":    rules,
	}
}

func ecrRegistryScanningRules(v any) ([]ecrsvc.RegistryScanningRule, error) {
	if v == nil {
		return nil, nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil, ecrsvc.ErrInvalidParameter
	}

	out := make([]ecrsvc.RegistryScanningRule, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecrsvc.ErrInvalidParameter
		}
		repositoryFilters, err := ecrScanningRepositoryFilters(entry["repositoryFilters"])
		if err != nil {
			return nil, err
		}
		out = append(out, ecrsvc.RegistryScanningRule{
			RepositoryFilters: repositoryFilters,
			ScanFrequency:     ecrString(entry["scanFrequency"]),
		})
	}
	return out, nil
}

func ecrScanningRepositoryFilters(v any) ([]ecrsvc.ScanningRepositoryFilter, error) {
	items, ok := v.([]any)
	if !ok {
		return nil, ecrsvc.ErrInvalidParameter
	}
	out := make([]ecrsvc.ScanningRepositoryFilter, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecrsvc.ErrInvalidParameter
		}
		out = append(out, ecrsvc.ScanningRepositoryFilter{
			Filter:     ecrString(entry["filter"]),
			FilterType: ecrString(entry["filterType"]),
		})
	}
	return out, nil
}

func ecrReplicationConfiguration(v any) (ecrsvc.ReplicationConfiguration, error) {
	entry, ok := v.(map[string]any)
	if !ok {
		return ecrsvc.ReplicationConfiguration{}, ecrsvc.ErrInvalidParameter
	}

	items, ok := entry["rules"].([]any)
	if !ok {
		return ecrsvc.ReplicationConfiguration{}, ecrsvc.ErrInvalidParameter
	}
	rules := make([]ecrsvc.ReplicationRule, 0, len(items))
	for _, item := range items {
		rule, ok := item.(map[string]any)
		if !ok {
			return ecrsvc.ReplicationConfiguration{}, ecrsvc.ErrInvalidParameter
		}

		destinationItems, ok := rule["destinations"].([]any)
		if !ok {
			return ecrsvc.ReplicationConfiguration{}, ecrsvc.ErrInvalidParameter
		}
		destinations := make([]ecrsvc.ReplicationDestination, 0, len(destinationItems))
		for _, d := range destinationItems {
			destination, ok := d.(map[string]any)
			if !ok {
				return ecrsvc.ReplicationConfiguration{}, ecrsvc.ErrInvalidParameter
			}
			destinations = append(destinations, ecrsvc.ReplicationDestination{
				Region:     ecrString(destination["region"]),
				RegistryID: ecrString(destination["registryId"]),
			})
		}

		repositoryFilters := make([]ecrsvc.RepositoryFilter, 0)
		if rawFilters, has := rule["repositoryFilters"]; has && rawFilters != nil {
			filterItems, ok := rawFilters.([]any)
			if !ok {
				return ecrsvc.ReplicationConfiguration{}, ecrsvc.ErrInvalidParameter
			}
			for _, f := range filterItems {
				filter, ok := f.(map[string]any)
				if !ok {
					return ecrsvc.ReplicationConfiguration{}, ecrsvc.ErrInvalidParameter
				}
				repositoryFilters = append(repositoryFilters, ecrsvc.RepositoryFilter{
					Filter:     ecrString(filter["filter"]),
					FilterType: ecrString(filter["filterType"]),
				})
			}
		}

		rules = append(rules, ecrsvc.ReplicationRule{
			Destinations:      destinations,
			RepositoryFilters: repositoryFilters,
		})
	}
	return ecrsvc.ReplicationConfiguration{Rules: rules}, nil
}

func ecrReplicationConfigurationPayload(configuration ecrsvc.ReplicationConfiguration) map[string]any {
	rules := make([]map[string]any, 0, len(configuration.Rules))
	for _, rule := range configuration.Rules {
		destinations := make([]map[string]any, 0, len(rule.Destinations))
		for _, destination := range rule.Destinations {
			destinations = append(destinations, map[string]any{
				"region":     destination.Region,
				"registryId": destination.RegistryID,
			})
		}

		repositoryFilters := make([]map[string]any, 0, len(rule.RepositoryFilters))
		for _, filter := range rule.RepositoryFilters {
			repositoryFilters = append(repositoryFilters, map[string]any{
				"filter":     filter.Filter,
				"filterType": filter.FilterType,
			})
		}

		rulePayload := map[string]any{"destinations": destinations}
		if len(repositoryFilters) > 0 {
			rulePayload["repositoryFilters"] = repositoryFilters
		}
		rules = append(rules, rulePayload)
	}
	return map[string]any{"rules": rules}
}

func ecrImageIdentifier(v any) (ecrsvc.ImageIdentifier, error) {
	if v == nil {
		return ecrsvc.ImageIdentifier{}, ecrsvc.ErrInvalidParameter
	}
	entry, ok := v.(map[string]any)
	if !ok {
		return ecrsvc.ImageIdentifier{}, ecrsvc.ErrInvalidParameter
	}
	return ecrsvc.ImageIdentifier{
		ImageDigest: ecrString(entry["imageDigest"]),
		ImageTag:    ecrString(entry["imageTag"]),
	}, nil
}

func ecrImageReplicationStatusPayload(status ecrsvc.ImageReplicationStatus) map[string]any {
	out := map[string]any{
		"region":     status.Region,
		"registryId": status.RegistryID,
		"status":     status.Status,
	}
	if status.FailureCode != "" {
		out["failureCode"] = status.FailureCode
	}
	return out
}

func ecrPullThroughCacheRulePayload(rule ecrsvc.PullThroughCacheRule) map[string]any {
	out := map[string]any{
		"createdAt":           ecrTimestamp(rule.CreatedAt),
		"ecrRepositoryPrefix": rule.ECRRepositoryPrefix,
		"registryId":          rule.RegistryID,
		"updatedAt":           ecrTimestamp(rule.UpdatedAt),
		"upstreamRegistry":    rule.UpstreamRegistry,
		"upstreamRegistryUrl": rule.UpstreamRegistryURL,
	}
	if rule.CredentialArn != "" {
		out["credentialArn"] = rule.CredentialArn
	}
	return out
}

func ecrRepositoryCreationTemplateInput(payload map[string]any, requireAppliedFor bool) (ecrsvc.RepositoryCreationTemplateInput, error) {
	input := ecrsvc.RepositoryCreationTemplateInput{
		Prefix: ecrString(payload["prefix"]),
	}
	if input.Prefix == "" {
		return ecrsvc.RepositoryCreationTemplateInput{}, ecrsvc.ErrInvalidParameter
	}

	if rawAppliedFor, has := payload["appliedFor"]; has {
		appliedFor, err := ecrStringSliceStrict(rawAppliedFor)
		if err != nil {
			return ecrsvc.RepositoryCreationTemplateInput{}, err
		}
		input.AppliedFor = appliedFor
		input.AppliedForSet = true
	}
	if requireAppliedFor && !input.AppliedForSet {
		return ecrsvc.RepositoryCreationTemplateInput{}, ecrsvc.ErrInvalidParameter
	}

	if raw, has := payload["customRoleArn"]; has {
		value := ecrString(raw)
		input.CustomRoleArn = &value
	}
	if raw, has := payload["description"]; has {
		value := ecrString(raw)
		input.Description = &value
	}
	if raw, has := payload["imageTagMutability"]; has {
		value := ecrString(raw)
		input.ImageTagMutability = &value
	}
	if raw, has := payload["lifecyclePolicy"]; has {
		value := ecrString(raw)
		input.LifecyclePolicy = &value
	}
	if raw, has := payload["repositoryPolicy"]; has {
		value := ecrString(raw)
		input.RepositoryPolicy = &value
	}
	if raw, has := payload["resourceTags"]; has {
		tags, err := ecrTagsToMap(raw)
		if err != nil {
			return ecrsvc.RepositoryCreationTemplateInput{}, err
		}
		input.ResourceTags = tags
		input.ResourceTagsSet = true
	}
	if raw, has := payload["encryptionConfiguration"]; has {
		config := ecrMap(raw)
		if len(config) == 0 {
			return ecrsvc.RepositoryCreationTemplateInput{}, ecrsvc.ErrInvalidParameter
		}
		if rawType, hasType := config["encryptionType"]; hasType {
			value := ecrString(rawType)
			input.EncryptionType = &value
		}
		if rawKMSKey, hasKey := config["kmsKey"]; hasKey {
			value := ecrString(rawKMSKey)
			input.KMSKey = &value
		}
	}

	return input, nil
}

func ecrRepositoryCreationTemplatePayload(template ecrsvc.RepositoryCreationTemplate) map[string]any {
	out := map[string]any{
		"appliedFor":         append([]string(nil), template.AppliedFor...),
		"createdAt":          ecrTimestamp(template.CreatedAt),
		"imageTagMutability": template.ImageTagMutability,
		"prefix":             template.Prefix,
		"updatedAt":          ecrTimestamp(template.UpdatedAt),
	}
	if template.CustomRoleArn != "" {
		out["customRoleArn"] = template.CustomRoleArn
	}
	if template.Description != "" {
		out["description"] = template.Description
	}
	if template.EncryptionType != "" {
		enc := map[string]any{"encryptionType": template.EncryptionType}
		if template.KMSKey != "" {
			enc["kmsKey"] = template.KMSKey
		}
		out["encryptionConfiguration"] = enc
	}
	if template.LifecyclePolicy != "" {
		out["lifecyclePolicy"] = template.LifecyclePolicy
	}
	if template.RepositoryPolicy != "" {
		out["repositoryPolicy"] = template.RepositoryPolicy
	}
	if len(template.ResourceTags) > 0 {
		out["resourceTags"] = ecrMapToTags(template.ResourceTags)
	} else {
		out["resourceTags"] = []map[string]any{}
	}
	return out
}

func ecrSigningConfiguration(v any) (ecrsvc.SigningConfiguration, error) {
	entry, ok := v.(map[string]any)
	if !ok {
		return ecrsvc.SigningConfiguration{}, ecrsvc.ErrInvalidParameter
	}
	rawRules, ok := entry["rules"].([]any)
	if !ok {
		return ecrsvc.SigningConfiguration{}, ecrsvc.ErrInvalidParameter
	}

	rules := make([]ecrsvc.SigningRule, 0, len(rawRules))
	for _, rawRule := range rawRules {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			return ecrsvc.SigningConfiguration{}, ecrsvc.ErrInvalidParameter
		}
		signingProfileArn := ecrString(rule["signingProfileArn"])
		if signingProfileArn == "" {
			return ecrsvc.SigningConfiguration{}, ecrsvc.ErrInvalidParameter
		}

		repositoryFilters := make([]ecrsvc.SigningRepositoryFilter, 0)
		if rawFilters, has := rule["repositoryFilters"]; has && rawFilters != nil {
			filterItems, ok := rawFilters.([]any)
			if !ok {
				return ecrsvc.SigningConfiguration{}, ecrsvc.ErrInvalidParameter
			}
			for _, rawFilter := range filterItems {
				filter, ok := rawFilter.(map[string]any)
				if !ok {
					return ecrsvc.SigningConfiguration{}, ecrsvc.ErrInvalidParameter
				}
				repositoryFilters = append(repositoryFilters, ecrsvc.SigningRepositoryFilter{
					Filter:     ecrString(filter["filter"]),
					FilterType: ecrString(filter["filterType"]),
				})
			}
		}

		rules = append(rules, ecrsvc.SigningRule{
			SigningProfileArn: signingProfileArn,
			RepositoryFilters: repositoryFilters,
		})
	}
	return ecrsvc.SigningConfiguration{Rules: rules}, nil
}

func ecrSigningConfigurationPayload(configuration ecrsvc.SigningConfiguration) map[string]any {
	rules := make([]map[string]any, 0, len(configuration.Rules))
	for _, rule := range configuration.Rules {
		item := map[string]any{
			"signingProfileArn": rule.SigningProfileArn,
		}
		if len(rule.RepositoryFilters) > 0 {
			repositoryFilters := make([]map[string]any, 0, len(rule.RepositoryFilters))
			for _, filter := range rule.RepositoryFilters {
				repositoryFilters = append(repositoryFilters, map[string]any{
					"filter":     filter.Filter,
					"filterType": filter.FilterType,
				})
			}
			item["repositoryFilters"] = repositoryFilters
		}
		rules = append(rules, item)
	}
	return map[string]any{"rules": rules}
}

func ecrImageSigningStatusPayload(status ecrsvc.ImageSigningStatus) map[string]any {
	out := map[string]any{
		"signingProfileArn": status.SigningProfileArn,
		"status":            status.Status,
	}
	if status.FailureCode != "" {
		out["failureCode"] = status.FailureCode
	}
	if status.FailureReason != "" {
		out["failureReason"] = status.FailureReason
	}
	return out
}

func ecrImageScanStatusPayload(status ecrsvc.ImageScanStatus) map[string]any {
	out := map[string]any{
		"status": status.Status,
	}
	if status.Description != "" {
		out["description"] = status.Description
	}
	return out
}

func ecrImageScanFindingsPayload(findings ecrsvc.ImageScanFindings) map[string]any {
	items := make([]map[string]any, 0, len(findings.Findings))
	for _, finding := range findings.Findings {
		attributes := make([]map[string]any, 0, len(finding.Attributes))
		for _, attribute := range finding.Attributes {
			attributes = append(attributes, map[string]any{
				"key":   attribute.Key,
				"value": attribute.Value,
			})
		}
		items = append(items, map[string]any{
			"attributes":  attributes,
			"description": finding.Description,
			"name":        finding.Name,
			"severity":    finding.Severity,
			"uri":         finding.URI,
		})
	}
	return map[string]any{
		"findingSeverityCounts":        findings.FindingSeverityCounts,
		"findings":                     items,
		"imageScanCompletedAt":         ecrTimestamp(findings.ImageScanCompletedAt),
		"vulnerabilitySourceUpdatedAt": ecrTimestamp(findings.VulnerabilitySourceUpdatedAt),
	}
}

func ecrImageReferrerPayload(referrer ecrsvc.ImageReferrer) map[string]any {
	out := map[string]any{
		"digest":    referrer.Digest,
		"mediaType": referrer.MediaType,
		"size":      referrer.Size,
		"status":    referrer.Status,
	}
	if len(referrer.Annotations) > 0 {
		out["annotations"] = referrer.Annotations
	}
	if referrer.ArtifactType != "" {
		out["artifactType"] = referrer.ArtifactType
	}
	return out
}

func ecrAuthorizationDataPayload(item ecrsvc.AuthorizationData) map[string]any {
	return map[string]any{
		"authorizationToken": item.AuthorizationToken,
		"expiresAt":          ecrTimestamp(item.ExpiresAt),
		"proxyEndpoint":      item.ProxyEndpoint,
	}
}

func ecrMapToTags(tags map[string]string) []map[string]any {
	if len(tags) == 0 {
		return []map[string]any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func ecrTagsToMap(v any) (map[string]string, error) {
	items, ok := v.([]any)
	if !ok {
		if v == nil {
			return map[string]string{}, nil
		}
		return nil, ecrsvc.ErrInvalidParameter
	}
	out := make(map[string]string, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecrsvc.ErrInvalidParameter
		}
		key := ecrString(entry["Key"])
		if key == "" {
			key = ecrString(entry["key"])
		}
		if key == "" {
			return nil, ecrsvc.ErrInvalidParameter
		}
		value := ecrString(entry["Value"])
		if value == "" {
			value = ecrString(entry["value"])
		}
		out[key] = value
	}
	return out, nil
}

func ecrImageIdentifiers(v any) ([]ecrsvc.ImageIdentifier, error) {
	items, ok := v.([]any)
	if !ok {
		if v == nil {
			return nil, nil
		}
		return nil, ecrsvc.ErrInvalidParameter
	}
	out := make([]ecrsvc.ImageIdentifier, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecrsvc.ErrInvalidParameter
		}
		out = append(out, ecrsvc.ImageIdentifier{
			ImageDigest: ecrString(entry["imageDigest"]),
			ImageTag:    ecrString(entry["imageTag"]),
		})
	}
	return out, nil
}

func ecrMap(v any) map[string]any {
	obj, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return obj
}

func ecrString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func ecrStringSlice(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := ecrString(item)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func ecrStringSliceStrict(v any) ([]string, error) {
	items, ok := v.([]any)
	if !ok {
		return nil, ecrsvc.ErrInvalidParameter
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value, ok := item.(string)
		if !ok {
			return nil, ecrsvc.ErrInvalidParameter
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, ecrsvc.ErrInvalidParameter
		}
		out = append(out, trimmed)
	}
	return out, nil
}

func ecrBlob(v any) ([]byte, bool) {
	switch value := v.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return []byte{}, true
		}
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, false
		}
		return decoded, true
	case []byte:
		return append([]byte(nil), value...), true
	default:
		return nil, false
	}
}

func ecrBool(v any) (bool, bool) {
	b, ok := v.(bool)
	if !ok {
		return false, false
	}
	return b, true
}

func ecrInt32(v any) (int32, bool) {
	switch value := v.(type) {
	case float64:
		return int32(value), true
	case int:
		return int32(value), true
	case int32:
		return value, true
	case int64:
		return int32(value), true
	default:
		return 0, false
	}
}

func ecrInt64(v any) (int64, bool) {
	switch value := v.(type) {
	case float64:
		return int64(value), true
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		if strings.TrimSpace(value) == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func ecrTimestamp(t time.Time) float64 {
	if t.IsZero() {
		return 0
	}
	return float64(t.UnixNano()) / float64(time.Second)
}
