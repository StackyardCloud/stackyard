package server

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/services/ses"
)

func (s *Server) handleSESQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSESQueryCandidate(r) {
		return false
	}
	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ses")
	if !ok {
		respondSESErrorXML(w, status, code, msg)
		return true
	}
	if err := r.ParseForm(); err != nil {
		respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form")
		return true
	}
	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondSESErrorXML(w, http.StatusBadRequest, "MissingAction", "missing Action")
		return true
	}
	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2010-12-01" {
		respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid Version")
		return true
	}

	switch action {
	case "SendEmail":
		input := ses.SendEmailInput{
			Source:       strings.TrimSpace(r.Form.Get("Source")),
			Destinations: parseSESDestination(r.Form, "Destination"),
			Subject:      r.Form.Get("Message.Subject.Data"),
			TextBody:     r.Form.Get("Message.Body.Text.Data"),
			HTMLBody:     r.Form.Get("Message.Body.Html.Data"),
			Tags:         parseSESTags(r.Form),
		}
		msgID, err := s.ses.SendEmail(input)
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSendEmailResult{MessageID: msgID})
		return true
	case "SendRawEmail":
		raw := strings.TrimSpace(r.Form.Get("RawMessage.Data"))
		if raw == "" {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "RawMessage.Data is required")
			return true
		}
		rawData, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "RawMessage.Data must be base64 encoded")
			return true
		}
		msgID, err := s.ses.SendRawEmail(ses.SendRawEmailInput{
			Source:       strings.TrimSpace(r.Form.Get("Source")),
			Destinations: parseSESStringMembers(r.Form, "Destinations.member"),
			RawData:      rawData,
			Tags:         parseSESTags(r.Form),
		})
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSendRawEmailResult{MessageID: msgID})
		return true
	case "SendTemplatedEmail":
		msgID, err := s.ses.SendTemplatedEmail(ses.SendTemplatedEmailInput{
			Source:       strings.TrimSpace(r.Form.Get("Source")),
			Destinations: parseSESDestination(r.Form, "Destination"),
			TemplateName: strings.TrimSpace(r.Form.Get("Template")),
			TemplateData: r.Form.Get("TemplateData"),
			Tags:         parseSESTags(r.Form),
		})
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSendTemplatedEmailResult{MessageID: msgID})
		return true
	case "CreateTemplate":
		err := s.ses.CreateTemplate(ses.Template{
			Name:     strings.TrimSpace(r.Form.Get("Template.TemplateName")),
			Subject:  r.Form.Get("Template.SubjectPart"),
			HTMLPart: r.Form.Get("Template.HtmlPart"),
			TextPart: r.Form.Get("Template.TextPart"),
		})
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesCreateTemplateResult{})
		return true
	case "UpdateTemplate":
		err := s.ses.UpdateTemplate(ses.Template{
			Name:     strings.TrimSpace(r.Form.Get("Template.TemplateName")),
			Subject:  r.Form.Get("Template.SubjectPart"),
			HTMLPart: r.Form.Get("Template.HtmlPart"),
			TextPart: r.Form.Get("Template.TextPart"),
		})
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesUpdateTemplateResult{})
		return true
	case "DeleteTemplate":
		err := s.ses.DeleteTemplate(strings.TrimSpace(r.Form.Get("TemplateName")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesDeleteTemplateResult{})
		return true
	case "GetTemplate":
		template, err := s.ses.GetTemplate(strings.TrimSpace(r.Form.Get("TemplateName")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesGetTemplateResult{Template: sesTemplate{
			TemplateName: template.Name,
			SubjectPart:  template.Subject,
			HtmlPart:     template.HTMLPart,
			TextPart:     template.TextPart,
		}})
		return true
	case "ListTemplates":
		maxItems, err := parseSESMaxItems(r.Form.Get("MaxItems"), 10)
		if err != nil {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxItems must be a positive integer")
			return true
		}
		items, nextToken, err := s.ses.ListTemplates(r.Form.Get("NextToken"), maxItems)
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		out := make([]sesTemplateMetadataEntry, 0, len(items))
		for _, item := range items {
			out = append(out, sesTemplateMetadataEntry{Name: item.Name, CreatedTimestamp: item.CreatedAt})
		}
		respondSESXML(w, action, sesListTemplatesResult{TemplatesMetadata: out, NextToken: nextToken})
		return true
	case "VerifyEmailIdentity":
		if err := s.ses.VerifyEmailIdentity(strings.TrimSpace(r.Form.Get("EmailAddress"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesVerifyEmailIdentityResult{})
		return true
	case "VerifyEmailAddress":
		if err := s.ses.VerifyEmailAddress(strings.TrimSpace(r.Form.Get("EmailAddress"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesVerifyEmailAddressResult{})
		return true
	case "VerifyDomainIdentity":
		token, err := s.ses.VerifyDomainIdentity(strings.TrimSpace(r.Form.Get("Domain")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesVerifyDomainIdentityResult{VerificationToken: token})
		return true
	case "VerifyDomainDkim":
		tokens, err := s.ses.VerifyDomainDkim(strings.TrimSpace(r.Form.Get("Domain")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesVerifyDomainDkimResult{DkimTokens: tokens})
		return true
	case "ListIdentities":
		maxItems, err := parseSESMaxItems(r.Form.Get("MaxItems"), 100)
		if err != nil {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxItems must be a positive integer")
			return true
		}
		identities, nextToken, err := s.ses.ListIdentities(strings.TrimSpace(r.Form.Get("IdentityType")), r.Form.Get("NextToken"), maxItems)
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesListIdentitiesResult{Identities: identities, NextToken: nextToken})
		return true
	case "ListVerifiedEmailAddresses":
		emails := s.ses.ListVerifiedEmailAddresses()
		respondSESXML(w, action, sesListVerifiedEmailAddressesResult{VerifiedEmailAddresses: emails})
		return true
	case "GetIdentityVerificationAttributes":
		idents := parseSESStringMembers(r.Form, "Identities.member")
		if len(idents) == 0 {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Identities are required")
			return true
		}
		attrs := s.ses.GetIdentityVerificationAttributes(idents)
		entries := make([]sesVerificationEntry, 0, len(attrs))
		for _, key := range sortedStringKeysFromMap(attrs) {
			attr := attrs[key]
			entries = append(entries, sesVerificationEntry{
				Key: key,
				Value: sesVerificationValue{
					VerificationStatus: attr.VerificationStatus,
					VerificationToken:  attr.VerificationToken,
				},
			})
		}
		respondSESXML(w, action, sesGetIdentityVerificationAttributesResult{VerificationAttributes: entries})
		return true
	case "GetIdentityDkimAttributes":
		idents := parseSESStringMembers(r.Form, "Identities.member")
		if len(idents) == 0 {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Identities are required")
			return true
		}
		attrs := s.ses.GetIdentityDkimAttributes(idents)
		entries := make([]sesDkimEntry, 0, len(attrs))
		for _, key := range sortedStringKeysFromMap(attrs) {
			attr := attrs[key]
			entries = append(entries, sesDkimEntry{
				Key: key,
				Value: sesDkimValue{
					DkimEnabled:            attr.DkimEnabled,
					DkimTokens:             attr.DkimTokens,
					DkimVerificationStatus: attr.VerificationState,
				},
			})
		}
		respondSESXML(w, action, sesGetIdentityDkimAttributesResult{DkimAttributes: entries})
		return true
	case "GetIdentityMailFromDomainAttributes":
		idents := parseSESStringMembers(r.Form, "Identities.member")
		if len(idents) == 0 {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Identities are required")
			return true
		}
		attrs := s.ses.GetIdentityMailFromDomainAttributes(idents)
		entries := make([]sesMailFromEntry, 0, len(attrs))
		for _, key := range sortedStringKeysFromMap(attrs) {
			attr := attrs[key]
			entries = append(entries, sesMailFromEntry{
				Key: key,
				Value: sesMailFromValue{
					BehaviorOnMXFailure:  attr.BehaviorOnMXFailure,
					MailFromDomain:       attr.MailFromDomain,
					MailFromDomainStatus: attr.MailFromDomainState,
				},
			})
		}
		respondSESXML(w, action, sesGetIdentityMailFromDomainAttributesResult{MailFromDomainAttributes: entries})
		return true
	case "GetIdentityNotificationAttributes":
		idents := parseSESStringMembers(r.Form, "Identities.member")
		if len(idents) == 0 {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Identities are required")
			return true
		}
		attrs := s.ses.GetIdentityNotificationAttributes(idents)
		entries := make([]sesNotificationEntry, 0, len(attrs))
		for _, key := range sortedStringKeysFromMap(attrs) {
			attr := attrs[key]
			entries = append(entries, sesNotificationEntry{
				Key: key,
				Value: sesNotificationValue{
					BounceTopic:                            attr.BounceTopic,
					ComplaintTopic:                         attr.ComplaintTopic,
					DeliveryTopic:                          attr.DeliveryTopic,
					ForwardingEnabled:                      attr.ForwardingEnabled,
					HeadersInBounceNotificationsEnabled:    attr.HeadersInBounceNotifications,
					HeadersInComplaintNotificationsEnabled: attr.HeadersInComplaintNotifications,
					HeadersInDeliveryNotificationsEnabled:  attr.HeadersInDeliveryNotifications,
				},
			})
		}
		respondSESXML(w, action, sesGetIdentityNotificationAttributesResult{NotificationAttributes: entries})
		return true
	case "DeleteIdentity":
		s.ses.DeleteIdentity(strings.TrimSpace(r.Form.Get("Identity")))
		respondSESXML(w, action, sesDeleteIdentityResult{})
		return true
	case "DeleteVerifiedEmailAddress":
		s.ses.DeleteVerifiedEmailAddress(strings.TrimSpace(r.Form.Get("EmailAddress")))
		respondSESXML(w, action, sesDeleteVerifiedEmailAddressResult{})
		return true
	case "SetIdentityDkimEnabled":
		identity := strings.TrimSpace(r.Form.Get("Identity"))
		enabled, err := parseSESRequiredBool(r.Form.Get("DkimEnabled"))
		if err != nil || identity == "" {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Identity and DkimEnabled are required")
			return true
		}
		if err := s.ses.SetIdentityDkimEnabled(identity, enabled); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSetIdentityDkimEnabledResult{})
		return true
	case "SetIdentityFeedbackForwardingEnabled":
		identity := strings.TrimSpace(r.Form.Get("Identity"))
		enabled, err := parseSESRequiredBool(r.Form.Get("ForwardingEnabled"))
		if err != nil || identity == "" {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Identity and ForwardingEnabled are required")
			return true
		}
		if err := s.ses.SetIdentityFeedbackForwardingEnabled(identity, enabled); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSetIdentityFeedbackForwardingEnabledResult{})
		return true
	case "SetIdentityHeadersInNotificationsEnabled":
		identity := strings.TrimSpace(r.Form.Get("Identity"))
		notificationType := strings.TrimSpace(r.Form.Get("NotificationType"))
		enabled, err := parseSESRequiredBool(r.Form.Get("Enabled"))
		if err != nil || identity == "" || notificationType == "" {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Identity, NotificationType and Enabled are required")
			return true
		}
		if err := s.ses.SetIdentityHeadersInNotificationsEnabled(identity, notificationType, enabled); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSetIdentityHeadersInNotificationsEnabledResult{})
		return true
	case "SetIdentityMailFromDomain":
		identity := strings.TrimSpace(r.Form.Get("Identity"))
		if identity == "" {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Identity is required")
			return true
		}
		if err := s.ses.SetIdentityMailFromDomain(identity, r.Form.Get("MailFromDomain"), strings.TrimSpace(r.Form.Get("BehaviorOnMXFailure"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSetIdentityMailFromDomainResult{})
		return true
	case "SetIdentityNotificationTopic":
		identity := strings.TrimSpace(r.Form.Get("Identity"))
		notificationType := strings.TrimSpace(r.Form.Get("NotificationType"))
		if identity == "" || notificationType == "" {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Identity and NotificationType are required")
			return true
		}
		if err := s.ses.SetIdentityNotificationTopic(identity, notificationType, r.Form.Get("SnsTopic")); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSetIdentityNotificationTopicResult{})
		return true
	case "GetAccountSendingEnabled":
		respondSESXML(w, action, sesGetAccountSendingEnabledResult{Enabled: s.ses.GetAccountSendingEnabled()})
		return true
	case "UpdateAccountSendingEnabled":
		enabled, err := parseSESOptionalBool(r.Form.Get("Enabled"), false)
		if err != nil {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Enabled must be a boolean")
			return true
		}
		s.ses.UpdateAccountSendingEnabled(enabled)
		respondSESXML(w, action, sesUpdateAccountSendingEnabledResult{})
		return true
	case "GetSendQuota":
		quota := s.ses.GetSendQuota()
		respondSESXML(w, action, sesGetSendQuotaResult{
			Max24HourSend:   quota.Max24HourSend,
			MaxSendRate:     quota.MaxSendRate,
			SentLast24Hours: quota.SentLast24Hours,
		})
		return true
	case "GetSendStatistics":
		stats := s.ses.GetSendStatistics()
		points := make([]sesSendDataPoint, 0, len(stats))
		for _, point := range stats {
			points = append(points, sesSendDataPoint{
				Timestamp:        point.Timestamp,
				DeliveryAttempts: point.DeliveryAttempts,
				Rejects:          point.Rejects,
				Bounces:          point.Bounces,
				Complaints:       point.Complaints,
			})
		}
		respondSESXML(w, action, sesGetSendStatisticsResult{SendDataPoints: points})
		return true
	default:
		if s.handleSESExtendedAction(w, r, action) {
			return true
		}
		respondSESErrorXML(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
		return true
	}
}

func respondSESErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ses.ErrTemplateNotFound):
		respondSESErrorXML(w, http.StatusBadRequest, "TemplateDoesNotExist", "template does not exist")
	case errors.Is(err, ses.ErrTemplateExists):
		respondSESErrorXML(w, http.StatusBadRequest, "AlreadyExists", "template already exists")
	case errors.Is(err, ses.ErrConfigurationSetExists),
		errors.Is(err, ses.ErrEventDestinationExists),
		errors.Is(err, ses.ErrCustomVerificationTemplateExists),
		errors.Is(err, ses.ErrReceiptFilterExists),
		errors.Is(err, ses.ErrReceiptRuleSetExists),
		errors.Is(err, ses.ErrReceiptRuleExists):
		respondSESErrorXML(w, http.StatusBadRequest, "AlreadyExists", "resource already exists")
	case errors.Is(err, ses.ErrConfigurationSetNotFound),
		errors.Is(err, ses.ErrEventDestinationNotFound),
		errors.Is(err, ses.ErrCustomVerificationTemplateNotFound),
		errors.Is(err, ses.ErrReceiptFilterNotFound),
		errors.Is(err, ses.ErrReceiptRuleSetNotFound),
		errors.Is(err, ses.ErrReceiptRuleNotFound),
		errors.Is(err, ses.ErrIdentityPolicyNotFound):
		respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "resource not found")
	case errors.Is(err, ses.ErrIdentityNotFound):
		respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "identity not found")
	case errors.Is(err, ses.ErrMessageRejected):
		respondSESErrorXML(w, http.StatusBadRequest, "MessageRejected", "message rejected")
	case errors.Is(err, ses.ErrInvalidParameter):
		respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid parameter")
	default:
		respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	}
}

func isSESQueryCandidate(r *http.Request) bool {
	action := r.URL.Query().Get("Action")
	if action != "" {
		return isSESAction(action)
	}
	if r.Method != http.MethodPost {
		return false
	}
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "application/x-www-form-urlencoded") {
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
	return isSESAction(action)
}

var sesActions = map[string]struct{}{
	"CloneReceiptRuleSet":                            {},
	"CreateConfigurationSet":                         {},
	"CreateConfigurationSetEventDestination":         {},
	"CreateConfigurationSetTrackingOptions":          {},
	"CreateCustomVerificationEmailTemplate":          {},
	"CreateReceiptFilter":                            {},
	"CreateReceiptRule":                              {},
	"CreateReceiptRuleSet":                           {},
	"CreateTemplate":                                 {},
	"DeleteConfigurationSet":                         {},
	"DeleteConfigurationSetEventDestination":         {},
	"DeleteConfigurationSetTrackingOptions":          {},
	"DeleteCustomVerificationEmailTemplate":          {},
	"DeleteIdentity":                                 {},
	"DeleteIdentityPolicy":                           {},
	"DeleteReceiptFilter":                            {},
	"DeleteReceiptRule":                              {},
	"DeleteReceiptRuleSet":                           {},
	"DeleteTemplate":                                 {},
	"DeleteVerifiedEmailAddress":                     {},
	"DescribeActiveReceiptRuleSet":                   {},
	"DescribeConfigurationSet":                       {},
	"DescribeReceiptRule":                            {},
	"DescribeReceiptRuleSet":                         {},
	"GetAccountSendingEnabled":                       {},
	"GetCustomVerificationEmailTemplate":             {},
	"GetIdentityDkimAttributes":                      {},
	"GetIdentityMailFromDomainAttributes":            {},
	"GetIdentityNotificationAttributes":              {},
	"GetIdentityPolicies":                            {},
	"GetIdentityVerificationAttributes":              {},
	"GetSendQuota":                                   {},
	"GetSendStatistics":                              {},
	"GetTemplate":                                    {},
	"ListConfigurationSets":                          {},
	"ListCustomVerificationEmailTemplates":           {},
	"ListIdentities":                                 {},
	"ListIdentityPolicies":                           {},
	"ListReceiptFilters":                             {},
	"ListReceiptRuleSets":                            {},
	"ListTemplates":                                  {},
	"ListVerifiedEmailAddresses":                     {},
	"PutConfigurationSetDeliveryOptions":             {},
	"PutIdentityPolicy":                              {},
	"ReorderReceiptRuleSet":                          {},
	"SendBounce":                                     {},
	"SendBulkTemplatedEmail":                         {},
	"SendCustomVerificationEmail":                    {},
	"SendEmail":                                      {},
	"SendRawEmail":                                   {},
	"SendTemplatedEmail":                             {},
	"SetActiveReceiptRuleSet":                        {},
	"SetIdentityDkimEnabled":                         {},
	"SetIdentityFeedbackForwardingEnabled":           {},
	"SetIdentityHeadersInNotificationsEnabled":       {},
	"SetIdentityMailFromDomain":                      {},
	"SetIdentityNotificationTopic":                   {},
	"SetReceiptRulePosition":                         {},
	"TestRenderTemplate":                             {},
	"UpdateAccountSendingEnabled":                    {},
	"UpdateConfigurationSetEventDestination":         {},
	"UpdateConfigurationSetReputationMetricsEnabled": {},
	"UpdateConfigurationSetSendingEnabled":           {},
	"UpdateConfigurationSetTrackingOptions":          {},
	"UpdateCustomVerificationEmailTemplate":          {},
	"UpdateReceiptRule":                              {},
	"UpdateTemplate":                                 {},
	"VerifyDomainDkim":                               {},
	"VerifyDomainIdentity":                           {},
	"VerifyEmailAddress":                             {},
	"VerifyEmailIdentity":                            {},
}

func isSESAction(action string) bool {
	_, ok := sesActions[action]
	return ok
}

func respondSESXML(w http.ResponseWriter, action string, result any) {
	env := sesResponseEnvelope{
		XMLName: xml.Name{Local: action + "Response"},
		Xmlns:   sesNamespace,
		Result:  result,
		Metadata: sesResponseMetadata{
			RequestID: "stackyard-request",
		},
	}
	respondXML(w, http.StatusOK, env)
}

func respondSESErrorXML(w http.ResponseWriter, status int, code, message string) {
	payload := sesErrorResponse{
		Xmlns: sesNamespace,
		Error: sesErrorBody{
			Type:    "Sender",
			Code:    code,
			Message: message,
		},
		RequestID: "stackyard-request",
	}
	respondXML(w, status, payload)
}

func parseSESDestination(values url.Values, prefix string) []string {
	out := make([]string, 0)
	out = append(out, parseSESStringMembers(values, prefix+".ToAddresses.member")...)
	out = append(out, parseSESStringMembers(values, prefix+".CcAddresses.member")...)
	out = append(out, parseSESStringMembers(values, prefix+".BccAddresses.member")...)
	return out
}

func parseSESStringMembers(values url.Values, prefix string) []string {
	type item struct {
		index int
		value string
	}
	items := make([]item, 0)
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix+".") {
			continue
		}
		if len(vals) == 0 {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+".")
		parts := strings.Split(rest, ".")
		if len(parts) == 0 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		value := strings.TrimSpace(vals[0])
		if value == "" {
			continue
		}
		items = append(items, item{index: idx, value: value})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].index < items[j].index })
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.value)
	}
	return out
}

func parseSESTags(values url.Values) map[string]string {
	type tag struct {
		name  string
		value string
	}
	items := map[string]*tag{}
	for key, vals := range values {
		if !strings.HasPrefix(key, "Tags.member.") || len(vals) == 0 {
			continue
		}
		rest := strings.TrimPrefix(key, "Tags.member.")
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx := parts[0]
		field := parts[1]
		entry := items[idx]
		if entry == nil {
			entry = &tag{}
			items[idx] = entry
		}
		switch field {
		case "Name":
			entry.name = strings.TrimSpace(vals[0])
		case "Value":
			entry.value = vals[0]
		}
	}
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]string)
	for _, item := range items {
		if item.name == "" {
			continue
		}
		out[item.name] = item.value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseSESMaxItems(raw string, defaultValue int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, errors.New("must be positive")
	}
	return value, nil
}

func parseSESRequiredBool(raw string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, errors.New("missing bool")
	}
	return strconv.ParseBool(raw)
}

func parseSESOptionalBool(raw string, defaultValue bool) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	return strconv.ParseBool(raw)
}

func sortedStringKeysFromMap[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type sesResponseEnvelope struct {
	XMLName  xml.Name            `xml:""`
	Xmlns    string              `xml:"xmlns,attr,omitempty"`
	Result   any                 `xml:",any"`
	Metadata sesResponseMetadata `xml:"ResponseMetadata"`
}

type sesResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type sesErrorResponse struct {
	XMLName   xml.Name     `xml:"ErrorResponse"`
	Xmlns     string       `xml:"xmlns,attr,omitempty"`
	Error     sesErrorBody `xml:"Error"`
	RequestID string       `xml:"RequestId"`
}

type sesErrorBody struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type sesSendEmailResult struct {
	XMLName   xml.Name `xml:"SendEmailResult"`
	MessageID string   `xml:"MessageId"`
}

type sesSendRawEmailResult struct {
	XMLName   xml.Name `xml:"SendRawEmailResult"`
	MessageID string   `xml:"MessageId"`
}

type sesSendTemplatedEmailResult struct {
	XMLName   xml.Name `xml:"SendTemplatedEmailResult"`
	MessageID string   `xml:"MessageId"`
}

type sesCreateTemplateResult struct {
	XMLName xml.Name `xml:"CreateTemplateResult"`
}

type sesUpdateTemplateResult struct {
	XMLName xml.Name `xml:"UpdateTemplateResult"`
}

type sesDeleteTemplateResult struct {
	XMLName xml.Name `xml:"DeleteTemplateResult"`
}

type sesGetTemplateResult struct {
	XMLName  xml.Name    `xml:"GetTemplateResult"`
	Template sesTemplate `xml:"Template"`
}

type sesTemplate struct {
	TemplateName string `xml:"TemplateName"`
	SubjectPart  string `xml:"SubjectPart,omitempty"`
	HtmlPart     string `xml:"HtmlPart,omitempty"`
	TextPart     string `xml:"TextPart,omitempty"`
}

type sesListTemplatesResult struct {
	XMLName           xml.Name                   `xml:"ListTemplatesResult"`
	TemplatesMetadata []sesTemplateMetadataEntry `xml:"TemplatesMetadata>member"`
	NextToken         string                     `xml:"NextToken,omitempty"`
}

type sesTemplateMetadataEntry struct {
	Name             string    `xml:"Name"`
	CreatedTimestamp time.Time `xml:"CreatedTimestamp"`
}

type sesVerifyEmailIdentityResult struct {
	XMLName xml.Name `xml:"VerifyEmailIdentityResult"`
}

type sesVerifyEmailAddressResult struct {
	XMLName xml.Name `xml:"VerifyEmailAddressResult"`
}

type sesVerifyDomainIdentityResult struct {
	XMLName           xml.Name `xml:"VerifyDomainIdentityResult"`
	VerificationToken string   `xml:"VerificationToken"`
}

type sesVerifyDomainDkimResult struct {
	XMLName    xml.Name `xml:"VerifyDomainDkimResult"`
	DkimTokens []string `xml:"DkimTokens>member"`
}

type sesListIdentitiesResult struct {
	XMLName    xml.Name `xml:"ListIdentitiesResult"`
	Identities []string `xml:"Identities>member"`
	NextToken  string   `xml:"NextToken,omitempty"`
}

type sesListVerifiedEmailAddressesResult struct {
	XMLName                xml.Name `xml:"ListVerifiedEmailAddressesResult"`
	VerifiedEmailAddresses []string `xml:"VerifiedEmailAddresses>member"`
}

type sesGetIdentityVerificationAttributesResult struct {
	XMLName                xml.Name               `xml:"GetIdentityVerificationAttributesResult"`
	VerificationAttributes []sesVerificationEntry `xml:"VerificationAttributes>entry"`
}

type sesVerificationEntry struct {
	Key   string               `xml:"key"`
	Value sesVerificationValue `xml:"value"`
}

type sesVerificationValue struct {
	VerificationStatus string `xml:"VerificationStatus"`
	VerificationToken  string `xml:"VerificationToken,omitempty"`
}

type sesGetIdentityDkimAttributesResult struct {
	XMLName        xml.Name       `xml:"GetIdentityDkimAttributesResult"`
	DkimAttributes []sesDkimEntry `xml:"DkimAttributes>entry"`
}

type sesDkimEntry struct {
	Key   string       `xml:"key"`
	Value sesDkimValue `xml:"value"`
}

type sesDkimValue struct {
	DkimEnabled            bool     `xml:"DkimEnabled"`
	DkimTokens             []string `xml:"DkimTokens>member,omitempty"`
	DkimVerificationStatus string   `xml:"DkimVerificationStatus"`
}

type sesGetIdentityMailFromDomainAttributesResult struct {
	XMLName                  xml.Name           `xml:"GetIdentityMailFromDomainAttributesResult"`
	MailFromDomainAttributes []sesMailFromEntry `xml:"MailFromDomainAttributes>entry"`
}

type sesMailFromEntry struct {
	Key   string           `xml:"key"`
	Value sesMailFromValue `xml:"value"`
}

type sesMailFromValue struct {
	BehaviorOnMXFailure  string `xml:"BehaviorOnMXFailure"`
	MailFromDomain       string `xml:"MailFromDomain,omitempty"`
	MailFromDomainStatus string `xml:"MailFromDomainStatus"`
}

type sesGetIdentityNotificationAttributesResult struct {
	XMLName                xml.Name               `xml:"GetIdentityNotificationAttributesResult"`
	NotificationAttributes []sesNotificationEntry `xml:"NotificationAttributes>entry"`
}

type sesNotificationEntry struct {
	Key   string               `xml:"key"`
	Value sesNotificationValue `xml:"value"`
}

type sesNotificationValue struct {
	BounceTopic                            string `xml:"BounceTopic,omitempty"`
	ComplaintTopic                         string `xml:"ComplaintTopic,omitempty"`
	DeliveryTopic                          string `xml:"DeliveryTopic,omitempty"`
	ForwardingEnabled                      bool   `xml:"ForwardingEnabled"`
	HeadersInBounceNotificationsEnabled    bool   `xml:"HeadersInBounceNotificationsEnabled"`
	HeadersInComplaintNotificationsEnabled bool   `xml:"HeadersInComplaintNotificationsEnabled"`
	HeadersInDeliveryNotificationsEnabled  bool   `xml:"HeadersInDeliveryNotificationsEnabled"`
}

type sesDeleteIdentityResult struct {
	XMLName xml.Name `xml:"DeleteIdentityResult"`
}

type sesDeleteVerifiedEmailAddressResult struct {
	XMLName xml.Name `xml:"DeleteVerifiedEmailAddressResult"`
}

type sesSetIdentityDkimEnabledResult struct {
	XMLName xml.Name `xml:"SetIdentityDkimEnabledResult"`
}

type sesSetIdentityFeedbackForwardingEnabledResult struct {
	XMLName xml.Name `xml:"SetIdentityFeedbackForwardingEnabledResult"`
}

type sesSetIdentityHeadersInNotificationsEnabledResult struct {
	XMLName xml.Name `xml:"SetIdentityHeadersInNotificationsEnabledResult"`
}

type sesSetIdentityMailFromDomainResult struct {
	XMLName xml.Name `xml:"SetIdentityMailFromDomainResult"`
}

type sesSetIdentityNotificationTopicResult struct {
	XMLName xml.Name `xml:"SetIdentityNotificationTopicResult"`
}

type sesGetAccountSendingEnabledResult struct {
	XMLName xml.Name `xml:"GetAccountSendingEnabledResult"`
	Enabled bool     `xml:"Enabled"`
}

type sesUpdateAccountSendingEnabledResult struct {
	XMLName xml.Name `xml:"UpdateAccountSendingEnabledResult"`
}

type sesGetSendQuotaResult struct {
	XMLName         xml.Name `xml:"GetSendQuotaResult"`
	Max24HourSend   float64  `xml:"Max24HourSend"`
	MaxSendRate     float64  `xml:"MaxSendRate"`
	SentLast24Hours float64  `xml:"SentLast24Hours"`
}

type sesGetSendStatisticsResult struct {
	XMLName        xml.Name           `xml:"GetSendStatisticsResult"`
	SendDataPoints []sesSendDataPoint `xml:"SendDataPoints>member"`
}

type sesSendDataPoint struct {
	Timestamp        time.Time `xml:"Timestamp"`
	DeliveryAttempts int64     `xml:"DeliveryAttempts"`
	Rejects          int64     `xml:"Rejects"`
	Bounces          int64     `xml:"Bounces"`
	Complaints       int64     `xml:"Complaints"`
}
