package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type chimeStore struct {
	mu sync.Mutex

	nextAccountID    int64
	nextBotID        int64
	nextUserID       int64
	nextPhoneID      int64
	nextPhoneOrderID int64
	nextRoomID       int64
	nextMemberID     int64

	accounts           map[string]map[string]any
	accountSettings    map[string]map[string]any
	users              map[string]map[string]any
	userSettings       map[string]map[string]any
	bots               map[string]map[string]any
	eventsConfigs      map[string]map[string]any
	phoneNumbers       map[string]map[string]any
	phoneNumberOrders  map[string]map[string]any
	rooms              map[string]map[string]any
	memberships        map[string]map[string]any
	retentionSettings  map[string]map[string]any
	globalSettings     map[string]any
	phoneNumberSetting map[string]any
}

func newChimeStore() *chimeStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &chimeStore{
		nextAccountID:      2,
		nextBotID:          2,
		nextUserID:         2,
		nextPhoneID:        2,
		nextPhoneOrderID:   2,
		nextRoomID:         2,
		nextMemberID:       2,
		accounts:           map[string]map[string]any{},
		accountSettings:    map[string]map[string]any{},
		users:              map[string]map[string]any{},
		userSettings:       map[string]map[string]any{},
		bots:               map[string]map[string]any{},
		eventsConfigs:      map[string]map[string]any{},
		phoneNumbers:       map[string]map[string]any{},
		phoneNumberOrders:  map[string]map[string]any{},
		rooms:              map[string]map[string]any{},
		memberships:        map[string]map[string]any{},
		retentionSettings:  map[string]map[string]any{},
		globalSettings:     map[string]any{"BusinessCalling": true, "VoiceConnector": true},
		phoneNumberSetting: map[string]any{"CallingName": "Stackyard"},
	}

	s.accounts["acc-000001"] = map[string]any{
		"AccountId":        "acc-000001",
		"Name":             "stackyard-account",
		"AccountType":      "Team",
		"AwsAccountId":     "123456789012",
		"CreatedTimestamp": now,
	}
	s.accountSettings["acc-000001"] = map[string]any{"DisableRemoteControl": false, "EnableDialOut": true}
	s.users["acc-000001:user-000001"] = map[string]any{
		"UserId":                 "user-000001",
		"AccountId":              "acc-000001",
		"PrimaryEmail":           "user@example.com",
		"DisplayName":            "Stackyard User",
		"UserRegistrationStatus": "Registered",
		"UserInvitationStatus":   "Accepted",
	}
	s.userSettings["acc-000001:user-000001"] = map[string]any{"Telephony": map[string]any{"InboundCalling": true, "OutboundCalling": true}}
	s.bots["acc-000001:bot-000001"] = map[string]any{
		"BotId":            "bot-000001",
		"AccountId":        "acc-000001",
		"DisplayName":      "stackyard-bot",
		"BotEmail":         "stackyard-bot@example.com",
		"BotType":          "ChatBot",
		"SecurityToken":    "stackyard-token",
		"Disabled":         false,
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.eventsConfigs["acc-000001:bot-000001"] = map[string]any{
		"BotId":             "bot-000001",
		"AccountId":         "acc-000001",
		"LambdaFunctionArn": "arn:aws:lambda:us-east-1:123456789012:function:stackyard-chime-events",
	}
	s.phoneNumbers["phone-number-000001"] = map[string]any{
		"PhoneNumberId":   "phone-number-000001",
		"E164PhoneNumber": "+12065550100",
		"Status":          "AcquireSucceeded",
		"ProductType":     "BusinessCalling",
	}
	s.phoneNumberOrders["order-000001"] = map[string]any{
		"PhoneNumberOrderId": "order-000001",
		"ProductType":        "BusinessCalling",
		"Status":             "Successful",
		"CreatedTimestamp":   now,
		"OrderedPhoneNumbers": []any{
			map[string]any{"E164PhoneNumber": "+12065550100", "Status": "Acquired"},
		},
	}
	s.rooms["acc-000001:room-000001"] = map[string]any{
		"RoomId":           "room-000001",
		"AccountId":        "acc-000001",
		"Name":             "stackyard-room",
		"CreatedTimestamp": now,
	}
	s.memberships["acc-000001:room-000001:member-000001"] = map[string]any{
		"RoomId":           "room-000001",
		"MemberId":         "member-000001",
		"AccountId":        "acc-000001",
		"Role":             "Administrator",
		"InvitedBy":        "user-000001",
		"UpdatedTimestamp": now,
	}
	s.retentionSettings["acc-000001"] = map[string]any{
		"RoomRetentionSettings":         map[string]any{"RetentionDays": 30},
		"ConversationRetentionSettings": map[string]any{"RetentionDays": 30},
	}

	return s
}

func (s *chimeStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := chimeMergeMaps(payload, pathParams, query)
	accountID := chimeString(ctx, "AccountId", "acc-000001")
	userID := chimeString(ctx, "UserId", "user-000001")
	botID := chimeString(ctx, "BotId", "bot-000001")
	phoneNumberID := chimeString(ctx, "PhoneNumberId", "phone-number-000001")
	phoneOrderID := chimeString(ctx, "PhoneNumberOrderId", "order-000001")
	meetingID := chimeString(ctx, "MeetingId", "meeting-000001")
	roomID := chimeString(ctx, "RoomId", "room-000001")
	memberID := chimeString(ctx, "MemberId", "member-000001")

	s.ensureAccountLocked(accountID)
	s.ensureUserLocked(accountID, userID)
	s.ensureBotLocked(accountID, botID)
	s.ensurePhoneLocked(phoneNumberID)
	s.ensurePhoneOrderLocked(phoneOrderID)
	s.ensureRoomLocked(accountID, roomID)
	s.ensureMembershipLocked(accountID, roomID, memberID)
	s.ensureRetentionLocked(accountID)

	switch action {
	case "CreateAccount":
		id := fmt.Sprintf("acc-%06d", s.nextAccountID)
		s.nextAccountID++
		name := chimeString(payload, "Name", "stackyard-account")
		s.accounts[id] = map[string]any{
			"AccountId":        id,
			"Name":             name,
			"AccountType":      "Team",
			"AwsAccountId":     "123456789012",
			"CreatedTimestamp": time.Now().UTC().Format(time.RFC3339),
		}
		s.accountSettings[id] = map[string]any{"DisableRemoteControl": false, "EnableDialOut": true}
		return map[string]any{"Account": chimeCloneMap(s.accounts[id])}
	case "GetAccount", "UpdateAccount":
		account := s.ensureAccountLocked(accountID)
		if action == "UpdateAccount" {
			if name := chimeString(payload, "Name", ""); name != "" {
				account["Name"] = name
			}
		}
		return map[string]any{"Account": chimeCloneMap(account)}
	case "DeleteAccount":
		delete(s.accounts, accountID)
		delete(s.accountSettings, accountID)
		return map[string]any{}
	case "ListAccounts":
		items := make([]any, 0, len(s.accounts))
		for _, account := range chimeSortedValues(s.accounts) {
			items = append(items, chimeCloneMap(account))
		}
		return map[string]any{"Accounts": items, "NextToken": ""}
	case "GetAccountSettings", "UpdateAccountSettings":
		settings := s.ensureAccountSettingsLocked(accountID)
		if action == "UpdateAccountSettings" {
			for key, value := range payload {
				settings[key] = value
			}
		}
		return map[string]any{"AccountSettings": chimeCloneMap(settings)}
	case "GetGlobalSettings", "UpdateGlobalSettings":
		if action == "UpdateGlobalSettings" {
			for key, value := range payload {
				s.globalSettings[key] = value
			}
		}
		return map[string]any{"GlobalSettings": chimeCloneMap(s.globalSettings)}
	case "GetPhoneNumberSettings", "UpdatePhoneNumberSettings":
		if action == "UpdatePhoneNumberSettings" {
			for key, value := range payload {
				s.phoneNumberSetting[key] = value
			}
		}
		return map[string]any{"PhoneNumberSettings": chimeCloneMap(s.phoneNumberSetting)}
	case "CreateUser", "InviteUsers":
		id := userID
		if action == "CreateUser" {
			id = fmt.Sprintf("user-%06d", s.nextUserID)
			s.nextUserID++
		}
		user := s.ensureUserLocked(accountID, id)
		if email := chimeString(payload, "Email", ""); email != "" {
			user["PrimaryEmail"] = email
		}
		if display := chimeString(payload, "DisplayName", ""); display != "" {
			user["DisplayName"] = display
		}
		if action == "InviteUsers" {
			return map[string]any{"Invites": []any{map[string]any{"InviteId": fmt.Sprintf("invite-%s", id), "Status": "Succeeded"}}}
		}
		return map[string]any{"User": chimeCloneMap(user)}
	case "GetUser", "UpdateUser":
		user := s.ensureUserLocked(accountID, userID)
		if action == "UpdateUser" {
			if email := chimeString(payload, "PrimaryEmail", ""); email != "" {
				user["PrimaryEmail"] = email
			}
			if display := chimeString(payload, "DisplayName", ""); display != "" {
				user["DisplayName"] = display
			}
		}
		return map[string]any{"User": chimeCloneMap(user)}
	case "BatchUpdateUser", "BatchSuspendUser", "BatchUnsuspendUser":
		return map[string]any{"UserErrors": []any{}}
	case "ListUsers":
		items := make([]any, 0)
		prefix := accountID + ":"
		for key, user := range s.users {
			if strings.HasPrefix(key, prefix) {
				items = append(items, chimeCloneMap(user))
			}
		}
		sort.Slice(items, func(i, j int) bool {
			lhs := chimeString(items[i].(map[string]any), "UserId", "")
			rhs := chimeString(items[j].(map[string]any), "UserId", "")
			return lhs < rhs
		})
		return map[string]any{"Users": items, "NextToken": ""}
	case "GetUserSettings", "UpdateUserSettings":
		settings := s.ensureUserSettingsLocked(accountID, userID)
		if action == "UpdateUserSettings" {
			for key, value := range payload {
				settings[key] = value
			}
		}
		return map[string]any{"UserSettings": chimeCloneMap(settings)}
	case "LogoutUser", "ResetPersonalPIN", "AssociatePhoneNumberWithUser", "DisassociatePhoneNumberFromUser", "AssociateSigninDelegateGroupsWithAccount", "DisassociateSigninDelegateGroupsFromAccount":
		return map[string]any{}
	case "CreateBot":
		id := fmt.Sprintf("bot-%06d", s.nextBotID)
		s.nextBotID++
		bot := s.ensureBotLocked(accountID, id)
		if display := chimeString(payload, "DisplayName", ""); display != "" {
			bot["DisplayName"] = display
		}
		return map[string]any{"Bot": chimeCloneMap(bot)}
	case "GetBot", "UpdateBot", "RegenerateSecurityToken":
		bot := s.ensureBotLocked(accountID, botID)
		if action == "UpdateBot" {
			if display := chimeString(payload, "DisplayName", ""); display != "" {
				bot["DisplayName"] = display
			}
		}
		if action == "RegenerateSecurityToken" {
			bot["SecurityToken"] = fmt.Sprintf("stackyard-token-%06d", time.Now().Unix()%1000000)
		}
		return map[string]any{"Bot": chimeCloneMap(bot)}
	case "ListBots":
		items := make([]any, 0)
		prefix := accountID + ":"
		for key, bot := range s.bots {
			if strings.HasPrefix(key, prefix) {
				items = append(items, chimeCloneMap(bot))
			}
		}
		return map[string]any{"Bots": items, "NextToken": ""}
	case "PutEventsConfiguration", "GetEventsConfiguration":
		cfg := s.ensureEventsConfigLocked(accountID, botID)
		if action == "PutEventsConfiguration" {
			for key, value := range payload {
				cfg[key] = value
			}
		}
		return map[string]any{"EventsConfiguration": chimeCloneMap(cfg)}
	case "DeleteEventsConfiguration":
		delete(s.eventsConfigs, chimeBotKey(accountID, botID))
		return map[string]any{}
	case "CreatePhoneNumberOrder":
		id := fmt.Sprintf("order-%06d", s.nextPhoneOrderID)
		s.nextPhoneOrderID++
		order := s.ensurePhoneOrderLocked(id)
		return map[string]any{"PhoneNumberOrder": chimeCloneMap(order)}
	case "GetPhoneNumberOrder":
		return map[string]any{"PhoneNumberOrder": chimeCloneMap(s.ensurePhoneOrderLocked(phoneOrderID))}
	case "ListPhoneNumberOrders":
		items := make([]any, 0, len(s.phoneNumberOrders))
		for _, order := range chimeSortedValues(s.phoneNumberOrders) {
			items = append(items, chimeCloneMap(order))
		}
		return map[string]any{"PhoneNumberOrders": items, "NextToken": ""}
	case "GetPhoneNumber", "UpdatePhoneNumber":
		phone := s.ensurePhoneLocked(phoneNumberID)
		if action == "UpdatePhoneNumber" {
			for key, value := range payload {
				phone[key] = value
			}
		}
		return map[string]any{"PhoneNumber": chimeCloneMap(phone)}
	case "DeletePhoneNumber":
		delete(s.phoneNumbers, phoneNumberID)
		return map[string]any{}
	case "ListPhoneNumbers":
		items := make([]any, 0, len(s.phoneNumbers))
		for _, phone := range chimeSortedValues(s.phoneNumbers) {
			items = append(items, chimeCloneMap(phone))
		}
		return map[string]any{"PhoneNumbers": items, "NextToken": ""}
	case "BatchDeletePhoneNumber", "BatchUpdatePhoneNumber":
		return map[string]any{"PhoneNumberErrors": []any{}}
	case "RestorePhoneNumber":
		phone := s.ensurePhoneLocked(phoneNumberID)
		phone["Status"] = "AcquireSucceeded"
		return map[string]any{"PhoneNumber": chimeCloneMap(phone)}
	case "ListSupportedPhoneNumberCountries":
		return map[string]any{"PhoneNumberCountries": []any{map[string]any{"CountryCode": "US", "SupportedPhoneNumberTypes": []any{"Local", "TollFree"}}}}
	case "SearchAvailablePhoneNumbers":
		return map[string]any{"E164PhoneNumbers": []any{"+12065550101", "+12065550102"}}
	case "CreateRoom":
		id := fmt.Sprintf("room-%06d", s.nextRoomID)
		s.nextRoomID++
		room := s.ensureRoomLocked(accountID, id)
		if name := chimeString(payload, "Name", ""); name != "" {
			room["Name"] = name
		}
		return map[string]any{"Room": chimeCloneMap(room)}
	case "GetRoom", "UpdateRoom":
		room := s.ensureRoomLocked(accountID, roomID)
		if action == "UpdateRoom" {
			if name := chimeString(payload, "Name", ""); name != "" {
				room["Name"] = name
			}
		}
		return map[string]any{"Room": chimeCloneMap(room)}
	case "DeleteRoom":
		delete(s.rooms, chimeRoomKey(accountID, roomID))
		return map[string]any{}
	case "ListRooms":
		items := make([]any, 0)
		prefix := accountID + ":"
		for key, room := range s.rooms {
			if strings.HasPrefix(key, prefix) {
				items = append(items, chimeCloneMap(room))
			}
		}
		return map[string]any{"Rooms": items, "NextToken": ""}
	case "CreateRoomMembership", "BatchCreateRoomMembership", "UpdateRoomMembership":
		membership := s.ensureMembershipLocked(accountID, roomID, memberID)
		if role := chimeString(payload, "Role", ""); role != "" {
			membership["Role"] = role
		}
		if action == "BatchCreateRoomMembership" {
			return map[string]any{"Errors": []any{}, "Invites": []any{map[string]any{"InviteId": "invite-member", "Status": "Succeeded"}}}
		}
		return map[string]any{"RoomMembership": chimeCloneMap(membership)}
	case "DeleteRoomMembership":
		delete(s.memberships, chimeMemberKey(accountID, roomID, memberID))
		return map[string]any{}
	case "ListRoomMemberships":
		items := make([]any, 0)
		prefix := fmt.Sprintf("%s:%s:", accountID, roomID)
		for key, membership := range s.memberships {
			if strings.HasPrefix(key, prefix) {
				items = append(items, chimeCloneMap(membership))
			}
		}
		return map[string]any{"RoomMemberships": items, "NextToken": ""}
	case "PutRetentionSettings", "GetRetentionSettings":
		settings := s.ensureRetentionLocked(accountID)
		if action == "PutRetentionSettings" {
			for key, value := range payload {
				settings[key] = value
			}
		}
		return map[string]any{"RetentionSettings": chimeCloneMap(settings)}
	case "CreateMeetingDialOut":
		return map[string]any{"TransactionId": fmt.Sprintf("%s-dialout", meetingID)}
	case "RedactConversationMessage", "RedactRoomMessage":
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *chimeStore) ensureAccountLocked(accountID string) map[string]any {
	if accountID = strings.TrimSpace(accountID); accountID == "" {
		accountID = "acc-000001"
	}
	if account := s.accounts[accountID]; account != nil {
		return account
	}
	account := map[string]any{
		"AccountId":        accountID,
		"Name":             "stackyard-account",
		"AccountType":      "Team",
		"AwsAccountId":     "123456789012",
		"CreatedTimestamp": time.Now().UTC().Format(time.RFC3339),
	}
	s.accounts[accountID] = account
	return account
}

func (s *chimeStore) ensureAccountSettingsLocked(accountID string) map[string]any {
	if settings := s.accountSettings[accountID]; settings != nil {
		return settings
	}
	settings := map[string]any{"DisableRemoteControl": false, "EnableDialOut": true}
	s.accountSettings[accountID] = settings
	return settings
}

func (s *chimeStore) ensureUserLocked(accountID, userID string) map[string]any {
	key := chimeUserKey(accountID, userID)
	if user := s.users[key]; user != nil {
		return user
	}
	user := map[string]any{
		"UserId":                 userID,
		"AccountId":              accountID,
		"PrimaryEmail":           "user@example.com",
		"DisplayName":            "Stackyard User",
		"UserRegistrationStatus": "Registered",
		"UserInvitationStatus":   "Accepted",
	}
	s.users[key] = user
	return user
}

func (s *chimeStore) ensureUserSettingsLocked(accountID, userID string) map[string]any {
	key := chimeUserKey(accountID, userID)
	if settings := s.userSettings[key]; settings != nil {
		return settings
	}
	settings := map[string]any{"Telephony": map[string]any{"InboundCalling": true, "OutboundCalling": true}}
	s.userSettings[key] = settings
	return settings
}

func (s *chimeStore) ensureBotLocked(accountID, botID string) map[string]any {
	key := chimeBotKey(accountID, botID)
	if bot := s.bots[key]; bot != nil {
		return bot
	}
	now := time.Now().UTC().Format(time.RFC3339)
	bot := map[string]any{
		"BotId":            botID,
		"AccountId":        accountID,
		"DisplayName":      "stackyard-bot",
		"BotEmail":         "stackyard-bot@example.com",
		"BotType":          "ChatBot",
		"SecurityToken":    "stackyard-token",
		"Disabled":         false,
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.bots[key] = bot
	return bot
}

func (s *chimeStore) ensureEventsConfigLocked(accountID, botID string) map[string]any {
	key := chimeBotKey(accountID, botID)
	if cfg := s.eventsConfigs[key]; cfg != nil {
		return cfg
	}
	cfg := map[string]any{
		"BotId":             botID,
		"AccountId":         accountID,
		"LambdaFunctionArn": "arn:aws:lambda:us-east-1:123456789012:function:stackyard-chime-events",
	}
	s.eventsConfigs[key] = cfg
	return cfg
}

func (s *chimeStore) ensurePhoneLocked(phoneNumberID string) map[string]any {
	if phone := s.phoneNumbers[phoneNumberID]; phone != nil {
		return phone
	}
	phone := map[string]any{
		"PhoneNumberId":   phoneNumberID,
		"E164PhoneNumber": "+12065550100",
		"Status":          "AcquireSucceeded",
		"ProductType":     "BusinessCalling",
	}
	s.phoneNumbers[phoneNumberID] = phone
	return phone
}

func (s *chimeStore) ensurePhoneOrderLocked(orderID string) map[string]any {
	if order := s.phoneNumberOrders[orderID]; order != nil {
		return order
	}
	order := map[string]any{
		"PhoneNumberOrderId": orderID,
		"ProductType":        "BusinessCalling",
		"Status":             "Successful",
		"CreatedTimestamp":   time.Now().UTC().Format(time.RFC3339),
		"OrderedPhoneNumbers": []any{
			map[string]any{"E164PhoneNumber": "+12065550100", "Status": "Acquired"},
		},
	}
	s.phoneNumberOrders[orderID] = order
	return order
}

func (s *chimeStore) ensureRoomLocked(accountID, roomID string) map[string]any {
	key := chimeRoomKey(accountID, roomID)
	if room := s.rooms[key]; room != nil {
		return room
	}
	room := map[string]any{
		"RoomId":           roomID,
		"AccountId":        accountID,
		"Name":             "stackyard-room",
		"CreatedTimestamp": time.Now().UTC().Format(time.RFC3339),
	}
	s.rooms[key] = room
	return room
}

func (s *chimeStore) ensureMembershipLocked(accountID, roomID, memberID string) map[string]any {
	key := chimeMemberKey(accountID, roomID, memberID)
	if membership := s.memberships[key]; membership != nil {
		return membership
	}
	membership := map[string]any{
		"RoomId":           roomID,
		"MemberId":         memberID,
		"AccountId":        accountID,
		"Role":             "Member",
		"InvitedBy":        "user-000001",
		"UpdatedTimestamp": time.Now().UTC().Format(time.RFC3339),
	}
	s.memberships[key] = membership
	return membership
}

func (s *chimeStore) ensureRetentionLocked(accountID string) map[string]any {
	if settings := s.retentionSettings[accountID]; settings != nil {
		return settings
	}
	settings := map[string]any{
		"RoomRetentionSettings":         map[string]any{"RetentionDays": 30},
		"ConversationRetentionSettings": map[string]any{"RetentionDays": 30},
	}
	s.retentionSettings[accountID] = settings
	return settings
}

func chimeMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for k, v := range payload {
		out[k] = v
	}
	for k, v := range pathParams {
		out[k] = v
	}
	for k, values := range query {
		if len(values) > 0 {
			out[k] = values[len(values)-1]
		}
	}
	return out
}

func chimeString(m map[string]any, key, def string) string {
	if m == nil {
		return def
	}
	if value, ok := m[key]; ok {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return def
}

func chimeCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = chimeCloneAny(v)
	}
	return out
}

func chimeCloneAny(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return chimeCloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = chimeCloneAny(typed[i])
		}
		return out
	default:
		return typed
	}
}

func chimeSortedValues(items map[string]map[string]any) []map[string]any {
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, items[key])
	}
	return out
}

func chimeUserKey(accountID, userID string) string {
	return strings.TrimSpace(accountID) + ":" + strings.TrimSpace(userID)
}

func chimeBotKey(accountID, botID string) string {
	return strings.TrimSpace(accountID) + ":" + strings.TrimSpace(botID)
}

func chimeRoomKey(accountID, roomID string) string {
	return strings.TrimSpace(accountID) + ":" + strings.TrimSpace(roomID)
}

func chimeMemberKey(accountID, roomID, memberID string) string {
	return strings.TrimSpace(accountID) + ":" + strings.TrimSpace(roomID) + ":" + strings.TrimSpace(memberID)
}
