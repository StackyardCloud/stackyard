package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Elemental MediaStore control plane: awsJson1_1, X-Amz-Target
// MediaStore_20170901.*, signed mediastore. The data plane (mediastoredata.go)
// shares the signing name but never sends X-Amz-Target.

const (
	mstTargetPrefix = "MediaStore_20170901."
	mstMaxBody      = 1 << 20
)

type mstHandler func(s *Server, in map[string]any, base string) (map[string]any, *mstErr)

var mstHandlers map[string]mstHandler

func init() {
	empty := func(err *mstErr) (map[string]any, *mstErr) { return map[string]any{}, err }
	policy := func(kind mstPolicyKind) [3]mstHandler {
		return [3]mstHandler{
			func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
				return empty(s.mstPutPolicy(in, kind))
			},
			func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
				return s.mediastore.getPolicy(mstStr(in, "ContainerName"), kind)
			},
			func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
				return empty(s.mediastore.deletePolicy(mstStr(in, "ContainerName"), kind))
			},
		}
	}
	mstHandlers = map[string]mstHandler{
		"CreateContainer": func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
			return s.mediastore.createContainer(mstStr(in, "ContainerName"), mstTags(in["Tags"]))
		},
		"DescribeContainer": func(s *Server, in map[string]any, base string) (map[string]any, *mstErr) {
			return s.mediastore.describeContainer(mstStr(in, "ContainerName"), base)
		},
		"ListContainers": func(s *Server, in map[string]any, base string) (map[string]any, *mstErr) {
			max := 100
			if v, ok := in["MaxResults"].(float64); ok {
				max = int(v) // the model bounds it to 1..100
			}
			return s.mediastore.listContainers(mstStr(in, "NextToken"), max, base)
		},
		"DeleteContainer": func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
			return empty(s.mediastore.deleteContainer(mstStr(in, "ContainerName")))
		},
		"StartAccessLogging": func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
			return empty(s.mediastore.setLogging(mstStr(in, "ContainerName"), true))
		},
		"StopAccessLogging": func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
			return empty(s.mediastore.setLogging(mstStr(in, "ContainerName"), false))
		},
		"TagResource": func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
			return empty(s.mediastore.tagResource(mstStr(in, "Resource"), mstTags(in["Tags"])))
		},
		"UntagResource": func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
			var keys []string
			for _, k := range mstList(in["TagKeys"]) {
				if s, ok := k.(string); ok {
					keys = append(keys, s)
				}
			}
			return empty(s.mediastore.untagResource(mstStr(in, "Resource"), keys))
		},
		"ListTagsForResource": func(s *Server, in map[string]any, _ string) (map[string]any, *mstErr) {
			return s.mediastore.listTags(mstStr(in, "Resource"))
		},
	}
	for kind, name := range map[mstPolicyKind]string{
		mstContainerPolicy: "ContainerPolicy", mstCorsPolicy: "CorsPolicy",
		mstLifecyclePolicy: "LifecyclePolicy", mstMetricPolicy: "MetricPolicy",
	} {
		h := policy(kind)
		mstHandlers["Put"+name], mstHandlers["Get"+name], mstHandlers["Delete"+name] = h[0], h[1], h[2]
	}
}

func (s *Server) handleMediaStoreJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), mstTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > mstMaxBody {
		respondMSTError(w, mstInvalid("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "mediastore"); !ok {
		respondMSTError(w, &mstErr{status, code, msg})
		return true
	}
	h := mstHandlers[op]
	if _, known := mstOperationByName[op]; !known || h == nil {
		respondMSTError(w, &mstErr{http.StatusBadRequest, "UnknownOperationException", "unknown operation " + op})
		return true
	}
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondMSTError(w, &mstErr{http.StatusBadRequest, "SerializationException", "invalid JSON body"})
		return true
	}
	model := awsmodels.MediaStoreModel()
	if problems := model.Validate(model.Operations[op].Input, in); len(problems) > 0 {
		respondMSTError(w, mstInvalid("%s", strings.Join(problems, "; ")))
		return true
	}
	// Conform drops members the input shape doesn't define.
	conformed, _ := model.Conform(model.Operations[op].Input, in).(map[string]any)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Outputs are shaped (deep-copied) by the store under its lock.
	out, merr := h(s, conformed, scheme+"://"+r.Host)
	if merr != nil {
		respondMSTError(w, merr)
		return true
	}
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

// mstPutPolicy checks a policy beyond the model and stores it.
func (s *Server) mstPutPolicy(in map[string]any, kind mstPolicyKind) *mstErr {
	name := mstStr(in, "ContainerName")
	switch kind {
	case mstContainerPolicy:
		doc := mstStr(in, "Policy")
		if err := mstCheckPolicyDocument(doc); err != nil {
			return err
		}
		return s.mediastore.putPolicy(name, kind, doc)
	case mstLifecyclePolicy:
		doc := mstStr(in, "LifecyclePolicy")
		if err := mstCheckLifecyclePolicy(doc); err != nil {
			return err
		}
		return s.mediastore.putPolicy(name, kind, doc)
	case mstCorsPolicy:
		return s.mediastore.putPolicy(name, kind, mstList(in["CorsPolicy"]))
	default:
		mp, _ := in["MetricPolicy"].(map[string]any)
		if err := mstCheckMetricPolicy(mp); err != nil {
			return err
		}
		return s.mediastore.putPolicy(name, kind, mp)
	}
}

func respondMSTError(w http.ResponseWriter, e *mstErr) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "Message": e.msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

func mstStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func mstList(v any) []any {
	l, _ := v.([]any)
	return l
}

func mstTags(v any) []mstTag {
	var out []mstTag
	for _, item := range mstList(v) {
		m, _ := item.(map[string]any)
		out = append(out, mstTag{key: mstStr(m, "Key"), value: mstStr(m, "Value")})
	}
	return out
}
