package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestChimeStage1AccountAndSettingsLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := chimeRequest(t, ts, http.MethodPost, "/accounts", []byte(`{"Name":"stackyard-stage1-account"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeChimePayload(t, resp)
	account := chimePayloadMap(createPayload, "Account")
	accountID := chimePayloadString(account, "AccountId")
	if accountID == "" {
		t.Fatalf("expected CreateAccount to return Account.AccountId")
	}

	resp = chimeRequest(t, ts, http.MethodGet, "/accounts/"+url.PathEscape(accountID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/"+url.PathEscape(accountID), []byte(`{"Name":"stackyard-stage1-account-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/accounts", nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeChimePayload(t, resp)
	accounts, ok := listPayload["Accounts"].([]any)
	if !ok || len(accounts) == 0 {
		t.Fatalf("expected ListAccounts to return Accounts")
	}

	resp = chimeRequest(t, ts, http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/settings", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPut, "/accounts/"+url.PathEscape(accountID)+"/settings", []byte(`{"DisableRemoteControl":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/settings", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPut, "/settings", []byte(`{"BusinessCalling":false}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodDelete, "/accounts/"+url.PathEscape(accountID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestChimeStage2UserLifecycleAndActions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users?operation=create", []byte(`{"Email":"stage2-user@example.com","DisplayName":"Stage2 User"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeChimePayload(t, resp)
	user := chimePayloadMap(createPayload, "User")
	userID := chimePayloadString(user, "UserId")
	if userID == "" {
		t.Fatalf("expected CreateUser to return User.UserId")
	}

	escapedUserID := url.PathEscape(userID)
	resp = chimeRequest(t, ts, http.MethodGet, "/accounts/acc-000001/users/"+escapedUserID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users/"+escapedUserID, []byte(`{"PrimaryEmail":"stage2-user-updated@example.com"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/accounts/acc-000001/users", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/accounts/acc-000001/users/"+escapedUserID+"/settings", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPut, "/accounts/acc-000001/users/"+escapedUserID+"/settings", []byte(`{"Telephony":{"InboundCalling":false}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users?operation=add", []byte(`{"UserEmailList":["invitee@example.com"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users", []byte(`{"UpdateUserRequestItems":[]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users?operation=suspend", []byte(`{"UserIdList":["`+userID+`"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users?operation=unsuspend", []byte(`{"UserIdList":["`+userID+`"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users/"+escapedUserID+"?operation=logout", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users/"+escapedUserID+"?operation=reset-personal-pin", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users/"+escapedUserID+"?operation=associate-phone-number", []byte(`{"E164PhoneNumber":"+12065550100"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/users/"+escapedUserID+"?operation=disassociate-phone-number", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestChimeStage3BotAndEventsLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/bots", []byte(`{"DisplayName":"stage3-bot"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeChimePayload(t, resp)
	bot := chimePayloadMap(createPayload, "Bot")
	botID := chimePayloadString(bot, "BotId")
	if botID == "" {
		t.Fatalf("expected CreateBot to return Bot.BotId")
	}

	escapedBotID := url.PathEscape(botID)
	resp = chimeRequest(t, ts, http.MethodGet, "/accounts/acc-000001/bots/"+escapedBotID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/bots/"+escapedBotID, []byte(`{"DisplayName":"stage3-bot-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/bots/"+escapedBotID+"?operation=regenerate-security-token", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/accounts/acc-000001/bots", nil)
	assertStatus(t, resp, http.StatusOK)

	eventsPath := "/accounts/acc-000001/bots/" + escapedBotID + "/events-configuration"
	resp = chimeRequest(t, ts, http.MethodPut, eventsPath, []byte(`{"LambdaFunctionArn":"arn:aws:lambda:us-east-1:123456789012:function:stage3-events"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, eventsPath, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodDelete, eventsPath, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestChimeStage4PhoneAndOrderLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := chimeRequest(t, ts, http.MethodPost, "/phone-number-orders", []byte(`{"ProductType":"BusinessCalling"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeChimePayload(t, resp)
	order := chimePayloadMap(createPayload, "PhoneNumberOrder")
	orderID := chimePayloadString(order, "PhoneNumberOrderId")
	if orderID == "" {
		t.Fatalf("expected CreatePhoneNumberOrder to return PhoneNumberOrder.PhoneNumberOrderId")
	}

	resp = chimeRequest(t, ts, http.MethodGet, "/phone-number-orders/"+url.PathEscape(orderID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/phone-number-orders", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/phone-numbers/phone-number-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/phone-numbers/phone-number-000001", []byte(`{"ProductType":"BusinessCalling"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/phone-numbers", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/phone-numbers?operation=batch-update", []byte(`{"UpdatePhoneNumberRequestItems":[]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/phone-numbers?operation=batch-delete", []byte(`{"PhoneNumberIds":["phone-number-000001"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/phone-numbers/phone-number-000001?operation=restore", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/phone-number-countries", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/search?type=phone-numbers", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestChimeStage5RoomMembershipAndRedaction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/rooms", []byte(`{"Name":"stage5-room"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeChimePayload(t, resp)
	room := chimePayloadMap(createPayload, "Room")
	roomID := chimePayloadString(room, "RoomId")
	if roomID == "" {
		t.Fatalf("expected CreateRoom to return Room.RoomId")
	}

	escapedRoomID := url.PathEscape(roomID)
	resp = chimeRequest(t, ts, http.MethodGet, "/accounts/acc-000001/rooms/"+escapedRoomID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/rooms/"+escapedRoomID, []byte(`{"Name":"stage5-room-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/accounts/acc-000001/rooms", nil)
	assertStatus(t, resp, http.StatusOK)

	membershipBase := "/accounts/acc-000001/rooms/" + escapedRoomID + "/memberships"
	resp = chimeRequest(t, ts, http.MethodPost, membershipBase, []byte(`{"Role":"Member"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, membershipBase+"/member-000001", []byte(`{"Role":"Administrator"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, membershipBase+"?operation=batch-create", []byte(`{"MembershipItemList":[]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, membershipBase, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/rooms/"+escapedRoomID+"/messages/message-000001?operation=redact", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/accounts/acc-000001/conversations/conversation-000001/messages/message-000001?operation=redact", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodPost, "/meetings/meeting-000001/dial-outs", []byte(`{"ToPhoneNumber":"+12065550100"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodDelete, membershipBase+"/member-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodDelete, "/accounts/acc-000001/rooms/"+escapedRoomID, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestChimeStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := chimeRequest(t, ts, http.MethodPost, "/unknown-chime-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/accounts",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"chime",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = chimeRequest(t, ts, http.MethodDelete, "/accounts/acc-000001/rooms/room-000001/memberships/member-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = chimeRequest(t, ts, http.MethodDelete, "/accounts/acc-000001/rooms/room-000001/memberships/member-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = chimeRequest(t, ts, http.MethodGet, "/accounts", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = chimeRequest(t, ts, http.MethodGet, "/accounts", nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeChimePayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func chimePayloadMap(payload map[string]any, key string) map[string]any {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return map[string]any{}
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func chimePayloadString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
