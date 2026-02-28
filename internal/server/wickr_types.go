package server

type wickrDataType struct {
	Name string
}

// AWS Wickr data types sourced from:
// https://docs.aws.amazon.com/wickr/latest/APIReference/API_Types.html
var wickrDataTypes = []wickrDataType{
	{Name: "BasicDeviceObject"},
	{Name: "BatchCreateUserRequestItem"},
	{Name: "BatchDeviceErrorResponseItem"},
	{Name: "BatchDeviceSuccessResponseItem"},
	{Name: "BatchUnameErrorResponseItem"},
	{Name: "BatchUnameSuccessResponseItem"},
	{Name: "BatchUserErrorResponseItem"},
	{Name: "BatchUserSuccessResponseItem"},
	{Name: "BlockedGuestUser"},
	{Name: "Bot"},
	{Name: "CallingSettings"},
	{Name: "ErrorDetail"},
	{Name: "GuestUser"},
	{Name: "GuestUserHistoryCount"},
	{Name: "Network"},
	{Name: "NetworkSettings"},
	{Name: "OidcConfigInfo"},
	{Name: "OidcTokenInfo"},
	{Name: "PasswordRequirements"},
	{Name: "PermittedWickrEnterpriseNetwork"},
	{Name: "ReadReceiptConfig"},
	{Name: "SecurityGroup"},
	{Name: "SecurityGroupSettings"},
	{Name: "SecurityGroupSettingsRequest"},
	{Name: "Setting"},
	{Name: "ShredderSettings"},
	{Name: "UpdateUserDetails"},
	{Name: "User"},
	{Name: "WickrAwsNetworks"},
	{Name: "UpdateUser"},
}

var wickrDataTypeByName = func() map[string]wickrDataType {
	out := make(map[string]wickrDataType, len(wickrDataTypes))
	for _, t := range wickrDataTypes {
		out[t.Name] = t
	}
	return out
}()
