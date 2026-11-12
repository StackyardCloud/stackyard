package server

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/services/sns"
)

func (s *Server) handleSNSQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSNSQueryCandidate(r) {
		return false
	}
	ok, status, code, msg, _ := s.validateSigV4WithService(r, "sns")
	if !ok {
		respondSNSErrorXML(w, status, code, msg)
		return true
	}
	if err := r.ParseForm(); err != nil {
		respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "invalid form")
		return true
	}
	action := r.Form.Get("Action")
	if action == "" {
		respondSNSErrorXML(w, http.StatusBadRequest, "MissingAction", "missing Action")
		return true
	}

	switch action {
	case "CreateTopic":
		name := strings.TrimSpace(r.Form.Get("Name"))
		if name == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "Name is required")
			return true
		}
		attrs := parseSNSAttributes(r.Form, "Attributes")
		topic, err := s.sns.CreateTopic(name, attrs)
		if err != nil {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", err.Error())
			return true
		}
		respondSNSXML(w, action, snsCreateTopicResult{TopicArn: topic.ARN})
		return true
	case "DeleteTopic":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		if topicArn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn is required")
			return true
		}
		if err := s.sns.DeleteTopic(topicArn); err != nil {
			if errors.Is(err, sns.ErrTopicNotFound) {
				respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "topic not found")
				return true
			}
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", err.Error())
			return true
		}
		respondSNSXML(w, action, snsDeleteTopicResult{})
		return true
	case "ListTopics":
		topics := s.sns.ListTopics()
		items := make([]snsTopicEntry, 0, len(topics))
		for _, topic := range topics {
			items = append(items, snsTopicEntry{TopicArn: topic.ARN})
		}
		page, nextToken := paginateSNS(items, r.Form.Get("NextToken"), r.Form.Get("MaxResults"))
		respondSNSXML(w, action, snsListTopicsResult{Topics: page, NextToken: nextToken})
		return true
	case "GetTopicAttributes":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		if topicArn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn is required")
			return true
		}
		topic, err := s.sns.GetTopic(topicArn)
		if err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "topic not found")
			return true
		}
		attrs := buildSNSAttributes(topic.Attributes)
		if topic.DataProtectionPolicy != "" {
			attrs = append(attrs, snsAttributeEntry{Key: "DataProtectionPolicy", Value: topic.DataProtectionPolicy})
		}
		respondSNSXML(w, action, snsGetTopicAttributesResult{Attributes: attrs})
		return true
	case "SetTopicAttributes":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		attrName := strings.TrimSpace(r.Form.Get("AttributeName"))
		attrValue := r.Form.Get("AttributeValue")
		if topicArn == "" || attrName == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn and AttributeName are required")
			return true
		}
		if err := s.sns.SetTopicAttributes(topicArn, map[string]string{attrName: attrValue}); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "topic not found")
			return true
		}
		respondSNSXML(w, action, snsSetTopicAttributesResult{})
		return true
	case "PutDataProtectionPolicy":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		if topicArn == "" {
			topicArn = strings.TrimSpace(r.Form.Get("ResourceArn"))
		}
		policy := strings.TrimSpace(r.Form.Get("DataProtectionPolicy"))
		if topicArn == "" || policy == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn or ResourceArn and DataProtectionPolicy are required")
			return true
		}
		if err := s.sns.PutDataProtectionPolicy(topicArn, policy); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "topic not found")
			return true
		}
		respondSNSXML(w, action, snsPutDataProtectionPolicyResult{})
		return true
	case "GetDataProtectionPolicy":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		if topicArn == "" {
			topicArn = strings.TrimSpace(r.Form.Get("ResourceArn"))
		}
		if topicArn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn or ResourceArn is required")
			return true
		}
		policy, err := s.sns.GetDataProtectionPolicy(topicArn)
		if err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "topic not found")
			return true
		}
		respondSNSXML(w, action, snsGetDataProtectionPolicyResult{DataProtectionPolicy: policy})
		return true
	case "Subscribe":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		protocol := strings.TrimSpace(r.Form.Get("Protocol"))
		endpoint := strings.TrimSpace(r.Form.Get("Endpoint"))
		if topicArn == "" || protocol == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn and Protocol are required")
			return true
		}
		returnArn := strings.TrimSpace(r.Form.Get("ReturnSubscriptionArn")) == "true"
		requireConfirm := !returnArn && snsProtocolRequiresConfirmation(protocol)
		sub, err := s.sns.Subscribe(topicArn, protocol, endpoint, parseSNSAttributes(r.Form, "Attributes"), requireConfirm)
		if err != nil {
			if errors.Is(err, sns.ErrTopicNotFound) {
				respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "topic not found")
				return true
			}
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", err.Error())
			return true
		}
		subArn := sub.ARN
		if sub.Status == "PendingConfirmation" && !returnArn {
			subArn = "pending confirmation"
		}
		respondSNSXML(w, action, snsSubscribeResult{SubscriptionArn: subArn})
		return true
	case "ConfirmSubscription":
		token := strings.TrimSpace(r.Form.Get("Token"))
		if token == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "Token is required")
			return true
		}
		sub, err := s.sns.ConfirmSubscription(token)
		if err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "subscription not found")
			return true
		}
		respondSNSXML(w, action, snsConfirmSubscriptionResult{SubscriptionArn: sub.ARN})
		return true
	case "GetSubscriptionAttributes":
		subArn := strings.TrimSpace(r.Form.Get("SubscriptionArn"))
		if subArn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "SubscriptionArn is required")
			return true
		}
		sub, err := s.sns.GetSubscription(subArn)
		if err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "subscription not found")
			return true
		}
		respondSNSXML(w, action, snsGetSubscriptionAttributesResult{Attributes: buildSNSAttributes(sub.Attributes)})
		return true
	case "SetSubscriptionAttributes":
		subArn := strings.TrimSpace(r.Form.Get("SubscriptionArn"))
		attrName := strings.TrimSpace(r.Form.Get("AttributeName"))
		attrValue := r.Form.Get("AttributeValue")
		if subArn == "" || attrName == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "SubscriptionArn and AttributeName are required")
			return true
		}
		if err := s.sns.SetSubscriptionAttributes(subArn, map[string]string{attrName: attrValue}); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "subscription not found")
			return true
		}
		respondSNSXML(w, action, snsSetSubscriptionAttributesResult{})
		return true
	case "ListSubscriptions":
		subs := s.sns.ListSubscriptions("")
		items := buildSNSSubscriptions(subs)
		page, nextToken := paginateSNS(items, r.Form.Get("NextToken"), r.Form.Get("MaxResults"))
		respondSNSXML(w, action, snsListSubscriptionsResult{Subscriptions: page, NextToken: nextToken})
		return true
	case "ListSubscriptionsByTopic":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		if topicArn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn is required")
			return true
		}
		subs := s.sns.ListSubscriptions(topicArn)
		items := buildSNSSubscriptions(subs)
		page, nextToken := paginateSNS(items, r.Form.Get("NextToken"), r.Form.Get("MaxResults"))
		respondSNSXML(w, action, snsListSubscriptionsByTopicResult{Subscriptions: page, NextToken: nextToken})
		return true
	case "Unsubscribe":
		subArn := strings.TrimSpace(r.Form.Get("SubscriptionArn"))
		if subArn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "SubscriptionArn is required")
			return true
		}
		if err := s.sns.Unsubscribe(subArn); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "subscription not found")
			return true
		}
		respondSNSXML(w, action, snsUnsubscribeResult{})
		return true
	case "Publish":
		input := sns.PublishInput{
			TopicARN:    strings.TrimSpace(r.Form.Get("TopicArn")),
			TargetARN:   strings.TrimSpace(r.Form.Get("TargetArn")),
			PhoneNumber: strings.TrimSpace(r.Form.Get("PhoneNumber")),
			Message:     r.Form.Get("Message"),
			Subject:     r.Form.Get("Subject"),
		}
		if input.Message == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "Message is required")
			return true
		}
		if input.TopicARN == "" && input.TargetARN == "" && input.PhoneNumber == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn, TargetArn, or PhoneNumber is required")
			return true
		}
		id, err := s.sns.Publish(input)
		if err != nil {
			if errors.Is(err, sns.ErrTopicNotFound) {
				respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "topic not found")
				return true
			}
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", err.Error())
			return true
		}
		respondSNSXML(w, action, snsPublishResult{MessageID: id})
		return true
	case "PublishBatch":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		if topicArn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn is required")
			return true
		}
		entries := parseSNSPublishBatchEntries(r.Form)
		if len(entries) == 0 {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "entries are required")
			return true
		}
		if len(entries) > 10 {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "too many entries")
			return true
		}
		result := s.sns.PublishBatch(topicArn, entries)
		respondSNSXML(w, action, snsPublishBatchResult{
			Successful: buildSNSPublishBatchSuccess(result.Successful),
			Failed:     buildSNSPublishBatchFailure(result.Failed),
		})
		return true
	case "AddPermission":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		label := strings.TrimSpace(r.Form.Get("Label"))
		if topicArn == "" || label == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn and Label are required")
			return true
		}
		if err := s.sns.SetTopicAttributes(topicArn, map[string]string{"Policy": fmt.Sprintf("permission:%s", label)}); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "topic not found")
			return true
		}
		respondSNSXML(w, action, snsAddPermissionResult{})
		return true
	case "RemovePermission":
		topicArn := strings.TrimSpace(r.Form.Get("TopicArn"))
		label := strings.TrimSpace(r.Form.Get("Label"))
		if topicArn == "" || label == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TopicArn and Label are required")
			return true
		}
		if err := s.sns.SetTopicAttributes(topicArn, map[string]string{"Policy": ""}); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "topic not found")
			return true
		}
		respondSNSXML(w, action, snsRemovePermissionResult{})
		return true
	case "TagResource":
		arn := strings.TrimSpace(r.Form.Get("ResourceArn"))
		if arn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "ResourceArn is required")
			return true
		}
		tags := parseSNSTags(r.Form)
		s.sns.TagResource(arn, tags)
		respondSNSXML(w, action, snsTagResourceResult{})
		return true
	case "UntagResource":
		arn := strings.TrimSpace(r.Form.Get("ResourceArn"))
		if arn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "ResourceArn is required")
			return true
		}
		keys := parseSNSTagKeys(r.Form)
		if len(keys) == 0 {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "TagKeys are required")
			return true
		}
		s.sns.UntagResource(arn, keys)
		respondSNSXML(w, action, snsUntagResourceResult{})
		return true
	case "ListTagsForResource":
		arn := strings.TrimSpace(r.Form.Get("ResourceArn"))
		if arn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "ResourceArn is required")
			return true
		}
		tags := buildSNSTags(s.sns.ListTags(arn))
		respondSNSXML(w, action, snsListTagsForResourceResult{Tags: tags})
		return true
	case "CreatePlatformApplication":
		name := strings.TrimSpace(r.Form.Get("Name"))
		platform := strings.TrimSpace(r.Form.Get("Platform"))
		if name == "" || platform == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "Name and Platform are required")
			return true
		}
		attrs := parseSNSAttributes(r.Form, "Attributes")
		app, err := s.sns.CreatePlatformApplication(name, platform, attrs)
		if err != nil {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", err.Error())
			return true
		}
		respondSNSXML(w, action, snsCreatePlatformApplicationResult{PlatformApplicationArn: app.ARN})
		return true
	case "DeletePlatformApplication":
		arn := strings.TrimSpace(r.Form.Get("PlatformApplicationArn"))
		if arn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PlatformApplicationArn is required")
			return true
		}
		if err := s.sns.DeletePlatformApplication(arn); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "platform application not found")
			return true
		}
		respondSNSXML(w, action, snsDeletePlatformApplicationResult{})
		return true
	case "ListPlatformApplications":
		apps := s.sns.ListPlatformApplications()
		items := make([]snsPlatformApplicationEntry, 0, len(apps))
		for _, app := range apps {
			items = append(items, snsPlatformApplicationEntry{PlatformApplicationArn: app.ARN})
		}
		page, nextToken := paginateSNS(items, r.Form.Get("NextToken"), r.Form.Get("MaxResults"))
		respondSNSXML(w, action, snsListPlatformApplicationsResult{PlatformApplications: page, NextToken: nextToken})
		return true
	case "GetPlatformApplicationAttributes":
		arn := strings.TrimSpace(r.Form.Get("PlatformApplicationArn"))
		if arn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PlatformApplicationArn is required")
			return true
		}
		app, err := s.sns.GetPlatformApplication(arn)
		if err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "platform application not found")
			return true
		}
		respondSNSXML(w, action, snsGetPlatformApplicationAttributesResult{Attributes: buildSNSAttributes(app.Attributes)})
		return true
	case "SetPlatformApplicationAttributes":
		arn := strings.TrimSpace(r.Form.Get("PlatformApplicationArn"))
		if arn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PlatformApplicationArn is required")
			return true
		}
		attrs := parseSNSAttributes(r.Form, "Attributes")
		if err := s.sns.SetPlatformApplicationAttributes(arn, attrs); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "platform application not found")
			return true
		}
		respondSNSXML(w, action, snsSetPlatformApplicationAttributesResult{})
		return true
	case "CreatePlatformEndpoint":
		appArn := strings.TrimSpace(r.Form.Get("PlatformApplicationArn"))
		token := strings.TrimSpace(r.Form.Get("Token"))
		if appArn == "" || token == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PlatformApplicationArn and Token are required")
			return true
		}
		custom := strings.TrimSpace(r.Form.Get("CustomUserData"))
		endpoint, err := s.sns.CreatePlatformEndpoint(appArn, token, custom, parseSNSAttributes(r.Form, "Attributes"))
		if err != nil {
			if errors.Is(err, sns.ErrPlatformNotFound) {
				respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "platform application not found")
				return true
			}
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", err.Error())
			return true
		}
		respondSNSXML(w, action, snsCreatePlatformEndpointResult{EndpointArn: endpoint.ARN})
		return true
	case "DeleteEndpoint":
		arn := strings.TrimSpace(r.Form.Get("EndpointArn"))
		if arn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "EndpointArn is required")
			return true
		}
		if err := s.sns.DeleteEndpoint(arn); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "endpoint not found")
			return true
		}
		respondSNSXML(w, action, snsDeleteEndpointResult{})
		return true
	case "ListEndpointsByPlatformApplication":
		appArn := strings.TrimSpace(r.Form.Get("PlatformApplicationArn"))
		if appArn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PlatformApplicationArn is required")
			return true
		}
		endpoints := s.sns.ListEndpoints(appArn)
		items := make([]snsEndpointEntry, 0, len(endpoints))
		for _, endpoint := range endpoints {
			items = append(items, snsEndpointEntry{
				EndpointArn: endpoint.ARN,
				Attributes:  buildSNSAttributes(endpoint.Attributes),
			})
		}
		page, nextToken := paginateSNS(items, r.Form.Get("NextToken"), r.Form.Get("MaxResults"))
		respondSNSXML(w, action, snsListEndpointsByPlatformApplicationResult{Endpoints: page, NextToken: nextToken})
		return true
	case "GetEndpointAttributes":
		arn := strings.TrimSpace(r.Form.Get("EndpointArn"))
		if arn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "EndpointArn is required")
			return true
		}
		endpoint, err := s.sns.GetEndpoint(arn)
		if err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "endpoint not found")
			return true
		}
		respondSNSXML(w, action, snsGetEndpointAttributesResult{Attributes: buildSNSAttributes(endpoint.Attributes)})
		return true
	case "SetEndpointAttributes":
		arn := strings.TrimSpace(r.Form.Get("EndpointArn"))
		if arn == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "EndpointArn is required")
			return true
		}
		attrs := parseSNSAttributes(r.Form, "Attributes")
		if err := s.sns.SetEndpointAttributes(arn, attrs); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "endpoint not found")
			return true
		}
		respondSNSXML(w, action, snsSetEndpointAttributesResult{})
		return true
	case "CreateSMSSandboxPhoneNumber":
		number := strings.TrimSpace(r.Form.Get("PhoneNumber"))
		if number == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PhoneNumber is required")
			return true
		}
		if _, err := s.sns.CreateSMSSandboxPhoneNumber(number); err != nil {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", err.Error())
			return true
		}
		respondSNSXML(w, action, snsCreateSMSSandboxPhoneNumberResult{})
		return true
	case "VerifySMSSandboxPhoneNumber":
		number := strings.TrimSpace(r.Form.Get("PhoneNumber"))
		if number == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PhoneNumber is required")
			return true
		}
		if err := s.sns.VerifySMSSandboxPhoneNumber(number); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "phone number not found")
			return true
		}
		respondSNSXML(w, action, snsVerifySMSSandboxPhoneNumberResult{})
		return true
	case "DeleteSMSSandboxPhoneNumber":
		number := strings.TrimSpace(r.Form.Get("PhoneNumber"))
		if number == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PhoneNumber is required")
			return true
		}
		if err := s.sns.DeleteSMSSandboxPhoneNumber(number); err != nil {
			respondSNSErrorXML(w, http.StatusNotFound, "NotFound", "phone number not found")
			return true
		}
		respondSNSXML(w, action, snsDeleteSMSSandboxPhoneNumberResult{})
		return true
	case "ListSMSSandboxPhoneNumbers":
		phones := s.sns.ListSMSSandboxPhoneNumbers()
		items := make([]snsSMSSandboxPhoneNumberEntry, 0, len(phones))
		for _, phone := range phones {
			items = append(items, snsSMSSandboxPhoneNumberEntry{
				PhoneNumber: phone.PhoneNumber,
				Status:      phone.Status,
			})
		}
		page, nextToken := paginateSNS(items, r.Form.Get("NextToken"), r.Form.Get("MaxResults"))
		respondSNSXML(w, action, snsListSMSSandboxPhoneNumbersResult{PhoneNumbers: page, NextToken: nextToken})
		return true
	case "GetSMSSandboxAccountStatus":
		respondSNSXML(w, action, snsGetSMSSandboxAccountStatusResult{IsInSandbox: s.sns.IsSMSSandboxEnabled()})
		return true
	case "GetSMSAttributes":
		attrs := s.sns.GetSMSAttributes()
		names := parseSNSAttributeNames(r.Form, "AttributeNames")
		if len(names) > 0 {
			filtered := map[string]string{}
			for name := range names {
				if val, ok := attrs[name]; ok {
					filtered[name] = val
				}
			}
			attrs = filtered
		}
		respondSNSXML(w, action, snsGetSMSAttributesResult{Attributes: buildSNSAttributes(attrs)})
		return true
	case "SetSMSAttributes":
		attrs := parseSNSAttributes(r.Form, "Attributes")
		s.sns.SetSMSAttributes(attrs)
		respondSNSXML(w, action, snsSetSMSAttributesResult{})
		return true
	case "CheckIfPhoneNumberIsOptedOut":
		number := strings.TrimSpace(r.Form.Get("PhoneNumber"))
		if number == "" {
			number = strings.TrimSpace(r.Form.Get("phoneNumber"))
		}
		if number == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PhoneNumber is required")
			return true
		}
		respondSNSXML(w, action, snsCheckIfPhoneNumberIsOptedOutResult{IsOptedOut: s.sns.CheckIfPhoneNumberIsOptedOut(number)})
		return true
	case "ListPhoneNumbersOptedOut":
		numbers := s.sns.ListPhoneNumbersOptedOut()
		page, nextToken := paginateSNSString(numbers, r.Form.Get("NextToken"), r.Form.Get("MaxResults"))
		respondSNSXML(w, action, snsListPhoneNumbersOptedOutResult{PhoneNumbers: page, NextToken: nextToken})
		return true
	case "OptInPhoneNumber":
		number := strings.TrimSpace(r.Form.Get("PhoneNumber"))
		if number == "" {
			respondSNSErrorXML(w, http.StatusBadRequest, "InvalidParameter", "PhoneNumber is required")
			return true
		}
		s.sns.OptInPhoneNumber(number)
		respondSNSXML(w, action, snsOptInPhoneNumberResult{})
		return true
	case "ListOriginationNumbers":
		page, nextToken := paginateSNS([]snsOriginationNumberEntry{}, r.Form.Get("NextToken"), r.Form.Get("MaxResults"))
		respondSNSXML(w, action, snsListOriginationNumbersResult{PhoneNumbers: page, NextToken: nextToken})
		return true
	default:
		respondSNSErrorXML(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
		return true
	}
}

func isSNSQueryCandidate(r *http.Request) bool {
	action := r.URL.Query().Get("Action")
	if action != "" {
		return isSNSAction(action)
	}
	if r.Method != http.MethodPost {
		return false
	}
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "application/x-www-form-urlencoded") {
		return false
	}
	bodyBytes, err := readBodyBytes(r)
	if err != nil {
		return false
	}
	values, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return false
	}
	action = values.Get("Action")
	if action == "" {
		return false
	}
	return isSNSAction(action)
}

func isSNSAction(action string) bool {
	switch action {
	case "AddPermission",
		"CheckIfPhoneNumberIsOptedOut",
		"ConfirmSubscription",
		"CreatePlatformApplication",
		"CreatePlatformEndpoint",
		"CreateSMSSandboxPhoneNumber",
		"CreateTopic",
		"DeleteEndpoint",
		"DeletePlatformApplication",
		"DeleteSMSSandboxPhoneNumber",
		"DeleteTopic",
		"GetDataProtectionPolicy",
		"GetEndpointAttributes",
		"GetPlatformApplicationAttributes",
		"GetSMSAttributes",
		"GetSMSSandboxAccountStatus",
		"GetSubscriptionAttributes",
		"GetTopicAttributes",
		"ListEndpointsByPlatformApplication",
		"ListOriginationNumbers",
		"ListPhoneNumbersOptedOut",
		"ListPlatformApplications",
		"ListSMSSandboxPhoneNumbers",
		"ListSubscriptions",
		"ListSubscriptionsByTopic",
		"ListTagsForResource",
		"ListTopics",
		"OptInPhoneNumber",
		"Publish",
		"PublishBatch",
		"PutDataProtectionPolicy",
		"RemovePermission",
		"SetEndpointAttributes",
		"SetPlatformApplicationAttributes",
		"SetSMSAttributes",
		"SetSubscriptionAttributes",
		"SetTopicAttributes",
		"Subscribe",
		"TagResource",
		"Unsubscribe",
		"UntagResource",
		"VerifySMSSandboxPhoneNumber":
		return true
	default:
		return false
	}
}

func respondSNSXML(w http.ResponseWriter, action string, result any) {
	env := snsResponseEnvelope{
		XMLName: xml.Name{Local: action + "Response"},
		Xmlns:   snsNamespace,
		Result:  result,
		Metadata: snsResponseMetadata{
			RequestId: "stackyard-request",
		},
	}
	respondXML(w, http.StatusOK, env)
}

func respondSNSErrorXML(w http.ResponseWriter, status int, code, message string) {
	payload := snsErrorResponse{
		Xmlns: snsNamespace,
		Error: snsErrorBody{
			Type:    "Sender",
			Code:    code,
			Message: message,
		},
		RequestId: "stackyard-request",
	}
	respondXML(w, status, payload)
}

func respondXML(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	if err := xml.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("sns response encode error: %v", err)
	}
}

type snsResponseEnvelope struct {
	XMLName  xml.Name            `xml:""`
	Xmlns    string              `xml:"xmlns,attr,omitempty"`
	Result   any                 `xml:",any"`
	Metadata snsResponseMetadata `xml:"ResponseMetadata"`
}

type snsResponseMetadata struct {
	RequestId string `xml:"RequestId"`
}

type snsErrorResponse struct {
	XMLName   xml.Name     `xml:"ErrorResponse"`
	Xmlns     string       `xml:"xmlns,attr,omitempty"`
	Error     snsErrorBody `xml:"Error"`
	RequestId string       `xml:"RequestId"`
}

type snsErrorBody struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type snsAttributeEntry struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type snsTagEntry struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type snsTopicEntry struct {
	TopicArn string `xml:"TopicArn"`
}

type snsSubscriptionEntry struct {
	SubscriptionArn string `xml:"SubscriptionArn"`
	Owner           string `xml:"Owner"`
	Protocol        string `xml:"Protocol"`
	Endpoint        string `xml:"Endpoint"`
	TopicArn        string `xml:"TopicArn"`
}

type snsPlatformApplicationEntry struct {
	PlatformApplicationArn string `xml:"PlatformApplicationArn"`
}

type snsEndpointEntry struct {
	EndpointArn string              `xml:"EndpointArn"`
	Attributes  []snsAttributeEntry `xml:"Attributes>entry"`
}

type snsSMSSandboxPhoneNumberEntry struct {
	PhoneNumber string `xml:"PhoneNumber"`
	Status      string `xml:"Status"`
}

type snsOriginationNumberEntry struct {
	PhoneNumber string `xml:"PhoneNumber"`
	Status      string `xml:"Status"`
}

type snsCreateTopicResult struct {
	XMLName  xml.Name `xml:"CreateTopicResult"`
	TopicArn string   `xml:"TopicArn"`
}

type snsDeleteTopicResult struct {
	XMLName xml.Name `xml:"DeleteTopicResult"`
}

type snsListTopicsResult struct {
	XMLName   xml.Name        `xml:"ListTopicsResult"`
	Topics    []snsTopicEntry `xml:"Topics>member"`
	NextToken string          `xml:"NextToken,omitempty"`
}

type snsGetTopicAttributesResult struct {
	XMLName    xml.Name            `xml:"GetTopicAttributesResult"`
	Attributes []snsAttributeEntry `xml:"Attributes>entry"`
}

type snsSetTopicAttributesResult struct {
	XMLName xml.Name `xml:"SetTopicAttributesResult"`
}

type snsGetDataProtectionPolicyResult struct {
	XMLName              xml.Name `xml:"GetDataProtectionPolicyResult"`
	DataProtectionPolicy string   `xml:"DataProtectionPolicy"`
}

type snsPutDataProtectionPolicyResult struct {
	XMLName xml.Name `xml:"PutDataProtectionPolicyResult"`
}

type snsSubscribeResult struct {
	XMLName         xml.Name `xml:"SubscribeResult"`
	SubscriptionArn string   `xml:"SubscriptionArn"`
}

type snsConfirmSubscriptionResult struct {
	XMLName         xml.Name `xml:"ConfirmSubscriptionResult"`
	SubscriptionArn string   `xml:"SubscriptionArn"`
}

type snsGetSubscriptionAttributesResult struct {
	XMLName    xml.Name            `xml:"GetSubscriptionAttributesResult"`
	Attributes []snsAttributeEntry `xml:"Attributes>entry"`
}

type snsSetSubscriptionAttributesResult struct {
	XMLName xml.Name `xml:"SetSubscriptionAttributesResult"`
}

type snsListSubscriptionsResult struct {
	XMLName       xml.Name               `xml:"ListSubscriptionsResult"`
	Subscriptions []snsSubscriptionEntry `xml:"Subscriptions>member"`
	NextToken     string                 `xml:"NextToken,omitempty"`
}

type snsListSubscriptionsByTopicResult struct {
	XMLName       xml.Name               `xml:"ListSubscriptionsByTopicResult"`
	Subscriptions []snsSubscriptionEntry `xml:"Subscriptions>member"`
	NextToken     string                 `xml:"NextToken,omitempty"`
}

type snsUnsubscribeResult struct {
	XMLName xml.Name `xml:"UnsubscribeResult"`
}

type snsPublishResult struct {
	XMLName   xml.Name `xml:"PublishResult"`
	MessageID string   `xml:"MessageId"`
}

type snsPublishBatchResult struct {
	XMLName    xml.Name                      `xml:"PublishBatchResult"`
	Successful []snsPublishBatchResultEntry  `xml:"Successful>member,omitempty"`
	Failed     []snsPublishBatchFailureEntry `xml:"Failed>member,omitempty"`
}

type snsPublishBatchResultEntry struct {
	ID        string `xml:"Id"`
	MessageID string `xml:"MessageId"`
}

type snsPublishBatchFailureEntry struct {
	ID          string `xml:"Id"`
	Code        string `xml:"Code"`
	Message     string `xml:"Message"`
	SenderFault bool   `xml:"SenderFault"`
}

type snsAddPermissionResult struct {
	XMLName xml.Name `xml:"AddPermissionResult"`
}

type snsRemovePermissionResult struct {
	XMLName xml.Name `xml:"RemovePermissionResult"`
}

type snsTagResourceResult struct {
	XMLName xml.Name `xml:"TagResourceResult"`
}

type snsUntagResourceResult struct {
	XMLName xml.Name `xml:"UntagResourceResult"`
}

type snsListTagsForResourceResult struct {
	XMLName xml.Name      `xml:"ListTagsForResourceResult"`
	Tags    []snsTagEntry `xml:"Tags>member"`
}

type snsCreatePlatformApplicationResult struct {
	XMLName                xml.Name `xml:"CreatePlatformApplicationResult"`
	PlatformApplicationArn string   `xml:"PlatformApplicationArn"`
}

type snsDeletePlatformApplicationResult struct {
	XMLName xml.Name `xml:"DeletePlatformApplicationResult"`
}

type snsListPlatformApplicationsResult struct {
	XMLName              xml.Name                      `xml:"ListPlatformApplicationsResult"`
	PlatformApplications []snsPlatformApplicationEntry `xml:"PlatformApplications>member"`
	NextToken            string                        `xml:"NextToken,omitempty"`
}

type snsGetPlatformApplicationAttributesResult struct {
	XMLName    xml.Name            `xml:"GetPlatformApplicationAttributesResult"`
	Attributes []snsAttributeEntry `xml:"Attributes>entry"`
}

type snsSetPlatformApplicationAttributesResult struct {
	XMLName xml.Name `xml:"SetPlatformApplicationAttributesResult"`
}

type snsCreatePlatformEndpointResult struct {
	XMLName     xml.Name `xml:"CreatePlatformEndpointResult"`
	EndpointArn string   `xml:"EndpointArn"`
}

type snsDeleteEndpointResult struct {
	XMLName xml.Name `xml:"DeleteEndpointResult"`
}

type snsListEndpointsByPlatformApplicationResult struct {
	XMLName   xml.Name           `xml:"ListEndpointsByPlatformApplicationResult"`
	Endpoints []snsEndpointEntry `xml:"Endpoints>member"`
	NextToken string             `xml:"NextToken,omitempty"`
}

type snsGetEndpointAttributesResult struct {
	XMLName    xml.Name            `xml:"GetEndpointAttributesResult"`
	Attributes []snsAttributeEntry `xml:"Attributes>entry"`
}

type snsSetEndpointAttributesResult struct {
	XMLName xml.Name `xml:"SetEndpointAttributesResult"`
}

type snsCreateSMSSandboxPhoneNumberResult struct {
	XMLName xml.Name `xml:"CreateSMSSandboxPhoneNumberResult"`
}

type snsVerifySMSSandboxPhoneNumberResult struct {
	XMLName xml.Name `xml:"VerifySMSSandboxPhoneNumberResult"`
}

type snsDeleteSMSSandboxPhoneNumberResult struct {
	XMLName xml.Name `xml:"DeleteSMSSandboxPhoneNumberResult"`
}

type snsListSMSSandboxPhoneNumbersResult struct {
	XMLName      xml.Name                        `xml:"ListSMSSandboxPhoneNumbersResult"`
	PhoneNumbers []snsSMSSandboxPhoneNumberEntry `xml:"PhoneNumbers>member"`
	NextToken    string                          `xml:"NextToken,omitempty"`
}

type snsGetSMSSandboxAccountStatusResult struct {
	XMLName     xml.Name `xml:"GetSMSSandboxAccountStatusResult"`
	IsInSandbox bool     `xml:"IsInSandbox"`
}

type snsGetSMSAttributesResult struct {
	XMLName    xml.Name            `xml:"GetSMSAttributesResult"`
	Attributes []snsAttributeEntry `xml:"Attributes>entry"`
}

type snsSetSMSAttributesResult struct {
	XMLName xml.Name `xml:"SetSMSAttributesResult"`
}

type snsCheckIfPhoneNumberIsOptedOutResult struct {
	XMLName    xml.Name `xml:"CheckIfPhoneNumberIsOptedOutResult"`
	IsOptedOut bool     `xml:"isOptedOut"`
}

type snsListPhoneNumbersOptedOutResult struct {
	XMLName      xml.Name `xml:"ListPhoneNumbersOptedOutResult"`
	PhoneNumbers []string `xml:"phoneNumbers>member"`
	NextToken    string   `xml:"NextToken,omitempty"`
}

type snsOptInPhoneNumberResult struct {
	XMLName xml.Name `xml:"OptInPhoneNumberResult"`
}

type snsListOriginationNumbersResult struct {
	XMLName      xml.Name                    `xml:"ListOriginationNumbersResult"`
	PhoneNumbers []snsOriginationNumberEntry `xml:"PhoneNumbers>member"`
	NextToken    string                      `xml:"NextToken,omitempty"`
}

func parseSNSAttributes(values url.Values, prefix string) map[string]string {
	items := map[string]*snsAttributeEntry{}
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix+".entry.") && !strings.HasPrefix(key, prefix+".member.") {
			continue
		}
		parts := strings.Split(key, ".")
		if len(parts) < 4 {
			continue
		}
		index := parts[2]
		field := strings.ToLower(parts[3])
		entry := items[index]
		if entry == nil {
			entry = &snsAttributeEntry{}
			items[index] = entry
		}
		if len(vals) == 0 {
			continue
		}
		switch field {
		case "key":
			entry.Key = vals[0]
		case "value":
			entry.Value = vals[0]
		}
	}
	out := map[string]string{}
	indexes := make([]string, 0, len(items))
	for idx := range items {
		indexes = append(indexes, idx)
	}
	sort.Strings(indexes)
	for _, idx := range indexes {
		entry := items[idx]
		if entry.Key == "" {
			continue
		}
		out[entry.Key] = entry.Value
	}
	return out
}

func parseSNSTags(values url.Values) map[string]string {
	items := map[string]*snsTagEntry{}
	for key, vals := range values {
		if !strings.HasPrefix(key, "Tags.member.") {
			continue
		}
		parts := strings.Split(key, ".")
		if len(parts) < 4 {
			continue
		}
		index := parts[2]
		field := strings.ToLower(parts[3])
		entry := items[index]
		if entry == nil {
			entry = &snsTagEntry{}
			items[index] = entry
		}
		if len(vals) == 0 {
			continue
		}
		switch field {
		case "key":
			entry.Key = vals[0]
		case "value":
			entry.Value = vals[0]
		}
	}
	out := map[string]string{}
	indexes := make([]string, 0, len(items))
	for idx := range items {
		indexes = append(indexes, idx)
	}
	sort.Strings(indexes)
	for _, idx := range indexes {
		entry := items[idx]
		if entry.Key == "" {
			continue
		}
		out[entry.Key] = entry.Value
	}
	return out
}

func parseSNSTagKeys(values url.Values) []string {
	keys := []string{}
	for key, vals := range values {
		if !strings.HasPrefix(key, "TagKeys.member.") {
			continue
		}
		if len(vals) == 0 {
			continue
		}
		keys = append(keys, vals[0])
	}
	sort.Strings(keys)
	return keys
}

func parseSNSAttributeNames(values url.Values, prefix string) map[string]bool {
	names := map[string]bool{}
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix+".member.") {
			continue
		}
		if len(vals) == 0 {
			continue
		}
		names[vals[0]] = true
	}
	return names
}

type snsPublishEntryInput struct {
	ID      string
	Message string
	Subject string
}

func parseSNSPublishBatchEntries(values url.Values) []sns.PublishEntry {
	items := map[string]*snsPublishEntryInput{}
	for key, vals := range values {
		if !strings.HasPrefix(key, "PublishBatchRequestEntries.member.") {
			continue
		}
		parts := strings.Split(key, ".")
		if len(parts) < 4 {
			continue
		}
		index := parts[2]
		field := strings.ToLower(parts[3])
		item := items[index]
		if item == nil {
			item = &snsPublishEntryInput{}
			items[index] = item
		}
		if len(vals) == 0 {
			continue
		}
		switch field {
		case "id":
			item.ID = vals[0]
		case "message":
			item.Message = vals[0]
		case "subject":
			item.Subject = vals[0]
		}
	}
	indexes := make([]string, 0, len(items))
	for idx := range items {
		indexes = append(indexes, idx)
	}
	sort.Strings(indexes)
	out := make([]sns.PublishEntry, 0, len(indexes))
	for _, idx := range indexes {
		item := items[idx]
		out = append(out, sns.PublishEntry{
			ID:      item.ID,
			Message: item.Message,
			Subject: item.Subject,
		})
	}
	return out
}

func buildSNSAttributes(attrs map[string]string) []snsAttributeEntry {
	if len(attrs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]snsAttributeEntry, 0, len(keys))
	for _, key := range keys {
		out = append(out, snsAttributeEntry{Key: key, Value: attrs[key]})
	}
	return out
}

func buildSNSTags(tags map[string]string) []snsTagEntry {
	if len(tags) == 0 {
		return nil
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]snsTagEntry, 0, len(keys))
	for _, key := range keys {
		out = append(out, snsTagEntry{Key: key, Value: tags[key]})
	}
	return out
}

func buildSNSSubscriptions(subs []sns.Subscription) []snsSubscriptionEntry {
	out := make([]snsSubscriptionEntry, 0, len(subs))
	for _, sub := range subs {
		out = append(out, snsSubscriptionEntry{
			SubscriptionArn: sub.ARN,
			Owner:           sub.Owner,
			Protocol:        sub.Protocol,
			Endpoint:        sub.Endpoint,
			TopicArn:        sub.TopicARN,
		})
	}
	return out
}

func buildSNSPublishBatchSuccess(items []sns.PublishBatchSuccess) []snsPublishBatchResultEntry {
	out := make([]snsPublishBatchResultEntry, 0, len(items))
	for _, item := range items {
		out = append(out, snsPublishBatchResultEntry{
			ID:        item.ID,
			MessageID: item.MessageID,
		})
	}
	return out
}

func buildSNSPublishBatchFailure(items []sns.PublishBatchFailure) []snsPublishBatchFailureEntry {
	out := make([]snsPublishBatchFailureEntry, 0, len(items))
	for _, item := range items {
		out = append(out, snsPublishBatchFailureEntry{
			ID:          item.ID,
			Code:        item.Code,
			Message:     item.Message,
			SenderFault: true,
		})
	}
	return out
}

func paginateSNS[T any](items []T, nextTokenRaw, maxResultsRaw string) ([]T, string) {
	start := parseSNSNextToken(nextTokenRaw)
	maxResults := parseSNSMaxResults(maxResultsRaw)
	if start < 0 || start > len(items) {
		start = 0
	}
	end := len(items)
	if maxResults > 0 && start+maxResults < end {
		end = start + maxResults
	}
	var nextToken string
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	}
	return items[start:end], nextToken
}

func paginateSNSString(items []string, nextTokenRaw, maxResultsRaw string) ([]string, string) {
	start := parseSNSNextToken(nextTokenRaw)
	maxResults := parseSNSMaxResults(maxResultsRaw)
	if start < 0 || start > len(items) {
		start = 0
	}
	end := len(items)
	if maxResults > 0 && start+maxResults < end {
		end = start + maxResults
	}
	var nextToken string
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	}
	return items[start:end], nextToken
}

func parseSNSNextToken(value string) int {
	if value == "" {
		return 0
	}
	v, err := strconv.Atoi(value)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func parseSNSMaxResults(value string) int {
	if value == "" {
		return 0
	}
	v, err := strconv.Atoi(value)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

func snsProtocolRequiresConfirmation(protocol string) bool {
	switch strings.ToLower(protocol) {
	case "email", "email-json", "http", "https":
		return true
	default:
		return false
	}
}
