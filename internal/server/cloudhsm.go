package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	cloudhsmsvc "github.com/stackyard/stackyard/internal/services/cloudhsm"
)

type cloudhsmError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCloudHSMJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudHSMJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "cloudhsm")
	if !ok {
		respondCloudHSMError(w, status, code, msg)
		return true
	}

	action := parseCloudHSMTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondCloudHSMError(w, http.StatusBadRequest, "CloudHsmInvalidRequestException", "missing X-Amz-Target")
		return true
	}
	if _, known := cloudhsmOperationByName[action]; !known {
		respondCloudHSMError(w, http.StatusBadRequest, "CloudHsmInvalidRequestException", "unknown action")
		return true
	}

	payload, err := parseCloudHSMPayload(r)
	if err != nil {
		respondCloudHSMError(w, http.StatusBadRequest, "CloudHsmInvalidRequestException", "invalid JSON body")
		return true
	}

	switch action {
	case "CopyBackupToRegion":
		backup, err := s.cloudhsm.CopyBackupToRegion(
			cloudhsmString(payload["DestinationRegion"]),
			cloudhsmString(payload["BackupId"]),
			cloudhsmTagsPayload(payload["TagList"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"DestinationBackup": cloudhsmBackupPayload(backup),
		})
		return true

	case "CreateCluster":
		cluster, err := s.cloudhsm.CreateCluster(
			cloudhsmBackupRetentionPolicyPayload(payload["BackupRetentionPolicy"]),
			cloudhsmString(payload["HsmType"]),
			cloudhsmString(payload["SourceBackupId"]),
			cloudhsmStringSlice(payload["SubnetIds"]),
			cloudhsmString(payload["NetworkType"]),
			cloudhsmTagsPayload(payload["TagList"]),
			cloudhsmString(payload["Mode"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"Cluster": cloudhsmClusterPayload(cluster),
		})
		return true

	case "CreateHsm":
		hsm, err := s.cloudhsm.CreateHsm(
			cloudhsmString(payload["ClusterId"]),
			cloudhsmString(payload["AvailabilityZone"]),
			cloudhsmString(payload["IpAddress"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"Hsm": cloudhsmHsmPayload(hsm),
		})
		return true

	case "DeleteBackup":
		backup, err := s.cloudhsm.DeleteBackup(cloudhsmString(payload["BackupId"]))
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"Backup": cloudhsmBackupPayload(backup),
		})
		return true

	case "DeleteCluster":
		cluster, err := s.cloudhsm.DeleteCluster(cloudhsmString(payload["ClusterId"]))
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"Cluster": cloudhsmClusterPayload(cluster),
		})
		return true

	case "DeleteHsm":
		hsmID, err := s.cloudhsm.DeleteHsm(
			cloudhsmString(payload["ClusterId"]),
			cloudhsmString(payload["HsmId"]),
			cloudhsmString(payload["EniId"]),
			cloudhsmString(payload["EniIp"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"HsmId": hsmID,
		})
		return true

	case "DeleteResourcePolicy":
		resourceARN, policy, err := s.cloudhsm.DeleteResourcePolicy(cloudhsmString(payload["ResourceArn"]))
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"ResourceArn": resourceARN,
			"Policy":      policy,
		})
		return true

	case "DescribeBackups":
		backups, nextToken, err := s.cloudhsm.DescribeBackups(
			cloudhsmString(payload["NextToken"]),
			cloudhsmInt32(payload["MaxResults"]),
			cloudhsmFiltersPayload(payload["Filters"]),
			cloudhsmBoolPtr(payload["Shared"]),
			cloudhsmBoolPtr(payload["SortAscending"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Backups": cloudhsmBackupListPayload(backups),
		}
		if strings.TrimSpace(nextToken) != "" {
			response["NextToken"] = nextToken
		}
		respondCloudHSMJSON(w, http.StatusOK, response)
		return true

	case "DescribeClusters":
		clusters, nextToken, err := s.cloudhsm.DescribeClusters(
			cloudhsmFiltersPayload(payload["Filters"]),
			cloudhsmString(payload["NextToken"]),
			cloudhsmInt32(payload["MaxResults"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Clusters": cloudhsmClusterListPayload(clusters),
		}
		if strings.TrimSpace(nextToken) != "" {
			response["NextToken"] = nextToken
		}
		respondCloudHSMJSON(w, http.StatusOK, response)
		return true

	case "GetResourcePolicy":
		policy, err := s.cloudhsm.GetResourcePolicy(cloudhsmString(payload["ResourceArn"]))
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"Policy": policy,
		})
		return true

	case "InitializeCluster":
		state, stateMessage, err := s.cloudhsm.InitializeCluster(
			cloudhsmString(payload["ClusterId"]),
			cloudhsmString(payload["SignedCert"]),
			cloudhsmString(payload["TrustAnchor"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"State":        state,
			"StateMessage": stateMessage,
		})
		return true

	case "ListTags":
		tags, nextToken, err := s.cloudhsm.ListTags(
			cloudhsmString(payload["ResourceId"]),
			cloudhsmString(payload["NextToken"]),
			cloudhsmInt32(payload["MaxResults"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"TagList": cloudhsmTagListPayload(tags),
		}
		if strings.TrimSpace(nextToken) != "" {
			response["NextToken"] = nextToken
		}
		respondCloudHSMJSON(w, http.StatusOK, response)
		return true

	case "ModifyBackupAttributes":
		backup, err := s.cloudhsm.ModifyBackupAttributes(
			cloudhsmString(payload["BackupId"]),
			cloudhsmBool(payload["NeverExpires"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"Backup": cloudhsmBackupPayload(backup),
		})
		return true

	case "ModifyCluster":
		cluster, err := s.cloudhsm.ModifyCluster(
			cloudhsmBackupRetentionPolicyPayload(payload["BackupRetentionPolicy"]),
			cloudhsmString(payload["ClusterId"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"Cluster": cloudhsmClusterPayload(cluster),
		})
		return true

	case "PutResourcePolicy":
		resourceARN, policy, err := s.cloudhsm.PutResourcePolicy(
			cloudhsmString(payload["ResourceArn"]),
			cloudhsmString(payload["Policy"]),
		)
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"ResourceArn": resourceARN,
			"Policy":      policy,
		})
		return true

	case "RestoreBackup":
		backup, err := s.cloudhsm.RestoreBackup(cloudhsmString(payload["BackupId"]))
		if err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{
			"Backup": cloudhsmBackupPayload(backup),
		})
		return true

	case "TagResource":
		if err := s.cloudhsm.TagResource(
			cloudhsmString(payload["ResourceId"]),
			cloudhsmTagsPayload(payload["TagList"]),
		); err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UntagResource":
		if err := s.cloudhsm.UntagResource(
			cloudhsmString(payload["ResourceId"]),
			cloudhsmStringSlice(payload["TagKeyList"]),
		); err != nil {
			respondCloudHSMErrorForErr(w, err)
			return true
		}
		respondCloudHSMJSON(w, http.StatusOK, map[string]any{})
		return true
	}

	respondCloudHSMError(w, http.StatusBadRequest, "CloudHsmInvalidRequestException", "unsupported action")
	return true
}

func isCloudHSMJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "BaldrApiService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "BaldrApiService")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "cloudhsm" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".cloudhsm.") || strings.HasPrefix(host, "cloudhsm.") || strings.Contains(host, "cloudhsmv2")
}

func parseCloudHSMTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "BaldrApiService.") {
		return strings.TrimPrefix(target, "BaldrApiService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseCloudHSMPayload(r *http.Request) (map[string]any, error) {
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

func respondCloudHSMJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudHSMError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudHSMJSON(w, status, cloudhsmError{Type: code, Message: msg})
}

func respondCloudHSMErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, cloudhsmsvc.ErrInvalidParameter):
		respondCloudHSMError(w, http.StatusBadRequest, "CloudHsmInvalidRequestException", err.Error())
	case errors.Is(err, cloudhsmsvc.ErrNotFound):
		respondCloudHSMError(w, http.StatusBadRequest, "CloudHsmResourceNotFoundException", err.Error())
	default:
		respondCloudHSMError(w, http.StatusInternalServerError, "CloudHsmServiceException", err.Error())
	}
}

func cloudhsmString(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func cloudhsmInt32(value any) int32 {
	switch typed := value.(type) {
	case int:
		return int32(typed)
	case int32:
		return typed
	case int64:
		return int32(typed)
	case float64:
		return int32(typed)
	}
	return 0
}

func cloudhsmBool(value any) bool {
	boolValue, _ := value.(bool)
	return boolValue
}

func cloudhsmBoolPtr(value any) *bool {
	boolValue, ok := value.(bool)
	if !ok {
		return nil
	}
	return &boolValue
}

func cloudhsmStringSlice(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		text, ok := item.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	return out
}

func cloudhsmTagsPayload(value any) []cloudhsmsvc.Tag {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]cloudhsmsvc.Tag, 0, len(values))
	for _, item := range values {
		tag, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := cloudhsmString(tag["Key"])
		if key == "" {
			continue
		}
		out = append(out, cloudhsmsvc.Tag{
			Key:   key,
			Value: cloudhsmString(tag["Value"]),
		})
	}
	return out
}

func cloudhsmFiltersPayload(value any) map[string][]string {
	filters, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := map[string][]string{}
	for key, item := range filters {
		out[key] = cloudhsmStringSlice(item)
	}
	return out
}

func cloudhsmBackupRetentionPolicyPayload(value any) cloudhsmsvc.BackupRetentionPolicy {
	obj, ok := value.(map[string]any)
	if !ok {
		return cloudhsmsvc.BackupRetentionPolicy{}
	}
	return cloudhsmsvc.BackupRetentionPolicy{
		Type:  cloudhsmString(obj["Type"]),
		Value: cloudhsmString(obj["Value"]),
	}
}

func cloudhsmTagListPayload(tags []cloudhsmsvc.Tag) []map[string]string {
	out := make([]map[string]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, map[string]string{
			"Key":   tag.Key,
			"Value": tag.Value,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i]["Key"] < out[j]["Key"]
	})
	return out
}

func cloudhsmHsmPayload(hsm cloudhsmsvc.Hsm) map[string]any {
	return map[string]any{
		"AvailabilityZone": hsm.AvailabilityZone,
		"ClusterId":        hsm.ClusterID,
		"SubnetId":         hsm.SubnetID,
		"EniId":            hsm.EniID,
		"EniIp":            hsm.EniIP,
		"EniIpV6":          hsm.EniIPv6,
		"HsmId":            hsm.HsmID,
		"State":            hsm.State,
		"StateMessage":     hsm.StateMessage,
	}
}

func cloudhsmClusterPayload(cluster cloudhsmsvc.Cluster) map[string]any {
	hsms := make([]map[string]any, 0, len(cluster.Hsms))
	for _, hsm := range cluster.Hsms {
		hsms = append(hsms, cloudhsmHsmPayload(hsm))
	}

	return map[string]any{
		"BackupPolicy": cluster.BackupPolicy,
		"BackupRetentionPolicy": map[string]any{
			"Type":  cluster.BackupRetentionPolicy.Type,
			"Value": cluster.BackupRetentionPolicy.Value,
		},
		"ClusterId":       cluster.ClusterID,
		"CreateTimestamp": cluster.CreateTimestamp,
		"Hsms":            hsms,
		"HsmType":         cluster.HsmType,
		"PreCoPassword":   cluster.PreCoPassword,
		"SecurityGroup":   cluster.SecurityGroup,
		"SourceBackupId":  cluster.SourceBackupID,
		"State":           cluster.State,
		"StateMessage":    cluster.StateMessage,
		"SubnetMapping":   cluster.SubnetMapping,
		"VpcId":           cluster.VpcID,
		"NetworkType":     cluster.NetworkType,
		"Certificates": map[string]any{
			"ClusterCsr":                      cluster.Certificates.ClusterCsr,
			"HsmCertificate":                  cluster.Certificates.HsmCertificate,
			"AwsHardwareCertificate":          cluster.Certificates.AwsHardwareCertificate,
			"ManufacturerHardwareCertificate": cluster.Certificates.ManufacturerHardwareCertificate,
			"ClusterCertificate":              cluster.Certificates.ClusterCertificate,
		},
		"TagList": cloudhsmTagListPayload(cluster.TagList),
		"Mode":    cluster.Mode,
	}
}

func cloudhsmClusterListPayload(clusters []cloudhsmsvc.Cluster) []map[string]any {
	out := make([]map[string]any, 0, len(clusters))
	for _, cluster := range clusters {
		out = append(out, cloudhsmClusterPayload(cluster))
	}
	return out
}

func cloudhsmBackupPayload(backup cloudhsmsvc.Backup) map[string]any {
	out := map[string]any{
		"BackupId":        backup.BackupID,
		"BackupArn":       backup.BackupARN,
		"BackupState":     backup.BackupState,
		"ClusterId":       backup.ClusterID,
		"CreateTimestamp": backup.CreateTimestamp,
		"NeverExpires":    backup.NeverExpires,
		"SourceRegion":    backup.SourceRegion,
		"SourceBackup":    backup.SourceBackup,
		"SourceCluster":   backup.SourceCluster,
		"TagList":         cloudhsmTagListPayload(backup.TagList),
		"HsmType":         backup.HsmType,
		"Mode":            backup.Mode,
	}
	if !backup.CopyTimestamp.IsZero() {
		out["CopyTimestamp"] = backup.CopyTimestamp
	}
	if !backup.DeleteTimestamp.IsZero() {
		out["DeleteTimestamp"] = backup.DeleteTimestamp
	}
	return out
}

func cloudhsmBackupListPayload(backups []cloudhsmsvc.Backup) []map[string]any {
	out := make([]map[string]any, 0, len(backups))
	for _, backup := range backups {
		out = append(out, cloudhsmBackupPayload(backup))
	}
	return out
}
