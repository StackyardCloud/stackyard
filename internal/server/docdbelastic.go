package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// handleDocDBElasticRESTRouter serves Amazon DocumentDB Elastic Clusters
// (REST-JSON, signing name docdb-elastic).
func (s *Server) handleDocDBElasticRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDocDBElasticCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > dbeMaxBody {
		respondDocDBElasticError(w, dbeValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "docdb-elastic"); !ok {
		respondDocDBElasticError(w, &dbeError{status: status, code: code, msg: msg})
		return true
	}
	out, err := s.docdbElasticRoute(r)
	if err != nil {
		respondDocDBElasticError(w, err)
		return true
	}
	raw, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
	return true
}

func isDocDBElasticCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if service != "docdb-elastic" && !(service == "" && (strings.HasPrefix(host, "docdb-elastic.") || strings.Contains(host, ".docdb-elastic."))) {
		return false
	}
	path := rawRequestPath(r)
	for _, p := range []string{"/cluster", "/clusters", "/cluster-snapshot", "/cluster-snapshots", "/pending-action", "/pending-actions"} {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return strings.HasPrefix(path, "/tags/")
}

func respondDocDBElasticError(w http.ResponseWriter, e *dbeError) {
	body := map[string]any{"message": e.msg}
	switch e.code {
	case "ValidationException":
		body["reason"] = "fieldValidationFailed"
	case "ResourceNotFoundException", "ConflictException":
		body["resourceId"], body["resourceType"] = e.resourceID, e.resourceType
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

// dbeUnknown reports a path or method no docdb-elastic operation matches.
func dbeUnknown(r *http.Request) *dbeError {
	return &dbeError{status: http.StatusNotFound, code: "UnknownOperationException", msg: "no DocumentDB Elastic operation matches " + r.Method + " " + rawRequestPath(r)}
}

// dbeTail returns the unescaped path after prefix and a trailing action
// segment (start, stop, copy, restore) when one of actions ends it. ARNs carry
// '/' and ':', which clients may or may not percent-encode.
func dbeTail(path, prefix string, actions ...string) (string, string, bool) {
	rest, ok := strings.CutPrefix(path, prefix)
	if !ok || rest == "" {
		return "", "", false
	}
	resource, err := url.PathUnescape(rest)
	if err != nil {
		return "", "", false
	}
	for _, a := range actions {
		if arn, found := strings.CutSuffix(resource, "/"+a); found && arn != "" {
			return arn, a, true
		}
	}
	return resource, "", true
}

func (s *Server) docdbElasticRoute(r *http.Request) (map[string]any, *dbeError) {
	path, q := rawRequestPath(r), r.URL.Query()
	switch {
	case path == "/cluster" && r.Method == http.MethodPost:
		return s.dbeCreateCluster(r)
	case path == "/clusters" && r.Method == http.MethodGet:
		limit, err := dbeLimit(q, 1, 100, 100)
		if err != nil {
			return nil, err
		}
		return s.docdbElastic.listClusters(q.Get("nextToken"), limit)
	case path == "/cluster-snapshot" && r.Method == http.MethodPost:
		in, err := dbeDecode(r, "CreateClusterSnapshotInput")
		if err != nil {
			return nil, err
		}
		name := dbeStr(in, "snapshotName")
		if err := dbeCheckName("snapshotName", name); err != nil {
			return nil, err
		}
		tags := dbeStringMap(in["tags"])
		if err := dbeCheckTags(tags); err != nil {
			return nil, err
		}
		return s.docdbElastic.createSnapshot(dbeStr(in, "clusterArn"), name, tags)
	case path == "/cluster-snapshots" && r.Method == http.MethodGet:
		limit, err := dbeLimit(q, 20, 100, 100)
		if err != nil {
			return nil, err
		}
		kind := strings.ToUpper(q.Get("snapshotType"))
		if kind != "" && kind != "MANUAL" && kind != "AUTOMATED" {
			return nil, dbeValidation("snapshotType must be manual or automated.")
		}
		return s.docdbElastic.listSnapshots(q.Get("clusterArn"), kind, q.Get("nextToken"), limit)
	case path == "/pending-action" && r.Method == http.MethodPost:
		in, err := dbeDecode(r, "ApplyPendingMaintenanceActionInput")
		if err != nil {
			return nil, err
		}
		return s.docdbElastic.applyPending(dbeStr(in, "resourceArn"), dbeStr(in, "applyAction"), dbeStr(in, "optInType"), dbeStr(in, "applyOn"))
	case path == "/pending-actions" && r.Method == http.MethodGet:
		limit, err := dbeLimit(q, 1, 100, 100)
		if err != nil {
			return nil, err
		}
		return s.docdbElastic.listPending(q.Get("nextToken"), limit)
	}
	if arn, _, ok := dbeTail(path, "/pending-action/"); ok && r.Method == http.MethodGet {
		return s.docdbElastic.getPending(arn)
	}
	if arn, action, ok := dbeTail(path, "/cluster/", "start", "stop"); ok {
		return s.dbeClusterRoute(r, arn, action)
	}
	if arn, action, ok := dbeTail(path, "/cluster-snapshot/", "copy", "restore"); ok {
		return s.dbeSnapshotRoute(r, arn, action)
	}
	if arn, _, ok := dbeTail(path, "/tags/"); ok {
		return s.dbeTagsRoute(r, arn)
	}
	return nil, dbeUnknown(r)
}

func (s *Server) dbeClusterRoute(r *http.Request, arn, action string) (map[string]any, *dbeError) {
	switch {
	case action != "" && r.Method == http.MethodPost:
		return s.docdbElastic.setRunning(arn, action == "start")
	case action != "":
	case r.Method == http.MethodGet:
		return s.docdbElastic.getCluster(arn)
	case r.Method == http.MethodDelete:
		return s.docdbElastic.deleteCluster(arn)
	case r.Method == http.MethodPut:
		return s.dbeUpdateCluster(r, arn)
	}
	return nil, dbeUnknown(r)
}

func (s *Server) dbeSnapshotRoute(r *http.Request, arn, action string) (map[string]any, *dbeError) {
	switch {
	case action == "copy" && r.Method == http.MethodPost:
		in, err := dbeDecode(r, "CopyClusterSnapshotInput")
		if err != nil {
			return nil, err
		}
		target := dbeStr(in, "targetSnapshotName")
		if err := dbeCheckName("targetSnapshotName", target); err != nil {
			return nil, err
		}
		tags := dbeStringMap(in["tags"])
		if err := dbeCheckTags(tags); err != nil {
			return nil, err
		}
		copyTags, _ := in["copyTags"].(bool)
		return s.docdbElastic.copySnapshot(arn, target, dbeStr(in, "kmsKeyId"), copyTags, tags)
	case action == "restore" && r.Method == http.MethodPost:
		return s.dbeRestore(r, arn)
	case action != "":
	case r.Method == http.MethodGet:
		return s.docdbElastic.getSnapshot(arn)
	case r.Method == http.MethodDelete:
		return s.docdbElastic.deleteSnapshot(arn)
	}
	return nil, dbeUnknown(r)
}

func (s *Server) dbeTagsRoute(r *http.Request, arn string) (map[string]any, *dbeError) {
	switch r.Method {
	case http.MethodGet:
		return s.docdbElastic.listTags(arn)
	case http.MethodPost:
		in, err := dbeDecode(r, "TagResourceRequest")
		if err != nil {
			return nil, err
		}
		tags := dbeStringMap(in["tags"])
		if err := dbeCheckTags(tags); err != nil {
			return nil, err
		}
		return map[string]any{}, s.docdbElastic.tag(arn, tags)
	case http.MethodDelete:
		keys := r.URL.Query()["tagKeys"]
		if len(keys) == 0 || len(keys) > dbeMaxTags {
			return nil, dbeValidation("tagKeys must name 1 to %d tag keys.", dbeMaxTags)
		}
		return map[string]any{}, s.docdbElastic.untag(arn, keys)
	}
	return nil, dbeUnknown(r)
}

// dbeSpec reads the cluster fields shared by create, restore and update.
func dbeSpec(in map[string]any) dbeClusterSpec {
	return dbeClusterSpec{
		name: strings.ToLower(dbeStr(in, "clusterName")), adminUser: dbeStr(in, "adminUserName"),
		authType: dbeStr(in, "authType"), password: dbeStr(in, "adminUserPassword"),
		shardCapacity: dbeInt32(in, "shardCapacity"), shardCount: dbeInt32(in, "shardCount"),
		shardInstance: dbeInt32(in, "shardInstanceCount"), backupRetention: dbeInt32(in, "backupRetentionPeriod"),
		backupWindow: dbeStr(in, "preferredBackupWindow"), maintWindow: dbeStr(in, "preferredMaintenanceWindow"),
		kmsKeyID: dbeStr(in, "kmsKeyId"), subnets: dbeStrings(in["subnetIds"]), groups: dbeStrings(in["vpcSecurityGroupIds"]),
		tags: dbeStringMap(in["tags"]),
	}
}

func (s *Server) dbeSecretExists(arn string) bool {
	_, err := s.secretsmgr.DescribeSecret(arn)
	return err == nil
}

func (s *Server) dbeCreateCluster(r *http.Request) (map[string]any, *dbeError) {
	in, err := dbeDecode(r, "CreateClusterInput")
	if err != nil {
		return nil, err
	}
	spec := dbeSpec(in)
	if err := dbeCheckName("clusterName", spec.name); err != nil {
		return nil, err
	}
	if !dbeUserPattern.MatchString(spec.adminUser) {
		return nil, dbeValidation("adminUserName must be 1 to 63 letters or numbers and start with a letter.")
	}
	if spec.shardCapacity == nil || spec.shardCount == nil {
		return nil, dbeValidation("shardCapacity and shardCount are required.")
	}
	if err := dbeCheckSizes(spec); err != nil {
		return nil, err
	}
	if err := dbeCheckAuth(spec.authType, spec.password, s.dbeSecretExists); err != nil {
		return nil, err
	}
	if err := dbeCheckTags(spec.tags); err != nil {
		return nil, err
	}
	if spec.subnets, spec.groups, err = s.dbeResolveNetwork(spec.subnets, spec.groups); err != nil {
		return nil, err
	}
	return s.docdbElastic.createCluster(spec, dbeStr(in, "clientToken"))
}

func (s *Server) dbeUpdateCluster(r *http.Request, arn string) (map[string]any, *dbeError) {
	in, err := dbeDecode(r, "UpdateClusterInput")
	if err != nil {
		return nil, err
	}
	spec := dbeSpec(in)
	if err := dbeCheckSizes(spec); err != nil {
		return nil, err
	}
	if _, ok := in["subnetIds"]; !ok {
		spec.subnets = nil
	}
	if _, ok := in["vpcSecurityGroupIds"]; !ok {
		spec.groups = nil
	}
	if spec.subnets != nil || spec.groups != nil {
		subnets, groups, err := s.docdbElastic.clusterNetwork(arn)
		if err != nil {
			return nil, err
		}
		if spec.subnets != nil {
			subnets = spec.subnets
		}
		if spec.groups != nil {
			groups = spec.groups
		}
		if spec.subnets, spec.groups, err = s.dbeResolveNetwork(subnets, groups); err != nil {
			return nil, err
		}
	}
	return s.docdbElastic.updateCluster(arn, spec, s.dbeSecretExists)
}

func (s *Server) dbeRestore(r *http.Request, arn string) (map[string]any, *dbeError) {
	in, err := dbeDecode(r, "RestoreClusterFromSnapshotInput")
	if err != nil {
		return nil, err
	}
	spec := dbeSpec(in)
	if err := dbeCheckName("clusterName", spec.name); err != nil {
		return nil, err
	}
	spec.shardCount, spec.backupRetention, spec.backupWindow, spec.maintWindow = nil, nil, "", ""
	if err := dbeCheckSizes(spec); err != nil {
		return nil, err
	}
	if err := dbeCheckTags(spec.tags); err != nil {
		return nil, err
	}
	snap, err := s.docdbElastic.snapshotForRestore(arn)
	if err != nil {
		return nil, err
	}
	if len(spec.subnets) == 0 {
		spec.subnets = snap.subnets
	}
	if len(spec.groups) == 0 {
		spec.groups = snap.groups
	}
	if spec.subnets, spec.groups, err = s.dbeResolveNetwork(spec.subnets, spec.groups); err != nil {
		return nil, err
	}
	return s.docdbElastic.restore(snap, spec)
}

// dbeResolveNetwork checks subnets and security groups against the EC2 store.
// With no subnets it uses the default VPC's; with no groups, the VPC's default
// group. All must share one VPC. It runs before the docdb-elastic lock is taken.
func (s *Server) dbeResolveNetwork(subnets, groups []string) ([]string, []string, *dbeError) {
	subnets, groups = dbeUnique(subnets), dbeUnique(groups)
	if len(subnets) == 0 {
		for _, v := range s.ec2.DescribeVpcs(nil) {
			if v.IsDefault {
				for _, sn := range s.ec2.DescribeSubnets(nil, []string{v.ID}) {
					subnets = append(subnets, sn.ID)
				}
				break
			}
		}
		if len(subnets) == 0 {
			return nil, nil, dbeValidation("subnetIds is required because there is no default VPC with subnets.")
		}
	}
	found := s.ec2.DescribeSubnets(subnets, nil)
	if len(found) != len(subnets) {
		return nil, nil, dbeValidation("One or more subnets in subnetIds do not exist.")
	}
	vpc := found[0].VpcID
	for _, sn := range found {
		if sn.VpcID != vpc {
			return nil, nil, dbeValidation("All subnets in subnetIds must belong to the same VPC.")
		}
	}
	if len(groups) == 0 {
		for _, g := range s.ec2.DescribeSecurityGroups(nil, []string{"default"}, vpc) {
			groups = append(groups, g.ID)
		}
		return subnets, groups, nil
	}
	sgs := s.ec2.DescribeSecurityGroups(groups, nil, "")
	if len(sgs) != len(groups) {
		return nil, nil, dbeValidation("One or more security groups in vpcSecurityGroupIds do not exist.")
	}
	for _, g := range sgs {
		if g.VpcID != vpc {
			return nil, nil, dbeValidation("Security group %s is not in the subnets' VPC %s.", g.ID, vpc)
		}
	}
	return subnets, groups, nil
}

// dbeDecode reads a JSON body and checks it against the model input shape.
func dbeDecode(r *http.Request, shape string) (map[string]any, *dbeError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, dbeValidation("unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, &dbeError{status: http.StatusBadRequest, code: "ValidationException", msg: "request body is not a JSON object"}
		}
	}
	if problems := awsmodels.DocDBElasticModel().Validate(shape, in); len(problems) > 0 {
		return nil, dbeValidation("%s", problems[0])
	}
	conformed, _ := awsmodels.DocDBElasticModel().Conform(shape, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	return conformed, nil
}

func dbeLimit(q url.Values, lo, hi, def int) (int, *dbeError) {
	raw := q.Get("maxResults")
	if raw == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || int(n) < lo || int(n) > hi {
		return 0, dbeValidation("maxResults must be between %d and %d.", lo, hi)
	}
	return int(n), nil
}

func dbeStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return strings.TrimSpace(v)
}

func dbeInt32(in map[string]any, key string) *int32 {
	f, ok := in[key].(float64)
	if !ok {
		return nil
	}
	if f < -2147483648 || f > 2147483647 || f != float64(int64(f)) {
		v := int32(-1) // out of range: rejected by the bounds checks
		return &v
	}
	v := int32(f)
	return &v
}

func dbeStrings(v any) []string {
	list, _ := v.([]any)
	if list == nil {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func dbeStringMap(v any) map[string]string {
	m, _ := v.(map[string]any)
	out := map[string]string{}
	for k, item := range m {
		if s, ok := item.(string); ok {
			out[k] = s
		}
	}
	return out
}

func dbeUnique(in []string) []string {
	if in == nil {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
