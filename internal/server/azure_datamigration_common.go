package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Helpers shared by the Data Migration, Data Protection, Data Replication,
// Data Share, Defender for Cloud and Defender EASM emulations.

// azureDMSOperationsView registers the provider operations list
// (GET /providers/{namespace}/operations): read, write and delete for each
// [type, display name] pair.
func azureDMSOperationsView(namespace, provider string, resources ...[2]string) {
	list := azureDMSOperationsList(namespace, provider, resources...)
	registerAzureARMView(namespace, "operations", func(*Server, string) []map[string]any { return list() })
}

// azureDMSOperationsList returns a builder for a provider operations list.
func azureDMSOperationsList(namespace, provider string, resources ...[2]string) func() []map[string]any {
	return func() []map[string]any {
		ops := make([]map[string]any, 0, 3*len(resources))
		for _, res := range resources {
			for _, verb := range []string{"read", "write", "delete"} {
				ops = append(ops, map[string]any{
					"name": namespace + "/" + res[0] + "/" + verb, "isDataAction": false, "origin": "user,system",
					"display": map[string]any{
						"provider": provider, "resource": res[1],
						"operation":   strings.ToUpper(verb[:1]) + verb[1:] + " " + res[1],
						"description": strings.ToUpper(verb[:1]) + verb[1:] + " " + res[1] + ".",
					},
				})
			}
		}
		return ops
	}
}

// azureDMSUnknownRoute answers 404 InvalidResourceType for a request under
// one of these providers that no registered route served, as ARM does for
// an unknown resource type.
func azureDMSUnknownRoute(w http.ResponseWriter, r *http.Request, namespace string) bool {
	if !strings.Contains(strings.ToLower(rawRequestPath(r)), "/providers/"+strings.ToLower(namespace)+"/") {
		return false
	}
	azureARMError(w, http.StatusNotFound, "InvalidResourceType", "The resource type could not be found in the namespace '"+namespace+"' for api version '"+r.URL.Query().Get("api-version")+"'.")
	return true
}

// azureDMSOperation is a finished long-running operation, which these
// emulations complete at once.
type azureDMSOperation struct {
	ID       string
	Scope    string // resource ID or subscription the operation ran against
	Name     string
	Started  time.Time
	Result   map[string]any // the operation's result payload, if any
	Location string
}

// azureDMSOperations keeps recent operations per provider so their status
// can be read back. ponytail: oldest dropped past azureDMSMaxOperations.
type azureDMSOperations struct {
	mu    sync.Mutex
	order []string
	byID  map[string]azureDMSOperation
}

const azureDMSMaxOperations = 1000

func (o *azureDMSOperations) add(scope, name, location string, result map[string]any) azureDMSOperation {
	op := azureDMSOperation{ID: uuid.NewString(), Scope: scope, Name: name, Started: time.Now().UTC(), Result: result, Location: location}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.byID == nil {
		o.byID = map[string]azureDMSOperation{}
	}
	if len(o.order) >= azureDMSMaxOperations {
		delete(o.byID, o.order[0])
		o.order = o.order[1:]
	}
	o.order = append(o.order, op.ID)
	o.byID[strings.ToLower(op.ID)] = op
	return op
}

// within returns the operations whose scope is under prefix, oldest first.
func (o *azureDMSOperations) within(prefix string) []azureDMSOperation {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []azureDMSOperation
	for _, id := range o.order {
		if op := o.byID[strings.ToLower(id)]; strings.HasPrefix(azureARMKey(op.Scope), azureARMKey(prefix)) {
			out = append(out, op)
		}
	}
	return out
}

func (o *azureDMSOperations) get(id string) (azureDMSOperation, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	op, ok := o.byID[strings.ToLower(id)]
	return op, ok
}

// status returns the ARM operation status body of a finished operation.
func (op azureDMSOperation) status(id string) map[string]any {
	return map[string]any{
		"id": id, "name": op.ID, "status": "Succeeded",
		"startTime": op.Started.Format(time.RFC3339), "endTime": op.Started.Format(time.RFC3339),
	}
}

// azureDMSCheckName answers a name availability check against existing
// records of fullType whose IDs start with scope.
func (s *Server) azureDMSCheckName(scope, fullType string, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required.")
	}
	prefix := azureARMKey(scope + "/")
	taken := len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, fullType) && strings.EqualFold(rec.Name, name) && strings.HasPrefix(azureARMKey(rec.ID), prefix)
	})) > 0
	if taken {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The name '" + name + "' is already in use."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true}
}

// azureDMSUpdateLocked stores a copy of the record at id with props merged
// into its properties and returns it, or nil when it is gone. Caller holds
// providerStorageMu.
func (s *Server) azureDMSUpdateLocked(id string, props map[string]any) *azureARMResource {
	current := s.azureARMResources[azureARMKey(id)]
	if current == nil {
		return nil
	}
	updated := *current
	updated.Properties = azureARMMerge(current.Properties, props)
	updated.UpdatedAt = time.Now().UTC()
	s.azureARMResources[azureARMKey(id)] = &updated
	return &updated
}

// azureDMSStateFor returns the server's state named name, made by create on first use.
// It lives on the Server, so it goes away with it.
func azureDMSStateFor[T any](s *Server, name string, create func() *T) *T {
	if st, ok := s.azureSideState.Load(name); ok {
		return st.(*T)
	}
	st, _ := s.azureSideState.LoadOrStore(name, create())
	return st.(*T)
}
