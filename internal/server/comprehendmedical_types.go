package server

type comprehendMedicalDataType struct {
	Name string
}

// Amazon Comprehend Medical data types sourced from:
// https://docs.aws.amazon.com/comprehend-medical/latest/api/API_Types.html
var comprehendMedicalDataTypes = []comprehendMedicalDataType{
	{Name: "Attribute"},
	{Name: "Characters"},
	{Name: "ComprehendMedicalAsyncJobFilter"},
	{Name: "ComprehendMedicalAsyncJobProperties"},
	{Name: "Entity"},
	{Name: "ICD10CMAttribute"},
	{Name: "ICD10CMConcept"},
	{Name: "ICD10CMEntity"},
	{Name: "ICD10CMTrait"},
	{Name: "InputDataConfig"},
	{Name: "OutputDataConfig"},
	{Name: "RxNormAttribute"},
	{Name: "RxNormConcept"},
	{Name: "RxNormEntity"},
	{Name: "RxNormTrait"},
	{Name: "SNOMEDCTAttribute"},
	{Name: "SNOMEDCTConcept"},
	{Name: "SNOMEDCTDetails"},
	{Name: "SNOMEDCTEntity"},
	{Name: "SNOMEDCTTrait"},
	{Name: "Trait"},
	{Name: "UnmappedAttribute"},
}

var comprehendMedicalDataTypeByName = func() map[string]comprehendMedicalDataType {
	out := make(map[string]comprehendMedicalDataType, len(comprehendMedicalDataTypes))
	for _, dt := range comprehendMedicalDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
