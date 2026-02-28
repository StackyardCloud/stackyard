package server

type stsDataType struct {
	Name string
}

// AWS Security Token Service (STS) data types sourced from:
// https://docs.aws.amazon.com/STS/latest/APIReference/API_Types.html
var stsDataTypes = []stsDataType{
	{Name: "AssumedRoleUser"},
	{Name: "Credentials"},
	{Name: "FederatedUser"},
	{Name: "GetWebIdentityToken"},
	{Name: "PolicyDescriptorType"},
	{Name: "ProvidedContext"},
	{Name: "Tag"},
	{Name: "Types"},
}

var stsDataTypeByName = func() map[string]stsDataType {
	out := make(map[string]stsDataType, len(stsDataTypes))
	for _, dt := range stsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
