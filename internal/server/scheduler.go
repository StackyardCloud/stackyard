package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/services/eventbridge"
	"github.com/stackyard/stackyard/internal/services/sns"
)

const (
	schedulerMaxBody = 1 << 20
	// schedulerFirePath is Stackyard's own endpoint for invoking a schedule's
	// target now, so tests can check delivery without waiting for the clock.
	schedulerFirePath = "/_stackyard/scheduler/"
)

// handleSchedulerRESTRouter serves Amazon EventBridge Scheduler (REST-JSON,
// signed as scheduler).
func (s *Server) handleSchedulerRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isSchedulerRESTCandidate(r, path) {
		return false
	}
	// A declared length bounds what net/http reads; refuse unknown lengths.
	if r.ContentLength < 0 || r.ContentLength > schedulerMaxBody {
		respondSchedulerError(w, schedulerValidation("Request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "scheduler"); !ok {
		respondSchedulerError(w, schedulerErr(status, code, "%s", msg))
		return true
	}
	if strings.HasPrefix(path, schedulerFirePath) {
		s.schedulerFire(w, r, strings.TrimPrefix(path, schedulerFirePath))
		return true
	}
	if !s.routeScheduler(w, r, path) {
		respondSchedulerError(w, schedulerErr(http.StatusNotImplemented, "NotImplementedException", "%s %s is not an EventBridge Scheduler operation", r.Method, path))
	}
	return true
}

func isSchedulerRESTCandidate(r *http.Request, path string) bool {
	if strings.HasPrefix(path, schedulerFirePath) {
		return true
	}
	if path != "/schedules" && path != "/schedule-groups" && !strings.HasPrefix(path, "/schedules/") &&
		!strings.HasPrefix(path, "/schedule-groups/") && !strings.HasPrefix(path, "/tags/") {
		return false
	}
	service := sigV4ServiceHint(r)
	return service == "scheduler" || (service == "" && strings.HasPrefix(strings.ToLower(r.Host), "scheduler."))
}

func (s *Server) routeScheduler(w http.ResponseWriter, r *http.Request, path string) bool {
	q := r.URL.Query()
	resource, rawName, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	name, err := url.PathUnescape(rawName)
	if err != nil {
		respondSchedulerError(w, schedulerValidation("invalid path segment %q", rawName))
		return true
	}
	st := s.scheduler
	switch {
	case resource == "schedules" && rawName == "" && r.Method == http.MethodGet:
		if max, ok := schedulerMaxResults(w, q); ok {
			out, err := st.listSchedules(q.Get("ScheduleGroup"), q.Get("NamePrefix"), q.Get("State"), q.Get("NextToken"), max)
			schedulerRespond(w, out, err)
		}
	case resource == "schedules" && rawName != "":
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			var in schedulerScheduleInput
			if !schedulerDecode(w, r, &in) {
				return true
			}
			var out map[string]any
			var err *schedulerError
			if r.Method == http.MethodPost {
				out, err = st.createSchedule(name, in)
			} else {
				out, err = st.updateSchedule(name, in)
			}
			schedulerRespond(w, out, err)
		case http.MethodGet:
			out, err := st.getSchedule(groupNameOr(schedulerQueryPtr(q, "groupName")), name)
			schedulerRespond(w, out, err)
		case http.MethodDelete:
			schedulerRespond(w, map[string]any{}, st.deleteSchedule(groupNameOr(schedulerQueryPtr(q, "groupName")), name, q.Get("clientToken")))
		default:
			return false
		}
	case resource == "schedule-groups" && rawName == "" && r.Method == http.MethodGet:
		if max, ok := schedulerMaxResults(w, q); ok {
			out, err := st.listGroups(q.Get("NamePrefix"), q.Get("NextToken"), max)
			schedulerRespond(w, out, err)
		}
	case resource == "schedule-groups" && rawName != "":
		switch r.Method {
		case http.MethodPost:
			var in struct {
				Tags        []schedulerTag `json:"Tags"`
				ClientToken *string        `json:"ClientToken"`
			}
			if !schedulerDecode(w, r, &in) {
				return true
			}
			token := ""
			if in.ClientToken != nil {
				token = *in.ClientToken
			}
			out, err := st.createGroup(name, token, in.Tags)
			schedulerRespond(w, out, err)
		case http.MethodGet:
			out, err := st.getGroup(name)
			schedulerRespond(w, out, err)
		case http.MethodDelete:
			schedulerRespond(w, map[string]any{}, st.deleteGroup(name, q.Get("clientToken")))
		default:
			return false
		}
	case resource == "tags" && rawName != "":
		switch r.Method {
		case http.MethodGet:
			out, err := st.listTags(name)
			schedulerRespond(w, out, err)
		case http.MethodPost:
			var in struct {
				Tags []schedulerTag `json:"Tags"`
			}
			if !schedulerDecode(w, r, &in) {
				return true
			}
			if in.Tags == nil {
				respondSchedulerError(w, schedulerValidation("Tags is required"))
				return true
			}
			schedulerRespond(w, map[string]any{}, st.tagResource(name, in.Tags))
		case http.MethodDelete:
			keys, ok := q["TagKeys"]
			if !ok {
				respondSchedulerError(w, schedulerValidation("TagKeys is required"))
				return true
			}
			schedulerRespond(w, map[string]any{}, st.untagResource(name, keys))
		default:
			return false
		}
	default:
		return false
	}
	return true
}

func schedulerQueryPtr(q url.Values, key string) *string {
	if v := q.Get(key); v != "" {
		return &v
	}
	return nil
}

func schedulerMaxResults(w http.ResponseWriter, q url.Values) (int, bool) {
	raw := q.Get("MaxResults")
	if raw == "" {
		return 100, true
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 1 || n > 100 {
		respondSchedulerError(w, schedulerValidation("MaxResults must be an integer from 1 to 100"))
		return 0, false
	}
	return int(n), true
}

// schedulerDecode decodes a JSON body, rejecting members the model doesn't
// define so callers can't set anything else.
func schedulerDecode(w http.ResponseWriter, r *http.Request, v any) bool {
	body, err := readBodyBytes(r)
	if err != nil {
		respondSchedulerError(w, schedulerValidation("unable to read request body"))
		return false
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		respondSchedulerError(w, schedulerValidation("Invalid request body: %v", err))
		return false
	}
	return true
}

func schedulerRespond(w http.ResponseWriter, body map[string]any, err *schedulerError) {
	if err != nil {
		respondSchedulerError(w, err)
		return
	}
	payload, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func respondSchedulerError(w http.ResponseWriter, err *schedulerError) {
	payload, _ := json.Marshal(map[string]string{"Message": err.msg})
	w.Header().Set("X-Amzn-ErrorType", err.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.status)
	_, _ = w.Write(payload)
}

// schedulerFire handles POST /_stackyard/scheduler/schedules/{group}/{name}/fire:
// it delivers the schedule's Input to an SQS, SNS or EventBridge target now.
func (s *Server) schedulerFire(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.Split(rest, "/")
	if r.Method != http.MethodPost || len(parts) != 4 || parts[0] != "schedules" || parts[3] != "fire" {
		respondSchedulerError(w, schedulerErr(http.StatusNotFound, "ResourceNotFoundException", "use POST %sschedules/{group}/{name}/fire", schedulerFirePath))
		return
	}
	group, err1 := url.PathUnescape(parts[1])
	name, err2 := url.PathUnescape(parts[2])
	if err1 != nil || err2 != nil || schedulerValidName("GroupName", group) != nil || schedulerValidName("Name", name) != nil {
		respondSchedulerError(w, schedulerValidation("invalid schedule group or name"))
		return
	}
	fire, ferr := s.scheduler.takeForFire(group, name)
	if ferr != nil {
		respondSchedulerError(w, ferr)
		return
	}
	input := "{}"
	if fire.target.Input != nil {
		input = *fire.target.Input
	}
	out := map[string]any{"TargetArn": fire.arn}
	m := schedulerTemplatedARN.FindStringSubmatch(fire.arn)
	service := ""
	if m != nil {
		service = m[1]
	}
	switch service {
	case "sqs":
		sysAttrs := map[string]string{}
		if p := fire.target.SqsParameters; p != nil && p.MessageGroupId != nil {
			sysAttrs["MessageGroupId"] = *p.MessageGroupId
			sysAttrs["MessageDeduplicationId"] = uuid.NewString()
		}
		msg, err := s.sqs.SendMessageWithDelay(sqsQueueNameFromARN(fire.arn), input, sysAttrs, nil, 0)
		if err != nil {
			respondSchedulerError(w, schedulerNotFound("Target queue %s: %v", fire.arn, err))
			return
		}
		out["MessageId"] = msg.ID
	case "sns":
		id, err := s.sns.Publish(sns.PublishInput{TopicARN: fire.arn, Message: input})
		if err != nil {
			respondSchedulerError(w, schedulerNotFound("Target topic %s: %v", fire.arn, err))
			return
		}
		out["MessageId"] = id
	case "events":
		p := fire.target.EventBridgeParameters
		if p == nil {
			p = fire.target.EventBridgePutEventsV2Parameters
		}
		if p == nil || p.Source == nil || p.DetailType == nil {
			respondSchedulerError(w, schedulerValidation("Firing raw EventBridge targets isn't supported"))
			return
		}
		bus, _ := strings.CutPrefix(m[2], "event-bus/")
		ok, failed := s.eventbridge.PutEvents([]eventbridge.EventEntry{{EventBus: bus, Source: *p.Source, DetailType: *p.DetailType, Detail: input}})
		if len(failed) > 0 || len(ok) == 0 {
			respondSchedulerError(w, schedulerNotFound("Target event bus %s could not take the event", fire.arn))
			return
		}
		out["EventId"] = ok[0].ID
	default:
		respondSchedulerError(w, schedulerValidation("Firing supports SQS, SNS and EventBridge targets, not %s", fire.arn))
		return
	}
	schedulerRespond(w, out, nil)
}
