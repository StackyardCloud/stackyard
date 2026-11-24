package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/services/s3vectors"
)

// handleS3VectorsModeRoutes serves index-mode and text-search metadata updates.
// It reports whether it handled the request.
func (s *Server) handleS3VectorsModeRoutes(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/PutVectorBucketDefaultIndexMode", "/UpdateIndexMode", "/UpdateMetadataConfiguration":
	default:
		return false
	}
	if r.Method != http.MethodPost {
		respondS3VectorsErrorJSON(w, http.StatusMethodNotAllowed, "MethodNotAllowedException", "unsupported method")
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondS3VectorsErrorJSON(w, http.StatusBadRequest, "BadRequestException", "invalid JSON")
		return true
	}
	switch r.URL.Path {
	case "/PutVectorBucketDefaultIndexMode":
		var req struct {
			VectorBucketName string `json:"vectorBucketName"`
			VectorBucketArn  string `json:"vectorBucketArn"`
			DefaultIndexMode string `json:"defaultIndexMode"`
		}
		if decodeS3VectorsJSON(body, &req) != nil {
			respondS3VectorsErrorJSON(w, http.StatusBadRequest, "BadRequestException", "invalid JSON")
			return true
		}
		bucket, msg := s3vResolveBucket(req.VectorBucketName, req.VectorBucketArn)
		if msg == "" && !s3vValidIndexMode(req.DefaultIndexMode) {
			msg = "defaultIndexMode must be CLASSIC or ENHANCED"
		}
		if msg != "" {
			respondS3VectorsErrorJSON(w, http.StatusBadRequest, "ValidationException", msg)
			return true
		}
		s3vRespond(w, s.s3vectors.SetVectorBucketDefaultIndexMode(bucket, req.DefaultIndexMode), s3VectorsEmptyResponse{})
	case "/UpdateIndexMode":
		var req struct {
			VectorBucketName string `json:"vectorBucketName"`
			IndexName        string `json:"indexName"`
			IndexArn         string `json:"indexArn"`
			IndexMode        string `json:"indexMode"`
		}
		if decodeS3VectorsJSON(body, &req) != nil {
			respondS3VectorsErrorJSON(w, http.StatusBadRequest, "BadRequestException", "invalid JSON")
			return true
		}
		bucket, index, msg := s3vResolveIndex(req.VectorBucketName, req.IndexName, req.IndexArn)
		if msg == "" && !s3vValidIndexMode(req.IndexMode) {
			msg = "indexMode must be CLASSIC or ENHANCED"
		}
		if msg != "" {
			respondS3VectorsErrorJSON(w, http.StatusBadRequest, "ValidationException", msg)
			return true
		}
		s3vRespond(w, s.s3vectors.SetIndexMode(bucket, index, req.IndexMode), s3VectorsEmptyResponse{})
	default: // /UpdateMetadataConfiguration
		type textSearch struct {
			MetadataKey string `json:"metadataKey"`
			Language    string `json:"language,omitempty"`
			Status      string `json:"status,omitempty"`
		}
		var req struct {
			VectorBucketName string `json:"vectorBucketName"`
			IndexName        string `json:"indexName"`
			IndexArn         string `json:"indexArn"`
			Add              *struct {
				TextSearchConfigurations []textSearch `json:"textSearchConfigurations"`
			} `json:"add,omitempty"`
			Delete *struct {
				TextSearchConfigurations []textSearch `json:"textSearchConfigurations"`
			} `json:"delete,omitempty"`
		}
		if decodeS3VectorsJSON(body, &req) != nil {
			respondS3VectorsErrorJSON(w, http.StatusBadRequest, "BadRequestException", "invalid JSON")
			return true
		}
		bucket, index, msg := s3vResolveIndex(req.VectorBucketName, req.IndexName, req.IndexArn)
		add := map[string]s3vectors.TextSearchConfig{}
		var remove []string
		if req.Add != nil {
			for _, c := range req.Add.TextSearchConfigurations {
				if strings.TrimSpace(c.MetadataKey) == "" {
					msg = "metadataKey is required"
				}
				add[c.MetadataKey] = s3vectors.TextSearchConfig{Language: c.Language, Status: "ACTIVE"}
			}
		}
		if req.Delete != nil {
			for _, c := range req.Delete.TextSearchConfigurations {
				remove = append(remove, c.MetadataKey)
			}
		}
		if len(add) > s3vectors.MaxTextSearchConfigs || len(remove) > s3vectors.MaxTextSearchConfigs {
			msg = "too many textSearchConfigurations"
		}
		if msg != "" {
			respondS3VectorsErrorJSON(w, http.StatusBadRequest, "ValidationException", msg)
			return true
		}
		s3vRespond(w, s.s3vectors.UpdateTextSearch(bucket, index, add, remove), struct {
			IndexArn string `json:"indexArn"`
		}{s3VectorsIndexArn(bucket, index)})
	}
	return true
}

func s3vValidIndexMode(mode string) bool { return mode == "CLASSIC" || mode == "ENHANCED" }

func s3vRespond(w http.ResponseWriter, err error, ok any) {
	switch {
	case err == nil:
		respondS3VectorsJSON(w, http.StatusOK, ok)
	case errors.Is(err, s3vectors.ErrVectorBucketNotFound), errors.Is(err, s3vectors.ErrIndexNotFound):
		respondS3VectorsErrorJSON(w, http.StatusNotFound, "NotFoundException", err.Error())
	default:
		respondS3VectorsErrorJSON(w, http.StatusBadRequest, "ValidationException", err.Error())
	}
}

// s3vResolveBucket returns the bucket name from name and/or ARN, or a validation message.
func s3vResolveBucket(name, arn string) (string, string) {
	name, arn = strings.TrimSpace(name), strings.TrimSpace(arn)
	if arn == "" {
		if name == "" {
			return "", "vectorBucketName or vectorBucketArn is required"
		}
		return name, ""
	}
	parsed, ok := s3VectorsBucketNameFromArn(arn)
	if !ok || (name != "" && name != parsed) {
		return "", "vectorBucketArn is invalid or does not match vectorBucketName"
	}
	return parsed, ""
}

// s3vResolveIndex returns (bucket, index) from names and/or the index ARN, or a validation message.
func s3vResolveIndex(bucket, index, arn string) (string, string, string) {
	bucket, index, arn = strings.TrimSpace(bucket), strings.TrimSpace(index), strings.TrimSpace(arn)
	if arn == "" {
		if bucket == "" || index == "" {
			return "", "", "indexArn, or vectorBucketName and indexName, are required"
		}
		return bucket, index, ""
	}
	pb, pi, ok := s3VectorsIndexFromArn(arn)
	if !ok || (bucket != "" && bucket != pb) || (index != "" && index != pi) {
		return "", "", "indexArn is invalid or does not match vectorBucketName/indexName"
	}
	return pb, pi, ""
}
