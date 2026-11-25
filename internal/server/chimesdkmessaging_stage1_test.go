package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const cmsgTestApp = "arn:aws:chime:us-east-1:123456789012:app-instance/app-1"

func cmsgUser(name string) string { return cmsgTestApp + "/user/" + name }

func TestChimeSDKMessagingStage1Channels(t *testing.T) {
	srv, ts := newMessagingTestServer(t)
	alice, bob, carol, mallory := cmsgUser("alice"), cmsgUser("bob"), cmsgUser("carol"), cmsgUser("mallory")
	eve := "arn:aws:chime:us-east-1:123456789012:app-instance/other/user/eve"

	create := `{"AppInstanceArn":"` + cmsgTestApp + `","Name":"ops","Mode":"RESTRICTED","Privacy":"PRIVATE","ClientRequestToken":"c-1","MemberArns":["` + bob + `"]}`
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/channels", create, "")
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPost, "/channels", create, eve)
	arn := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/channels", create, alice)["ChannelArn"].(string)
	if again := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/channels", create, alice); again["ChannelArn"] != arn {
		t.Fatalf("idempotent CreateChannel: %v", again)
	}
	ch := "/channels/" + chimeSDKEscape(arn)

	// Errors carry the exception in X-Amzn-ErrorType and the ErrorCode enum in Code.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+ch, nil, map[string]string{"X-Amz-Chime-Bearer": carol}, "chime")
	var wire struct{ Code, Message string }
	if err := json.Unmarshal(mustBody(t, resp), &wire); err != nil || resp.Header.Get("X-Amzn-ErrorType") != "ForbiddenException" || wire.Code != "Forbidden" || wire.Message == "" {
		t.Fatalf("error shape: %v %v %+v", err, resp.Header, wire)
	}
	// Private channels are visible to members only; eve isn't in the app instance.
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodGet, ch, "", carol)
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodGet, ch, "", eve)
	desc := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch, "", bob)["Channel"].(map[string]any)
	if desc["Mode"] != "RESTRICTED" || desc["CreatedBy"].(map[string]any)["Arn"] != alice {
		t.Fatalf("DescribeChannel: %v", desc)
	}
	// Restricted channels: only moderators add members or update the channel.
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPost, ch+"/memberships", `{"MemberArn":"`+carol+`","Type":"DEFAULT"}`, bob)
	chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, ch+"/memberships", `{"MemberArn":"`+carol+`","Type":"DEFAULT"}`, alice)
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPut, ch, `{"Name":"x"}`, bob)
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, ch, `{"Name":"ops-2","Mode":"UNRESTRICTED"}`, alice)
	// Unrestricted: members may add members.
	chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, ch+"/memberships", `{"MemberArn":"`+cmsgUser("dave")+`","Type":"HIDDEN"}`, bob)
	list := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/memberships", "", bob)
	if n := len(list["ChannelMemberships"].([]any)); n != 3 {
		t.Fatalf("ListChannelMemberships (DEFAULT): %d %v", n, list)
	}
	hidden := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/memberships?type=HIDDEN", "", bob)
	if n := len(hidden["ChannelMemberships"].([]any)); n != 1 {
		t.Fatalf("ListChannelMemberships (HIDDEN): %v", hidden)
	}

	// Moderators.
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPost, ch+"/moderators", `{"ChannelModeratorArn":"`+carol+`"}`, bob)
	chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, ch+"/moderators", `{"ChannelModeratorArn":"`+bob+`"}`, alice)
	mod := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/moderators/"+chimeSDKEscape(bob), "", carol)
	if mod["ChannelModerator"].(map[string]any)["CreatedBy"].(map[string]any)["Arn"] != alice {
		t.Fatalf("DescribeChannelModerator: %v", mod)
	}
	moderated := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/channels?scope=app-instance-user-moderated-channels", "", bob)
	if len(moderated["Channels"].([]any)) != 1 {
		t.Fatalf("ListChannelsModeratedByAppInstanceUser: %v", moderated)
	}
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodGet, "/channels?scope=app-instance-user-moderated-channels&app-instance-user-arn="+url.QueryEscape(alice), "", bob)
	chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"?scope=app-instance-user-moderated-channel&app-instance-user-arn="+url.QueryEscape(bob), "", bob)
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, ch+"/moderators/"+chimeSDKEscape(bob), "", alice)
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, ch+"?scope=app-instance-user-moderated-channel&app-instance-user-arn="+url.QueryEscape(bob), "", bob)

	mine := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/channels?scope=app-instance-user-memberships", "", carol)
	if len(mine["ChannelMemberships"].([]any)) != 1 {
		t.Fatalf("ListChannelMembershipsForAppInstanceUser: %v", mine)
	}

	// Preferences belong to the member.
	prefs := ch + "/memberships/" + chimeSDKEscape(bob) + "/preferences"
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPut, prefs, `{"Preferences":{"PushNotifications":{"AllowNotifications":"FILTERED"}}}`, bob)
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, prefs, `{"Preferences":{"PushNotifications":{"AllowNotifications":"NONE"}}}`, bob)
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodGet, prefs, "", alice)
	if got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, prefs, "", bob); got["Preferences"].(map[string]any)["PushNotifications"].(map[string]any)["AllowNotifications"] != "NONE" {
		t.Fatalf("GetChannelMembershipPreferences: %v", got)
	}

	// Search by members.
	search := chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, "/channels?operation=search", `{"Fields":[{"Key":"MEMBERS","Values":["`+bob+`","`+carol+`"],"Operator":"INCLUDES"}]}`, bob)
	if len(search["Channels"].([]any)) != 1 {
		t.Fatalf("SearchChannels INCLUDES: %v", search)
	}
	search = chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, "/channels?operation=search", `{"Fields":[{"Key":"MEMBERS","Values":["`+bob+`"],"Operator":"EQUALS"}]}`, bob)
	if len(search["Channels"].([]any)) != 0 {
		t.Fatalf("SearchChannels EQUALS: %v", search)
	}

	// Bans remove the member and keep it out.
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPost, ch+"/bans", `{"MemberArn":"`+carol+`"}`, bob)
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, ch+"/bans", `{"MemberArn":"`+alice+`"}`, alice)
	chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, ch+"/bans", `{"MemberArn":"`+carol+`"}`, alice)
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, ch+"/memberships/"+chimeSDKEscape(carol), "", alice)
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPost, ch+"/memberships", `{"MemberArn":"`+carol+`","Type":"DEFAULT"}`, alice)
	if bans := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/bans", "", alice); len(bans["ChannelBans"].([]any)) != 1 {
		t.Fatalf("ListChannelBans: %v", bans)
	}
	chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/bans/"+chimeSDKEscape(carol), "", alice)
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, ch+"/bans/"+chimeSDKEscape(carol), "", alice)
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodDelete, ch+"/bans/"+chimeSDKEscape(carol), "", alice)

	// Members leave; moderators remove others.
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodDelete, ch+"/memberships/"+chimeSDKEscape(cmsgUser("dave")), "", mallory)
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, ch+"/memberships/"+chimeSDKEscape(cmsgUser("dave")), "", cmsgUser("dave"))

	// Expiration settings delete the channel once they lapse.
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPut, ch+"/expiration-settings", `{"ExpirationSettings":{"ExpirationDays":1,"ExpirationCriterion":"CREATED_TIMESTAMP"}}`, bob)
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, ch+"/expiration-settings", `{"ExpirationSettings":{"ExpirationDays":1,"ExpirationCriterion":"CREATED_TIMESTAMP"}}`, alice)
	st := srv.chimeSDKMessaging
	st.mu.Lock()
	st.now = func() time.Time { return time.Now().Add(49 * time.Hour) }
	st.mu.Unlock()
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, ch, "", alice)
	st.mu.Lock()
	gone := st.channels[arn] == nil && len(st.userChannels[bob]) == 0 && st.memberships == 0
	st.now = time.Now
	st.mu.Unlock()
	if !gone {
		t.Fatalf("expired channel was not removed from the indexes")
	}
}

func TestChimeSDKMessagingStage1Messages(t *testing.T) {
	srv, ts := newMessagingTestServer(t)
	alice, bob, carol := cmsgUser("alice"), cmsgUser("bob"), cmsgUser("carol")
	arn := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/channels",
		`{"AppInstanceArn":"`+cmsgTestApp+`","Name":"chat","ClientRequestToken":"c-2","MemberArns":["`+bob+`"]}`, alice)["ChannelArn"].(string)
	ch := "/channels/" + chimeSDKEscape(arn)
	send := func(bearer, content, token string) map[string]any {
		return chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, ch+"/messages",
			`{"Content":"`+content+`","Type":"STANDARD","Persistence":"PERSISTENT","ClientRequestToken":"`+token+`"}`, bearer)
	}
	first := send(bob, "hello", "t-1")
	id := first["MessageId"].(string)
	if first["Status"].(map[string]any)["Value"] != "SENT" || send(bob, "hello", "t-1")["MessageId"] != id {
		t.Fatalf("SendChannelMessage: %v", first)
	}
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPost, ch+"/messages", `{"Content":"x","Type":"STANDARD","Persistence":"PERSISTENT","ClientRequestToken":"t-9"}`, carol)
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, ch+"/messages",
		`{"Content":"`+strings.Repeat("a", cmsgMaxContent+1)+`","Type":"STANDARD","Persistence":"PERSISTENT","ClientRequestToken":"t-8"}`, bob)
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, ch+"/messages",
		`{"Content":"`+strings.Repeat("a", 31)+`","Type":"CONTROL","Persistence":"NON_PERSISTENT","ClientRequestToken":"t-7"}`, bob)
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, ch+"/messages",
		`{"Content":"x","Type":"STANDARD","Persistence":"PERSISTENT","ClientRequestToken":"t-6","SubChannelId":"sub-1"}`, bob)
	// Non-persistent messages aren't stored.
	np := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, ch+"/messages", `{"Content":"typing","Type":"CONTROL","Persistence":"NON_PERSISTENT","ClientRequestToken":"t-5"}`, bob)
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, ch+"/messages/"+np["MessageId"].(string), "", bob)

	// The public channel is readable by any app instance user; only the sender edits.
	got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/messages/"+id, "", carol)["ChannelMessage"].(map[string]any)
	if got["Content"] != "hello" || got["Sender"].(map[string]any)["Arn"] != bob {
		t.Fatalf("GetChannelMessage: %v", got)
	}
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPut, ch+"/messages/"+id, `{"Content":"hijack"}`, alice)
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, ch+"/messages/"+id, `{"Content":"hello, edited"}`, bob)
	got = chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/messages/"+id, "", bob)["ChannelMessage"].(map[string]any)
	if got["Content"] != "hello, edited" || got["LastEditedTimestamp"] == nil {
		t.Fatalf("UpdateChannelMessage: %v", got)
	}
	for i := 2; i <= 4; i++ {
		send(alice, fmt.Sprintf("m%d", i), fmt.Sprintf("t-%d0", i))
	}
	page := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/messages?max-results=3", "", bob)
	msgs := page["ChannelMessages"].([]any)
	if len(msgs) != 3 || msgs[0].(map[string]any)["Content"] != "m4" || page["NextToken"] == nil {
		t.Fatalf("ListChannelMessages: %v", page)
	}
	rest := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/messages?max-results=3&next-token="+page["NextToken"].(string), "", bob)
	if r := rest["ChannelMessages"].([]any); len(r) != 1 || r[0].(map[string]any)["MessageId"] != id || rest["NextToken"] != nil {
		t.Fatalf("ListChannelMessages page 2: %v", rest)
	}
	asc := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/messages?sort-order=ASCENDING&max-results=1", "", bob)
	if asc["ChannelMessages"].([]any)[0].(map[string]any)["MessageId"] != id {
		t.Fatalf("ascending: %v", asc)
	}
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodGet, ch+"/messages?next-token=abc", "", bob)

	// Redact (sender or moderator) and delete (moderator).
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPost, ch+"/messages/"+id+"?operation=redact", "", carol)
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, ch+"/messages/"+id+"?operation=redact", "", alice)
	got = chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/messages/"+id, "", bob)["ChannelMessage"].(map[string]any)
	if got["Redacted"] != true || got["Content"] != nil {
		t.Fatalf("redacted: %v", got)
	}
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodDelete, ch+"/messages/"+id, "", bob)
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, ch+"/messages/"+id, "", alice)
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, ch+"/messages/"+id, "", bob)
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, ch+"/readMarker", "", bob)
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodPut, ch+"/readMarker", "", carol)

	// Stored messages per channel are capped; the oldest are evicted and their bytes released.
	for i := 0; i < cmsgMaxMessages+5; i++ {
		send(bob, "x", fmt.Sprintf("bulk-%d", i))
	}
	st := srv.chimeSDKMessaging
	st.mu.Lock()
	c := st.channels[arn]
	count, bytes := len(c.messages), st.bytes
	st.mu.Unlock()
	if count != cmsgMaxMessages || bytes != cmsgMaxMessages*cmsgMessageSize("x", "", "", nil) {
		t.Fatalf("eviction: %d messages, %d bytes", count, bytes)
	}

	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodDelete, ch, "", bob)
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, ch, "", alice)
	st.mu.Lock()
	bytes = st.bytes
	st.mu.Unlock()
	if bytes != 0 {
		t.Fatalf("deleted channel kept %d message bytes", bytes)
	}
}

func TestChimeSDKMessagingStage1FlowsAndElastic(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	alice, bob := cmsgUser("alice"), cmsgUser("bob")
	flowBody := `{"AppInstanceArn":"` + cmsgTestApp + `","Name":"profanity","ClientRequestToken":"f-1","Processors":[{"Name":"filter","ExecutionOrder":1,"FallbackAction":"ABORT",
		"Configuration":{"Lambda":{"ResourceArn":"arn:aws:lambda:us-east-1:123456789012:function:filter","InvocationType":"ASYNC"}}}]}`
	flow := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/channel-flows", flowBody, "")["ChannelFlowArn"].(string)
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/channel-flows", strings.Replace(strings.Replace(flowBody, "f-1", "f-2", 1), "arn:aws:lambda", "arn:aws:sqs", 1), "")
	arn := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/channels",
		`{"AppInstanceArn":"`+cmsgTestApp+`","Name":"flowed","ClientRequestToken":"c-3","MemberArns":["`+bob+`"]}`, alice)["ChannelArn"].(string)
	ch := "/channels/" + chimeSDKEscape(arn)
	chimeSDKFail(t, ts, http.StatusForbidden, "ForbiddenException", http.MethodPut, ch+"/channel-flow", `{"ChannelFlowArn":"`+flow+`"}`, bob)
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, ch+"/channel-flow", `{"ChannelFlowArn":"`+flow+`"}`, alice)
	assoc := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/channels?scope=channel-flow-associations&channel-flow-arn="+url.QueryEscape(flow), "", "")
	if len(assoc["Channels"].([]any)) != 1 {
		t.Fatalf("ListChannelsAssociatedWithChannelFlow: %v", assoc)
	}
	// Messages wait for the flow processor's callback.
	sent := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, ch+"/messages", `{"Content":"darn","Type":"STANDARD","Persistence":"PERSISTENT","ClientRequestToken":"t-1"}`, bob)
	id := sent["MessageId"].(string)
	if sent["Status"].(map[string]any)["Value"] != "PENDING" {
		t.Fatalf("flowed message: %v", sent)
	}
	status := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/messages/"+id+"?scope=message-status", "", bob)
	if status["Status"].(map[string]any)["Value"] != "PENDING" {
		t.Fatalf("GetChannelMessageStatus: %v", status)
	}
	cb := strings.Repeat("c", 32)
	callback := `{"CallbackId":"` + cb + `","ChannelMessage":{"MessageId":"` + id + `","Content":"d**n"}}`
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, ch+"?operation=channel-flow-callback", callback, "")
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, ch+"?operation=channel-flow-callback", callback, "")
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodPost, ch+"?operation=channel-flow-callback", strings.Replace(callback, cb, strings.Repeat("d", 32), 1), "")
	got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/messages/"+id, "", bob)["ChannelMessage"].(map[string]any)
	if got["Content"] != "d**n" || got["Status"].(map[string]any)["Value"] != "SENT" {
		t.Fatalf("processed message: %v", got)
	}
	denied := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, ch+"/messages", `{"Content":"spam","Type":"STANDARD","Persistence":"PERSISTENT","ClientRequestToken":"t-2"}`, bob)["MessageId"].(string)
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, ch+"?operation=channel-flow-callback", `{"CallbackId":"`+strings.Repeat("e", 32)+`","DeleteResource":true,"ChannelMessage":{"MessageId":"`+denied+`"}}`, "")
	if s := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ch+"/messages/"+denied+"?scope=message-status", "", bob); s["Status"].(map[string]any)["Value"] != "DENIED" {
		t.Fatalf("denied: %v", s)
	}
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodDelete, "/channel-flows/"+chimeSDKEscape(flow), "", "")
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, ch+"/channel-flow/"+chimeSDKEscape(flow), "", alice)
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, "/channel-flows/"+chimeSDKEscape(flow), "", "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, "/channel-flows/"+chimeSDKEscape(flow), "", "")

	// Elastic channels spread members over sub-channels up to their maximum.
	elastic := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/channels", `{"AppInstanceArn":"`+cmsgTestApp+`","Name":"big","ClientRequestToken":"c-4",
		"ElasticChannelConfiguration":{"MaximumSubChannels":2,"TargetMembershipsPerSubChannel":2,"MinimumMembershipPercentage":10}}`, alice)["ChannelArn"].(string)
	ech := "/channels/" + chimeSDKEscape(elastic)
	var members []string
	for i := 0; i < 5; i++ {
		members = append(members, `"`+cmsgUser(fmt.Sprintf("u%d", i))+`"`)
	}
	batch := chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, ech+"/memberships?operation=batch-create", `{"MemberArns":[`+strings.Join(members, ",")+`]}`, alice)
	if errs := batch["Errors"].([]any); len(errs) != 1 || errs[0].(map[string]any)["ErrorCode"] != "ResourceLimitExceeded" {
		t.Fatalf("BatchCreateChannelMembership: %v", batch)
	}
	subs := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ech+"/subchannels", "", alice)["SubChannels"].([]any)
	if len(subs) != 2 || subs[0].(map[string]any)["MembershipCount"] != float64(2) {
		t.Fatalf("ListSubChannels: %v", subs)
	}
	u0 := cmsgUser("u0")
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, ech+"/messages", `{"Content":"x","Type":"STANDARD","Persistence":"PERSISTENT","ClientRequestToken":"t-3"}`, u0)
	m := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ech+"/memberships/"+chimeSDKEscape(u0), "", u0)["ChannelMembership"].(map[string]any)
	sub := m["SubChannelId"].(string)
	chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, ech+"/messages", `{"Content":"x","Type":"STANDARD","Persistence":"PERSISTENT","ClientRequestToken":"t-3","SubChannelId":"`+sub+`"}`, u0)
	if msgs := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, ech+"/messages?sub-channel-id="+sub, "", u0); len(msgs["ChannelMessages"].([]any)) != 1 {
		t.Fatalf("sub-channel messages: %v", msgs)
	}

	// Streaming configurations.
	sc := "/app-instances/" + chimeSDKEscape(cmsgTestApp) + "/streaming-configurations"
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, sc, "", "")
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPut, sc, `{"StreamingConfigurations":[{"DataType":"Channel","ResourceArn":"arn:aws:sqs:us-east-1:123456789012:q"}]}`, "")
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, sc, `{"StreamingConfigurations":[{"DataType":"Channel","ResourceArn":"arn:aws:kinesis:us-east-1:123456789012:stream/chat"}]}`, "")
	if got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, sc, "", ""); len(got["StreamingConfigurations"].([]any)) != 1 {
		t.Fatalf("GetMessagingStreamingConfigurations: %v", got)
	}
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, sc, "", "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, sc, "", "")
	if ep := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/endpoints/messaging-session", "", ""); ep["Endpoint"].(map[string]any)["Url"] == nil {
		t.Fatalf("GetMessagingSessionEndpoint: %v", ep)
	}
}
