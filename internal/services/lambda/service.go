package lambda

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrNotFound         = errors.New("resource not found")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type Function struct {
	Name          string
	ARN           string
	Runtime       string
	Role          string
	Handler       string
	Description   string
	Timeout       int32
	MemorySize    int32
	CodeSHA256    string
	CodeSize      int64
	LastModified  time.Time
	RevisionID    string
	Version       string
	PackageType   string
	State         string
	LastUpdate    string
	Architectures []string
}

type Alias struct {
	Name            string
	ARN             string
	FunctionVersion string
	Description     string
	RevisionID      string
}

type PermissionStatement struct {
	SID           string `json:"Sid"`
	Effect        string `json:"Effect"`
	Principal     string `json:"Principal"`
	Action        string `json:"Action"`
	Resource      string `json:"Resource"`
	SourceArn     string `json:"SourceArn,omitempty"`
	SourceAccount string `json:"SourceAccount,omitempty"`
}

type functionRecord struct {
	latest   *Function
	code     []byte
	nextVer  int64
	versions map[string]*Function
	aliases  map[string]*Alias
	policy   map[string]map[string]PermissionStatement
	tags     map[string]string
}

type Service struct {
	mu        sync.Mutex
	seq       uint64
	functions map[string]*functionRecord
	extras    *extraState
}

func NewService() *Service {
	return &Service{
		functions: map[string]*functionRecord{},
		extras:    newExtraState(),
	}
}

func (s *Service) CreateFunction(name, role, handler, runtime, description string, timeout, memory int32, code []byte, tags map[string]string, architectures []string) (Function, error) {
	name = strings.TrimSpace(name)
	role = strings.TrimSpace(role)
	handler = strings.TrimSpace(handler)
	runtime = strings.TrimSpace(runtime)
	if name == "" || role == "" || handler == "" || runtime == "" {
		return Function{}, ErrInvalidParameter
	}
	if len(code) == 0 {
		return Function{}, ErrInvalidParameter
	}
	if timeout == 0 {
		timeout = 3
	}
	if memory == 0 {
		memory = 128
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.functions[name]; ok {
		return Function{}, ErrAlreadyExists
	}
	fn := &Function{
		Name:          name,
		ARN:           functionARN(name),
		Runtime:       runtime,
		Role:          role,
		Handler:       handler,
		Description:   strings.TrimSpace(description),
		Timeout:       timeout,
		MemorySize:    memory,
		CodeSHA256:    codeSHA256(code),
		CodeSize:      int64(len(code)),
		LastModified:  now,
		RevisionID:    s.nextRevisionIDLocked(),
		Version:       "$LATEST",
		PackageType:   "Zip",
		State:         "Active",
		LastUpdate:    "Successful",
		Architectures: normalizeArchitectures(architectures),
	}
	s.functions[name] = &functionRecord{
		latest:   fn,
		code:     append([]byte(nil), code...),
		nextVer:  1,
		versions: map[string]*Function{},
		aliases:  map[string]*Alias{},
		policy:   map[string]map[string]PermissionStatement{},
		tags:     cloneStringMap(tags),
	}
	return cloneFunction(fn), nil
}

func (s *Service) DeleteFunction(ref, qualifier string) error {
	name, q := parseFunctionRef(ref)
	if qualifier != "" {
		q = qualifier
	}
	name = strings.TrimSpace(name)
	q = strings.TrimSpace(q)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.functions[name]
	if rec == nil {
		return ErrNotFound
	}
	if q == "" || q == "$LATEST" {
		delete(s.functions, name)
		return nil
	}
	if _, ok := rec.aliases[q]; ok {
		delete(rec.aliases, q)
		return nil
	}
	if _, ok := rec.versions[q]; ok {
		delete(rec.versions, q)
		return nil
	}
	return ErrNotFound
}

func (s *Service) ListFunctions(maxItems, marker int) ([]Function, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Function, 0, len(s.functions))
	for _, rec := range s.functions {
		out = append(out, cloneFunction(rec.latest))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return paginateLambdaSlice(out, maxItems, marker)
}

func (s *Service) GetFunction(ref, qualifier string) (Function, map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return Function{}, nil, err
	}
	return fn, cloneStringMap(rec.tags), nil
}

func (s *Service) GetFunctionConfiguration(ref, qualifier string) (Function, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return Function{}, err
	}
	return fn, nil
}

func (s *Service) UpdateFunctionConfiguration(ref string, qualifier string, runtime, role, handler, description *string, timeout, memory *int32, architectures []string) (Function, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, q := parseFunctionRef(ref)
	if qualifier != "" {
		q = qualifier
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Function{}, ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return Function{}, ErrNotFound
	}
	if strings.TrimSpace(q) != "" && q != "$LATEST" {
		return Function{}, ErrInvalidParameter
	}
	if runtime != nil && strings.TrimSpace(*runtime) != "" {
		rec.latest.Runtime = strings.TrimSpace(*runtime)
	}
	if role != nil && strings.TrimSpace(*role) != "" {
		rec.latest.Role = strings.TrimSpace(*role)
	}
	if handler != nil && strings.TrimSpace(*handler) != "" {
		rec.latest.Handler = strings.TrimSpace(*handler)
	}
	if description != nil {
		rec.latest.Description = strings.TrimSpace(*description)
	}
	if timeout != nil {
		rec.latest.Timeout = *timeout
	}
	if memory != nil {
		rec.latest.MemorySize = *memory
	}
	if architectures != nil {
		rec.latest.Architectures = normalizeArchitectures(architectures)
	}
	rec.latest.LastModified = time.Now().UTC()
	rec.latest.RevisionID = s.nextRevisionIDLocked()
	return cloneFunction(rec.latest), nil
}

func (s *Service) UpdateFunctionCode(ref string, qualifier string, code []byte, publish bool) (Function, error) {
	if len(code) == 0 {
		return Function{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name, q := parseFunctionRef(ref)
	if qualifier != "" {
		q = qualifier
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Function{}, ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return Function{}, ErrNotFound
	}
	if strings.TrimSpace(q) != "" && q != "$LATEST" {
		return Function{}, ErrInvalidParameter
	}
	rec.code = append([]byte(nil), code...)
	rec.latest.CodeSHA256 = codeSHA256(code)
	rec.latest.CodeSize = int64(len(code))
	rec.latest.LastModified = time.Now().UTC()
	rec.latest.RevisionID = s.nextRevisionIDLocked()
	if publish {
		version := s.publishVersionLocked(rec, "")
		return cloneFunction(version), nil
	}
	return cloneFunction(rec.latest), nil
}

func (s *Service) PublishVersion(ref, description string) (Function, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return Function{}, ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return Function{}, ErrNotFound
	}
	version := s.publishVersionLocked(rec, description)
	return cloneFunction(version), nil
}

func (s *Service) ListVersionsByFunction(ref string) ([]Function, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return nil, ErrNotFound
	}
	out := []Function{cloneFunction(rec.latest)}
	versionKeys := make([]string, 0, len(rec.versions))
	for version := range rec.versions {
		versionKeys = append(versionKeys, version)
	}
	sort.Slice(versionKeys, func(i, j int) bool {
		li, _ := strconv.Atoi(versionKeys[i])
		lj, _ := strconv.Atoi(versionKeys[j])
		return li < lj
	})
	for _, version := range versionKeys {
		out = append(out, cloneFunction(rec.versions[version]))
	}
	return out, nil
}

func (s *Service) CreateAlias(ref, aliasName, functionVersion, description string) (Alias, error) {
	aliasName = strings.TrimSpace(aliasName)
	functionVersion = strings.TrimSpace(functionVersion)
	if aliasName == "" || functionVersion == "" {
		return Alias{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return Alias{}, ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return Alias{}, ErrNotFound
	}
	if _, ok := rec.aliases[aliasName]; ok {
		return Alias{}, ErrAlreadyExists
	}
	if functionVersion != "$LATEST" {
		if _, ok := rec.versions[functionVersion]; !ok {
			return Alias{}, ErrNotFound
		}
	}
	alias := &Alias{
		Name:            aliasName,
		ARN:             aliasARN(name, aliasName),
		FunctionVersion: functionVersion,
		Description:     strings.TrimSpace(description),
		RevisionID:      s.nextRevisionIDLocked(),
	}
	rec.aliases[aliasName] = alias
	return cloneAlias(alias), nil
}

func (s *Service) GetAlias(ref, aliasName string) (Alias, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	aliasName = strings.TrimSpace(aliasName)
	if name == "" || aliasName == "" {
		return Alias{}, ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return Alias{}, ErrNotFound
	}
	alias := rec.aliases[aliasName]
	if alias == nil {
		return Alias{}, ErrNotFound
	}
	return cloneAlias(alias), nil
}

func (s *Service) ListAliases(ref string) ([]Alias, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return nil, ErrNotFound
	}
	out := make([]Alias, 0, len(rec.aliases))
	for _, alias := range rec.aliases {
		out = append(out, cloneAlias(alias))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) UpdateAlias(ref, aliasName, functionVersion string, description *string) (Alias, error) {
	functionVersion = strings.TrimSpace(functionVersion)
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	aliasName = strings.TrimSpace(aliasName)
	if name == "" || aliasName == "" {
		return Alias{}, ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return Alias{}, ErrNotFound
	}
	alias := rec.aliases[aliasName]
	if alias == nil {
		return Alias{}, ErrNotFound
	}
	if functionVersion != "" {
		if functionVersion != "$LATEST" {
			if _, ok := rec.versions[functionVersion]; !ok {
				return Alias{}, ErrNotFound
			}
		}
		alias.FunctionVersion = functionVersion
	}
	if description != nil {
		alias.Description = strings.TrimSpace(*description)
	}
	alias.RevisionID = s.nextRevisionIDLocked()
	return cloneAlias(alias), nil
}

func (s *Service) DeleteAlias(ref, aliasName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	aliasName = strings.TrimSpace(aliasName)
	if name == "" || aliasName == "" {
		return ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return ErrNotFound
	}
	if _, ok := rec.aliases[aliasName]; !ok {
		return ErrNotFound
	}
	delete(rec.aliases, aliasName)
	return nil
}

func (s *Service) AddPermission(ref, qualifier, statementID, action, principal, sourceArn, sourceAccount string) (string, error) {
	statementID = strings.TrimSpace(statementID)
	action = strings.TrimSpace(action)
	principal = strings.TrimSpace(principal)
	if statementID == "" || action == "" || principal == "" {
		return "", ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return "", err
	}
	policyKey := normalizePolicyQualifier(fn.Version)
	if _, ok := rec.policy[policyKey]; !ok {
		rec.policy[policyKey] = map[string]PermissionStatement{}
	}
	if _, exists := rec.policy[policyKey][statementID]; exists {
		return "", ErrAlreadyExists
	}
	statement := PermissionStatement{
		SID:           statementID,
		Effect:        "Allow",
		Principal:     principal,
		Action:        action,
		Resource:      fn.ARN,
		SourceArn:     strings.TrimSpace(sourceArn),
		SourceAccount: strings.TrimSpace(sourceAccount),
	}
	rec.policy[policyKey][statementID] = statement
	raw, _ := json.Marshal(statement)
	return string(raw), nil
}

func (s *Service) GetPolicy(ref, qualifier string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return "", "", err
	}
	policyKey := normalizePolicyQualifier(fn.Version)
	statements := make([]PermissionStatement, 0, len(rec.policy[policyKey]))
	for _, st := range rec.policy[policyKey] {
		statements = append(statements, st)
	}
	sort.Slice(statements, func(i, j int) bool { return statements[i].SID < statements[j].SID })
	policy := map[string]any{
		"Version":   "2012-10-17",
		"Id":        "default",
		"Statement": statements,
	}
	raw, _ := json.Marshal(policy)
	return string(raw), rec.latest.RevisionID, nil
}

func (s *Service) RemovePermission(ref, qualifier, statementID string) error {
	statementID = strings.TrimSpace(statementID)
	if statementID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return err
	}
	policyKey := normalizePolicyQualifier(fn.Version)
	statements := rec.policy[policyKey]
	if len(statements) == 0 {
		return ErrNotFound
	}
	if _, ok := statements[statementID]; !ok {
		return ErrNotFound
	}
	delete(statements, statementID)
	return nil
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.resolveByResourceARNLocked(resourceARN)
	if rec == nil {
		return ErrNotFound
	}
	for k, v := range tags {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		rec.tags[k] = strings.TrimSpace(v)
	}
	return nil
}

func (s *Service) UntagResource(resourceARN string, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.resolveByResourceARNLocked(resourceARN)
	if rec == nil {
		return ErrNotFound
	}
	for _, key := range keys {
		delete(rec.tags, strings.TrimSpace(key))
	}
	return nil
}

func (s *Service) ListTags(resourceARN string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.resolveByResourceARNLocked(resourceARN)
	if rec == nil {
		return nil, ErrNotFound
	}
	return cloneStringMap(rec.tags), nil
}

type InvokeResult struct {
	StatusCode      int
	ExecutedVersion string
	Payload         []byte
}

func (s *Service) Invoke(ref, qualifier, invocationType string, payload []byte) (InvokeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return InvokeResult{}, err
	}
	itype := strings.TrimSpace(invocationType)
	if itype == "" {
		itype = "RequestResponse"
	}
	switch itype {
	case "DryRun":
		return InvokeResult{StatusCode: 204, ExecutedVersion: fn.Version}, nil
	case "Event":
		return InvokeResult{StatusCode: 202, ExecutedVersion: fn.Version}, nil
	case "RequestResponse":
		// continue
	default:
		return InvokeResult{}, ErrInvalidParameter
	}

	respPayload := payload
	if len(respPayload) == 0 {
		respPayload = []byte(`{"statusCode":200}`)
	}
	return InvokeResult{
		StatusCode:      200,
		ExecutedVersion: fn.Version,
		Payload:         append([]byte(nil), respPayload...),
	}, nil
}

func (s *Service) resolveFunctionLocked(ref, qualifier string) (*functionRecord, Function, error) {
	name, parsedQualifier := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, Function{}, ErrInvalidParameter
	}
	rec := s.functions[name]
	if rec == nil {
		return nil, Function{}, ErrNotFound
	}
	q := strings.TrimSpace(qualifier)
	if q == "" {
		q = strings.TrimSpace(parsedQualifier)
	}
	if q == "" || q == "$LATEST" {
		return rec, cloneFunction(rec.latest), nil
	}
	if alias := rec.aliases[q]; alias != nil {
		if alias.FunctionVersion == "$LATEST" {
			fn := cloneFunction(rec.latest)
			fn.ARN = alias.ARN
			return rec, fn, nil
		}
		version := rec.versions[alias.FunctionVersion]
		if version == nil {
			return nil, Function{}, ErrNotFound
		}
		fn := cloneFunction(version)
		fn.ARN = alias.ARN
		return rec, fn, nil
	}
	version := rec.versions[q]
	if version == nil {
		return nil, Function{}, ErrNotFound
	}
	return rec, cloneFunction(version), nil
}

func (s *Service) resolveByResourceARNLocked(resourceARN string) *functionRecord {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return nil
	}
	if decoded, err := url.PathUnescape(resourceARN); err == nil && decoded != resourceARN {
		if rec := s.resolveByResourceARNLocked(decoded); rec != nil {
			return rec
		}
	}
	if strings.HasPrefix(resourceARN, "arn:") {
		name := functionNameFromARN(resourceARN)
		if rec := s.functions[name]; rec != nil {
			return rec
		}
	}
	if strings.Contains(resourceARN, ":function:") {
		name, _ := parseFunctionRef(resourceARN)
		if rec := s.functions[strings.TrimSpace(name)]; rec != nil {
			return rec
		}
	}
	if idx := strings.LastIndex(resourceARN, "/"); idx >= 0 {
		resourceARN = resourceARN[idx+1:]
	}
	return s.functions[resourceARN]
}

func (s *Service) publishVersionLocked(rec *functionRecord, description string) *Function {
	version := strconv.FormatInt(rec.nextVer, 10)
	rec.nextVer++
	now := time.Now().UTC()
	clone := cloneFunction(rec.latest)
	clone.Version = version
	clone.ARN = versionARN(clone.Name, version)
	clone.LastModified = now
	clone.RevisionID = s.nextRevisionIDLocked()
	if strings.TrimSpace(description) != "" {
		clone.Description = strings.TrimSpace(description)
	}
	rec.versions[version] = &clone
	return &clone
}

func (s *Service) nextRevisionIDLocked() string {
	seq := atomic.AddUint64(&s.seq, 1)
	return fmt.Sprintf("rev-%08d", seq)
}

func parseFunctionRef(ref string) (string, string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", ""
	}
	if idx := strings.Index(ref, ":function:"); idx >= 0 {
		ref = ref[idx+len(":function:"):]
	}
	if strings.Contains(ref, "/") {
		parts := strings.SplitN(ref, "/", 2)
		ref = parts[0]
	}
	if idx := strings.Index(ref, ":"); idx >= 0 {
		return strings.TrimSpace(ref[:idx]), strings.TrimSpace(ref[idx+1:])
	}
	return ref, ""
}

func normalizePolicyQualifier(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "$LATEST" {
		return ""
	}
	return version
}

func normalizeArchitectures(in []string) []string {
	if len(in) == 0 {
		return []string{"x86_64"}
	}
	out := make([]string, 0, len(in))
	for _, arch := range in {
		arch = strings.TrimSpace(arch)
		if arch == "" {
			continue
		}
		out = append(out, arch)
	}
	if len(out) == 0 {
		return []string{"x86_64"}
	}
	return out
}

func functionARN(name string) string {
	return fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s", DefaultRegion, DefaultAccountID, name)
}

func versionARN(name, version string) string {
	return fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s:%s", DefaultRegion, DefaultAccountID, name, version)
}

func aliasARN(name, alias string) string {
	return fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s:%s", DefaultRegion, DefaultAccountID, name, alias)
}

func functionNameFromARN(arn string) string {
	arn = strings.TrimSpace(arn)
	idx := strings.Index(arn, ":function:")
	if idx < 0 {
		return arn
	}
	rest := arn[idx+len(":function:"):]
	if rest == "" {
		return ""
	}
	parts := strings.Split(rest, ":")
	return strings.TrimSpace(parts[0])
}

func codeSHA256(code []byte) string {
	sum := sha256.Sum256(code)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func cloneFunction(in *Function) Function {
	out := *in
	out.Architectures = append([]string(nil), in.Architectures...)
	return out
}

func cloneAlias(in *Alias) Alias {
	return *in
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	return out
}

func paginateLambdaSlice[T any](in []T, maxItems, marker int) ([]T, string) {
	if marker < 0 {
		marker = 0
	}
	if marker > len(in) {
		return []T{}, ""
	}
	end := len(in)
	if maxItems > 0 && marker+maxItems < end {
		end = marker + maxItems
	}
	out := append([]T(nil), in[marker:end]...)
	if end >= len(in) {
		return out, ""
	}
	return out, strconv.Itoa(end)
}
