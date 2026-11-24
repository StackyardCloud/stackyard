package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Azure Event Grid publishing data plane (2018-01-01): publishing Event Grid,
// Cloud and custom-schema events to a custom topic, domain or partner
// namespace, served at /azure/eventgrid-dataplane/2018-01-01/{topicHostname}.
// The host's first label names a topic, domain or partner namespace in the
// emulator (Microsoft.EventGrid on the shared ARM handler). Events are
// validated and accepted; there is no delivery to subscribers.

const (
	azureEventGridDataPlane20180101APIVersion = "2018-01-01"
	azureEventGridDataPlane20180101Prefix     = "/azure/eventgrid-dataplane/2018-01-01"
	// egdpMaxBody is Event Grid's limit on one publish request.
	egdpMaxBody = 1 << 20
	// egdpMaxEvents caps the events in one request.
	egdpMaxEvents = 5000
)

// egdpPublishers are the resource types a topic host name may resolve to.
var egdpPublishers = []string{"topics", "domains", "partnerNamespaces"}

func (s *Server) handleAzureEventGridDataPlane20180101Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureEventGridDataPlane20180101Prefix+"/") {
		return false
	}
	host := strings.Trim(strings.TrimPrefix(path, azureEventGridDataPlane20180101Prefix), "/")
	host = strings.TrimSuffix(host, "/api/events")
	switch {
	case r.Method != http.MethodPost:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Events are published with POST.")
	case r.URL.Query().Get("api-version") != azureEventGridDataPlane20180101APIVersion:
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "api-version must be "+azureEventGridDataPlane20180101APIVersion+".")
	case host == "" || strings.Contains(host, "/"):
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "The path must name the topic host, as in {topic}.{region}-1.eventgrid.azure.net.")
	default:
		s.egdpPublish(w, r, host)
	}
	return true
}

func (s *Server) egdpPublish(w http.ResponseWriter, r *http.Request, host string) {
	overload := r.URL.Query().Get("overload")
	if overload != "EventGridEvent" && overload != "cloudEvent" && overload != "customEvent" {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "overload must be EventGridEvent, cloudEvent or customEvent.")
		return
	}
	publisher := s.egdpPublisher(strings.SplitN(host, ".", 2)[0])
	if publisher == nil {
		azureARMError(w, http.StatusNotFound, "ResourceNotFound", "No topic, domain or partner namespace has the host "+host+".")
		return
	}
	if status, code, msg := s.egdpAuthorize(r, publisher); status != 0 {
		azureARMError(w, status, code, msg)
		return
	}
	if r.ContentLength < 0 || r.ContentLength > egdpMaxBody {
		azureARMError(w, http.StatusRequestEntityTooLarge, "RequestEntityTooLarge", "The request must carry a Content-Length of at most 1 MiB.")
		return
	}
	var events []map[string]any
	if err := json.NewDecoder(io.LimitReader(r.Body, egdpMaxBody)).Decode(&events); err != nil {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "The body must be a JSON array of events.")
		return
	}
	if len(events) == 0 || len(events) > egdpMaxEvents {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "A request must carry 1 to 5000 events.")
		return
	}
	for i, event := range events {
		if msg := egdpValidateEvent(overload, event); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "Event "+strconv.Itoa(i)+": "+msg)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

// egdpPublisher returns the topic, domain or partner namespace named name.
func (s *Server) egdpPublisher(name string) *azureARMResource {
	found := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		if !strings.EqualFold(rec.Name, name) {
			return false
		}
		for _, typ := range egdpPublishers {
			if strings.EqualFold(rec.Type, egcpNamespace+"/"+typ) {
				return true
			}
		}
		return false
	})
	if len(found) == 0 {
		return nil
	}
	return found[0]
}

// egdpAuthorize checks the request's credentials: an aeg-sas-key must be one
// of the publisher's keys; an aeg-sas-token or Authorization header is
// accepted as is. It returns a zero status when authorized.
func (s *Server) egdpAuthorize(r *http.Request, publisher *azureARMResource) (int, string, string) {
	if key := r.Header.Get("aeg-sas-key"); key != "" {
		if key != s.azureResourceKey(publisher, "key1") && key != s.azureResourceKey(publisher, "key2") {
			return http.StatusUnauthorized, "Unauthorized", "The aeg-sas-key doesn't match a key of " + publisher.Name + "."
		}
		return 0, "", ""
	}
	if r.Header.Get("aeg-sas-token") == "" && r.Header.Get("Authorization") == "" {
		return http.StatusUnauthorized, "Unauthorized", "The request needs an aeg-sas-key, aeg-sas-token or Authorization header."
	}
	if publisher.Properties["disableLocalAuth"] == true && !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		return http.StatusUnauthorized, "Unauthorized", "Local authentication is disabled; use a Microsoft Entra token."
	}
	return 0, "", ""
}

// egdpValidateEvent checks one event against its schema's required fields.
func egdpValidateEvent(overload string, event map[string]any) string {
	var required []string
	switch overload {
	case "EventGridEvent":
		required = []string{"id", "subject", "eventType", "eventTime", "dataVersion"}
	case "cloudEvent":
		required = []string{"id", "source", "type", "specversion"}
		if event["specversion"] != nil && event["specversion"] != "1.0" {
			return "specversion must be 1.0."
		}
	default:
		return ""
	}
	for _, field := range required {
		if azureARMString(event[field]) == "" {
			return field + " is required."
		}
	}
	if overload == "EventGridEvent" {
		if _, err := time.Parse(time.RFC3339, azureARMString(event["eventTime"])); err != nil {
			return "eventTime must be an RFC 3339 timestamp."
		}
	}
	return ""
}
