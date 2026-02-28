package server

type controlCatalogDataType struct {
	Name string
}

// AWS Control Catalog data types sourced from:
// https://docs.aws.amazon.com/controlcatalog/latest/APIReference/API_Types.html
var controlCatalogDataTypes = []controlCatalogDataType{
	{Name: "AssociatedDomainSummary"},
	{Name: "AssociatedObjectiveSummary"},
	{Name: "CommonControlFilter"},
	{Name: "CommonControlMappingDetails"},
	{Name: "CommonControlSummary"},
	{Name: "ControlFilter"},
	{Name: "ControlMapping"},
	{Name: "ControlMappingFilter"},
	{Name: "ControlParameter"},
	{Name: "ControlSummary"},
	{Name: "DomainResourceFilter"},
	{Name: "DomainSummary"},
	{Name: "FrameworkMappingDetails"},
	{Name: "ImplementationDetails"},
	{Name: "ImplementationFilter"},
	{Name: "ImplementationSummary"},
	{Name: "ListObjectives"},
	{Name: "Mapping"},
	{Name: "ObjectiveFilter"},
	{Name: "ObjectiveResourceFilter"},
	{Name: "ObjectiveSummary"},
	{Name: "RegionConfiguration"},
	{Name: "RelatedControlMappingDetails"},
}

var controlCatalogDataTypeByName = func() map[string]controlCatalogDataType {
	out := make(map[string]controlCatalogDataType, len(controlCatalogDataTypes))
	for _, dt := range controlCatalogDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
