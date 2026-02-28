package server

type workspacesWebType struct {
	Name string
}

// Amazon WorkSpaces Secure Browser data types sourced from:

// https://docs.aws.amazon.com/workspaces-web/latest/APIReference/API_Types.html

var workspacesWebTypes = []workspacesWebType{
	{Name: "BrandingConfiguration"},
	{Name: "BrandingConfigurationCreateInput"},
	{Name: "BrandingConfigurationUpdateInput"},
	{Name: "BrowserSettings"},
	{Name: "BrowserSettingsSummary"},
	{Name: "Certificate"},
	{Name: "CertificateSummary"},
	{Name: "CookieSpecification"},
	{Name: "CookieSynchronizationConfiguration"},
	{Name: "CustomPattern"},
	{Name: "DataProtectionSettings"},
	{Name: "DataProtectionSettingsSummary"},
	{Name: "EventFilter"},
	{Name: "IconImageInput"},
	{Name: "IdentityProvider"},
	{Name: "IdentityProviderSummary"},
	{Name: "ImageMetadata"},
	{Name: "InlineRedactionConfiguration"},
	{Name: "InlineRedactionPattern"},
	{Name: "IpAccessSettings"},
	{Name: "IpAccessSettingsSummary"},
	{Name: "IpRule"},
	{Name: "LocalizedBrandingStrings"},
	{Name: "LogConfiguration"},
	{Name: "NetworkSettings"},
	{Name: "NetworkSettingsSummary"},
	{Name: "Portal"},
	{Name: "PortalSummary"},
	{Name: "RedactionPlaceHolder"},
	{Name: "S3LogConfiguration"},
	{Name: "Session"},
	{Name: "SessionLogger"},
	{Name: "SessionLoggerSummary"},
	{Name: "SessionSummary"},
	{Name: "Tag"},
	{Name: "ToolbarConfiguration"},
	{Name: "TrustStore"},
	{Name: "TrustStoreSummary"},
	{Name: "UserAccessLoggingSettings"},
	{Name: "UserAccessLoggingSettingsSummary"},
	{Name: "UserSettings"},
	{Name: "UserSettingsSummary"},
	{Name: "ValidationExceptionField"},
	{Name: "WallpaperImageInput"},
	{Name: "WebContentFilteringPolicy"},
}

var workspacesWebTypeByName = func() map[string]workspacesWebType {
	out := make(map[string]workspacesWebType, len(workspacesWebTypes))
	for _, typ := range workspacesWebTypes {
		out[typ.Name] = typ
	}
	return out
}()
