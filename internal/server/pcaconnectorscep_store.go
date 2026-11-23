package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	pscAccount = "123456789012"
	pscRegion  = "us-east-1"
	// ponytail: caps keep the store bounded; they're set above AWS quotas so
	// repeated test runs against one server don't exhaust them.
	pscMaxConnectors           = 1000
	pscMaxChallengesPerConnect = 50
	pscMaxTags                 = 50
)

type pscError struct {
	status       int
	code         string
	msg          string
	reason       string // ValidationException
	resourceID   string // ResourceNotFound / Conflict
	resourceType string
}

func pscValidation(reason, format string, args ...any) *pscError {
	return &pscError{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...), reason: reason}
}

func pscNotFound(resourceType, id string) *pscError {
	return &pscError{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: fmt.Sprintf("%s %s does not exist.", resourceType, id), resourceID: id, resourceType: resourceType}
}

func pscConflict(resourceType, id, format string, args ...any) *pscError {
	return &pscError{status: http.StatusConflict, code: "ConflictException", msg: fmt.Sprintf(format, args...), resourceID: id, resourceType: resourceType}
}

type pscConnector struct {
	id, arn, caARN, typ, status, endpoint, vpce string
	azureAppID, domain                          string // INTUNE only
	created, updated                            time.Time
}

type pscChallenge struct {
	arn, connectorARN, password string
	created, updated            time.Time
}

type pscStore struct {
	mu         sync.Mutex
	connectors map[string]*pscConnector // by ARN
	challenges map[string]*pscChallenge // by ARN
	tokens     map[string]string        // idempotency key -> ARN
	tags       map[string]map[string]string
}

func newPCAConnectorSCEPStore() *pscStore {
	return &pscStore{
		connectors: map[string]*pscConnector{}, challenges: map[string]*pscChallenge{},
		tokens: map[string]string{}, tags: map[string]map[string]string{},
	}
}

func pscConnectorARN(id string) string {
	return "arn:aws:pca-connector-scep:" + pscRegion + ":" + pscAccount + ":connector/" + id
}

func pscTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func pscConform(op string, out map[string]any) map[string]any {
	m := awsmodels.PCAConnectorSCEPModel()
	if shaped, ok := m.Conform(m.Operations[op].Output, out).(map[string]any); ok {
		return shaped
	}
	return map[string]any{}
}

// pscPassword is a random challenge password; it's only ever returned by
// CreateChallenge and GetChallengePassword.
func pscPassword() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// settle moves a new connector to ACTIVE; reads call it, so CREATING is
// observable exactly once. ponytail: no background work, no timers.
func (c *pscConnector) settle() {
	if c.status == "CREATING" {
		c.status, c.updated = "ACTIVE", time.Now().UTC()
	}
}

func (c *pscConnector) out() map[string]any {
	m := map[string]any{
		"Arn": c.arn, "CertificateAuthorityArn": c.caARN, "Type": c.typ, "Status": c.status,
		"Endpoint": c.endpoint, "CreatedAt": pscTime(c.created), "UpdatedAt": pscTime(c.updated),
	}
	if c.typ == "INTUNE" {
		m["MobileDeviceManagement"] = map[string]any{"Intune": map[string]any{"AzureApplicationId": c.azureAppID, "Domain": c.domain}}
		// ponytail: a fixed OpenID configuration for Intune's token exchange.
		m["OpenIdConfiguration"] = map[string]any{"Issuer": c.endpoint, "Subject": c.arn, "Audience": "api://AzureADTokenExchange"}
	}
	return m
}

func (ch *pscChallenge) metadata() map[string]any {
	return map[string]any{"Arn": ch.arn, "ConnectorArn": ch.connectorARN, "CreatedAt": pscTime(ch.created), "UpdatedAt": pscTime(ch.updated)}
}

type pscConnectorInput struct {
	caARN, azureAppID, domain, vpce, token, endpointBase string
	tags                                                 map[string]string
}

func (st *pscStore) createConnector(in pscConnectorInput) (map[string]any, *pscError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	key := "connector|" + in.token
	if arn, ok := st.tokens[key]; ok && in.token != "" && st.connectors[arn] != nil {
		return pscConform("CreateConnector", map[string]any{"ConnectorArn": arn}), nil
	}
	if len(st.connectors) >= pscMaxConnectors {
		return nil, &pscError{status: http.StatusPaymentRequired, code: "ServiceQuotaExceededException", msg: fmt.Sprintf("An account can have at most %d connectors.", pscMaxConnectors), resourceType: "Connector"}
	}
	if err := pscCheckTags(nil, in.tags); err != nil {
		return nil, err
	}
	id := uuid.NewString()
	now := time.Now().UTC()
	c := &pscConnector{
		id: id, arn: pscConnectorARN(id), caARN: in.caARN, typ: "GENERAL_PURPOSE", status: "CREATING",
		endpoint: strings.TrimRight(in.endpointBase, "/") + "/_stackyard/pca-connector-scep/" + id,
		vpce:     in.vpce, created: now, updated: now,
	}
	if in.azureAppID != "" {
		c.typ, c.azureAppID, c.domain = "INTUNE", in.azureAppID, in.domain
	}
	st.connectors[c.arn] = c
	if in.token != "" {
		st.tokens[key] = c.arn
	}
	if len(in.tags) > 0 {
		st.tags[c.arn] = in.tags
	}
	return pscConform("CreateConnector", map[string]any{"ConnectorArn": c.arn}), nil
}

func (st *pscStore) connectorLocked(arn string) (*pscConnector, *pscError) {
	c := st.connectors[arn]
	if c == nil {
		return nil, pscNotFound("Connector", arn)
	}
	return c, nil
}

func (st *pscStore) getConnector(arn string) (map[string]any, *pscError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.connectorLocked(arn)
	if err != nil {
		return nil, err
	}
	c.settle()
	return pscConform("GetConnector", map[string]any{"Connector": c.out()}), nil
}

func (st *pscStore) listConnectors(nextToken string, max int) (map[string]any, *pscError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	arns := make([]string, 0, len(st.connectors))
	for arn := range st.connectors {
		arns = append(arns, arn)
	}
	page, token, err := pscPage(arns, nextToken, max)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, arn := range page {
		c := st.connectors[arn]
		c.settle()
		items = append(items, c.out())
	}
	out := map[string]any{"Connectors": items}
	if token != "" {
		out["NextToken"] = token
	}
	return pscConform("ListConnectors", out), nil
}

// deleteConnector removes the connector with its challenges, tags and tokens.
// ponytail: AWS reports DELETING for a while; here it's gone at once.
func (st *pscStore) deleteConnector(arn string) *pscError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.connectorLocked(arn); err != nil {
		return err
	}
	for chArn, ch := range st.challenges {
		if ch.connectorARN == arn {
			st.dropLocked(chArn)
			delete(st.challenges, chArn)
		}
	}
	st.dropLocked(arn)
	delete(st.connectors, arn)
	return nil
}

// dropLocked forgets a resource's tags and idempotency tokens.
func (st *pscStore) dropLocked(arn string) {
	delete(st.tags, arn)
	for k, v := range st.tokens {
		if v == arn {
			delete(st.tokens, k)
		}
	}
}

func (st *pscStore) createChallenge(connectorARN, token string, tags map[string]string) (map[string]any, *pscError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.connectorLocked(connectorARN)
	if err != nil {
		return nil, err
	}
	key := "challenge|" + connectorARN + "|" + token
	if arn, ok := st.tokens[key]; ok && token != "" && st.challenges[arn] != nil {
		ch := st.challenges[arn]
		out := ch.metadata()
		out["Password"] = ch.password
		return pscConform("CreateChallenge", map[string]any{"Challenge": out}), nil
	}
	if c.typ != "GENERAL_PURPOSE" {
		return nil, &pscError{status: http.StatusBadRequest, code: "BadRequestException", msg: "Challenge passwords can only be created for GENERAL_PURPOSE connectors; Intune connectors validate devices through Intune."}
	}
	if c.status != "ACTIVE" {
		return nil, pscConflict("Connector", connectorARN, "Connector %s is %s; challenges need an ACTIVE connector.", connectorARN, c.status)
	}
	n := 0
	for _, ch := range st.challenges {
		if ch.connectorARN == connectorARN {
			n++
		}
	}
	if n >= pscMaxChallengesPerConnect {
		return nil, &pscError{status: http.StatusPaymentRequired, code: "ServiceQuotaExceededException", msg: fmt.Sprintf("A connector can have at most %d challenges.", pscMaxChallengesPerConnect), resourceType: "Challenge"}
	}
	if err := pscCheckTags(nil, tags); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	ch := &pscChallenge{arn: connectorARN + "/challenge/" + uuid.NewString(), connectorARN: connectorARN, password: pscPassword(), created: now, updated: now}
	st.challenges[ch.arn] = ch
	if token != "" {
		st.tokens[key] = ch.arn
	}
	if len(tags) > 0 {
		st.tags[ch.arn] = tags
	}
	out := ch.metadata()
	out["Password"] = ch.password
	return pscConform("CreateChallenge", map[string]any{"Challenge": out}), nil
}

func (st *pscStore) challengeLocked(arn string) (*pscChallenge, *pscError) {
	ch := st.challenges[arn]
	if ch == nil {
		return nil, pscNotFound("Challenge", arn)
	}
	return ch, nil
}

func (st *pscStore) getChallengeMetadata(arn string) (map[string]any, *pscError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ch, err := st.challengeLocked(arn)
	if err != nil {
		return nil, err
	}
	return pscConform("GetChallengeMetadata", map[string]any{"ChallengeMetadata": ch.metadata()}), nil
}

func (st *pscStore) getChallengePassword(arn string) (map[string]any, *pscError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ch, err := st.challengeLocked(arn)
	if err != nil {
		return nil, err
	}
	return pscConform("GetChallengePassword", map[string]any{"Password": ch.password}), nil
}

func (st *pscStore) listChallengeMetadata(connectorARN, nextToken string, max int) (map[string]any, *pscError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.connectorLocked(connectorARN); err != nil {
		return nil, err
	}
	var arns []string
	for arn, ch := range st.challenges {
		if ch.connectorARN == connectorARN {
			arns = append(arns, arn)
		}
	}
	page, token, err := pscPage(arns, nextToken, max)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, arn := range page {
		items = append(items, st.challenges[arn].metadata())
	}
	out := map[string]any{"Challenges": items}
	if token != "" {
		out["NextToken"] = token
	}
	return pscConform("ListChallengeMetadata", out), nil
}

func (st *pscStore) deleteChallenge(arn string) *pscError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.challengeLocked(arn); err != nil {
		return err
	}
	st.dropLocked(arn)
	delete(st.challenges, arn)
	return nil
}

func (st *pscStore) existsLocked(arn string) bool {
	return st.connectors[arn] != nil || st.challenges[arn] != nil
}

func (st *pscStore) listTags(arn string) (map[string]any, *pscError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return nil, pscNotFound("Resource", arn)
	}
	tags := map[string]any{}
	for k, v := range st.tags[arn] {
		tags[k] = v
	}
	return pscConform("ListTagsForResource", map[string]any{"Tags": tags}), nil
}

func (st *pscStore) tag(arn string, add map[string]string) *pscError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return pscNotFound("Resource", arn)
	}
	merged := make(map[string]string, len(st.tags[arn])+len(add))
	for k, v := range st.tags[arn] {
		merged[k] = v
	}
	if err := pscCheckTags(merged, add); err != nil {
		return err
	}
	st.tags[arn] = merged
	return nil
}

func (st *pscStore) untag(arn string, keys []string) *pscError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return pscNotFound("Resource", arn)
	}
	next := map[string]string{}
	for k, v := range st.tags[arn] {
		next[k] = v
	}
	for _, k := range keys {
		delete(next, k)
	}
	st.tags[arn] = next
	return nil
}

// pscCheckTags merges add into into (when non-nil) and enforces the limits.
func pscCheckTags(into, add map[string]string) *pscError {
	if into == nil {
		into = map[string]string{}
	}
	for k, v := range add {
		if k == "" || len(k) > 128 || len(v) > 256 {
			return pscValidation("OTHER", "Tag keys must be 1-128 characters and values at most 256.")
		}
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return pscValidation("OTHER", "Tag keys can't start with aws:.")
		}
		into[k] = v
	}
	if len(into) > pscMaxTags {
		return pscValidation("OTHER", "A resource can have at most %d tags.", pscMaxTags)
	}
	return nil
}

// pscPage pages sorted ARNs with a base64url offset token (the model's
// NextToken pattern is base64).
func pscPage(arns []string, nextToken string, max int) ([]string, string, *pscError) {
	sort.Strings(arns)
	start := 0
	if nextToken != "" {
		raw, err := base64.URLEncoding.DecodeString(nextToken)
		n, perr := strconv.ParseInt(string(raw), 10, 32)
		if err != nil || perr != nil || n < 0 || int(n) > len(arns) {
			return nil, "", pscValidation("OTHER", "The NextToken is not valid.")
		}
		start = int(n)
	}
	end := min(start+max, len(arns))
	token := ""
	if end < len(arns) {
		token = base64.URLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}
	return arns[start:end], token, nil
}
