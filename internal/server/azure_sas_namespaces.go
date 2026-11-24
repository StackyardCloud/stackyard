package server

import (
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Shared pieces of the Service Bus family (Service Bus, Relay, Notification
// Hubs): namespaces under servicebus.windows.net and SAS authorization rules
// with Listen/Send/Manage rights.

// azureSASNamespaceTypes share one DNS namespace, so a name taken by any of
// them is unavailable to all.
var azureSASNamespaceTypes = []string{
	"Microsoft.ServiceBus/namespaces",
	"Microsoft.Relay/namespaces",
	"Microsoft.NotificationHubs/namespaces",
	"Microsoft.EventHub/namespaces",
}

var azureSASNamespaceNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{4,48}[a-zA-Z0-9]$`)

func azureSASValidateNamespaceName(name string) string {
	lower := strings.ToLower(name)
	if !azureSASNamespaceNamePattern.MatchString(name) || strings.HasSuffix(lower, "-sb") || strings.HasSuffix(lower, "-mgmt") {
		return "Namespace names must be 6-50 letters, digits or hyphens, start with a letter, end with a letter or digit, and not end with -sb or -mgmt."
	}
	return ""
}

func (s *Server) azureSASNamespaceNameTaken(name string) bool {
	return len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		if !strings.EqualFold(rec.Name, name) {
			return false
		}
		for _, typ := range azureSASNamespaceTypes {
			if strings.EqualFold(rec.Type, typ) {
				return true
			}
		}
		return false
	})) > 0
}

func azureSASNamespaceProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"serviceBusEndpoint": "https://" + strings.ToLower(rec.Name) + ".servicebus.windows.net:443/",
		"status":             "Active",
	}
}

// azureSASRuleType registers authorization rules at typePath. withEntityPath
// adds ";EntityPath={entity}" to connection strings of rules that belong to
// an entity (a queue or hybrid connection) rather than the namespace.
func azureSASRuleType(namespace, typePath string, withEntityPath bool) azureARMResourceType {
	return azureARMResourceType{
		Namespace: namespace,
		Type:      typePath,
		Proxy:     true,
		Validate:  azureSASValidateRule,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"listkeys": func(s *Server, record *azureARMResource, _ map[string]any) (int, any) {
				return http.StatusOK, s.azureSASListKeys(record, withEntityPath)
			},
			"regeneratekeys": func(s *Server, record *azureARMResource, payload map[string]any) (int, any) {
				label := map[string]string{"PrimaryKey": "primary", "SecondaryKey": "secondary"}[azureARMString(payload["keyType"])]
				if label == "" {
					return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "keyType must be PrimaryKey or SecondaryKey.")
				}
				key := azureARMString(payload["key"])
				if len(key) > 256 {
					return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "key must be at most 256 characters.")
				}
				s.azureRegenerateResourceKey(record, label, key)
				return http.StatusOK, s.azureSASListKeys(record, withEntityPath)
			},
		},
	}
}

// azureKeySlot names one access key of one resource instance. CreatedAt keeps
// a resource recreated under the same ID from inheriting regenerated keys.
type azureKeySlot struct {
	s       *Server
	id      string
	created int64
	label   string
}

// azureRegeneratedKeys holds keys clients regenerated; untouched keys are
// derived from the resource ID. Records are immutable and keys must never
// appear in GET responses, so they live beside the store.
// ponytail: entries outlive deleted resources (one string per regenerated key);
// key them into the store if that ever matters.
var azureRegeneratedKeys sync.Map

// azureResourceKey returns the current access key labelled label.
func (s *Server) azureResourceKey(rec *azureARMResource, label string) string {
	if v, ok := azureRegeneratedKeys.Load(azureKeySlot{s, azureARMKey(rec.ID), rec.CreatedAt.UnixNano(), label}); ok {
		return v.(string)
	}
	return azureARMDerivedKey(rec.ID, label)
}

// azureRegenerateResourceKey replaces the key labelled label with key, or with
// a new random one when key is empty.
func (s *Server) azureRegenerateResourceKey(rec *azureARMResource, label, key string) {
	if key == "" {
		key = azureARMDerivedKey(rec.ID, label+"#"+uuid.NewString())
	}
	azureRegeneratedKeys.Store(azureKeySlot{s, azureARMKey(rec.ID), rec.CreatedAt.UnixNano(), label}, key)
}

// azureSASRootRule returns a DefaultChildren hook creating the
// RootManageSharedAccessKey rule Azure adds to every new namespace.
func azureSASRootRule(ruleType azureARMResourceType) func(*azureARMResource) []*azureARMResource {
	return func(parent *azureARMResource) []*azureARMResource {
		return []*azureARMResource{azureARMChild(parent, ruleType, "RootManageSharedAccessKey",
			map[string]any{"rights": []any{"Listen", "Manage", "Send"}})}
	}
}

// azureSASValidateRule requires Listen/Send/Manage rights, where Manage also
// needs Listen and Send.
func azureSASValidateRule(payload map[string]any) string {
	rights, _ := azureARMObject(payload["properties"])["rights"].([]any)
	granted := map[string]bool{}
	for _, right := range rights {
		value := azureARMString(right)
		if value != "Listen" && value != "Send" && value != "Manage" {
			return "properties.rights may only contain Listen, Send and Manage."
		}
		granted[value] = true
	}
	switch {
	case len(granted) == 0:
		return "properties.rights is required."
	case granted["Manage"] && (!granted["Listen"] || !granted["Send"]):
		return "Manage rights require Listen and Send."
	}
	return ""
}

// azureSASListKeys returns a rule's keys and connection strings. The record ID
// is .../namespaces/{namespace}[/{entityType}/{entity}]/authorizationRules/{rule}.
func (s *Server) azureSASListKeys(record *azureARMResource, withEntityPath bool) map[string]any {
	parts := strings.Split(record.ID, "/")
	var namespace, entity string
	for i := 0; i+1 < len(parts); i++ {
		if strings.EqualFold(parts[i], "namespaces") {
			namespace = parts[i+1]
			// An entity sits between the namespace and authorizationRules/{rule}.
			if i+3 < len(parts)-2 {
				entity = parts[i+3]
			}
			break
		}
	}
	primary, secondary := s.azureResourceKey(record, "primary"), s.azureResourceKey(record, "secondary")
	connection := func(key string) string {
		conn := "Endpoint=sb://" + strings.ToLower(namespace) + ".servicebus.windows.net/;SharedAccessKeyName=" + record.Name + ";SharedAccessKey=" + key
		if withEntityPath && entity != "" {
			conn += ";EntityPath=" + entity
		}
		return conn
	}
	return map[string]any{
		"keyName":                   record.Name,
		"primaryKey":                primary,
		"secondaryKey":              secondary,
		"primaryConnectionString":   connection(primary),
		"secondaryConnectionString": connection(secondary),
	}
}
