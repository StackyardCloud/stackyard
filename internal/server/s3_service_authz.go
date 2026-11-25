package server

import (
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/services/s3"
)

// s3Caller is who a service is acting for when it touches S3 on a caller's
// behalf. It is captured from the request so async jobs can be authorized
// later, after the request is gone. The zero value is an unsigned anonymous
// caller, which only public ACLs admit.
type s3Caller struct {
	requester string
	signed    bool
}

// s3CallerKey carries an s3Caller through a service's input map for handlers
// that don't see the *http.Request. It is set after the input is conformed to
// the model, and JSON can't produce an s3Caller value, so callers can't forge it.
const s3CallerKey = "\x00stackyard.s3caller"

func (s *Server) s3CallerOf(r *http.Request) s3Caller {
	if r == nil {
		return s3Caller{requester: "anonymous"}
	}
	return s3Caller{requester: authRequester(r, s.accessKey), signed: hasAuthRequest(r)}
}

// withS3Caller returns a copy of in that also carries the request's caller.
func (s *Server) withS3Caller(in map[string]any, r *http.Request) map[string]any {
	out := make(map[string]any, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	out[s3CallerKey] = s.s3CallerOf(r)
	return out
}

// s3CallerIn reads the caller withS3Caller stored; missing means anonymous.
func s3CallerIn(in map[string]any) s3Caller {
	c, _ := in[s3CallerKey].(s3Caller)
	return c
}

// authorizeServiceS3 decides whether servicePrincipal (e.g.
// "textract.amazonaws.com") may perform action on bucket/key for the caller
// of r.
func (s *Server) authorizeServiceS3(r *http.Request, servicePrincipal, bucket, key, action string) bool {
	return s.authorizeServiceS3As(s.s3CallerOf(r), servicePrincipal, bucket, key, action)
}

// authorizeServiceS3As closes the confused-deputy gap: the bucket policy is
// evaluated for both the caller and the service principal. An explicit Deny
// for either refuses; otherwise an explicit Allow for either admits; otherwise
// the caller gets what a direct S3 call would (signed → allowed, unsigned →
// public ACLs), so buckets without a policy behave as before.
func (s *Server) authorizeServiceS3As(c s3Caller, servicePrincipal, bucket, key, action string) bool {
	return authorizeServiceS3On(s.s3, c, servicePrincipal, bucket, key, action)
}

// authorizeServiceS3On is authorizeServiceS3As for stores that hold the S3
// service rather than the Server.
func authorizeServiceS3On(svc *s3.Service, c s3Caller, servicePrincipal, bucket, key, action string) bool {
	if policy, err := svc.GetBucketPolicy(bucket); err == nil && strings.TrimSpace(policy) != "" {
		requester := c.requester
		if requester == "" {
			requester = "anonymous"
		}
		userAllow, userDeny := evaluatePolicy(policy, requester, action, bucket, key)
		svcAllow, svcDeny := evaluatePolicyFor(policy, func(p any) bool {
			return servicePrincipalMatches(p, servicePrincipal)
		}, action, bucket, key)
		if userDeny || svcDeny {
			return false
		}
		if userAllow || svcAllow {
			return true
		}
	}
	if c.signed {
		return true
	}
	acl := ""
	if b, err := svc.GetBucket(bucket); err == nil {
		acl = b.ACL
	}
	if strings.EqualFold(action, "s3:GetObject") {
		if o, err := svc.GetObject(bucket, key); err == nil {
			acl = o.ACL
		}
	}
	switch strings.ToLower(action) {
	case "s3:getobject":
		return strings.EqualFold(acl, "public-read")
	case "s3:listbucket":
		return bucketIsPublic(acl)
	case "s3:putobject", "s3:deleteobject":
		return strings.EqualFold(acl, "public-read-write")
	default:
		return false
	}
}

// servicePrincipalMatches reports whether a policy Principal names service.
// Only a bare "*" or a {"Service": ...} entry can match; user principals
// ({"AWS": ...}) never do, and principalMatches never matches "Service", so
// a user can't pose as a service or the reverse.
func servicePrincipalMatches(principal any, service string) bool {
	switch val := principal.(type) {
	case string:
		return val == "*"
	case map[string]any:
		for _, name := range parsePolicyStringList(val["Service"]) {
			if strings.EqualFold(strings.TrimSpace(name), service) {
				return true
			}
		}
	case []any:
		for _, entry := range val {
			if servicePrincipalMatches(entry, service) {
				return true
			}
		}
	}
	return false
}
