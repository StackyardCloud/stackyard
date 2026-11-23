package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	vpMaxEntities     = 5000 // entities per authorization request
	vpMaxBatchAuthz   = 30   // AWS's batch limit
	vpMaxValueDepth   = 32
	vpCognitoIssuerFm = "https://cognito-idp.%s.amazonaws.com/%s"
)

// ---- identity sources ----

func vpCognitoPoolID(arn string) (region, pool string) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 {
		return "", ""
	}
	return parts[3], strings.TrimPrefix(parts[5], "userpool/")
}

func (src *vpIdentitySource) issuer() string {
	if c := vpMap(src.config, "cognitoUserPoolConfiguration"); c != nil {
		region, pool := vpCognitoPoolID(vpStr(c, "userPoolArn"))
		return fmt.Sprintf(vpCognitoIssuerFm, region, pool)
	}
	return strings.TrimSuffix(vpStr(vpMap(src.config, "openIdConnectConfiguration"), "issuer"), "/")
}

func (src *vpIdentitySource) out(ps *vpPolicyStore, detail bool) map[string]any {
	out := map[string]any{"identitySourceId": src.id, "policyStoreId": ps.id, "principalEntityType": src.principalType,
		"createdDate": vpTime(src.created), "lastUpdatedDate": vpTime(src.updated)}
	cfg := map[string]any{}
	if c := vpMap(src.config, "cognitoUserPoolConfiguration"); c != nil {
		d := map[string]any{"userPoolArn": vpStr(c, "userPoolArn"), "clientIds": vpListOr(c, "clientIds"), "issuer": src.issuer()}
		if g := vpMap(c, "groupConfiguration"); g != nil {
			d["groupConfiguration"] = g
		}
		cfg["cognitoUserPoolConfiguration"] = d
		legacy := map[string]any{"clientIds": vpListOr(c, "clientIds"), "userPoolArn": vpStr(c, "userPoolArn"),
			"discoveryUrl": src.issuer() + "/.well-known/openid-configuration", "openIdIssuer": "COGNITO"}
		out["details"] = legacy
	} else if c := vpMap(src.config, "openIdConnectConfiguration"); c != nil {
		cfg["openIdConnectConfiguration"] = c
	}
	out["configuration"] = cfg
	_ = detail
	return out
}

func vpListOr(m map[string]any, key string) []any {
	if l := vpList(m, key); l != nil {
		return l
	}
	return []any{}
}

// vpCheckConfig validates a Configuration (or UpdateConfiguration) union.
func vpCheckConfig(cfg map[string]any, x vpExtra) *vpErr {
	cog, oidc := vpMap(cfg, "cognitoUserPoolConfiguration"), vpMap(cfg, "openIdConnectConfiguration")
	if (cog == nil) == (oidc == nil) {
		return vpValidation("configuration must contain exactly one of cognitoUserPoolConfiguration or openIdConnectConfiguration")
	}
	if cog != nil {
		if !x.userPoolExists {
			return &vpErr{status: 400, code: "ValidationException", msg: "user pool " + vpStr(cog, "userPoolArn") + " does not exist"}
		}
		return nil
	}
	sel := vpMap(oidc, "tokenSelection")
	if (vpMap(sel, "accessTokenOnly") == nil) == (vpMap(sel, "identityTokenOnly") == nil) {
		return vpValidation("tokenSelection must contain exactly one of accessTokenOnly or identityTokenOnly")
	}
	return nil
}

func vpCreateIdentitySource(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	key, prev := st.idempotent("CreateIdentitySource", in)
	if src := ps.identitySources[prev]; src != nil {
		return vpIdentitySourceRef(ps, src), nil
	}
	cfg := vpMap(in, "configuration")
	if err := vpCheckConfig(cfg, x); err != nil {
		return nil, err
	}
	if len(ps.identitySources) >= vpMaxIdentitySources {
		return nil, vpQuota("IDENTITY_SOURCE", "a policy store can have at most %d identity source", vpMaxIdentitySources)
	}
	src := &vpIdentitySource{id: vpNewID(), principalType: vpStr(in, "principalEntityType"), config: cfg, created: x.now, updated: x.now}
	if src.principalType == "" {
		src.principalType = "User"
	}
	ps.identitySources[src.id] = src
	if key != "" {
		st.tokens[key] = src.id
	}
	return vpIdentitySourceRef(ps, src), nil
}

func vpIdentitySourceRef(ps *vpPolicyStore, src *vpIdentitySource) map[string]any {
	return map[string]any{"identitySourceId": src.id, "policyStoreId": ps.id, "createdDate": vpTime(src.created), "lastUpdatedDate": vpTime(src.updated)}
}

func (st *vpStore) identitySource(in map[string]any) (*vpPolicyStore, *vpIdentitySource, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, nil, err
	}
	src := ps.identitySources[vpStr(in, "identitySourceId")]
	if src == nil {
		return nil, nil, vpNotFound("IDENTITY_SOURCE", vpStr(in, "identitySourceId"))
	}
	return ps, src, nil
}

func vpGetIdentitySource(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, src, err := st.identitySource(in)
	if err != nil {
		return nil, err
	}
	return src.out(ps, true), nil
}

func vpUpdateIdentitySource(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, src, err := st.identitySource(in)
	if err != nil {
		return nil, err
	}
	cfg := vpMap(in, "updateConfiguration")
	if err := vpCheckConfig(cfg, x); err != nil {
		return nil, err
	}
	src.config = cfg
	if t := vpStr(in, "principalEntityType"); t != "" {
		src.principalType = t
	}
	src.updated = x.now
	return vpIdentitySourceRef(ps, src), nil
}

func vpDeleteIdentitySource(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, src, err := st.identitySource(in)
	if err != nil {
		return nil, err
	}
	delete(ps.identitySources, src.id)
	return map[string]any{}, nil
}

func vpListIdentitySources(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, f := range vpList(in, "filters") {
		fm, _ := f.(map[string]any)
		if t := vpStr(fm, "principalEntityType"); t != "" {
			want[t] = true
		}
	}
	var all []*vpIdentitySource
	for _, src := range ps.identitySources {
		if len(want) == 0 || want[src.principalType] {
			all = append(all, src)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].id < all[j].id })
	start, end, next, perr := vpPage(len(all), in)
	if perr != nil {
		return nil, perr
	}
	items := []any{}
	for _, src := range all[start:end] {
		items = append(items, src.out(ps, false))
	}
	return vpWithNext(map[string]any{"identitySources": items}, next), nil
}

// ---- authorization ----

// policySet returns the store's policies as evaluated (template links filled).
func (ps *vpPolicyStore) policySet() map[string]*cedarPolicy {
	out := make(map[string]*cedarPolicy, len(ps.policies))
	for id, p := range ps.policies {
		if eff := ps.effective(p); eff != nil {
			out[id] = eff
		}
	}
	return out
}

type vpEntities struct {
	m     map[cedarEntity]*cedarEntityData
	count int
}

// vpBuildEntities converts an EntitiesDefinition (entityList or cedarJson)
// and adds action groups from the schema.
func vpBuildEntities(ps *vpPolicyStore, def map[string]any) (*vpEntities, *vpErr) {
	ents := &vpEntities{m: map[cedarEntity]*cedarEntityData{}}
	if list := vpList(def, "entityList"); list != nil {
		for _, raw := range list {
			item, _ := raw.(map[string]any)
			e := vpEntityOf(vpMap(item, "identifier"))
			if e == nil {
				continue
			}
			attrs, err := vpAttrMap(vpMap(item, "attributes"), 0)
			if err != nil {
				return nil, vpValidation("entity %s: %v", e, err)
			}
			data := &cedarEntityData{Attrs: attrs}
			for _, p := range vpList(item, "parents") {
				if pe := vpEntityOf(asMap(p)); pe != nil {
					data.Parents = append(data.Parents, *pe)
				}
			}
			if err := ents.add(*e, data); err != nil {
				return nil, err
			}
		}
	} else if raw := vpStr(def, "cedarJson"); raw != "" {
		var list []struct {
			UID     json.RawMessage   `json:"uid"`
			Attrs   map[string]any    `json:"attrs"`
			Parents []json.RawMessage `json:"parents"`
		}
		dec := json.NewDecoder(strings.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&list); err != nil {
			return nil, vpValidation("entities.cedarJson is not a Cedar JSON entity list: %v", err)
		}
		for _, item := range list {
			e, err := vpCedarJSONEntity(item.UID)
			if err != nil {
				return nil, vpValidation("entities.cedarJson: %v", err)
			}
			data := &cedarEntityData{Attrs: cedarRecord{}}
			for k, v := range item.Attrs {
				cv, err := vpCedarJSONValue(v, 0)
				if err != nil {
					return nil, vpValidation("entity %s attribute %s: %v", e, k, err)
				}
				data.Attrs[k] = cv
			}
			for _, p := range item.Parents {
				pe, err := vpCedarJSONEntity(p)
				if err != nil {
					return nil, vpValidation("entity %s parents: %v", e, err)
				}
				data.Parents = append(data.Parents, pe)
			}
			if err := ents.add(e, data); err != nil {
				return nil, err
			}
		}
	}
	if ps.schema != nil { // action groups come from the schema
		for a, def := range ps.schema.actions {
			if len(def.memberOf) == 0 {
				continue
			}
			data := ents.m[a]
			if data == nil {
				data = &cedarEntityData{Attrs: cedarRecord{}}
				ents.m[a] = data
			}
			data.Parents = append(data.Parents, def.memberOf...)
		}
	}
	return ents, nil
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func (ents *vpEntities) add(e cedarEntity, data *cedarEntityData) *vpErr {
	if _, dup := ents.m[e]; dup {
		return vpValidation("entity %s appears more than once", e)
	}
	ents.count += 1 + len(data.Parents)
	if ents.count > vpMaxEntities {
		return vpValidation("a request can carry at most %d entities and parent links", vpMaxEntities)
	}
	ents.m[e] = data
	return nil
}

// vpAttrValue converts an AttributeValue union to a Cedar value.
func vpAttrValue(v map[string]any, depth int) (any, error) {
	if depth > vpMaxValueDepth {
		return nil, fmt.Errorf("values nest more than %d levels", vpMaxValueDepth)
	}
	for k, raw := range v {
		switch k {
		case "boolean":
			b, _ := raw.(bool)
			return b, nil
		case "long":
			f, _ := raw.(float64)
			if f != math.Trunc(f) || f < math.MinInt64 || f >= math.MaxInt64 {
				return nil, fmt.Errorf("long %v is out of range", raw)
			}
			return int64(f), nil
		case "string":
			s, _ := raw.(string)
			return s, nil
		case "entityIdentifier":
			if e := vpEntityOf(asMap(raw)); e != nil {
				return *e, nil
			}
		case "set":
			items, _ := raw.([]any)
			out := make(cedarSet, 0, len(items))
			for _, item := range items {
				cv, err := vpAttrValue(asMap(item), depth+1)
				if err != nil {
					return nil, err
				}
				out = append(out, cv)
			}
			return out, nil
		case "record":
			return vpAttrMap(asMap(raw), depth+1)
		case "ipaddr", "decimal", "datetime", "duration":
			s, _ := raw.(string)
			return cedarExt{Kind: k, Text: s}, nil
		}
	}
	return nil, fmt.Errorf("attribute value has no recognized member")
}

func vpAttrMap(m map[string]any, depth int) (cedarRecord, error) {
	out := make(cedarRecord, len(m))
	for k, v := range m {
		cv, err := vpAttrValue(asMap(v), depth)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", k, err)
		}
		out[k] = cv
	}
	return out, nil
}

// vpCedarJSONEntity reads {"type":..,"id":..} or {"__entity":{...}}.
func vpCedarJSONEntity(raw json.RawMessage) (cedarEntity, error) {
	var ref struct {
		Type   string `json:"type"`
		ID     string `json:"id"`
		Entity *struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"__entity"`
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		return cedarEntity{}, fmt.Errorf("invalid entity reference: %v", err)
	}
	if ref.Entity != nil {
		ref.Type, ref.ID = ref.Entity.Type, ref.Entity.ID
	}
	if ref.Type == "" {
		return cedarEntity{}, fmt.Errorf("entity reference needs a type and id")
	}
	return cedarEntity{Type: ref.Type, ID: ref.ID}, nil
}

// vpCedarJSONValue converts a Cedar JSON value (decoded with UseNumber).
func vpCedarJSONValue(v any, depth int) (any, error) {
	if depth > vpMaxValueDepth {
		return nil, fmt.Errorf("values nest more than %d levels", vpMaxValueDepth)
	}
	switch t := v.(type) {
	case bool, string:
		return t, nil
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return nil, fmt.Errorf("number %s is not a Cedar long", t)
		}
		return n, nil
	case []any:
		out := make(cedarSet, 0, len(t))
		for _, item := range t {
			cv, err := vpCedarJSONValue(item, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, cv)
		}
		return out, nil
	case map[string]any:
		if ent, ok := t["__entity"].(map[string]any); ok && len(t) == 1 {
			return cedarEntity{Type: vpStr(ent, "type"), ID: vpStr(ent, "id")}, nil
		}
		if ext, ok := t["__extn"].(map[string]any); ok && len(t) == 1 {
			arg, _ := ext["arg"].(string)
			return cedarExt{Kind: vpStr(ext, "fn"), Text: arg}, nil
		}
		out := make(cedarRecord, len(t))
		for k, item := range t {
			cv, err := vpCedarJSONValue(item, depth+1)
			if err != nil {
				return nil, err
			}
			out[k] = cv
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported value %v", v)
}

func vpContext(def map[string]any) (cedarRecord, *vpErr) {
	if m := vpMap(def, "contextMap"); m != nil {
		rec, err := vpAttrMap(m, 0)
		if err != nil {
			return nil, vpValidation("context: %v", err)
		}
		return rec, nil
	}
	if raw := vpStr(def, "cedarJson"); raw != "" {
		dec := json.NewDecoder(strings.NewReader(raw))
		dec.UseNumber()
		var obj map[string]any
		if err := dec.Decode(&obj); err != nil {
			return nil, vpValidation("context.cedarJson must be a JSON object: %v", err)
		}
		v, err := vpCedarJSONValue(obj, 0)
		if err != nil {
			return nil, vpValidation("context: %v", err)
		}
		rec, _ := v.(cedarRecord)
		return rec, nil
	}
	return cedarRecord{}, nil
}

func vpAction(m map[string]any) cedarEntity {
	return cedarEntity{Type: vpStr(m, "actionType"), ID: vpStr(m, "actionId")}
}

// decide runs one authorization request and shapes the result.
// tokenCtx holds access-token claims; the request's own context wins.
func vpDecide(policies map[string]*cedarPolicy, ents *vpEntities, principal *cedarEntity, req map[string]any, tokenCtx cedarRecord) (map[string]any, *vpErr) {
	ctx, err := vpContext(vpMap(req, "context"))
	if err != nil {
		return nil, err
	}
	for k, v := range tokenCtx {
		if _, set := ctx[k]; !set {
			ctx[k] = v
		}
	}
	r := &cedarRequest{Action: vpAction(vpMap(req, "action")), Context: ctx, Entities: ents.m}
	if principal != nil {
		r.Principal = *principal
	}
	if res := vpEntityOf(vpMap(req, "resource")); res != nil {
		r.Resource = *res
	}
	allow, determining, errs := cedarAuthorize(policies, r)
	out := map[string]any{"decision": "DENY", "determiningPolicies": []any{}, "errors": []any{}}
	if allow {
		out["decision"] = "ALLOW"
	}
	for _, id := range determining {
		out["determiningPolicies"] = append(out["determiningPolicies"].([]any), map[string]any{"policyId": id})
	}
	for _, e := range errs {
		out["errors"] = append(out["errors"].([]any), map[string]any{"errorDescription": e})
	}
	return out, nil
}

func vpIsAuthorized(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	ents, err := vpBuildEntities(ps, vpMap(in, "entities"))
	if err != nil {
		return nil, err
	}
	return vpDecide(ps.policySet(), ents, vpEntityOf(vpMap(in, "principal")), in, nil)
}

func vpBatchIsAuthorized(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	reqs := vpList(in, "requests")
	if len(reqs) > vpMaxBatchAuthz {
		return nil, vpValidation("a batch can contain at most %d requests", vpMaxBatchAuthz)
	}
	ents, err := vpBuildEntities(ps, vpMap(in, "entities"))
	if err != nil {
		return nil, err
	}
	policies := ps.policySet()
	results := []any{}
	for _, raw := range reqs {
		req := asMap(raw)
		out, err := vpDecide(policies, ents, vpEntityOf(vpMap(req, "principal")), req, nil)
		if err != nil {
			return nil, err
		}
		out["request"] = req
		results = append(results, out)
	}
	return map[string]any{"results": results}, nil
}

// ---- token-based authorization ----

// vpTokenPrincipal decodes the identity or access token, finds the identity
// source whose issuer matches, and returns the principal (with claim
// attributes and group parents) plus access-token claims for the context.
// ponytail: token signatures aren't verified; Stackyard has no issuer keys.
func vpTokenPrincipal(ps *vpPolicyStore, in map[string]any, ents *vpEntities, now time.Time) (*cedarEntity, cedarRecord, *vpErr) {
	idTok, accTok := vpStr(in, "identityToken"), vpStr(in, "accessToken")
	if idTok == "" && accTok == "" {
		return nil, nil, vpValidation("specify an identityToken or an accessToken")
	}
	tok, kind := idTok, "id"
	if tok == "" {
		tok, kind = accTok, "access"
	}
	claims, err := vpDecodeJWT(tok)
	if err != nil {
		return nil, nil, vpValidation("token is not a valid JWT: %v", err)
	}
	if exp, ok := claims["exp"].(json.Number); ok {
		if n, err := exp.Int64(); err == nil && now.Unix() >= n {
			return nil, nil, vpValidation("the token has expired")
		}
	}
	iss := strings.TrimSuffix(vpClaimString(claims["iss"]), "/")
	var src *vpIdentitySource
	for _, s := range ps.identitySources {
		if s.issuer() == iss {
			src = s
		}
	}
	if src == nil {
		return nil, nil, vpValidation("no identity source in policy store %s trusts issuer %q", ps.id, iss)
	}
	var id, groupClaim, groupType string
	if c := vpMap(src.config, "cognitoUserPoolConfiguration"); c != nil {
		if use := vpClaimString(claims["token_use"]); use != "" && use != kind {
			return nil, nil, vpValidation("the %s token has token_use %q", kind, use)
		}
		client := vpClaimString(claims["aud"])
		if kind == "access" {
			client = vpClaimString(claims["client_id"])
		}
		if ids := vpList(c, "clientIds"); len(ids) > 0 && !vpContainsString(ids, client) {
			return nil, nil, vpValidation("the token's client %q isn't one of the identity source's clientIds", client)
		}
		_, pool := vpCognitoPoolID(vpStr(c, "userPoolArn"))
		id = pool + "|" + vpClaimString(claims["sub"])
		groupClaim, groupType = "cognito:groups", vpStr(vpMap(c, "groupConfiguration"), "groupEntityType")
	} else {
		c := vpMap(src.config, "openIdConnectConfiguration")
		sel := vpMap(c, "tokenSelection")
		cfg, want := vpMap(sel, "identityTokenOnly"), "id"
		if cfg == nil {
			cfg, want = vpMap(sel, "accessTokenOnly"), "access"
		}
		if kind != want {
			return nil, nil, vpValidation("this identity source accepts only %s tokens", want)
		}
		aud := vpClaimStrings(claims["aud"])
		for list, label := range map[string]string{"clientIds": "clientIds", "audiences": "audiences"} {
			if allowed := vpList(cfg, list); len(allowed) > 0 && !vpAnyIn(aud, allowed) {
				return nil, nil, vpValidation("the token's audience isn't one of the identity source's %s", label)
			}
		}
		claim := vpStr(cfg, "principalIdClaim")
		if claim == "" {
			claim = "sub"
		}
		id = vpClaimString(claims[claim])
		if prefix := vpStr(c, "entityIdPrefix"); prefix != "" {
			id = prefix + "|" + id
		}
		g := vpMap(c, "groupConfiguration")
		groupClaim, groupType = vpStr(g, "groupClaim"), vpStr(g, "groupEntityType")
	}
	if id == "" || strings.HasSuffix(id, "|") {
		return nil, nil, vpValidation("the token has no principal id claim")
	}
	principal := cedarEntity{Type: src.principalType, ID: id}
	data := &cedarEntityData{Attrs: cedarRecord{}}
	context := cedarRecord{}
	for k, v := range claims {
		cv, err := vpCedarJSONValue(v, 0)
		if err != nil {
			continue // a claim Cedar can't represent (e.g. a float) is left out
		}
		if kind == "id" {
			data.Attrs[k] = cv
		} else {
			context[k] = cv
		}
	}
	if groupType != "" {
		prefix := ""
		if c := vpMap(src.config, "cognitoUserPoolConfiguration"); c != nil {
			_, pool := vpCognitoPoolID(vpStr(c, "userPoolArn"))
			prefix = pool + "|"
		}
		for _, g := range vpClaimStrings(claims[groupClaim]) {
			data.Parents = append(data.Parents, cedarEntity{Type: groupType, ID: prefix + g})
		}
	}
	delete(ents.m, principal) // the token defines the principal
	if err := ents.add(principal, data); err != nil {
		return nil, nil, err
	}
	return &principal, context, nil
}

func vpDecodeJWT(tok string) (map[string]any, error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("expected three dot-separated parts")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, fmt.Errorf("payload is not base64url: %v", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.UseNumber()
	var claims map[string]any
	if err := dec.Decode(&claims); err != nil {
		return nil, fmt.Errorf("payload is not a JSON object: %v", err)
	}
	return claims, nil
}

func vpClaimString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case []any:
		if len(t) > 0 {
			return vpClaimString(t[0])
		}
	}
	return ""
}

func vpClaimStrings(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		var out []string
		for _, item := range t {
			if s := vpClaimString(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func vpContainsString(list []any, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func vpAnyIn(have []string, allowed []any) bool {
	for _, h := range have {
		if vpContainsString(allowed, h) {
			return true
		}
	}
	return false
}

func vpIsAuthorizedWithToken(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	ents, err := vpBuildEntities(ps, vpMap(in, "entities"))
	if err != nil {
		return nil, err
	}
	principal, tokenCtx, err := vpTokenPrincipal(ps, in, ents, x.now)
	if err != nil {
		return nil, err
	}
	out, err := vpDecide(ps.policySet(), ents, principal, in, tokenCtx)
	if err != nil {
		return nil, err
	}
	out["principal"] = vpEntityOut(principal)
	return out, nil
}

func vpBatchIsAuthorizedWithToken(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	reqs := vpList(in, "requests")
	if len(reqs) > vpMaxBatchAuthz {
		return nil, vpValidation("a batch can contain at most %d requests", vpMaxBatchAuthz)
	}
	ents, err := vpBuildEntities(ps, vpMap(in, "entities"))
	if err != nil {
		return nil, err
	}
	principal, tokenCtx, err := vpTokenPrincipal(ps, in, ents, x.now)
	if err != nil {
		return nil, err
	}
	policies := ps.policySet()
	results := []any{}
	for _, raw := range reqs {
		req := asMap(raw)
		out, err := vpDecide(policies, ents, principal, req, tokenCtx)
		if err != nil {
			return nil, err
		}
		out["request"] = req
		results = append(results, out)
	}
	return map[string]any{"principal": vpEntityOut(principal), "results": results}, nil
}
