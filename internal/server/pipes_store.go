package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	pipesAccountID = "123456789012"
	// pipesMaxPipes bounds the store; AWS's default quota is 1,000 per Region.
	pipesMaxPipes = 1000
	pipesMaxTags  = 50
)

// pipesPipe is one stored pipe. Fields are typed so request JSON can only
// reach the members the API defines.
type pipesPipe struct {
	Name, Arn, Description              string
	DesiredState, CurrentState          string
	StateReason                         string
	Source, Enrichment, Target, RoleArn string
	KmsKeyIdentifier                    string
	SourceParameters                    map[string]any
	EnrichmentParameters                map[string]any
	TargetParameters                    map[string]any
	LogConfiguration                    map[string]any
	Tags                                map[string]string
	CreationTime, LastModifiedTime      time.Time

	// failure, when set, is why the pipe can't run (a missing source,
	// enrichment or target); the next read settles it into a *_FAILED state.
	failure string
}

type pipesStore struct {
	mu    sync.Mutex
	pipes map[string]*pipesPipe
}

func newPipesStore() *pipesStore {
	return &pipesStore{pipes: map[string]*pipesPipe{}}
}

type pipesError struct {
	status int
	code   string
	msg    string
	fields []map[string]string
}

func pipesValidation(msg string) *pipesError {
	return &pipesError{status: 400, code: "ValidationException", msg: msg}
}

func pipesNotFound(name string) *pipesError {
	return &pipesError{status: 404, code: "NotFoundException", msg: fmt.Sprintf("Pipe %s does not exist.", name)}
}

func pipesConflict(name, msg string) *pipesError {
	return &pipesError{status: 409, code: "ConflictException", msg: msg, fields: []map[string]string{{"resourceId": name, "resourceType": "pipe"}}}
}

func pipesARN(name string) string {
	return "arn:aws:pipes:" + defaultSigV4Region + ":" + pipesAccountID + ":pipe/" + name
}

// settleLocked advances a pipe one step out of a transitional state. Reads
// (DescribePipe, ListPipes) settle; changes don't, so a change made while the
// pipe is still CREATING/UPDATING/STARTING/STOPPING gets ConflictException,
// as in AWS.
func (p *pipesPipe) settleLocked() {
	switch p.CurrentState {
	case "CREATING", "UPDATING", "STARTING":
		if p.failure != "" {
			p.CurrentState = map[string]string{"CREATING": "CREATE_FAILED", "UPDATING": "UPDATE_FAILED", "STARTING": "START_FAILED"}[p.CurrentState]
			p.StateReason = p.failure
			return
		}
		p.StateReason = ""
		if p.DesiredState == "STOPPED" {
			p.CurrentState = "STOPPED"
		} else {
			p.CurrentState = "RUNNING"
		}
	case "STOPPING":
		p.CurrentState, p.StateReason = "STOPPED", ""
	}
}

// pipeLocked returns a pipe by name, settled when read is set.
func (st *pipesStore) pipeLocked(name string, read bool) (*pipesPipe, *pipesError) {
	p, ok := st.pipes[name]
	if !ok {
		return nil, pipesNotFound(name)
	}
	if read {
		p.settleLocked()
	}
	return p, nil
}

// pipeByARNLocked resolves a pipe ARN (the tagging resource).
func (st *pipesStore) pipeByARNLocked(arn string) (*pipesPipe, *pipesError) {
	name, ok := strings.CutPrefix(arn, "arn:aws:pipes:"+defaultSigV4Region+":"+pipesAccountID+":pipe/")
	if !ok || name == "" {
		return nil, &pipesError{status: 404, code: "NotFoundException", msg: "Resource " + arn + " does not exist."}
	}
	p, ok := st.pipes[name]
	if !ok {
		return nil, &pipesError{status: 404, code: "NotFoundException", msg: "Resource " + arn + " does not exist."}
	}
	return p, nil
}

// pipesValidateShape checks a request body against the botocore input shape.
func pipesValidateShape(shape string, in map[string]any) *pipesError {
	problems := awsmodels.PipesModel().Validate(shape, in)
	if len(problems) == 0 {
		return nil
	}
	err := pipesValidation(strings.Join(problems, "; "))
	for _, p := range problems {
		name, reason, _ := strings.Cut(p, ": ")
		err.fields = append(err.fields, map[string]string{"name": name, "message": reason})
	}
	return err
}

func pipesStr(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func pipesObj(m map[string]any, key string) map[string]any {
	o, _ := m[key].(map[string]any)
	return o
}

// pipesARNService returns the service and resource part of an ARN.
func pipesARNService(arn string) (service, resource string) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 {
		return "", ""
	}
	return parts[2], parts[5]
}

// pipesSourceKind maps a source ARN/URL to the SourceParameters member that
// configures it ("" when the source type isn't supported by Pipes).
func pipesSourceKind(source string) string {
	if strings.HasPrefix(source, "smk://") {
		return "SelfManagedKafkaParameters"
	}
	service, resource := pipesARNService(source)
	switch {
	case service == "sqs":
		return "SqsQueueParameters"
	case service == "kinesis" && strings.HasPrefix(resource, "stream/"):
		return "KinesisStreamParameters"
	case service == "dynamodb" && strings.Contains(resource, "/stream/"):
		return "DynamoDBStreamParameters"
	case service == "kafka":
		return "ManagedStreamingKafkaParameters"
	case service == "mq":
		// The broker engine isn't in the ARN; either MQ member is accepted.
		return "ActiveMQBrokerParameters|RabbitMQBrokerParameters"
	}
	return ""
}

// pipesTargetKind maps a target ARN to the TargetParameters member that
// configures it ("" for an unsupported target, "-" for targets without one).
func pipesTargetKind(target string) string {
	service, resource := pipesARNService(target)
	switch service {
	case "lambda":
		return "LambdaFunctionParameters"
	case "states":
		return "StepFunctionStateMachineParameters"
	case "kinesis":
		return "KinesisStreamParameters"
	case "ecs":
		return "EcsTaskParameters"
	case "batch":
		return "BatchJobParameters"
	case "sqs":
		return "SqsQueueParameters"
	case "execute-api":
		return "HttpParameters"
	case "events":
		if strings.HasPrefix(resource, "api-destination/") {
			return "HttpParameters"
		}
		return "EventBridgeEventBusParameters"
	case "redshift":
		return "RedshiftDataParameters"
	case "sagemaker":
		return "SageMakerPipelineParameters"
	case "logs":
		return "CloudWatchLogsParameters"
	case "timestream":
		return "TimestreamParameters"
	case "sns", "firehose", "inspector2":
		return "-"
	}
	return ""
}

// pipesCheckParams enforces that a parameters object only configures the
// member matching its resource type (InputTemplate/FilterCriteria are shared).
func pipesCheckParams(field string, params map[string]any, allowed string, shared ...string) *pipesError {
	for key := range params {
		if pipesContains(shared, key) || (allowed != "-" && pipesContains(strings.Split(allowed, "|"), key)) {
			continue
		}
		return pipesValidation(fmt.Sprintf("%s.%s does not apply to this resource type.", field, key))
	}
	return nil
}

func pipesContains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// pipesCheckFilters requires every filter pattern to be a JSON object.
func pipesCheckFilters(sourceParams map[string]any) *pipesError {
	filters, _ := pipesObj(sourceParams, "FilterCriteria")["Filters"].([]any)
	for i, f := range filters {
		filter, _ := f.(map[string]any)
		pattern := pipesStr(filter, "Pattern")
		if pattern == "" {
			continue
		}
		var obj map[string]any
		if json.Unmarshal([]byte(pattern), &obj) != nil {
			return pipesValidation(fmt.Sprintf("SourceParameters.FilterCriteria.Filters[%d].Pattern is not a valid JSON event pattern.", i))
		}
	}
	return nil
}

// pipesCheckEnrichment requires a Lambda function, Step Functions state
// machine, API destination or API Gateway ARN, and only HttpParameters for
// the HTTP ones.
func pipesCheckEnrichment(enrichment string, params map[string]any) *pipesError {
	if enrichment == "" {
		if len(params) > 0 && params["HttpParameters"] != nil {
			return pipesValidation("EnrichmentParameters.HttpParameters requires an Enrichment.")
		}
		return nil
	}
	service, resource := pipesARNService(enrichment)
	http := service == "execute-api" || (service == "events" && strings.HasPrefix(resource, "api-destination/"))
	if !http && service != "lambda" && service != "states" {
		return pipesValidation("Enrichment must be a Lambda function, Step Functions state machine, API destination or API Gateway ARN.")
	}
	if !http && params["HttpParameters"] != nil {
		return pipesValidation("EnrichmentParameters.HttpParameters only applies to API destination and API Gateway enrichments.")
	}
	return nil
}

// pipesCheckLogConfiguration requires a destination for any level but OFF.
func pipesCheckLogConfiguration(cfg map[string]any) *pipesError {
	if cfg == nil || pipesStr(cfg, "Level") == "OFF" {
		return nil
	}
	if cfg["S3LogDestination"] == nil && cfg["FirehoseLogDestination"] == nil && cfg["CloudwatchLogsLogDestination"] == nil {
		return pipesValidation("LogConfiguration must specify at least one log destination.")
	}
	return nil
}

// pipesCheckDefinition validates the parts of a pipe a create or update sets.
func pipesCheckDefinition(source string, in map[string]any, target string) *pipesError {
	sourceKind := pipesSourceKind(source)
	if sourceKind == "" {
		return pipesValidation("Source must be an Amazon SQS queue, Kinesis stream, DynamoDB stream, Amazon MQ broker, Amazon MSK cluster or self-managed Kafka (smk://) endpoint.")
	}
	sourceParams := pipesObj(in, "SourceParameters")
	if err := pipesCheckParams("SourceParameters", sourceParams, sourceKind, "FilterCriteria"); err != nil {
		return err
	}
	if err := pipesCheckFilters(sourceParams); err != nil {
		return err
	}
	if err := pipesCheckEnrichment(pipesStr(in, "Enrichment"), pipesObj(in, "EnrichmentParameters")); err != nil {
		return err
	}
	targetKind := pipesTargetKind(target)
	if targetKind == "" {
		return pipesValidation("Target " + target + " is not a supported Pipes target.")
	}
	if err := pipesCheckParams("TargetParameters", pipesObj(in, "TargetParameters"), targetKind, "InputTemplate"); err != nil {
		return err
	}
	return pipesCheckLogConfiguration(pipesObj(in, "LogConfiguration"))
}

func pipesCheckTags(tags map[string]string) *pipesError {
	for k := range tags {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return pipesValidation("Tag keys can't start with aws:.")
		}
	}
	if len(tags) > pipesMaxTags {
		return pipesValidation(fmt.Sprintf("A pipe can have at most %d tags.", pipesMaxTags))
	}
	return nil
}

func pipesStringMap(v any) map[string]string {
	obj, _ := v.(map[string]any)
	out := make(map[string]string, len(obj))
	for k, val := range obj {
		out[k], _ = val.(string)
	}
	return out
}

func pipesTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// summaryLocked is the CreatePipe/UpdatePipe/StartPipe/StopPipe/DeletePipe
// response: identity and state only.
func (p *pipesPipe) summaryLocked() map[string]any {
	return map[string]any{
		"Arn": p.Arn, "Name": p.Name, "DesiredState": p.DesiredState, "CurrentState": p.CurrentState,
		"CreationTime": pipesTime(p.CreationTime), "LastModifiedTime": pipesTime(p.LastModifiedTime),
	}
}

// listEntryLocked is one ListPipes entry.
func (p *pipesPipe) listEntryLocked() map[string]any {
	out := p.summaryLocked()
	out["Source"], out["Target"] = p.Source, p.Target
	if p.Enrichment != "" {
		out["Enrichment"] = p.Enrichment
	}
	if p.StateReason != "" {
		out["StateReason"] = p.StateReason
	}
	return out
}

// describeLocked is the full DescribePipe response, conformed to the model
// so only API members leave the store (and nothing aliases stored maps).
func (p *pipesPipe) describeLocked() map[string]any {
	out := p.listEntryLocked()
	out["RoleArn"] = p.RoleArn
	tags := make(map[string]any, len(p.Tags))
	for k, v := range p.Tags {
		tags[k] = v
	}
	out["Tags"] = tags
	for key, value := range map[string]any{
		"Description": p.Description, "KmsKeyIdentifier": p.KmsKeyIdentifier,
	} {
		if value != "" {
			out[key] = value
		}
	}
	for key, value := range map[string]map[string]any{
		"SourceParameters": p.SourceParameters, "EnrichmentParameters": p.EnrichmentParameters,
		"TargetParameters": p.TargetParameters, "LogConfiguration": p.LogConfiguration,
	} {
		if value != nil {
			out[key] = value
		}
	}
	return awsmodels.PipesModel().Conform("DescribePipeResponse", out).(map[string]any)
}

func (st *pipesStore) create(name string, in map[string]any, missing string) (map[string]any, *pipesError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, exists := st.pipes[name]; exists {
		return nil, pipesConflict(name, fmt.Sprintf("Pipe %s already exists.", name))
	}
	if len(st.pipes) >= pipesMaxPipes {
		return nil, &pipesError{status: 402, code: "ServiceQuotaExceededException", msg: fmt.Sprintf("The account already has %d pipes.", pipesMaxPipes)}
	}
	tags := pipesStringMap(in["Tags"])
	if err := pipesCheckTags(tags); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	p := &pipesPipe{
		Name: name, Arn: pipesARN(name), Description: pipesStr(in, "Description"),
		DesiredState: pipesDesired(in, "RUNNING"), CurrentState: "CREATING",
		Source: pipesStr(in, "Source"), Enrichment: pipesStr(in, "Enrichment"), Target: pipesStr(in, "Target"),
		RoleArn: pipesStr(in, "RoleArn"), KmsKeyIdentifier: pipesStr(in, "KmsKeyIdentifier"),
		SourceParameters: pipesObj(in, "SourceParameters"), EnrichmentParameters: pipesObj(in, "EnrichmentParameters"),
		TargetParameters: pipesObj(in, "TargetParameters"), LogConfiguration: pipesObj(in, "LogConfiguration"),
		Tags: tags, CreationTime: now, LastModifiedTime: now, failure: missing,
	}
	st.pipes[name] = p
	return p.summaryLocked(), nil
}

func pipesDesired(in map[string]any, fallback string) string {
	if v := pipesStr(in, "DesiredState"); v != "" {
		return v
	}
	return fallback
}

// pipesBusy reports a state no new change can start from.
func pipesBusy(state string) bool {
	switch state {
	case "CREATING", "UPDATING", "STARTING", "STOPPING", "DELETING":
		return true
	}
	return false
}

func (st *pipesStore) update(name string, in map[string]any, resolve func(source, enrichment, target string) string) (map[string]any, *pipesError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeLocked(name, false)
	if err != nil {
		return nil, err
	}
	if pipesBusy(p.CurrentState) {
		return nil, pipesConflict(name, fmt.Sprintf("Pipe %s is %s and can't be updated.", name, p.CurrentState))
	}
	target, enrichment := p.Target, p.Enrichment
	if v, ok := in["Target"].(string); ok {
		target = v
	}
	if v, ok := in["Enrichment"].(string); ok {
		enrichment = v
	}
	sourceParams := p.SourceParameters
	if v := pipesObj(in, "SourceParameters"); v != nil {
		sourceParams = pipesMergeSourceParams(p.SourceParameters, v)
	}
	enrichParams := p.EnrichmentParameters
	if v, ok := in["EnrichmentParameters"].(map[string]any); ok {
		enrichParams = v
	}
	targetParams := p.TargetParameters
	if v, ok := in["TargetParameters"].(map[string]any); ok {
		targetParams = v
	}
	merged := map[string]any{"SourceParameters": sourceParams, "Enrichment": enrichment, "EnrichmentParameters": enrichParams, "TargetParameters": targetParams, "LogConfiguration": pipesObj(in, "LogConfiguration")}
	if cerr := pipesCheckDefinition(p.Source, merged, target); cerr != nil {
		return nil, cerr
	}
	p.Target, p.Enrichment, p.RoleArn = target, enrichment, pipesStr(in, "RoleArn")
	p.SourceParameters, p.EnrichmentParameters, p.TargetParameters = sourceParams, enrichParams, targetParams
	if v, ok := in["Description"].(string); ok {
		p.Description = v
	}
	if v, ok := in["KmsKeyIdentifier"].(string); ok {
		p.KmsKeyIdentifier = v
	}
	if cfg := pipesObj(in, "LogConfiguration"); cfg != nil {
		p.LogConfiguration = cfg
	}
	p.DesiredState = pipesDesired(in, p.DesiredState)
	p.CurrentState, p.StateReason = "UPDATING", ""
	p.failure = resolve(p.Source, p.Enrichment, p.Target)
	p.LastModifiedTime = time.Now().UTC()
	return p.summaryLocked(), nil
}

// pipesMergeSourceParams applies an update's source parameters: FilterCriteria
// is replaced, and each source member's fields are merged over the old ones.
func pipesMergeSourceParams(old, update map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range old {
		out[k] = v
	}
	for k, v := range update {
		prev, _ := out[k].(map[string]any)
		next, _ := v.(map[string]any)
		if k == "FilterCriteria" || prev == nil || next == nil {
			out[k] = v
			continue
		}
		merged := map[string]any{}
		for pk, pv := range prev {
			merged[pk] = pv
		}
		for nk, nv := range next {
			merged[nk] = nv
		}
		out[k] = merged
	}
	return out
}

func (st *pipesStore) setDesired(name, desired string) (map[string]any, *pipesError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeLocked(name, false)
	if err != nil {
		return nil, err
	}
	if pipesBusy(p.CurrentState) {
		return nil, pipesConflict(name, fmt.Sprintf("Pipe %s is %s.", name, p.CurrentState))
	}
	p.DesiredState = desired
	switch {
	case desired == "RUNNING" && p.CurrentState != "RUNNING":
		p.CurrentState, p.StateReason = "STARTING", ""
	case desired == "STOPPED" && p.CurrentState != "STOPPED":
		p.CurrentState, p.StateReason = "STOPPING", ""
	}
	p.LastModifiedTime = time.Now().UTC()
	return p.summaryLocked(), nil
}

func (st *pipesStore) delete(name string) (map[string]any, *pipesError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeLocked(name, false)
	if err != nil {
		return nil, err
	}
	if pipesBusy(p.CurrentState) {
		return nil, pipesConflict(name, fmt.Sprintf("Pipe %s is %s and can't be deleted.", name, p.CurrentState))
	}
	delete(st.pipes, name)
	p.DesiredState, p.CurrentState = "DELETED", "DELETING"
	p.LastModifiedTime = time.Now().UTC()
	return p.summaryLocked(), nil
}

func (st *pipesStore) describe(name string) (map[string]any, *pipesError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeLocked(name, true)
	if err != nil {
		return nil, err
	}
	return p.describeLocked(), nil
}

type pipesListFilter struct {
	namePrefix, desired, current, sourcePrefix, targetPrefix string
	limit                                                    int
	nextToken                                                string
}

func (st *pipesStore) list(f pipesListFilter) (map[string]any, *pipesError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	after := ""
	if f.nextToken != "" {
		raw, err := base64.RawURLEncoding.DecodeString(f.nextToken)
		if err != nil || len(raw) == 0 {
			return nil, pipesValidation("NextToken is invalid.")
		}
		after = string(raw)
	}
	names := make([]string, 0, len(st.pipes))
	for name := range st.pipes {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := []any{}
	next := ""
	for _, name := range names {
		if name <= after {
			continue
		}
		p := st.pipes[name]
		p.settleLocked()
		if !strings.HasPrefix(p.Name, f.namePrefix) || !strings.HasPrefix(p.Source, f.sourcePrefix) || !strings.HasPrefix(p.Target, f.targetPrefix) ||
			(f.desired != "" && p.DesiredState != f.desired) || (f.current != "" && p.CurrentState != f.current) {
			continue
		}
		if len(entries) == f.limit {
			next = base64.RawURLEncoding.EncodeToString([]byte(entries[len(entries)-1].(map[string]any)["Name"].(string)))
			break
		}
		entries = append(entries, p.listEntryLocked())
	}
	out := map[string]any{"Pipes": entries}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func (st *pipesStore) tags(arn string) (map[string]any, *pipesError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeByARNLocked(arn)
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any, len(p.Tags))
	for k, v := range p.Tags {
		tags[k] = v
	}
	return map[string]any{"tags": tags}, nil
}

func (st *pipesStore) tag(arn string, add map[string]string) *pipesError {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeByARNLocked(arn)
	if err != nil {
		return err
	}
	next := make(map[string]string, len(p.Tags)+len(add))
	for k, v := range p.Tags {
		next[k] = v
	}
	for k, v := range add {
		next[k] = v
	}
	if terr := pipesCheckTags(next); terr != nil {
		return terr
	}
	p.Tags = next
	return nil
}

func (st *pipesStore) untag(arn string, keys []string) *pipesError {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeByARNLocked(arn)
	if err != nil {
		return err
	}
	for _, k := range keys {
		delete(p.Tags, k)
	}
	return nil
}
