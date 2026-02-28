package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type transferStore struct {
	mu     sync.Mutex
	nextID int64
	tags   map[string]map[string]string
}

func newTransferStore() *transferStore {
	return &transferStore{
		nextID: 1,
		tags: map[string]map[string]string{
			"arn:aws:transfer:us-east-1:123456789012:server/s-00000000000000000": {
				"seed": "true",
			},
		},
	}
}

func (s *transferStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	template, ok := transferResponseTemplates[action]
	if !ok {
		return map[string]any{}
	}
	response := transferCloneMap(template)
	s.applyActionLocked(action, payload, response)
	return response
}

func (s *transferStore) applyActionLocked(action string, payload map[string]any, response map[string]any) {
	serverID := transferPayloadString(payload, "ServerId", "s-00000000000000000")
	userName := transferPayloadString(payload, "UserName", "stackyard-user")
	profileID := transferPayloadString(payload, "ProfileId", "p-00000000000000000")
	connectorID := transferPayloadString(payload, "ConnectorId", "c-00000000000000000")
	agreementID := transferPayloadString(payload, "AgreementId", "a-00000000000000000")
	workflowID := transferPayloadString(payload, "WorkflowId", "w-00000000000000000")
	webAppID := transferPayloadString(payload, "WebAppId", "wa-00000000000000000")
	hostKeyID := transferPayloadString(payload, "HostKeyId", "hk-00000000000000000")
	certificateID := transferPayloadString(payload, "CertificateId", "cert-00000000000000000")
	externalID := transferPayloadString(payload, "ExternalId", "ext-00000000000000000")

	switch action {
	case "CreateServer":
		serverID = s.nextIdentifierLocked("s")
		response["ServerId"] = serverID
	case "CreateUser":
		response["ServerId"] = serverID
		response["UserName"] = userName
	case "CreateProfile":
		profileID = s.nextIdentifierLocked("p")
		response["ProfileId"] = profileID
	case "CreateConnector":
		connectorID = s.nextIdentifierLocked("c")
		response["ConnectorId"] = connectorID
	case "CreateAgreement":
		agreementID = s.nextIdentifierLocked("a")
		response["AgreementId"] = agreementID
	case "CreateWorkflow":
		workflowID = s.nextIdentifierLocked("w")
		response["WorkflowId"] = workflowID
	case "CreateWebApp":
		webAppID = s.nextIdentifierLocked("wa")
		response["WebAppId"] = webAppID
	case "CreateAccess", "UpdateAccess":
		response["ServerId"] = serverID
		response["ExternalId"] = externalID
	case "UpdateAgreement":
		response["AgreementId"] = agreementID
	case "UpdateCertificate":
		response["CertificateId"] = certificateID
	case "UpdateConnector":
		response["ConnectorId"] = connectorID
	case "UpdateHostKey":
		response["ServerId"] = serverID
		response["HostKeyId"] = hostKeyID
	case "UpdateProfile":
		response["ProfileId"] = profileID
	case "UpdateServer":
		response["ServerId"] = serverID
	case "UpdateUser":
		response["ServerId"] = serverID
		response["UserName"] = userName
	case "UpdateWebApp", "UpdateWebAppCustomization":
		response["WebAppId"] = webAppID
	case "ImportCertificate":
		certificateID = s.nextIdentifierLocked("cert")
		response["CertificateId"] = certificateID
	case "ImportHostKey":
		hostKeyID = s.nextIdentifierLocked("hk")
		response["ServerId"] = serverID
		response["HostKeyId"] = hostKeyID
	case "ImportSshPublicKey":
		response["ServerId"] = serverID
		response["UserName"] = userName
		response["SshPublicKeyId"] = s.nextIdentifierLocked("spk")
	case "StartDirectoryListing":
		response["ListingId"] = s.nextIdentifierLocked("listing")
		response["OutputFileName"] = transferPayloadString(payload, "OutputFileName", "stackyard-listing.csv")
	case "StartFileTransfer":
		response["TransferId"] = s.nextIdentifierLocked("transfer")
	case "StartRemoteDelete":
		response["DeleteId"] = s.nextIdentifierLocked("delete")
	case "StartRemoteMove":
		response["MoveId"] = s.nextIdentifierLocked("move")
	case "TestIdentityProvider":
		response["StatusCode"] = 200
		response["Url"] = transferPayloadString(payload, "Url", "https://stackyard.local/transfer-idp")
	case "DescribeAccess", "ListAccesses":
		response["ServerId"] = serverID
	case "DescribeExecution", "ListExecutions":
		response["WorkflowId"] = workflowID
	case "DescribeServer":
		if server := transferNestedMap(response, "Server"); server != nil {
			server["Arn"] = transferServerARN(serverID)
		}
	case "DescribeUser":
		response["ServerId"] = serverID
		if user := transferNestedMap(response, "User"); user != nil {
			user["Arn"] = transferUserARN(serverID, userName)
		}
	case "DescribeAgreement":
		if agreement := transferNestedMap(response, "Agreement"); agreement != nil {
			agreement["Arn"] = transferARN("agreement", agreementID)
		}
	case "DescribeCertificate":
		if certificate := transferNestedMap(response, "Certificate"); certificate != nil {
			certificate["Arn"] = transferARN("certificate", certificateID)
		}
	case "DescribeConnector":
		if connector := transferNestedMap(response, "Connector"); connector != nil {
			connector["Arn"] = transferARN("connector", connectorID)
		}
	case "DescribeHostKey":
		if hostKey := transferNestedMap(response, "HostKey"); hostKey != nil {
			hostKey["Arn"] = transferARN("host-key", hostKeyID)
		}
	case "DescribeProfile":
		if profile := transferNestedMap(response, "Profile"); profile != nil {
			profile["Arn"] = transferARN("profile", profileID)
		}
	case "DescribeWorkflow":
		if workflow := transferNestedMap(response, "Workflow"); workflow != nil {
			workflow["Arn"] = transferARN("workflow", workflowID)
		}
	case "DescribeWebApp":
		if webApp := transferNestedMap(response, "WebApp"); webApp != nil {
			webApp["Arn"] = transferARN("webapp", webAppID)
			webApp["WebAppId"] = webAppID
		}
	case "DescribeWebAppCustomization":
		if customization := transferNestedMap(response, "WebAppCustomization"); customization != nil {
			customization["Arn"] = transferARN("webapp", webAppID)
			customization["WebAppId"] = webAppID
		}
	case "ListServers":
		if item := transferFirstListMap(response, "Servers"); item != nil {
			item["Arn"] = transferServerARN(serverID)
		}
	case "ListUsers":
		response["ServerId"] = serverID
		if item := transferFirstListMap(response, "Users"); item != nil {
			item["Arn"] = transferUserARN(serverID, userName)
		}
	case "ListHostKeys":
		response["ServerId"] = serverID
		if item := transferFirstListMap(response, "HostKeys"); item != nil {
			item["Arn"] = transferARN("host-key", hostKeyID)
		}
	case "ListWebApps":
		if item := transferFirstListMap(response, "WebApps"); item != nil {
			item["Arn"] = transferARN("webapp", webAppID)
			item["WebAppId"] = webAppID
		}
	case "TagResource":
		arn := transferPayloadString(payload, "Arn", transferServerARN(serverID))
		tags := transferTagsFromAny(payload["Tags"])
		if len(tags) > 0 {
			if s.tags[arn] == nil {
				s.tags[arn] = map[string]string{}
			}
			for k, v := range tags {
				s.tags[arn][k] = v
			}
		}
	case "UntagResource":
		arn := transferPayloadString(payload, "Arn", transferServerARN(serverID))
		tagKeys := transferPayloadStringSlice(payload, "TagKeys")
		if existing := s.tags[arn]; len(existing) > 0 {
			for _, key := range tagKeys {
				delete(existing, key)
			}
		}
	case "ListTagsForResource":
		arn := transferPayloadString(payload, "Arn", transferServerARN(serverID))
		response["Tags"] = transferTagsToList(s.tags[arn])
	}
}

func (s *transferStore) nextIdentifierLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	if strings.TrimSpace(prefix) == "" {
		prefix = "id"
	}
	return fmt.Sprintf("%s-%017d", prefix, id)
}

var transferResponseTemplates = map[string]map[string]any{
	"CreateAccess": map[string]any{
		"ExternalId": "stackyard",
		"ServerId":   "stackyard",
	},
	"CreateAgreement": map[string]any{
		"AgreementId": "stackyard",
	},
	"CreateConnector": map[string]any{
		"ConnectorId": "stackyard",
	},
	"CreateProfile": map[string]any{
		"ProfileId": "stackyard",
	},
	"CreateServer": map[string]any{
		"ServerId": "stackyard",
	},
	"CreateUser": map[string]any{
		"ServerId": "stackyard",
		"UserName": "stackyard",
	},
	"CreateWebApp": map[string]any{
		"WebAppId": "stackyard",
	},
	"CreateWorkflow": map[string]any{
		"WorkflowId": "stackyard",
	},
	"DeleteAccess":              map[string]any{},
	"DeleteAgreement":           map[string]any{},
	"DeleteCertificate":         map[string]any{},
	"DeleteConnector":           map[string]any{},
	"DeleteHostKey":             map[string]any{},
	"DeleteProfile":             map[string]any{},
	"DeleteServer":              map[string]any{},
	"DeleteSshPublicKey":        map[string]any{},
	"DeleteUser":                map[string]any{},
	"DeleteWebApp":              map[string]any{},
	"DeleteWebAppCustomization": map[string]any{},
	"DeleteWorkflow":            map[string]any{},
	"DescribeAccess": map[string]any{
		"Access":   map[string]any{},
		"ServerId": "stackyard",
	},
	"DescribeAgreement": map[string]any{
		"Agreement": map[string]any{
			"Arn": "stackyard",
		},
	},
	"DescribeCertificate": map[string]any{
		"Certificate": map[string]any{
			"Arn": "stackyard",
		},
	},
	"DescribeConnector": map[string]any{
		"Connector": map[string]any{
			"Arn":        "stackyard",
			"EgressType": "SERVICE_MANAGED",
			"Status":     "ACTIVE",
		},
	},
	"DescribeExecution": map[string]any{
		"Execution":  map[string]any{},
		"WorkflowId": "stackyard",
	},
	"DescribeHostKey": map[string]any{
		"HostKey": map[string]any{
			"Arn": "stackyard",
		},
	},
	"DescribeProfile": map[string]any{
		"Profile": map[string]any{
			"Arn": "stackyard",
		},
	},
	"DescribeSecurityPolicy": map[string]any{
		"SecurityPolicy": map[string]any{
			"SecurityPolicyName": "stackyard",
		},
	},
	"DescribeServer": map[string]any{
		"Server": map[string]any{
			"Arn": "stackyard",
		},
	},
	"DescribeUser": map[string]any{
		"ServerId": "stackyard",
		"User": map[string]any{
			"Arn": "stackyard",
		},
	},
	"DescribeWebApp": map[string]any{
		"WebApp": map[string]any{
			"Arn":      "stackyard",
			"WebAppId": "stackyard",
		},
	},
	"DescribeWebAppCustomization": map[string]any{
		"WebAppCustomization": map[string]any{
			"Arn":      "stackyard",
			"WebAppId": "stackyard",
		},
	},
	"DescribeWorkflow": map[string]any{
		"Workflow": map[string]any{
			"Arn": "stackyard",
		},
	},
	"ImportCertificate": map[string]any{
		"CertificateId": "stackyard",
	},
	"ImportHostKey": map[string]any{
		"HostKeyId": "stackyard",
		"ServerId":  "stackyard",
	},
	"ImportSshPublicKey": map[string]any{
		"ServerId":       "stackyard",
		"SshPublicKeyId": "stackyard",
		"UserName":       "stackyard",
	},
	"ListAccesses": map[string]any{
		"Accesses": []any{
			map[string]any{},
		},
		"ServerId": "stackyard",
	},
	"ListAgreements": map[string]any{
		"Agreements": []any{
			map[string]any{},
		},
	},
	"ListCertificates": map[string]any{
		"Certificates": []any{
			map[string]any{},
		},
	},
	"ListConnectors": map[string]any{
		"Connectors": []any{
			map[string]any{},
		},
	},
	"ListExecutions": map[string]any{
		"Executions": []any{
			map[string]any{},
		},
		"WorkflowId": "stackyard",
	},
	"ListFileTransferResults": map[string]any{
		"FileTransferResults": []any{
			map[string]any{
				"FilePath":   "stackyard",
				"StatusCode": "QUEUED",
			},
		},
	},
	"ListHostKeys": map[string]any{
		"HostKeys": []any{
			map[string]any{
				"Arn": "stackyard",
			},
		},
		"ServerId": "stackyard",
	},
	"ListProfiles": map[string]any{
		"Profiles": []any{
			map[string]any{},
		},
	},
	"ListSecurityPolicies": map[string]any{
		"SecurityPolicyNames": []any{
			"stackyard",
		},
	},
	"ListServers": map[string]any{
		"Servers": []any{
			map[string]any{
				"Arn": "stackyard",
			},
		},
	},
	"ListTagsForResource": map[string]any{},
	"ListUsers": map[string]any{
		"ServerId": "stackyard",
		"Users": []any{
			map[string]any{
				"Arn": "stackyard",
			},
		},
	},
	"ListWebApps": map[string]any{
		"WebApps": []any{
			map[string]any{
				"Arn":      "stackyard",
				"WebAppId": "stackyard",
			},
		},
	},
	"ListWorkflows": map[string]any{
		"Workflows": []any{
			map[string]any{},
		},
	},
	"SendWorkflowStepState": map[string]any{},
	"StartDirectoryListing": map[string]any{
		"ListingId":      "stackyard",
		"OutputFileName": "stackyard",
	},
	"StartFileTransfer": map[string]any{
		"TransferId": "stackyard",
	},
	"StartRemoteDelete": map[string]any{
		"DeleteId": "stackyard",
	},
	"StartRemoteMove": map[string]any{
		"MoveId": "stackyard",
	},
	"StartServer":    map[string]any{},
	"StopServer":     map[string]any{},
	"TagResource":    map[string]any{},
	"TestConnection": map[string]any{},
	"TestIdentityProvider": map[string]any{
		"StatusCode": 1,
		"Url":        "stackyard",
	},
	"UntagResource": map[string]any{},
	"UpdateAccess": map[string]any{
		"ExternalId": "stackyard",
		"ServerId":   "stackyard",
	},
	"UpdateAgreement": map[string]any{
		"AgreementId": "stackyard",
	},
	"UpdateCertificate": map[string]any{
		"CertificateId": "stackyard",
	},
	"UpdateConnector": map[string]any{
		"ConnectorId": "stackyard",
	},
	"UpdateHostKey": map[string]any{
		"HostKeyId": "stackyard",
		"ServerId":  "stackyard",
	},
	"UpdateProfile": map[string]any{
		"ProfileId": "stackyard",
	},
	"UpdateServer": map[string]any{
		"ServerId": "stackyard",
	},
	"UpdateUser": map[string]any{
		"ServerId": "stackyard",
		"UserName": "stackyard",
	},
	"UpdateWebApp": map[string]any{
		"WebAppId": "stackyard",
	},
	"UpdateWebAppCustomization": map[string]any{
		"WebAppId": "stackyard",
	},
}

func transferCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = transferCloneAny(value)
	}
	return out
}

func transferCloneSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, value := range in {
		out = append(out, transferCloneAny(value))
	}
	return out
}

func transferCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return transferCloneMap(typed)
	case []any:
		return transferCloneSlice(typed)
	default:
		return typed
	}
}

func transferPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for current, raw := range payload {
		if strings.EqualFold(strings.TrimSpace(current), key) {
			if value := strings.TrimSpace(fmt.Sprintf("%v", raw)); value != "" {
				return value
			}
			break
		}
	}
	return fallback
}

func transferPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	for current, raw := range payload {
		if !strings.EqualFold(strings.TrimSpace(current), key) {
			continue
		}
		list, ok := raw.([]any)
		if !ok {
			return nil
		}
		out := make([]string, 0, len(list))
		for _, item := range list {
			value := strings.TrimSpace(fmt.Sprintf("%v", item))
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	}
	return nil
}

func transferNestedMap(container map[string]any, key string) map[string]any {
	value, ok := container[key]
	if !ok {
		return nil
	}
	nested, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return nested
}

func transferFirstListMap(container map[string]any, key string) map[string]any {
	value, ok := container[key]
	if !ok {
		return nil
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	return item
}

func transferTagsFromAny(raw any) map[string]string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := transferPayloadString(entry, "Key", "")
		if key == "" {
			continue
		}
		out[key] = transferPayloadString(entry, "Value", "")
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func transferTagsToList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func transferARN(resourceType, resourceID string) string {
	resourceType = strings.TrimSpace(resourceType)
	resourceID = strings.TrimSpace(resourceID)
	if resourceType == "" {
		resourceType = "resource"
	}
	if resourceID == "" {
		resourceID = "stackyard"
	}
	return fmt.Sprintf("arn:aws:transfer:us-east-1:123456789012:%s/%s", resourceType, resourceID)
}

func transferServerARN(serverID string) string {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		serverID = "s-00000000000000000"
	}
	return fmt.Sprintf("arn:aws:transfer:us-east-1:123456789012:server/%s", serverID)
}

func transferUserARN(serverID, userName string) string {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		serverID = "s-00000000000000000"
	}
	userName = strings.TrimSpace(userName)
	if userName == "" {
		userName = "stackyard-user"
	}
	return fmt.Sprintf("arn:aws:transfer:us-east-1:123456789012:user/%s/%s", serverID, userName)
}
