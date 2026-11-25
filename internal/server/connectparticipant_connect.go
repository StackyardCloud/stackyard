package server

import (
	"net/http"
	"strings"
	"time"
)

// cnpConnectOp serves the Amazon Connect operations that start and manage
// chat contacts (StartChatContact, CreateParticipant, StopContact and
// DescribeContact for chats). handled is false when the generic Connect store
// should answer instead. It takes the Connect lock before the chat store's.
func (s *Server) cnpConnectOp(action string, payload map[string]any, params map[string]string) (out map[string]any, err *cnpErr, handled bool) {
	switch action {
	case "StartChatContact", "CreateParticipant", "StopContact", "DescribeContact":
	default:
		return nil, nil, false
	}
	instanceID := strings.TrimSpace(connectStringAny(payload, []string{"InstanceId"}, params["InstanceId"]))
	contactID := strings.TrimSpace(connectStringAny(payload, []string{"ContactId"}, params["ContactId"]))
	invalid := func(msg string) *cnpErr { return &cnpErr{http.StatusBadRequest, "InvalidParameterException", msg} }

	s.connect.mu.Lock()
	defer s.connect.mu.Unlock()
	known := s.connect.instances[instanceID] != nil
	chats := s.connectParticipant

	switch action {
	case "StartChatContact":
		if instanceID == "" {
			return nil, invalid("InstanceId is required"), true
		}
		if !known {
			return nil, &cnpErr{http.StatusNotFound, "ResourceNotFoundException", "instance " + instanceID + " not found"}, true
		}
		details, _ := payload["ParticipantDetails"].(map[string]any)
		name := strings.TrimSpace(connectString(details, "DisplayName", ""))
		flow := connectString(payload, "ContactFlowId", "")
		if name == "" || flow == "" {
			return nil, invalid("ContactFlowId and ParticipantDetails.DisplayName are required"), true
		}
		if len(name) > 256 {
			return nil, invalid("DisplayName is at most 256 characters"), true
		}
		initial, _ := payload["InitialMessage"].(map[string]any)
		req := cnpChatStart{instanceID: instanceID, flowID: flow, displayName: name}
		if initial != nil {
			req.initialType, req.initialContent = connectString(initial, "ContentType", ""), connectString(initial, "Content", "")
			if !cnpMessageTypes[req.initialType] || req.initialContent == "" || len(req.initialContent) > 16384 {
				return nil, invalid("InitialMessage needs a supported ContentType and 1 to 16384 characters of Content"), true
			}
		}
		contact, participant, token, err := chats.startChat(req)
		if err != nil {
			return nil, err, true
		}
		return map[string]any{"ContactId": contact, "ParticipantId": participant, "ParticipantToken": token}, nil, true

	case "CreateParticipant":
		details, _ := payload["ParticipantDetails"].(map[string]any)
		role := connectString(details, "ParticipantRole", "")
		name := connectString(details, "DisplayName", "")
		if role != "CUSTOM_BOT" && role != "SUPERVISOR" {
			return nil, invalid("ParticipantDetails.ParticipantRole must be CUSTOM_BOT or SUPERVISOR"), true
		}
		if name == "" || len(name) > 256 {
			name = role
		}
		id, token, err := chats.addParticipant(instanceID, contactID, name, role)
		if err != nil {
			return nil, err, true
		}
		return map[string]any{"ParticipantId": id, "ParticipantCredentials": map[string]any{
			"ParticipantToken": token, "Expiry": cnpTime(time.Now().Add(cnpCredentialLifetime)),
		}}, nil, true

	case "StopContact":
		if !chats.endChat(instanceID, contactID) {
			return nil, nil, false
		}
		return map[string]any{}, nil, true
	}

	// DescribeContact
	v, ok := chats.contact(instanceID, contactID, false)
	if !ok {
		return nil, nil, false
	}
	contact := map[string]any{
		"Id": v.contactID, "Arn": cnpContactARN(v.instanceID, v.contactID), "InitialContactId": v.contactID,
		"Channel": "CHAT", "InitiationMethod": "API", "InitiationTimestamp": float64(v.started.UnixMilli()) / 1000,
	}
	if !v.ended.IsZero() {
		contact["DisconnectTimestamp"] = float64(v.ended.UnixMilli()) / 1000
	}
	return map[string]any{"Contact": contact}, nil, true
}
