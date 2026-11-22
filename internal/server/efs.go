package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const efsAPIPrefix = "/2015-02-01/"

// handleEFSRESTRouter serves Amazon EFS (REST-JSON under /2015-02-01/, signed
// as elasticfilesystem).
func (s *Server) handleEFSRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isEFSRESTCandidate(r) {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "elasticfilesystem"); !ok {
		respondEFSError(w, &efsError{status: status, code: code, msg: msg})
		return true
	}
	var segs []string
	for _, raw := range strings.Split(strings.Trim(strings.TrimPrefix(rawRequestPath(r), efsAPIPrefix), "/"), "/") {
		seg, err := url.PathUnescape(raw)
		if err != nil {
			respondEFSError(w, efsBadRequest("invalid path segment %q", raw))
			return true
		}
		segs = append(segs, seg)
	}
	if !s.routeEFS(w, r, segs) {
		respondEFSError(w, efsErr(http.StatusNotImplemented, "NotImplementedException", "%s %s is not an EFS operation", r.Method, rawRequestPath(r)))
	}
	return true
}

func isEFSRESTCandidate(r *http.Request) bool {
	if !strings.HasPrefix(rawRequestPath(r), efsAPIPrefix) {
		return false
	}
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(r.Host)
	return service == "elasticfilesystem" || (service == "" && strings.Contains(host, "elasticfilesystem."))
}

// routeEFS dispatches on the path below /2015-02-01/; false means no route.
func (s *Server) routeEFS(w http.ResponseWriter, r *http.Request, segs []string) bool {
	q := r.URL.Query()
	n, m := len(segs), r.Method
	at := func(i int) string {
		if i < n {
			return segs[i]
		}
		return ""
	}
	switch at(0) {
	case "file-systems":
		if n == 1 || (n == 2 && at(1) == "replication-configurations") {
			return s.routeEFSFileSystems(w, r, q, n == 2)
		}
		return s.routeEFSFileSystem(w, r, segs[1], segs[2:])
	case "mount-targets":
		switch {
		case n == 1 && m == http.MethodPost:
			var in efsCreateMountTargetInput
			if efsDecode(w, r, &in) {
				out, err := s.efs.createMountTarget(in, s.lookupEFSSubnet(efsDeref(in.SubnetId), in.SecurityGroups))
				efsRespond(w, http.StatusOK, out, err)
			}
		case n == 1 && m == http.MethodGet:
			limit, ok := efsQueryInt(w, q, "MaxItems")
			if ok {
				out, err := s.efs.describeMountTargets(q.Get("FileSystemId"), q.Get("MountTargetId"), q.Get("AccessPointId"), q.Get("Marker"), limit)
				efsRespond(w, http.StatusOK, out, err)
			}
		case n == 2 && m == http.MethodDelete:
			efsRespond(w, http.StatusNoContent, nil, s.efs.deleteMountTarget(segs[1]))
		case n == 3 && at(2) == "security-groups" && m == http.MethodGet:
			groups, err := s.efs.mountTargetSecurityGroups(segs[1], nil, nil)
			efsRespond(w, http.StatusOK, map[string]any{"SecurityGroups": groups}, err)
		case n == 3 && at(2) == "security-groups" && m == http.MethodPut:
			var in struct {
				SecurityGroups []string `json:"SecurityGroups"`
			}
			if efsDecode(w, r, &in) {
				s.modifyEFSMountTargetGroups(w, segs[1], in.SecurityGroups)
			}
		default:
			return false
		}
	case "access-points":
		switch {
		case n == 1 && m == http.MethodPost:
			var in efsCreateAccessPointInput
			if efsDecode(w, r, &in) {
				out, err := s.efs.createAccessPoint(in)
				efsRespond(w, http.StatusOK, out, err)
			}
		case n == 1 && m == http.MethodGet:
			limit, ok := efsQueryInt(w, q, "MaxResults")
			if ok {
				out, err := s.efs.describeAccessPoints(q.Get("AccessPointId"), q.Get("FileSystemId"), q.Get("NextToken"), limit)
				efsRespond(w, http.StatusOK, out, err)
			}
		case n == 2 && m == http.MethodDelete:
			efsRespond(w, http.StatusNoContent, nil, s.efs.deleteAccessPoint(segs[1]))
		default:
			return false
		}
	case "account-preferences":
		if n != 1 || (m != http.MethodGet && m != http.MethodPut) {
			return false
		}
		var in struct {
			ResourceIdType *string `json:"ResourceIdType"`
		}
		if efsDecode(w, r, &in) {
			if m == http.MethodPut && efsDeref(in.ResourceIdType) == "" {
				respondEFSError(w, efsBadRequest("ResourceIdType is required."))
				return true
			}
			out, err := s.efs.accountPreferences(efsDeref(in.ResourceIdType))
			efsRespond(w, http.StatusOK, out, err)
		}
	case "create-tags", "delete-tags", "tags", "resource-tags":
		return s.routeEFSTags(w, r, q, segs)
	default:
		return false
	}
	return true
}

func (s *Server) routeEFSFileSystems(w http.ResponseWriter, r *http.Request, q url.Values, replications bool) bool {
	switch {
	case replications && r.Method == http.MethodGet:
		limit, ok := efsQueryInt(w, q, "MaxResults")
		if ok {
			out, err := s.efs.describeReplications(q.Get("FileSystemId"), q.Get("NextToken"), limit)
			efsRespond(w, http.StatusOK, out, err)
		}
	case !replications && r.Method == http.MethodPost:
		var in efsCreateFileSystemInput
		if efsDecode(w, r, &in) {
			out, err := s.efs.createFileSystem(in)
			efsRespond(w, http.StatusCreated, out, err)
		}
	case !replications && r.Method == http.MethodGet:
		limit, ok := efsQueryInt(w, q, "MaxItems")
		if ok {
			out, err := s.efs.describeFileSystems(q.Get("FileSystemId"), q.Get("CreationToken"), q.Get("Marker"), limit)
			efsRespond(w, http.StatusOK, out, err)
		}
	default:
		return false
	}
	return true
}

// routeEFSFileSystem handles /file-systems/{id}[/sub].
func (s *Server) routeEFSFileSystem(w http.ResponseWriter, r *http.Request, id string, rest []string) bool {
	if len(rest) > 1 {
		return false
	}
	sub := ""
	if len(rest) == 1 {
		sub = rest[0]
	}
	switch sub + " " + r.Method {
	case " PUT":
		var in efsUpdateFileSystemInput
		if efsDecode(w, r, &in) {
			out, err := s.efs.updateFileSystem(id, in)
			efsRespond(w, http.StatusAccepted, out, err)
		}
	case " DELETE":
		efsRespond(w, http.StatusNoContent, nil, s.efs.deleteFileSystem(id))
	case "protection PUT":
		var in struct {
			ReplicationOverwriteProtection *string `json:"ReplicationOverwriteProtection"`
		}
		if efsDecode(w, r, &in) {
			out, err := s.efs.updateProtection(id, efsDeref(in.ReplicationOverwriteProtection))
			efsRespond(w, http.StatusOK, out, err)
		}
	case "policy GET":
		out, err := s.efs.fileSystemPolicy(id, nil, false)
		efsRespond(w, http.StatusOK, out, err)
	case "policy DELETE":
		out, err := s.efs.fileSystemPolicy(id, nil, true)
		efsRespond(w, http.StatusOK, out, err)
	case "policy PUT":
		var in struct {
			Policy *string `json:"Policy"`
		}
		if efsDecode(w, r, &in) {
			policy := ""
			if in.Policy != nil {
				policy = *in.Policy
			}
			out, err := s.efs.fileSystemPolicy(id, &policy, false)
			efsRespond(w, http.StatusOK, out, err)
		}
	case "backup-policy GET":
		out, err := s.efs.backupPolicy(id, "")
		efsRespond(w, http.StatusOK, out, err)
	case "backup-policy PUT":
		var in struct {
			BackupPolicy *struct {
				Status *string `json:"Status"`
			} `json:"BackupPolicy"`
		}
		if efsDecode(w, r, &in) {
			if in.BackupPolicy == nil || efsDeref(in.BackupPolicy.Status) == "" {
				respondEFSError(w, efsErr(http.StatusBadRequest, "ValidationException", "BackupPolicy.Status is required."))
				return true
			}
			out, err := s.efs.backupPolicy(id, efsDeref(in.BackupPolicy.Status))
			efsRespond(w, http.StatusOK, out, err)
		}
	case "lifecycle-configuration GET":
		out, err := s.efs.lifecycle(id, nil, false)
		efsRespond(w, http.StatusOK, out, err)
	case "lifecycle-configuration PUT":
		var in struct {
			LifecyclePolicies *[]map[string]string `json:"LifecyclePolicies"`
		}
		if efsDecode(w, r, &in) {
			if in.LifecyclePolicies == nil {
				respondEFSError(w, efsBadRequest("LifecyclePolicies is required."))
				return true
			}
			out, err := s.efs.lifecycle(id, *in.LifecyclePolicies, true)
			efsRespond(w, http.StatusOK, out, err)
		}
	case "replication-configuration POST":
		var in struct {
			Destinations []efsDestinationInput `json:"Destinations"`
		}
		if efsDecode(w, r, &in) {
			out, err := s.efs.createReplication(id, in.Destinations)
			efsRespond(w, http.StatusOK, out, err)
		}
	case "replication-configuration DELETE":
		efsRespond(w, http.StatusNoContent, nil, s.efs.deleteReplication(id))
	default:
		return false
	}
	return true
}

func (s *Server) routeEFSTags(w http.ResponseWriter, r *http.Request, q url.Values, segs []string) bool {
	if len(segs) != 2 {
		return false
	}
	id := segs[1]
	switch segs[0] + " " + r.Method {
	case "create-tags POST", "resource-tags POST":
		var in struct {
			Tags []efsTag `json:"Tags"`
		}
		if efsDecode(w, r, &in) {
			if segs[0] == "create-tags" && !efsFileSystemIDPattern.MatchString(id) {
				respondEFSError(w, efsBadRequest("FileSystemId %q is not a valid file system ID.", id))
				return true
			}
			status := http.StatusOK
			if segs[0] == "create-tags" {
				status = http.StatusNoContent
			}
			efsRespond(w, status, map[string]any{}, s.efs.tagResource(id, in.Tags))
		}
	case "delete-tags POST":
		var in struct {
			TagKeys []string `json:"TagKeys"`
		}
		if efsDecode(w, r, &in) {
			if !efsFileSystemIDPattern.MatchString(id) {
				respondEFSError(w, efsBadRequest("FileSystemId %q is not a valid file system ID.", id))
				return true
			}
			efsRespond(w, http.StatusNoContent, nil, s.efs.untagResource(id, in.TagKeys))
		}
	case "resource-tags DELETE":
		efsRespond(w, http.StatusOK, map[string]any{}, s.efs.untagResource(id, q["tagKeys"]))
	case "tags GET":
		limit, ok := efsQueryInt(w, q, "MaxItems")
		if ok {
			out, err := s.efs.listTags(id, q.Get("Marker"), limit, true)
			efsRespond(w, http.StatusOK, out, err)
		}
	case "resource-tags GET":
		limit, ok := efsQueryInt(w, q, "MaxResults")
		if ok {
			out, err := s.efs.listTags(id, q.Get("NextToken"), limit, false)
			efsRespond(w, http.StatusOK, out, err)
		}
	default:
		return false
	}
	return true
}

// lookupEFSSubnet reads a subnet and the security groups a mount target names (or
// the VPC default group) from the EC2 store; nil means the subnet is unknown.
func (s *Server) lookupEFSSubnet(subnetID string, groups []string) *efsSubnet {
	if subnetID == "" {
		return nil
	}
	subnets := s.ec2.DescribeSubnets([]string{subnetID}, nil)
	if len(subnets) != 1 {
		return nil
	}
	sn := subnets[0]
	out := &efsSubnet{ID: sn.ID, VpcID: sn.VpcID, CIDR: sn.CidrBlock, AZ: sn.AvailabilityZone, SecurityGroups: map[string]string{}}
	for _, g := range s.ec2.DescribeSecurityGroups(nil, []string{"default"}, sn.VpcID) {
		out.DefaultGroup = g.ID
		out.SecurityGroups[g.ID] = g.VpcID
	}
	if len(groups) > 0 && len(groups) <= efsMaxSecurityGroups {
		for _, g := range s.ec2.DescribeSecurityGroups(groups, nil, "") {
			out.SecurityGroups[g.ID] = g.VpcID
		}
	}
	return out
}

func (s *Server) modifyEFSMountTargetGroups(w http.ResponseWriter, mtID string, groups []string) {
	subnetID, err := s.efs.mountTargetSubnetID(mtID)
	if err != nil {
		respondEFSError(w, err)
		return
	}
	subnet := s.lookupEFSSubnet(subnetID, groups)
	if subnet == nil {
		respondEFSError(w, efsErr(http.StatusBadRequest, "SubnetNotFound", "Subnet '%s' of mount target '%s' no longer exists.", subnetID, mtID))
		return
	}
	_, err = s.efs.mountTargetSecurityGroups(mtID, groups, subnet)
	efsRespond(w, http.StatusNoContent, nil, err)
}

// efsDecode reads an optional JSON body; it answers 400 and returns false on
// malformed JSON.
func efsDecode(w http.ResponseWriter, r *http.Request, out any) bool {
	body, err := readBodyBytes(r)
	if err != nil {
		respondEFSError(w, efsBadRequest("unable to read request body"))
		return false
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return true
	}
	if err := json.Unmarshal(body, out); err != nil {
		respondEFSError(w, efsBadRequest("request body is not valid JSON: %v", err))
		return false
	}
	return true
}

// efsQueryInt parses an optional positive page-size query parameter.
func efsQueryInt(w http.ResponseWriter, q url.Values, key string) (int, bool) {
	raw := strings.TrimSpace(q.Get(key))
	if raw == "" {
		return 0, true
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 1 {
		respondEFSError(w, efsBadRequest("%s must be a positive integer.", key))
		return 0, false
	}
	return int(n), true
}

func efsRespond(w http.ResponseWriter, status int, body any, err *efsError) {
	if err != nil {
		respondEFSError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if status != http.StatusNoContent && body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func respondEFSError(w http.ResponseWriter, err *efsError) {
	body := map[string]string{"ErrorCode": err.code, "Message": err.msg}
	for k, v := range err.extra {
		body[k] = v
	}
	w.Header().Set("X-Amzn-ErrorType", err.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.status)
	_ = json.NewEncoder(w).Encode(body)
}
