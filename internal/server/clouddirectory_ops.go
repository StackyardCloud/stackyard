package server

type cloudDirectoryOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Cloud Directory operations sourced from:

// https://docs.aws.amazon.com/clouddirectory/latest/APIReference/API_Operations.html

var cloudDirectoryOperations = []cloudDirectoryOperation{
	{Name: "AddFacetToObject", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/object/facets"},
	{Name: "ApplySchema", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/schema/apply"},
	{Name: "AttachObject", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/object/attach"},
	{Name: "AttachPolicy", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/policy/attach"},
	{Name: "AttachToIndex", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/index/attach"},
	{Name: "AttachTypedLink", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/typedlink/attach"},
	{Name: "BatchRead", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/batchread"},
	{Name: "BatchWrite", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/batchwrite"},
	{Name: "CreateDirectory", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/directory/create"},
	{Name: "CreateFacet", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/facet/create"},
	{Name: "CreateIndex", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/index"},
	{Name: "CreateObject", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/object"},
	{Name: "CreateSchema", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/schema/create"},
	{Name: "CreateTypedLinkFacet", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/typedlink/facet/create"},
	{Name: "DeleteDirectory", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/directory"},
	{Name: "DeleteFacet", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/facet/delete"},
	{Name: "DeleteObject", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/object/delete"},
	{Name: "DeleteSchema", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/schema"},
	{Name: "DeleteTypedLinkFacet", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/typedlink/facet/delete"},
	{Name: "DetachFromIndex", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/index/detach"},
	{Name: "DetachObject", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/object/detach"},
	{Name: "DetachPolicy", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/policy/detach"},
	{Name: "DetachTypedLink", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/typedlink/detach"},
	{Name: "DisableDirectory", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/directory/disable"},
	{Name: "EnableDirectory", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/directory/enable"},
	{Name: "GetAppliedSchemaVersion", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/schema/getappliedschema"},
	{Name: "GetDirectory", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/directory/get"},
	{Name: "GetFacet", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/facet"},
	{Name: "GetLinkAttributes", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/typedlink/attributes/get"},
	{Name: "GetObjectAttributes", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/object/attributes/get"},
	{Name: "GetObjectInformation", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/object/information"},
	{Name: "GetSchemaAsJson", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/schema/json"},
	{Name: "GetTypedLinkFacetInformation", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/typedlink/facet/get"},
	{Name: "ListAppliedSchemaArns", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/schema/applied"},
	{Name: "ListAttachedIndices", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/object/indices"},
	{Name: "ListDevelopmentSchemaArns", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/schema/development"},
	{Name: "ListDirectories", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/directory/list"},
	{Name: "ListFacetAttributes", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/facet/attributes"},
	{Name: "ListFacetNames", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/facet/list"},
	{Name: "ListIncomingTypedLinks", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/typedlink/incoming"},
	{Name: "ListIndex", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/index/targets"},
	{Name: "ListManagedSchemaArns", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/schema/managed"},
	{Name: "ListObjectAttributes", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/object/attributes"},
	{Name: "ListObjectChildren", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/object/children"},
	{Name: "ListObjectParentPaths", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/object/parentpaths"},
	{Name: "ListObjectParents", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/object/parent"},
	{Name: "ListObjectPolicies", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/object/policy"},
	{Name: "ListOutgoingTypedLinks", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/typedlink/outgoing"},
	{Name: "ListPolicyAttachments", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/policy/attachment"},
	{Name: "ListPublishedSchemaArns", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/schema/published"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/tags"},
	{Name: "ListTypedLinkFacetAttributes", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/typedlink/facet/attributes"},
	{Name: "ListTypedLinkFacetNames", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/typedlink/facet/list"},
	{Name: "LookupPolicy", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/policy/lookup"},
	{Name: "PublishSchema", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/schema/publish"},
	{Name: "PutSchemaFromJson", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/schema/json"},
	{Name: "RemoveFacetFromObject", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/object/facets/delete"},
	{Name: "TagResource", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/tags/add"},
	{Name: "UntagResource", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/tags/remove"},
	{Name: "UpdateFacet", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/facet"},
	{Name: "UpdateLinkAttributes", Method: "POST", URI: "/amazonclouddirectory/2017-01-11/typedlink/attributes/update"},
	{Name: "UpdateObjectAttributes", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/object/update"},
	{Name: "UpdateSchema", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/schema/update"},
	{Name: "UpdateTypedLinkFacet", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/typedlink/facet"},
	{Name: "UpgradeAppliedSchema", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/schema/upgradeapplied"},
	{Name: "UpgradePublishedSchema", Method: "PUT", URI: "/amazonclouddirectory/2017-01-11/schema/upgradepublished"},
}

var cloudDirectoryOperationByName = func() map[string]cloudDirectoryOperation {
	out := make(map[string]cloudDirectoryOperation, len(cloudDirectoryOperations))
	for _, op := range cloudDirectoryOperations {
		out[op.Name] = op
	}
	return out
}()
