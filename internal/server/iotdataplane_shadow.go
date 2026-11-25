package server

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Device shadows: a classic shadow per thing plus named shadows (?name=).
// Shadows need no registry entry, as in AWS. UpdateThingShadow merges the
// desired and reported sections (null deletes a key or a whole section),
// keeps per-field metadata timestamps, bumps the version and refuses a
// stale "version" with ConflictException. GetThingShadow adds the delta
// (desired fields that differ from reported). Deleting a shadow keeps its
// version number, so a re-created shadow continues counting.

const (
	iotdMaxShadowDoc    = 8 << 10  // state document limit
	iotdMaxShadowDepth  = 6        // levels in desired / reported
	iotdMaxShadows      = 10000    // shadows, deleted ones included
	iotdMaxShadowsBytes = 32 << 20 // stored state + metadata bytes
	iotdMaxClientToken  = 64
)

type iotdShadow struct {
	desired, reported         map[string]any
	metaDesired, metaReported map[string]any
	version                   int64
	deleted                   bool
	updated                   time.Time
	size                      int // marshaled state + metadata bytes
}

type iotdStore struct {
	mu          sync.Mutex
	shadows     map[string]*iotdShadow // thing + "\x00" + shadow name
	shadowBytes int
	retained    map[string]*iotdMessage
	retainedSz  int
	clients     map[string]*iotdClient
}

func newIoTDataPlaneStore() *iotdStore {
	return &iotdStore{shadows: map[string]*iotdShadow{}, retained: map[string]*iotdMessage{}, clients: map[string]*iotdClient{}}
}

func iotdShadowKey(thing, name string) string { return thing + "\x00" + name }

func iotdNoShadow(thing, name string) *iotdErr {
	if name == "" {
		return iotdErrorf("ResourceNotFoundException", "No shadow exists with name: '%s'", thing)
	}
	return iotdErrorf("ResourceNotFoundException", "No shadow exists with name: '%s~%s'", thing, name)
}

func (st *iotdStore) getShadow(thing, name string) (map[string]any, *iotdErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.getShadowLocked(thing, name)
}

func (st *iotdStore) getShadowLocked(thing, name string) (map[string]any, *iotdErr) {
	sh := st.shadows[iotdShadowKey(thing, name)]
	if sh == nil || sh.deleted {
		return nil, iotdNoShadow(thing, name)
	}
	state, meta := map[string]any{}, map[string]any{}
	if sh.desired != nil {
		state["desired"], meta["desired"] = iotdCopy(sh.desired), iotdCopy(sh.metaDesired)
	}
	if sh.reported != nil {
		state["reported"], meta["reported"] = iotdCopy(sh.reported), iotdCopy(sh.metaReported)
	}
	if d := iotdDelta(sh.desired, sh.reported); d != nil {
		state["delta"] = d
	}
	return map[string]any{"state": state, "metadata": meta, "version": sh.version, "timestamp": sh.updated.Unix()}, nil
}

func (st *iotdStore) deleteShadow(thing, name string) (map[string]any, *iotdErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.deleteShadowLocked(thing, name)
}

func (st *iotdStore) deleteShadowLocked(thing, name string) (map[string]any, *iotdErr) {
	sh := st.shadows[iotdShadowKey(thing, name)]
	if sh == nil || sh.deleted {
		return nil, iotdNoShadow(thing, name)
	}
	now := time.Now().UTC()
	st.shadowBytes -= sh.size
	st.shadows[iotdShadowKey(thing, name)] = &iotdShadow{version: sh.version, deleted: true, updated: now}
	return map[string]any{"version": sh.version, "timestamp": now.Unix()}, nil
}

// iotdShadowRequest is an UpdateThingShadow document.
type iotdShadowRequest struct {
	State       map[string]json.RawMessage `json:"state"`
	ClientToken *string                    `json:"clientToken"`
	Version     *int64                     `json:"version"`
}

func (st *iotdStore) updateShadow(thing, name string, body []byte) (map[string]any, *iotdErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.updateShadowLocked(thing, name, body)
}

func (st *iotdStore) updateShadowLocked(thing, name string, body []byte) (map[string]any, *iotdErr) {
	if len(body) > iotdMaxShadowDoc {
		return nil, iotdErrorf("RequestEntityTooLargeException", "The payload exceeds the maximum size allowed (%d bytes)", iotdMaxShadowDoc)
	}
	var req iotdShadowRequest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	dec.UseNumber()
	if err := dec.Decode(&req); err != nil {
		return nil, iotdInvalid("Payload contains invalid json or unknown fields")
	}
	if req.State == nil {
		return nil, iotdInvalid("Missing required node: state")
	}
	if req.ClientToken != nil && len(*req.ClientToken) > iotdMaxClientToken {
		return nil, iotdInvalid("clientToken can be at most %d characters", iotdMaxClientToken)
	}
	sections := map[string]map[string]any{}
	given := map[string]bool{}
	for key, raw := range req.State {
		if key != "desired" && key != "reported" {
			return nil, iotdInvalid("State contains an invalid node: '%s'", key)
		}
		given[key] = true
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		var section map[string]any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if err := d.Decode(&section); err != nil || section == nil {
			return nil, iotdInvalid("state.%s must be an object or null", key)
		}
		if iotdDepth(section) > iotdMaxShadowDepth {
			return nil, iotdInvalid("JSON contains too many levels of nesting; maximum is %d", iotdMaxShadowDepth)
		}
		sections[key] = section
	}
	if len(given) == 0 {
		return nil, iotdInvalid("State node must contain desired or reported")
	}
	key := iotdShadowKey(thing, name)
	old := st.shadows[key]
	live := old != nil && !old.deleted
	if req.Version != nil && (!live || *req.Version != old.version) {
		return nil, iotdErrorf("ConflictException", "Version conflict")
	}
	if old == nil && len(st.shadows) >= iotdMaxShadows && !st.evictTombstoneLocked() {
		return nil, iotdErrorf("ThrottlingException", "at most %d shadows can exist", iotdMaxShadows)
	}
	now := time.Now().UTC()
	next := &iotdShadow{updated: now, version: 1}
	if old != nil {
		next.version = old.version + 1
	}
	if live {
		next.desired, next.reported = old.desired, old.reported
		next.metaDesired, next.metaReported = old.metaDesired, old.metaReported
	}
	ts := json.Number(strconv.FormatInt(now.Unix(), 10))
	respState, respMeta := map[string]any{}, map[string]any{}
	if given["desired"] {
		next.desired, next.metaDesired = iotdMergeSection(next.desired, next.metaDesired, sections["desired"], ts)
		respState["desired"], respMeta["desired"] = iotdOrNull(sections["desired"]), iotdMetaFor(sections["desired"], ts)
	}
	if given["reported"] {
		next.reported, next.metaReported = iotdMergeSection(next.reported, next.metaReported, sections["reported"], ts)
		respState["reported"], respMeta["reported"] = iotdOrNull(sections["reported"]), iotdMetaFor(sections["reported"], ts)
	}
	stateBytes, _ := json.Marshal(map[string]any{"desired": next.desired, "reported": next.reported})
	if len(stateBytes) > iotdMaxShadowDoc {
		return nil, iotdErrorf("RequestEntityTooLargeException", "The shadow state exceeds the maximum size allowed (%d bytes)", iotdMaxShadowDoc)
	}
	if iotdDepth(next.desired) > iotdMaxShadowDepth || iotdDepth(next.reported) > iotdMaxShadowDepth {
		return nil, iotdInvalid("JSON contains too many levels of nesting; maximum is %d", iotdMaxShadowDepth)
	}
	metaBytes, _ := json.Marshal([]any{next.metaDesired, next.metaReported})
	next.size = len(stateBytes) + len(metaBytes)
	prevSize := 0
	if old != nil {
		prevSize = old.size
	}
	if st.shadowBytes-prevSize+next.size > iotdMaxShadowsBytes {
		return nil, iotdErrorf("ThrottlingException", "stored shadows would exceed %d bytes", iotdMaxShadowsBytes)
	}
	st.shadowBytes += next.size - prevSize
	st.shadows[key] = next
	out := map[string]any{"state": respState, "metadata": respMeta, "version": next.version, "timestamp": now.Unix()}
	if req.ClientToken != nil {
		out["clientToken"] = *req.ClientToken
	}
	return out, nil
}

// evictTombstoneLocked forgets one deleted shadow's version to make room.
func (st *iotdStore) evictTombstoneLocked() bool {
	for key, sh := range st.shadows {
		if sh.deleted {
			delete(st.shadows, key)
			return true
		}
	}
	return false
}

func (st *iotdStore) listNamedShadows(thing string, start, size int) map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	prefix := thing + "\x00"
	var names []string
	// ponytail: scans every shadow (at most iotdMaxShadows); index by thing if needed.
	for key, sh := range st.shadows {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix && !sh.deleted {
			names = append(names, key[len(prefix):])
		}
	}
	sort.Strings(names)
	from, to, next := iotdSlice(len(names), start, size)
	results := make([]any, 0, to-from)
	for _, n := range names[from:to] {
		results = append(results, n)
	}
	out := map[string]any{"results": results, "timestamp": time.Now().Unix()}
	if next != "" {
		out["nextToken"] = next
	}
	return out
}

// iotdMergeSection merges patch into a copy of section (nil patch deletes
// the section) and updates a copy of its metadata. Stored maps are never
// modified in place.
func iotdMergeSection(section, meta, patch map[string]any, ts json.Number) (map[string]any, map[string]any) {
	if patch == nil {
		return nil, nil
	}
	return iotdMerge(section, patch), iotdMergeMeta(meta, patch, ts)
}

func iotdMerge(dst, patch map[string]any) map[string]any {
	out := make(map[string]any, len(dst)+len(patch))
	for k, v := range dst {
		out[k] = v
	}
	for k, v := range patch {
		switch pv := v.(type) {
		case nil:
			delete(out, k)
		case map[string]any:
			old, _ := out[k].(map[string]any)
			out[k] = iotdMerge(old, pv)
		default:
			out[k] = iotdCopy(v)
		}
	}
	return out
}

func iotdMergeMeta(meta, patch map[string]any, ts json.Number) map[string]any {
	out := make(map[string]any, len(meta)+len(patch))
	for k, v := range meta {
		out[k] = v
	}
	for k, v := range patch {
		switch pv := v.(type) {
		case nil:
			delete(out, k)
		case map[string]any:
			old, _ := out[k].(map[string]any)
			out[k] = iotdMergeMeta(old, pv, ts)
		default:
			out[k] = iotdLeafMeta(v, ts)
		}
	}
	return out
}

// iotdMetaFor is the metadata echoed for an update's section.
func iotdMetaFor(patch map[string]any, ts json.Number) any {
	if patch == nil {
		return map[string]any{"timestamp": ts}
	}
	out := make(map[string]any, len(patch))
	for k, v := range patch {
		if m, ok := v.(map[string]any); ok {
			out[k] = iotdMetaFor(m, ts)
		} else {
			out[k] = iotdLeafMeta(v, ts)
		}
	}
	return out
}

func iotdLeafMeta(v any, ts json.Number) any {
	if list, ok := v.([]any); ok {
		out := make([]any, len(list))
		for i := range list {
			out[i] = map[string]any{"timestamp": ts}
		}
		return out
	}
	return map[string]any{"timestamp": ts}
}

// iotdDelta returns the desired fields that differ from reported (nil if none).
func iotdDelta(desired, reported map[string]any) map[string]any {
	out := map[string]any{}
	for k, dv := range desired {
		rv, ok := reported[k]
		dm, dIsMap := dv.(map[string]any)
		rm, rIsMap := rv.(map[string]any)
		switch {
		case ok && dIsMap && rIsMap:
			if sub := iotdDelta(dm, rm); sub != nil {
				out[k] = sub
			}
		case !ok || !reflect.DeepEqual(dv, rv):
			out[k] = iotdCopy(dv)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// iotdDepth counts nesting levels of objects and arrays (a flat object is 1).
func iotdDepth(v any) int {
	deepest := 0
	switch t := v.(type) {
	case map[string]any:
		for _, item := range t {
			deepest = max(deepest, iotdDepth(item))
		}
	case []any:
		for _, item := range t {
			deepest = max(deepest, iotdDepth(item))
		}
	default:
		return 0
	}
	return deepest + 1
}

func iotdCopy(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, item := range t {
			out[k] = iotdCopy(item)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = iotdCopy(item)
		}
		return out
	}
	return v
}

func iotdOrNull(m map[string]any) any {
	if m == nil {
		return nil
	}
	return iotdCopy(m)
}
