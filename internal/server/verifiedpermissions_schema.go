package server

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Cedar JSON schemas: {"<namespace>": {"entityTypes": {...}, "actions": {...},
// "commonTypes": {...}}}. PutSchema checks the structure and that every
// referenced entity type and action group is declared. In STRICT mode,
// policies are checked against it: the entity types and actions they name must
// exist, and a scope's principal and resource types must be ones the scoped
// actions apply to.
// ponytail: attribute access and condition types aren't type-checked against
// the schema; add a type checker when STRICT needs to reject those.

const (
	vpMaxSchemaBytes = 100000 // AWS's schema size quota
	vpMaxSchemaDepth = 32
)

var vpCedarName = regexp.MustCompile(`^[_a-zA-Z][_a-zA-Z0-9]*$`)

type vpSchema struct {
	raw         string
	namespaces  []string
	entityTypes map[string]bool                 // fully qualified
	actions     map[cedarEntity]*vpSchemaAction // NS::Action::"id"
}

type vpSchemaAction struct {
	principalTypes, resourceTypes map[string]bool
	memberOf                      []cedarEntity
}

type vpSchemaNamespace struct {
	EntityTypes map[string]struct {
		MemberOfTypes []string        `json:"memberOfTypes"`
		Shape         json.RawMessage `json:"shape"`
		Tags          json.RawMessage `json:"tags"`
		Annotations   json.RawMessage `json:"annotations"`
	} `json:"entityTypes"`
	Actions map[string]struct {
		AppliesTo *struct {
			PrincipalTypes []string        `json:"principalTypes"`
			ResourceTypes  []string        `json:"resourceTypes"`
			Context        json.RawMessage `json:"context"`
		} `json:"appliesTo"`
		MemberOf []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"memberOf"`
		Annotations json.RawMessage `json:"annotations"`
	} `json:"actions"`
	CommonTypes map[string]json.RawMessage `json:"commonTypes"`
	Annotations json.RawMessage            `json:"annotations"`
}

func vpQualify(ns, name string) string {
	if ns == "" || strings.Contains(name, "::") {
		return name
	}
	return ns + "::" + name
}

func vpParseSchema(raw string) (*vpSchema, error) {
	if len(raw) > vpMaxSchemaBytes {
		return nil, fmt.Errorf("schema exceeds the maximum size of %d bytes", vpMaxSchemaBytes)
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	var doc map[string]vpSchemaNamespace
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("schema is not a valid Cedar JSON schema: %v", err)
	}
	s := &vpSchema{raw: raw, entityTypes: map[string]bool{}, actions: map[cedarEntity]*vpSchemaAction{}}
	for ns, def := range doc {
		if ns != "" {
			for _, part := range strings.Split(ns, "::") {
				if !vpCedarName.MatchString(part) {
					return nil, fmt.Errorf("namespace %q is not a valid Cedar name", ns)
				}
			}
			s.namespaces = append(s.namespaces, ns)
		}
		if def.EntityTypes == nil || def.Actions == nil {
			return nil, fmt.Errorf("namespace %q must declare entityTypes and actions", ns)
		}
		for name := range def.EntityTypes {
			if !vpCedarName.MatchString(name) {
				return nil, fmt.Errorf("entity type %q is not a valid Cedar name", name)
			}
			s.entityTypes[vpQualify(ns, name)] = true
		}
	}
	sort.Strings(s.namespaces)
	for ns, def := range doc {
		common := def.CommonTypes
		for name, et := range def.EntityTypes {
			for _, parent := range et.MemberOfTypes {
				if !s.entityTypes[vpQualify(ns, parent)] {
					return nil, fmt.Errorf("entity type %s is a member of undeclared type %s", vpQualify(ns, name), parent)
				}
			}
			if len(et.Shape) > 0 {
				if err := s.checkType(ns, common, et.Shape, 0); err != nil {
					return nil, fmt.Errorf("entity type %s: %v", vpQualify(ns, name), err)
				}
			}
		}
		actionType := vpQualify(ns, "Action")
		for id, a := range def.Actions {
			act := &vpSchemaAction{principalTypes: map[string]bool{}, resourceTypes: map[string]bool{}}
			if a.AppliesTo != nil {
				for _, t := range a.AppliesTo.PrincipalTypes {
					if !s.entityTypes[vpQualify(ns, t)] {
						return nil, fmt.Errorf("action %q applies to undeclared principal type %s", id, t)
					}
					act.principalTypes[vpQualify(ns, t)] = true
				}
				for _, t := range a.AppliesTo.ResourceTypes {
					if !s.entityTypes[vpQualify(ns, t)] {
						return nil, fmt.Errorf("action %q applies to undeclared resource type %s", id, t)
					}
					act.resourceTypes[vpQualify(ns, t)] = true
				}
				if len(a.AppliesTo.Context) > 0 {
					if err := s.checkType(ns, common, a.AppliesTo.Context, 0); err != nil {
						return nil, fmt.Errorf("action %q context: %v", id, err)
					}
				}
			}
			for _, m := range a.MemberOf {
				typ := actionType
				if m.Type != "" {
					typ = vpQualify(ns, m.Type)
				}
				act.memberOf = append(act.memberOf, cedarEntity{Type: typ, ID: m.ID})
			}
			s.actions[cedarEntity{Type: actionType, ID: id}] = act
		}
	}
	for e, a := range s.actions {
		for _, group := range a.memberOf {
			if s.actions[group] == nil {
				return nil, fmt.Errorf("action %s is a member of undeclared action %s", e, group)
			}
		}
	}
	return s, nil
}

// checkType validates a Cedar JSON type expression.
func (s *vpSchema) checkType(ns string, common map[string]json.RawMessage, raw json.RawMessage, depth int) error {
	if depth > vpMaxSchemaDepth {
		return fmt.Errorf("types nest more than %d levels", vpMaxSchemaDepth)
	}
	var t struct {
		Type       string                     `json:"type"`
		Name       string                     `json:"name"`
		Element    json.RawMessage            `json:"element"`
		Attributes map[string]json.RawMessage `json:"attributes"`
	}
	if err := json.Unmarshal(raw, &t); err != nil {
		return fmt.Errorf("invalid type: %v", err)
	}
	switch t.Type {
	case "String", "Long", "Boolean":
		return nil
	case "Set":
		if len(t.Element) == 0 {
			return fmt.Errorf("a Set type needs an element type")
		}
		return s.checkType(ns, common, t.Element, depth+1)
	case "Record":
		for name, attr := range t.Attributes {
			if err := s.checkType(ns, common, attr, depth+1); err != nil {
				return fmt.Errorf("attribute %q: %v", name, err)
			}
		}
		return nil
	case "Entity":
		if !s.entityTypes[vpQualify(ns, t.Name)] {
			return fmt.Errorf("undeclared entity type %s", t.Name)
		}
		return nil
	case "Extension":
		switch t.Name {
		case "ipaddr", "decimal", "datetime", "duration":
			return nil
		}
		return fmt.Errorf("unknown extension type %s", t.Name)
	case "EntityOrCommon":
		if _, ok := common[t.Name]; ok || s.entityTypes[vpQualify(ns, t.Name)] {
			return nil
		}
		switch t.Name {
		case "String", "Long", "Boolean":
			return nil
		}
		return fmt.Errorf("undeclared type %s", t.Name)
	case "":
		return fmt.Errorf("type is missing")
	}
	if _, ok := common[t.Type]; ok {
		return nil
	}
	return fmt.Errorf("unknown type %s", t.Type)
}

// validatePolicy checks a parsed policy (or template) against the schema.
func (s *vpSchema) validatePolicy(p *cedarPolicy) error {
	for _, sc := range []cedarScope{p.Principal, p.Resource} {
		if sc.Entity != nil && !s.entityTypes[sc.Entity.Type] {
			return fmt.Errorf("unrecognized entity type `%s`", sc.Entity.Type)
		}
		if sc.IsType != "" && !s.entityTypes[sc.IsType] {
			return fmt.Errorf("unrecognized entity type `%s`", sc.IsType)
		}
	}
	for _, a := range p.Action.Entities {
		if s.actions[a] == nil {
			return fmt.Errorf("unrecognized action `%s`", a)
		}
	}
	var bad error
	cedarWalk(p, func(v any) {
		if e, ok := v.(cedarEntity); ok && bad == nil && !s.entityTypes[e.Type] && s.actions[e] == nil {
			bad = fmt.Errorf("unrecognized entity type or action `%s`", e)
		}
	})
	if bad != nil {
		return bad
	}
	// The scope's principal and resource types must be ones some scoped action
	// applies to (the actions a group covers count too).
	if len(p.Action.Entities) == 0 {
		return nil
	}
	actions := s.expandActions(p.Action.Entities)
	if t := vpScopeType(p.Principal); t != "" && !s.anyApplies(actions, t, true) {
		return fmt.Errorf("unable to find an applicable action given the policy scope constraints: principal type `%s`", t)
	}
	if t := vpScopeType(p.Resource); t != "" && !s.anyApplies(actions, t, false) {
		return fmt.Errorf("unable to find an applicable action given the policy scope constraints: resource type `%s`", t)
	}
	return nil
}

func vpScopeType(sc cedarScope) string {
	if sc.IsType != "" {
		return sc.IsType
	}
	if sc.Op == "==" && sc.Entity != nil {
		return sc.Entity.Type
	}
	return "" // `in` covers descendants of any type
}

// expandActions returns the given actions plus every action that is a member
// (directly or through groups) of one of them.
func (s *vpSchema) expandActions(roots []cedarEntity) []*vpSchemaAction {
	want := map[cedarEntity]bool{}
	for _, r := range roots {
		want[r] = true
	}
	var out []*vpSchemaAction
	for e, a := range s.actions {
		if want[e] || s.actionIn(e, want, map[cedarEntity]bool{}) {
			out = append(out, a)
		}
	}
	return out
}

func (s *vpSchema) actionIn(e cedarEntity, want, seen map[cedarEntity]bool) bool {
	if seen[e] {
		return false
	}
	seen[e] = true
	a := s.actions[e]
	if a == nil {
		return false
	}
	for _, g := range a.memberOf {
		if want[g] || s.actionIn(g, want, seen) {
			return true
		}
	}
	return false
}

func (s *vpSchema) anyApplies(actions []*vpSchemaAction, typ string, principal bool) bool {
	for _, a := range actions {
		set := a.resourceTypes
		if principal {
			set = a.principalTypes
		}
		if set[typ] {
			return true
		}
	}
	return false
}

// cedarWalk calls fn for every literal value in a policy's conditions.
func cedarWalk(p *cedarPolicy, fn func(any)) {
	var walk func(cedarExpr)
	walk = func(x cedarExpr) {
		switch n := x.(type) {
		case cedarLit:
			fn(n.V)
		case cedarUnary:
			walk(n.X)
		case cedarBinary:
			walk(n.L)
			walk(n.R)
		case cedarAttr:
			walk(n.X)
		case cedarHas:
			walk(n.X)
		case cedarLike:
			walk(n.X)
		case cedarIs:
			walk(n.X)
			if n.In != nil {
				walk(n.In)
			}
		case cedarIf:
			walk(n.C)
			walk(n.T)
			walk(n.E)
		case cedarSetLit:
			for _, e := range n.Elems {
				walk(e)
			}
		case cedarRecLit:
			for _, v := range n.Vals {
				walk(v)
			}
		case cedarCall:
			walk(n.Recv)
			for _, a := range n.Args {
				walk(a)
			}
		}
	}
	for _, c := range p.Conditions {
		walk(c.Expr)
	}
}
