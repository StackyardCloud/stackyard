package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.Support tickets with their communications and chat transcripts,
// file workspaces with their files, and the services and problem
// classifications tickets are filed against, at subscription and tenant
// scope, served by the shared ARM handler. No engineer ever answers: a
// ticket stays open until the client closes it, and chats are empty.
func init() {
	const ns = "Microsoft.Support"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "supportTickets", Tenant: true, Subscription: true,
		// The API answers a create with 200 (or 202), never 201.
		CreateStatus:       http.StatusOK,
		Validate:           azureSupportTicketValidate,
		ReadOnlyProperties: azureSupportTicketProperties,
		// Only an update closes or reopens a ticket.
		StateProperties: map[string]any{"status": "Open"},
		PatchBody:       azureSupportTicketPatch,
		ServerActions:   map[string]func(*Server, *azureARMResource, map[string]any) (int, any){"checknameavailability": azureSupportCommunicationNameCheck},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "supportTickets/communications", Tenant: true, Subscription: true, NoPatch: true,
		CreateStatus:     http.StatusOK,
		Validate:         azureSupportCommunicationValidate,
		ValidateInParent: azureSupportCommunicationInTicket,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"communicationType": "web", "communicationDirection": "inbound", "createdDate": rec.CreatedAt.Format(time.RFC3339)}
		},
	})
	registerAzureARMView(ns, "supportTickets/chatTranscripts", func(*Server, string) []map[string]any { return nil })
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "fileWorkspaces", Tenant: true, Subscription: true, NoPatch: true,
		ValidateName: func(name string) string {
			if !azureSupportWorkspaceName.MatchString(name) {
				return "File workspace names may contain only letters, digits, spaces, underscores, hyphens and periods."
			}
			return ""
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"createdOn": rec.CreatedAt.Format(time.RFC3339), "expirationTime": rec.CreatedAt.AddDate(0, 0, 30).Format(time.RFC3339)}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "fileWorkspaces/files", Tenant: true, Subscription: true, NoPatch: true,
		Validate: azureSupportFileValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"createdOn": rec.CreatedAt.Format(time.RFC3339)}
		},
		Actions: map[string]azureARMAction{"upload": azureSupportFileUpload},
	})
	registerAzureARMView(ns, "services", azureSupportServices)
	registerAzureARMView(ns, "services/problemClassifications", azureSupportProblemClassifications)
	registerAzureARMProviderAction(ns, "checkNameAvailability", azureSupportTicketNameCheck)
}

// Support file limits: 5 MB per file in at most two chunks of 2.5 MB.
const (
	azureSupportMaxFileSize  = 5_000_000
	azureSupportMaxChunkSize = 2_500_000
	azureSupportMaxChunks    = 2
)

var (
	azureSupportWorkspaceName = regexp.MustCompile(`^[0-9a-zA-Z_\-. ]+$`)
	azureSupportEmail         = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	// azureSupportSLAMinutes is the first-response time per severity.
	azureSupportSLAMinutes = map[string]int{"minimal": 480, "moderate": 240, "critical": 60, "highestcriticalimpact": 60}
)

// azureSupportCatalog is the services tickets can be filed against, each
// with its problem classifications.
// ponytail: a few common services; add others when a test needs one.
var azureSupportCatalog = []struct {
	ID, Name      string
	ResourceTypes []any
	Problems      []string
}{
	{"06bfd9d3-516b-d5c6-5802-169c800dec89", "Service and subscription limits (quotas)", []any{}, []string{"Compute-VM (cores-vCPUs) subscription limit increases", "Networking", "Storage: Azure Storage"}},
	{"517f2da6-78fd-0498-4e22-ad26996b1dfc", "Billing", []any{}, []string{"Refund request", "Invoice or billing statement"}},
	{"6f16735c-b0ae-b275-ad3a-03479cfa1396", "Virtual Machine running Windows", []any{"MICROSOFT.CLASSICCOMPUTE/VIRTUALMACHINES", "MICROSOFT.COMPUTE/VIRTUALMACHINES"}, []string{"Cannot connect to my VM", "VM performance"}},
	{"cddd3eb5-1830-b494-44fd-782f691479dc", "Virtual Machine running Linux", []any{"MICROSOFT.CLASSICCOMPUTE/VIRTUALMACHINES", "MICROSOFT.COMPUTE/VIRTUALMACHINES"}, []string{"Cannot connect to my VM", "VM performance"}},
	{"aa9daeb0-1e43-5f68-f3f7-6c22c6ed97bd", "Storage Account Management", []any{"MICROSOFT.STORAGE/STORAGEACCOUNTS"}, []string{"Storage account access", "Create or delete a storage account"}},
}

// azureSupportProblemID is a stable problem classification GUID.
func azureSupportProblemID(service, problem string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(service+"#"+problem)).String()
}

func azureSupportServices(_ *Server, scope string) []map[string]any {
	if scope != "" {
		return nil
	}
	items := make([]map[string]any, 0, len(azureSupportCatalog))
	for _, svc := range azureSupportCatalog {
		items = append(items, map[string]any{
			"id": "/providers/Microsoft.Support/services/" + svc.ID, "name": svc.ID, "type": "Microsoft.Support/services",
			"properties": map[string]any{"displayName": svc.Name, "resourceTypes": svc.ResourceTypes},
		})
	}
	return items
}

func azureSupportProblemClassifications(_ *Server, scope string) []map[string]any {
	for _, svc := range azureSupportCatalog {
		if !strings.EqualFold(scope, "/providers/Microsoft.Support/services/"+svc.ID) {
			continue
		}
		items := make([]map[string]any, 0, len(svc.Problems))
		for _, problem := range svc.Problems {
			id := azureSupportProblemID(svc.ID, problem)
			items = append(items, map[string]any{
				"id": scope + "/problemClassifications/" + id, "name": id, "type": "Microsoft.Support/problemClassifications",
				"properties": map[string]any{"displayName": problem, "secondaryConsentEnabled": []any{}},
			})
		}
		return items
	}
	return nil
}

// azureSupportDisplayNames returns the catalog names of a ticket's service
// and problem classification, if known.
func azureSupportDisplayNames(serviceID, problemID string) (service, problem string) {
	for _, svc := range azureSupportCatalog {
		if !strings.EqualFold(serviceID, "/providers/Microsoft.Support/services/"+svc.ID) {
			continue
		}
		for _, name := range svc.Problems {
			if strings.EqualFold(problemID, serviceID+"/problemClassifications/"+azureSupportProblemID(svc.ID, name)) {
				return svc.Name, name
			}
		}
		return svc.Name, ""
	}
	return "", ""
}

func azureSupportTicketValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"title", "description"} {
		if strings.TrimSpace(azureARMString(props[key])) == "" {
			return "properties." + key + " is required."
		}
	}
	service := azureARMString(props["serviceId"])
	if _, err := uuid.Parse(strings.TrimPrefix(strings.ToLower(service), "/providers/microsoft.support/services/")); err != nil || !strings.HasPrefix(strings.ToLower(service), "/providers/microsoft.support/services/") {
		return "properties.serviceId must be a support service ID, such as /providers/Microsoft.Support/services/{guid}."
	}
	problem := strings.ToLower(azureARMString(props["problemClassificationId"]))
	if _, err := uuid.Parse(strings.TrimPrefix(problem, strings.ToLower(service)+"/problemclassifications/")); err != nil || !strings.HasPrefix(problem, strings.ToLower(service)+"/problemclassifications/") {
		return "properties.problemClassificationId must be a problem classification of the ticket's service."
	}
	for _, check := range []func(map[string]any) string{
		azureEnumRequired("severity", "minimal", "moderate", "critical", "highestcriticalimpact"),
		azureEnumRequired("advancedDiagnosticConsent", "Yes", "No"),
	} {
		if msg := check(payload); msg != "" {
			return msg
		}
	}
	contact, ok := props["contactDetails"].(map[string]any)
	if !ok {
		return "properties.contactDetails is required."
	}
	for _, key := range []string{"firstName", "lastName", "preferredTimeZone", "country", "preferredSupportLanguage", "primaryEmailAddress", "preferredContactMethod"} {
		if strings.TrimSpace(azureARMString(contact[key])) == "" {
			return "properties.contactDetails." + key + " is required."
		}
	}
	if msg := azureSupportContactValidate(contact); msg != "" {
		return msg
	}
	severity := props["severity"]
	if (contact["preferredContactMethod"] == "phone" || severity == "critical" || severity == "highestcriticalimpact") && strings.TrimSpace(azureARMString(contact["phoneNumber"])) == "" {
		return "properties.contactDetails.phoneNumber is required for phone contact and critical or highestcriticalimpact severity."
	}
	if raw, present := props["problemStartTime"]; present {
		if _, err := time.Parse(time.RFC3339, azureARMString(raw)); err != nil {
			return "properties.problemStartTime must be an RFC 3339 date-time."
		}
	}
	for i, raw := range azureARMSlice(props["secondaryConsent"]) {
		if consent := azureARMObject(raw)["userConsent"]; consent != "Yes" && consent != "No" {
			return fmt.Sprintf("properties.secondaryConsent[%d].userConsent must be Yes or No.", i)
		}
	}
	return ""
}

// azureSupportContactValidate checks the email address and contact method
// of contact details that set them.
func azureSupportContactValidate(contact map[string]any) string {
	if raw, present := contact["primaryEmailAddress"]; present && !azureSupportEmail.MatchString(azureARMString(raw)) {
		return "contactDetails.primaryEmailAddress must be an email address."
	}
	if raw, present := contact["preferredContactMethod"]; present && raw != "email" && raw != "phone" {
		return "contactDetails.preferredContactMethod must be email or phone."
	}
	return ""
}

// azureSupportTicketPatch takes the update model's top-level fields into
// properties, checks them, and turns status into the ticket's state.
func azureSupportTicketPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	props := map[string]any{}
	for _, key := range []string{"severity", "contactDetails", "advancedDiagnosticConsent", "secondaryConsent"} {
		if value, present := payload[key]; present {
			props[key] = value
		}
	}
	checks := map[string][]any{"severity": {"minimal", "moderate", "critical", "highestcriticalimpact"}, "advancedDiagnosticConsent": {"Yes", "No"}}
	for key, values := range checks {
		if value, present := props[key]; present && !slices.Contains(values, value) {
			return nil, nil, key + " is not a valid value."
		}
	}
	if msg := azureSupportContactValidate(azureARMObject(props["contactDetails"])); msg != "" {
		return nil, nil, msg
	}
	var state map[string]any
	switch payload["status"] {
	case nil:
	case "open":
		state = map[string]any{"status": "Open"}
	case "closed":
		state = map[string]any{"status": "Closed"}
	default:
		return nil, nil, "status must be open or closed."
	}
	return map[string]any{"properties": props}, state, ""
}

// azureSupportTicketProperties reports a stable ticket number, the SLA
// for its severity, and display names from the catalog.
func azureSupportTicketProperties(rec *azureARMResource) map[string]any {
	sum := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)))
	minutes := azureSupportSLAMinutes[azureARMString(rec.Properties["severity"])]
	props := map[string]any{
		"supportPlanType":        "Standard",
		"supportPlanDisplayName": "Standard",
		"isTemporaryTicket":      "No",
		"createdDate":            rec.CreatedAt.Format(time.RFC3339),
		"modifiedDate":           rec.UpdatedAt.Format(time.RFC3339),
		"serviceLevelAgreement": map[string]any{
			"startTime": rec.CreatedAt.Format(time.RFC3339), "expirationTime": rec.CreatedAt.Add(time.Duration(minutes) * time.Minute).Format(time.RFC3339), "slaMinutes": minutes,
		},
	}
	if _, present := rec.Properties["supportTicketId"]; !present {
		props["supportTicketId"] = fmt.Sprintf("%s%010d", rec.CreatedAt.Format("060102"), uint64(sum[0])<<24|uint64(sum[1])<<16|uint64(sum[2])<<8|uint64(sum[3]))
	}
	service, problem := azureSupportDisplayNames(azureARMString(rec.Properties["serviceId"]), azureARMString(rec.Properties["problemClassificationId"]))
	if service != "" {
		props["serviceDisplayName"] = service
	}
	if problem != "" {
		props["problemClassificationDisplayName"] = problem
	}
	return props
}

func azureSupportCommunicationValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"subject", "body"} {
		if strings.TrimSpace(azureARMString(props[key])) == "" {
			return "properties." + key + " is required."
		}
	}
	if raw, present := props["sender"]; present && !azureSupportEmail.MatchString(azureARMString(raw)) {
		return "properties.sender must be an email address."
	}
	return ""
}

// azureSupportCommunicationInTicket refuses messages on a closed ticket.
func azureSupportCommunicationInTicket(ticket map[string]any, _ string, _ map[string]any) string {
	if strings.EqualFold(azureARMString(ticket["status"]), "closed") {
		return "Communications can't be added to a closed support ticket."
	}
	return ""
}

func azureSupportFileValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch {
	case !azureWholeNumber(props["fileSize"], 1, azureSupportMaxFileSize):
		return fmt.Sprintf("properties.fileSize must be an integer from 1 to %d bytes.", azureSupportMaxFileSize)
	case !azureWholeNumber(props["chunkSize"], 1, azureSupportMaxChunkSize):
		return fmt.Sprintf("properties.chunkSize must be an integer from 1 to %d bytes.", azureSupportMaxChunkSize)
	case !azureWholeNumber(props["numberOfChunks"], 1, azureSupportMaxChunks):
		return fmt.Sprintf("properties.numberOfChunks must be 1 to %d.", azureSupportMaxChunks)
	}
	return ""
}

// azureSupportFileUpload accepts one base64 chunk of a file.
func azureSupportFileUpload(rec *azureARMResource, payload map[string]any) (int, any) {
	chunks, _ := rec.Properties["numberOfChunks"].(float64)
	if !azureWholeNumber(payload["chunkIndex"], 0, chunks-1) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "chunkIndex must be an integer from 0 to numberOfChunks - 1.")
	}
	content, err := base64.StdEncoding.DecodeString(azureARMString(payload["content"]))
	if err != nil || len(content) == 0 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "content must be base64.")
	}
	if size, _ := rec.Properties["chunkSize"].(float64); float64(len(content)) > size {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "The chunk is larger than the file's chunkSize.")
	}
	return http.StatusNoContent, nil
}

// azureSupportNameResult reports whether name is free, given whether a
// resource of the type already uses it.
func azureSupportNameResult(taken bool) map[string]any {
	if taken {
		return map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The name is already in use."}
	}
	return map[string]any{"nameAvailable": true}
}

// azureSupportTicketNameCheck checks a ticket name across all scopes.
func azureSupportTicketNameCheck(s *Server, payload map[string]any) (int, any) {
	name := azureARMString(payload["name"])
	if name == "" || payload["type"] != "Microsoft.Support/supportTickets" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name and type Microsoft.Support/supportTickets are required.")
	}
	return http.StatusOK, azureSupportNameResult(len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.Support/supportTickets") && strings.EqualFold(rec.Name, name)
	})) > 0)
}

// azureSupportCommunicationNameCheck checks a communication name within a
// ticket.
func azureSupportCommunicationNameCheck(s *Server, ticket *azureARMResource, payload map[string]any) (int, any) {
	name := azureARMString(payload["name"])
	if name == "" || payload["type"] != "Microsoft.Support/communications" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name and type Microsoft.Support/communications are required.")
	}
	id := ticket.ID + "/communications/" + name
	return http.StatusOK, azureSupportNameResult(len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.ID, id)
	})) > 0)
}
