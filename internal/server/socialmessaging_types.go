package server

type socialMessagingDataType struct {
	Name string
}

// Amazon End User Messaging Social data types sourced from:
// https://docs.aws.amazon.com/social-messaging/latest/APIReference/API_Types.html
var socialMessagingDataTypes = []socialMessagingDataType{
	{Name: "LibraryTemplateBodyInputs"},
	{Name: "LibraryTemplateButtonInput"},
	{Name: "LibraryTemplateButtonList"},
	{Name: "LinkedWhatsAppBusinessAccount"},
	{Name: "LinkedWhatsAppBusinessAccountIdMetaData"},
	{Name: "LinkedWhatsAppBusinessAccountSummary"},
	{Name: "MetaLibraryTemplate"},
	{Name: "MetaLibraryTemplateDefinition"},
	{Name: "S3File"},
	{Name: "S3PresignedUrl"},
	{Name: "Tag"},
	{Name: "TemplateSummary"},
	{Name: "UpdateWhatsAppMessageTemplate"},
	{Name: "WabaPhoneNumberSetupFinalization"},
	{Name: "WabaSetupFinalization"},
	{Name: "WhatsAppBusinessAccountEventDestination"},
	{Name: "WhatsAppPhoneNumberDetail"},
	{Name: "WhatsAppPhoneNumberSummary"},
	{Name: "WhatsAppSetupFinalization"},
	{Name: "WhatsAppSignupCallback"},
	{Name: "WhatsAppSignupCallbackResult"},
}

var socialMessagingDataTypeByName = func() map[string]socialMessagingDataType {
	out := make(map[string]socialMessagingDataType, len(socialMessagingDataTypes))
	for _, dt := range socialMessagingDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
