package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// handleCodeStarNotificationsRESTRouter serves AWS CodeStar Notifications
// (REST-JSON, signed as codestar-notifications).
func (s *Server) handleCodeStarNotificationsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	op, ok := csnOperationFor(path)
	if !ok || !isCodeStarNotificationsCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads; refuse unknown lengths.
	if r.ContentLength < 0 || r.ContentLength > csnMaxBody {
		respondCSNError(w, csnValidation("Request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "codestar-notifications"); !ok {
		respondCSNError(w, csnErr(status, code, "%s", msg))
		return true
	}
	if r.Method != http.MethodPost {
		respondCSNError(w, csnErr(http.StatusNotImplemented, "NotImplementedException", "%s %s is not a CodeStar Notifications operation", r.Method, path))
		return true
	}
	out, err := s.csnDispatch(op.Name, r, path)
	if err != nil {
		respondCSNError(w, err)
		return true
	}
	payload, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
	return true
}

func csnOperationFor(path string) (codestarnotificationsOperation, bool) {
	if strings.HasPrefix(path, "/untagResource/") {
		return codestarnotificationsOperationByPath["/untagResource/"], true
	}
	op, ok := codestarnotificationsOperationByPath[path]
	return op, ok && path != "/untagResource/"
}

func isCodeStarNotificationsCandidate(r *http.Request) bool {
	service := sigV4ServiceHint(r)
	return service == "codestar-notifications" ||
		(service == "" && strings.HasPrefix(strings.ToLower(r.Host), "codestar-notifications."))
}

func respondCSNError(w http.ResponseWriter, err *csnError) {
	payload, _ := json.Marshal(map[string]string{"Message": err.msg})
	w.Header().Set("X-Amzn-ErrorType", err.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.status)
	_, _ = w.Write(payload)
}

// csnDecode decodes a JSON body into v, rejecting members the model doesn't
// define so callers can't set anything else.
func csnDecode(r *http.Request, v any) *csnError {
	body, err := readBodyBytes(r)
	if err != nil {
		return csnValidation("unable to read request body")
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return csnValidation("Invalid request body: %v", err)
	}
	return nil
}

type csnListInput struct {
	Filters    []csnFilter `json:"Filters"`
	NextToken  string      `json:"NextToken"`
	MaxResults *int32      `json:"MaxResults"`
}

// check validates filters against the allowed names and returns the page size.
func (in csnListInput) check(names ...string) (int, *csnError) {
	if len(in.Filters) > 20 {
		return 0, csnValidation("At most 20 filters are allowed")
	}
	for _, f := range in.Filters {
		if !containsString(names, f.Name) {
			return 0, csnValidation("Filter name must be one of %s", strings.Join(names, ", "))
		}
		if f.Value == "" || len(f.Value) > 2048 {
			return 0, csnValidation("Filter value must be 1-2048 characters")
		}
	}
	if in.NextToken != "" && !csnNextTokenPattern.MatchString(in.NextToken) {
		return 0, csnErr(http.StatusBadRequest, "InvalidNextTokenException", "The specified NextToken is not valid")
	}
	if in.MaxResults == nil {
		return 100, nil
	}
	if *in.MaxResults < 1 || *in.MaxResults > 100 {
		return 0, csnValidation("MaxResults must be from 1 to 100")
	}
	return int(*in.MaxResults), nil
}

func (s *Server) csnDispatch(op string, r *http.Request, path string) (map[string]any, *csnError) {
	st := s.codestarNotifications
	switch op {
	case "CreateNotificationRule":
		var in csnCreateInput
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		return s.csnCreate(in)
	case "UpdateNotificationRule":
		var in csnUpdateInput
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		return map[string]any{}, s.csnUpdate(in)
	case "DeleteNotificationRule", "DescribeNotificationRule", "ListTagsForResource":
		var in struct {
			Arn string `json:"Arn"`
		}
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		if err := csnValidRuleARN(in.Arn); err != nil {
			return nil, err
		}
		switch op {
		case "DeleteNotificationRule":
			st.delete(in.Arn)
			return map[string]any{"Arn": in.Arn}, nil
		case "DescribeNotificationRule":
			return st.describe(in.Arn, s.csnTargetStatus)
		default:
			return st.listTags(in.Arn)
		}
	case "ListEventTypes":
		var in csnListInput
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		max, err := in.check("RESOURCE_TYPE", "SERVICE_NAME")
		if err != nil {
			return nil, err
		}
		return csnListEventTypes(in.Filters, in.NextToken, max)
	case "ListNotificationRules":
		var in csnListInput
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		max, err := in.check("EVENT_TYPE_ID", "CREATED_BY", "RESOURCE", "TARGET_ADDRESS")
		if err != nil {
			return nil, err
		}
		return st.listRules(in.Filters, in.NextToken, max)
	case "ListTargets":
		var in csnListInput
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		max, err := in.check("TARGET_TYPE", "TARGET_ADDRESS", "TARGET_STATUS")
		if err != nil {
			return nil, err
		}
		return st.listTargets(in.Filters, in.NextToken, max, s.csnTargetStatus)
	case "Subscribe":
		var in struct {
			Arn                string     `json:"Arn"`
			Target             *csnTarget `json:"Target"`
			ClientRequestToken string     `json:"ClientRequestToken"`
		}
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		if err := csnValidRuleARN(in.Arn); err != nil {
			return nil, err
		}
		if in.Target == nil {
			return nil, csnValidation("Target is required")
		}
		if err := csnValidToken(in.ClientRequestToken); err != nil {
			return nil, err
		}
		if err := s.csnCheckTargets([]csnTarget{*in.Target}); err != nil {
			return nil, err
		}
		return map[string]any{"Arn": in.Arn}, st.subscribe(in.Arn, *in.Target)
	case "Unsubscribe":
		var in struct {
			Arn           string `json:"Arn"`
			TargetAddress string `json:"TargetAddress"`
		}
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		if err := csnValidRuleARN(in.Arn); err != nil {
			return nil, err
		}
		if err := csnValidAddress(in.TargetAddress); err != nil {
			return nil, err
		}
		return map[string]any{"Arn": in.Arn}, st.unsubscribe(in.Arn, in.TargetAddress)
	case "DeleteTarget":
		var in struct {
			TargetAddress       string `json:"TargetAddress"`
			ForceUnsubscribeAll bool   `json:"ForceUnsubscribeAll"`
		}
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		if err := csnValidAddress(in.TargetAddress); err != nil {
			return nil, err
		}
		return map[string]any{}, st.deleteTarget(in.TargetAddress, in.ForceUnsubscribeAll)
	case "TagResource":
		var in struct {
			Arn  string            `json:"Arn"`
			Tags map[string]string `json:"Tags"`
		}
		if err := csnDecode(r, &in); err != nil {
			return nil, err
		}
		if err := csnValidRuleARN(in.Arn); err != nil {
			return nil, err
		}
		if in.Tags == nil {
			return nil, csnValidation("Tags is required")
		}
		if err := csnValidTags(in.Tags); err != nil {
			return nil, err
		}
		return st.tag(in.Arn, in.Tags)
	case "UntagResource":
		arn, err := url.PathUnescape(strings.TrimPrefix(path, "/untagResource/"))
		if err != nil {
			return nil, csnValidation("invalid resource ARN in path")
		}
		if verr := csnValidRuleARN(arn); verr != nil {
			return nil, verr
		}
		keys, ok := r.URL.Query()["tagKeys"]
		if !ok {
			return nil, csnValidation("TagKeys is required")
		}
		if len(keys) > csnMaxTags {
			return nil, csnValidation("At most %d tag keys can be removed at once", csnMaxTags)
		}
		return map[string]any{}, st.untag(arn, keys)
	}
	return nil, csnErr(http.StatusNotImplemented, "NotImplementedException", "%s is not implemented", op)
}

func csnValidToken(token string) *csnError {
	if token != "" && (len(token) > 256 || !csnTokenPattern.MatchString(token)) {
		return csnValidation("ClientRequestToken is not valid")
	}
	return nil
}

func csnValidAddress(address string) *csnError {
	if len(address) < 1 || len(address) > 320 {
		return csnValidation("TargetAddress must be 1-320 characters")
	}
	return nil
}

func csnValidEnum(field, value string, allowed ...string) *csnError {
	if !containsString(allowed, value) {
		return csnValidation("%s must be one of %s", field, strings.Join(allowed, ", "))
	}
	return nil
}

func (s *Server) csnCreate(in csnCreateInput) (map[string]any, *csnError) {
	if err := csnValidRuleName(in.Name); err != nil {
		return nil, err
	}
	if err := csnValidEnum("DetailType", in.DetailType, "BASIC", "FULL"); err != nil {
		return nil, err
	}
	if in.Status != "" {
		if err := csnValidEnum("Status", in.Status, "ENABLED", "DISABLED"); err != nil {
			return nil, err
		}
	}
	if err := csnValidToken(in.ClientRequestToken); err != nil {
		return nil, err
	}
	if err := csnValidTags(in.Tags); err != nil {
		return nil, err
	}
	service, err := s.csnCheckResource(in.Resource)
	if err != nil {
		return nil, err
	}
	eventTypes, err := csnEventTypesFor(in.EventTypeIds, service)
	if err != nil {
		return nil, err
	}
	if in.Targets == nil {
		return nil, csnValidation("Targets is required")
	}
	if err := s.csnCheckTargets(in.Targets); err != nil {
		return nil, err
	}
	return s.codestarNotifications.create(in, eventTypes)
}

func (s *Server) csnUpdate(in csnUpdateInput) *csnError {
	if err := csnValidRuleARN(in.Arn); err != nil {
		return err
	}
	if in.Name != nil {
		if err := csnValidRuleName(*in.Name); err != nil {
			return err
		}
	}
	if in.Status != nil {
		if err := csnValidEnum("Status", *in.Status, "ENABLED", "DISABLED"); err != nil {
			return err
		}
	}
	if in.DetailType != nil {
		if err := csnValidEnum("DetailType", *in.DetailType, "BASIC", "FULL"); err != nil {
			return err
		}
	}
	resource, ok := s.codestarNotifications.ruleResource(in.Arn)
	if !ok {
		return csnNotFound("Notification rule %s does not exist", in.Arn)
	}
	var eventTypes []string
	if in.EventTypeIds != nil {
		service, _ := csnResourceService(resource)
		var err *csnError
		if eventTypes, err = csnEventTypesFor(in.EventTypeIds, service); err != nil {
			return err
		}
	}
	if in.Targets != nil {
		if err := s.csnCheckTargets(in.Targets); err != nil {
			return err
		}
	}
	return s.codestarNotifications.update(in, eventTypes)
}

// csnResourceService maps a resource ARN to its event-catalog service name
// and the resource's name.
func csnResourceService(arn string) (service, name string) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) != 6 {
		return "", ""
	}
	switch rest := parts[5]; parts[2] {
	case "codecommit":
		return "CodeCommit", rest
	case "codebuild":
		if n, ok := strings.CutPrefix(rest, "project/"); ok {
			return "CodeBuild", n
		}
	case "codedeploy":
		if n, ok := strings.CutPrefix(rest, "application:"); ok {
			return "CodeDeploy", n
		}
	case "codepipeline":
		if !strings.ContainsAny(rest, ":/") {
			return "CodePipeline", rest
		}
	}
	return "", ""
}

// csnCheckResource validates the rule's resource ARN and, for services
// Stackyard emulates, that the resource exists. It returns the service name.
func (s *Server) csnCheckResource(arn string) (string, *csnError) {
	if !csnResourcePattern.MatchString(arn) {
		return "", csnValidation("Resource %q is not a valid ARN", arn)
	}
	service, name := csnResourceService(arn)
	if service == "" || name == "" {
		return "", csnValidation("Resource %s isn't a CodeCommit repository, CodeBuild project, CodeDeploy application or CodePipeline pipeline", arn)
	}
	exists := true
	switch service {
	case "CodeBuild":
		s.codebuild.mu.Lock()
		_, exists = s.codebuild.projects[name]
		s.codebuild.mu.Unlock()
	case "CodeDeploy":
		s.codedeploy.mu.Lock()
		_, exists = s.codedeploy.applications[name]
		s.codedeploy.mu.Unlock()
	case "CodePipeline":
		s.codepipeline.mu.Lock()
		_, exists = s.codepipeline.pipelines[name]
		s.codepipeline.mu.Unlock()
	}
	// ponytail: CodeCommit isn't emulated yet, so repositories are accepted
	// by ARN; check the codecommit store once it exists.
	if !exists {
		return "", csnValidation("Resource %s does not exist", arn)
	}
	return service, nil
}

// csnCheckTargets validates target types and addresses; SNS topics must exist.
func (s *Server) csnCheckTargets(targets []csnTarget) *csnError {
	if len(targets) > csnMaxTargets {
		return csnValidation("A notification rule can have at most %d targets", csnMaxTargets)
	}
	types := map[string]string{}
	for _, t := range targets {
		if have, ok := types[t.TargetAddress]; ok && have != t.TargetType {
			return csnValidation("Target %s is listed with two types", t.TargetAddress)
		}
		types[t.TargetAddress] = t.TargetType
		if !csnTargetTypePattern.MatchString(t.TargetType) {
			return csnValidation("TargetType is required")
		}
		if err := csnValidAddress(t.TargetAddress); err != nil {
			return err
		}
		switch t.TargetType {
		case "SNS":
			if _, err := s.sns.GetTopic(t.TargetAddress); err != nil {
				return csnValidation("SNS topic %s does not exist", t.TargetAddress)
			}
		case "AWSChatbotSlack", "AWSChatbotMicrosoftTeams":
			m := csnChatbotPattern.FindStringSubmatch(t.TargetAddress)
			want := "slack"
			if t.TargetType == "AWSChatbotMicrosoftTeams" {
				want = "microsoft-teams"
			}
			if m == nil || m[1] != want {
				return csnValidation("%s is not an AWS Chatbot %s channel configuration ARN", t.TargetAddress, want)
			}
		default:
			return csnValidation("TargetType must be SNS, AWSChatbotSlack or AWSChatbotMicrosoftTeams")
		}
	}
	return nil
}

// csnTargetStatus reports SNS targets whose topic is gone as UNREACHABLE.
func (s *Server) csnTargetStatus(targetType, address string) string {
	if targetType == "SNS" {
		if _, err := s.sns.GetTopic(address); err != nil {
			return "UNREACHABLE"
		}
	}
	return "ACTIVE"
}

func csnListEventTypes(filters []csnFilter, token string, max int) (map[string]any, *csnError) {
	var matched []csnEventType
	for _, e := range csnEventTypes {
		keep := true
		for _, f := range filters {
			if f.Name == "RESOURCE_TYPE" {
				keep = keep && e.ResourceType == f.Value
			} else {
				keep = keep && e.ServiceName == f.Value
			}
		}
		if keep {
			matched = append(matched, e)
		}
	}
	page, next, err := csnPage(matched, token, max)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"EventTypes": append([]csnEventType{}, page...)}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}
