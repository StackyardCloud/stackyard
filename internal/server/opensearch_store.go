package server

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
)

type openSearchStore struct {
	mu                 sync.Mutex
	defaultDomainName  string
	defaultAccountID   string
	defaultVPCEndpoint string
}

func newOpenSearchStore() *openSearchStore {
	return &openSearchStore{
		defaultDomainName:  "stackyard-opensearch-domain",
		defaultAccountID:   "123456789012",
		defaultVPCEndpoint: "vpce-1234567890abcdef0",
	}
}

func (s *openSearchStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	domainName := s.resolveDomainName(payload, pathParams)
	principal := s.resolvePrincipal(payload)

	switch action {
	case "AuthorizeVpcEndpointAccess":
		return map[string]any{
			"AuthorizedPrincipal": map[string]any{
				"PrincipalType": "AWS_ACCOUNT",
				"Principal":     principal,
			},
		}
	case "CreateVpcEndpoint", "UpdateVpcEndpoint":
		return map[string]any{"VpcEndpoint": map[string]any{}}
	case "DeleteVpcEndpoint":
		return map[string]any{"VpcEndpointSummary": map[string]any{}}
	case "DescribeDomain":
		return map[string]any{"DomainStatus": s.domainStatus(domainName)}
	case "DescribeDomainConfig", "UpdateDomainConfig":
		return map[string]any{"DomainConfig": map[string]any{}}
	case "DescribeDomains":
		return map[string]any{"DomainStatusList": []any{s.domainStatus(domainName)}}
	case "DescribeVpcEndpoints":
		return map[string]any{
			"VpcEndpoints":      []any{},
			"VpcEndpointErrors": []any{},
		}
	case "ListVpcEndpointAccess":
		return map[string]any{
			"AuthorizedPrincipalList": []any{
				map[string]any{
					"PrincipalType": "AWS_ACCOUNT",
					"Principal":     principal,
				},
			},
			"NextToken": "",
		}
	case "ListVpcEndpoints", "ListVpcEndpointsForDomain":
		return map[string]any{
			"VpcEndpointSummaryList": []any{},
			"NextToken":              "",
		}
	default:
		return map[string]any{}
	}
}

func (s *openSearchStore) domainStatus(domainName string) map[string]any {
	if strings.TrimSpace(domainName) == "" {
		domainName = s.defaultDomainName
	}
	return map[string]any{
		"DomainId":      fmt.Sprintf("%s/%s", s.defaultAccountID, domainName),
		"DomainName":    domainName,
		"ARN":           fmt.Sprintf("arn:aws:es:us-east-1:%s:domain/%s", s.defaultAccountID, domainName),
		"ClusterConfig": map[string]any{},
	}
}

func (s *openSearchStore) resolveDomainName(payload map[string]any, pathParams map[string]string) string {
	if value := opensearchPathParam(pathParams, "DomainName"); value != "" {
		return value
	}
	if value := opensearchPayloadString(payload, "DomainName"); value != "" {
		return value
	}
	if value := opensearchPayloadString(payload, "domainName"); value != "" {
		return value
	}
	return s.defaultDomainName
}

func (s *openSearchStore) resolvePrincipal(payload map[string]any) string {
	for _, key := range []string{"AuthorizedPrincipal", "Account", "Principal", "account"} {
		if value := opensearchPayloadString(payload, key); value != "" {
			return value
		}
	}
	return s.defaultAccountID
}

func opensearchPathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	if value, ok := pathParams[key]; ok {
		return strings.TrimSpace(value)
	}
	for k, value := range pathParams {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func opensearchPayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		return strings.TrimSpace(opensearchAnyToString(value))
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(opensearchAnyToString(value))
		}
	}
	return ""
}

func opensearchAnyToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
