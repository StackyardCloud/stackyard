package server

type amplifyUIBuilderResource struct {
	Name string
}

// AWS Amplify UI Builder API types sourced from:
// https://docs.aws.amazon.com/amplifyuibuilder/latest/APIReference/API_Types.html
var amplifyUIBuilderResources = []amplifyUIBuilderResource{
	{Name: "ActionParameters"},
	{Name: "ApiConfiguration"},
	{Name: "CodegenDependency"},
	{Name: "CodegenFeatureFlags"},
	{Name: "CodegenGenericDataEnum"},
	{Name: "CodegenGenericDataField"},
	{Name: "CodegenGenericDataModel"},
	{Name: "CodegenGenericDataNonModel"},
	{Name: "CodegenGenericDataRelationshipType"},
	{Name: "CodegenJob"},
	{Name: "CodegenJobAsset"},
	{Name: "CodegenJobGenericDataSchema"},
	{Name: "CodegenJobRenderConfig"},
	{Name: "CodegenJobSummary"},
	{Name: "Component"},
	{Name: "ComponentBindingPropertiesValue"},
	{Name: "ComponentBindingPropertiesValueProperties"},
	{Name: "ComponentChild"},
	{Name: "ComponentConditionProperty"},
	{Name: "ComponentDataConfiguration"},
	{Name: "ComponentEvent"},
	{Name: "ComponentProperty"},
	{Name: "ComponentPropertyBindingProperties"},
	{Name: "ComponentSummary"},
	{Name: "ComponentVariant"},
	{Name: "CreateComponentData"},
	{Name: "CreateFormData"},
	{Name: "CreateThemeData"},
	{Name: "DataStoreRenderConfig"},
	{Name: "ExchangeCodeForTokenRequestBody"},
	{Name: "FieldConfig"},
	{Name: "FieldInputConfig"},
	{Name: "FieldPosition"},
	{Name: "FieldValidationConfiguration"},
	{Name: "FileUploaderFieldConfig"},
	{Name: "Form"},
	{Name: "FormBindingElement"},
	{Name: "FormButton"},
	{Name: "FormCTA"},
	{Name: "FormDataTypeConfig"},
	{Name: "FormInputBindingPropertiesValue"},
	{Name: "FormInputBindingPropertiesValueProperties"},
	{Name: "FormInputValueProperty"},
	{Name: "FormInputValuePropertyBindingProperties"},
	{Name: "FormStyle"},
	{Name: "FormStyleConfig"},
	{Name: "FormSummary"},
	{Name: "GraphQLRenderConfig"},
	{Name: "MutationActionSetStateParameter"},
	{Name: "NoApiRenderConfig"},
	{Name: "Predicate"},
	{Name: "PutMetadataFlagBody"},
	{Name: "ReactStartCodegenJobData"},
	{Name: "RefreshTokenRequestBody"},
	{Name: "SectionalElement"},
	{Name: "SortProperty"},
	{Name: "StartCodegenJobData"},
	{Name: "Theme"},
	{Name: "ThemeSummary"},
	{Name: "ThemeValue"},
	{Name: "ThemeValues"},
	{Name: "UpdateComponentData"},
	{Name: "UpdateFormData"},
	{Name: "UpdateThemeData"},
	{Name: "ValueMapping"},
	{Name: "ValueMappings"},
}

var amplifyUIBuilderResourceByName = func() map[string]amplifyUIBuilderResource {
	out := make(map[string]amplifyUIBuilderResource, len(amplifyUIBuilderResources))
	for _, r := range amplifyUIBuilderResources {
		out[r.Name] = r
	}
	return out
}()
