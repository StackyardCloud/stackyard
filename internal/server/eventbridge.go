package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/services/eventbridge"
)

type eventBridgeError struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

type eventBridgeTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type eventBridgeDeadLetterConfig struct {
	Arn string `json:"Arn,omitempty"`
}

type eventBridgeFailedEntry struct {
	ErrorCode    string `json:"ErrorCode"`
	ErrorMessage string `json:"ErrorMessage"`
	TargetId     string `json:"TargetId"`
}

type eventBridgeTarget struct {
	Id      string `json:"Id"`
	Arn     string `json:"Arn"`
	RoleArn string `json:"RoleArn,omitempty"`
	Input   string `json:"Input,omitempty"`
}

type eventBridgeRuleEntry struct {
	Name               string `json:"Name"`
	Arn                string `json:"Arn"`
	EventBusName       string `json:"EventBusName"`
	EventPattern       string `json:"EventPattern,omitempty"`
	ScheduleExpression string `json:"ScheduleExpression,omitempty"`
	State              string `json:"State"`
	Description        string `json:"Description,omitempty"`
	RoleArn            string `json:"RoleArn,omitempty"`
	ManagedBy          string `json:"ManagedBy,omitempty"`
}

type eventBridgeBusEntry struct {
	Name             string                       `json:"Name"`
	Arn              string                       `json:"Arn"`
	Description      string                       `json:"Description,omitempty"`
	KmsKeyIdentifier string                       `json:"KmsKeyIdentifier,omitempty"`
	DeadLetterConfig *eventBridgeDeadLetterConfig `json:"DeadLetterConfig,omitempty"`
	EventSourceName  string                       `json:"EventSourceName,omitempty"`
	Policy           string                       `json:"Policy,omitempty"`
}

type eventBridgeEventSourceEntry struct {
	Name      string    `json:"Name"`
	Arn       string    `json:"Arn"`
	State     string    `json:"State"`
	CreatedAt time.Time `json:"CreatedAt,omitempty"`
}

type eventBridgePartnerEventSourceEntry struct {
	Name            string    `json:"Name"`
	Arn             string    `json:"Arn"`
	Account         string    `json:"Account"`
	EventSourceName string    `json:"EventSourceName"`
	State           string    `json:"State"`
	CreatedAt       time.Time `json:"CreatedAt,omitempty"`
}

type eventBridgeReplayEntry struct {
	ReplayName string `json:"ReplayName"`
	ReplayArn  string `json:"ReplayArn"`
	State      string `json:"State"`
}

func (s *Server) handleEventBridgeJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isEventBridgeJSONCandidate(r) {
		return false
	}
	ok, status, code, msg, _ := s.validateSigV4WithService(r, "events")
	if !ok {
		respondEventBridgeJSONError(w, status, code, msg)
		return true
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	action := parseEventBridgeTarget(target)
	if action == "" {
		respondEventBridgeJSONError(w, http.StatusBadRequest, "InvalidAction", "missing X-Amz-Target")
		return true
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		respondEventBridgeJSONError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return true
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}

	switch action {
	case "CreateEventBus":
		var input struct {
			Name            string `json:"Name"`
			EventSourceName string `json:"EventSourceName"`
			Description     string `json:"Description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.Name) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "Name is required")
			return true
		}
		bus, err := s.eventbridge.CreateEventBus(input.Name, input.EventSourceName, input.Description)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"EventBusArn": bus.ARN})
		return true
	case "DescribeEventBus":
		var input struct {
			Name string `json:"Name"`
		}
		_ = json.Unmarshal(body, &input)
		bus, err := s.eventbridge.DescribeEventBus(input.Name)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", "event bus not found")
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgeBusEntry{
			Name:             bus.Name,
			Arn:              bus.ARN,
			Description:      bus.Description,
			KmsKeyIdentifier: bus.KmsKeyIdentifier,
			DeadLetterConfig: eventBridgeDeadLetterConfigOrNil(bus.DeadLetterArn),
			EventSourceName:  bus.EventSourceName,
			Policy:           bus.Policy,
		})
		return true
	case "ListEventBuses":
		buses := s.eventbridge.ListEventBuses()
		out := make([]eventBridgeBusEntry, 0, len(buses))
		for _, bus := range buses {
			out = append(out, eventBridgeBusEntry{
				Name:            bus.Name,
				Arn:             bus.ARN,
				EventSourceName: bus.EventSourceName,
				Policy:          bus.Policy,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"EventBuses": out,
		})
		return true
	case "ActivateEventSource":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		src, err := s.eventbridge.ActivateEventSource(input.Name)
		if err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrEventSourceNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgeEventSourceEntry{
			Name:      src.Name,
			Arn:       src.ARN,
			State:     src.State,
			CreatedAt: src.CreatedAt,
		})
		return true
	case "DeactivateEventSource":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		src, err := s.eventbridge.DeactivateEventSource(input.Name)
		if err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrEventSourceNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgeEventSourceEntry{
			Name:      src.Name,
			Arn:       src.ARN,
			State:     src.State,
			CreatedAt: src.CreatedAt,
		})
		return true
	case "DescribeEventSource":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		src, err := s.eventbridge.DescribeEventSource(input.Name)
		if err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrEventSourceNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgeEventSourceEntry{
			Name:      src.Name,
			Arn:       src.ARN,
			State:     src.State,
			CreatedAt: src.CreatedAt,
		})
		return true
	case "ListEventSources":
		var input struct {
			NamePrefix string `json:"NamePrefix"`
		}
		_ = json.Unmarshal(body, &input)
		sources := s.eventbridge.ListEventSources(input.NamePrefix)
		out := make([]eventBridgeEventSourceEntry, 0, len(sources))
		for _, src := range sources {
			out = append(out, eventBridgeEventSourceEntry{
				Name:      src.Name,
				Arn:       src.ARN,
				State:     src.State,
				CreatedAt: src.CreatedAt,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"EventSources": out})
		return true
	case "CreatePartnerEventSource":
		var input struct {
			Name            string `json:"Name"`
			Account         string `json:"Account"`
			EventSourceName string `json:"EventSourceName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		src, err := s.eventbridge.CreatePartnerEventSource(input.Name, input.Account, input.EventSourceName)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"EventSourceArn": src.ARN})
		return true
	case "DeletePartnerEventSource":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.DeletePartnerEventSource(input.Name); err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrPartnerEventSourceNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DescribePartnerEventSource":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		src, err := s.eventbridge.DescribePartnerEventSource(input.Name)
		if err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrPartnerEventSourceNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgePartnerEventSourceEntry{
			Name:            src.Name,
			Arn:             src.ARN,
			Account:         src.Account,
			EventSourceName: src.EventSourceName,
			State:           src.State,
			CreatedAt:       src.CreatedAt,
		})
		return true
	case "ListPartnerEventSourceAccounts":
		var input struct {
			EventSourceName string `json:"EventSourceName"`
		}
		_ = json.Unmarshal(body, &input)
		accounts := s.eventbridge.ListPartnerEventSourceAccounts(input.EventSourceName)
		out := make([]map[string]string, 0, len(accounts))
		for _, account := range accounts {
			out = append(out, map[string]string{"Account": account})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"PartnerEventSourceAccounts": out})
		return true
	case "ListPartnerEventSources":
		var input struct {
			NamePrefix string `json:"NamePrefix"`
		}
		_ = json.Unmarshal(body, &input)
		sources := s.eventbridge.ListPartnerEventSources(input.NamePrefix)
		out := make([]eventBridgePartnerEventSourceEntry, 0, len(sources))
		for _, src := range sources {
			out = append(out, eventBridgePartnerEventSourceEntry{
				Name:            src.Name,
				Arn:             src.ARN,
				Account:         src.Account,
				EventSourceName: src.EventSourceName,
				State:           src.State,
				CreatedAt:       src.CreatedAt,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"PartnerEventSources": out})
		return true
	case "DeleteEventBus":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.Name) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "Name is required")
			return true
		}
		if err := s.eventbridge.DeleteEventBus(input.Name); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", "event bus not found")
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "PutRule":
		var input struct {
			Name               string `json:"Name"`
			EventBusName       string `json:"EventBusName"`
			EventPattern       string `json:"EventPattern"`
			ScheduleExpression string `json:"ScheduleExpression"`
			State              string `json:"State"`
			Description        string `json:"Description"`
			RoleArn            string `json:"RoleArn"`
			ManagedBy          string `json:"ManagedBy"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.Name) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "Name is required")
			return true
		}
		rule, err := s.eventbridge.PutRule(eventbridge.Rule{
			Name:               input.Name,
			EventBusName:       input.EventBusName,
			EventPattern:       input.EventPattern,
			ScheduleExpression: input.ScheduleExpression,
			State:              input.State,
			Description:        input.Description,
			RoleArn:            input.RoleArn,
			ManagedBy:          input.ManagedBy,
		})
		if err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrEventBusNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"RuleArn": rule.ARN})
		return true
	case "DescribeRule":
		var input struct {
			Name         string `json:"Name"`
			EventBusName string `json:"EventBusName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		rule, err := s.eventbridge.DescribeRule(input.EventBusName, input.Name)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", "rule not found")
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgeRuleEntry{
			Name:               rule.Name,
			Arn:                rule.ARN,
			EventBusName:       rule.EventBusName,
			EventPattern:       rule.EventPattern,
			ScheduleExpression: rule.ScheduleExpression,
			State:              rule.State,
			Description:        rule.Description,
			RoleArn:            rule.RoleArn,
			ManagedBy:          rule.ManagedBy,
		})
		return true
	case "ListRules":
		var input struct {
			EventBusName string `json:"EventBusName"`
			NamePrefix   string `json:"NamePrefix"`
		}
		_ = json.Unmarshal(body, &input)
		rules := s.eventbridge.ListRules(input.EventBusName, input.NamePrefix)
		out := make([]eventBridgeRuleEntry, 0, len(rules))
		for _, rule := range rules {
			out = append(out, eventBridgeRuleEntry{
				Name:               rule.Name,
				Arn:                rule.ARN,
				EventBusName:       rule.EventBusName,
				EventPattern:       rule.EventPattern,
				ScheduleExpression: rule.ScheduleExpression,
				State:              rule.State,
				Description:        rule.Description,
				RoleArn:            rule.RoleArn,
				ManagedBy:          rule.ManagedBy,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"Rules": out})
		return true
	case "EnableRule":
		var input struct {
			Name         string `json:"Name"`
			EventBusName string `json:"EventBusName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.SetRuleState(input.EventBusName, input.Name, "ENABLED"); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", "rule not found")
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DisableRule":
		var input struct {
			Name         string `json:"Name"`
			EventBusName string `json:"EventBusName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.SetRuleState(input.EventBusName, input.Name, "DISABLED"); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", "rule not found")
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteRule":
		var input struct {
			Name         string `json:"Name"`
			EventBusName string `json:"EventBusName"`
			Force        bool   `json:"Force"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.DeleteRule(input.EventBusName, input.Name, input.Force); err != nil {
			code := "ResourceNotFoundException"
			if err == eventbridge.ErrRuleHasTargets {
				code = "ValidationException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "PutTargets":
		var input struct {
			EventBusName string              `json:"EventBusName"`
			Rule         string              `json:"Rule"`
			Targets      []eventBridgeTarget `json:"Targets"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		targets := make([]eventbridge.Target, 0, len(input.Targets))
		for _, target := range input.Targets {
			targets = append(targets, eventbridge.Target{
				ID:      target.Id,
				Arn:     target.Arn,
				RoleArn: target.RoleArn,
				Input:   target.Input,
			})
		}
		failed, err := s.eventbridge.PutTargets(input.EventBusName, input.Rule, targets)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		failedEntries := make([]eventBridgeFailedEntry, 0, len(failed))
		for _, target := range failed {
			failedEntries = append(failedEntries, eventBridgeFailedEntry{
				ErrorCode:    "ValidationException",
				ErrorMessage: "Id and Arn are required",
				TargetId:     target.ID,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"FailedEntryCount": len(failedEntries),
			"FailedEntries":    failedEntries,
		})
		return true
	case "RemoveTargets":
		var input struct {
			EventBusName string   `json:"EventBusName"`
			Rule         string   `json:"Rule"`
			Ids          []string `json:"Ids"`
			Force        bool     `json:"Force"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		failed, err := s.eventbridge.RemoveTargets(input.EventBusName, input.Rule, input.Ids, input.Force)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		failedEntries := make([]eventBridgeFailedEntry, 0, len(failed))
		for _, id := range failed {
			failedEntries = append(failedEntries, eventBridgeFailedEntry{
				ErrorCode:    "ResourceNotFoundException",
				ErrorMessage: "target not found",
				TargetId:     id,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"FailedEntryCount": len(failedEntries),
			"FailedEntries":    failedEntries,
		})
		return true
	case "ListTargetsByRule":
		var input struct {
			EventBusName string `json:"EventBusName"`
			Rule         string `json:"Rule"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		targets, err := s.eventbridge.ListTargetsByRule(input.EventBusName, input.Rule)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		out := make([]eventBridgeTarget, 0, len(targets))
		for _, target := range targets {
			out = append(out, eventBridgeTarget{
				Id:      target.ID,
				Arn:     target.Arn,
				RoleArn: target.RoleArn,
				Input:   target.Input,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"Targets": out})
		return true
	case "ListRuleNamesByTarget":
		var input struct {
			EventBusName string `json:"EventBusName"`
			TargetArn    string `json:"TargetArn"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.TargetArn) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "TargetArn is required")
			return true
		}
		names := s.eventbridge.ListRuleNamesByTarget(input.EventBusName, input.TargetArn)
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"RuleNames": names})
		return true
	case "PutEvents":
		var input struct {
			Entries []struct {
				EventBusName string          `json:"EventBusName"`
				Source       string          `json:"Source"`
				DetailType   string          `json:"DetailType"`
				Detail       string          `json:"Detail"`
				Resources    []string        `json:"Resources"`
				Time         json.RawMessage `json:"Time"`
			} `json:"Entries"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		entries := make([]eventbridge.EventEntry, 0, len(input.Entries))
		for _, entry := range input.Entries {
			entries = append(entries, eventbridge.EventEntry{
				EventBus:   entry.EventBusName,
				Source:     entry.Source,
				DetailType: entry.DetailType,
				Detail:     entry.Detail,
				Resources:  entry.Resources,
				Time:       parseEventBridgeTime(entry.Time),
			})
		}
		success, failed := s.eventbridge.PutEvents(entries)
		resultEntries := make([]map[string]string, 0, len(entries))
		for _, entry := range success {
			resultEntries = append(resultEntries, map[string]string{"EventId": entry.ID})
		}
		for _, entry := range failed {
			resultEntries = append(resultEntries, map[string]string{
				"ErrorCode":    "ValidationException",
				"ErrorMessage": entry.Detail,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"FailedEntryCount": len(failed),
			"Entries":          resultEntries,
		})
		return true
	case "PutPartnerEvents":
		var input struct {
			Entries []struct {
				EventBusName string          `json:"EventBusName"`
				Source       string          `json:"Source"`
				DetailType   string          `json:"DetailType"`
				Detail       string          `json:"Detail"`
				Resources    []string        `json:"Resources"`
				Time         json.RawMessage `json:"Time"`
			} `json:"Entries"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		entries := make([]eventbridge.EventEntry, 0, len(input.Entries))
		for _, entry := range input.Entries {
			entries = append(entries, eventbridge.EventEntry{
				EventBus:   entry.EventBusName,
				Source:     entry.Source,
				DetailType: entry.DetailType,
				Detail:     entry.Detail,
				Resources:  entry.Resources,
				Time:       parseEventBridgeTime(entry.Time),
			})
		}
		success, failed := s.eventbridge.PutEvents(entries)
		resultEntries := make([]map[string]string, 0, len(entries))
		for _, entry := range success {
			resultEntries = append(resultEntries, map[string]string{"EventId": entry.ID})
		}
		for _, entry := range failed {
			resultEntries = append(resultEntries, map[string]string{
				"ErrorCode":    "ValidationException",
				"ErrorMessage": entry.Detail,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"FailedEntryCount": len(failed),
			"Entries":          resultEntries,
		})
		return true
	case "PutPermission":
		var input struct {
			Action       string `json:"Action"`
			EventBusName string `json:"EventBusName"`
			Principal    string `json:"Principal"`
			StatementId  string `json:"StatementId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.PutPermission(input.EventBusName, input.StatementId, input.Action, input.Principal); err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrEventBusNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "RemovePermission":
		var input struct {
			EventBusName         string `json:"EventBusName"`
			StatementId          string `json:"StatementId"`
			RemoveAllPermissions bool   `json:"RemoveAllPermissions"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.RemovePermission(input.EventBusName, input.StatementId, input.RemoveAllPermissions); err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrEventBusNotFound || err == eventbridge.ErrPermissionNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "TagResource":
		var input struct {
			ResourceARN string           `json:"ResourceARN"`
			Tags        []eventBridgeTag `json:"Tags"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.ResourceARN) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "ResourceARN is required")
			return true
		}
		tagMap := map[string]string{}
		for _, tag := range input.Tags {
			if tag.Key == "" {
				continue
			}
			tagMap[tag.Key] = tag.Value
		}
		s.eventbridge.TagResource(input.ResourceARN, tagMap)
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UntagResource":
		var input struct {
			ResourceARN string   `json:"ResourceARN"`
			TagKeys     []string `json:"TagKeys"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.ResourceARN) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "ResourceARN is required")
			return true
		}
		s.eventbridge.UntagResource(input.ResourceARN, input.TagKeys)
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListTagsForResource":
		var input struct {
			ResourceARN string `json:"ResourceARN"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.ResourceARN) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "ResourceARN is required")
			return true
		}
		tags := s.eventbridge.ListTags(input.ResourceARN)
		out := make([]eventBridgeTag, 0, len(tags))
		for k, v := range tags {
			out = append(out, eventBridgeTag{Key: k, Value: v})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"Tags": out})
		return true
	case "TestEventPattern":
		var input struct {
			EventPattern string `json:"EventPattern"`
			Event        string `json:"Event"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		result := strings.TrimSpace(input.EventPattern) != "" && strings.TrimSpace(input.Event) != ""
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"Result": result})
		return true
	case "CreateArchive":
		var input struct {
			ArchiveName    string `json:"ArchiveName"`
			EventSourceArn string `json:"EventSourceArn"`
			EventPattern   string `json:"EventPattern"`
			Description    string `json:"Description"`
			RetentionDays  int32  `json:"RetentionDays"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.ArchiveName) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "ArchiveName is required")
			return true
		}
		archive, err := s.eventbridge.CreateArchive(eventbridge.Archive{
			Name:           input.ArchiveName,
			EventSourceArn: input.EventSourceArn,
			EventPattern:   input.EventPattern,
			Description:    input.Description,
			RetentionDays:  input.RetentionDays,
		})
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"ArchiveArn": archive.ARN})
		return true
	case "DescribeArchive":
		var input struct {
			ArchiveName string `json:"ArchiveName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		archive, err := s.eventbridge.DescribeArchive(input.ArchiveName)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"ArchiveName":    archive.Name,
			"ArchiveArn":     archive.ARN,
			"EventSourceArn": archive.EventSourceArn,
			"State":          archive.State,
			"EventPattern":   archive.EventPattern,
			"Description":    archive.Description,
			"RetentionDays":  archive.RetentionDays,
		})
		return true
	case "UpdateArchive":
		var input struct {
			ArchiveName   string `json:"ArchiveName"`
			Description   string `json:"Description"`
			EventPattern  string `json:"EventPattern"`
			RetentionDays int32  `json:"RetentionDays"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		archive, err := s.eventbridge.UpdateArchive(input.ArchiveName, input.Description, input.EventPattern, input.RetentionDays)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"ArchiveArn": archive.ARN})
		return true
	case "DeleteArchive":
		var input struct {
			ArchiveName string `json:"ArchiveName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.DeleteArchive(input.ArchiveName); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListArchives":
		archives := s.eventbridge.ListArchives()
		out := make([]map[string]any, 0, len(archives))
		for _, archive := range archives {
			out = append(out, map[string]any{
				"ArchiveName":    archive.Name,
				"ArchiveArn":     archive.ARN,
				"EventSourceArn": archive.EventSourceArn,
				"State":          archive.State,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"Archives": out})
		return true
	case "StartReplay":
		var input struct {
			ReplayName     string          `json:"ReplayName"`
			EventSourceArn string          `json:"EventSourceArn"`
			EventStartTime json.RawMessage `json:"EventStartTime"`
			EventEndTime   json.RawMessage `json:"EventEndTime"`
			Description    string          `json:"Description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		replay, err := s.eventbridge.StartReplay(input.ReplayName, input.EventSourceArn, input.Description, parseEventBridgeTime(input.EventStartTime), parseEventBridgeTime(input.EventEndTime))
		if err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrReplayNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"ReplayArn": replay.ARN})
		return true
	case "CancelReplay":
		var input struct {
			ReplayName string `json:"ReplayName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		replay, err := s.eventbridge.CancelReplay(input.ReplayName)
		if err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrReplayNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"ReplayArn": replay.ARN})
		return true
	case "DescribeReplay":
		var input struct {
			ReplayName string `json:"ReplayName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		replay, err := s.eventbridge.DescribeReplay(input.ReplayName)
		if err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrReplayNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"ReplayName":     replay.Name,
			"ReplayArn":      replay.ARN,
			"State":          replay.State,
			"EventSourceArn": replay.EventSourceArn,
			"Description":    replay.Description,
			"EventStartTime": replay.EventStartTime,
			"EventEndTime":   replay.EventEndTime,
		})
		return true
	case "ListReplays":
		var input struct {
			NamePrefix string `json:"NamePrefix"`
		}
		_ = json.Unmarshal(body, &input)
		replays := s.eventbridge.ListReplays(input.NamePrefix)
		out := make([]eventBridgeReplayEntry, 0, len(replays))
		for _, replay := range replays {
			out = append(out, eventBridgeReplayEntry{
				ReplayName: replay.Name,
				ReplayArn:  replay.ARN,
				State:      replay.State,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"Replays": out})
		return true
	case "CreateConnection":
		var input struct {
			Name              string `json:"Name"`
			AuthorizationType string `json:"AuthorizationType"`
			Description       string `json:"Description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.Name) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "Name is required")
			return true
		}
		conn, err := s.eventbridge.CreateConnection(eventbridge.Connection{
			Name:              input.Name,
			AuthorizationType: input.AuthorizationType,
			Description:       input.Description,
		})
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"ConnectionArn": conn.ARN})
		return true
	case "DescribeConnection":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		conn, err := s.eventbridge.DescribeConnection(input.Name)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"Name":              conn.Name,
			"ConnectionArn":     conn.ARN,
			"AuthorizationType": conn.AuthorizationType,
			"Description":       conn.Description,
			"State":             conn.State,
		})
		return true
	case "UpdateConnection":
		var input struct {
			Name        string `json:"Name"`
			Description string `json:"Description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		conn, err := s.eventbridge.UpdateConnection(input.Name, input.Description)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"ConnectionArn": conn.ARN})
		return true
	case "DeleteConnection":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		conn, err := s.eventbridge.DescribeConnection(input.Name)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		if err := s.eventbridge.DeleteConnection(input.Name); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgeConnectionMutationPayload(conn))
		return true
	case "DeauthorizeConnection":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.DeauthorizeConnection(input.Name); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		conn, err := s.eventbridge.DescribeConnection(input.Name)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgeConnectionMutationPayload(conn))
		return true
	case "ListConnections":
		conns := s.eventbridge.ListConnections()
		out := make([]map[string]any, 0, len(conns))
		for _, conn := range conns {
			out = append(out, map[string]any{
				"Name":          conn.Name,
				"ConnectionArn": conn.ARN,
				"State":         conn.State,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"Connections": out})
		return true
	case "CreateApiDestination":
		var input struct {
			Name                         string `json:"Name"`
			ConnectionArn                string `json:"ConnectionArn"`
			InvocationEndpoint           string `json:"InvocationEndpoint"`
			HttpMethod                   string `json:"HttpMethod"`
			InvocationRateLimitPerSecond int32  `json:"InvocationRateLimitPerSecond"`
			Description                  string `json:"Description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.Name) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "Name is required")
			return true
		}
		dest, err := s.eventbridge.CreateApiDestination(eventbridge.ApiDestination{
			Name:                         input.Name,
			ConnectionArn:                input.ConnectionArn,
			InvocationEndpoint:           input.InvocationEndpoint,
			HttpMethod:                   input.HttpMethod,
			InvocationRateLimitPerSecond: input.InvocationRateLimitPerSecond,
			Description:                  input.Description,
		})
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"ApiDestinationArn": dest.ARN})
		return true
	case "DescribeApiDestination":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		dest, err := s.eventbridge.DescribeApiDestination(input.Name)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"Name":                         dest.Name,
			"ApiDestinationArn":            dest.ARN,
			"ConnectionArn":                dest.ConnectionArn,
			"InvocationEndpoint":           dest.InvocationEndpoint,
			"HttpMethod":                   dest.HttpMethod,
			"InvocationRateLimitPerSecond": dest.InvocationRateLimitPerSecond,
			"Description":                  dest.Description,
		})
		return true
	case "UpdateApiDestination":
		var input struct {
			Name                         string `json:"Name"`
			InvocationEndpoint           string `json:"InvocationEndpoint"`
			InvocationRateLimitPerSecond int32  `json:"InvocationRateLimitPerSecond"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		dest, err := s.eventbridge.UpdateApiDestination(input.Name, input.InvocationEndpoint, input.InvocationRateLimitPerSecond)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]string{"ApiDestinationArn": dest.ARN})
		return true
	case "DeleteApiDestination":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.DeleteApiDestination(input.Name); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListApiDestinations":
		dests := s.eventbridge.ListApiDestinations()
		out := make([]map[string]any, 0, len(dests))
		for _, dest := range dests {
			out = append(out, map[string]any{
				"Name":              dest.Name,
				"ApiDestinationArn": dest.ARN,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"ApiDestinations": out})
		return true
	case "CreateEndpoint":
		var input struct {
			Name        string `json:"Name"`
			Description string `json:"Description"`
			EventBuses  []struct {
				EventBusArn string `json:"EventBusArn"`
			} `json:"EventBuses"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.Name) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "Name is required")
			return true
		}
		buses := make([]string, 0, len(input.EventBuses))
		for _, bus := range input.EventBuses {
			buses = append(buses, bus.EventBusArn)
		}
		ep, err := s.eventbridge.CreateEndpoint(eventbridge.Endpoint{
			Name:        input.Name,
			Description: input.Description,
			EventBuses:  buses,
		})
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgeEndpointMutationPayload(ep, false))
		return true
	case "DescribeEndpoint":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		ep, err := s.eventbridge.DescribeEndpoint(input.Name)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{
			"Name":        ep.Name,
			"EndpointArn": ep.ARN,
			"Description": ep.Description,
			"State":       ep.State,
		})
		return true
	case "UpdateEndpoint":
		var input struct {
			Name        string `json:"Name"`
			Description string `json:"Description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		ep, err := s.eventbridge.UpdateEndpoint(input.Name, input.Description)
		if err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, eventBridgeEndpointMutationPayload(ep, true))
		return true
	case "DeleteEndpoint":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.eventbridge.DeleteEndpoint(input.Name); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
			return true
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListEndpoints":
		eps := s.eventbridge.ListEndpoints()
		out := make([]map[string]any, 0, len(eps))
		for _, ep := range eps {
			out = append(out, map[string]any{
				"Name":        ep.Name,
				"EndpointArn": ep.ARN,
				"State":       ep.State,
			})
		}
		respondEventBridgeJSON(w, http.StatusOK, map[string]any{"Endpoints": out})
		return true
	case "UpdateEventBus":
		var input struct {
			Name             string `json:"Name"`
			KmsKeyIdentifier string `json:"KmsKeyIdentifier"`
			Description      string `json:"Description"`
			DeadLetterConfig struct {
				Arn string `json:"Arn"`
			} `json:"DeadLetterConfig"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.Name) == "" {
			respondEventBridgeJSONError(w, http.StatusBadRequest, "ValidationException", "Name is required")
			return true
		}
		bus, err := s.eventbridge.UpdateEventBus(input.Name, input.KmsKeyIdentifier, input.Description, input.DeadLetterConfig.Arn)
		if err != nil {
			code := "ValidationException"
			if err == eventbridge.ErrEventBusNotFound {
				code = "ResourceNotFoundException"
			}
			respondEventBridgeJSONError(w, http.StatusBadRequest, code, err.Error())
			return true
		}
		resp := map[string]any{
			"Arn":              bus.ARN,
			"Name":             bus.Name,
			"KmsKeyIdentifier": bus.KmsKeyIdentifier,
			"Description":      bus.Description,
		}
		if strings.TrimSpace(bus.DeadLetterArn) != "" {
			resp["DeadLetterConfig"] = map[string]string{"Arn": bus.DeadLetterArn}
		}
		respondEventBridgeJSON(w, http.StatusOK, resp)
		return true
	default:
		respondEventBridgeJSONError(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
		return true
	}
}

func parseEventBridgeTime(raw json.RawMessage) time.Time {
	if len(raw) == 0 {
		return time.Time{}
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return time.Time{}
	}
	if raw[0] == '"' {
		var ts string
		if err := json.Unmarshal(raw, &ts); err != nil {
			return time.Time{}
		}
		if ts == "" {
			return time.Time{}
		}
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			return t
		}
		return time.Time{}
	}
	var seconds float64
	if err := json.Unmarshal(raw, &seconds); err != nil {
		return time.Time{}
	}
	if seconds <= 0 {
		return time.Time{}
	}
	sec := int64(seconds)
	nsec := int64((seconds - float64(sec)) * float64(time.Second))
	return time.Unix(sec, nsec).UTC()
}

func respondEventBridgeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondEventBridgeJSONError(w http.ResponseWriter, status int, code, msg string) {
	respondEventBridgeJSON(w, status, eventBridgeError{Type: code, Message: msg})
}

func isEventBridgeJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AWSEvents") {
		return true
	}
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "AWSEvents")
	}
	return false
}

func parseEventBridgeTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSEvents.") {
		return strings.TrimPrefix(target, "AWSEvents.")
	}
	if strings.HasPrefix(target, "AWSEvents_2015-10-07.") {
		return strings.TrimPrefix(target, "AWSEvents_2015-10-07.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseEventBridgeLimit(limit string, max int) int {
	limit = strings.TrimSpace(limit)
	if limit == "" {
		return max
	}
	val, err := strconv.Atoi(limit)
	if err != nil || val <= 0 {
		return max
	}
	if val > max {
		return max
	}
	return val
}

func eventBridgeDeadLetterConfigOrNil(arn string) *eventBridgeDeadLetterConfig {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return nil
	}
	return &eventBridgeDeadLetterConfig{Arn: arn}
}

func eventBridgeConnectionMutationPayload(conn eventbridge.Connection) map[string]any {
	lastModified := conn.CreatedAt
	if !conn.LastAuthorizedAt.IsZero() {
		lastModified = conn.LastAuthorizedAt
	}
	if lastModified.IsZero() {
		lastModified = time.Now().UTC()
	}
	lastAuthorized := conn.LastAuthorizedAt
	if lastAuthorized.IsZero() {
		lastAuthorized = lastModified
	}
	return map[string]any{
		"ConnectionArn":      conn.ARN,
		"ConnectionState":    conn.State,
		"CreationTime":       conn.CreatedAt,
		"LastModifiedTime":   lastModified,
		"LastAuthorizedTime": lastAuthorized,
	}
}

func eventBridgeEndpointMutationPayload(ep eventbridge.Endpoint, includeUpdateFields bool) map[string]any {
	eventBuses := make([]map[string]any, 0, len(ep.EventBuses))
	for _, arn := range ep.EventBuses {
		arn = strings.TrimSpace(arn)
		if arn == "" {
			continue
		}
		eventBuses = append(eventBuses, map[string]any{"EventBusArn": arn})
	}
	if len(eventBuses) == 0 {
		eventBuses = append(eventBuses, map[string]any{"EventBusArn": "arn:aws:events:us-east-1:123456789012:event-bus/default"})
	}
	for len(eventBuses) < 2 {
		eventBuses = append(eventBuses, eventBuses[len(eventBuses)-1])
	}
	out := map[string]any{
		"Name": ep.Name,
		"Arn":  ep.ARN,
		"RoutingConfig": map[string]any{
			"FailoverConfig": map[string]any{
				"Primary": map[string]any{
					"HealthCheck": "arn:aws:route53:::healthcheck/stackyard",
				},
				"Secondary": map[string]any{
					"Route": eventbridge.DefaultRegion,
				},
			},
		},
		"ReplicationConfig": map[string]any{
			"State": "DISABLED",
		},
		"EventBuses": eventBuses,
		"RoleArn":    "arn:aws:iam::123456789012:role/stackyard-eventbridge-endpoint",
		"State":      ep.State,
	}
	if includeUpdateFields {
		out["EndpointId"] = ep.Name
		out["EndpointUrl"] = "https://" + ep.Name + ".events.amazonaws.com"
	}
	return out
}
