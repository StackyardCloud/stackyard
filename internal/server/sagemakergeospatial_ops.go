package server

type sageMakerGeospatialOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon SageMaker geospatial capabilities operations sourced from:
// https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_Operations_Amazon_SageMaker_geospatial_capabilities.html
var sageMakerGeospatialOperations = []sageMakerGeospatialOperation{
	{Name: "DeleteEarthObservationJob", Method: "DELETE", URI: "/earth-observation-jobs/{Arn}"},
	{Name: "DeleteVectorEnrichmentJob", Method: "DELETE", URI: "/vector-enrichment-jobs/{Arn}"},
	{Name: "ExportEarthObservationJob", Method: "POST", URI: "/export-earth-observation-job"},
	{Name: "ExportVectorEnrichmentJob", Method: "POST", URI: "/export-vector-enrichment-jobs"},
	{Name: "GetEarthObservationJob", Method: "GET", URI: "/earth-observation-jobs/{Arn}"},
	{Name: "GetRasterDataCollection", Method: "GET", URI: "/raster-data-collection/{Arn}"},
	{Name: "GetTile", Method: "GET", URI: "/tile/{z}/{x}/{y}"},
	{Name: "GetVectorEnrichmentJob", Method: "GET", URI: "/vector-enrichment-jobs/{Arn}"},
	{Name: "ListEarthObservationJobs", Method: "POST", URI: "/list-earth-observation-jobs"},
	{Name: "ListRasterDataCollections", Method: "GET", URI: "/raster-data-collections"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "ListVectorEnrichmentJobs", Method: "POST", URI: "/list-vector-enrichment-jobs"},
	{Name: "SearchRasterDataCollection", Method: "POST", URI: "/search-raster-data-collection"},
	{Name: "StartEarthObservationJob", Method: "POST", URI: "/earth-observation-jobs"},
	{Name: "StartVectorEnrichmentJob", Method: "POST", URI: "/vector-enrichment-jobs"},
	{Name: "StopEarthObservationJob", Method: "POST", URI: "/earth-observation-jobs/stop"},
	{Name: "StopVectorEnrichmentJob", Method: "POST", URI: "/vector-enrichment-jobs/stop"},
	{Name: "TagResource", Method: "PUT", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
}
