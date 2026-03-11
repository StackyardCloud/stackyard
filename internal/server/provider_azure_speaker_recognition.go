package server

import (
	"net/http"
	"strings"
)

const azureSpeakerRecognitionPrefix = "/azure/speaker-recognition/"

func (s *Server) handleAzureSpeakerRecognitionRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSpeakerRecognitionPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureSpeakerRecognitionPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) < 3 {
		respondAzureImplemented(w, path)
		return true
	}

	plane := strings.ToLower(strings.TrimSpace(segments[0]))
	domain := strings.ToLower(strings.TrimSpace(segments[1]))
	resource := strings.ToLower(strings.TrimSpace(segments[2]))

	if plane == "verification" && (domain == "text-dependent" || domain == "text-independent") {
		if handleAzureSpeakerRecognitionVerificationRoute(w, r, path, segments, resource) {
			return true
		}
	}

	if plane == "identification" && domain == "text-independent" {
		if handleAzureSpeakerRecognitionIdentificationRoute(w, r, path, segments, resource) {
			return true
		}
	}

	// Keep staged ownership for unknown routes under this prefix.
	respondAzureImplemented(w, path)
	return true
}

func handleAzureSpeakerRecognitionVerificationRoute(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	segments []string,
	resource string,
) bool {
	switch resource {
	case "profiles":
		switch len(segments) {
		case 3:
			if r.Method == http.MethodGet || r.Method == http.MethodPost {
				respondAzureImplemented(w, path)
				return true
			}
		case 4:
			profileOrAction := strings.TrimSpace(segments[3])
			if profileOrAction == "" {
				break
			}
			lower := strings.ToLower(profileOrAction)
			if strings.HasSuffix(lower, ":reset") || strings.HasSuffix(lower, ":verify") {
				if r.Method == http.MethodPost {
					respondAzureImplemented(w, path)
					return true
				}
				break
			}
			if r.Method == http.MethodGet || r.Method == http.MethodDelete {
				respondAzureImplemented(w, path)
				return true
			}
		case 5:
			if strings.TrimSpace(segments[3]) != "" && strings.EqualFold(segments[4], "enrollments") && r.Method == http.MethodPost {
				respondAzureImplemented(w, path)
				return true
			}
		}
	case "phrases":
		if len(segments) == 4 && strings.TrimSpace(segments[3]) != "" && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	}
	return false
}

func handleAzureSpeakerRecognitionIdentificationRoute(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	segments []string,
	resource string,
) bool {
	if resource == "profiles:identifysinglespeaker" {
		if len(segments) == 3 && r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
		return false
	}

	switch resource {
	case "profiles":
		switch len(segments) {
		case 3:
			if r.Method == http.MethodGet || r.Method == http.MethodPost {
				respondAzureImplemented(w, path)
				return true
			}
		case 4:
			profileOrAction := strings.TrimSpace(segments[3])
			if profileOrAction == "" {
				break
			}
			lower := strings.ToLower(profileOrAction)
			if strings.HasSuffix(lower, ":reset") {
				if r.Method == http.MethodPost {
					respondAzureImplemented(w, path)
					return true
				}
				break
			}
			if r.Method == http.MethodGet || r.Method == http.MethodDelete {
				respondAzureImplemented(w, path)
				return true
			}
		case 5:
			if strings.TrimSpace(segments[3]) != "" && strings.EqualFold(segments[4], "enrollments") && r.Method == http.MethodPost {
				respondAzureImplemented(w, path)
				return true
			}
		}
	case "phrases":
		if len(segments) == 4 && strings.TrimSpace(segments[3]) != "" && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	}
	return false
}
