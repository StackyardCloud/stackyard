package server

type healthImagingOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon HealthImaging operations sourced from:
// https://docs.aws.amazon.com/healthimaging/latest/APIReference/API_Operations.html
var healthImagingOperations = []healthImagingOperation{
	{Name: "CopyImageSet", Method: "POST", URI: "/datastore/{datastoreId}/imageSet/{sourceImageSetId}/copyImageSet"},
	{Name: "CreateDatastore", Method: "POST", URI: "/datastore"},
	{Name: "DeleteDatastore", Method: "DELETE", URI: "/datastore/{datastoreId}"},
	{Name: "DeleteImageSet", Method: "POST", URI: "/datastore/{datastoreId}/imageSet/{imageSetId}/deleteImageSet"},
	{Name: "GetDatastore", Method: "GET", URI: "/datastore/{datastoreId}"},
	{Name: "GetDICOMImportJob", Method: "GET", URI: "/getDICOMImportJob/datastore/{datastoreId}/job/{jobId}"},
	{Name: "GetImageFrame", Method: "POST", URI: "/datastore/{datastoreId}/imageSet/{imageSetId}/getImageFrame"},
	{Name: "GetImageSet", Method: "POST", URI: "/datastore/{datastoreId}/imageSet/{imageSetId}/getImageSet"},
	{Name: "GetImageSetMetadata", Method: "POST", URI: "/datastore/{datastoreId}/imageSet/{imageSetId}/getImageSetMetadata"},
	{Name: "ListDatastores", Method: "GET", URI: "/datastore"},
	{Name: "ListDICOMImportJobs", Method: "GET", URI: "/listDICOMImportJobs/datastore/{datastoreId}"},
	{Name: "ListImageSetVersions", Method: "POST", URI: "/datastore/{datastoreId}/imageSet/{imageSetId}/listImageSetVersions"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "SearchImageSets", Method: "POST", URI: "/datastore/{datastoreId}/searchImageSets"},
	{Name: "StartDICOMImportJob", Method: "POST", URI: "/startDICOMImportJob/datastore/{datastoreId}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateImageSetMetadata", Method: "POST", URI: "/datastore/{datastoreId}/imageSet/{imageSetId}/updateImageSetMetadata"},
}

var healthImagingOperationByName = func() map[string]healthImagingOperation {
	out := make(map[string]healthImagingOperation, len(healthImagingOperations))
	for _, op := range healthImagingOperations {
		out[op.Name] = op
	}
	return out
}()
