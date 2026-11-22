package server

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

// DAX control plane (JSON 1.1, target AmazonDAXV3, signing name dax).
// ponytail: the DAX data plane (the binary DAX protocol clients speak to the
// cluster endpoint) isn't emulated; discovery endpoints point at Stackyard's
// host only so clients have a well-formed address.

const (
	daxTargetPrefix = "AmazonDAXV3."
	daxMaxBody      = 1 << 20
)

type daxHandler func(st *daxStore, body []byte, host string) (any, *daxErr)

// daxDecodeInto decodes body into a typed input; unknown members are refused so
// callers can only set modelled fields.
func daxDecodeInto[T any](body []byte, run func(T) (any, *daxErr)) (any, *daxErr) {
	var in T
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return nil, daxError("SerializationException", "invalid request body: %v", err)
	}
	return run(in)
}

var daxHandlers = map[string]daxHandler{
	"CreateCluster": func(st *daxStore, b []byte, host string) (any, *daxErr) {
		return daxDecodeInto(b, func(in daxCreateClusterIn) (any, *daxErr) { return st.createCluster(in, host) })
	},
	"DescribeClusters": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.describeClusters)
	},
	"UpdateCluster": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, func(in daxUpdateClusterIn) (any, *daxErr) {
			if in.ClusterName == "" {
				return nil, daxInvalid("ClusterName is required")
			}
			return st.updateCluster(in)
		})
	},
	"DeleteCluster": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, func(in daxDeleteClusterIn) (any, *daxErr) {
			if in.ClusterName == "" {
				return nil, daxInvalid("ClusterName is required")
			}
			return st.deleteCluster(in)
		})
	},
	"IncreaseReplicationFactor": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, func(in daxReplicationIn) (any, *daxErr) {
			if len(in.NodeIdsToRemove) > 0 {
				return nil, daxInvalid("NodeIdsToRemove is not a member of IncreaseReplicationFactor")
			}
			return st.increaseReplication(in)
		})
	},
	"DecreaseReplicationFactor": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.decreaseReplication)
	},
	"RebootNode": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.rebootNode)
	},
	"CreateParameterGroup": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.createParamGroup)
	},
	"DeleteParameterGroup": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, func(in struct {
			ParameterGroupName string `json:"ParameterGroupName"`
		}) (any, *daxErr) {
			return st.deleteParamGroup(daxNameIn{ParameterGroupName: in.ParameterGroupName})
		})
	},
	"DescribeParameterGroups": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.describeParamGroups)
	},
	"DescribeParameters": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.describeParameters)
	},
	"DescribeDefaultParameters": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.describeDefaultParameters)
	},
	"UpdateParameterGroup": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.updateParamGroup)
	},
	"CreateSubnetGroup": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.createSubnetGroup)
	},
	"UpdateSubnetGroup": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.updateSubnetGroup)
	},
	"DeleteSubnetGroup": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, func(in struct {
			SubnetGroupName string `json:"SubnetGroupName"`
		}) (any, *daxErr) {
			return st.deleteSubnetGroup(daxNameIn{SubnetGroupName: in.SubnetGroupName})
		})
	},
	"DescribeSubnetGroups": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.describeSubnetGroups)
	},
	"DescribeEvents": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, st.describeEvents)
	},
	"TagResource": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, func(in daxTagIn) (any, *daxErr) {
			if in.TagKeys != nil || in.NextToken != nil {
				return nil, daxInvalid("TagResource takes ResourceName and Tags")
			}
			if len(in.Tags) == 0 {
				return nil, daxInvalid("Tags must list at least one tag")
			}
			return st.tagResource(in)
		})
	},
	"UntagResource": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, func(in daxTagIn) (any, *daxErr) {
			if in.Tags != nil || in.NextToken != nil {
				return nil, daxInvalid("UntagResource takes ResourceName and TagKeys")
			}
			return st.untagResource(in)
		})
	},
	"ListTags": func(st *daxStore, b []byte, _ string) (any, *daxErr) {
		return daxDecodeInto(b, func(in daxTagIn) (any, *daxErr) {
			if in.Tags != nil || in.TagKeys != nil {
				return nil, daxInvalid("ListTags takes ResourceName and NextToken")
			}
			return st.listTags(in)
		})
	},
}

func isDAXJSONCandidate(r *http.Request) bool {
	return r.Method == http.MethodPost && strings.HasPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), daxTargetPrefix)
}

func (s *Server) handleDAXJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDAXJSONCandidate(r) {
		return false
	}
	// A declared length bounds the body (net/http stops reading there), so
	// require one; chunked bodies of unknown length are refused. r.Body is
	// not reassigned (CodeQL treats that as a fresh request source).
	if r.ContentLength < 0 || r.ContentLength > daxMaxBody {
		respondDAXError(w, http.StatusBadRequest, "InvalidParameterValueException", "request body must declare a Content-Length of at most 1 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "dax"); !ok {
		respondDAXError(w, status, code, msg)
		return true
	}
	action := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), daxTargetPrefix)
	handler, known := daxHandlers[action]
	if _, listed := daxOperationByName[action]; !known || !listed {
		respondDAXError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+action)
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondDAXError(w, http.StatusBadRequest, "SerializationException", "unable to read request body")
		return true
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}
	out, derr := handler(s.dax, body, daxEndpointHost(r))
	if derr != nil {
		respondDAXError(w, http.StatusBadRequest, derr.code, derr.msg)
		return true
	}
	payload, merr := json.Marshal(out)
	if merr != nil {
		respondDAXError(w, http.StatusInternalServerError, "InternalServerError", "unable to encode response")
		return true
	}
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
	return true
}

func respondDAXError(w http.ResponseWriter, status int, code, msg string) {
	payload, _ := json.Marshal(map[string]string{"__type": code, "message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

// daxEndpointHost is the host cluster endpoints report: the request's host
// name when it is a plain host name or IP, else localhost.
func daxEndpointHost(r *http.Request) string {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host == "" || len(host) > 253 {
		return "localhost"
	}
	for _, c := range host {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-') {
			return "localhost"
		}
	}
	return strings.ToLower(host)
}
