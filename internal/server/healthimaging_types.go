package server

type healthImagingDataType struct {
	Name string
}

// Amazon HealthImaging data types sourced from:
// https://docs.aws.amazon.com/healthimaging/latest/APIReference/API_Types.html
var healthImagingDataTypes = []healthImagingDataType{
	{Name: "CopyDestinationImageSet"},
	{Name: "CopyDestinationImageSetProperties"},
	{Name: "CopyImageSetInformation"},
	{Name: "CopySourceImageSetInformation"},
	{Name: "CopySourceImageSetProperties"},
	{Name: "DatastoreProperties"},
	{Name: "DatastoreSummary"},
	{Name: "DICOMImportJobProperties"},
	{Name: "DICOMImportJobSummary"},
	{Name: "DICOMStudyDateAndTime"},
	{Name: "DICOMTags"},
	{Name: "DICOMUpdates"},
	{Name: "ImageFrameInformation"},
	{Name: "ImageSetProperties"},
	{Name: "ImageSetsMetadataSummary"},
	{Name: "MetadataCopies"},
	{Name: "MetadataUpdates"},
	{Name: "Overrides"},
	{Name: "SearchByAttributeValue"},
	{Name: "SearchCriteria"},
	{Name: "SearchFilter"},
	{Name: "Sort"},
}

var healthImagingDataTypeByName = func() map[string]healthImagingDataType {
	out := make(map[string]healthImagingDataType, len(healthImagingDataTypes))
	for _, dt := range healthImagingDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
