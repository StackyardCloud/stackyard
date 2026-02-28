package server

type healthLakeDataType struct {
	Name string
}

// Amazon HealthLake data types sourced from:
// https://docs.aws.amazon.com/healthlake/latest/APIReference/API_Types.html
var healthLakeDataTypes = []healthLakeDataType{
	{Name: "DatastoreFilter"},
	{Name: "DatastoreProperties"},
	{Name: "ErrorCause"},
	{Name: "ExportJobProperties"},
	{Name: "IdentityProviderConfiguration"},
	{Name: "ImportJobProperties"},
	{Name: "InputDataConfig"},
	{Name: "JobProgressReport"},
	{Name: "KmsEncryptionConfig"},
	{Name: "OutputDataConfig"},
	{Name: "PreloadDataConfig"},
	{Name: "S3Configuration"},
	{Name: "SseConfiguration"},
	{Name: "Tag"},
}

var healthLakeDataTypeByName = func() map[string]healthLakeDataType {
	out := make(map[string]healthLakeDataType, len(healthLakeDataTypes))
	for _, dt := range healthLakeDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
