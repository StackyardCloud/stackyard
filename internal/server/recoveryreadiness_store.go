package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	recoveryReadinessDefaultCellName            = "stackyard-cell"
	recoveryReadinessDefaultRecoveryGroupName   = "stackyard-recovery-group"
	recoveryReadinessDefaultResourceSetName     = "stackyard-resource-set"
	recoveryReadinessDefaultReadinessCheckName  = "stackyard-readiness-check"
	recoveryReadinessDefaultResourceIdentifier  = "stackyard-resource"
	recoveryReadinessDefaultCrossAuthorization  = "123456789012"
	recoveryReadinessDefaultHostedZoneARN       = "arn:aws:route53:::hostedzone/Z000000000STACKYARD"
	recoveryReadinessDefaultRuleID              = "AWSARC.CommonRecoveryChecks"
	recoveryReadinessDefaultRuleDescription     = "Validate that recovery resources are available and healthy"
	recoveryReadinessDefaultRecommendationText  = "Distribute endpoints across independent cells"
	recoveryReadinessDefaultReadiness           = "READY"
	recoveryReadinessDefaultResourceSetType     = "AWS::Route53RecoveryReadiness::DNSTargetResource"
	recoveryReadinessDefaultRecordSetIdentifier = "stackyard-recordset"
	recoveryReadinessDefaultResourceComponentID = "primary"
)

type recoveryReadinessStore struct {
	mu sync.Mutex

	cells                      map[string]map[string]any
	crossAccountAuthorizations map[string]map[string]any
	readinessChecks            map[string]map[string]any
	recoveryGroups             map[string]map[string]any
	resourceSets               map[string]map[string]any
	rules                      []map[string]any
	tags                       map[string]map[string]string
}

func newRecoveryReadinessStore() *recoveryReadinessStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &recoveryReadinessStore{
		cells:                      map[string]map[string]any{},
		crossAccountAuthorizations: map[string]map[string]any{},
		readinessChecks:            map[string]map[string]any{},
		recoveryGroups:             map[string]map[string]any{},
		resourceSets:               map[string]map[string]any{},
		rules: []map[string]any{
			{
				"RuleId":      recoveryReadinessDefaultRuleID,
				"Description": recoveryReadinessDefaultRuleDescription,
			},
		},
		tags: map[string]map[string]string{},
	}

	cell := s.ensureCellLocked(recoveryReadinessDefaultCellName, now)
	resourceSet := s.ensureResourceSetLocked(recoveryReadinessDefaultResourceSetName, now)
	check := s.ensureReadinessCheckLocked(recoveryReadinessDefaultReadinessCheckName, now)
	recoveryGroup := s.ensureRecoveryGroupLocked(recoveryReadinessDefaultRecoveryGroupName, now)
	crossAccountAuthorization := s.ensureCrossAccountAuthorizationLocked(recoveryReadinessDefaultCrossAuthorization, now)

	check["ResourceSetName"] = recoveryReadinessDefaultResourceSetName
	check["ResourceSet"] = recoveryReadinessDefaultResourceSetName
	recoveryGroup["Cells"] = []any{recoveryReadinessDefaultCellName}
	resourceSet["Resources"] = []any{
		map[string]any{
			"ComponentId": recoveryReadinessDefaultResourceComponentID,
			"DnsTargetResource": map[string]any{
				"DomainName":    "stackyard.example.com",
				"HostedZoneArn": recoveryReadinessDefaultHostedZoneARN,
				"RecordSetId":   recoveryReadinessDefaultRecordSetIdentifier,
			},
		},
	}

	s.ensureTagMapLocked(rrCellARN(rrMapString(cell, "CellName", recoveryReadinessDefaultCellName)))["seed"] = "true"
	s.ensureTagMapLocked(rrResourceSetARN(rrMapString(resourceSet, "ResourceSetName", recoveryReadinessDefaultResourceSetName)))["seed"] = "true"
	s.ensureTagMapLocked(rrReadinessCheckARN(rrMapString(check, "ReadinessCheckName", recoveryReadinessDefaultReadinessCheckName)))["seed"] = "true"
	s.ensureTagMapLocked(rrRecoveryGroupARN(rrMapString(recoveryGroup, "RecoveryGroupName", recoveryReadinessDefaultRecoveryGroupName)))["seed"] = "true"
	s.ensureTagMapLocked(rrCrossAccountAuthorizationARN(rrMapString(crossAccountAuthorization, "CrossAccountAuthorization", recoveryReadinessDefaultCrossAuthorization)))["seed"] = "true"

	return s
}

func (s *recoveryReadinessStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncPayloadWithQuery(payload, query)

	now := time.Now().UTC().Format(time.RFC3339)

	s.ensureCellLocked(recoveryReadinessDefaultCellName, now)
	s.ensureRecoveryGroupLocked(recoveryReadinessDefaultRecoveryGroupName, now)
	s.ensureResourceSetLocked(recoveryReadinessDefaultResourceSetName, now)
	s.ensureReadinessCheckLocked(recoveryReadinessDefaultReadinessCheckName, now)
	s.ensureCrossAccountAuthorizationLocked(recoveryReadinessDefaultCrossAuthorization, now)

	switch action {
	case "CreateCell":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "CellName", ""),
			rrPayloadString(payload, "CellName", ""),
			recoveryReadinessDefaultCellName,
		)
		cell := s.ensureCellLocked(name, now)
		if cells, ok := rrLookupCI(payload, "Cells"); ok {
			cell["Cells"] = rrAnySlice(cells)
		}
		if parents, ok := rrLookupCI(payload, "ParentReadinessScopes"); ok {
			cell["ParentReadinessScopes"] = rrAnySlice(parents)
		}
		s.mergeTagsIntoResourceLocked(rrCellARN(name), payload)
		return rrCloneMap(cell)

	case "CreateCrossAccountAuthorization":
		authorization := rrFirstNonEmpty(
			rrPayloadString(payload, "CrossAccountAuthorization", ""),
			recoveryReadinessDefaultCrossAuthorization,
		)
		created := s.ensureCrossAccountAuthorizationLocked(authorization, now)
		s.mergeTagsIntoResourceLocked(rrCrossAccountAuthorizationARN(authorization), payload)
		return rrCloneMap(created)

	case "CreateReadinessCheck":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "ReadinessCheckName", ""),
			rrPayloadString(payload, "ReadinessCheckName", ""),
			recoveryReadinessDefaultReadinessCheckName,
		)
		check := s.ensureReadinessCheckLocked(name, now)
		resourceSetName := rrFirstNonEmpty(
			rrPayloadString(payload, "ResourceSetName", ""),
			rrPayloadString(payload, "ResourceSet", ""),
			rrMapString(check, "ResourceSetName", recoveryReadinessDefaultResourceSetName),
		)
		check["ResourceSetName"] = resourceSetName
		check["ResourceSet"] = resourceSetName
		s.mergeTagsIntoResourceLocked(rrReadinessCheckARN(name), payload)
		return rrCloneMap(check)

	case "CreateRecoveryGroup":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "RecoveryGroupName", ""),
			rrPayloadString(payload, "RecoveryGroupName", ""),
			recoveryReadinessDefaultRecoveryGroupName,
		)
		group := s.ensureRecoveryGroupLocked(name, now)
		if cells, ok := rrLookupCI(payload, "Cells"); ok {
			group["Cells"] = rrAnySlice(cells)
		}
		s.mergeTagsIntoResourceLocked(rrRecoveryGroupARN(name), payload)
		return rrCloneMap(group)

	case "CreateResourceSet":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "ResourceSetName", ""),
			rrPayloadString(payload, "ResourceSetName", ""),
			recoveryReadinessDefaultResourceSetName,
		)
		set := s.ensureResourceSetLocked(name, now)
		set["ResourceSetType"] = rrFirstNonEmpty(rrPayloadString(payload, "ResourceSetType", ""), rrMapString(set, "ResourceSetType", recoveryReadinessDefaultResourceSetType))
		if resources, ok := rrLookupCI(payload, "Resources"); ok {
			set["Resources"] = rrAnySlice(resources)
		}
		s.mergeTagsIntoResourceLocked(rrResourceSetARN(name), payload)
		return rrCloneMap(set)

	case "DeleteCell":
		name := rrFirstNonEmpty(rrMapString(pathParams, "CellName", ""), rrPayloadString(payload, "CellName", ""), recoveryReadinessDefaultCellName)
		delete(s.cells, name)
		delete(s.tags, rrCellARN(name))
		return map[string]any{}

	case "DeleteCrossAccountAuthorization":
		authorization := rrFirstNonEmpty(
			rrMapString(pathParams, "CrossAccountAuthorization", ""),
			rrPayloadString(payload, "CrossAccountAuthorization", ""),
			recoveryReadinessDefaultCrossAuthorization,
		)
		delete(s.crossAccountAuthorizations, authorization)
		delete(s.tags, rrCrossAccountAuthorizationARN(authorization))
		return map[string]any{}

	case "DeleteReadinessCheck":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "ReadinessCheckName", ""),
			rrPayloadString(payload, "ReadinessCheckName", ""),
			recoveryReadinessDefaultReadinessCheckName,
		)
		delete(s.readinessChecks, name)
		delete(s.tags, rrReadinessCheckARN(name))
		return map[string]any{}

	case "DeleteRecoveryGroup":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "RecoveryGroupName", ""),
			rrPayloadString(payload, "RecoveryGroupName", ""),
			recoveryReadinessDefaultRecoveryGroupName,
		)
		delete(s.recoveryGroups, name)
		delete(s.tags, rrRecoveryGroupARN(name))
		return map[string]any{}

	case "DeleteResourceSet":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "ResourceSetName", ""),
			rrPayloadString(payload, "ResourceSetName", ""),
			recoveryReadinessDefaultResourceSetName,
		)
		delete(s.resourceSets, name)
		delete(s.tags, rrResourceSetARN(name))
		return map[string]any{}

	case "GetArchitectureRecommendations":
		groupName := rrFirstNonEmpty(
			rrMapString(pathParams, "RecoveryGroupName", ""),
			rrPayloadString(payload, "RecoveryGroupName", ""),
			recoveryReadinessDefaultRecoveryGroupName,
		)
		group := s.ensureRecoveryGroupLocked(groupName, now)
		return map[string]any{
			"RecoveryGroupName":  groupName,
			"RecoveryGroupArn":   rrRecoveryGroupARN(groupName),
			"LastAuditTimestamp": now,
			"Recommendations": []any{
				map[string]any{
					"RecommendationText": recoveryReadinessDefaultRecommendationText,
					"Readiness":          recoveryReadinessDefaultReadiness,
					"RecoveryGroupName":  groupName,
					"Cells":              rrCloneAny(group["Cells"]),
				},
			},
		}

	case "GetCell":
		name := rrFirstNonEmpty(rrMapString(pathParams, "CellName", ""), rrPayloadString(payload, "CellName", ""), recoveryReadinessDefaultCellName)
		return rrCloneMap(s.ensureCellLocked(name, now))

	case "GetCellReadinessSummary":
		cellName := rrFirstNonEmpty(rrMapString(pathParams, "CellName", ""), rrPayloadString(payload, "CellName", ""), recoveryReadinessDefaultCellName)
		s.ensureCellLocked(cellName, now)
		checks := make([]any, 0, len(s.readinessChecks))
		for _, check := range s.listReadinessChecksLocked() {
			checks = append(checks, map[string]any{
				"ReadinessCheckName": rrMapString(check, "ReadinessCheckName", recoveryReadinessDefaultReadinessCheckName),
				"Readiness":          recoveryReadinessDefaultReadiness,
			})
		}
		return map[string]any{
			"CellName":        cellName,
			"Readiness":       recoveryReadinessDefaultReadiness,
			"ReadinessChecks": checks,
			"NextToken":       "",
		}

	case "GetReadinessCheck":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "ReadinessCheckName", ""),
			rrPayloadString(payload, "ReadinessCheckName", ""),
			recoveryReadinessDefaultReadinessCheckName,
		)
		return rrCloneMap(s.ensureReadinessCheckLocked(name, now))

	case "GetReadinessCheckResourceStatus":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "ReadinessCheckName", ""),
			rrPayloadString(payload, "ReadinessCheckName", ""),
			recoveryReadinessDefaultReadinessCheckName,
		)
		resourceIdentifier := rrFirstNonEmpty(
			rrMapString(pathParams, "ResourceIdentifier", ""),
			rrPayloadString(payload, "ResourceIdentifier", ""),
			recoveryReadinessDefaultResourceIdentifier,
		)
		check := s.ensureReadinessCheckLocked(name, now)
		return map[string]any{
			"Readiness":          recoveryReadinessDefaultReadiness,
			"ReadinessCheckName": name,
			"ResourceIdentifier": resourceIdentifier,
			"Messages":           []any{},
			"Rules":              rrCloneRuleSlice(s.rules),
			"ResourceResult": map[string]any{
				"ResourceArn":        rrResourceSetARN(rrMapString(check, "ResourceSetName", recoveryReadinessDefaultResourceSetName)),
				"Readiness":          recoveryReadinessDefaultReadiness,
				"ComponentId":        recoveryReadinessDefaultResourceComponentID,
				"ResourceIdentifier": resourceIdentifier,
			},
		}

	case "GetReadinessCheckStatus":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "ReadinessCheckName", ""),
			rrPayloadString(payload, "ReadinessCheckName", ""),
			recoveryReadinessDefaultReadinessCheckName,
		)
		check := s.ensureReadinessCheckLocked(name, now)
		resourceSetName := rrMapString(check, "ResourceSetName", recoveryReadinessDefaultResourceSetName)
		return map[string]any{
			"Readiness":          recoveryReadinessDefaultReadiness,
			"ReadinessCheckName": name,
			"Messages":           []any{},
			"Resources": []any{
				map[string]any{
					"ComponentId": recoveryReadinessDefaultResourceComponentID,
					"Readiness":   recoveryReadinessDefaultReadiness,
					"ResourceArn": rrResourceSetARN(resourceSetName),
				},
			},
			"NextToken": "",
		}

	case "GetRecoveryGroup":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "RecoveryGroupName", ""),
			rrPayloadString(payload, "RecoveryGroupName", ""),
			recoveryReadinessDefaultRecoveryGroupName,
		)
		return rrCloneMap(s.ensureRecoveryGroupLocked(name, now))

	case "GetRecoveryGroupReadinessSummary":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "RecoveryGroupName", ""),
			rrPayloadString(payload, "RecoveryGroupName", ""),
			recoveryReadinessDefaultRecoveryGroupName,
		)
		s.ensureRecoveryGroupLocked(name, now)
		summaries := make([]any, 0, len(s.readinessChecks))
		for _, check := range s.listReadinessChecksLocked() {
			summaries = append(summaries, map[string]any{
				"ReadinessCheckName": rrMapString(check, "ReadinessCheckName", recoveryReadinessDefaultReadinessCheckName),
				"Readiness":          recoveryReadinessDefaultReadiness,
			})
		}
		return map[string]any{
			"RecoveryGroupName": name,
			"Readiness":         recoveryReadinessDefaultReadiness,
			"ReadinessChecks":   summaries,
			"NextToken":         "",
		}

	case "GetResourceSet":
		name := rrFirstNonEmpty(
			rrMapString(pathParams, "ResourceSetName", ""),
			rrPayloadString(payload, "ResourceSetName", ""),
			recoveryReadinessDefaultResourceSetName,
		)
		return rrCloneMap(s.ensureResourceSetLocked(name, now))

	case "ListCells":
		items := make([]any, 0, len(s.cells))
		for _, cell := range s.listCellsLocked() {
			items = append(items, rrCloneMap(cell))
		}
		return map[string]any{"Cells": items, "NextToken": ""}

	case "ListCrossAccountAuthorizations":
		items := make([]any, 0, len(s.crossAccountAuthorizations))
		for _, item := range s.listCrossAccountAuthorizationsLocked() {
			items = append(items, rrCloneMap(item))
		}
		return map[string]any{"CrossAccountAuthorizations": items, "NextToken": ""}

	case "ListReadinessChecks":
		items := make([]any, 0, len(s.readinessChecks))
		for _, check := range s.listReadinessChecksLocked() {
			items = append(items, rrCloneMap(check))
		}
		return map[string]any{"ReadinessChecks": items, "NextToken": ""}

	case "ListRecoveryGroups":
		items := make([]any, 0, len(s.recoveryGroups))
		for _, group := range s.listRecoveryGroupsLocked() {
			items = append(items, rrCloneMap(group))
		}
		return map[string]any{"RecoveryGroups": items, "NextToken": ""}

	case "ListResourceSets":
		items := make([]any, 0, len(s.resourceSets))
		for _, set := range s.listResourceSetsLocked() {
			items = append(items, rrCloneMap(set))
		}
		return map[string]any{"ResourceSets": items, "NextToken": ""}

	case "ListRules":
		return map[string]any{"Rules": rrCloneRuleSlice(s.rules), "NextToken": ""}

	case "ListTagsForResources":
		resourceArn := rrFirstNonEmpty(
			rrMapString(pathParams, "ResourceArn", ""),
			rrPayloadString(payload, "ResourceArn", ""),
			rrRecoveryReadinessDefaultTagARN(),
		)
		return map[string]any{"Tags": rrCloneStringMap(s.ensureTagMapLocked(resourceArn))}

	case "TagResource":
		resourceArn := rrFirstNonEmpty(
			rrMapString(pathParams, "ResourceArn", ""),
			rrPayloadString(payload, "ResourceArn", ""),
			rrRecoveryReadinessDefaultTagARN(),
		)
		s.mergeTagsIntoResourceLocked(resourceArn, payload)
		return map[string]any{}

	case "UntagResource":
		resourceArn := rrFirstNonEmpty(
			rrMapString(pathParams, "ResourceArn", ""),
			rrPayloadString(payload, "ResourceArn", ""),
			rrRecoveryReadinessDefaultTagARN(),
		)
		tagKeys := rrStringSlice(rrLookupAny(payload, "TagKeys"))
		if len(tagKeys) == 0 {
			tagKeys = rrStringSlice(rrLookupAny(payload, "tagKeys"))
		}
		tags := s.ensureTagMapLocked(resourceArn)
		for _, key := range tagKeys {
			delete(tags, key)
		}
		return map[string]any{}

	case "UpdateCell":
		name := rrFirstNonEmpty(rrMapString(pathParams, "CellName", ""), rrPayloadString(payload, "CellName", ""), recoveryReadinessDefaultCellName)
		cell := s.ensureCellLocked(name, now)
		if cells, ok := rrLookupCI(payload, "Cells"); ok {
			cell["Cells"] = rrAnySlice(cells)
		}
		if parents, ok := rrLookupCI(payload, "ParentReadinessScopes"); ok {
			cell["ParentReadinessScopes"] = rrAnySlice(parents)
		}
		s.mergeTagsIntoResourceLocked(rrCellARN(name), payload)
		cell["LastUpdatedTime"] = now
		return rrCloneMap(cell)

	case "UpdateReadinessCheck":
		name := rrFirstNonEmpty(rrMapString(pathParams, "ReadinessCheckName", ""), rrPayloadString(payload, "ReadinessCheckName", ""), recoveryReadinessDefaultReadinessCheckName)
		check := s.ensureReadinessCheckLocked(name, now)
		resourceSetName := rrFirstNonEmpty(rrPayloadString(payload, "ResourceSetName", ""), rrPayloadString(payload, "ResourceSet", ""), rrMapString(check, "ResourceSetName", recoveryReadinessDefaultResourceSetName))
		check["ResourceSetName"] = resourceSetName
		check["ResourceSet"] = resourceSetName
		s.mergeTagsIntoResourceLocked(rrReadinessCheckARN(name), payload)
		check["LastUpdatedTime"] = now
		return rrCloneMap(check)

	case "UpdateRecoveryGroup":
		name := rrFirstNonEmpty(rrMapString(pathParams, "RecoveryGroupName", ""), rrPayloadString(payload, "RecoveryGroupName", ""), recoveryReadinessDefaultRecoveryGroupName)
		group := s.ensureRecoveryGroupLocked(name, now)
		if cells, ok := rrLookupCI(payload, "Cells"); ok {
			group["Cells"] = rrAnySlice(cells)
		}
		s.mergeTagsIntoResourceLocked(rrRecoveryGroupARN(name), payload)
		group["LastUpdatedTime"] = now
		return rrCloneMap(group)

	case "UpdateResourceSet":
		name := rrFirstNonEmpty(rrMapString(pathParams, "ResourceSetName", ""), rrPayloadString(payload, "ResourceSetName", ""), recoveryReadinessDefaultResourceSetName)
		set := s.ensureResourceSetLocked(name, now)
		if resources, ok := rrLookupCI(payload, "Resources"); ok {
			set["Resources"] = rrAnySlice(resources)
		}
		set["ResourceSetType"] = rrFirstNonEmpty(rrPayloadString(payload, "ResourceSetType", ""), rrMapString(set, "ResourceSetType", recoveryReadinessDefaultResourceSetType))
		s.mergeTagsIntoResourceLocked(rrResourceSetARN(name), payload)
		set["LastUpdatedTime"] = now
		return rrCloneMap(set)
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"Items": []any{}, "NextToken": ""}
	}
	if strings.HasPrefix(action, "Get") {
		return map[string]any{"Readiness": recoveryReadinessDefaultReadiness}
	}
	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") {
		return map[string]any{"Status": "ACTIVE"}
	}
	if strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Tag") || strings.HasPrefix(action, "Untag") {
		return map[string]any{}
	}
	return map[string]any{}
}

func (s *recoveryReadinessStore) syncPayloadWithQuery(payload map[string]any, query url.Values) {
	if payload == nil {
		return
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			payload[key] = values[0]
			continue
		}
		items := make([]any, 0, len(values))
		for _, value := range values {
			items = append(items, value)
		}
		payload[key] = items
	}
}

func (s *recoveryReadinessStore) ensureCellLocked(name, now string) map[string]any {
	name = rrNormalizeName(name, recoveryReadinessDefaultCellName)
	if cell := s.cells[name]; cell != nil {
		return cell
	}
	cell := map[string]any{
		"CellArn":               rrCellARN(name),
		"CellName":              name,
		"Cells":                 []any{},
		"ParentReadinessScopes": []any{},
		"Tags":                  map[string]any{},
		"CreatedTime":           now,
	}
	s.cells[name] = cell
	return cell
}

func (s *recoveryReadinessStore) ensureCrossAccountAuthorizationLocked(authorization, now string) map[string]any {
	authorization = rrNormalizeName(authorization, recoveryReadinessDefaultCrossAuthorization)
	if item := s.crossAccountAuthorizations[authorization]; item != nil {
		return item
	}
	item := map[string]any{
		"CrossAccountAuthorization":    authorization,
		"CrossAccountAuthorizationArn": rrCrossAccountAuthorizationARN(authorization),
		"CreatedTime":                  now,
	}
	s.crossAccountAuthorizations[authorization] = item
	return item
}

func (s *recoveryReadinessStore) ensureReadinessCheckLocked(name, now string) map[string]any {
	name = rrNormalizeName(name, recoveryReadinessDefaultReadinessCheckName)
	if check := s.readinessChecks[name]; check != nil {
		return check
	}
	check := map[string]any{
		"ReadinessCheckArn":  rrReadinessCheckARN(name),
		"ReadinessCheckName": name,
		"ResourceSet":        recoveryReadinessDefaultResourceSetName,
		"ResourceSetName":    recoveryReadinessDefaultResourceSetName,
		"Tags":               map[string]any{},
		"CreatedTime":        now,
	}
	s.readinessChecks[name] = check
	return check
}

func (s *recoveryReadinessStore) ensureRecoveryGroupLocked(name, now string) map[string]any {
	name = rrNormalizeName(name, recoveryReadinessDefaultRecoveryGroupName)
	if group := s.recoveryGroups[name]; group != nil {
		return group
	}
	group := map[string]any{
		"RecoveryGroupArn":  rrRecoveryGroupARN(name),
		"RecoveryGroupName": name,
		"Cells":             []any{recoveryReadinessDefaultCellName},
		"Tags":              map[string]any{},
		"CreatedTime":       now,
	}
	s.recoveryGroups[name] = group
	return group
}

func (s *recoveryReadinessStore) ensureResourceSetLocked(name, now string) map[string]any {
	name = rrNormalizeName(name, recoveryReadinessDefaultResourceSetName)
	if set := s.resourceSets[name]; set != nil {
		return set
	}
	set := map[string]any{
		"ResourceSetArn":  rrResourceSetARN(name),
		"ResourceSetName": name,
		"ResourceSetType": recoveryReadinessDefaultResourceSetType,
		"Resources":       []any{},
		"Tags":            map[string]any{},
		"CreatedTime":     now,
		"LastUpdatedTime": now,
	}
	s.resourceSets[name] = set
	return set
}

func (s *recoveryReadinessStore) ensureTagMapLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = rrRecoveryReadinessDefaultTagARN()
	}
	if s.tags[resourceArn] == nil {
		s.tags[resourceArn] = map[string]string{}
	}
	return s.tags[resourceArn]
}

func (s *recoveryReadinessStore) mergeTagsIntoResourceLocked(resourceArn string, payload map[string]any) {
	tags := s.ensureTagMapLocked(resourceArn)
	for key, value := range rrStringMap(rrLookupAny(payload, "Tags")) {
		tags[key] = value
	}
	for key, value := range rrStringMap(rrLookupAny(payload, "tags")) {
		tags[key] = value
	}
}

func (s *recoveryReadinessStore) listCellsLocked() []map[string]any {
	keys := make([]string, 0, len(s.cells))
	for key := range s.cells {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.cells[key])
	}
	return out
}

func (s *recoveryReadinessStore) listCrossAccountAuthorizationsLocked() []map[string]any {
	keys := make([]string, 0, len(s.crossAccountAuthorizations))
	for key := range s.crossAccountAuthorizations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.crossAccountAuthorizations[key])
	}
	return out
}

func (s *recoveryReadinessStore) listReadinessChecksLocked() []map[string]any {
	keys := make([]string, 0, len(s.readinessChecks))
	for key := range s.readinessChecks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.readinessChecks[key])
	}
	return out
}

func (s *recoveryReadinessStore) listRecoveryGroupsLocked() []map[string]any {
	keys := make([]string, 0, len(s.recoveryGroups))
	for key := range s.recoveryGroups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.recoveryGroups[key])
	}
	return out
}

func (s *recoveryReadinessStore) listResourceSetsLocked() []map[string]any {
	keys := make([]string, 0, len(s.resourceSets))
	for key := range s.resourceSets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.resourceSets[key])
	}
	return out
}

func rrCellARN(name string) string {
	return fmt.Sprintf("arn:aws:route53-recovery-readiness:us-east-1:123456789012:cell/%s", rrNormalizeName(name, recoveryReadinessDefaultCellName))
}

func rrCrossAccountAuthorizationARN(authorization string) string {
	return fmt.Sprintf("arn:aws:route53-recovery-readiness:us-east-1:123456789012:crossaccountauthorization/%s", rrNormalizeName(authorization, recoveryReadinessDefaultCrossAuthorization))
}

func rrReadinessCheckARN(name string) string {
	return fmt.Sprintf("arn:aws:route53-recovery-readiness:us-east-1:123456789012:readinesscheck/%s", rrNormalizeName(name, recoveryReadinessDefaultReadinessCheckName))
}

func rrRecoveryGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:route53-recovery-readiness:us-east-1:123456789012:recoverygroup/%s", rrNormalizeName(name, recoveryReadinessDefaultRecoveryGroupName))
}

func rrResourceSetARN(name string) string {
	return fmt.Sprintf("arn:aws:route53-recovery-readiness:us-east-1:123456789012:resourceset/%s", rrNormalizeName(name, recoveryReadinessDefaultResourceSetName))
}

func rrRecoveryReadinessDefaultTagARN() string {
	return rrRecoveryGroupARN(recoveryReadinessDefaultRecoveryGroupName)
}

func rrNormalizeName(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func rrMapString(m any, key, fallback string) string {
	switch typed := m.(type) {
	case map[string]string:
		for currentKey, value := range typed {
			if !strings.EqualFold(currentKey, key) {
				continue
			}
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
			break
		}
	case map[string]any:
		for currentKey, value := range typed {
			if !strings.EqualFold(currentKey, key) {
				continue
			}
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" && text != "<nil>" {
				return text
			}
			break
		}
	}
	return fallback
}

func rrPayloadString(payload map[string]any, key, fallback string) string {
	if value := rrLookupAny(payload, key); value != nil {
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return fallback
}

func rrLookupAny(payload map[string]any, key string) any {
	value, _ := rrLookupCI(payload, key)
	return value
}

func rrLookupCI(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for currentKey, value := range payload {
		if strings.EqualFold(currentKey, key) {
			return value, true
		}
	}
	return nil, false
}

func rrAnySlice(value any) []any {
	if value == nil {
		return []any{}
	}
	if list, ok := value.([]any); ok {
		return rrCloneSlice(list)
	}
	if list, ok := value.([]string); ok {
		out := make([]any, 0, len(list))
		for _, item := range list {
			out = append(out, item)
		}
		return out
	}
	return []any{value}
}

func rrStringSlice(value any) []string {
	if value == nil {
		return nil
	}
	if list, ok := value.([]string); ok {
		out := make([]string, 0, len(list))
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	}
	if list, ok := value.([]any); ok {
		out := make([]string, 0, len(list))
		for _, item := range list {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text != "" && text != "<nil>" {
				out = append(out, text)
			}
		}
		return out
	}
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "" || text == "<nil>" {
		return nil
	}
	return []string{text}
}

func rrStringMap(value any) map[string]string {
	out := map[string]string{}
	if value == nil {
		return out
	}
	if m, ok := value.(map[string]string); ok {
		for key, val := range m {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(val)
		}
		return out
	}
	if m, ok := value.(map[string]any); ok {
		for key, val := range m {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", val))
		}
	}
	return out
}

func rrFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func rrCloneRuleSlice(items []map[string]any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, rrCloneMap(item))
	}
	return out
}

func rrCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func rrCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = rrCloneAny(value)
	}
	return out
}

func rrCloneSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, value := range in {
		out = append(out, rrCloneAny(value))
	}
	return out
}

func rrCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return rrCloneMap(typed)
	case []any:
		return rrCloneSlice(typed)
	case map[string]string:
		return rrCloneStringMap(typed)
	default:
		return value
	}
}
