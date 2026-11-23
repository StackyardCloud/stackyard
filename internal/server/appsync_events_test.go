package server

import (
	"net/url"
	"strings"
	"testing"
)

func TestAppSyncEventsAPILifecycle(t *testing.T) {
	st := newAppSyncStore()
	modes := []any{map[string]any{"authType": "API_KEY"}}
	cfg := map[string]any{
		"authProviders": modes, "connectionAuthModes": modes,
		"defaultPublishAuthModes": modes, "defaultSubscribeAuthModes": modes,
	}
	call := func(action string, payload map[string]any, params map[string]string) (map[string]any, *appSyncEventsError) {
		return st.handleEvents(action, payload, params, url.Values{})
	}

	if _, err := call("CreateApi", map[string]any{"name": "bad/name", "eventConfig": cfg}, nil); err == nil || err.code != "BadRequestException" {
		t.Fatalf("invalid name: got %+v", err)
	}
	if _, err := call("CreateApi", map[string]any{"name": "ok"}, nil); err == nil {
		t.Fatal("missing eventConfig accepted")
	}
	out, err := call("CreateApi", map[string]any{"name": "events", "eventConfig": cfg, "tags": map[string]any{"k": "v"}}, nil)
	if err != nil {
		t.Fatalf("CreateApi: %+v", err)
	}
	api := out["api"].(map[string]any)
	id := api["apiId"].(string)
	if api["tags"].(map[string]string)["k"] != "v" || api["dns"] == nil {
		t.Fatalf("api shape: %#v", api)
	}

	p := map[string]string{"apiId": id}
	if _, err := call("CreateChannelNamespace", map[string]any{"name": "chat"}, p); err != nil {
		t.Fatalf("CreateChannelNamespace: %+v", err)
	}
	if _, err := call("CreateChannelNamespace", map[string]any{"name": "chat"}, p); err == nil || err.code != "ConflictException" {
		t.Fatalf("duplicate namespace: got %+v", err)
	}
	list, err := call("ListChannelNamespaces", nil, p)
	if err != nil || len(list["channelNamespaces"].([]any)) != 1 {
		t.Fatalf("ListChannelNamespaces: %#v %+v", list, err)
	}
	if _, err := call("UpdateApi", map[string]any{"name": "renamed", "eventConfig": cfg}, p); err != nil {
		t.Fatalf("UpdateApi: %+v", err)
	}
	if _, err := call("DeleteApi", nil, p); err != nil {
		t.Fatalf("DeleteApi: %+v", err)
	}
	if _, err := call("GetChannelNamespace", nil, map[string]string{"apiId": id, "name": "chat"}); err == nil || err.code != "NotFoundException" {
		t.Fatalf("namespace survived api delete: %+v", err)
	}
	if _, err := st.handleEvents("ListApis", nil, nil, url.Values{"maxResults": {"26"}}); err == nil {
		t.Fatal("maxResults over 25 accepted")
	}
}

func TestAppSyncRouteExactBeforeGreedy(t *testing.T) {
	cases := map[string]string{
		"GET /v2/apis":                         "ListApis",
		"GET /v2/apis/a1":                      "GetApi",
		"GET /v2/apis/a1/channelNamespaces":    "ListChannelNamespaces",
		"GET /v2/apis/a1/channelNamespaces/n":  "GetChannelNamespace",
		"POST /v2/apis/a1/channelNamespaces/n": "UpdateChannelNamespace",
	}
	for in, want := range cases {
		method, path, _ := strings.Cut(in, " ")
		if got, _ := parseAppSyncRoute(method, path); got != want {
			t.Errorf("%s: got %q, want %q", in, got, want)
		}
	}
	got, params := parseAppSyncRoute("GET", "/v1/tags/arn:aws:appsync:us-east-1:123456789012:apis/x/channelNamespace/n")
	if got != "ListTagsForResource" || !strings.HasSuffix(params["resourceArn"], "/channelNamespace/n") {
		t.Errorf("tag route: %q %v", got, params)
	}
}
