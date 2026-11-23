package server

type servicecatalogappregistryOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Service Catalog AppRegistry operations sourced from:
// https://docs.aws.amazon.com/servicecatalog/latest/dg/API_Operations_AWS_Service_Catalog_App_Registry.html
var servicecatalogappregistryOperations = []servicecatalogappregistryOperation{
	{Name: "AssociateAttributeGroup", Method: "PUT", URI: "/applications/{application}/attribute-groups/{attributeGroup}"},
	{Name: "AssociateResource", Method: "PUT", URI: "/applications/{application}/resources/{resourceType}/{resource}"},
	{Name: "CreateApplication", Method: "POST", URI: "/applications"},
	{Name: "CreateAttributeGroup", Method: "POST", URI: "/attribute-groups"},
	{Name: "DeleteApplication", Method: "DELETE", URI: "/applications/{application}"},
	{Name: "DeleteAttributeGroup", Method: "DELETE", URI: "/attribute-groups/{attributeGroup}"},
	{Name: "DisassociateAttributeGroup", Method: "DELETE", URI: "/applications/{application}/attribute-groups/{attributeGroup}"},
	{Name: "DisassociateResource", Method: "DELETE", URI: "/applications/{application}/resources/{resourceType}/{resource}"},
	{Name: "GetApplication", Method: "GET", URI: "/applications/{application}"},
	{Name: "GetAssociatedResource", Method: "GET", URI: "/applications/{application}/resources/{resourceType}/{resource}"},
	{Name: "GetAttributeGroup", Method: "GET", URI: "/attribute-groups/{attributeGroup}"},
	{Name: "GetConfiguration", Method: "GET", URI: "/configuration"},
	{Name: "ListApplications", Method: "GET", URI: "/applications"},
	{Name: "ListAssociatedAttributeGroups", Method: "GET", URI: "/applications/{application}/attribute-groups"},
	{Name: "ListAssociatedResources", Method: "GET", URI: "/applications/{application}/resources"},
	{Name: "ListAttributeGroups", Method: "GET", URI: "/attribute-groups"},
	{Name: "ListAttributeGroupsForApplication", Method: "GET", URI: "/applications/{application}/attribute-group-details"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PutConfiguration", Method: "PUT", URI: "/configuration"},
	{Name: "SyncResource", Method: "POST", URI: "/sync/{resourceType}/{resource}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateApplication", Method: "PATCH", URI: "/applications/{application}"},
	{Name: "UpdateAttributeGroup", Method: "PATCH", URI: "/attribute-groups/{attributeGroup}"},
}

var servicecatalogappregistryOperationByName = func() map[string]servicecatalogappregistryOperation {
	out := make(map[string]servicecatalogappregistryOperation, len(servicecatalogappregistryOperations))
	for _, op := range servicecatalogappregistryOperations {
		out[op.Name] = op
	}
	return out
}()
