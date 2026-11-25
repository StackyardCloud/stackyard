package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Personalize Events: REST JSON, signed personalize. Events land in
// the datasets of the event tracker's (or dataset's) dataset group.

var pszevPaths = map[string]string{
	"/events": "PutEvents", "/items": "PutItems", "/users": "PutUsers",
	"/actions": "PutActions", "/action-interactions": "PutActionInteractions",
}

// pszevActionEvents are the event types PutActionInteractions accepts.
var pszevActionEvents = []string{"Taken", "Not taken", "Viewed"}

// pszRESTCandidate reports whether r is a Personalize REST call for host prefix.
func pszRESTCandidate(r *http.Request, hostPrefix string, paths map[string]string) (string, bool) {
	if r.Method != http.MethodPost || r.Header.Get("X-Amz-Target") != "" {
		return "", false
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.TrimSpace(sigV4ServiceHint(r)) != "personalize" && !strings.HasPrefix(host, hostPrefix) {
		return "", false
	}
	op, ok := paths[rawRequestPath(r)]
	return op, ok
}

func (s *Server) handlePersonalizeEventsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	op, ok := pszRESTCandidate(r, "personalize-events.", pszevPaths)
	if !ok {
		return false
	}
	if _, known := personalizeEventsOperationByName[op]; !known {
		return false
	}
	s.pszServeREST(w, r, awsmodels.PersonalizeEventsModel(), op, func(in map[string]any) (map[string]any, *pszErr) {
		return nil, s.personalize.putEvents(op, in)
	})
	return true
}

func respondPSZRESTError(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "message": msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (st *personalizeStore) putEvents(op string, in map[string]any) *pszErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	switch op {
	case "PutEvents", "PutActionInteractions":
		return st.putInteractions(op, in)
	}
	typ, listKey, idKey := pszItems, "items", "itemId"
	switch op {
	case "PutUsers":
		typ, listKey, idKey = pszUsers, "users", "userId"
	case "PutActions":
		typ, listKey, idKey = pszActions, "actions", "actionId"
	}
	ds, err := st.get("dataset", pszStr(in, "datasetArn"))
	if err != nil {
		return err
	}
	if ds.data.typ != typ {
		return pszInvalid("%s needs a %s dataset; %s is %s.", op, typ, ds.arn, ds.data.typ)
	}
	if ds.status != pszStatusActive {
		return pszError(pszErrInUse, "Dataset %s is %s, not ACTIVE.", ds.arn, ds.status)
	}
	type rec struct {
		id    string
		props map[string]string
	}
	var recs []rec
	for _, item := range pszList(in, listKey) {
		m, _ := item.(map[string]any)
		props, err := pszProperties(pszStr(m, "properties"), ds.data.fields, pszIDField[typ])
		if err != nil {
			return err
		}
		recs = append(recs, rec{pszStr(m, idKey), props})
	}
	// Check the cap before writing so a batch is all-or-nothing.
	added := 0
	for _, r := range recs {
		if ds.data.entities[r.id] == nil {
			added++
		}
	}
	if len(ds.data.entities)+added > pszMaxRecords {
		return pszInvalid("Stackyard keeps at most %d records per dataset.", pszMaxRecords)
	}
	for _, r := range recs {
		_ = ds.data.putEntity(r.id, r.props, false)
	}
	return nil
}

func (st *personalizeStore) putInteractions(op string, in map[string]any) *pszErr {
	trackerARN, ok := st.trackers[pszStr(in, "trackingId")]
	if !ok {
		return pszInvalid("trackingId %s is invalid: no event tracker has it.", pszStr(in, "trackingId"))
	}
	typ, listKey := pszInteractions, "eventList"
	if op == "PutActionInteractions" {
		typ, listKey = pszActionInteractions, "actionInteractions"
	}
	ds := st.dataset(st.res[trackerARN].rel["datasetGroupArn"], typ)
	if ds == nil {
		if op == "PutEvents" {
			return pszInvalid("The event tracker's dataset group has no Interactions dataset.")
		}
		return pszError(pszErrNotFound, "The event tracker's dataset group has no Action_Interactions dataset.")
	}
	if ds.status != pszStatusActive {
		return pszError(pszErrInUse, "Dataset %s is %s, not ACTIVE.", ds.arn, ds.status)
	}
	var events []pszEvent
	for _, item := range pszList(in, listKey) {
		m, _ := item.(map[string]any)
		e := pszEvent{eventType: pszStr(m, "eventType")}
		var at time.Time
		if op == "PutEvents" {
			e.user, e.target = pszStr(in, "userId"), pszStr(m, "itemId")
			at, ok = pszTime(m["sentAt"])
			if e.target == "" {
				return pszInvalid("Each event needs an itemId.")
			}
			if e.user == "" {
				e.user = "anonymous:" + pszStr(in, "sessionId")
			}
		} else {
			e.user, e.target = pszStr(m, "userId"), pszStr(m, "actionId")
			at, ok = pszTime(m["timestamp"])
			if !pszContains(pszevActionEvents, e.eventType) {
				return pszInvalid("Action interaction eventType must be Taken, Not taken or Viewed.")
			}
			if e.user == "" {
				e.user = "anonymous:" + pszStr(m, "sessionId")
			}
		}
		if !ok {
			return pszInvalid("Each event needs a valid timestamp.")
		}
		if raw := pszStr(m, "properties"); raw != "" {
			var obj map[string]any
			if json.Unmarshal([]byte(raw), &obj) != nil {
				return pszInvalid("properties must be a JSON object.")
			}
		}
		e.at = at
		events = append(events, e)
	}
	for _, e := range events {
		ds.data.addEvent(e)
	}
	return nil
}
