package server

type b2biDataType struct {
	Name string
}

// AWS B2B Data Interchange data types sourced from:
// https://docs.aws.amazon.com/b2bi/latest/APIReference/API_Types.html
var b2biDataTypes = []b2biDataType{
	{Name: "AdvancedOptions"},
	{Name: "CapabilityConfiguration"},
	{Name: "CapabilityOptions"},
	{Name: "CapabilitySummary"},
	{Name: "ConversionSource"},
	{Name: "ConversionTarget"},
	{Name: "ConversionTargetFormatDetails"},
	{Name: "EdiConfiguration"},
	{Name: "EdiType"},
	{Name: "FormatOptions"},
	{Name: "InboundEdiOptions"},
	{Name: "InputConversion"},
	{Name: "InputFileSource"},
	{Name: "Mapping"},
	{Name: "OutboundEdiOptions"},
	{Name: "OutputConversion"},
	{Name: "OutputSampleFileSource"},
	{Name: "PartnershipSummary"},
	{Name: "ProfileSummary"},
	{Name: "S3Location"},
	{Name: "SampleDocumentKeys"},
	{Name: "SampleDocuments"},
	{Name: "Tag"},
	{Name: "TemplateDetails"},
	{Name: "TransformerSummary"},
	{Name: "UpdateTransformer"},
	{Name: "WrapOptions"},
	{Name: "X12AcknowledgmentOptions"},
	{Name: "X12AdvancedOptions"},
	{Name: "X12CodeListValidationRule"},
	{Name: "X12ControlNumbers"},
	{Name: "X12Delimiters"},
	{Name: "X12Details"},
	{Name: "X12ElementLengthValidationRule"},
	{Name: "X12ElementRequirementValidationRule"},
	{Name: "X12Envelope"},
	{Name: "X12FunctionalGroupHeaders"},
	{Name: "X12InboundEdiOptions"},
	{Name: "X12InterchangeControlHeaders"},
	{Name: "X12OutboundEdiHeaders"},
	{Name: "X12SplitOptions"},
	{Name: "X12ValidationOptions"},
	{Name: "X12ValidationRule"},
}

var b2biDataTypeByName = func() map[string]b2biDataType {
	out := make(map[string]b2biDataType, len(b2biDataTypes))
	for _, dataType := range b2biDataTypes {
		out[dataType.Name] = dataType
	}
	return out
}()
