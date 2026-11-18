package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.SerialConsole: serial ports on virtual machines and scale set
// instances, and the subscription switch that turns Serial Console off for
// all of them. No console runs: connect returns a WebSocket address that
// nothing serves. The subscription's status is stored as a record of an
// unregistered type, so only the view and actions below serve it.
const (
	azureSerialConsoleNamespace   = "Microsoft.SerialConsole"
	azureSerialConsoleServiceType = azureSerialConsoleNamespace + "/consoleServices"
)

// Serial ports attach to a virtual machine or a scale set instance.
var azureSerialPortScope = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.Compute/(virtualMachines/[^/]+|virtualMachineScaleSets/[^/]+/virtualMachines/[^/]+)$`)

func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureSerialConsoleNamespace, Type: "serialPorts", Extension: true,
		ValidateScope: func(scope string) string {
			if !azureSerialPortScope.MatchString(scope) {
				return "Serial ports belong to a Microsoft.Compute virtual machine or virtual machine scale set instance."
			}
			return ""
		},
		Validate: func(payload map[string]any) string {
			if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
				return msg
			}
			props := azureARMObject(payload["properties"])
			if msg := azureLabEnum(props, "properties.", "state", []string{"enabled", "disabled"}); msg != "" {
				return msg
			}
			return azureLabEnum(props, "properties.", "connectionState", []string{"active", "inactive"})
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			if rec.Properties["connectionState"] == nil {
				return map[string]any{"connectionState": "inactive"}
			}
			return nil
		},
		NoPatch: true,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"connect": azureSerialPortConnect,
		},
	})
	// The subscription list holds every port in the subscription; a
	// resource's list and GET hold only its own ports.
	registerAzureARMView(azureSerialConsoleNamespace, "serialPorts", func(s *Server, scope string) []map[string]any {
		prefix := azureARMKey(scope + "/")
		nested := strings.Count(scope, "/") > 2
		items := []map[string]any{}
		for _, rec := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
			key := azureARMKey(rec.ID)
			if rec.Type != azureSerialConsoleNamespace+"/serialPorts" || !strings.HasPrefix(key, prefix) {
				return false
			}
			return !nested || strings.HasPrefix(key[len(prefix):], azureARMKey("providers/"+azureSerialConsoleNamespace+"/serialPorts/"))
		}) {
			items = append(items, rec.fixture())
		}
		return items
	})
	registerAzureARMView(azureSerialConsoleNamespace, "consoleServices", func(s *Server, scope string) []map[string]any {
		sub, ok := azureARMSubscriptionScope(scope)
		if !ok {
			return nil
		}
		return []map[string]any{{
			"id": azureSerialConsoleServiceID(sub), "name": "default", "type": azureSerialConsoleServiceType,
			"properties": map[string]any{"disabled": s.azureSerialConsoleDisabled(sub)},
		}}
	})
	registerAzureARMScopeAction(azureSerialConsoleNamespace, "consoleServices/default/enableConsole", azureSerialConsoleSwitch(false))
	registerAzureARMScopeAction(azureSerialConsoleNamespace, "consoleServices/default/disableConsole", azureSerialConsoleSwitch(true))
}

// azureARMSubscriptionScope returns the subscription ID of a
// subscription scope, and false for any narrower scope.
func azureARMSubscriptionScope(scope string) (string, bool) {
	seg := strings.Split(strings.TrimPrefix(scope, "/"), "/")
	if len(seg) != 2 || !strings.EqualFold(seg[0], "subscriptions") {
		return "", false
	}
	return seg[1], true
}

func azureSerialConsoleServiceID(sub string) string {
	return "/subscriptions/" + sub + "/providers/" + azureSerialConsoleServiceType + "/default"
}

// azureSerialConsoleDisabled reports whether Serial Console is off for the
// subscription; it is on until disableConsole turns it off.
func (s *Server) azureSerialConsoleDisabled(sub string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	rec := s.azureARMResources[azureARMKey(azureSerialConsoleServiceID(sub))]
	return rec != nil && rec.Properties["disabled"] == true
}

func azureSerialConsoleSwitch(disabled bool) azureARMScopeAction {
	return func(s *Server, scope string, _ map[string]any) (int, any) {
		sub, ok := azureARMSubscriptionScope(scope)
		if !ok {
			return http.StatusBadRequest, azureARMErrorBody("InvalidScope", "Serial Console is turned on and off for a whole subscription.")
		}
		id := azureSerialConsoleServiceID(sub)
		now := time.Now().UTC()
		s.providerStorageMu.Lock()
		rec := &azureARMResource{ID: id, SubscriptionID: sub, Name: "default", Type: azureSerialConsoleServiceType, CreatedAt: now, UpdatedAt: now}
		if existing := s.azureARMResources[azureARMKey(id)]; existing != nil {
			rec.CreatedAt = existing.CreatedAt
		}
		rec.Properties = map[string]any{"disabled": disabled}
		s.azureARMResources[azureARMKey(id)] = rec
		s.providerStorageMu.Unlock()
		return http.StatusOK, map[string]any{"properties": map[string]any{"disabled": disabled}}
	}
}

// azureSerialPortConnect returns the WebSocket address of a port's console
// session, refused while the port or the subscription's Serial Console is
// disabled.
func azureSerialPortConnect(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	switch {
	case s.azureSerialConsoleDisabled(rec.SubscriptionID):
		return http.StatusConflict, azureARMErrorBody("SerialConsoleDisabled", "Serial Console is disabled for subscription '"+rec.SubscriptionID+"'.")
	case rec.Properties["state"] == "disabled":
		return http.StatusConflict, azureARMErrorBody("SerialPortDisabled", "Serial port '"+rec.Name+"' is disabled.")
	}
	sum := sha256.Sum256([]byte(azureARMKey(rec.ID)))
	container := hex.EncodeToString(sum[:6])
	return http.StatusOK, map[string]any{
		"connectionString": "wss://serialconsole.stackyard.invalid/n/connector/" + container + "/sessions/" + uuid.NewString() + "/client",
	}
}
