package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/services/sesv2"
)

type sesv2Error struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSESV2JSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSESV2JSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ses")
	if !ok {
		respondSESV2Error(w, status, code, msg)
		return true
	}

	op, params, matched := matchSESV2Operation(r.Method, r.URL.Path)
	if !matched {
		respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "operation not found")
		return true
	}

	payload, err := parseSESV2Payload(r)
	if err != nil {
		respondSESV2Error(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return true
	}

	switch op.Name {
	case "CreateConfigurationSet":
		name := sesv2String(payload["ConfigurationSetName"])
		if err := s.sesv2.CreateConfigurationSet(name); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetConfigurationSet":
		cfg, ok := s.sesv2.GetConfigurationSet(params["ConfigurationSetName"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "configuration set not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"ConfigurationSetName": cfg.Name,
		})
		return true
	case "DeleteConfigurationSet":
		if !s.sesv2.DeleteConfigurationSet(params["ConfigurationSetName"]) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "configuration set not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListConfigurationSets":
		cfgs := s.sesv2.ListConfigurationSets()
		names := make([]string, 0, len(cfgs))
		for _, cfg := range cfgs {
			names = append(names, cfg.Name)
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"ConfigurationSets": names})
		return true
	case "CreateConfigurationSetEventDestination", "UpdateConfigurationSetEventDestination":
		cfg := params["ConfigurationSetName"]
		dest := params["EventDestinationName"]
		if dest == "" {
			dest = sesv2String(payload["EventDestinationName"])
		}
		if err := s.sesv2.UpsertConfigurationSetEventDestination(cfg, dest, payload); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteConfigurationSetEventDestination":
		if !s.sesv2.DeleteConfigurationSetEventDestination(params["ConfigurationSetName"], params["EventDestinationName"]) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "event destination not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetConfigurationSetEventDestinations":
		destinations, ok := s.sesv2.GetConfigurationSetEventDestinations(params["ConfigurationSetName"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "configuration set not found")
			return true
		}
		entries := make([]map[string]any, 0, len(destinations))
		for name, stored := range destinations {
			entry := map[string]any{"Name": name}
			payload := stored
			if wrapped := sesv2Map(stored["EventDestination"]); len(wrapped) != 0 {
				payload = wrapped
			}
			if enabled, ok := sesv2Bool(payload["Enabled"]); ok {
				entry["Enabled"] = enabled
			}
			matchingEventTypes := sesv2StringSlice(payload["MatchingEventTypes"])
			if len(matchingEventTypes) == 0 {
				matchingEventTypes = []string{"SEND"}
			}
			entry["MatchingEventTypes"] = matchingEventTypes
			if sns := sesv2Map(payload["SnsDestination"]); len(sns) != 0 {
				entry["SnsDestination"] = map[string]any{"TopicArn": sesv2String(sns["TopicArn"])}
			}
			if firehose := sesv2Map(payload["KinesisFirehoseDestination"]); len(firehose) != 0 {
				entry["KinesisFirehoseDestination"] = map[string]any{
					"IamRoleArn":        sesv2String(firehose["IamRoleArn"]),
					"DeliveryStreamArn": sesv2String(firehose["DeliveryStreamArn"]),
				}
			}
			if cloudWatch := sesv2Map(payload["CloudWatchDestination"]); len(cloudWatch) != 0 {
				entry["CloudWatchDestination"] = map[string]any{
					"DimensionConfigurations": cloudWatch["DimensionConfigurations"],
				}
			}
			if eventBridge := sesv2Map(payload["EventBridgeDestination"]); len(eventBridge) != 0 {
				entry["EventBridgeDestination"] = map[string]any{"EventBusArn": sesv2String(eventBridge["EventBusArn"])}
			}
			if pinpoint := sesv2Map(payload["PinpointDestination"]); len(pinpoint) != 0 {
				entry["PinpointDestination"] = map[string]any{"ApplicationArn": sesv2String(pinpoint["ApplicationArn"])}
			}
			entries = append(entries, entry)
		}
		sort.Slice(entries, func(i, j int) bool {
			return sesv2String(entries[i]["Name"]) < sesv2String(entries[j]["Name"])
		})
		respondSESV2JSON(w, http.StatusOK, map[string]any{"EventDestinations": entries})
		return true
	case "CreateEmailIdentity":
		identity, err := s.sesv2.CreateEmailIdentity(
			sesv2String(payload["EmailIdentity"]),
			sesv2String(payload["ConfigurationSetName"]),
			sesv2TagsToMap(payload["Tags"]),
		)
		if err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"IdentityType":             identity.IdentityType,
			"VerifiedForSendingStatus": identity.VerifiedForSendingStatus,
			"DkimAttributes": map[string]any{
				"SigningEnabled": identity.DkimSigningEnabled,
				"Status":         identity.DkimStatus,
				"Tokens":         identity.DkimTokens,
			},
		})
		return true
	case "GetEmailIdentity":
		identity, ok := s.sesv2.GetEmailIdentity(params["EmailIdentity"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "identity not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"ConfigurationSetName":     identity.ConfigurationSetName,
			"FeedbackForwardingStatus": identity.FeedbackForwardingStatus,
			"IdentityType":             identity.IdentityType,
			"VerificationStatus":       identity.VerificationStatus,
			"VerifiedForSendingStatus": identity.VerifiedForSendingStatus,
			"DkimAttributes": map[string]any{
				"SigningEnabled": identity.DkimSigningEnabled,
				"Status":         identity.DkimStatus,
				"Tokens":         identity.DkimTokens,
			},
			"MailFromAttributes": map[string]any{
				"BehaviorOnMxFailure":  identity.BehaviorOnMxFailure,
				"MailFromDomain":       identity.MailFromDomain,
				"MailFromDomainStatus": identity.MailFromDomainStatus,
			},
			"Policies": sesv2StringMapToAny(identity.Policies),
			"Tags":     sesv2MapToTags(identity.Tags),
		})
		return true
	case "ListEmailIdentities":
		identities := s.sesv2.ListEmailIdentities()
		items := make([]map[string]any, 0, len(identities))
		for _, identity := range identities {
			items = append(items, map[string]any{
				"IdentityName":       identity.EmailIdentity,
				"IdentityType":       identity.IdentityType,
				"SendingEnabled":     identity.VerifiedForSendingStatus,
				"VerificationStatus": identity.VerificationStatus,
			})
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"EmailIdentities": items})
		return true
	case "DeleteEmailIdentity":
		if !s.sesv2.DeleteEmailIdentity(params["EmailIdentity"]) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "identity not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "PutEmailIdentityConfigurationSetAttributes":
		if err := s.sesv2.PutEmailIdentityConfigurationSet(params["EmailIdentity"], sesv2String(payload["ConfigurationSetName"])); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "PutEmailIdentityFeedbackAttributes":
		enabled, ok := sesv2Bool(payload["EmailForwardingEnabled"])
		if !ok {
			enabled = true
		}
		if err := s.sesv2.PutEmailIdentityFeedback(params["EmailIdentity"], enabled); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "PutEmailIdentityDkimAttributes":
		signingEnabled := true
		if v, ok := sesv2Bool(payload["SigningEnabled"]); ok {
			signingEnabled = v
		}
		if err := s.sesv2.PutEmailIdentityDkim(params["EmailIdentity"], signingEnabled, nil); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "PutEmailIdentityDkimSigningAttributes":
		signingEnabled := true
		if v, ok := sesv2Bool(payload["SigningEnabled"]); ok {
			signingEnabled = v
		}
		if err := s.sesv2.PutEmailIdentityDkim(params["EmailIdentity"], signingEnabled, payload); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		identity, _ := s.sesv2.GetEmailIdentity(params["EmailIdentity"])
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DkimStatus": identity.DkimStatus,
			"DkimTokens": identity.DkimTokens,
		})
		return true
	case "PutEmailIdentityMailFromAttributes":
		if err := s.sesv2.PutEmailIdentityMailFrom(
			params["EmailIdentity"],
			sesv2String(payload["MailFromDomain"]),
			sesv2String(payload["BehaviorOnMxFailure"]),
		); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateEmailIdentityPolicy", "UpdateEmailIdentityPolicy":
		if err := s.sesv2.CreateEmailIdentityPolicy(params["EmailIdentity"], params["PolicyName"], sesv2String(payload["Policy"])); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteEmailIdentityPolicy":
		if !s.sesv2.DeleteEmailIdentityPolicy(params["EmailIdentity"], params["PolicyName"]) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "policy not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetEmailIdentityPolicies":
		policies, ok := s.sesv2.GetEmailIdentityPolicies(params["EmailIdentity"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "identity not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"Policies": sesv2StringMapToAny(policies)})
		return true
	case "CreateEmailTemplate":
		templateName := sesv2String(payload["TemplateName"])
		templateContent := sesv2Map(payload["TemplateContent"])
		if err := s.sesv2.CreateEmailTemplate(
			templateName,
			sesv2String(templateContent["Subject"]),
			sesv2String(templateContent["Text"]),
			sesv2String(templateContent["Html"]),
		); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetEmailTemplate":
		tpl, ok := s.sesv2.GetEmailTemplate(params["TemplateName"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "template not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"TemplateName": tpl.TemplateName,
			"TemplateContent": map[string]any{
				"Subject": tpl.Subject,
				"Text":    tpl.Text,
				"Html":    tpl.HTML,
			},
		})
		return true
	case "ListEmailTemplates":
		templates := s.sesv2.ListEmailTemplates()
		items := make([]map[string]any, 0, len(templates))
		for _, tpl := range templates {
			items = append(items, map[string]any{
				"TemplateName":     tpl.TemplateName,
				"CreatedTimestamp": sesv2Timestamp(tpl.CreatedAt),
			})
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"TemplatesMetadata": items})
		return true
	case "UpdateEmailTemplate":
		templateContent := sesv2Map(payload["TemplateContent"])
		if err := s.sesv2.UpdateEmailTemplate(
			params["TemplateName"],
			sesv2String(templateContent["Subject"]),
			sesv2String(templateContent["Text"]),
			sesv2String(templateContent["Html"]),
		); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteEmailTemplate":
		if !s.sesv2.DeleteEmailTemplate(params["TemplateName"]) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "template not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "TestRenderEmailTemplate":
		rendered, err := s.sesv2.RenderEmailTemplate(params["TemplateName"], sesv2String(payload["TemplateData"]))
		if err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"RenderedTemplate": rendered})
		return true
	case "SendEmail":
		from := sesv2String(payload["FromEmailAddress"])
		destination := sesv2Map(payload["Destination"])
		recipients := append(sesv2StringSlice(destination["ToAddresses"]), sesv2StringSlice(destination["CcAddresses"])...)
		recipients = append(recipients, sesv2StringSlice(destination["BccAddresses"])...)
		messageID, err := s.sesv2.SendEmail(from, recipients)
		if err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"MessageId": messageID})
		return true
	case "SendBulkEmail":
		entries := sesv2List(payload["BulkEmailEntries"])
		results, err := s.sesv2.SendBulkEmail(len(entries))
		if err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(results))
		for _, res := range results {
			items = append(items, map[string]any{"Status": res.Status, "MessageId": res.MessageID})
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"BulkEmailEntryResults": items})
		return true
	case "PutAccountSendingAttributes":
		// AWS SDK v2 serializes this shape with "omitempty" semantics for bool and
		// sends {} when false. Treat missing as false for SDK compatibility.
		enabled, ok := sesv2Bool(payload["SendingEnabled"])
		if !ok {
			enabled = false
		}
		s.sesv2.PutAccountSendingAttributes(enabled)
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "PutAccountSuppressionAttributes":
		s.sesv2.PutAccountSuppressionAttributes(sesv2StringSlice(payload["SuppressedReasons"]))
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetAccount":
		account := s.sesv2.GetAccount()
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"SendingEnabled":               account.SendingEnabled,
			"ProductionAccessEnabled":      account.ProductionAccessEnabled,
			"DedicatedIpAutoWarmupEnabled": account.DedicatedIpAutoWarmupEnabled,
			"SendQuota": map[string]any{
				"Max24HourSend":   account.SendQuota.Max24HourSend,
				"MaxSendRate":     account.SendQuota.MaxSendRate,
				"SentLast24Hours": account.SendQuota.SentLast24Hours,
			},
			"SuppressionAttributes": map[string]any{"SuppressedReasons": account.SuppressedReasons},
		})
		return true
	case "PutSuppressedDestination":
		destination, err := s.sesv2.PutSuppressedDestination(
			sesv2String(payload["EmailAddress"]),
			sesv2String(payload["Reason"]),
		)
		if err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"SuppressedDestination": map[string]any{
				"EmailAddress":   destination.EmailAddress,
				"Reason":         destination.Reason,
				"LastUpdateTime": sesv2Timestamp(destination.LastUpdateTime),
			},
		})
		return true
	case "GetSuppressedDestination":
		destination, ok := s.sesv2.GetSuppressedDestination(params["EmailAddress"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "suppressed destination not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"SuppressedDestination": map[string]any{
				"EmailAddress":   destination.EmailAddress,
				"Reason":         destination.Reason,
				"LastUpdateTime": sesv2Timestamp(destination.LastUpdateTime),
			},
		})
		return true
	case "DeleteSuppressedDestination":
		if !s.sesv2.DeleteSuppressedDestination(params["EmailAddress"]) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "suppressed destination not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListSuppressedDestinations":
		items := s.sesv2.ListSuppressedDestinations()
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			out = append(out, map[string]any{
				"EmailAddress":   item.EmailAddress,
				"Reason":         item.Reason,
				"LastUpdateTime": sesv2Timestamp(item.LastUpdateTime),
			})
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"SuppressedDestinationSummaries": out})
		return true
	case "TagResource":
		if err := s.sesv2.TagResource(sesv2String(payload["ResourceArn"]), sesv2TagsToMap(payload["Tags"])); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "UntagResource":
		resource := firstNonEmpty(r.URL.Query().Get("ResourceArn"), sesv2String(payload["ResourceArn"]))
		s.sesv2.UntagResource(resource, sesv2StringSlice(payload["TagKeys"]))
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListTagsForResource":
		resource := firstNonEmpty(r.URL.Query().Get("ResourceArn"), sesv2String(payload["ResourceArn"]))
		tags := s.sesv2.ListTags(resource)
		respondSESV2JSON(w, http.StatusOK, map[string]any{"Tags": sesv2MapToTags(tags)})
		return true
	case "CreateContactList":
		if err := s.sesv2.CreateContactList(sesv2String(payload["ContactListName"]), sesv2String(payload["Description"])); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "UpdateContactList":
		if err := s.sesv2.UpdateContactList(params["ContactListName"], sesv2String(payload["Description"])); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetContactList":
		list, ok := s.sesv2.GetContactList(params["ContactListName"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "contact list not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"ContactListName":      list.Name,
			"CreatedTimestamp":     sesv2Timestamp(list.CreatedAt),
			"LastUpdatedTimestamp": sesv2Timestamp(list.LastUpdatedAt),
		})
		return true
	case "ListContactLists":
		lists := s.sesv2.ListContactLists()
		out := make([]map[string]any, 0, len(lists))
		for _, list := range lists {
			out = append(out, map[string]any{
				"ContactListName":      list.Name,
				"LastUpdatedTimestamp": sesv2Timestamp(list.LastUpdatedAt),
			})
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"ContactLists": out})
		return true
	case "DeleteContactList":
		if !s.sesv2.DeleteContactList(params["ContactListName"]) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "contact list not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateContact":
		if err := s.sesv2.CreateContact(
			params["ContactListName"],
			sesv2String(payload["EmailAddress"]),
			sesv2String(payload["AttributesData"]),
			sesv2BoolValue(payload["UnsubscribeAll"]),
			sesv2TopicPreferences(payload["TopicPreferences"]),
		); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "UpdateContact":
		if err := s.sesv2.UpdateContact(
			params["ContactListName"],
			params["EmailAddress"],
			sesv2String(payload["AttributesData"]),
			sesv2BoolValue(payload["UnsubscribeAll"]),
			sesv2TopicPreferences(payload["TopicPreferences"]),
		); err != nil {
			respondSESV2ErrorForErr(w, err)
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetContact":
		contact, ok := s.sesv2.GetContact(params["ContactListName"], params["EmailAddress"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "contact not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"ContactListName":      params["ContactListName"],
			"EmailAddress":         contact.EmailAddress,
			"AttributesData":       contact.AttributesData,
			"UnsubscribeAll":       contact.UnsubscribeAll,
			"TopicPreferences":     contact.TopicPreferences,
			"CreatedTimestamp":     sesv2Timestamp(contact.CreatedAt),
			"LastUpdatedTimestamp": sesv2Timestamp(contact.LastUpdatedAt),
		})
		return true
	case "ListContacts":
		contacts, ok := s.sesv2.ListContacts(params["ContactListName"])
		if !ok {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "contact list not found")
			return true
		}
		items := make([]map[string]any, 0, len(contacts))
		for _, contact := range contacts {
			items = append(items, map[string]any{
				"EmailAddress":         contact.EmailAddress,
				"TopicPreferences":     contact.TopicPreferences,
				"UnsubscribeAll":       contact.UnsubscribeAll,
				"LastUpdatedTimestamp": sesv2Timestamp(contact.LastUpdatedAt),
			})
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"Contacts": items})
		return true
	case "DeleteContact":
		if !s.sesv2.DeleteContact(params["ContactListName"], params["EmailAddress"]) {
			respondSESV2Error(w, http.StatusNotFound, "NotFoundException", "contact not found")
			return true
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateDeliverabilityTestReport":
		reportID := "sesv2-report-000001"
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"ReportId":                 reportID,
			"DeliverabilityTestStatus": "COMPLETED",
		})
		return true
	case "CreateExportJob":
		respondSESV2JSON(w, http.StatusOK, map[string]any{"JobId": "sesv2-export-job-000001"})
		return true
	case "CreateImportJob":
		respondSESV2JSON(w, http.StatusOK, map[string]any{"JobId": "sesv2-import-job-000001"})
		return true
	case "SendCustomVerificationEmail":
		respondSESV2JSON(w, http.StatusOK, map[string]any{"MessageId": "sesv2-custom-verification-000001"})
		return true
	case "GetBlacklistReports":
		report := map[string]any{}
		blacklistNames := sesv2StringSlice(payload["BlacklistItemNames"])
		if len(blacklistNames) == 0 {
			blacklistNames = []string{"spamhaus"}
		}
		for _, name := range blacklistNames {
			report[name] = []map[string]any{{
				"RblName":     name,
				"ListingTime": sesv2PlaceholderTimestamp(),
				"Description": "stackyard blacklist entry",
			}}
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{"BlacklistReport": report})
		return true
	case "GetCustomVerificationEmailTemplate":
		templateName := firstNonEmpty(params["TemplateName"], sesv2String(payload["TemplateName"]), "stackyard-sesv2-custom-verify")
		respondSESV2JSON(w, http.StatusOK, sesv2DefaultCustomVerificationTemplate(templateName))
		return true
	case "GetDedicatedIp":
		ip := firstNonEmpty(params["Ip"], sesv2String(payload["Ip"]), "198.51.100.10")
		poolName := firstNonEmpty(params["PoolName"], sesv2String(payload["PoolName"]), "stackyard-sesv2-pool")
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DedicatedIp": sesv2DefaultDedicatedIP(ip, poolName),
		})
		return true
	case "GetDedicatedIpPool":
		poolName := firstNonEmpty(params["PoolName"], sesv2String(payload["PoolName"]), "stackyard-sesv2-pool")
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DedicatedIpPool": map[string]any{
				"PoolName":    poolName,
				"ScalingMode": "MANAGED",
			},
		})
		return true
	case "GetDedicatedIps":
		poolName := firstNonEmpty(params["PoolName"], sesv2String(payload["PoolName"]), "stackyard-sesv2-pool")
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DedicatedIps": []map[string]any{sesv2DefaultDedicatedIP("198.51.100.10", poolName)},
			"NextToken":    "",
		})
		return true
	case "GetDeliverabilityDashboardOptions":
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DashboardEnabled": true,
			"AccountStatus":    "ACTIVE",
			"ActiveSubscribedDomains": []map[string]any{{
				"Domain":                "example.com",
				"SubscriptionStartDate": sesv2PlaceholderTimestamp(),
			}},
			"PendingExpirationSubscribedDomains": []map[string]any{},
		})
		return true
	case "GetDeliverabilityTestReport":
		reportID := firstNonEmpty(params["ReportId"], sesv2String(payload["ReportId"]), "sesv2-report-000001")
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DeliverabilityTestReport": sesv2DefaultDeliverabilityTestReport(reportID),
			"OverallPlacement":         sesv2DefaultPlacementStatistics(),
			"IspPlacements": []map[string]any{{
				"IspName":             "gmail.com",
				"PlacementStatistics": sesv2DefaultPlacementStatistics(),
			}},
			"Message": "stackyard deliverability test message",
			"Tags":    []map[string]any{{"Key": "env", "Value": "coverage"}},
		})
		return true
	case "GetDomainDeliverabilityCampaign":
		campaignID := firstNonEmpty(params["CampaignId"], sesv2String(payload["CampaignId"]), "sesv2-campaign-000001")
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DomainDeliverabilityCampaign": sesv2DefaultDomainDeliverabilityCampaign(campaignID),
		})
		return true
	case "GetDomainStatisticsReport":
		now := time.Now().UTC()
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"OverallVolume": sesv2DefaultOverallVolume(),
			"DailyVolumes": []map[string]any{{
				"StartDate":        sesv2Timestamp(now.Add(-24 * time.Hour)),
				"VolumeStatistics": sesv2DefaultVolumeStatistics(),
			}},
		})
		return true
	case "GetExportJob":
		jobID := firstNonEmpty(params["JobId"], sesv2String(payload["JobId"]), "sesv2-export-job-000001")
		respondSESV2JSON(w, http.StatusOK, sesv2DefaultExportJob(jobID))
		return true
	case "GetImportJob":
		jobID := firstNonEmpty(params["JobId"], sesv2String(payload["JobId"]), "sesv2-import-job-000001")
		respondSESV2JSON(w, http.StatusOK, sesv2DefaultImportJob(jobID))
		return true
	case "GetMessageInsights":
		messageID := firstNonEmpty(params["MessageId"], sesv2String(payload["MessageId"]), "sesv2-message-000001")
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"MessageId":        messageID,
			"FromEmailAddress": "sender@example.com",
			"Subject":          "stackyard subject",
			"EmailTags":        []map[string]any{{"Name": "env", "Value": "coverage"}},
			"Insights": []map[string]any{{
				"Destination": "recipient@example.com",
				"Isp":         "gmail.com",
			}},
		})
		return true
	case "ListCustomVerificationEmailTemplates":
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"CustomVerificationEmailTemplates": []map[string]any{sesv2DefaultCustomVerificationTemplateMetadata("stackyard-sesv2-custom-verify")},
			"NextToken":                        "",
		})
		return true
	case "ListDedicatedIpPools":
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DedicatedIpPools": []string{"stackyard-sesv2-pool"},
			"NextToken":        "",
		})
		return true
	case "ListDeliverabilityTestReports":
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DeliverabilityTestReports": []map[string]any{sesv2DefaultDeliverabilityTestReport("sesv2-report-000001")},
			"NextToken":                 "",
		})
		return true
	case "ListDomainDeliverabilityCampaigns":
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"DomainDeliverabilityCampaigns": []map[string]any{sesv2DefaultDomainDeliverabilityCampaign("sesv2-campaign-000001")},
			"NextToken":                     "",
		})
		return true
	case "ListExportJobs":
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"ExportJobs": []map[string]any{sesv2DefaultExportJobSummary("sesv2-export-job-000001")},
			"NextToken":  "",
		})
		return true
	case "ListImportJobs":
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"ImportJobs": []map[string]any{sesv2DefaultImportJobSummary("sesv2-import-job-000001")},
			"NextToken":  "",
		})
		return true
	case "ListRecommendations":
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"Recommendations": []map[string]any{{
				"ResourceArn":          "arn:aws:ses:us-east-1:123456789012:configuration-set/stackyard-sesv2-config",
				"Type":                 "DKIM",
				"Description":          "Enable DKIM signing for the identity.",
				"Status":               "OPEN",
				"CreatedTimestamp":     sesv2PlaceholderTimestamp(),
				"LastUpdatedTimestamp": sesv2PlaceholderTimestamp(),
				"Impact":               "HIGH",
			}},
			"NextToken": "",
		})
		return true
	case "ListReputationEntities":
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"ReputationEntities": []map[string]any{sesv2DefaultReputationEntity("IP_ADDRESS", "198.51.100.10")},
			"NextToken":          "",
		})
		return true
	case "BatchGetMetricData":
		results := make([]map[string]any, 0, len(sesv2List(payload["Queries"])))
		for _, item := range sesv2List(payload["Queries"]) {
			query := sesv2Map(item)
			queryID := firstNonEmpty(sesv2String(query["Id"]), "metric")
			results = append(results, map[string]any{
				"Id":         queryID,
				"Timestamps": []float64{sesv2PlaceholderTimestamp()},
				"Values":     []float64{1.0},
			})
		}
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"Results": results,
			"Errors":  []map[string]any{},
		})
		return true
	default:
		respondSESV2JSON(w, http.StatusOK, map[string]any{
			"Operation": op.Name,
			"Status":    "OK",
		})
		return true
	}
}

func respondSESV2JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSESV2Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSESV2JSON(w, status, sesv2Error{Type: code, Message: msg})
}

func respondSESV2ErrorForErr(w http.ResponseWriter, err error) {
	switch err {
	case sesv2.ErrInvalidParameter:
		respondSESV2Error(w, http.StatusBadRequest, "BadRequestException", err.Error())
	case sesv2.ErrAlreadyExists:
		respondSESV2Error(w, http.StatusBadRequest, "AlreadyExistsException", err.Error())
	case sesv2.ErrNotFound:
		respondSESV2Error(w, http.StatusNotFound, "NotFoundException", err.Error())
	case sesv2.ErrSendingDisabled:
		respondSESV2Error(w, http.StatusBadRequest, "SendingPausedException", err.Error())
	default:
		respondSESV2Error(w, http.StatusBadRequest, "BadRequestException", err.Error())
	}
}

func isSESV2JSONCandidate(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}
	path := normalizeSESV2Path(r.URL.Path)
	return strings.HasPrefix(path, "/v2/email") || strings.HasPrefix(path, "/v1/email")
}

func matchSESV2Operation(method, path string) (sesv2Operation, map[string]string, bool) {
	for _, op := range sesv2Operations {
		if method != op.Method {
			continue
		}
		if params, ok := matchSESV2PathPattern(op.Pattern, path); ok {
			return op, params, true
		}
	}
	return sesv2Operation{}, nil, false
}

func matchSESV2PathPattern(pattern, actual string) (map[string]string, bool) {
	patternSegs := splitSESV2Path(pattern)
	actualSegs := splitSESV2Path(actual)
	if len(patternSegs) != len(actualSegs) {
		return nil, false
	}
	params := map[string]string{}
	for i, seg := range patternSegs {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") && len(seg) > 2 {
			name := seg[1 : len(seg)-1]
			value, err := url.PathUnescape(actualSegs[i])
			if err != nil {
				value = actualSegs[i]
			}
			params[name] = value
			continue
		}
		if seg != actualSegs[i] {
			return nil, false
		}
	}
	return params, true
}

func normalizeSESV2Path(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
		if path == "" {
			return "/"
		}
	}
	return path
}

func splitSESV2Path(path string) []string {
	path = normalizeSESV2Path(path)
	if path == "/" {
		return nil
	}
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

func parseSESV2Payload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return obj, nil
}

func sesv2Map(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return obj
}

func sesv2List(value any) []any {
	if value == nil {
		return nil
	}
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	return list
}

func sesv2String(value any) string {
	if value == nil {
		return ""
	}
	v, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func sesv2StringSlice(value any) []string {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		v := sesv2String(item)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func sesv2Bool(value any) (bool, bool) {
	if value == nil {
		return false, false
	}
	v, ok := value.(bool)
	return v, ok
}

func sesv2BoolValue(value any) bool {
	v, _ := sesv2Bool(value)
	return v
}

func sesv2TagsToMap(value any) map[string]string {
	list, ok := value.([]any)
	if !ok {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		k := sesv2String(entry["Key"])
		if k == "" {
			continue
		}
		out[k] = sesv2String(entry["Value"])
	}
	return out
}

func sesv2MapToTags(tags map[string]string) []map[string]string {
	if len(tags) == 0 {
		return []map[string]string{}
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]string{"Key": key, "Value": tags[key]})
	}
	return out
}

func sesv2StringMapToAny(input map[string]string) map[string]any {
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func sesv2TopicPreferences(value any) []map[string]string {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]string, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		topic := sesv2String(entry["TopicName"])
		if topic == "" {
			continue
		}
		status := strings.ToUpper(sesv2String(entry["SubscriptionStatus"]))
		if status == "" {
			status = "OPT_IN"
		}
		out = append(out, map[string]string{
			"TopicName":          topic,
			"SubscriptionStatus": status,
		})
	}
	return out
}

func sesv2PlaceholderTimestamp() float64 {
	return sesv2Timestamp(time.Now().UTC())
}

func sesv2DefaultPlacementStatistics() map[string]any {
	return map[string]any{
		"InboxPercentage":   98.0,
		"SpamPercentage":    1.0,
		"MissingPercentage": 1.0,
		"SpfPercentage":     100.0,
		"DkimPercentage":    100.0,
	}
}

func sesv2DefaultDeliverabilityTestReport(reportID string) map[string]any {
	return map[string]any{
		"ReportId":                 reportID,
		"ReportName":               "stackyard deliverability report",
		"Subject":                  "stackyard subject",
		"FromEmailAddress":         "sender@example.com",
		"CreateDate":               sesv2PlaceholderTimestamp(),
		"DeliverabilityTestStatus": "COMPLETED",
	}
}

func sesv2DefaultDomainDeliverabilityCampaign(campaignID string) map[string]any {
	return map[string]any{
		"CampaignId":        campaignID,
		"ImageUrl":          "https://example.com/campaign.png",
		"Subject":           "stackyard campaign",
		"FromAddress":       "sender@example.com",
		"SendingIps":        []string{"198.51.100.10"},
		"FirstSeenDateTime": sesv2PlaceholderTimestamp(),
		"LastSeenDateTime":  sesv2PlaceholderTimestamp(),
		"InboxCount":        100,
		"SpamCount":         1,
		"ReadRate":          0.65,
		"DeleteRate":        0.02,
		"ReadDeleteRate":    0.01,
		"ProjectedVolume":   101,
		"Esps":              []string{"gmail.com"},
	}
}

func sesv2DefaultVolumeStatistics() map[string]any {
	return map[string]any{
		"InboxRawCount":  100,
		"SpamRawCount":   1,
		"ProjectedInbox": 101,
		"ProjectedSpam":  1,
	}
}

func sesv2DefaultOverallVolume() map[string]any {
	return map[string]any{
		"VolumeStatistics": sesv2DefaultVolumeStatistics(),
		"ReadRatePercent":  0.65,
		"DomainIspPlacements": []map[string]any{{
			"IspName":         "gmail.com",
			"InboxRawCount":   100,
			"SpamRawCount":    1,
			"InboxPercentage": 99.0,
			"SpamPercentage":  1.0,
		}},
	}
}

func sesv2DefaultExportJob(jobID string) map[string]any {
	return map[string]any{
		"JobId":            jobID,
		"ExportSourceType": "METRICS_DATA",
		"JobStatus":        "COMPLETED",
		"ExportDestination": map[string]any{
			"DataFormat": "CSV",
			"S3Url":      "s3://stackyard-bucket/exports/report.csv",
		},
		"ExportDataSource": map[string]any{
			"MetricsDataSource": map[string]any{
				"Dimensions": map[string]any{"EMAIL_IDENTITY": []string{"sender@example.com"}},
				"Namespace":  "VDM",
				"Metrics": []map[string]any{{
					"Name":        "SEND",
					"Aggregation": "VOLUME",
				}},
				"StartDate": sesv2Timestamp(time.Now().UTC().Add(-24 * time.Hour)),
				"EndDate":   sesv2PlaceholderTimestamp(),
			},
		},
		"CreatedTimestamp":   sesv2Timestamp(time.Now().UTC().Add(-5 * time.Minute)),
		"CompletedTimestamp": sesv2PlaceholderTimestamp(),
		"Statistics": map[string]any{
			"ProcessedRecordsCount": 1,
			"ExportedRecordsCount":  1,
		},
	}
}

func sesv2DefaultExportJobSummary(jobID string) map[string]any {
	job := sesv2DefaultExportJob(jobID)
	return map[string]any{
		"JobId":              job["JobId"],
		"ExportSourceType":   job["ExportSourceType"],
		"JobStatus":          job["JobStatus"],
		"CreatedTimestamp":   job["CreatedTimestamp"],
		"CompletedTimestamp": job["CompletedTimestamp"],
	}
}

func sesv2DefaultImportJob(jobID string) map[string]any {
	return map[string]any{
		"JobId": jobID,
		"ImportDestination": map[string]any{
			"SuppressionListDestination": map[string]any{
				"SuppressionListImportAction": "PUT",
			},
		},
		"ImportDataSource": map[string]any{
			"S3Url":      "s3://stackyard-bucket/imports/contacts.csv",
			"DataFormat": "CSV",
		},
		"JobStatus":             "COMPLETED",
		"CreatedTimestamp":      sesv2Timestamp(time.Now().UTC().Add(-5 * time.Minute)),
		"CompletedTimestamp":    sesv2PlaceholderTimestamp(),
		"ProcessedRecordsCount": 1,
		"FailedRecordsCount":    0,
	}
}

func sesv2DefaultImportJobSummary(jobID string) map[string]any {
	job := sesv2DefaultImportJob(jobID)
	return map[string]any{
		"JobId":                 job["JobId"],
		"ImportDestination":     job["ImportDestination"],
		"JobStatus":             job["JobStatus"],
		"CreatedTimestamp":      job["CreatedTimestamp"],
		"ProcessedRecordsCount": job["ProcessedRecordsCount"],
		"FailedRecordsCount":    job["FailedRecordsCount"],
	}
}

func sesv2DefaultCustomVerificationTemplate(name string) map[string]any {
	return map[string]any{
		"TemplateName":          name,
		"FromEmailAddress":      "sender@example.com",
		"TemplateSubject":       "stackyard verify",
		"TemplateContent":       "stackyard verification content",
		"SuccessRedirectionURL": "https://example.com/success",
		"FailureRedirectionURL": "https://example.com/failure",
	}
}

func sesv2DefaultCustomVerificationTemplateMetadata(name string) map[string]any {
	return map[string]any{
		"TemplateName":          name,
		"FromEmailAddress":      "sender@example.com",
		"TemplateSubject":       "stackyard verify",
		"SuccessRedirectionURL": "https://example.com/success",
		"FailureRedirectionURL": "https://example.com/failure",
	}
}

func sesv2DefaultDedicatedIP(ip, poolName string) map[string]any {
	return map[string]any{
		"Ip":               ip,
		"WarmupStatus":     "DONE",
		"WarmupPercentage": 100,
		"PoolName":         poolName,
	}
}

func sesv2DefaultReputationEntity(entityType, reference string) map[string]any {
	status := map[string]any{
		"Status":               "HEALTHY",
		"Cause":                "stackyard synthetic state",
		"LastUpdatedTimestamp": sesv2PlaceholderTimestamp(),
	}
	return map[string]any{
		"ReputationEntityReference":  reference,
		"ReputationEntityType":       entityType,
		"ReputationManagementPolicy": "arn:aws:ses:us-east-1:123456789012:reputation-policy/default",
		"CustomerManagedStatus":      status,
		"AwsSesManagedStatus":        status,
		"SendingStatusAggregate":     "ENABLED",
		"ReputationImpact":           "LOW",
	}
}

func sesv2Timestamp(t time.Time) float64 {
	if t.IsZero() {
		return 0
	}
	return float64(t.UnixNano()) / float64(time.Second)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
