package server

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/services/ses"
)

func (s *Server) handleSESExtendedAction(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CloneReceiptRuleSet":
		if err := s.ses.CloneReceiptRuleSet(strings.TrimSpace(r.Form.Get("OriginalRuleSetName")), strings.TrimSpace(r.Form.Get("RuleSetName"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "CreateConfigurationSet":
		name := strings.TrimSpace(r.Form.Get("ConfigurationSet.Name"))
		if name == "" {
			name = strings.TrimSpace(r.Form.Get("ConfigurationSetName"))
		}
		if err := s.ses.CreateConfigurationSet(name); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "CreateConfigurationSetEventDestination":
		destination, err := parseSESEventDestinationInput(r.Form, "EventDestination")
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		if err := s.ses.CreateConfigurationSetEventDestination(strings.TrimSpace(r.Form.Get("ConfigurationSetName")), destination); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "CreateConfigurationSetTrackingOptions":
		if err := s.ses.CreateConfigurationSetTrackingOptions(strings.TrimSpace(r.Form.Get("ConfigurationSetName")), strings.TrimSpace(r.Form.Get("TrackingOptions.CustomRedirectDomain"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "CreateCustomVerificationEmailTemplate":
		err := s.ses.CreateCustomVerificationEmailTemplate(ses.CustomVerificationEmailTemplate{
			TemplateName:          strings.TrimSpace(r.Form.Get("TemplateName")),
			FromEmailAddress:      strings.TrimSpace(r.Form.Get("FromEmailAddress")),
			TemplateSubject:       r.Form.Get("TemplateSubject"),
			TemplateContent:       r.Form.Get("TemplateContent"),
			SuccessRedirectionURL: r.Form.Get("SuccessRedirectionURL"),
			FailureRedirectionURL: r.Form.Get("FailureRedirectionURL"),
		})
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "CreateReceiptFilter":
		err := s.ses.CreateReceiptFilter(ses.ReceiptFilter{
			Name:   strings.TrimSpace(r.Form.Get("Filter.Name")),
			Policy: strings.TrimSpace(r.Form.Get("Filter.IpFilter.Policy")),
			CIDR:   strings.TrimSpace(r.Form.Get("Filter.IpFilter.Cidr")),
		})
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "CreateReceiptRule":
		rule, err := parseSESReceiptRuleInput(r.Form, "Rule")
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		if err := s.ses.CreateReceiptRule(strings.TrimSpace(r.Form.Get("RuleSetName")), strings.TrimSpace(r.Form.Get("After")), rule); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "CreateReceiptRuleSet":
		if err := s.ses.CreateReceiptRuleSet(strings.TrimSpace(r.Form.Get("RuleSetName"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "DeleteConfigurationSet":
		s.ses.DeleteConfigurationSet(strings.TrimSpace(r.Form.Get("ConfigurationSetName")))
		respondSESEmptyResult(w, action)
		return true
	case "DeleteConfigurationSetEventDestination":
		if err := s.ses.DeleteConfigurationSetEventDestination(strings.TrimSpace(r.Form.Get("ConfigurationSetName")), strings.TrimSpace(r.Form.Get("EventDestinationName"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "DeleteConfigurationSetTrackingOptions":
		if err := s.ses.DeleteConfigurationSetTrackingOptions(strings.TrimSpace(r.Form.Get("ConfigurationSetName"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "DeleteCustomVerificationEmailTemplate":
		s.ses.DeleteCustomVerificationEmailTemplate(strings.TrimSpace(r.Form.Get("TemplateName")))
		respondSESEmptyResult(w, action)
		return true
	case "DeleteIdentityPolicy":
		if err := s.ses.DeleteIdentityPolicy(strings.TrimSpace(r.Form.Get("Identity")), strings.TrimSpace(r.Form.Get("PolicyName"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "DeleteReceiptFilter":
		if err := s.ses.DeleteReceiptFilter(strings.TrimSpace(r.Form.Get("FilterName"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "DeleteReceiptRule":
		if err := s.ses.DeleteReceiptRule(strings.TrimSpace(r.Form.Get("RuleSetName")), strings.TrimSpace(r.Form.Get("RuleName"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "DeleteReceiptRuleSet":
		if err := s.ses.DeleteReceiptRuleSet(strings.TrimSpace(r.Form.Get("RuleSetName"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "DescribeActiveReceiptRuleSet":
		ruleSet, err := s.ses.DescribeActiveReceiptRuleSet()
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		if ruleSet == nil {
			respondSESXML(w, action, sesDescribeActiveReceiptRuleSetResult{})
			return true
		}
		respondSESXML(w, action, sesDescribeActiveReceiptRuleSetResult{
			Metadata: &sesReceiptRuleSetMetadata{Name: ruleSet.Name, CreatedTimestamp: ruleSet.CreatedAt},
			Rules:    toSESReceiptRules(ruleSet.Rules),
		})
		return true
	case "DescribeConfigurationSet":
		cfg, err := s.ses.DescribeConfigurationSet(strings.TrimSpace(r.Form.Get("ConfigurationSetName")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesDescribeConfigurationSetResult{
			ConfigurationSet:  sesConfigurationSetMetadata{Name: cfg.Name},
			EventDestinations: toSESEventDestinations(cfg.EventDestinations),
			TrackingOptions:   sesTrackingOptions{CustomRedirectDomain: cfg.TrackingDomain},
			ReputationOptions: sesReputationOptions{ReputationMetricsEnabled: cfg.ReputationMetricsEnabled},
			SendingOptions:    sesSendingOptions{SendingEnabled: cfg.SendingEnabled},
			DeliveryOptions:   sesDeliveryOptions{TLSPolicy: cfg.TLSPolicy},
		})
		return true
	case "DescribeReceiptRule":
		rule, err := s.ses.DescribeReceiptRule(strings.TrimSpace(r.Form.Get("RuleSetName")), strings.TrimSpace(r.Form.Get("RuleName")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesDescribeReceiptRuleResult{Rule: toSESReceiptRule(rule)})
		return true
	case "DescribeReceiptRuleSet":
		ruleSet, err := s.ses.DescribeReceiptRuleSet(strings.TrimSpace(r.Form.Get("RuleSetName")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesDescribeReceiptRuleSetResult{
			Metadata: sesReceiptRuleSetMetadata{Name: ruleSet.Name, CreatedTimestamp: ruleSet.CreatedAt},
			Rules:    toSESReceiptRules(ruleSet.Rules),
		})
		return true
	case "GetCustomVerificationEmailTemplate":
		tpl, err := s.ses.GetCustomVerificationEmailTemplate(strings.TrimSpace(r.Form.Get("TemplateName")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesGetCustomVerificationEmailTemplateResult{
			TemplateName:          tpl.TemplateName,
			FromEmailAddress:      tpl.FromEmailAddress,
			TemplateSubject:       tpl.TemplateSubject,
			TemplateContent:       tpl.TemplateContent,
			SuccessRedirectionURL: tpl.SuccessRedirectionURL,
			FailureRedirectionURL: tpl.FailureRedirectionURL,
		})
		return true
	case "GetIdentityPolicies":
		policies, err := s.ses.GetIdentityPolicies(strings.TrimSpace(r.Form.Get("Identity")), parseSESStringMembers(r.Form, "PolicyNames.member"))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		entries := make([]sesIdentityPolicyEntry, 0, len(policies))
		for _, key := range sortedStringKeysFromMap(policies) {
			entries = append(entries, sesIdentityPolicyEntry{Key: key, Value: policies[key]})
		}
		respondSESXML(w, action, sesGetIdentityPoliciesResult{Policies: entries})
		return true
	case "ListConfigurationSets":
		maxItems, err := parseSESMaxItems(r.Form.Get("MaxItems"), 100)
		if err != nil {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxItems must be a positive integer")
			return true
		}
		sets, nextToken, err := s.ses.ListConfigurationSets(r.Form.Get("NextToken"), maxItems)
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		items := make([]sesConfigurationSetMetadata, 0, len(sets))
		for _, item := range sets {
			items = append(items, sesConfigurationSetMetadata{Name: item.Name})
		}
		respondSESXML(w, action, sesListConfigurationSetsResult{ConfigurationSets: items, NextToken: nextToken})
		return true
	case "ListCustomVerificationEmailTemplates":
		maxItems, err := parseSESMaxItems(r.Form.Get("MaxResults"), 10)
		if err != nil {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxResults must be a positive integer")
			return true
		}
		templates, nextToken, err := s.ses.ListCustomVerificationEmailTemplates(r.Form.Get("NextToken"), maxItems)
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		items := make([]sesCustomVerificationTemplateMetadata, 0, len(templates))
		for _, tpl := range templates {
			items = append(items, sesCustomVerificationTemplateMetadata{
				TemplateName:          tpl.TemplateName,
				FromEmailAddress:      tpl.FromEmailAddress,
				TemplateSubject:       tpl.TemplateSubject,
				SuccessRedirectionURL: tpl.SuccessRedirectionURL,
				FailureRedirectionURL: tpl.FailureRedirectionURL,
			})
		}
		respondSESXML(w, action, sesListCustomVerificationEmailTemplatesResult{Templates: items, NextToken: nextToken})
		return true
	case "ListIdentityPolicies":
		policyNames, err := s.ses.ListIdentityPolicies(strings.TrimSpace(r.Form.Get("Identity")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesListIdentityPoliciesResult{PolicyNames: policyNames})
		return true
	case "ListReceiptFilters":
		filters := s.ses.ListReceiptFilters()
		items := make([]sesReceiptFilterXML, 0, len(filters))
		for _, filter := range filters {
			items = append(items, sesReceiptFilterXML{
				Name: filter.Name,
				IPFilter: sesReceiptIPFilterXML{
					Policy: filter.Policy,
					CIDR:   filter.CIDR,
				},
			})
		}
		respondSESXML(w, action, sesListReceiptFiltersResult{Filters: items})
		return true
	case "ListReceiptRuleSets":
		maxItems, err := parseSESMaxItems(r.Form.Get("MaxItems"), 100)
		if err != nil {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxItems must be a positive integer")
			return true
		}
		ruleSets, nextToken, err := s.ses.ListReceiptRuleSets(r.Form.Get("NextToken"), maxItems)
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		items := make([]sesReceiptRuleSetMetadata, 0, len(ruleSets))
		for _, ruleSet := range ruleSets {
			items = append(items, sesReceiptRuleSetMetadata{Name: ruleSet.Name, CreatedTimestamp: ruleSet.CreatedAt})
		}
		respondSESXML(w, action, sesListReceiptRuleSetsResult{RuleSets: items, NextToken: nextToken})
		return true
	case "PutConfigurationSetDeliveryOptions":
		if err := s.ses.PutConfigurationSetDeliveryOptions(strings.TrimSpace(r.Form.Get("ConfigurationSetName")), strings.TrimSpace(r.Form.Get("TlsPolicy"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "PutIdentityPolicy":
		if err := s.ses.PutIdentityPolicy(strings.TrimSpace(r.Form.Get("Identity")), strings.TrimSpace(r.Form.Get("PolicyName")), r.Form.Get("Policy")); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "ReorderReceiptRuleSet":
		if err := s.ses.ReorderReceiptRuleSet(strings.TrimSpace(r.Form.Get("RuleSetName")), parseSESStringMembers(r.Form, "RuleNames.member")); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "SendBounce":
		messageID, err := s.ses.SendBounce(strings.TrimSpace(r.Form.Get("OriginalMessageId")), strings.TrimSpace(r.Form.Get("BounceSender")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSendBounceResult{MessageID: messageID})
		return true
	case "SendBulkTemplatedEmail":
		statuses, err := s.ses.SendBulkTemplatedEmail(strings.TrimSpace(r.Form.Get("Source")), strings.TrimSpace(r.Form.Get("Template")), parseSESBulkTemplatedDestinations(r.Form))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		items := make([]sesBulkTemplatedEmailStatus, 0, len(statuses))
		for _, status := range statuses {
			items = append(items, sesBulkTemplatedEmailStatus{Status: status.Status, MessageID: status.MessageID, Error: status.Error})
		}
		respondSESXML(w, action, sesSendBulkTemplatedEmailResult{Status: items})
		return true
	case "SendCustomVerificationEmail":
		messageID, err := s.ses.SendCustomVerificationEmail(strings.TrimSpace(r.Form.Get("TemplateName")), strings.TrimSpace(r.Form.Get("EmailAddress")))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesSendCustomVerificationEmailResult{MessageID: messageID})
		return true
	case "SetActiveReceiptRuleSet":
		if err := s.ses.SetActiveReceiptRuleSet(strings.TrimSpace(r.Form.Get("RuleSetName"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "SetReceiptRulePosition":
		if err := s.ses.SetReceiptRulePosition(strings.TrimSpace(r.Form.Get("RuleSetName")), strings.TrimSpace(r.Form.Get("RuleName")), strings.TrimSpace(r.Form.Get("After"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "TestRenderTemplate":
		rendered, err := s.ses.TestRenderTemplate(strings.TrimSpace(r.Form.Get("TemplateName")), r.Form.Get("TemplateData"))
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESXML(w, action, sesTestRenderTemplateResult{RenderedTemplate: rendered})
		return true
	case "UpdateConfigurationSetEventDestination":
		destination, err := parseSESEventDestinationInput(r.Form, "EventDestination")
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		if err := s.ses.UpdateConfigurationSetEventDestination(strings.TrimSpace(r.Form.Get("ConfigurationSetName")), destination); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "UpdateConfigurationSetReputationMetricsEnabled":
		enabled, err := parseSESRequiredBool(r.Form.Get("Enabled"))
		if err != nil {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Enabled is required")
			return true
		}
		if err := s.ses.UpdateConfigurationSetReputationMetricsEnabled(strings.TrimSpace(r.Form.Get("ConfigurationSetName")), enabled); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "UpdateConfigurationSetSendingEnabled":
		enabled, err := parseSESRequiredBool(r.Form.Get("Enabled"))
		if err != nil {
			respondSESErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Enabled is required")
			return true
		}
		if err := s.ses.UpdateConfigurationSetSendingEnabled(strings.TrimSpace(r.Form.Get("ConfigurationSetName")), enabled); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "UpdateConfigurationSetTrackingOptions":
		if err := s.ses.UpdateConfigurationSetTrackingOptions(strings.TrimSpace(r.Form.Get("ConfigurationSetName")), strings.TrimSpace(r.Form.Get("TrackingOptions.CustomRedirectDomain"))); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "UpdateCustomVerificationEmailTemplate":
		err := s.ses.UpdateCustomVerificationEmailTemplate(ses.CustomVerificationEmailTemplate{
			TemplateName:          strings.TrimSpace(r.Form.Get("TemplateName")),
			FromEmailAddress:      strings.TrimSpace(r.Form.Get("FromEmailAddress")),
			TemplateSubject:       r.Form.Get("TemplateSubject"),
			TemplateContent:       r.Form.Get("TemplateContent"),
			SuccessRedirectionURL: r.Form.Get("SuccessRedirectionURL"),
			FailureRedirectionURL: r.Form.Get("FailureRedirectionURL"),
		})
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	case "UpdateReceiptRule":
		rule, err := parseSESReceiptRuleInput(r.Form, "Rule")
		if err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		if err := s.ses.UpdateReceiptRule(strings.TrimSpace(r.Form.Get("RuleSetName")), rule); err != nil {
			respondSESErrorForErr(w, err)
			return true
		}
		respondSESEmptyResult(w, action)
		return true
	default:
		return false
	}
}

func parseSESEventDestinationInput(values url.Values, prefix string) (ses.ConfigurationSetEventDestination, error) {
	name := strings.TrimSpace(values.Get(prefix + ".Name"))
	if name == "" {
		return ses.ConfigurationSetEventDestination{}, ses.ErrInvalidParameter
	}
	enabled, err := parseSESOptionalBool(values.Get(prefix+".Enabled"), true)
	if err != nil {
		return ses.ConfigurationSetEventDestination{}, ses.ErrInvalidParameter
	}
	return ses.ConfigurationSetEventDestination{
		Name:                          name,
		Enabled:                       enabled,
		MatchingEventTypes:            parseSESStringMembers(values, prefix+".MatchingEventTypes.member"),
		SNSDestinationTopicARN:        strings.TrimSpace(values.Get(prefix + ".SNSDestination.TopicARN")),
		KinesisFirehoseIAMRoleARN:     strings.TrimSpace(values.Get(prefix + ".KinesisFirehoseDestination.IAMRoleARN")),
		KinesisFirehoseDeliveryStream: strings.TrimSpace(values.Get(prefix + ".KinesisFirehoseDestination.DeliveryStreamARN")),
	}, nil
}

func parseSESReceiptRuleInput(values url.Values, prefix string) (ses.ReceiptRule, error) {
	name := strings.TrimSpace(values.Get(prefix + ".Name"))
	if name == "" {
		return ses.ReceiptRule{}, ses.ErrInvalidParameter
	}
	enabled, err := parseSESOptionalBool(values.Get(prefix+".Enabled"), true)
	if err != nil {
		return ses.ReceiptRule{}, ses.ErrInvalidParameter
	}
	scanEnabled, err := parseSESOptionalBool(values.Get(prefix+".ScanEnabled"), false)
	if err != nil {
		return ses.ReceiptRule{}, ses.ErrInvalidParameter
	}
	return ses.ReceiptRule{
		Name:        name,
		Enabled:     enabled,
		TLSPolicy:   strings.TrimSpace(values.Get(prefix + ".TlsPolicy")),
		Recipients:  parseSESStringMembers(values, prefix+".Recipients.member"),
		ScanEnabled: scanEnabled,
	}, nil
}

func parseSESBulkTemplatedDestinations(values url.Values) []ses.BulkTemplatedDestination {
	indices := make([]int, 0)
	seen := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, "Destinations.member.") {
			continue
		}
		rest := strings.TrimPrefix(key, "Destinations.member.")
		parts := strings.Split(rest, ".")
		if len(parts) < 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		if _, ok := seen[idx]; ok {
			continue
		}
		seen[idx] = struct{}{}
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	out := make([]ses.BulkTemplatedDestination, 0, len(indices))
	for _, idx := range indices {
		prefix := fmt.Sprintf("Destinations.member.%d.Destination", idx)
		out = append(out, ses.BulkTemplatedDestination{
			Destinations:            parseSESDestination(values, prefix),
			ReplacementTemplateData: values.Get(fmt.Sprintf("Destinations.member.%d.ReplacementTemplateData", idx)),
		})
	}
	return out
}

func toSESEventDestinations(in map[string]*ses.ConfigurationSetEventDestination) []sesConfigurationSetEventDestination {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]sesConfigurationSetEventDestination, 0, len(keys))
	for _, key := range keys {
		destination := in[key]
		out = append(out, sesConfigurationSetEventDestination{
			Name:               destination.Name,
			Enabled:            destination.Enabled,
			MatchingEventTypes: append([]string(nil), destination.MatchingEventTypes...),
			SNSDestination:     sesEventDestinationSNS{TopicARN: destination.SNSDestinationTopicARN},
			KinesisFirehoseDestination: sesEventDestinationKinesisFirehose{
				IAMRoleARN:        destination.KinesisFirehoseIAMRoleARN,
				DeliveryStreamARN: destination.KinesisFirehoseDeliveryStream,
			},
		})
	}
	return out
}

func toSESReceiptRules(in []*ses.ReceiptRule) []sesReceiptRule {
	out := make([]sesReceiptRule, 0, len(in))
	for _, rule := range in {
		out = append(out, toSESReceiptRule(*rule))
	}
	return out
}

func toSESReceiptRule(in ses.ReceiptRule) sesReceiptRule {
	return sesReceiptRule{
		Name:        in.Name,
		Enabled:     in.Enabled,
		TLSPolicy:   in.TLSPolicy,
		Recipients:  append([]string(nil), in.Recipients...),
		ScanEnabled: in.ScanEnabled,
	}
}

func respondSESEmptyResult(w http.ResponseWriter, action string) {
	respondSESXML(w, action, sesDynamicResult{XMLName: xml.Name{Local: action + "Result"}})
}

type sesDynamicResult struct {
	XMLName xml.Name `xml:""`
}

type sesListConfigurationSetsResult struct {
	XMLName           xml.Name                      `xml:"ListConfigurationSetsResult"`
	ConfigurationSets []sesConfigurationSetMetadata `xml:"ConfigurationSets>member"`
	NextToken         string                        `xml:"NextToken,omitempty"`
}

type sesConfigurationSetMetadata struct {
	Name string `xml:"Name"`
}

type sesDescribeConfigurationSetResult struct {
	XMLName           xml.Name                              `xml:"DescribeConfigurationSetResult"`
	ConfigurationSet  sesConfigurationSetMetadata           `xml:"ConfigurationSet"`
	EventDestinations []sesConfigurationSetEventDestination `xml:"EventDestinations>member,omitempty"`
	TrackingOptions   sesTrackingOptions                    `xml:"TrackingOptions,omitempty"`
	ReputationOptions sesReputationOptions                  `xml:"ReputationOptions,omitempty"`
	SendingOptions    sesSendingOptions                     `xml:"SendingOptions,omitempty"`
	DeliveryOptions   sesDeliveryOptions                    `xml:"DeliveryOptions,omitempty"`
}

type sesTrackingOptions struct {
	CustomRedirectDomain string `xml:"CustomRedirectDomain,omitempty"`
}

type sesReputationOptions struct {
	ReputationMetricsEnabled bool `xml:"ReputationMetricsEnabled"`
}

type sesSendingOptions struct {
	SendingEnabled bool `xml:"SendingEnabled"`
}

type sesDeliveryOptions struct {
	TLSPolicy string `xml:"TlsPolicy,omitempty"`
}

type sesConfigurationSetEventDestination struct {
	Name                       string                             `xml:"Name"`
	Enabled                    bool                               `xml:"Enabled"`
	MatchingEventTypes         []string                           `xml:"MatchingEventTypes>member,omitempty"`
	SNSDestination             sesEventDestinationSNS             `xml:"SNSDestination,omitempty"`
	KinesisFirehoseDestination sesEventDestinationKinesisFirehose `xml:"KinesisFirehoseDestination,omitempty"`
}

type sesEventDestinationSNS struct {
	TopicARN string `xml:"TopicARN,omitempty"`
}

type sesEventDestinationKinesisFirehose struct {
	IAMRoleARN        string `xml:"IAMRoleARN,omitempty"`
	DeliveryStreamARN string `xml:"DeliveryStreamARN,omitempty"`
}

type sesSendCustomVerificationEmailResult struct {
	XMLName   xml.Name `xml:"SendCustomVerificationEmailResult"`
	MessageID string   `xml:"MessageId"`
}

type sesGetCustomVerificationEmailTemplateResult struct {
	XMLName               xml.Name `xml:"GetCustomVerificationEmailTemplateResult"`
	TemplateName          string   `xml:"TemplateName"`
	FromEmailAddress      string   `xml:"FromEmailAddress"`
	TemplateSubject       string   `xml:"TemplateSubject"`
	TemplateContent       string   `xml:"TemplateContent"`
	SuccessRedirectionURL string   `xml:"SuccessRedirectionURL,omitempty"`
	FailureRedirectionURL string   `xml:"FailureRedirectionURL,omitempty"`
}

type sesListCustomVerificationEmailTemplatesResult struct {
	XMLName   xml.Name                                `xml:"ListCustomVerificationEmailTemplatesResult"`
	Templates []sesCustomVerificationTemplateMetadata `xml:"CustomVerificationEmailTemplates>member"`
	NextToken string                                  `xml:"NextToken,omitempty"`
}

type sesCustomVerificationTemplateMetadata struct {
	TemplateName          string `xml:"TemplateName"`
	FromEmailAddress      string `xml:"FromEmailAddress"`
	TemplateSubject       string `xml:"TemplateSubject"`
	SuccessRedirectionURL string `xml:"SuccessRedirectionURL,omitempty"`
	FailureRedirectionURL string `xml:"FailureRedirectionURL,omitempty"`
}

type sesListIdentityPoliciesResult struct {
	XMLName     xml.Name `xml:"ListIdentityPoliciesResult"`
	PolicyNames []string `xml:"PolicyNames>member"`
}

type sesGetIdentityPoliciesResult struct {
	XMLName  xml.Name                 `xml:"GetIdentityPoliciesResult"`
	Policies []sesIdentityPolicyEntry `xml:"Policies>entry"`
}

type sesIdentityPolicyEntry struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type sesListReceiptFiltersResult struct {
	XMLName xml.Name              `xml:"ListReceiptFiltersResult"`
	Filters []sesReceiptFilterXML `xml:"Filters>member"`
}

type sesReceiptFilterXML struct {
	Name     string                `xml:"Name"`
	IPFilter sesReceiptIPFilterXML `xml:"IpFilter"`
}

type sesReceiptIPFilterXML struct {
	Policy string `xml:"Policy"`
	CIDR   string `xml:"Cidr"`
}

type sesListReceiptRuleSetsResult struct {
	XMLName   xml.Name                    `xml:"ListReceiptRuleSetsResult"`
	RuleSets  []sesReceiptRuleSetMetadata `xml:"RuleSets>member"`
	NextToken string                      `xml:"NextToken,omitempty"`
}

type sesReceiptRuleSetMetadata struct {
	Name             string    `xml:"Name"`
	CreatedTimestamp time.Time `xml:"CreatedTimestamp"`
}

type sesDescribeActiveReceiptRuleSetResult struct {
	XMLName  xml.Name                   `xml:"DescribeActiveReceiptRuleSetResult"`
	Metadata *sesReceiptRuleSetMetadata `xml:"Metadata,omitempty"`
	Rules    []sesReceiptRule           `xml:"Rules>member,omitempty"`
}

type sesDescribeReceiptRuleSetResult struct {
	XMLName  xml.Name                  `xml:"DescribeReceiptRuleSetResult"`
	Metadata sesReceiptRuleSetMetadata `xml:"Metadata"`
	Rules    []sesReceiptRule          `xml:"Rules>member,omitempty"`
}

type sesDescribeReceiptRuleResult struct {
	XMLName xml.Name       `xml:"DescribeReceiptRuleResult"`
	Rule    sesReceiptRule `xml:"Rule"`
}

type sesReceiptRule struct {
	Name        string   `xml:"Name"`
	Enabled     bool     `xml:"Enabled"`
	TLSPolicy   string   `xml:"TlsPolicy,omitempty"`
	Recipients  []string `xml:"Recipients>member,omitempty"`
	ScanEnabled bool     `xml:"ScanEnabled"`
}

type sesSendBounceResult struct {
	XMLName   xml.Name `xml:"SendBounceResult"`
	MessageID string   `xml:"MessageId"`
}

type sesSendBulkTemplatedEmailResult struct {
	XMLName xml.Name                      `xml:"SendBulkTemplatedEmailResult"`
	Status  []sesBulkTemplatedEmailStatus `xml:"Status>member"`
}

type sesBulkTemplatedEmailStatus struct {
	Status    string `xml:"Status"`
	MessageID string `xml:"MessageId,omitempty"`
	Error     string `xml:"Error,omitempty"`
}

type sesTestRenderTemplateResult struct {
	XMLName          xml.Name `xml:"TestRenderTemplateResult"`
	RenderedTemplate string   `xml:"RenderedTemplate"`
}
