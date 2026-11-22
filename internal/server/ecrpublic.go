package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	ecrPublicTargetPrefix = "SpencerFrontendService."
	ecrPublicSigningName  = "ecr-public"
	ecrPublicRegion       = "us-east-1"
	// ecrPublicMaxBody matches readBodyBytes' 25 MiB read limit, which is what
	// bounds a base64 layer part (about 18 MiB of raw bytes per part).
	ecrPublicMaxBody = 25 << 20
)

// ecrPublicInput holds every request member of the ECR Public API. Requests
// decode into it with unknown members rejected, so callers only reach the
// fields below.
type ecrPublicInput struct {
	RegistryID             *string                `json:"registryId"`
	RepositoryName         string                 `json:"repositoryName"`
	RepositoryNames        []string               `json:"repositoryNames"`
	CatalogData            *ecrPublicCatalogInput `json:"catalogData"`
	Tags                   []ecrPublicTag         `json:"tags"`
	TagKeys                []string               `json:"tagKeys"`
	ResourceArn            string                 `json:"resourceArn"`
	Force                  bool                   `json:"force"`
	PolicyText             string                 `json:"policyText"`
	NextToken              string                 `json:"nextToken"`
	MaxResults             *int32                 `json:"maxResults"`
	ImageIDs               []ecrPublicImageID     `json:"imageIds"`
	LayerDigests           []string               `json:"layerDigests"`
	UploadID               string                 `json:"uploadId"`
	PartFirstByte          *int64                 `json:"partFirstByte"`
	PartLastByte           *int64                 `json:"partLastByte"`
	LayerPartBlob          []byte                 `json:"layerPartBlob"`
	ImageManifest          string                 `json:"imageManifest"`
	ImageManifestMediaType string                 `json:"imageManifestMediaType"`
	ImageTag               string                 `json:"imageTag"`
	ImageDigest            string                 `json:"imageDigest"`
	DisplayName            *string                `json:"displayName"`
}

type ecrPublicCatalogInput struct {
	Description      *string  `json:"description"`
	Architectures    []string `json:"architectures"`
	OperatingSystems []string `json:"operatingSystems"`
	LogoImageBlob    []byte   `json:"logoImageBlob"`
	AboutText        *string  `json:"aboutText"`
	UsageText        *string  `json:"usageText"`
}

type ecrPublicTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type ecrPublicImageID struct {
	ImageDigest string `json:"imageDigest,omitempty"`
	ImageTag    string `json:"imageTag,omitempty"`
}

type ecrPublicError struct {
	status int
	code   string
	msg    string
}

func ecrPublicInvalid(msg string) *ecrPublicError {
	return &ecrPublicError{http.StatusBadRequest, "InvalidParameterException", msg}
}

func ecrPublicFail(code, msg string) *ecrPublicError {
	return &ecrPublicError{http.StatusBadRequest, code, msg}
}

func (s *Server) handleECRPublicJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost || !strings.HasPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), ecrPublicTargetPrefix) {
		return false
	}
	// A declared length bounds the body (net/http stops reading there), so
	// require one; chunked bodies of unknown length are refused. r.Body is
	// not reassigned (CodeQL treats that as a fresh request source).
	if r.ContentLength < 0 || r.ContentLength > ecrPublicMaxBody {
		respondECRPublicError(w, ecrPublicInvalid("request body must declare a Content-Length of at most 25 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, ecrPublicSigningName); !ok {
		respondECRPublicError(w, &ecrPublicError{status, code, msg})
		return true
	}
	if region := ecrPublicRequestRegion(r); region != "" && region != ecrPublicRegion {
		respondECRPublicError(w, ecrPublicFail("UnsupportedCommandException", "Amazon ECR Public is only available in "+ecrPublicRegion))
		return true
	}
	action := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), ecrPublicTargetPrefix)
	if _, known := ecrPublicOperationByName[action]; !known {
		respondECRPublicError(w, ecrPublicInvalid("unknown action "+action))
		return true
	}
	in, err := decodeECRPublicInput(r)
	if err != nil {
		respondECRPublicError(w, ecrPublicInvalid("invalid request body: "+err.Error()))
		return true
	}
	out, apiErr := s.ecrPublic.handle(action, in)
	if apiErr != nil {
		respondECRPublicError(w, apiErr)
		return true
	}
	respondECRPublicJSON(w, http.StatusOK, out)
	return true
}

func ecrPublicRequestRegion(r *http.Request) string {
	if req, err := parseSigV4Authorization(r.Header.Get("Authorization")); err == nil {
		return strings.TrimSpace(req.Region)
	}
	if req, err := parseSigV4Query(r.URL.Query()); err == nil {
		return strings.TrimSpace(req.Region)
	}
	return ""
}

func decodeECRPublicInput(r *http.Request) (ecrPublicInput, error) {
	var in ecrPublicInput
	body, err := readBodyBytes(r)
	if err != nil {
		return in, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return in, nil
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	return in, dec.Decode(&in)
}

// respondECRPublicJSON writes an already-marshalled body; the content type is
// fixed here, next to the write.
func respondECRPublicJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func respondECRPublicError(w http.ResponseWriter, e *ecrPublicError) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "message": e.msg})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}
