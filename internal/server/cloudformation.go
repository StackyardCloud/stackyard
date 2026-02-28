package server

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const cloudFormationNamespace = "http://cloudformation.amazonaws.com/doc/2010-05-15/"

type cloudFormationStore struct {
	mu         sync.Mutex
	nextID     int64
	stacks     map[string]string
	stackSets  map[string]string
	changeSets map[string]string
}

func newCloudFormationStore() *cloudFormationStore {
	return &cloudFormationStore{
		nextID: 1,
		stacks: map[string]string{
			"stackyard-default": "arn:aws:cloudformation:us-east-1:123456789012:stack/stackyard-default/000000000001",
		},
		stackSets: map[string]string{
			"stackyard-stackset": "arn:aws:cloudformation:us-east-1:123456789012:stackset/stackyard-stackset:000000000001",
		},
		changeSets: map[string]string{
			"stackyard-changeset": "arn:aws:cloudformation:us-east-1:123456789012:changeSet/stackyard-changeset/000000000001",
		},
	}
}

func (s *Server) handleCloudFormationQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudFormationQueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "cloudformation")
	if !ok {
		respondCloudFormationError(w, status, code, msg)
		return true
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondCloudFormationError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondCloudFormationError(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form body")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondCloudFormationError(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return true
	}
	if _, ok := cloudFormationOperationByName[action]; !ok {
		respondCloudFormationError(w, http.StatusBadRequest, "InvalidAction", "unknown operation")
		return true
	}

	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2010-05-15" {
		respondCloudFormationError(w, http.StatusBadRequest, "InvalidParameterValue", "unsupported Version")
		return true
	}

	response := s.cloudformation.Handle(action, r.Form)
	respondCloudFormationXML(w, http.StatusOK, action, response)
	return true
}

func isCloudFormationQueryCandidate(r *http.Request) bool {
	if strings.Contains(strings.ToLower(r.Host), "cloudformation") {
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/cloudformation") {
		return true
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "cloudformation" {
		return false
	}

	action := strings.TrimSpace(r.URL.Query().Get("Action"))
	if action != "" {
		if service != "cloudformation" {
			if _, ok := cloudFormationOperationByName[action]; !ok {
				return false
			}
		}
		if version := strings.TrimSpace(r.URL.Query().Get("Version")); version != "" && version != "2010-05-15" {
			return false
		}
		return true
	}

	if r.Method != http.MethodPost {
		return false
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
		return false
	}
	body, err := readBodyBytes(r)
	if err != nil {
		return false
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return false
	}
	action = strings.TrimSpace(values.Get("Action"))
	if action == "" {
		return false
	}
	if service != "cloudformation" {
		if _, ok := cloudFormationOperationByName[action]; !ok {
			return false
		}
	}
	if version := strings.TrimSpace(values.Get("Version")); version != "" && version != "2010-05-15" {
		return false
	}
	return true
}

func (s *cloudFormationStore) Handle(action string, form url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateStack":
		stackName := cloudFormationFormString(form, "StackName", "stackyard-stack")
		stackID := fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:stack/%s/%s", stackName, s.nextTokenLocked(12))
		s.stacks[stackName] = stackID
		return map[string]any{"StackId": stackID}

	case "DeleteStack":
		stackName := cloudFormationFormString(form, "StackName", "")
		if stackName != "" {
			delete(s.stacks, stackName)
		}
		return map[string]any{}

	case "DescribeStacks":
		stackName := cloudFormationFormString(form, "StackName", "")
		stacks := make([]any, 0, len(s.stacks))
		for name, id := range s.stacks {
			if stackName != "" && stackName != name && stackName != id {
				continue
			}
			stacks = append(stacks, map[string]any{
				"StackId":         id,
				"StackName":       name,
				"CreationTime":    now,
				"StackStatus":     "CREATE_COMPLETE",
				"DisableRollback": false,
			})
		}
		if len(stacks) == 0 {
			stacks = append(stacks, map[string]any{
				"StackId":      "arn:aws:cloudformation:us-east-1:123456789012:stack/stackyard-default/000000000001",
				"StackName":    "stackyard-default",
				"CreationTime": now,
				"StackStatus":  "CREATE_COMPLETE",
			})
		}
		return map[string]any{"Stacks": stacks}

	case "ListStacks":
		summaries := make([]any, 0, len(s.stacks))
		for name, id := range s.stacks {
			summaries = append(summaries, map[string]any{
				"StackId":          id,
				"StackName":        name,
				"CreationTime":     now,
				"StackStatus":      "CREATE_COMPLETE",
				"DriftInformation": map[string]any{"StackDriftStatus": "NOT_CHECKED"},
			})
		}
		if len(summaries) == 0 {
			summaries = append(summaries, map[string]any{
				"StackId":      "arn:aws:cloudformation:us-east-1:123456789012:stack/stackyard-default/000000000001",
				"StackName":    "stackyard-default",
				"CreationTime": now,
				"StackStatus":  "CREATE_COMPLETE",
			})
		}
		return map[string]any{
			"StackSummaries": summaries,
			"NextToken":      "",
		}

	case "UpdateStack":
		stackName := cloudFormationFormString(form, "StackName", "stackyard-stack")
		stackID := s.stacks[stackName]
		if stackID == "" {
			stackID = fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:stack/%s/%s", stackName, s.nextTokenLocked(12))
			s.stacks[stackName] = stackID
		}
		return map[string]any{"StackId": stackID}

	case "CreateChangeSet":
		name := cloudFormationFormString(form, "ChangeSetName", "stackyard-changeset")
		id := fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:changeSet/%s/%s", name, s.nextTokenLocked(12))
		s.changeSets[name] = id
		return map[string]any{
			"Id":      id,
			"StackId": cloudFormationFormString(form, "StackName", "arn:aws:cloudformation:us-east-1:123456789012:stack/stackyard-default/000000000001"),
		}

	case "DescribeChangeSet":
		name := cloudFormationFormString(form, "ChangeSetName", "stackyard-changeset")
		id := s.changeSets[name]
		if id == "" {
			id = fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:changeSet/%s/%s", name, s.nextTokenLocked(12))
			s.changeSets[name] = id
		}
		return map[string]any{
			"ChangeSetName":       name,
			"ChangeSetId":         id,
			"StackId":             cloudFormationFormString(form, "StackName", "arn:aws:cloudformation:us-east-1:123456789012:stack/stackyard-default/000000000001"),
			"Status":              "CREATE_COMPLETE",
			"ExecutionStatus":     "AVAILABLE",
			"NotificationARNs":    []any{},
			"Capabilities":        []any{"CAPABILITY_IAM"},
			"IncludeNestedStacks": false,
		}

	case "ListChangeSets":
		items := []any{
			map[string]any{
				"ChangeSetName":   "stackyard-changeset",
				"ChangeSetId":     "arn:aws:cloudformation:us-east-1:123456789012:changeSet/stackyard-changeset/000000000001",
				"StackId":         "arn:aws:cloudformation:us-east-1:123456789012:stack/stackyard-default/000000000001",
				"Status":          "CREATE_COMPLETE",
				"ExecutionStatus": "AVAILABLE",
			},
		}
		return map[string]any{"Summaries": items, "NextToken": ""}

	case "CreateStackSet":
		name := cloudFormationFormString(form, "StackSetName", "stackyard-stackset")
		id := fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:stackset/%s:%s", name, s.nextTokenLocked(12))
		s.stackSets[name] = id
		return map[string]any{"StackSetId": id}

	case "ListStackSets":
		items := make([]any, 0, len(s.stackSets))
		for name, id := range s.stackSets {
			items = append(items, map[string]any{
				"StackSetName":    name,
				"StackSetId":      id,
				"Status":          "ACTIVE",
				"DriftStatus":     "NOT_CHECKED",
				"PermissionModel": "SELF_MANAGED",
			})
		}
		if len(items) == 0 {
			items = append(items, map[string]any{
				"StackSetName": "stackyard-stackset",
				"StackSetId":   "arn:aws:cloudformation:us-east-1:123456789012:stackset/stackyard-stackset:000000000001",
				"Status":       "ACTIVE",
			})
		}
		return map[string]any{"Summaries": items, "NextToken": ""}

	case "DescribeStackSet":
		name := cloudFormationFormString(form, "StackSetName", "stackyard-stackset")
		id := s.stackSets[name]
		if id == "" {
			id = fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:stackset/%s:%s", name, s.nextTokenLocked(12))
			s.stackSets[name] = id
		}
		return map[string]any{
			"StackSet": map[string]any{
				"StackSetName":    name,
				"StackSetId":      id,
				"Status":          "ACTIVE",
				"PermissionModel": "SELF_MANAGED",
			},
		}

	case "DescribeAccountLimits":
		return map[string]any{
			"AccountLimits": []any{
				map[string]any{"Name": "StackLimit", "Value": "2000"},
				map[string]any{"Name": "StacksPerRegionLimit", "Value": "2000"},
			},
		}

	case "ValidateTemplate":
		return map[string]any{
			"Description":        "Stackyard template validation",
			"Capabilities":       []any{"CAPABILITY_IAM"},
			"CapabilitiesReason": "",
			"DeclaredTransforms": []any{},
			"Parameters":         []any{},
		}

	case "GetTemplateSummary":
		return map[string]any{
			"Description":  "Stackyard template summary",
			"Parameters":   []any{},
			"Capabilities": []any{"CAPABILITY_IAM"},
		}

	case "GetTemplate":
		return map[string]any{
			"TemplateBody":    "{\"AWSTemplateFormatVersion\":\"2010-09-09\",\"Resources\":{}}",
			"StagesAvailable": []any{"Original", "Processed"},
		}

	case "EstimateTemplateCost":
		return map[string]any{"Url": "https://calculator.aws/#stackyard"}
	}

	switch {
	case strings.HasPrefix(action, "List"):
		return map[string]any{"NextToken": ""}
	case strings.HasPrefix(action, "Describe"):
		return map[string]any{}
	case strings.HasPrefix(action, "Get"):
		return map[string]any{}
	case strings.HasPrefix(action, "Create"):
		return map[string]any{"Id": "stackyard-" + strings.ToLower(action)}
	case strings.HasPrefix(action, "Update"),
		strings.HasPrefix(action, "Delete"),
		strings.HasPrefix(action, "Execute"),
		strings.HasPrefix(action, "Cancel"),
		strings.HasPrefix(action, "Continue"),
		strings.HasPrefix(action, "Activate"),
		strings.HasPrefix(action, "Deactivate"),
		strings.HasPrefix(action, "Detect"),
		strings.HasPrefix(action, "Import"),
		strings.HasPrefix(action, "Publish"),
		strings.HasPrefix(action, "Record"),
		strings.HasPrefix(action, "Register"),
		strings.HasPrefix(action, "Rollback"),
		strings.HasPrefix(action, "Set"),
		strings.HasPrefix(action, "Signal"),
		strings.HasPrefix(action, "Start"),
		strings.HasPrefix(action, "Stop"),
		strings.HasPrefix(action, "Test"):
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func (s *cloudFormationStore) nextTokenLocked(width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%0*d", width, id)
}

func cloudFormationFormString(form url.Values, key, fallback string) string {
	value := strings.TrimSpace(form.Get(key))
	if value == "" {
		return fallback
	}
	return value
}

func respondCloudFormationXML(w http.ResponseWriter, status int, action string, result map[string]any) {
	if result == nil {
		result = map[string]any{}
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<%sResponse xmlns=\"%s\">", action, cloudFormationNamespace)
	fmt.Fprintf(&buf, "<%sResult>", action)
	writeCloudFormationMap(&buf, result)
	fmt.Fprintf(&buf, "</%sResult>", action)
	buf.WriteString("<ResponseMetadata><RequestId>stackyard-request</RequestId></ResponseMetadata>")
	fmt.Fprintf(&buf, "</%sResponse>", action)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func respondCloudFormationError(w http.ResponseWriter, status int, code, message string) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<ErrorResponse xmlns=\"%s\">", cloudFormationNamespace)
	buf.WriteString("<Error><Type>Sender</Type>")
	writeCloudFormationTextElement(&buf, "Code", code)
	writeCloudFormationTextElement(&buf, "Message", message)
	buf.WriteString("</Error><RequestId>stackyard-request</RequestId></ErrorResponse>")

	w.Header().Set("X-Amzn-ErrorType", code)
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeCloudFormationMap(buf *bytes.Buffer, values map[string]any) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeCloudFormationValue(buf, key, values[key])
	}
}

func writeCloudFormationValue(buf *bytes.Buffer, name string, value any) {
	switch v := value.(type) {
	case nil:
		fmt.Fprintf(buf, "<%s/>", name)
	case string:
		writeCloudFormationTextElement(buf, name, v)
	case bool:
		if v {
			writeCloudFormationTextElement(buf, name, "true")
		} else {
			writeCloudFormationTextElement(buf, name, "false")
		}
	case time.Time:
		writeCloudFormationTextElement(buf, name, v.UTC().Format(time.RFC3339))
	case map[string]any:
		fmt.Fprintf(buf, "<%s>", name)
		writeCloudFormationMap(buf, v)
		fmt.Fprintf(buf, "</%s>", name)
	case []any:
		fmt.Fprintf(buf, "<%s>", name)
		for _, item := range v {
			writeCloudFormationValue(buf, "member", item)
		}
		fmt.Fprintf(buf, "</%s>", name)
	case []string:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeCloudFormationValue(buf, name, items)
	case []map[string]any:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeCloudFormationValue(buf, name, items)
	default:
		writeCloudFormationTextElement(buf, name, fmt.Sprintf("%v", v))
	}
}

func writeCloudFormationTextElement(buf *bytes.Buffer, name, value string) {
	fmt.Fprintf(buf, "<%s>", name)
	_ = xml.EscapeText(buf, []byte(value))
	fmt.Fprintf(buf, "</%s>", name)
}
