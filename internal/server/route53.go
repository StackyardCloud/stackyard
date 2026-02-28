package server

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type route53ErrorEnvelope struct {
	XMLName   xml.Name         `xml:"ErrorResponse"`
	Xmlns     string           `xml:"xmlns,attr,omitempty"`
	Error     route53ErrorBody `xml:"Error"`
	RequestID string           `xml:"RequestId"`
}

type route53ErrorBody struct {
	Type    string `xml:"Type,omitempty"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

var route53SDKOperationRegex = regexp.MustCompile(`(?i)api/route53#([A-Za-z0-9]+)`)

var route53OperationByCLICommand = func() map[string]string {
	out := make(map[string]string, len(route53Operations))
	for _, op := range route53Operations {
		out[route53OperationToCLICommand(op.Name)] = op.Name
	}
	return out
}()

func (s *Server) handleRoute53RESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRoute53RESTRouterCandidate(r) {
		return false
	}

	action := parseRoute53Operation(r)
	if action == "" {
		respondRoute53Error(w, http.StatusBadRequest, "InvalidInput", "unable to determine Route 53 action")
		return true
	}
	if _, known := route53OperationByName[action]; !known {
		respondRoute53Error(w, http.StatusBadRequest, "NoSuchAction", "unknown Route 53 action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "route53")
	if !ok {
		respondRoute53Error(w, status, code, msg)
		return true
	}

	pathParams := route53PathParams(rawRequestPath(r))
	result := s.route53.Handle(action, pathParams)
	if result.Status == 0 {
		result.Status = http.StatusOK
	}
	for key, value := range result.Headers {
		w.Header().Set(key, value)
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/xml")
	}
	w.WriteHeader(result.Status)
	_, _ = w.Write(result.Body)
	return true
}

func isRoute53RESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "route53" {
		return false
	}
	if service == "route53" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "route53") {
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(ua, "command#route53") || strings.Contains(ua, "api/route53#") || strings.Contains(ua, " route53/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/2013-04-01")
}

func parseRoute53Operation(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Stackyard-Operation")); value != "" {
		return value
	}

	if op := parseRoute53OperationFromUserAgent(r.Header.Get("User-Agent")); op != "" {
		return op
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	cleanedPath := strings.TrimSuffix(path, "/")

	switch {
	case method == http.MethodGet && cleanedPath == "/2013-04-01/hostedzone":
		return "ListHostedZones"
	case method == http.MethodPost && cleanedPath == "/2013-04-01/hostedzone":
		return "CreateHostedZone"
	case method == http.MethodGet && strings.HasPrefix(cleanedPath, "/2013-04-01/hostedzone/") && strings.HasSuffix(cleanedPath, "/rrset"):
		return "ListResourceRecordSets"
	case method == http.MethodPost && strings.HasPrefix(cleanedPath, "/2013-04-01/hostedzone/") && strings.HasSuffix(cleanedPath, "/rrset"):
		return "ChangeResourceRecordSets"
	case method == http.MethodGet && strings.HasPrefix(cleanedPath, "/2013-04-01/hostedzone/"):
		return "GetHostedZone"
	case method == http.MethodDelete && strings.HasPrefix(cleanedPath, "/2013-04-01/hostedzone/"):
		return "DeleteHostedZone"
	case method == http.MethodGet && cleanedPath == "/2013-04-01/hostedzonecount":
		return "GetHostedZoneCount"
	case method == http.MethodGet && cleanedPath == "/2013-04-01/healthcheck":
		return "ListHealthChecks"
	case method == http.MethodPost && cleanedPath == "/2013-04-01/healthcheck":
		return "CreateHealthCheck"
	case method == http.MethodGet && strings.HasPrefix(cleanedPath, "/2013-04-01/healthcheck/"):
		return "GetHealthCheck"
	case method == http.MethodDelete && strings.HasPrefix(cleanedPath, "/2013-04-01/healthcheck/"):
		return "DeleteHealthCheck"
	case method == http.MethodPost && strings.HasPrefix(cleanedPath, "/2013-04-01/healthcheck/"):
		return "UpdateHealthCheck"
	case method == http.MethodGet && cleanedPath == "/2013-04-01/delegationset":
		return "ListReusableDelegationSets"
	case method == http.MethodPost && cleanedPath == "/2013-04-01/delegationset":
		return "CreateReusableDelegationSet"
	case method == http.MethodGet && strings.HasPrefix(cleanedPath, "/2013-04-01/delegationset/"):
		return "GetReusableDelegationSet"
	case method == http.MethodDelete && strings.HasPrefix(cleanedPath, "/2013-04-01/delegationset/"):
		return "DeleteReusableDelegationSet"
	case method == http.MethodGet && strings.HasPrefix(cleanedPath, "/2013-04-01/change/"):
		return "GetChange"
	case method == http.MethodGet && strings.HasPrefix(cleanedPath, "/2013-04-01/hostedzonebyname"):
		return "ListHostedZonesByName"
	case method == http.MethodGet && strings.HasPrefix(cleanedPath, "/2013-04-01/geolocation"):
		return "ListGeoLocations"
	case method == http.MethodGet && strings.HasPrefix(cleanedPath, "/2013-04-01/checkeripranges"):
		return "GetCheckerIpRanges"
	default:
		if method == http.MethodGet {
			return "ListHostedZones"
		}
		return "CreateHostedZone"
	}
}

func parseRoute53OperationFromUserAgent(userAgent string) string {
	ua := strings.TrimSpace(userAgent)
	lower := strings.ToLower(ua)
	marker := "md/command#route53."
	if idx := strings.Index(lower, marker); idx >= 0 {
		rest := lower[idx+len(marker):]
		command := route53ReadUntilDelimiter(rest)
		if op := route53OperationByCLICommand[strings.TrimSpace(command)]; op != "" {
			return op
		}
	}

	if matches := route53SDKOperationRegex.FindStringSubmatch(ua); len(matches) == 2 {
		candidate := strings.TrimSpace(matches[1])
		if _, ok := route53OperationByName[candidate]; ok {
			return candidate
		}
	}
	return ""
}

func route53ReadUntilDelimiter(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	end := len(value)
	for _, delimiter := range []string{" ", ";", ")", "("} {
		if idx := strings.Index(value, delimiter); idx >= 0 && idx < end {
			end = idx
		}
	}
	return strings.TrimSpace(value[:end])
}

func route53OperationToCLICommand(operation string) string {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return ""
	}
	var b strings.Builder
	runes := []rune(operation)
	for idx, r := range runes {
		if r >= 'A' && r <= 'Z' {
			if idx > 0 {
				prev := runes[idx-1]
				nextLower := idx+1 < len(runes) && runes[idx+1] >= 'a' && runes[idx+1] <= 'z'
				if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') || ((prev >= 'A' && prev <= 'Z') && nextLower) {
					b.WriteRune('-')
				}
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func route53PathParams(requestPath string) map[string]string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	path = strings.TrimSuffix(path, "/")
	params := map[string]string{}

	if strings.HasPrefix(path, "/2013-04-01/hostedzone/") {
		rel := strings.TrimPrefix(path, "/2013-04-01/hostedzone/")
		parts := strings.Split(rel, "/")
		if len(parts) > 0 && parts[0] != "" {
			params["hostedZoneId"] = parts[0]
		}
	}
	if strings.HasPrefix(path, "/2013-04-01/healthcheck/") {
		rel := strings.TrimPrefix(path, "/2013-04-01/healthcheck/")
		parts := strings.Split(rel, "/")
		if len(parts) > 0 && parts[0] != "" {
			params["healthCheckId"] = parts[0]
		}
	}
	if strings.HasPrefix(path, "/2013-04-01/change/") {
		rel := strings.TrimPrefix(path, "/2013-04-01/change/")
		parts := strings.Split(rel, "/")
		if len(parts) > 0 && parts[0] != "" {
			params["changeId"] = parts[0]
		}
	}
	if strings.HasPrefix(path, "/2013-04-01/delegationset/") {
		rel := strings.TrimPrefix(path, "/2013-04-01/delegationset/")
		parts := strings.Split(rel, "/")
		if len(parts) > 0 && parts[0] != "" {
			params["delegationSetId"] = parts[0]
		}
	}
	return params
}

func respondRoute53Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	env := route53ErrorEnvelope{
		Xmlns: "https://route53.amazonaws.com/doc/2013-04-01/",
		Error: route53ErrorBody{
			Type:    "Sender",
			Code:    code,
			Message: msg,
		},
		RequestID: "stackyard-request",
	}
	if err := xml.NewEncoder(w).Encode(env); err != nil {
		_, _ = fmt.Fprint(w, "<ErrorResponse/>")
	}
}
