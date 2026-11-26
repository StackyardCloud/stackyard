package server

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Mechanical Turk (requester API, sandbox semantics): awsJson1_1,
// X-Amz-Target MTurkRequesterServiceV20170117.*, signed mturk-requester.
//
// Workers are simulated through Stackyard links under /_stackyard/mturk/
// (accept, submit and return assignments, request qualifications), which
// drive the real HIT lifecycle: Assignable -> Unassignable -> Reviewable
// (-> Reviewing), assignment deadlines and auto-approval. The sandbox balance
// is a constant $10,000.00 that rewards and bonuses never draw down.

const (
	mtkTargetPrefix  = "MTurkRequesterServiceV20170117."
	mtkSigningName   = "mturk-requester"
	mtkMaxBody       = 1 << 20
	mtkMaxWorkerBody = 4 << 20 // worker submissions carry uploaded files
	mtkContentType   = "application/x-amz-json-1.1"
	mtkLinkPrefix    = "/_stackyard/mturk/"
	mtkFilePath      = mtkLinkPrefix + "files/"
)

type mtkErr struct{ code, turk, msg string }

func mtkError(turk, format string, args ...any) *mtkErr {
	return &mtkErr{code: "RequestError", turk: turk, msg: fmt.Sprintf(format, args...)}
}

func mtkInvalid(format string, args ...any) *mtkErr {
	return mtkError("AWS.ParameterOutOfRange", format, args...)
}

func (s *Server) handleMTurkJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, mtkLinkPrefix) {
		s.handleMTurkWorkerLinks(w, r)
		return true
	}
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), mtkTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what is read; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > mtkMaxBody {
		writeMTurkError(w, http.StatusBadRequest, mtkInvalid("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, mtkSigningName); !ok {
		writeMTurkError(w, status, &mtkErr{code: code, msg: msg})
		return true
	}
	model := awsmodels.MTurkModel()
	shapes, known := model.Operations[op]
	if !known {
		writeMTurkError(w, http.StatusBadRequest, &mtkErr{code: "UnknownOperationException", msg: "unknown operation " + op})
		return true
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, mtkMaxBody+1))
	if err != nil || len(raw) > mtkMaxBody {
		writeMTurkError(w, http.StatusBadRequest, &mtkErr{code: "SerializationException", msg: "request body can't be read"})
		return true
	}
	in, problem := awsModelDecode(raw, model, shapes.Input)
	if problem != "" {
		writeMTurkError(w, http.StatusBadRequest, mtkError("AWS.MechanicalTurk.ParameterValidationError", "%s", problem))
		return true
	}
	body, merr := s.mturk.run(op, in, mmgrBaseURL(r))
	if merr != nil {
		writeMTurkError(w, http.StatusBadRequest, merr)
		return true
	}
	w.Header().Set("Content-Type", mtkContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func writeMTurkError(w http.ResponseWriter, status int, e *mtkErr) {
	payload := map[string]any{"__type": e.code, "Message": e.msg}
	if e.turk != "" {
		payload["TurkErrorCode"] = e.turk
	}
	body, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", mtkContentType)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// mtkWorkerRequest is the body of a simulated worker action.
type mtkWorkerRequest struct {
	HITId                string            `json:"HITId"`
	WorkerId             string            `json:"WorkerId"`
	AssignmentId         string            `json:"AssignmentId"`
	QualificationTypeId  string            `json:"QualificationTypeId"`
	Answer               string            `json:"Answer"`
	Country, Subdivision string            // the worker's locale (default US)
	Files                map[string]string `json:"Files"` // question identifier -> base64 bytes
}

// handleMTurkWorkerLinks serves the simulated worker actions and downloads
// of files workers uploaded with their answers.
func (s *Server) handleMTurkWorkerLinks(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if r.Method == http.MethodGet && strings.HasPrefix(path, mtkFilePath) {
		assign, qid, _ := strings.Cut(strings.TrimPrefix(path, mtkFilePath), "/")
		raw, ok := s.mturk.file(assign, qid, r.URL.Query().Get("key"))
		if !ok {
			http.Error(w, "the file upload URL is not valid or has expired", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", "attachment")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
		return
	}
	action := strings.TrimPrefix(path, mtkLinkPrefix)
	if r.Method != http.MethodPost || !mtkWorkerActions[action] {
		writeMTurkError(w, http.StatusNotFound, mtkError("AWS.MechanicalTurk.NotFound", "no worker action at %s", path))
		return
	}
	if r.ContentLength < 0 || r.ContentLength > mtkMaxWorkerBody {
		writeMTurkError(w, http.StatusBadRequest, mtkInvalid("worker requests must declare a Content-Length of at most 4 MiB"))
		return
	}
	var req mtkWorkerRequest
	raw, err := io.ReadAll(io.LimitReader(r.Body, mtkMaxWorkerBody+1))
	if err == nil && len(raw) <= mtkMaxWorkerBody {
		err = json.Unmarshal(raw, &req)
	}
	if err != nil || len(raw) > mtkMaxWorkerBody {
		writeMTurkError(w, http.StatusBadRequest, &mtkErr{code: "SerializationException", msg: "worker request body must be a JSON object"})
		return
	}
	files := make(map[string][]byte, len(req.Files))
	for qid, b64 := range req.Files {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			writeMTurkError(w, http.StatusBadRequest, mtkInvalid("Files[%s] is not base64", qid))
			return
		}
		files[qid] = data
	}
	out, merr := s.mturk.worker(action, req, files)
	if merr != nil {
		writeMTurkError(w, http.StatusBadRequest, merr)
		return
	}
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

var mtkWorkerActions = map[string]bool{"accept": true, "submit": true, "return": true, "request-qualification": true}

// file returns an uploaded answer file if key is its current download key.
func (st *mtkStore) file(assignID, qid, key string) ([]byte, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a := st.assigns[assignID]
	if a == nil || key == "" {
		return nil, false
	}
	link := a.links[qid]
	if link.key == "" || subtle.ConstantTimeCompare([]byte(key), []byte(link.key)) != 1 || st.now().After(link.expires) {
		return nil, false
	}
	return a.files[qid], true
}
