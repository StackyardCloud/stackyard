package server

type chimeOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Chime actions sourced from:
// https://docs.aws.amazon.com/chime/latest/APIReference/API_Operations.html
var chimeOperations = []chimeOperation{
	{Name: "AssociatePhoneNumberWithUser", Method: "POST", URI: "/accounts/{AccountId}/users/{UserId}?operation=associate-phone-number"},
	{Name: "AssociateSigninDelegateGroupsWithAccount", Method: "POST", URI: "/accounts/{AccountId}?operation=associate-signin-delegate-groups"},
	{Name: "BatchCreateRoomMembership", Method: "POST", URI: "/accounts/{AccountId}/rooms/{RoomId}/memberships?operation=batch-create"},
	{Name: "BatchDeletePhoneNumber", Method: "POST", URI: "/phone-numbers?operation=batch-delete"},
	{Name: "BatchSuspendUser", Method: "POST", URI: "/accounts/{AccountId}/users?operation=suspend"},
	{Name: "BatchUnsuspendUser", Method: "POST", URI: "/accounts/{AccountId}/users?operation=unsuspend"},
	{Name: "BatchUpdatePhoneNumber", Method: "POST", URI: "/phone-numbers?operation=batch-update"},
	{Name: "BatchUpdateUser", Method: "POST", URI: "/accounts/{AccountId}/users"},
	{Name: "CreateAccount", Method: "POST", URI: "/accounts"},
	{Name: "CreateBot", Method: "POST", URI: "/accounts/{AccountId}/bots"},
	{Name: "CreateMeetingDialOut", Method: "POST", URI: "/meetings/{MeetingId}/dial-outs"},
	{Name: "CreatePhoneNumberOrder", Method: "POST", URI: "/phone-number-orders"},
	{Name: "CreateRoom", Method: "POST", URI: "/accounts/{AccountId}/rooms"},
	{Name: "CreateRoomMembership", Method: "POST", URI: "/accounts/{AccountId}/rooms/{RoomId}/memberships"},
	{Name: "CreateUser", Method: "POST", URI: "/accounts/{AccountId}/users?operation=create"},
	{Name: "DeleteAccount", Method: "DELETE", URI: "/accounts/{AccountId}"},
	{Name: "DeleteEventsConfiguration", Method: "DELETE", URI: "/accounts/{AccountId}/bots/{BotId}/events-configuration"},
	{Name: "DeletePhoneNumber", Method: "DELETE", URI: "/phone-numbers/{PhoneNumberId}"},
	{Name: "DeleteRoom", Method: "DELETE", URI: "/accounts/{AccountId}/rooms/{RoomId}"},
	{Name: "DeleteRoomMembership", Method: "DELETE", URI: "/accounts/{AccountId}/rooms/{RoomId}/memberships/{MemberId}"},
	{Name: "DisassociatePhoneNumberFromUser", Method: "POST", URI: "/accounts/{AccountId}/users/{UserId}?operation=disassociate-phone-number"},
	{Name: "DisassociateSigninDelegateGroupsFromAccount", Method: "POST", URI: "/accounts/{AccountId}?operation=disassociate-signin-delegate-groups"},
	{Name: "GetAccount", Method: "GET", URI: "/accounts/{AccountId}"},
	{Name: "GetAccountSettings", Method: "GET", URI: "/accounts/{AccountId}/settings"},
	{Name: "GetBot", Method: "GET", URI: "/accounts/{AccountId}/bots/{BotId}"},
	{Name: "GetEventsConfiguration", Method: "GET", URI: "/accounts/{AccountId}/bots/{BotId}/events-configuration"},
	{Name: "GetGlobalSettings", Method: "GET", URI: "/settings"},
	{Name: "GetPhoneNumber", Method: "GET", URI: "/phone-numbers/{PhoneNumberId}"},
	{Name: "GetPhoneNumberOrder", Method: "GET", URI: "/phone-number-orders/{PhoneNumberOrderId}"},
	{Name: "GetPhoneNumberSettings", Method: "GET", URI: "/settings/phone-number"},
	{Name: "GetRetentionSettings", Method: "GET", URI: "/accounts/{AccountId}/retention-settings"},
	{Name: "GetRoom", Method: "GET", URI: "/accounts/{AccountId}/rooms/{RoomId}"},
	{Name: "GetUser", Method: "GET", URI: "/accounts/{AccountId}/users/{UserId}"},
	{Name: "GetUserSettings", Method: "GET", URI: "/accounts/{AccountId}/users/{UserId}/settings"},
	{Name: "InviteUsers", Method: "POST", URI: "/accounts/{AccountId}/users?operation=add"},
	{Name: "ListAccounts", Method: "GET", URI: "/accounts"},
	{Name: "ListBots", Method: "GET", URI: "/accounts/{AccountId}/bots"},
	{Name: "ListPhoneNumberOrders", Method: "GET", URI: "/phone-number-orders"},
	{Name: "ListPhoneNumbers", Method: "GET", URI: "/phone-numbers"},
	{Name: "ListRoomMemberships", Method: "GET", URI: "/accounts/{AccountId}/rooms/{RoomId}/memberships"},
	{Name: "ListRooms", Method: "GET", URI: "/accounts/{AccountId}/rooms"},
	{Name: "ListSupportedPhoneNumberCountries", Method: "GET", URI: "/phone-number-countries"},
	{Name: "ListUsers", Method: "GET", URI: "/accounts/{AccountId}/users"},
	{Name: "LogoutUser", Method: "POST", URI: "/accounts/{AccountId}/users/{UserId}?operation=logout"},
	{Name: "PutEventsConfiguration", Method: "PUT", URI: "/accounts/{AccountId}/bots/{BotId}/events-configuration"},
	{Name: "PutRetentionSettings", Method: "PUT", URI: "/accounts/{AccountId}/retention-settings"},
	{Name: "RedactConversationMessage", Method: "POST", URI: "/accounts/{AccountId}/conversations/{ConversationId}/messages/{MessageId}?operation=redact"},
	{Name: "RedactRoomMessage", Method: "POST", URI: "/accounts/{AccountId}/rooms/{RoomId}/messages/{MessageId}?operation=redact"},
	{Name: "RegenerateSecurityToken", Method: "POST", URI: "/accounts/{AccountId}/bots/{BotId}?operation=regenerate-security-token"},
	{Name: "ResetPersonalPIN", Method: "POST", URI: "/accounts/{AccountId}/users/{UserId}?operation=reset-personal-pin"},
	{Name: "RestorePhoneNumber", Method: "POST", URI: "/phone-numbers/{PhoneNumberId}?operation=restore"},
	{Name: "SearchAvailablePhoneNumbers", Method: "GET", URI: "/search?type=phone-numbers"},
	{Name: "UpdateAccount", Method: "POST", URI: "/accounts/{AccountId}"},
	{Name: "UpdateAccountSettings", Method: "PUT", URI: "/accounts/{AccountId}/settings"},
	{Name: "UpdateBot", Method: "POST", URI: "/accounts/{AccountId}/bots/{BotId}"},
	{Name: "UpdateGlobalSettings", Method: "PUT", URI: "/settings"},
	{Name: "UpdatePhoneNumber", Method: "POST", URI: "/phone-numbers/{PhoneNumberId}"},
	{Name: "UpdatePhoneNumberSettings", Method: "PUT", URI: "/settings/phone-number"},
	{Name: "UpdateRoom", Method: "POST", URI: "/accounts/{AccountId}/rooms/{RoomId}"},
	{Name: "UpdateRoomMembership", Method: "POST", URI: "/accounts/{AccountId}/rooms/{RoomId}/memberships/{MemberId}"},
	{Name: "UpdateUser", Method: "POST", URI: "/accounts/{AccountId}/users/{UserId}"},
	{Name: "UpdateUserSettings", Method: "PUT", URI: "/accounts/{AccountId}/users/{UserId}/settings"},
}

var chimeOperationByName = func() map[string]chimeOperation {
	out := make(map[string]chimeOperation, len(chimeOperations))
	for _, op := range chimeOperations {
		out[op.Name] = op
	}
	return out
}()
