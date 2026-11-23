package server

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/stackyard/stackyard/internal/services/sesv2"
)

func sesv2TenantSummary(t sesv2.Tenant) map[string]any {
	return map[string]any{
		"TenantName":       t.Name,
		"TenantId":         t.ID,
		"TenantArn":        t.ARN,
		"CreatedTimestamp": sesv2Timestamp(t.CreatedAt),
		"SendingStatus":    "ENABLED",
	}
}

func sesv2TenantDetail(t sesv2.Tenant) map[string]any {
	out := sesv2TenantSummary(t)
	out["Tags"] = sesv2MapToTags(t.Tags)
	if len(t.SuppressedReasons) > 0 || t.SuppressionScope != "" {
		attrs := map[string]any{"SuppressedReasons": t.SuppressedReasons}
		if t.SuppressionScope != "" {
			attrs["SuppressionScope"] = t.SuppressionScope
		}
		out["SuppressionAttributes"] = attrs
	}
	return out
}

// handleSESV2Tenant serves the SES tenant operations, all of which take a JSON
// body.
func (s *Server) handleSESV2Tenant(w http.ResponseWriter, op string, payload map[string]any) {
	name := sesv2String(payload["TenantName"])
	resourceArn := sesv2String(payload["ResourceArn"])
	switch op {
	case "CreateTenant":
		attrs := sesv2Map(payload["SuppressionAttributes"])
		t, err := s.sesv2.CreateTenant(name, sesv2TagsToMap(payload["Tags"]), sesv2StringSlice(attrs["SuppressedReasons"]), sesv2String(attrs["SuppressionScope"]))
		if err != nil {
			respondSESV2ErrorForErr(w, err)
			return
		}
		respondSESV2JSON(w, http.StatusOK, sesv2TenantDetail(t))
	case "GetTenant":
		t, ok := s.sesv2.GetTenant(name)
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "tenant not found")
			return
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"Tenant": sesv2TenantDetail(t)})
	case "ListTenants":
		tenants := s.sesv2.ListTenants()
		items := make([]map[string]any, 0, len(tenants))
		for _, t := range tenants {
			items = append(items, sesv2TenantSummary(t))
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"Tenants": items})
	case "DeleteTenant":
		if !s.sesv2.DeleteTenant(name) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "tenant not found")
			return
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
	case "CreateTenantResourceAssociation", "DeleteTenantResourceAssociation":
		var err error
		if op == "CreateTenantResourceAssociation" {
			err = s.sesv2.AssociateTenantResource(name, resourceArn)
		} else {
			err = s.sesv2.DisassociateTenantResource(name, resourceArn)
		}
		if err != nil {
			respondSESV2ErrorForErr(w, err)
			return
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
	case "ListTenantResources":
		resources, ok := s.sesv2.ListTenantResources(name, sesv2String(sesv2Map(payload["Filter"])["RESOURCE_TYPE"]))
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "tenant not found")
			return
		}
		items := make([]map[string]any, 0, len(resources))
		for _, res := range resources {
			items = append(items, map[string]any{"ResourceArn": res.ResourceArn, "ResourceType": res.ResourceType})
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"TenantResources": items})
	case "ListResourceTenants":
		if resourceArn == "" {
			respondSESV2Error(w, http.StatusBadRequest, "BadRequestException", "ResourceArn is required")
			return
		}
		tenants := s.sesv2.ListResourceTenants(resourceArn)
		items := make([]map[string]any, 0, len(tenants))
		for _, t := range tenants {
			items = append(items, map[string]any{
				"TenantName":          t.TenantName,
				"TenantId":            t.TenantID,
				"ResourceArn":         t.ResourceArn,
				"AssociatedTimestamp": sesv2Timestamp(t.AssociatedAt),
			})
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"ResourceTenants": items})
	}
}

func sesv2MultiRegionEndpointView(e sesv2.MultiRegionEndpoint) map[string]any {
	return map[string]any{
		"EndpointName":         e.Name,
		"EndpointId":           e.ID,
		"Status":               "READY",
		"CreatedTimestamp":     sesv2Timestamp(e.CreatedAt),
		"LastUpdatedTimestamp": sesv2Timestamp(e.CreatedAt),
	}
}

func (s *Server) handleSESV2MultiRegionEndpoint(w http.ResponseWriter, op string, params map[string]string, payload map[string]any) {
	switch op {
	case "CreateMultiRegionEndpoint":
		var regions []string
		for _, route := range sesv2List(sesv2Map(payload["Details"])["RoutesDetails"]) {
			if region := sesv2String(sesv2Map(route)["Region"]); region != "" {
				regions = append(regions, region)
			}
		}
		e, err := s.sesv2.CreateMultiRegionEndpoint(sesv2String(payload["EndpointName"]), regions, sesv2TagsToMap(payload["Tags"]))
		if err != nil {
			respondSESV2ErrorForErr(w, err)
			return
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"Status": "READY", "EndpointId": e.ID})
	case "GetMultiRegionEndpoint":
		e, ok := s.sesv2.GetMultiRegionEndpoint(params["EndpointName"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "multi-region endpoint not found")
			return
		}
		view := sesv2MultiRegionEndpointView(e)
		routes := make([]map[string]any, 0, len(e.Regions))
		for _, region := range e.Regions {
			routes = append(routes, map[string]any{"Region": region})
		}
		view["Routes"] = routes
		respondSESV2JSON(w, http.StatusOK, view)
	case "DeleteMultiRegionEndpoint":
		if !s.sesv2.DeleteMultiRegionEndpoint(params["EndpointName"]) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "multi-region endpoint not found")
			return
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"Status": "DELETING"})
	case "ListMultiRegionEndpoints":
		endpoints := s.sesv2.ListMultiRegionEndpoints()
		items := make([]map[string]any, 0, len(endpoints))
		for _, e := range endpoints {
			view := sesv2MultiRegionEndpointView(e)
			view["Regions"] = e.Regions
			items = append(items, view)
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"MultiRegionEndpoints": items})
	}
}

// sesv2MailboxValidation derives GetEmailAddressInsights verdicts locally: real
// syntax and role-address checks; DNS and mailbox existence are assumed.
func sesv2MailboxValidation(address string) map[string]any {
	verdict := func(ok bool) map[string]any {
		if ok {
			return map[string]any{"ConfidenceVerdict": "HIGH"}
		}
		return map[string]any{"ConfidenceVerdict": "LOW"}
	}
	parsed, err := mail.ParseAddress(address)
	valid := err == nil && parsed.Address == address && strings.Contains(address, "@")
	local := strings.ToLower(strings.SplitN(address, "@", 2)[0])
	role := false
	switch local {
	case "admin", "administrator", "info", "support", "sales", "postmaster", "abuse", "noreply", "no-reply", "webmaster", "hostmaster":
		role = true
	}
	return map[string]any{
		"IsValid": verdict(valid),
		"Evaluations": map[string]any{
			"HasValidSyntax":     verdict(valid),
			"HasValidDnsRecords": verdict(valid),
			"MailboxExists":      verdict(valid),
			"IsRoleAddress":      verdict(role),
			"IsDisposable":       verdict(false),
			"IsRandomInput":      verdict(false),
		},
	}
}
