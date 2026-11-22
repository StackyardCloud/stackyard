package server

type route53profilesOperation struct {
	Name   string
	Method string
	URI    string
}

// Route 53 Profiles operations sourced from:
// https://docs.aws.amazon.com/Route53/latest/APIReference/API_Operations_Route_53_Profiles.html
var route53profilesOperations = []route53profilesOperation{
	{Name: "AssociateProfile", Method: "POST", URI: "/profileassociation"},
	{Name: "AssociateResourceToProfile", Method: "POST", URI: "/profileresourceassociation"},
	{Name: "CreateProfile", Method: "POST", URI: "/profile"},
	{Name: "DeleteProfile", Method: "DELETE", URI: "/profile/{ProfileId}"},
	{Name: "DisassociateProfile", Method: "DELETE", URI: "/profileassociation/Profileid/{ProfileId}/resourceid/{ResourceId}"},
	{Name: "DisassociateResourceFromProfile", Method: "DELETE", URI: "/profileresourceassociation/profileid/{ProfileId}/resourcearn/{ResourceArn}"},
	{Name: "GetProfile", Method: "GET", URI: "/profile/{ProfileId}"},
	{Name: "GetProfileAssociation", Method: "GET", URI: "/profileassociation/{ProfileAssociationId}"},
	{Name: "GetProfileResourceAssociation", Method: "GET", URI: "/profileresourceassociation/{ProfileResourceAssociationId}"},
	{Name: "ListProfileAssociations", Method: "GET", URI: "/profileassociations"},
	{Name: "ListProfileResourceAssociations", Method: "GET", URI: "/profileresourceassociations/profileid/{ProfileId}"},
	{Name: "ListProfiles", Method: "GET", URI: "/profiles"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateProfileAssociation", Method: "PATCH", URI: "/profileassociation/{ProfileAssociationId}"},
	{Name: "UpdateProfileResourceAssociation", Method: "PATCH", URI: "/profileresourceassociation/{ProfileResourceAssociationId}"},
}

var route53profilesOperationByName = func() map[string]route53profilesOperation {
	out := make(map[string]route53profilesOperation, len(route53profilesOperations))
	for _, op := range route53profilesOperations {
		out[op.Name] = op
	}
	return out
}()
