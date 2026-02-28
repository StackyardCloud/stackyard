package server

type rolesAnywhereOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS IAM Roles Anywhere operations sourced from:
// https://docs.aws.amazon.com/rolesanywhere/latest/APIReference/API_Operations.html
var rolesAnywhereOperations = []rolesAnywhereOperation{
	{Name: "CreateProfile", Method: "POST", URI: "/profiles"},
	{Name: "CreateTrustAnchor", Method: "POST", URI: "/trustanchors"},
	{Name: "DeleteAttributeMapping", Method: "DELETE", URI: "/profiles/{profileId}/mappings?certificateField={certificateField}&specifiers={specifiers}"},
	{Name: "DeleteCrl", Method: "DELETE", URI: "/crl/{crlId}"},
	{Name: "DeleteProfile", Method: "DELETE", URI: "/profile/{profileId}"},
	{Name: "DeleteTrustAnchor", Method: "DELETE", URI: "/trustanchor/{trustAnchorId}"},
	{Name: "DisableCrl", Method: "POST", URI: "/crl/{crlId}/disable"},
	{Name: "DisableProfile", Method: "POST", URI: "/profile/{profileId}/disable"},
	{Name: "DisableTrustAnchor", Method: "POST", URI: "/trustanchor/{trustAnchorId}/disable"},
	{Name: "EnableCrl", Method: "POST", URI: "/crl/{crlId}/enable"},
	{Name: "EnableProfile", Method: "POST", URI: "/profile/{profileId}/enable"},
	{Name: "EnableTrustAnchor", Method: "POST", URI: "/trustanchor/{trustAnchorId}/enable"},
	{Name: "GetCrl", Method: "GET", URI: "/crl/{crlId}"},
	{Name: "GetProfile", Method: "GET", URI: "/profile/{profileId}"},
	{Name: "GetSubject", Method: "GET", URI: "/subject/{subjectId}"},
	{Name: "GetTrustAnchor", Method: "GET", URI: "/trustanchor/{trustAnchorId}"},
	{Name: "ImportCrl", Method: "POST", URI: "/crls"},
	{Name: "ListCrls", Method: "GET", URI: "/crls?nextToken={nextToken}&pageSize={pageSize}"},
	{Name: "ListProfiles", Method: "GET", URI: "/profiles?nextToken={nextToken}&pageSize={pageSize}"},
	{Name: "ListSubjects", Method: "GET", URI: "/subjects?nextToken={nextToken}&pageSize={pageSize}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/ListTagsForResource?resourceArn={resourceArn}"},
	{Name: "ListTrustAnchors", Method: "GET", URI: "/trustanchors?nextToken={nextToken}&pageSize={pageSize}"},
	{Name: "PutAttributeMapping", Method: "PUT", URI: "/profiles/{profileId}/mappings"},
	{Name: "PutNotificationSettings", Method: "PATCH", URI: "/put-notifications-settings"},
	{Name: "ResetNotificationSettings", Method: "PATCH", URI: "/reset-notifications-settings"},
	{Name: "TagResource", Method: "POST", URI: "/TagResource"},
	{Name: "UntagResource", Method: "POST", URI: "/UntagResource"},
	{Name: "UpdateCrl", Method: "PATCH", URI: "/crl/{crlId}"},
	{Name: "UpdateProfile", Method: "PATCH", URI: "/profile/{profileId}"},
	{Name: "UpdateTrustAnchor", Method: "PATCH", URI: "/trustanchor/{trustAnchorId}"},
}

var rolesAnywhereOperationByName = func() map[string]rolesAnywhereOperation {
	out := make(map[string]rolesAnywhereOperation, len(rolesAnywhereOperations))
	for _, op := range rolesAnywhereOperations {
		out[op.Name] = op
	}
	return out
}()
