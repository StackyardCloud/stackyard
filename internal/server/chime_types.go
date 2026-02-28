package server

type chimeType struct {
	Name string
}

// Amazon Chime data types sourced from:
// https://docs.aws.amazon.com/chime/latest/APIReference/API_Types.html
var chimeTypes = []chimeType{
	{Name: "Account"},
	{Name: "AccountSettings"},
	{Name: "AlexaForBusinessMetadata"},
	{Name: "Bot"},
	{Name: "BusinessCallingSettings"},
	{Name: "ConversationRetentionSettings"},
	{Name: "EventsConfiguration"},
	{Name: "Invite"},
	{Name: "Member"},
	{Name: "MemberError"},
	{Name: "MembershipItem"},
	{Name: "OrderedPhoneNumber"},
	{Name: "PhoneNumber"},
	{Name: "PhoneNumberAssociation"},
	{Name: "PhoneNumberCapabilities"},
	{Name: "PhoneNumberCountry"},
	{Name: "PhoneNumberError"},
	{Name: "PhoneNumberOrder"},
	{Name: "RetentionSettings"},
	{Name: "Room"},
	{Name: "RoomMembership"},
	{Name: "RoomRetentionSettings"},
	{Name: "SigninDelegateGroup"},
	{Name: "TelephonySettings"},
	{Name: "UpdatePhoneNumberRequestItem"},
	{Name: "UpdateUserRequestItem"},
	{Name: "UpdateUserSettings"},
	{Name: "User"},
	{Name: "UserError"},
	{Name: "UserSettings"},
	{Name: "VoiceConnectorSettings"},
}

var chimeTypeByName = func() map[string]chimeType {
	out := make(map[string]chimeType, len(chimeTypes))
	for _, dt := range chimeTypes {
		out[dt.Name] = dt
	}
	return out
}()
