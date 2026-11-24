package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	azureDataShare20210801APIVersion = "2021-08-01"
	azureDSHRNamespace               = "Microsoft.DataShare"
	azureDSHRMaxSyncs                = 100 // kept per share subscription, newest first
)

// Azure Data Share (Microsoft.DataShare 2021-08-01) on the shared ARM
// handler. A provider account's share invites a consumer; a share
// subscription that names the invitation's invitationId accepts it, links
// the two (provider share subscriptions, consumer source data sets, source
// synchronization settings) and can synchronize. Synchronizations finish at
// once and copy nothing.
var (
	azureDSHRDataSetKinds = []string{"Blob", "Container", "BlobFolder", "AdlsGen2File", "AdlsGen2Folder", "AdlsGen2FileSystem", "AdlsGen1File", "AdlsGen1Folder", "KustoCluster", "KustoDatabase", "KustoTable", "SqlDBTable", "SqlDWTable", "SynapseWorkspaceSqlPoolTable"}
	azureDSHRIntervals    = []string{"Hour", "Day"}
	azureDSHREmailRE      = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	// azureDSHRProviderRouteRE matches provider share subscription actions.
	azureDSHRProviderRouteRE = regexp.MustCompile(`(?i)^(/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.DataShare/accounts/[^/]+/shares/[^/]+)/providerShareSubscriptions/([^/]+)/(adjust|reinstate|revoke)$`)
	azureDSHRLocationRouteRE = regexp.MustCompile(`(?i)^/providers/Microsoft\.DataShare/locations/([^/]+)/(rejectInvitation|registerEmail|activateEmail)$`)
)

type azureDSHRSyncs struct {
	mu    sync.Mutex
	bySub map[string][]map[string]any
}

func (s *Server) azureDSHRSyncState() *azureDSHRSyncs {
	return azureDMSStateFor(s, "datashare-syncs", func() *azureDSHRSyncs { return &azureDSHRSyncs{bySub: map[string][]map[string]any{}} })
}

func init() {
	ns := azureDSHRNamespace
	owner := func(rec *azureARMResource) map[string]any {
		return map[string]any{"createdAt": rec.CreatedAt.Format(time.RFC3339), "userName": "stackyard", "userEmail": "stackyard@example.com"}
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts",
		Validate: func(payload map[string]any) string {
			if azureARMString(azureARMObject(payload["identity"])["type"]) != "SystemAssigned" {
				return "identity.type must be SystemAssigned."
			}
			return ""
		},
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			rec.Identity = map[string]any{"type": "SystemAssigned", "principalId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(), "tenantId": "00000000-0000-0000-0000-000000000000"}
			return nil, ""
		},
		ReadOnlyProperties: owner,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/shares", Proxy: true, NoPatch: true,
		Validate:           azureEnumProperty("shareKind", "CopyBased", "InPlace"),
		ReadOnlyProperties: owner,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"listsynchronizations": func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
				return s.azureDSHRShareSyncs(rec.ID, "")
			},
			"listsynchronizationdetails": func(s *Server, rec *azureARMResource, p map[string]any) (int, any) {
				return s.azureDSHRShareSyncDetails(rec.ID, p)
			},
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/shares/invitations", Proxy: true, NoPatch: true,
		Validate:           azureDSHRInvitationValidate,
		StateProperties:    map[string]any{"invitationStatus": "Pending"},
		ReadOnlyProperties: azureDSHRInvitationProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/shares/dataSets", Proxy: true, NoPatch: true,
		Validate: azureDSHRDataSetValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"dataSetId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/shares/synchronizationSettings", Proxy: true, NoPatch: true,
		Validate: azureDSHRScheduleValidate, ReadOnlyProperties: owner,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/shareSubscriptions", Proxy: true, NoPatch: true,
		Validate:  azureDSHRShareSubscriptionValidate,
		Reconcile: azureDSHRShareSubscriptionReconcile,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"synchronize":           azureDSHRSynchronize,
			"cancelsynchronization": azureDSHRCancelSync,
			"listsynchronizations": func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
				return s.azureDSHRShareSyncs("", rec.ID)
			},
			"listsynchronizationdetails": func(s *Server, rec *azureARMResource, p map[string]any) (int, any) {
				return s.azureDSHRSubscriptionSyncDetails(rec, p)
			},
			"listsourcesharesynchronizationsettings": azureDSHRSourceSettings,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/shareSubscriptions/dataSetMappings", Proxy: true, NoPatch: true,
		Validate:  azureDSHRMappingValidate,
		Reconcile: azureDSHRMappingReconcile,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/shareSubscriptions/triggers", Proxy: true, NoPatch: true,
		Validate: azureDSHRScheduleValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"triggerStatus": "Active", "createdAt": rec.CreatedAt.Format(time.RFC3339), "userName": "stackyard"}
		},
	})
	registerAzureARMView(ns, "accounts/shareSubscriptions/consumerSourceDataSets", azureDSHRConsumerSourceDataSets)
	registerAzureARMView(ns, "accounts/shares/providerShareSubscriptions", azureDSHRProviderShareSubscriptions)
	registerAzureARMView(ns, "locations/consumerInvitations", func(s *Server, scope string) []map[string]any {
		return s.azureDSHRConsumerInvitations(scope + "/consumerInvitations/")
	})
	registerAzureARMView(ns, "listInvitations", func(s *Server, _ string) []map[string]any {
		return s.azureDSHRConsumerInvitations("")
	})
	azureDMSOperationsView(ns, "Microsoft Data Share",
		[2]string{"accounts", "accounts"}, [2]string{"accounts/shares", "shares"}, [2]string{"accounts/shares/invitations", "invitations"},
		[2]string{"accounts/shares/dataSets", "data sets"}, [2]string{"accounts/shares/synchronizationSettings", "synchronization settings"},
		[2]string{"accounts/shareSubscriptions", "share subscriptions"}, [2]string{"accounts/shareSubscriptions/dataSetMappings", "data set mappings"},
		[2]string{"accounts/shareSubscriptions/triggers", "triggers"})
}

func azureDSHRInvitationValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	email := azureARMString(props["targetEmail"])
	directory, object := azureARMString(props["targetActiveDirectoryId"]), azureARMString(props["targetObjectId"])
	switch {
	case email == "" && (directory == "" || object == ""):
		return "properties.targetEmail, or targetActiveDirectoryId and targetObjectId, is required."
	case email != "" && !azureDSHREmailRE.MatchString(email):
		return "properties.targetEmail must be an email address."
	}
	if expiry := azureARMString(props["expirationDate"]); expiry != "" {
		if _, err := time.Parse(time.RFC3339, expiry); err != nil {
			return "properties.expirationDate must be an RFC 3339 timestamp."
		}
	}
	return ""
}

func azureDSHRInvitationProperties(rec *azureARMResource) map[string]any {
	// The invitation ID is stable per invitation resource.
	share := rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/invitations/")]
	return map[string]any{
		"invitationId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
		"sentAt":       rec.CreatedAt.Format(time.RFC3339), "shareName": share[strings.LastIndex(share, "/")+1:],
		"userName": "stackyard", "userEmail": "stackyard@example.com",
	}
}

func azureDSHRDataSetValidate(payload map[string]any) string {
	kind := azureARMString(payload["kind"])
	if !slices.Contains(azureDSHRDataSetKinds, kind) {
		return "kind must be one of " + strings.Join(azureDSHRDataSetKinds, ", ") + "."
	}
	props := azureARMObject(payload["properties"])
	required := []string{"resourceGroup", "storageAccountName", "subscriptionId"}
	switch {
	case strings.HasPrefix(kind, "Kusto"):
		required = []string{"kustoClusterResourceId"}
		if kind != "KustoCluster" {
			required = []string{"kustoDatabaseResourceId"}
		}
	case strings.HasPrefix(kind, "Sql") || strings.HasPrefix(kind, "Synapse"):
		required = []string{"tableName"}
	}
	for _, key := range required {
		if strings.TrimSpace(azureARMString(props[key])) == "" {
			return "properties." + key + " is required for " + kind + " data sets."
		}
	}
	return ""
}

func azureDSHRScheduleValidate(payload map[string]any) string {
	if azureARMString(payload["kind"]) != "ScheduleBased" {
		return "kind must be ScheduleBased."
	}
	props := azureARMObject(payload["properties"])
	if !slices.Contains(azureDSHRIntervals, azureARMString(props["recurrenceInterval"])) {
		return "properties.recurrenceInterval must be Hour or Day."
	}
	if _, err := time.Parse(time.RFC3339, azureARMString(props["synchronizationTime"])); err != nil {
		return "properties.synchronizationTime must be an RFC 3339 timestamp."
	}
	if mode := props["synchronizationMode"]; mode != nil && mode != "Incremental" && mode != "FullSync" {
		return "properties.synchronizationMode must be Incremental or FullSync."
	}
	return ""
}

func azureDSHRShareSubscriptionValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["invitationId"])) == "" || strings.TrimSpace(azureARMString(props["sourceShareLocation"])) == "" {
		return "properties.invitationId and properties.sourceShareLocation are required."
	}
	return ""
}

// azureDSHRInvitationLocked finds the invitation with invitationId. Caller
// holds providerStorageMu.
func (s *Server) azureDSHRInvitationLocked(invitationID string) *azureARMResource {
	for _, rec := range s.azureARMResources {
		if strings.EqualFold(rec.Type, azureDSHRNamespace+"/accounts/shares/invitations") && strings.EqualFold(azureARMString(rec.Properties["invitationId"]), invitationID) {
			return rec
		}
	}
	return nil
}

// azureDSHRShareSubscriptionReconcile accepts the invitation the share
// subscription names and copies the source share's details.
func azureDSHRShareSubscriptionReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	invitationID := azureARMString(rec.Properties["invitationId"])
	invitation := s.azureDSHRInvitationLocked(invitationID)
	switch {
	case existing != nil && !strings.EqualFold(azureARMString(existing.Properties["invitationId"]), invitationID):
		return nil, "properties.invitationId can't change."
	case invitation == nil:
		return nil, "No invitation has invitationId '" + invitationID + "'."
	case existing == nil && invitation.Properties["invitationStatus"] != "Pending":
		return nil, "The invitation is " + azureARMString(invitation.Properties["invitationStatus"]) + "."
	}
	shareID := invitation.ID[:strings.LastIndex(strings.ToLower(invitation.ID), "/invitations/")]
	share := s.azureARMResources[azureARMKey(shareID)]
	if share == nil {
		return nil, "The invited share no longer exists."
	}
	s.azureDMSUpdateLocked(invitation.ID, map[string]any{"invitationStatus": "Accepted", "respondedAt": time.Now().UTC().Format(time.RFC3339)})
	props := map[string]any{
		"shareName": share.Name, "shareKind": share.Properties["shareKind"], "shareDescription": share.Properties["description"],
		"shareTerms": share.Properties["terms"], "providerName": "stackyard", "providerEmail": "stackyard@example.com", "providerTenantName": "Stackyard",
		"shareSubscriptionStatus": "Active", "shareSubscriptionObjectId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
		"createdAt": rec.CreatedAt.Format(time.RFC3339), "userName": "stackyard", "userEmail": "stackyard@example.com",
	}
	if existing != nil {
		props["shareSubscriptionStatus"] = existing.Properties["shareSubscriptionStatus"]
	}
	return props, ""
}

// azureDSHRSourceShareLocked returns the share a share subscription
// subscribes to, or nil. Caller holds providerStorageMu.
func (s *Server) azureDSHRSourceShareLocked(sub *azureARMResource) *azureARMResource {
	invitation := s.azureDSHRInvitationLocked(azureARMString(sub.Properties["invitationId"]))
	if invitation == nil {
		return nil
	}
	return s.azureARMResources[azureARMKey(invitation.ID[:strings.LastIndex(strings.ToLower(invitation.ID), "/invitations/")])]
}

// azureDSHRChildrenLocked returns records of type under parentID. Caller holds providerStorageMu.
func (s *Server) azureDSHRChildrenLocked(parentID, typ string) []*azureARMResource {
	prefix := azureARMKey(parentID + "/")
	return s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureDSHRNamespace+"/"+typ) && strings.HasPrefix(azureARMKey(rec.ID), prefix)
	})
}

func azureDSHRConsumerSourceDataSets(s *Server, scope string) []map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	sub := s.azureARMResources[azureARMKey(scope)]
	if sub == nil {
		return nil
	}
	share := s.azureDSHRSourceShareLocked(sub)
	if share == nil {
		return nil
	}
	var out []map[string]any
	for _, ds := range s.azureDSHRChildrenLocked(share.ID, "accounts/shares/dataSets") {
		out = append(out, map[string]any{
			"id": scope + "/consumerSourceDataSets/" + ds.Name, "name": ds.Name, "type": azureDSHRNamespace + "/accounts/shareSubscriptions/consumerSourceDataSets",
			"properties": map[string]any{
				"dataSetId": ds.Properties["dataSetId"], "dataSetName": ds.Name, "dataSetType": ds.Kind,
				"dataSetLocation": sub.Properties["sourceShareLocation"], "dataSetPath": azureARMString(ds.Properties["filePath"]),
			},
		})
	}
	return out
}

func (s *Server) azureDSHRProviderShareSubscriptionsLocked(shareID string) []map[string]any {
	var out []map[string]any
	for _, sub := range s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureDSHRNamespace+"/accounts/shareSubscriptions")
	}) {
		share := s.azureDSHRSourceShareLocked(sub)
		if share == nil || !strings.EqualFold(share.ID, shareID) {
			continue
		}
		objectID := azureARMString(sub.Properties["shareSubscriptionObjectId"])
		props := map[string]any{
			"consumerEmail": "stackyard@example.com", "consumerName": "stackyard", "consumerTenantName": "Stackyard",
			"createdAt": sub.CreatedAt.Format(time.RFC3339), "providerEmail": "stackyard@example.com", "providerName": "stackyard",
			"shareSubscriptionObjectId": objectID, "shareSubscriptionStatus": sub.Properties["shareSubscriptionStatus"],
			"sharedAt": sub.CreatedAt.Format(time.RFC3339),
		}
		if expiry, ok := sub.Properties["expirationDate"]; ok {
			props["expirationDate"] = expiry
		}
		out = append(out, map[string]any{
			"id": shareID + "/providerShareSubscriptions/" + objectID, "name": objectID,
			"type": azureDSHRNamespace + "/accounts/shares/providerShareSubscriptions", "properties": props,
		})
	}
	return out
}

func azureDSHRProviderShareSubscriptions(s *Server, scope string) []map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureDSHRProviderShareSubscriptionsLocked(scope)
}

// azureDSHRConsumerInvitations lists pending invitations as the consumer
// sees them; idPrefix, when set, scopes their IDs to a location.
func (s *Server) azureDSHRConsumerInvitations(idPrefix string) []map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	var out []map[string]any
	for _, inv := range s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureDSHRNamespace+"/accounts/shares/invitations") && rec.Properties["invitationStatus"] == "Pending"
	}) {
		id := azureARMString(inv.Properties["invitationId"])
		share := inv.ID[:strings.LastIndex(strings.ToLower(inv.ID), "/invitations/")]
		account := s.azureARMResources[azureARMKey(share[:strings.LastIndex(strings.ToLower(share), "/shares/")])]
		location := ""
		if account != nil {
			location = account.Location
		}
		prefix := idPrefix
		if prefix == "" {
			prefix = "/providers/" + azureDSHRNamespace + "/locations/" + location + "/consumerInvitations/"
		}
		out = append(out, map[string]any{
			"id": prefix + id, "name": inv.Name, "type": azureDSHRNamespace + "/consumerInvitations",
			"properties": map[string]any{
				"invitationId": id, "invitationStatus": "Pending", "location": location, "shareName": inv.Properties["shareName"],
				"sentAt": inv.Properties["sentAt"], "expirationDate": inv.Properties["expirationDate"], "providerName": "stackyard",
				"providerEmail": "stackyard@example.com", "providerTenantName": "Stackyard", "termsOfUse": "", "description": "",
				"dataSetCount": len(s.azureDSHRChildrenLocked(share, "accounts/shares/dataSets")), "userName": "stackyard", "userEmail": "stackyard@example.com",
			},
		})
	}
	return out
}

func azureDSHRSourceSettings(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	share := s.azureDSHRSourceShareLocked(rec)
	value := []any{}
	if share != nil {
		for _, setting := range s.azureDSHRChildrenLocked(share.ID, "accounts/shares/synchronizationSettings") {
			value = append(value, map[string]any{"kind": "ScheduleBased", "properties": map[string]any{
				"recurrenceInterval": setting.Properties["recurrenceInterval"], "synchronizationTime": setting.Properties["synchronizationTime"],
			}})
		}
	}
	return http.StatusOK, map[string]any{"value": value}
}

func azureDSHRMappingValidate(payload map[string]any) string {
	if !slices.Contains(azureDSHRDataSetKinds, azureARMString(payload["kind"])) {
		return "kind must be one of " + strings.Join(azureDSHRDataSetKinds, ", ") + "."
	}
	if strings.TrimSpace(azureARMString(azureARMObject(payload["properties"])["dataSetId"])) == "" {
		return "properties.dataSetId is required."
	}
	return azureEnumProperty("outputType", "Csv", "Parquet")(payload)
}

// azureDSHRMappingReconcile requires dataSetId to name a source data set.
func azureDSHRMappingReconcile(s *Server, _, rec *azureARMResource) (map[string]any, string) {
	sub := s.azureARMResources[azureARMKey(rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/datasetmappings/")])]
	if sub == nil {
		return nil, "The share subscription no longer exists."
	}
	share := s.azureDSHRSourceShareLocked(sub)
	if share != nil {
		for _, ds := range s.azureDSHRChildrenLocked(share.ID, "accounts/shares/dataSets") {
			if strings.EqualFold(azureARMString(ds.Properties["dataSetId"]), azureARMString(rec.Properties["dataSetId"])) {
				return map[string]any{"dataSetMappingStatus": "Ok"}, ""
			}
		}
	}
	return nil, "properties.dataSetId must name a data set of the source share."
}

func azureDSHRSynchronize(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	mode := azureARMString(payload["synchronizationMode"])
	if mode != "Incremental" && mode != "FullSync" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "synchronizationMode must be Incremental or FullSync.")
	}
	if rec.Properties["shareSubscriptionStatus"] != "Active" {
		return http.StatusConflict, azureARMErrorBody("Conflict", "The share subscription is "+azureARMString(rec.Properties["shareSubscriptionStatus"])+".")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sync := map[string]any{
		"synchronizationId": uuid.NewString(), "synchronizationMode": mode, "status": "Succeeded",
		"startTime": now, "endTime": now, "durationMs": 0, "message": "", "consumerEmail": "stackyard@example.com",
	}
	st := s.azureDSHRSyncState()
	st.mu.Lock()
	defer st.mu.Unlock()
	key := azureARMKey(rec.ID)
	syncs := append([]map[string]any{sync}, st.bySub[key]...)
	st.bySub[key] = syncs[:min(len(syncs), azureDSHRMaxSyncs)]
	return http.StatusOK, azureARMMerge(sync, nil)
}

func (st *azureDSHRSyncs) find(subID, syncID string) map[string]any {
	for _, sync := range st.bySub[azureARMKey(subID)] {
		if strings.EqualFold(azureARMString(sync["synchronizationId"]), syncID) {
			return sync
		}
	}
	return nil
}

// azureDSHRCancelSync returns the synchronization; finished ones stay as they are.
func azureDSHRCancelSync(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	st := s.azureDSHRSyncState()
	st.mu.Lock()
	defer st.mu.Unlock()
	sync := st.find(rec.ID, azureARMString(payload["synchronizationId"]))
	if sync == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "No synchronization has that synchronizationId.")
	}
	return http.StatusOK, azureARMMerge(sync, nil)
}

// azureDSHRShareSyncs lists synchronizations of one share subscription, or
// of every share subscription of a share.
func (s *Server) azureDSHRShareSyncs(shareID, subID string) (int, any) {
	subs := []string{subID}
	if shareID != "" {
		s.providerStorageMu.Lock()
		subs = nil
		for _, sub := range s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
			return strings.EqualFold(rec.Type, azureDSHRNamespace+"/accounts/shareSubscriptions")
		}) {
			if share := s.azureDSHRSourceShareLocked(sub); share != nil && strings.EqualFold(share.ID, shareID) {
				subs = append(subs, sub.ID)
			}
		}
		s.providerStorageMu.Unlock()
	}
	st := s.azureDSHRSyncState()
	st.mu.Lock()
	defer st.mu.Unlock()
	value := []any{}
	for _, id := range subs {
		for _, sync := range st.bySub[azureARMKey(id)] {
			value = append(value, azureARMMerge(sync, nil))
		}
	}
	return http.StatusOK, map[string]any{"value": value}
}

func (s *Server) azureDSHRSyncDetails(shareID string, sync map[string]any) map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	value := []any{}
	for _, ds := range s.azureDSHRChildrenLocked(shareID, "accounts/shares/dataSets") {
		value = append(value, map[string]any{
			"dataSetId": ds.Properties["dataSetId"], "dataSetType": ds.Kind, "name": ds.Name, "status": "Succeeded",
			"startTime": sync["startTime"], "endTime": sync["endTime"], "durationMs": 0, "filesRead": 0, "filesWritten": 0,
			"rowsCopied": 0, "rowsRead": 0, "sizeRead": 0, "sizeWritten": 0, "message": "",
		})
	}
	return map[string]any{"value": value}
}

func (s *Server) azureDSHRSubscriptionSyncDetails(rec *azureARMResource, payload map[string]any) (int, any) {
	st := s.azureDSHRSyncState()
	st.mu.Lock()
	sync := azureARMMerge(st.find(rec.ID, azureARMString(payload["synchronizationId"])), nil)
	st.mu.Unlock()
	if len(sync) == 0 {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "No synchronization has that synchronizationId.")
	}
	s.providerStorageMu.Lock()
	share := s.azureDSHRSourceShareLocked(rec)
	s.providerStorageMu.Unlock()
	if share == nil {
		return http.StatusOK, map[string]any{"value": []any{}}
	}
	return http.StatusOK, s.azureDSHRSyncDetails(share.ID, sync)
}

func (s *Server) azureDSHRShareSyncDetails(shareID string, payload map[string]any) (int, any) {
	syncID := azureARMString(payload["synchronizationId"])
	_, body := s.azureDSHRShareSyncs(shareID, "")
	for _, raw := range azureARMSlice(azureARMObject(body)["value"]) {
		if sync := azureARMObject(raw); strings.EqualFold(azureARMString(sync["synchronizationId"]), syncID) {
			return http.StatusOK, s.azureDSHRSyncDetails(shareID, sync)
		}
	}
	return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "No synchronization has that synchronizationId.")
}

// handleAzureDataShare20210801Router serves the provider share subscription
// actions and the consumer's location actions, which the shared ARM handler
// has no route for.
func (s *Server) handleAzureDataShare20210801Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimPrefix(rawRequestPath(r), "/azure")
	provider := azureDSHRProviderRouteRE.FindStringSubmatch(path)
	location := azureDSHRLocationRouteRE.FindStringSubmatch(path)
	if provider == nil && location == nil {
		return azureDMSUnknownRoute(w, r, azureDSHRNamespace)
	}
	s.serveAzureARMPost(w, r, func(payload map[string]any) (int, any) {
		if location != nil {
			return s.azureDSHRLocationAction(location[1], strings.ToLower(location[2]), payload)
		}
		return s.azureDSHRProviderAction(provider[1], provider[2], strings.ToLower(provider[3]), payload)
	})
	return true
}

func (s *Server) azureDSHRProviderAction(shareID, objectID, action string, payload map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	var target *azureARMResource
	for _, sub := range s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureDSHRNamespace+"/accounts/shareSubscriptions") && strings.EqualFold(azureARMString(rec.Properties["shareSubscriptionObjectId"]), objectID)
	}) {
		if share := s.azureDSHRSourceShareLocked(sub); share != nil && strings.EqualFold(share.ID, shareID) {
			target = sub
		}
	}
	if target == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The provider share subscription '"+objectID+"' was not found.")
	}
	update := map[string]any{}
	switch action {
	case "revoke":
		update["shareSubscriptionStatus"] = "Revoked"
	case "reinstate":
		if target.Properties["shareSubscriptionStatus"] != "Revoked" {
			return http.StatusConflict, azureARMErrorBody("Conflict", "Only a revoked share subscription can be reinstated.")
		}
		update["shareSubscriptionStatus"] = "Active"
	}
	if expiry, present := azureARMObject(payload["properties"])["expirationDate"]; present {
		if _, err := time.Parse(time.RFC3339, azureARMString(expiry)); err != nil {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.expirationDate must be an RFC 3339 timestamp.")
		}
		update["expirationDate"] = expiry
	}
	s.azureDMSUpdateLocked(target.ID, update)
	for _, item := range s.azureDSHRProviderShareSubscriptionsLocked(shareID) {
		if item["name"] == objectID {
			return http.StatusOK, item
		}
	}
	return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The provider share subscription '"+objectID+"' was not found.")
}

func (s *Server) azureDSHRLocationAction(location, action string, payload map[string]any) (int, any) {
	switch action {
	case "rejectinvitation":
		invitationID := azureARMString(azureARMObject(payload["properties"])["invitationId"])
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		invitation := s.azureDSHRInvitationLocked(invitationID)
		if invitation == nil || invitation.Properties["invitationStatus"] != "Pending" {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "No pending invitation has invitationId '"+invitationID+"'.")
		}
		updated := s.azureDMSUpdateLocked(invitation.ID, map[string]any{"invitationStatus": "Rejected", "respondedAt": time.Now().UTC().Format(time.RFC3339)})
		return http.StatusOK, map[string]any{
			"id": "/providers/" + azureDSHRNamespace + "/locations/" + location + "/consumerInvitations/" + invitationID, "name": updated.Name,
			"type":       azureDSHRNamespace + "/consumerInvitations",
			"properties": map[string]any{"invitationId": invitationID, "invitationStatus": "Rejected", "location": location, "respondedAt": updated.Properties["respondedAt"]},
		}
	case "registeremail", "activateemail":
		status := "ActivationPending"
		if action == "activateemail" {
			if strings.TrimSpace(azureARMString(payload["activationCode"])) == "" {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "activationCode is required.")
			}
			status = "Activated"
		}
		return http.StatusOK, map[string]any{"email": "stackyard@example.com", "location": location, "registrationStatus": status, "tenantId": "00000000-0000-0000-0000-000000000000"}
	}
	return http.StatusNotFound, azureARMErrorBody("InvalidResourceType", "unknown action")
}
