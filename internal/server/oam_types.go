package server

type oamDataType struct {
	Name string
}

// Amazon CloudWatch Observability Access Manager data types sourced from:
// https://docs.aws.amazon.com/OAM/latest/APIReference/API_Types.html
var oamDataTypes = []oamDataType{
	{Name: "LinkConfiguration"},
	{Name: "ListAttachedLinksItem"},
	{Name: "ListLinksItem"},
	{Name: "ListSinksItem"},
	{Name: "LogGroupConfiguration"},
	{Name: "MetricConfiguration"},
}

var oamDataTypeByName = func() map[string]oamDataType {
	out := make(map[string]oamDataType, len(oamDataTypes))
	for _, dt := range oamDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
