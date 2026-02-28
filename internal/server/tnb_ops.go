package server

type tnbOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Telco Network Builder (TNB) actions sourced from:
// https://docs.aws.amazon.com/tnb/latest/APIReference/API_Operations.html
var tnbOperations = []tnbOperation{
	{Name: "CancelSolNetworkOperation", Method: "POST", URI: "/sol/nslcm/v1/ns_lcm_op_occs/{nsLcmOpOccId}/cancel"},
	{Name: "CreateSolFunctionPackage", Method: "POST", URI: "/sol/vnfpkgm/v1/vnf_packages"},
	{Name: "CreateSolNetworkInstance", Method: "POST", URI: "/sol/nslcm/v1/ns_instances"},
	{Name: "CreateSolNetworkPackage", Method: "POST", URI: "/sol/nsd/v1/ns_descriptors"},
	{Name: "DeleteSolFunctionPackage", Method: "DELETE", URI: "/sol/vnfpkgm/v1/vnf_packages/{vnfPkgId}"},
	{Name: "DeleteSolNetworkInstance", Method: "DELETE", URI: "/sol/nslcm/v1/ns_instances/{nsInstanceId}"},
	{Name: "DeleteSolNetworkPackage", Method: "DELETE", URI: "/sol/nsd/v1/ns_descriptors/{nsdInfoId}"},
	{Name: "GetSolFunctionInstance", Method: "GET", URI: "/sol/vnflcm/v1/vnf_instances/{vnfInstanceId}"},
	{Name: "GetSolFunctionPackage", Method: "GET", URI: "/sol/vnfpkgm/v1/vnf_packages/{vnfPkgId}"},
	{Name: "GetSolFunctionPackageContent", Method: "GET", URI: "/sol/vnfpkgm/v1/vnf_packages/{vnfPkgId}/package_content"},
	{Name: "GetSolFunctionPackageDescriptor", Method: "GET", URI: "/sol/vnfpkgm/v1/vnf_packages/{vnfPkgId}/vnfd"},
	{Name: "GetSolNetworkInstance", Method: "GET", URI: "/sol/nslcm/v1/ns_instances/{nsInstanceId}"},
	{Name: "GetSolNetworkOperation", Method: "GET", URI: "/sol/nslcm/v1/ns_lcm_op_occs/{nsLcmOpOccId}"},
	{Name: "GetSolNetworkPackage", Method: "GET", URI: "/sol/nsd/v1/ns_descriptors/{nsdInfoId}"},
	{Name: "GetSolNetworkPackageContent", Method: "GET", URI: "/sol/nsd/v1/ns_descriptors/{nsdInfoId}/nsd_content"},
	{Name: "GetSolNetworkPackageDescriptor", Method: "GET", URI: "/sol/nsd/v1/ns_descriptors/{nsdInfoId}/nsd"},
	{Name: "InstantiateSolNetworkInstance", Method: "POST", URI: "/sol/nslcm/v1/ns_instances/{nsInstanceId}/instantiate?dry_run={dryRun}"},
	{Name: "ListSolFunctionInstances", Method: "GET", URI: "/sol/vnflcm/v1/vnf_instances?max_results={maxResults}&nextpage_opaque_marker={nextToken}"},
	{Name: "ListSolFunctionPackages", Method: "GET", URI: "/sol/vnfpkgm/v1/vnf_packages?max_results={maxResults}&nextpage_opaque_marker={nextToken}"},
	{Name: "ListSolNetworkInstances", Method: "GET", URI: "/sol/nslcm/v1/ns_instances?max_results={maxResults}&nextpage_opaque_marker={nextToken}"},
	{Name: "ListSolNetworkOperations", Method: "GET", URI: "/sol/nslcm/v1/ns_lcm_op_occs?max_results={maxResults}&nextpage_opaque_marker={nextToken}&nsInstanceId={nsInstanceId}"},
	{Name: "ListSolNetworkPackages", Method: "GET", URI: "/sol/nsd/v1/ns_descriptors?max_results={maxResults}&nextpage_opaque_marker={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PutSolFunctionPackageContent", Method: "PUT", URI: "/sol/vnfpkgm/v1/vnf_packages/{vnfPkgId}/package_content"},
	{Name: "PutSolNetworkPackageContent", Method: "PUT", URI: "/sol/nsd/v1/ns_descriptors/{nsdInfoId}/nsd_content"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "TerminateSolNetworkInstance", Method: "POST", URI: "/sol/nslcm/v1/ns_instances/{nsInstanceId}/terminate"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateSolFunctionPackage", Method: "PATCH", URI: "/sol/vnfpkgm/v1/vnf_packages/{vnfPkgId}"},
	{Name: "UpdateSolNetworkInstance", Method: "POST", URI: "/sol/nslcm/v1/ns_instances/{nsInstanceId}/update"},
	{Name: "UpdateSolNetworkPackage", Method: "PATCH", URI: "/sol/nsd/v1/ns_descriptors/{nsdInfoId}"},
	{Name: "ValidateSolFunctionPackageContent", Method: "PUT", URI: "/sol/vnfpkgm/v1/vnf_packages/{vnfPkgId}/package_content/validate"},
	{Name: "ValidateSolNetworkPackageContent", Method: "PUT", URI: "/sol/nsd/v1/ns_descriptors/{nsdInfoId}/nsd_content/validate"},
}

var tnbOperationByName = func() map[string]tnbOperation {
	out := make(map[string]tnbOperation, len(tnbOperations))
	for _, op := range tnbOperations {
		out[op.Name] = op
	}
	return out
}()
