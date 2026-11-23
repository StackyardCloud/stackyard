package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const vpTestSchema = `{"Photo": {
  "entityTypes": {
    "User": {"memberOfTypes": ["Group"], "shape": {"type": "Record", "attributes": {"department": {"type": "String"}}}},
    "Group": {},
    "Album": {},
    "Image": {"memberOfTypes": ["Album"], "shape": {"type": "Record", "attributes": {"owner": {"type": "Entity", "name": "User"}}}}
  },
  "actions": {
    "read": {},
    "view": {"memberOf": [{"id": "read"}], "appliesTo": {"principalTypes": ["User"], "resourceTypes": ["Image"]}},
    "edit": {"appliesTo": {"principalTypes": ["User"], "resourceTypes": ["Image"]}}
  }
}}`

func vpJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func vpNewStore(t *testing.T, ts *httptest.Server, mode string) string {
	t.Helper()
	return vpOK(t, ts, "CreatePolicyStore", `{"validationSettings":{"mode":"`+mode+`"}}`)["policyStoreId"].(string)
}

func vpStatic(t *testing.T, ts *httptest.Server, store, statement string) string {
	t.Helper()
	return vpOK(t, ts, "CreatePolicy", vpJSON(map[string]any{"policyStoreId": store, "definition": map[string]any{"static": map[string]any{"statement": statement}}}))["policyId"].(string)
}

func vpAuthz(t *testing.T, ts *httptest.Server, store, principal, action, resource string, entities []any) map[string]any {
	t.Helper()
	ent := func(s string) map[string]any {
		i := strings.LastIndex(s, ":")
		return map[string]any{"entityType": s[:i], "entityId": s[i+1:]}
	}
	a := ent(action)
	return vpOK(t, ts, "IsAuthorized", vpJSON(map[string]any{
		"policyStoreId": store, "principal": ent(principal), "resource": ent(resource),
		"action":   map[string]any{"actionType": a["entityType"], "actionId": a["entityId"]},
		"entities": map[string]any{"entityList": entities},
	}))
}

func TestVerifiedPermissionsStage1PolicyStoresAndSchema(t *testing.T) {
	_, ts := newVPTestServer(t)
	created := vpOK(t, ts, "CreatePolicyStore", `{"validationSettings":{"mode":"OFF"},"clientToken":"tok-1","description":"photos","tags":{"env":"dev"}}`)
	id := created["policyStoreId"].(string)
	if again := vpOK(t, ts, "CreatePolicyStore", `{"validationSettings":{"mode":"OFF"},"clientToken":"tok-1"}`); again["policyStoreId"] != id {
		t.Fatalf("clientToken isn't idempotent: %v", again)
	}
	got := vpOK(t, ts, "GetPolicyStore", `{"policyStoreId":"`+id+`","tags":true}`)
	if got["description"] != "photos" || got["deletionProtection"] != "DISABLED" || got["tags"].(map[string]any)["env"] != "dev" {
		t.Fatalf("GetPolicyStore: %v", got)
	}
	if code := vpErrCode(t, ts, "GetSchema", `{"policyStoreId":"`+id+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing schema: %s", code)
	}
	for _, bad := range []string{`not json`, `{"Photo":{"entityTypes":{}}}`, `{"Photo":{"entityTypes":{"User":{"memberOfTypes":["Nope"]}},"actions":{}}}`,
		`{"Photo":{"entityTypes":{},"actions":{"view":{"appliesTo":{"principalTypes":["User"]}}}}}`, `{"Bad Name":{"entityTypes":{},"actions":{}}}`} {
		if code := vpErrCode(t, ts, "PutSchema", vpJSON(map[string]any{"policyStoreId": id, "definition": map[string]any{"cedarJson": bad}})); code != "ValidationException" {
			t.Errorf("schema %s: %s", bad, code)
		}
	}
	put := vpOK(t, ts, "PutSchema", vpJSON(map[string]any{"policyStoreId": id, "definition": map[string]any{"cedarJson": vpTestSchema}}))
	if fmt.Sprint(put["namespaces"]) != "[Photo]" {
		t.Fatalf("PutSchema namespaces: %v", put)
	}
	if vpOK(t, ts, "GetSchema", `{"policyStoreId":"`+id+`"}`)["schema"] != vpTestSchema {
		t.Fatal("GetSchema doesn't return the schema")
	}
	// Deletion protection blocks deletes; deleting a missing store succeeds.
	vpOK(t, ts, "UpdatePolicyStore", `{"policyStoreId":"`+id+`","validationSettings":{"mode":"OFF"},"deletionProtection":"ENABLED"}`)
	if code := vpErrCode(t, ts, "DeletePolicyStore", `{"policyStoreId":"`+id+`"}`); code != "InvalidStateException" {
		t.Fatalf("protected delete: %s", code)
	}
	vpOK(t, ts, "UpdatePolicyStore", `{"policyStoreId":"`+id+`","validationSettings":{"mode":"OFF"},"deletionProtection":"DISABLED"}`)
	vpOK(t, ts, "DeletePolicyStore", `{"policyStoreId":"`+id+`"}`)
	vpOK(t, ts, "DeletePolicyStore", `{"policyStoreId":"`+id+`"}`)
	if code := vpErrCode(t, ts, "GetPolicyStore", `{"policyStoreId":"`+id+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("deleted store: %s", code)
	}
}

func TestVerifiedPermissionsStage1StrictValidation(t *testing.T) {
	_, ts := newVPTestServer(t)
	store := vpNewStore(t, ts, "STRICT")
	stmt := `permit(principal == Photo::User::"alice", action == Photo::Action::"view", resource);`
	body := vpJSON(map[string]any{"policyStoreId": store, "definition": map[string]any{"static": map[string]any{"statement": stmt}}})
	if code := vpErrCode(t, ts, "CreatePolicy", body); code != "ValidationException" {
		t.Fatalf("STRICT without schema: %s", code)
	}
	vpOK(t, ts, "PutSchema", vpJSON(map[string]any{"policyStoreId": store, "definition": map[string]any{"cedarJson": vpTestSchema}}))
	vpOK(t, ts, "CreatePolicy", body)
	for _, bad := range []string{
		`permit(principal == Photo::Robot::"r2", action, resource);`,                             // unknown type
		`permit(principal, action == Photo::Action::"delete", resource);`,                        // unknown action
		`permit(principal == Photo::Group::"g", action == Photo::Action::"view", resource);`,     // view doesn't apply to Group
		`permit(principal, action, resource) when { resource == Photo::Robot::"x" };`,            // unknown type in a condition
		`permit(principal, action == Photo::Action::"read", resource == Photo::Album::"trips");`, // read's members apply to Image, not Album
	} {
		b := vpJSON(map[string]any{"policyStoreId": store, "definition": map[string]any{"static": map[string]any{"statement": bad}}})
		if code := vpErrCode(t, ts, "CreatePolicy", b); code != "ValidationException" {
			t.Errorf("STRICT accepted %s (%s)", bad, code)
		}
	}
	// Template links are checked too: the slot value's type must be declared.
	tmpl := vpOK(t, ts, "CreatePolicyTemplate", vpJSON(map[string]any{"policyStoreId": store, "statement": `permit(principal == ?principal, action == Photo::Action::"view", resource);`}))["policyTemplateId"].(string)
	linkBody := func(typ string) string {
		return vpJSON(map[string]any{"policyStoreId": store, "definition": map[string]any{"templateLinked": map[string]any{"policyTemplateId": tmpl, "principal": map[string]any{"entityType": typ, "entityId": "x"}}}})
	}
	if code := vpErrCode(t, ts, "CreatePolicy", linkBody("Photo::Robot")); code != "ValidationException" {
		t.Fatalf("STRICT link with an undeclared type: %s", code)
	}
	vpOK(t, ts, "CreatePolicy", linkBody("Photo::User"))
	// read is a group containing view, so an Image resource is applicable.
	vpStatic(t, ts, store, `permit(principal, action in Photo::Action::"read", resource is Photo::Image);`)
	// Switching an OFF store with a non-conforming policy to STRICT fails.
	off := vpNewStore(t, ts, "OFF")
	vpStatic(t, ts, off, `permit(principal == Nope::"x", action, resource);`)
	vpOK(t, ts, "PutSchema", vpJSON(map[string]any{"policyStoreId": off, "definition": map[string]any{"cedarJson": vpTestSchema}}))
	if code := vpErrCode(t, ts, "UpdatePolicyStore", `{"policyStoreId":"`+off+`","validationSettings":{"mode":"STRICT"}}`); code != "ValidationException" {
		t.Fatalf("STRICT switch with a bad policy: %s", code)
	}
}

func TestVerifiedPermissionsStage1PoliciesAndAuthorization(t *testing.T) {
	_, ts := newVPTestServer(t)
	store := vpNewStore(t, ts, "OFF")
	vpOK(t, ts, "PutSchema", vpJSON(map[string]any{"policyStoreId": store, "definition": map[string]any{"cedarJson": vpTestSchema}}))
	if code := vpErrCode(t, ts, "CreatePolicy", vpJSON(map[string]any{"policyStoreId": store, "definition": map[string]any{"static": map[string]any{"statement": "permit(principal, action);"}}})); code != "ValidationException" {
		t.Fatalf("syntax error: %s", code)
	}
	permit := vpStatic(t, ts, store, `permit(principal in Photo::Group::"admins", action in Photo::Action::"read", resource) when { principal.department == "eng" };`)
	forbid := vpStatic(t, ts, store, `forbid(principal, action, resource == Photo::Image::"secret");`)
	got := vpOK(t, ts, "GetPolicy", `{"policyStoreId":"`+store+`","policyId":"`+permit+`"}`)
	if got["effect"] != "Permit" || got["principal"].(map[string]any)["entityId"] != "admins" || got["policyType"] != "STATIC" {
		t.Fatalf("GetPolicy: %v", got)
	}
	alice := map[string]any{"identifier": map[string]any{"entityType": "Photo::User", "entityId": "alice"},
		"attributes": map[string]any{"department": map[string]any{"string": "eng"}},
		"parents":    []any{map[string]any{"entityType": "Photo::Group", "entityId": "admins"}}}
	// view is a member of read through the schema, so the permit applies.
	res := vpAuthz(t, ts, store, "Photo::User:alice", "Photo::Action:view", "Photo::Image:beach", []any{alice})
	if res["decision"] != "ALLOW" || res["determiningPolicies"].([]any)[0].(map[string]any)["policyId"] != permit {
		t.Fatalf("allow: %v", res)
	}
	// The forbid overrides it on the secret image.
	res = vpAuthz(t, ts, store, "Photo::User:alice", "Photo::Action:view", "Photo::Image:secret", []any{alice})
	if res["decision"] != "DENY" || res["determiningPolicies"].([]any)[0].(map[string]any)["policyId"] != forbid {
		t.Fatalf("forbid: %v", res)
	}
	// Without the group membership nothing applies: default deny.
	res = vpAuthz(t, ts, store, "Photo::User:bob", "Photo::Action:view", "Photo::Image:beach", nil)
	if res["decision"] != "DENY" || len(res["determiningPolicies"].([]any)) != 0 || len(res["errors"].([]any)) != 0 {
		t.Fatalf("default deny: %v", res)
	}
	// In the group but with no department: the condition errors, the policy
	// is skipped and the error reported.
	bob := map[string]any{"identifier": map[string]any{"entityType": "Photo::User", "entityId": "bob"},
		"parents": []any{map[string]any{"entityType": "Photo::Group", "entityId": "admins"}}}
	res = vpAuthz(t, ts, store, "Photo::User:bob", "Photo::Action:view", "Photo::Image:beach", []any{bob})
	if res["decision"] != "DENY" || len(res["errors"].([]any)) != 1 {
		t.Fatalf("erroring policy: %v", res)
	}
	// Batch evaluation shares the entity list.
	batch := vpOK(t, ts, "BatchIsAuthorized", vpJSON(map[string]any{"policyStoreId": store, "entities": map[string]any{"entityList": []any{alice}},
		"requests": []any{
			map[string]any{"principal": alice["identifier"], "action": map[string]any{"actionType": "Photo::Action", "actionId": "view"}, "resource": map[string]any{"entityType": "Photo::Image", "entityId": "beach"}},
			map[string]any{"principal": alice["identifier"], "action": map[string]any{"actionType": "Photo::Action", "actionId": "edit"}, "resource": map[string]any{"entityType": "Photo::Image", "entityId": "beach"}},
		}}))
	results := batch["results"].([]any)
	if len(results) != 2 || results[0].(map[string]any)["decision"] != "ALLOW" || results[1].(map[string]any)["decision"] != "DENY" {
		t.Fatalf("batch: %v", batch)
	}
	// Updates may change conditions and actions but not the effect.
	if code := vpErrCode(t, ts, "UpdatePolicy", vpJSON(map[string]any{"policyStoreId": store, "policyId": permit, "definition": map[string]any{"static": map[string]any{"statement": `forbid(principal in Photo::Group::"admins", action, resource);`}}})); code != "ValidationException" {
		t.Fatalf("effect change: %s", code)
	}
	vpOK(t, ts, "UpdatePolicy", vpJSON(map[string]any{"policyStoreId": store, "policyId": permit, "definition": map[string]any{"static": map[string]any{"statement": `permit(principal in Photo::Group::"admins", action, resource);`}}}))
	// Listing and filtering.
	list := vpOK(t, ts, "ListPolicies", vpJSON(map[string]any{"policyStoreId": store, "filter": map[string]any{"resource": map[string]any{"unspecified": true}}}))
	if len(list["policies"].([]any)) != 1 {
		t.Fatalf("filter resource unspecified: %v", list)
	}
	page := vpOK(t, ts, "ListPolicies", `{"policyStoreId":"`+store+`","maxResults":1}`)
	if len(page["policies"].([]any)) != 1 || page["nextToken"] == nil {
		t.Fatalf("paging: %v", page)
	}
	bg := vpOK(t, ts, "BatchGetPolicy", vpJSON(map[string]any{"requests": []any{map[string]any{"policyStoreId": store, "policyId": permit}, map[string]any{"policyStoreId": store, "policyId": "missing"}}}))
	if len(bg["results"].([]any)) != 1 || bg["errors"].([]any)[0].(map[string]any)["code"] != "POLICY_NOT_FOUND" {
		t.Fatalf("BatchGetPolicy: %v", bg)
	}
	vpOK(t, ts, "DeletePolicy", `{"policyStoreId":"`+store+`","policyId":"`+forbid+`"}`)
	if code := vpErrCode(t, ts, "GetPolicy", `{"policyStoreId":"`+store+`","policyId":"`+forbid+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("deleted policy: %s", code)
	}
}

func TestVerifiedPermissionsStage1TemplatesAndAliases(t *testing.T) {
	_, ts := newVPTestServer(t)
	store := vpNewStore(t, ts, "OFF")
	if code := vpErrCode(t, ts, "CreatePolicyTemplate", `{"policyStoreId":"`+store+`","statement":"permit(principal, action, resource);"}`); code != "ValidationException" {
		t.Fatalf("template without placeholders: %s", code)
	}
	tmpl := vpOK(t, ts, "CreatePolicyTemplate", `{"policyStoreId":"`+store+`","statement":"permit(principal == ?principal, action, resource in ?resource);"}`)["policyTemplateId"].(string)
	link := func(principal, resource map[string]any) (int, map[string]any) {
		tl := map[string]any{"policyTemplateId": tmpl}
		if principal != nil {
			tl["principal"] = principal
		}
		if resource != nil {
			tl["resource"] = resource
		}
		return vpCall(t, ts, "CreatePolicy", vpJSON(map[string]any{"policyStoreId": store, "definition": map[string]any{"templateLinked": tl}}))
	}
	alice := map[string]any{"entityType": "User", "entityId": "alice"}
	album := map[string]any{"entityType": "Album", "entityId": "trips"}
	if status, _ := link(alice, nil); status == 200 {
		t.Fatal("a link that leaves ?resource empty was accepted")
	}
	status, linked := link(alice, album)
	if status != 200 || linked["policyType"] != "TEMPLATE_LINKED" {
		t.Fatalf("link: %d %v", status, linked)
	}
	photo := []any{map[string]any{"identifier": map[string]any{"entityType": "Photo", "entityId": "p1"}, "parents": []any{album}}}
	if res := vpAuthz(t, ts, store, "User:alice", "Action:view", "Photo:p1", photo); res["decision"] != "ALLOW" {
		t.Fatalf("linked policy: %v", res)
	}
	// Updating the template changes what linked policies do.
	vpOK(t, ts, "UpdatePolicyTemplate", `{"policyStoreId":"`+store+`","policyTemplateId":"`+tmpl+`","statement":"permit(principal == ?principal, action, resource in ?resource) when { false };"}`)
	if res := vpAuthz(t, ts, store, "User:alice", "Action:view", "Photo:p1", photo); res["decision"] != "DENY" {
		t.Fatalf("after template update: %v", res)
	}
	// Aliases resolve to the store anywhere a policyStoreId is taken.
	vpOK(t, ts, "CreatePolicyStoreAlias", `{"aliasName":"photos","policyStoreId":"`+store+`"}`)
	if code := vpErrCode(t, ts, "CreatePolicyStoreAlias", `{"aliasName":"photos","policyStoreId":"`+store+`"}`); code != "ConflictException" {
		t.Fatalf("duplicate alias: %s", code)
	}
	if got := vpOK(t, ts, "GetPolicyStore", `{"policyStoreId":"policy-store-alias/photos"}`); got["policyStoreId"] != store {
		t.Fatalf("alias resolution: %v", got)
	}
	// Deleting the template deletes its linked policies.
	vpOK(t, ts, "DeletePolicyTemplate", `{"policyStoreId":"`+store+`","policyTemplateId":"`+tmpl+`"}`)
	if n := len(vpOK(t, ts, "ListPolicies", `{"policyStoreId":"`+store+`"}`)["policies"].([]any)); n != 0 {
		t.Fatalf("linked policies after template delete: %d", n)
	}
	vpOK(t, ts, "DeletePolicyStoreAlias", `{"aliasName":"photos"}`)
	if code := vpErrCode(t, ts, "GetPolicyStoreAlias", `{"aliasName":"photos"}`); code != "ResourceNotFoundException" {
		t.Fatalf("deleted alias: %s", code)
	}
}

func vpTestJWT(claims map[string]any) string {
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]any{"alg": "RS256"}) + "." + enc(claims) + ".c2ln"
}

func TestVerifiedPermissionsStage1TokenAuthorization(t *testing.T) {
	srv, ts := newVPTestServer(t)
	store := vpNewStore(t, ts, "OFF")
	// A Cognito user pool must exist in the Cognito store.
	if code := vpErrCode(t, ts, "CreateIdentitySource", `{"policyStoreId":"`+store+`","configuration":{"cognitoUserPoolConfiguration":{"userPoolArn":"arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_missing"}}}`); code != "ValidationException" {
		t.Fatalf("missing user pool: %s", code)
	}
	pool, err := srv.cognitouserpools.CreateUserPool(cognitoUserPoolsCreateUserPoolInput{PoolName: "vp"})
	if err != nil {
		t.Fatal(err)
	}
	arn := "arn:aws:cognito-idp:us-east-1:123456789012:userpool/" + pool.ID
	src := vpOK(t, ts, "CreateIdentitySource", vpJSON(map[string]any{"policyStoreId": store, "principalEntityType": "User",
		"configuration": map[string]any{"cognitoUserPoolConfiguration": map[string]any{"userPoolArn": arn, "clientIds": []any{"app"}, "groupConfiguration": map[string]any{"groupEntityType": "Group"}}}}))
	if code := vpErrCode(t, ts, "CreateIdentitySource", vpJSON(map[string]any{"policyStoreId": store, "configuration": map[string]any{"openIdConnectConfiguration": map[string]any{"issuer": "https://idp.example.com", "tokenSelection": map[string]any{"identityTokenOnly": map[string]any{}}}}})); code != "ServiceQuotaExceededException" {
		t.Fatalf("second identity source: %s", code)
	}
	got := vpOK(t, ts, "GetIdentitySource", `{"policyStoreId":"`+store+`","identitySourceId":"`+src["identitySourceId"].(string)+`"}`)
	issuer := "https://cognito-idp.us-east-1.amazonaws.com/" + pool.ID
	if got["configuration"].(map[string]any)["cognitoUserPoolConfiguration"].(map[string]any)["issuer"] != issuer {
		t.Fatalf("GetIdentitySource: %v", got)
	}
	vpStatic(t, ts, store, `permit(principal in Group::"`+pool.ID+`|admins", action, resource) when { principal.email like "*@example.com" };`)
	token := func(claims map[string]any) string {
		base := map[string]any{"iss": issuer, "sub": "u-1", "aud": "app", "token_use": "id", "email": "alice@example.com",
			"cognito:groups": []any{"admins"}, "exp": time.Now().Add(time.Hour).Unix()}
		for k, v := range claims {
			base[k] = v
		}
		return vpTestJWT(base)
	}
	body := func(tok string) string {
		return vpJSON(map[string]any{"policyStoreId": store, "identityToken": tok,
			"action": map[string]any{"actionType": "Action", "actionId": "view"}, "resource": map[string]any{"entityType": "Photo", "entityId": "p1"}})
	}
	res := vpOK(t, ts, "IsAuthorizedWithToken", body(token(nil)))
	if res["decision"] != "ALLOW" || res["principal"].(map[string]any)["entityId"] != pool.ID+"|u-1" {
		t.Fatalf("token allow: %v", res)
	}
	if res := vpOK(t, ts, "IsAuthorizedWithToken", body(token(map[string]any{"cognito:groups": []any{"viewers"}}))); res["decision"] != "DENY" {
		t.Fatalf("token without the group: %v", res)
	}
	for name, claims := range map[string]map[string]any{
		"expired":      {"exp": time.Now().Add(-time.Hour).Unix()},
		"wrong client": {"aud": "other"},
		"wrong issuer": {"iss": "https://elsewhere.example.com"},
		"access token": {"token_use": "access"},
	} {
		if code := vpErrCode(t, ts, "IsAuthorizedWithToken", body(token(claims))); code != "ValidationException" {
			t.Errorf("%s: %s", name, code)
		}
	}
	batch := vpOK(t, ts, "BatchIsAuthorizedWithToken", vpJSON(map[string]any{"policyStoreId": store, "identityToken": token(nil),
		"requests": []any{map[string]any{"action": map[string]any{"actionType": "Action", "actionId": "view"}, "resource": map[string]any{"entityType": "Photo", "entityId": "p1"}}}}))
	if batch["results"].([]any)[0].(map[string]any)["decision"] != "ALLOW" || batch["principal"] == nil {
		t.Fatalf("batch with token: %v", batch)
	}
}

func TestVerifiedPermissionsStage1TagsAndLimits(t *testing.T) {
	_, ts := newVPTestServer(t)
	store := vpNewStore(t, ts, "OFF")
	arn := vpOK(t, ts, "GetPolicyStore", `{"policyStoreId":"`+store+`"}`)["arn"].(string)
	vpOK(t, ts, "TagResource", `{"resourceArn":"`+arn+`","tags":{"team":"auth"}}`)
	if tags := vpOK(t, ts, "ListTagsForResource", `{"resourceArn":"`+arn+`"}`)["tags"].(map[string]any); tags["team"] != "auth" {
		t.Fatalf("tags: %v", tags)
	}
	if code := vpErrCode(t, ts, "TagResource", `{"resourceArn":"`+arn+`","tags":{"aws:x":"y"}}`); code != "ValidationException" {
		t.Fatalf("aws: tag: %s", code)
	}
	vpOK(t, ts, "UntagResource", `{"resourceArn":"`+arn+`","tagKeys":["team"]}`)
	// Requests are bounded: batch size and entity count.
	reqs := make([]any, vpMaxBatchAuthz+1)
	for i := range reqs {
		reqs[i] = map[string]any{"principal": map[string]any{"entityType": "User", "entityId": "a"}, "action": map[string]any{"actionType": "Action", "actionId": "v"}, "resource": map[string]any{"entityType": "Photo", "entityId": "p"}}
	}
	if code := vpErrCode(t, ts, "BatchIsAuthorized", vpJSON(map[string]any{"policyStoreId": store, "requests": reqs})); code != "ValidationException" {
		t.Fatalf("oversized batch: %s", code)
	}
	ents := make([]any, vpMaxEntities+1)
	for i := range ents {
		ents[i] = map[string]any{"identifier": map[string]any{"entityType": "User", "entityId": fmt.Sprint(i)}}
	}
	if code := vpErrCode(t, ts, "IsAuthorized", vpJSON(map[string]any{"policyStoreId": store, "entities": map[string]any{"entityList": ents}})); code != "ValidationException" {
		t.Fatalf("too many entities: %s", code)
	}
}
