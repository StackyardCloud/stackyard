package lambda

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type extraState struct {
	eventSourceMappings        map[string]map[string]any
	codeSigningConfigs         map[string]map[string]any
	functionCodeSigningConfig  map[string]string
	functionConcurrency        map[string]int32
	functionEventInvokeConfigs map[string]map[string]any
	functionURLConfigs         map[string]map[string]any
	functionRecursionConfigs   map[string]map[string]any
	functionScalingConfigs     map[string]map[string]any
	provisionedConcurrency     map[string]map[string]any
	runtimeManagementConfigs   map[string]map[string]any
	capacityProviders          map[string]map[string]any
	durableExecutions          map[string]map[string]any
	durableExecutionHistory    map[string][]map[string]any
	layers                     map[string]*layerRecord
}

type layerRecord struct {
	nextVersion int64
	versions    map[string]map[string]any
	policies    map[string]map[string]PermissionStatement
}

func newExtraState() *extraState {
	return &extraState{
		eventSourceMappings:        map[string]map[string]any{},
		codeSigningConfigs:         map[string]map[string]any{},
		functionCodeSigningConfig:  map[string]string{},
		functionConcurrency:        map[string]int32{},
		functionEventInvokeConfigs: map[string]map[string]any{},
		functionURLConfigs:         map[string]map[string]any{},
		functionRecursionConfigs:   map[string]map[string]any{},
		functionScalingConfigs:     map[string]map[string]any{},
		provisionedConcurrency:     map[string]map[string]any{},
		runtimeManagementConfigs:   map[string]map[string]any{},
		capacityProviders:          map[string]map[string]any{},
		durableExecutions:          map[string]map[string]any{},
		durableExecutionHistory:    map[string][]map[string]any{},
		layers:                     map[string]*layerRecord{},
	}
}

func (s *Service) GetAccountSettings() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{
		"AccountLimit": map[string]any{
			"CodeSizeUnzipped":               int64(262144000),
			"CodeSizeZipped":                 int64(80530636800),
			"ConcurrentExecutions":           int32(1000),
			"TotalCodeSize":                  int64(80530636800),
			"UnreservedConcurrentExecutions": int32(1000),
		},
		"AccountUsage": map[string]any{
			"FunctionCount": int64(len(s.functions)),
			"TotalCodeSize": totalCodeSizeLocked(s.functions),
		},
	}
}

func totalCodeSizeLocked(functions map[string]*functionRecord) int64 {
	var total int64
	for _, rec := range functions {
		total += rec.latest.CodeSize
	}
	return total
}

func (s *Service) CreateEventSourceMapping(functionRef string, payload map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, fn, err := s.resolveFunctionLocked(functionRef, "")
	if err != nil {
		return nil, err
	}
	uuid := "esm-" + strings.TrimPrefix(s.nextRevisionIDLocked(), "rev-")
	now := time.Now().UTC()
	mapping := map[string]any{
		"UUID":                  uuid,
		"FunctionArn":           fn.ARN,
		"EventSourceArn":        strings.TrimSpace(stringValue(payload["EventSourceArn"])),
		"BatchSize":             int32Value(payload["BatchSize"], 100),
		"State":                 boolState(payload["Enabled"], true),
		"StateTransitionReason": "USER_INITIATED",
		"LastProcessingResult":  "OK",
		"LastModified":          float64(now.Unix()),
	}
	s.extras.eventSourceMappings[uuid] = cloneAnyMap(mapping)
	return cloneAnyMap(mapping), nil
}

func (s *Service) GetEventSourceMapping(uuid string) (map[string]any, error) {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	mapping := s.extras.eventSourceMappings[uuid]
	if mapping == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(mapping), nil
}

func (s *Service) ListEventSourceMappings(functionRef, eventSourceArn string, maxItems, marker int) ([]map[string]any, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, 0, len(s.extras.eventSourceMappings))
	for _, mapping := range s.extras.eventSourceMappings {
		if functionRef != "" {
			name, _ := parseFunctionRef(functionRef)
			if !strings.Contains(stringValue(mapping["FunctionArn"]), ":function:"+strings.TrimSpace(name)) {
				continue
			}
		}
		if strings.TrimSpace(eventSourceArn) != "" && strings.TrimSpace(eventSourceArn) != strings.TrimSpace(stringValue(mapping["EventSourceArn"])) {
			continue
		}
		out = append(out, cloneAnyMap(mapping))
	}
	sort.Slice(out, func(i, j int) bool {
		return stringValue(out[i]["UUID"]) < stringValue(out[j]["UUID"])
	})
	paged, next := paginateLambdaSlice(out, maxItems, marker)
	return paged, next, nil
}

func (s *Service) UpdateEventSourceMapping(uuid string, payload map[string]any) (map[string]any, error) {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	mapping := s.extras.eventSourceMappings[uuid]
	if mapping == nil {
		return nil, ErrNotFound
	}
	if fnRef := strings.TrimSpace(stringValue(payload["FunctionName"])); fnRef != "" {
		_, fn, err := s.resolveFunctionLocked(fnRef, "")
		if err != nil {
			return nil, err
		}
		mapping["FunctionArn"] = fn.ARN
	}
	if v, ok := payload["BatchSize"]; ok {
		mapping["BatchSize"] = int32Value(v, int32Value(mapping["BatchSize"], 100))
	}
	if _, ok := payload["Enabled"]; ok {
		mapping["State"] = boolState(payload["Enabled"], true)
	}
	mapping["LastModified"] = float64(time.Now().UTC().Unix())
	return cloneAnyMap(mapping), nil
}

func (s *Service) DeleteEventSourceMapping(uuid string) (map[string]any, error) {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	mapping := s.extras.eventSourceMappings[uuid]
	if mapping == nil {
		return nil, ErrNotFound
	}
	delete(s.extras.eventSourceMappings, uuid)
	mapping = cloneAnyMap(mapping)
	mapping["State"] = "Deleting"
	return mapping, nil
}

func (s *Service) PutFunctionConcurrency(ref string, reserved int32) (int32, error) {
	if reserved < 0 {
		return 0, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return 0, ErrNotFound
	}
	s.extras.functionConcurrency[name] = reserved
	return reserved, nil
}

func (s *Service) GetFunctionConcurrency(ref string) (int32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return 0, ErrNotFound
	}
	value, ok := s.extras.functionConcurrency[name]
	if !ok {
		return 0, ErrNotFound
	}
	return value, nil
}

func (s *Service) DeleteFunctionConcurrency(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return ErrNotFound
	}
	delete(s.extras.functionConcurrency, name)
	return nil
}

func (s *Service) PutFunctionEventInvokeConfig(ref, qualifier string, payload map[string]any, merge bool) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return nil, err
	}
	key := functionQualifierKey(ref, qualifier)
	current := map[string]any{}
	if merge {
		current = cloneAnyMap(s.extras.functionEventInvokeConfigs[key])
	}
	if current == nil || !merge {
		current = map[string]any{}
	}
	for _, field := range []string{"MaximumRetryAttempts", "MaximumEventAgeInSeconds", "DestinationConfig"} {
		if v, ok := payload[field]; ok {
			current[field] = v
		}
	}
	current["FunctionArn"] = fn.ARN
	current["LastModified"] = time.Now().UTC().Format(time.RFC3339)
	s.extras.functionEventInvokeConfigs[key] = cloneAnyMap(current)
	return cloneAnyMap(current), nil
}

func (s *Service) GetFunctionEventInvokeConfig(ref, qualifier string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return nil, err
	}
	key := functionQualifierKey(ref, qualifier)
	cfg := s.extras.functionEventInvokeConfigs[key]
	if cfg == nil {
		return nil, ErrNotFound
	}
	out := cloneAnyMap(cfg)
	out["FunctionArn"] = fn.ARN
	return out, nil
}

func (s *Service) DeleteFunctionEventInvokeConfig(ref, qualifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.resolveFunctionLocked(ref, qualifier); err != nil {
		return err
	}
	delete(s.extras.functionEventInvokeConfigs, functionQualifierKey(ref, qualifier))
	return nil
}

func (s *Service) ListFunctionEventInvokeConfigs(ref string, maxItems, marker int) ([]map[string]any, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return nil, "", ErrNotFound
	}
	prefix := name + "|"
	out := make([]map[string]any, 0)
	for key, cfg := range s.extras.functionEventInvokeConfigs {
		if strings.HasPrefix(key, prefix) {
			out = append(out, cloneAnyMap(cfg))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return stringValue(out[i]["FunctionArn"]) < stringValue(out[j]["FunctionArn"])
	})
	paged, next := paginateLambdaSlice(out, maxItems, marker)
	return paged, next, nil
}

func (s *Service) PutFunctionURLConfig(ref, qualifier string, payload map[string]any, createOnly bool) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return nil, err
	}
	key := functionQualifierKey(ref, qualifier)
	current := s.extras.functionURLConfigs[key]
	if createOnly && current != nil {
		return nil, ErrAlreadyExists
	}
	if current == nil {
		current = map[string]any{
			"FunctionArn": fn.ARN,
			"FunctionUrl": fmt.Sprintf("https://%s.lambda-url.local/", fn.Name),
		}
	}
	for _, field := range []string{"AuthType", "Cors", "InvokeMode"} {
		if v, ok := payload[field]; ok {
			current[field] = v
		}
	}
	if strings.TrimSpace(stringValue(current["AuthType"])) == "" {
		current["AuthType"] = "NONE"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, ok := current["CreationTime"]; !ok {
		current["CreationTime"] = now
	}
	current["LastModifiedTime"] = now
	s.extras.functionURLConfigs[key] = cloneAnyMap(current)
	return cloneAnyMap(current), nil
}

func (s *Service) GetFunctionURLConfig(ref, qualifier string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.resolveFunctionLocked(ref, qualifier); err != nil {
		return nil, err
	}
	cfg := s.extras.functionURLConfigs[functionQualifierKey(ref, qualifier)]
	if cfg == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(cfg), nil
}

func (s *Service) DeleteFunctionURLConfig(ref, qualifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.resolveFunctionLocked(ref, qualifier); err != nil {
		return err
	}
	delete(s.extras.functionURLConfigs, functionQualifierKey(ref, qualifier))
	return nil
}

func (s *Service) ListFunctionURLConfigs(ref string, maxItems, marker int) ([]map[string]any, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return nil, "", ErrNotFound
	}
	prefix := name + "|"
	out := make([]map[string]any, 0)
	for key, cfg := range s.extras.functionURLConfigs {
		if strings.HasPrefix(key, prefix) {
			out = append(out, cloneAnyMap(cfg))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return stringValue(out[i]["FunctionUrl"]) < stringValue(out[j]["FunctionUrl"])
	})
	paged, next := paginateLambdaSlice(out, maxItems, marker)
	return paged, next, nil
}

func (s *Service) PutRuntimeManagementConfig(ref, qualifier string, payload map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return nil, err
	}
	cfg := cloneAnyMap(payload)
	cfg["FunctionArn"] = fn.ARN
	cfg["UpdateRuntimeOn"] = firstNonEmpty(stringValue(cfg["UpdateRuntimeOn"]), "Auto")
	s.extras.runtimeManagementConfigs[functionQualifierKey(ref, qualifier)] = cloneAnyMap(cfg)
	return cfg, nil
}

func (s *Service) GetRuntimeManagementConfig(ref, qualifier string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.resolveFunctionLocked(ref, qualifier); err != nil {
		return nil, err
	}
	cfg := s.extras.runtimeManagementConfigs[functionQualifierKey(ref, qualifier)]
	if cfg == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(cfg), nil
}

func (s *Service) PutFunctionRecursionConfig(ref string, payload map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	fn := s.functions[name]
	if fn == nil {
		return nil, ErrNotFound
	}
	cfg := map[string]any{
		"FunctionArn":   fn.latest.ARN,
		"RecursiveLoop": firstNonEmpty(stringValue(payload["RecursiveLoop"]), "Terminate"),
	}
	s.extras.functionRecursionConfigs[name] = cloneAnyMap(cfg)
	return cfg, nil
}

func (s *Service) GetFunctionRecursionConfig(ref string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return nil, ErrNotFound
	}
	cfg := s.extras.functionRecursionConfigs[name]
	if cfg == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(cfg), nil
}

func (s *Service) PutFunctionScalingConfig(ref string, payload map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	fn := s.functions[name]
	if fn == nil {
		return nil, ErrNotFound
	}
	cfg := map[string]any{
		"FunctionArn":        fn.latest.ARN,
		"MaximumConcurrency": int32Value(payload["MaximumConcurrency"], 0),
	}
	s.extras.functionScalingConfigs[name] = cloneAnyMap(cfg)
	return cfg, nil
}

func (s *Service) GetFunctionScalingConfig(ref string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return nil, ErrNotFound
	}
	cfg := s.extras.functionScalingConfigs[name]
	if cfg == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(cfg), nil
}

func (s *Service) PutProvisionedConcurrencyConfig(ref, qualifier string, payload map[string]any) (map[string]any, error) {
	qualifier = strings.TrimSpace(qualifier)
	if qualifier == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, fn, err := s.resolveFunctionLocked(ref, qualifier)
	if err != nil {
		return nil, err
	}
	requested := int32Value(payload["ProvisionedConcurrentExecutions"], 0)
	cfg := map[string]any{
		"FunctionArn": fn.ARN,
		"RequestedProvisionedConcurrentExecutions": requested,
		"AllocatedProvisionedConcurrentExecutions": requested,
		"AvailableProvisionedConcurrentExecutions": requested,
		"Status":       "READY",
		"StatusReason": "",
		"LastModified": time.Now().UTC().Format(time.RFC3339),
	}
	s.extras.provisionedConcurrency[functionQualifierKey(ref, qualifier)] = cloneAnyMap(cfg)
	return cfg, nil
}

func (s *Service) GetProvisionedConcurrencyConfig(ref, qualifier string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.resolveFunctionLocked(ref, qualifier); err != nil {
		return nil, err
	}
	cfg := s.extras.provisionedConcurrency[functionQualifierKey(ref, qualifier)]
	if cfg == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(cfg), nil
}

func (s *Service) DeleteProvisionedConcurrencyConfig(ref, qualifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.resolveFunctionLocked(ref, qualifier); err != nil {
		return err
	}
	delete(s.extras.provisionedConcurrency, functionQualifierKey(ref, qualifier))
	return nil
}

func (s *Service) ListProvisionedConcurrencyConfigs(ref string, maxItems, marker int) ([]map[string]any, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return nil, "", ErrNotFound
	}
	prefix := name + "|"
	out := make([]map[string]any, 0)
	for key, cfg := range s.extras.provisionedConcurrency {
		if strings.HasPrefix(key, prefix) {
			out = append(out, cloneAnyMap(cfg))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return stringValue(out[i]["FunctionArn"]) < stringValue(out[j]["FunctionArn"])
	})
	paged, next := paginateLambdaSlice(out, maxItems, marker)
	return paged, next, nil
}

func (s *Service) CreateCodeSigningConfig(payload map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := "csc-" + strings.TrimPrefix(s.nextRevisionIDLocked(), "rev-")
	arn := codeSigningConfigARN(id)
	cfg := map[string]any{
		"CodeSigningConfigId":  id,
		"CodeSigningConfigArn": arn,
		"Description":          stringValue(payload["Description"]),
		"AllowedPublishers":    mapValue(payload["AllowedPublishers"]),
		"CodeSigningPolicies":  mapValue(payload["CodeSigningPolicies"]),
		"LastModified":         time.Now().UTC().Format(time.RFC3339),
	}
	s.extras.codeSigningConfigs[arn] = cloneAnyMap(cfg)
	return cloneAnyMap(cfg), nil
}

func (s *Service) GetCodeSigningConfig(ref string) (map[string]any, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := resolveCodeSigningConfigLocked(s.extras.codeSigningConfigs, ref)
	if cfg == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(cfg), nil
}

func (s *Service) UpdateCodeSigningConfig(ref string, payload map[string]any) (map[string]any, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := resolveCodeSigningConfigLocked(s.extras.codeSigningConfigs, ref)
	if cfg == nil {
		return nil, ErrNotFound
	}
	for _, field := range []string{"Description", "AllowedPublishers", "CodeSigningPolicies"} {
		if v, ok := payload[field]; ok {
			cfg[field] = v
		}
	}
	cfg["LastModified"] = time.Now().UTC().Format(time.RFC3339)
	return cloneAnyMap(cfg), nil
}

func (s *Service) DeleteCodeSigningConfig(ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	arn, cfg := resolveCodeSigningConfigWithARNLocked(s.extras.codeSigningConfigs, ref)
	if cfg == nil {
		return ErrNotFound
	}
	delete(s.extras.codeSigningConfigs, arn)
	for fnName, bound := range s.extras.functionCodeSigningConfig {
		if bound == arn {
			delete(s.extras.functionCodeSigningConfig, fnName)
		}
	}
	return nil
}

func (s *Service) ListCodeSigningConfigs(maxItems, marker int) ([]map[string]any, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	arns := make([]string, 0, len(s.extras.codeSigningConfigs))
	for arn := range s.extras.codeSigningConfigs {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]map[string]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, cloneAnyMap(s.extras.codeSigningConfigs[arn]))
	}
	return paginateLambdaSlice(out, maxItems, marker)
}

func (s *Service) PutFunctionCodeSigningConfig(ref, codeSigningConfigArn string) (map[string]any, error) {
	codeSigningConfigArn = strings.TrimSpace(codeSigningConfigArn)
	if codeSigningConfigArn == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return nil, ErrNotFound
	}
	cfg := resolveCodeSigningConfigLocked(s.extras.codeSigningConfigs, codeSigningConfigArn)
	if cfg == nil {
		return nil, ErrNotFound
	}
	s.extras.functionCodeSigningConfig[name] = stringValue(cfg["CodeSigningConfigArn"])
	return map[string]any{"CodeSigningConfigArn": stringValue(cfg["CodeSigningConfigArn"])}, nil
}

func (s *Service) GetFunctionCodeSigningConfig(ref string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return nil, ErrNotFound
	}
	arn := strings.TrimSpace(s.extras.functionCodeSigningConfig[name])
	if arn == "" {
		return nil, ErrNotFound
	}
	return map[string]any{"CodeSigningConfigArn": arn}, nil
}

func (s *Service) DeleteFunctionCodeSigningConfig(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return ErrNotFound
	}
	delete(s.extras.functionCodeSigningConfig, name)
	return nil
}

func (s *Service) ListFunctionsByCodeSigningConfig(ref string, maxItems, marker int) ([]map[string]any, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, "", ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := ref
	if cfg := resolveCodeSigningConfigLocked(s.extras.codeSigningConfigs, ref); cfg != nil {
		arn = stringValue(cfg["CodeSigningConfigArn"])
	} else {
		return nil, "", ErrNotFound
	}
	out := make([]map[string]any, 0)
	for fnName, bound := range s.extras.functionCodeSigningConfig {
		if bound != arn {
			continue
		}
		out = append(out, map[string]any{"FunctionArn": functionARN(fnName)})
	}
	sort.Slice(out, func(i, j int) bool {
		return stringValue(out[i]["FunctionArn"]) < stringValue(out[j]["FunctionArn"])
	})
	paged, next := paginateLambdaSlice(out, maxItems, marker)
	return paged, next, nil
}

func (s *Service) PublishLayerVersion(layerName string, payload map[string]any) (map[string]any, error) {
	layerName = strings.TrimSpace(layerName)
	if layerName == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	layer := s.extras.layers[layerName]
	if layer == nil {
		layer = &layerRecord{nextVersion: 1, versions: map[string]map[string]any{}, policies: map[string]map[string]PermissionStatement{}}
		s.extras.layers[layerName] = layer
	}
	version := strconv.FormatInt(layer.nextVersion, 10)
	layer.nextVersion++
	content := map[string]any{
		"Location":   fmt.Sprintf("https://stackyard.local/layers/%s/%s", layerName, version),
		"CodeSha256": codeSHA256([]byte(layerName + ":" + version)),
		"CodeSize":   int64(len(layerName) + len(version)),
	}
	if in := mapValue(payload["Content"]); in != nil {
		for _, key := range []string{"S3Bucket", "S3Key", "S3ObjectVersion", "ZipFile"} {
			if v, ok := in[key]; ok {
				content[key] = v
			}
		}
	}
	rec := map[string]any{
		"LayerArn":                layerARN(layerName),
		"LayerVersionArn":         layerVersionARN(layerName, version),
		"Description":             stringValue(payload["Description"]),
		"CreatedDate":             time.Now().UTC().Format(time.RFC3339),
		"Version":                 int64Value(version, 1),
		"CompatibleRuntimes":      sliceValue(payload["CompatibleRuntimes"]),
		"CompatibleArchitectures": sliceValue(payload["CompatibleArchitectures"]),
		"LicenseInfo":             stringValue(payload["LicenseInfo"]),
		"Content":                 content,
	}
	layer.versions[version] = cloneAnyMap(rec)
	layer.policies[version] = map[string]PermissionStatement{}
	return cloneAnyMap(rec), nil
}

func (s *Service) GetLayerVersion(layerName, version string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.layerVersionLocked(layerName, version)
	if rec == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(rec), nil
}

func (s *Service) GetLayerVersionByARN(layerVersionARN string) (map[string]any, error) {
	layerName, version := parseLayerVersionARN(layerVersionARN)
	if layerName == "" || version == "" {
		return nil, ErrInvalidParameter
	}
	return s.GetLayerVersion(layerName, version)
}

func (s *Service) DeleteLayerVersion(layerName, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	layerName = strings.TrimSpace(layerName)
	version = strings.TrimSpace(version)
	if layerName == "" || version == "" {
		return ErrInvalidParameter
	}
	layer := s.extras.layers[layerName]
	if layer == nil || layer.versions[version] == nil {
		return ErrNotFound
	}
	delete(layer.versions, version)
	delete(layer.policies, version)
	return nil
}

func (s *Service) ListLayerVersions(layerName string, maxItems, marker int) ([]map[string]any, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	layer := s.extras.layers[strings.TrimSpace(layerName)]
	if layer == nil {
		return nil, "", ErrNotFound
	}
	versions := make([]int, 0, len(layer.versions))
	for v := range layer.versions {
		n, err := strconv.Atoi(v)
		if err == nil {
			versions = append(versions, n)
		}
	}
	sort.Ints(versions)
	out := make([]map[string]any, 0, len(versions))
	for _, n := range versions {
		out = append(out, cloneAnyMap(layer.versions[strconv.Itoa(n)]))
	}
	paged, next := paginateLambdaSlice(out, maxItems, marker)
	return paged, next, nil
}

func (s *Service) ListLayers(maxItems, marker int) ([]map[string]any, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.extras.layers))
	for name := range s.extras.layers {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		layer := s.extras.layers[name]
		latest := int64(0)
		for version := range layer.versions {
			n, err := strconv.ParseInt(version, 10, 64)
			if err == nil && n > latest {
				latest = n
			}
		}
		if latest == 0 {
			continue
		}
		ver := cloneAnyMap(layer.versions[strconv.FormatInt(latest, 10)])
		out = append(out, map[string]any{
			"LayerName":             name,
			"LayerArn":              layerARN(name),
			"LatestMatchingVersion": ver,
		})
	}
	return paginateLambdaSlice(out, maxItems, marker)
}

func (s *Service) AddLayerVersionPermission(layerName, version, statementID, action, principal, orgID string) (string, string, error) {
	statementID = strings.TrimSpace(statementID)
	if statementID == "" {
		return "", "", ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.layerVersionLocked(layerName, version)
	if rec == nil {
		return "", "", ErrNotFound
	}
	layer := s.extras.layers[strings.TrimSpace(layerName)]
	if layer.policies[version] == nil {
		layer.policies[version] = map[string]PermissionStatement{}
	}
	if _, exists := layer.policies[version][statementID]; exists {
		return "", "", ErrAlreadyExists
	}
	st := PermissionStatement{
		SID:       statementID,
		Effect:    "Allow",
		Principal: firstNonEmpty(strings.TrimSpace(principal), "*"),
		Action:    firstNonEmpty(strings.TrimSpace(action), "lambda:GetLayerVersion"),
		Resource:  stringValue(rec["LayerVersionArn"]),
	}
	if orgID != "" {
		st.SourceAccount = orgID
	}
	layer.policies[version][statementID] = st
	raw, _ := json.Marshal(st)
	return string(raw), s.nextRevisionIDLocked(), nil
}

func (s *Service) GetLayerVersionPolicy(layerName, version string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	layer := s.extras.layers[strings.TrimSpace(layerName)]
	if layer == nil || layer.versions[strings.TrimSpace(version)] == nil {
		return "", "", ErrNotFound
	}
	statements := make([]PermissionStatement, 0, len(layer.policies[version]))
	for _, st := range layer.policies[version] {
		statements = append(statements, st)
	}
	sort.Slice(statements, func(i, j int) bool { return statements[i].SID < statements[j].SID })
	policy := map[string]any{"Version": "2012-10-17", "Id": "default", "Statement": statements}
	raw, _ := json.Marshal(policy)
	return string(raw), s.nextRevisionIDLocked(), nil
}

func (s *Service) RemoveLayerVersionPermission(layerName, version, statementID string) error {
	statementID = strings.TrimSpace(statementID)
	if statementID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	layer := s.extras.layers[strings.TrimSpace(layerName)]
	if layer == nil || layer.versions[strings.TrimSpace(version)] == nil {
		return ErrNotFound
	}
	statements := layer.policies[strings.TrimSpace(version)]
	if statements == nil {
		return ErrNotFound
	}
	if _, ok := statements[statementID]; !ok {
		return ErrNotFound
	}
	delete(statements, statementID)
	return nil
}

func (s *Service) CreateCapacityProvider(payload map[string]any) (map[string]any, error) {
	name := firstNonEmpty(stringValue(payload["CapacityProviderName"]), stringValue(payload["Name"]))
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.extras.capacityProviders[name] != nil {
		return nil, ErrAlreadyExists
	}
	rec := map[string]any{
		"CapacityProviderName": name,
		"CapacityProviderArn":  capacityProviderARN(name),
		"Status":               "ACTIVE",
		"FunctionArns":         sliceValue(payload["FunctionArns"]),
	}
	s.extras.capacityProviders[name] = cloneAnyMap(rec)
	return cloneAnyMap(rec), nil
}

func (s *Service) GetCapacityProvider(name string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.extras.capacityProviders[name]
	if rec == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(rec), nil
}

func (s *Service) UpdateCapacityProvider(name string, payload map[string]any) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.extras.capacityProviders[name]
	if rec == nil {
		return nil, ErrNotFound
	}
	for _, field := range []string{"Status", "FunctionArns"} {
		if v, ok := payload[field]; ok {
			rec[field] = v
		}
	}
	return cloneAnyMap(rec), nil
}

func (s *Service) DeleteCapacityProvider(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.extras.capacityProviders[name] == nil {
		return ErrNotFound
	}
	delete(s.extras.capacityProviders, name)
	return nil
}

func (s *Service) ListCapacityProviders(maxItems, marker int) ([]map[string]any, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.extras.capacityProviders))
	for name := range s.extras.capacityProviders {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, cloneAnyMap(s.extras.capacityProviders[name]))
	}
	return paginateLambdaSlice(out, maxItems, marker)
}

func (s *Service) ListFunctionVersionsByCapacityProvider(name string, maxItems, marker int) ([]map[string]any, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.extras.capacityProviders[name]
	if rec == nil {
		return nil, "", ErrNotFound
	}
	out := make([]map[string]any, 0)
	for _, raw := range sliceValue(rec["FunctionArns"]) {
		out = append(out, map[string]any{"FunctionArn": raw})
	}
	paged, next := paginateLambdaSlice(out, maxItems, marker)
	return paged, next, nil
}

func (s *Service) ListDurableExecutionsByFunction(ref string, maxItems, marker int) ([]map[string]any, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, _ := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", ErrInvalidParameter
	}
	if s.functions[name] == nil {
		return nil, "", ErrNotFound
	}
	out := make([]map[string]any, 0)
	for _, exec := range s.extras.durableExecutions {
		arn := strings.TrimSpace(stringValue(exec["FunctionArn"]))
		if strings.Contains(arn, ":function:"+name) {
			out = append(out, cloneAnyMap(exec))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return stringValue(out[i]["DurableExecutionArn"]) < stringValue(out[j]["DurableExecutionArn"])
	})
	paged, next := paginateLambdaSlice(out, maxItems, marker)
	return paged, next, nil
}

func (s *Service) GetDurableExecution(arn string) (map[string]any, error) {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.extras.durableExecutions[arn]
	if rec == nil {
		return nil, ErrNotFound
	}
	return cloneAnyMap(rec), nil
}

func (s *Service) GetDurableExecutionHistory(arn string, maxItems, marker int) ([]map[string]any, string, error) {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return nil, "", ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.extras.durableExecutions[arn] == nil {
		return nil, "", ErrNotFound
	}
	items := cloneAnyMapSlice(s.extras.durableExecutionHistory[arn])
	paged, next := paginateLambdaSlice(items, maxItems, marker)
	return paged, next, nil
}

func (s *Service) GetDurableExecutionState(arn string) (map[string]any, error) {
	rec, err := s.GetDurableExecution(arn)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"DurableExecutionArn": rec["DurableExecutionArn"],
		"Status":              rec["Status"],
		"Output":              rec["Output"],
		"FailureReason":       rec["FailureReason"],
	}, nil
}

func (s *Service) CheckpointDurableExecution(arn string, payload map[string]any) (map[string]any, error) {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return nil, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.extras.durableExecutions[arn]
	if rec == nil {
		rec = map[string]any{
			"DurableExecutionArn": arn,
			"FunctionArn":         stringValue(payload["FunctionArn"]),
			"Status":              "RUNNING",
			"CreatedAt":           time.Now().UTC().Format(time.RFC3339),
		}
		s.extras.durableExecutions[arn] = rec
	}
	rec["LastUpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
	if v, ok := payload["Output"]; ok {
		rec["Output"] = v
	}
	entry := map[string]any{
		"Type":      "CHECKPOINT",
		"Timestamp": time.Now().UTC().Format(time.RFC3339),
		"Details":   cloneAnyMap(payload),
	}
	s.extras.durableExecutionHistory[arn] = append(s.extras.durableExecutionHistory[arn], entry)
	return map[string]any{}, nil
}

func (s *Service) StopDurableExecution(arn string, payload map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.extras.durableExecutions[strings.TrimSpace(arn)]
	if rec == nil {
		return nil, ErrNotFound
	}
	rec["Status"] = "STOPPED"
	rec["FailureReason"] = stringValue(payload["Reason"])
	rec["LastUpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
	s.extras.durableExecutionHistory[arn] = append(s.extras.durableExecutionHistory[arn], map[string]any{
		"Type":      "STOP",
		"Timestamp": time.Now().UTC().Format(time.RFC3339),
		"Details":   cloneAnyMap(payload),
	})
	return map[string]any{}, nil
}

func (s *Service) DurableExecutionCallback(callbackID, status string, payload map[string]any) (map[string]any, error) {
	if strings.TrimSpace(callbackID) == "" {
		return nil, ErrInvalidParameter
	}
	_ = status
	_ = payload
	return map[string]any{}, nil
}

func (s *Service) layerVersionLocked(layerName, version string) map[string]any {
	layer := s.extras.layers[strings.TrimSpace(layerName)]
	if layer == nil {
		return nil
	}
	return layer.versions[strings.TrimSpace(version)]
}

func codeSigningConfigARN(id string) string {
	return fmt.Sprintf("arn:aws:lambda:%s:%s:code-signing-config:%s", DefaultRegion, DefaultAccountID, id)
}

func layerARN(name string) string {
	return fmt.Sprintf("arn:aws:lambda:%s:%s:layer:%s", DefaultRegion, DefaultAccountID, strings.TrimSpace(name))
}

func layerVersionARN(name, version string) string {
	return fmt.Sprintf("arn:aws:lambda:%s:%s:layer:%s:%s", DefaultRegion, DefaultAccountID, strings.TrimSpace(name), strings.TrimSpace(version))
}

func capacityProviderARN(name string) string {
	return fmt.Sprintf("arn:aws:lambda:%s:%s:capacity-provider:%s", DefaultRegion, DefaultAccountID, strings.TrimSpace(name))
}

func parseLayerVersionARN(arn string) (string, string) {
	idx := strings.Index(arn, ":layer:")
	if idx < 0 {
		return "", ""
	}
	rest := arn[idx+len(":layer:"):]
	parts := strings.Split(rest, ":")
	if len(parts) < 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func resolveCodeSigningConfigLocked(configs map[string]map[string]any, ref string) map[string]any {
	_, cfg := resolveCodeSigningConfigWithARNLocked(configs, ref)
	return cfg
}

func resolveCodeSigningConfigWithARNLocked(configs map[string]map[string]any, ref string) (string, map[string]any) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	if cfg := configs[ref]; cfg != nil {
		return ref, cfg
	}
	for arn, cfg := range configs {
		if strings.EqualFold(strings.TrimSpace(stringValue(cfg["CodeSigningConfigId"])), ref) {
			return arn, cfg
		}
	}
	return "", nil
}

func functionQualifierKey(ref, qualifier string) string {
	name, q := parseFunctionRef(ref)
	name = strings.TrimSpace(name)
	q = strings.TrimSpace(firstNonEmpty(strings.TrimSpace(qualifier), q, "$LATEST"))
	return name + "|" + q
}

func boolState(v any, def bool) string {
	if b, ok := boolValue(v, def); ok {
		if b {
			return "Enabled"
		}
		return "Disabled"
	}
	if def {
		return "Enabled"
	}
	return "Disabled"
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneAny(v)
	}
	return out
}

func cloneAnyMapSlice(in []map[string]any) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, cloneAnyMap(item))
	}
	return out
}

func cloneAny(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		return cloneAnyMap(tv)
	case []map[string]any:
		return cloneAnyMapSlice(tv)
	case []any:
		out := make([]any, 0, len(tv))
		for _, item := range tv {
			out = append(out, cloneAny(item))
		}
		return out
	case []string:
		return append([]string(nil), tv...)
	default:
		return tv
	}
}

func stringValue(v any) string {
	switch tv := v.(type) {
	case string:
		return tv
	case fmt.Stringer:
		return tv.String()
	default:
		return ""
	}
}

func int32Value(v any, def int32) int32 {
	switch tv := v.(type) {
	case int:
		return int32(tv)
	case int32:
		return tv
	case int64:
		return int32(tv)
	case float64:
		return int32(tv)
	case float32:
		return int32(tv)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(tv))
		if err == nil {
			return int32(n)
		}
	}
	return def
}

func int64Value(v string, def int64) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return def
	}
	return n
}

func boolValue(v any, def bool) (bool, bool) {
	switch tv := v.(type) {
	case bool:
		return tv, true
	case string:
		s := strings.TrimSpace(strings.ToLower(tv))
		switch s {
		case "true", "1", "yes", "on", "enabled":
			return true, true
		case "false", "0", "no", "off", "disabled":
			return false, true
		}
	case float64:
		return tv != 0, true
	case int:
		return tv != 0, true
	}
	return def, false
}

func mapValue(v any) map[string]any {
	if v == nil {
		return nil
	}
	if out, ok := v.(map[string]any); ok {
		return cloneAnyMap(out)
	}
	return nil
}

func sliceValue(v any) []string {
	switch tv := v.(type) {
	case []string:
		return append([]string(nil), tv...)
	case []any:
		out := make([]string, 0, len(tv))
		for _, item := range tv {
			if s := strings.TrimSpace(stringValue(item)); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return []string{}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
