package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

const cloudtraildataAccountID = "123456789012"

// cloudtraildataRecord is one ingested event as CloudTrail Lake stores it.
type cloudtraildataRecord struct {
	EventID    string
	ChannelArn string
	EventTime  string
	Recipient  string
	EventData  map[string]any
	IngestedAt time.Time
}

// columns flattens the record into the top-level Lake columns for integration events.
func (r cloudtraildataRecord) columns() map[string]any {
	return map[string]any{
		"eventVersion":       "1.0",
		"eventCategory":      "ActivityAuditLog",
		"eventType":          "ActivityLog",
		"eventID":            r.EventID,
		"eventTime":          r.EventTime,
		"awsRegion":          "us-east-1",
		"recipientAccountId": r.Recipient,
		"channelArn":         r.ChannelArn,
		"eventData":          r.EventData,
	}
}

// PutAuditEvents validates every event against the channel, then appends the accepted
// ones to each event data store the channel delivers to.
func (s *cloudTrailStore) PutAuditEvents(channelArn, externalID string, events []cloudtraildataAuditEvent) (cloudtraildataPutResponse, *cloudtraildataFault) {
	s.mu.Lock()
	defer s.mu.Unlock()

	channel, fault := s.resolveAuditChannelLocked(channelArn)
	if fault != nil {
		return cloudtraildataPutResponse{}, fault
	}
	arn, _ := channel["ChannelArn"].(string)
	if fault := s.checkAuditExternalIDLocked(arn, externalID); fault != nil {
		return cloudtraildataPutResponse{}, fault
	}

	resp := cloudtraildataPutResponse{Failed: []cloudtraildataFailure{}, Successful: []cloudtraildataSuccess{}}
	stores := cloudTrailChannelEventStores(channel)
	now := time.Now().UTC()
	for _, event := range events {
		data, code, msg := parseCloudTrailAuditEventData(event)
		if code != "" {
			resp.Failed = append(resp.Failed, cloudtraildataFailure{ErrorCode: code, ErrorMessage: msg, ID: event.ID})
			continue
		}
		record := cloudtraildataRecord{
			EventID:    uuid.NewString(),
			ChannelArn: arn,
			EventTime:  data["eventTime"].(string),
			Recipient:  data["recipientAccountId"].(string),
			EventData:  data,
			IngestedAt: now,
		}
		if s.auditEvents == nil {
			s.auditEvents = map[string][]cloudtraildataRecord{}
		}
		for _, store := range stores {
			s.auditEvents[store] = append(s.auditEvents[store], record)
		}
		resp.Successful = append(resp.Successful, cloudtraildataSuccess{EventID: record.EventID, ID: event.ID})
	}
	return resp, nil
}

// resolveAuditChannelLocked accepts a channel ARN or its ID (the ARN suffix).
func (s *cloudTrailStore) resolveAuditChannelLocked(channelArn string) (map[string]any, *cloudtraildataFault) {
	name := channelArn
	if strings.HasPrefix(channelArn, "arn:") {
		parts := strings.SplitN(channelArn, ":", 6)
		if len(parts) != 6 || parts[2] != "cloudtrail" || !strings.HasPrefix(parts[5], "channel/") || parts[5] == "channel/" {
			return nil, &cloudtraildataFault{"InvalidChannelARN", "the specified channel ARN is not a valid channel ARN"}
		}
		if parts[4] != cloudtraildataAccountID {
			return nil, &cloudtraildataFault{"ChannelInsufficientPermission", "the caller's account ID must be the same as the channel owner's account ID"}
		}
		name = strings.TrimPrefix(parts[5], "channel/")
	}
	channel, ok := s.channels[name]
	if !ok || (strings.HasPrefix(channelArn, "arn:") && channel["ChannelArn"] != channelArn) {
		return nil, &cloudtraildataFault{"ChannelNotFound", "the channel could not be found"}
	}
	return channel, nil
}

// checkAuditExternalIDLocked enforces a cloudtrail:ExternalId StringEquals condition in a
// resource policy statement that covers the channel.
// ponytail: the store keeps one resource policy for all resources, so any matching statement applies.
func (s *cloudTrailStore) checkAuditExternalIDLocked(channelArn, externalID string) *cloudtraildataFault {
	var policy struct {
		Statement []struct {
			Resource  any                       `json:"Resource"`
			Condition map[string]map[string]any `json:"Condition"`
		} `json:"Statement"`
	}
	if err := json.Unmarshal([]byte(s.resourcePolicy), &policy); err != nil {
		return nil
	}
	required := false
	for _, stmt := range policy.Statement {
		if !cloudTrailPolicyCovers(stmt.Resource, channelArn) {
			continue
		}
		for key, value := range stmt.Condition["StringEquals"] {
			if !strings.EqualFold(key, "cloudtrail:ExternalId") {
				continue
			}
			required = true
			for _, allowed := range cloudTrailStringList(value) {
				if allowed == externalID {
					return nil
				}
			}
		}
	}
	if required {
		return &cloudtraildataFault{"ChannelInsufficientPermission", "externalId does not match the channel's resource policy"}
	}
	return nil
}

func cloudTrailPolicyCovers(resource any, arn string) bool {
	for _, value := range cloudTrailStringList(resource) {
		if value == "*" || value == arn {
			return true
		}
	}
	return false
}

func cloudTrailStringList(value any) []string {
	switch typed := value.(type) {
	case string:
		return []string{typed}
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

// cloudTrailChannelEventStores returns the names of the event data stores a channel delivers to.
func cloudTrailChannelEventStores(channel map[string]any) []string {
	destinations, _ := channel["Destinations"].([]any)
	out := make([]string, 0, len(destinations))
	for _, raw := range destinations {
		dest, _ := raw.(map[string]any)
		if t, _ := dest["Type"].(string); t != "EventDataStore" {
			continue
		}
		if loc, _ := dest["Location"].(string); loc != "" {
			out = append(out, cloudTrailNameFromARN(loc))
		}
	}
	return out
}

// parseCloudTrailAuditEventData checks an event against the CloudTrail Lake integration
// schema and returns its parsed eventData, or a per-event error code.
func parseCloudTrailAuditEventData(event cloudtraildataAuditEvent) (map[string]any, string, string) {
	if event.EventData == nil || *event.EventData == "" {
		return nil, "FieldNotFound", "eventData is required"
	}
	raw := *event.EventData
	if event.EventDataChecksum != "" {
		sum := sha256.Sum256([]byte(raw))
		if base64.StdEncoding.EncodeToString(sum[:]) != event.EventDataChecksum {
			return nil, "InvalidChecksum", "eventDataChecksum does not match eventData"
		}
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil || data == nil {
		return nil, "InvalidData", "eventData must be a JSON object"
	}
	for _, field := range []string{"version", "eventSource", "eventName", "eventTime", "UID", "recipientAccountId"} {
		if str, _ := data[field].(string); str == "" {
			return nil, "FieldNotFound", "eventData." + field + " is required"
		}
	}
	identity, _ := data["userIdentity"].(map[string]any)
	for _, field := range []string{"type", "principalId"} {
		if str, _ := identity[field].(string); str == "" {
			return nil, "FieldNotFound", "eventData.userIdentity." + field + " is required"
		}
	}
	if data["recipientAccountId"] != cloudtraildataAccountID {
		return nil, "InvalidRecipient", "recipientAccountId must match the channel owner's account ID"
	}
	if _, err := time.Parse(time.RFC3339, data["eventTime"].(string)); err != nil {
		return nil, "InvalidData", "eventTime must be an RFC 3339 timestamp"
	}
	return data, "", ""
}
