package ses

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrConfigurationSetExists             = errors.New("configuration set already exists")
	ErrConfigurationSetNotFound           = errors.New("configuration set not found")
	ErrEventDestinationExists             = errors.New("configuration set event destination already exists")
	ErrEventDestinationNotFound           = errors.New("configuration set event destination not found")
	ErrCustomVerificationTemplateExists   = errors.New("custom verification email template already exists")
	ErrCustomVerificationTemplateNotFound = errors.New("custom verification email template not found")
	ErrReceiptFilterExists                = errors.New("receipt filter already exists")
	ErrReceiptFilterNotFound              = errors.New("receipt filter not found")
	ErrReceiptRuleSetExists               = errors.New("receipt rule set already exists")
	ErrReceiptRuleSetNotFound             = errors.New("receipt rule set not found")
	ErrReceiptRuleExists                  = errors.New("receipt rule already exists")
	ErrReceiptRuleNotFound                = errors.New("receipt rule not found")
	ErrIdentityPolicyNotFound             = errors.New("identity policy not found")
)

type ConfigurationSet struct {
	Name                     string
	CreatedAt                time.Time
	EventDestinations        map[string]*ConfigurationSetEventDestination
	TrackingDomain           string
	SendingEnabled           bool
	ReputationMetricsEnabled bool
	TLSPolicy                string
}

type ConfigurationSetEventDestination struct {
	Name                          string
	Enabled                       bool
	MatchingEventTypes            []string
	SNSDestinationTopicARN        string
	KinesisFirehoseIAMRoleARN     string
	KinesisFirehoseDeliveryStream string
}

type CustomVerificationEmailTemplate struct {
	TemplateName          string
	FromEmailAddress      string
	TemplateSubject       string
	TemplateContent       string
	SuccessRedirectionURL string
	FailureRedirectionURL string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ReceiptFilter struct {
	Name   string
	Policy string
	CIDR   string
}

type ReceiptRuleSet struct {
	Name      string
	CreatedAt time.Time
	Rules     []*ReceiptRule
}

type ReceiptRule struct {
	Name        string
	Enabled     bool
	TLSPolicy   string
	Recipients  []string
	ScanEnabled bool
}

type BulkTemplatedDestination struct {
	Destinations            []string
	ReplacementTemplateData string
}

type BulkTemplatedEmailStatus struct {
	Status    string
	MessageID string
	Error     string
}

func (s *Service) CreateConfigurationSet(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.configurationSets[name]; ok {
		return ErrConfigurationSetExists
	}
	s.configurationSets[name] = &ConfigurationSet{
		Name:                     name,
		CreatedAt:                time.Now().UTC(),
		EventDestinations:        map[string]*ConfigurationSetEventDestination{},
		SendingEnabled:           true,
		ReputationMetricsEnabled: true,
		TLSPolicy:                "Optional",
	}
	return nil
}

func (s *Service) DeleteConfigurationSet(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.configurationSets, name)
}

func (s *Service) ListConfigurationSets(nextToken string, maxItems int) ([]ConfigurationSet, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if maxItems <= 0 {
		maxItems = 100
	}

	items := make([]ConfigurationSet, 0, len(s.configurationSets))
	for _, cfg := range s.configurationSets {
		items = append(items, ConfigurationSet{Name: cfg.Name, CreatedAt: cfg.CreatedAt})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	start, err := parseToken(nextToken)
	if err != nil || start < 0 || start > len(items) {
		return nil, "", ErrInvalidParameter
	}
	end := start + maxItems
	if end > len(items) {
		end = len(items)
	}
	page := append([]ConfigurationSet(nil), items[start:end]...)
	next := ""
	if end < len(items) {
		next = fmt.Sprintf("%d", end)
	}
	return page, next, nil
}

func (s *Service) DescribeConfigurationSet(name string) (ConfigurationSet, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ConfigurationSet{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := s.configurationSets[name]
	if !ok {
		return ConfigurationSet{}, ErrConfigurationSetNotFound
	}
	out := *cfg
	out.EventDestinations = make(map[string]*ConfigurationSetEventDestination, len(cfg.EventDestinations))
	for key, dest := range cfg.EventDestinations {
		copied := *dest
		copied.MatchingEventTypes = append([]string(nil), dest.MatchingEventTypes...)
		out.EventDestinations[key] = &copied
	}
	return out, nil
}

func (s *Service) CreateConfigurationSetEventDestination(configurationSetName string, destination ConfigurationSetEventDestination) error {
	return s.putConfigurationSetEventDestination(configurationSetName, destination, true)
}

func (s *Service) UpdateConfigurationSetEventDestination(configurationSetName string, destination ConfigurationSetEventDestination) error {
	return s.putConfigurationSetEventDestination(configurationSetName, destination, false)
}

func (s *Service) putConfigurationSetEventDestination(configurationSetName string, destination ConfigurationSetEventDestination, createOnly bool) error {
	configurationSetName = strings.TrimSpace(configurationSetName)
	destination.Name = strings.TrimSpace(destination.Name)
	if configurationSetName == "" || destination.Name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := s.configurationSets[configurationSetName]
	if !ok {
		return ErrConfigurationSetNotFound
	}
	if cfg.EventDestinations == nil {
		cfg.EventDestinations = map[string]*ConfigurationSetEventDestination{}
	}
	if _, exists := cfg.EventDestinations[destination.Name]; exists && createOnly {
		return ErrEventDestinationExists
	}
	if _, exists := cfg.EventDestinations[destination.Name]; !exists && !createOnly {
		return ErrEventDestinationNotFound
	}

	copied := destination
	copied.MatchingEventTypes = append([]string(nil), destination.MatchingEventTypes...)
	cfg.EventDestinations[destination.Name] = &copied
	return nil
}

func (s *Service) DeleteConfigurationSetEventDestination(configurationSetName, destinationName string) error {
	configurationSetName = strings.TrimSpace(configurationSetName)
	destinationName = strings.TrimSpace(destinationName)
	if configurationSetName == "" || destinationName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := s.configurationSets[configurationSetName]
	if !ok {
		return ErrConfigurationSetNotFound
	}
	if _, exists := cfg.EventDestinations[destinationName]; !exists {
		return ErrEventDestinationNotFound
	}
	delete(cfg.EventDestinations, destinationName)
	return nil
}

func (s *Service) CreateConfigurationSetTrackingOptions(configurationSetName, customRedirectDomain string) error {
	return s.updateConfigurationSetTrackingDomain(configurationSetName, customRedirectDomain)
}

func (s *Service) UpdateConfigurationSetTrackingOptions(configurationSetName, customRedirectDomain string) error {
	return s.updateConfigurationSetTrackingDomain(configurationSetName, customRedirectDomain)
}

func (s *Service) updateConfigurationSetTrackingDomain(configurationSetName, customRedirectDomain string) error {
	configurationSetName = strings.TrimSpace(configurationSetName)
	customRedirectDomain = strings.TrimSpace(customRedirectDomain)
	if configurationSetName == "" || customRedirectDomain == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configurationSets[configurationSetName]
	if !ok {
		return ErrConfigurationSetNotFound
	}
	cfg.TrackingDomain = customRedirectDomain
	return nil
}

func (s *Service) DeleteConfigurationSetTrackingOptions(configurationSetName string) error {
	configurationSetName = strings.TrimSpace(configurationSetName)
	if configurationSetName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configurationSets[configurationSetName]
	if !ok {
		return ErrConfigurationSetNotFound
	}
	cfg.TrackingDomain = ""
	return nil
}

func (s *Service) PutConfigurationSetDeliveryOptions(configurationSetName, tlsPolicy string) error {
	configurationSetName = strings.TrimSpace(configurationSetName)
	tlsPolicy = strings.TrimSpace(tlsPolicy)
	if configurationSetName == "" {
		return ErrInvalidParameter
	}
	if tlsPolicy == "" {
		tlsPolicy = "Optional"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configurationSets[configurationSetName]
	if !ok {
		return ErrConfigurationSetNotFound
	}
	cfg.TLSPolicy = tlsPolicy
	return nil
}

func (s *Service) UpdateConfigurationSetReputationMetricsEnabled(configurationSetName string, enabled bool) error {
	configurationSetName = strings.TrimSpace(configurationSetName)
	if configurationSetName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configurationSets[configurationSetName]
	if !ok {
		return ErrConfigurationSetNotFound
	}
	cfg.ReputationMetricsEnabled = enabled
	return nil
}

func (s *Service) UpdateConfigurationSetSendingEnabled(configurationSetName string, enabled bool) error {
	configurationSetName = strings.TrimSpace(configurationSetName)
	if configurationSetName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configurationSets[configurationSetName]
	if !ok {
		return ErrConfigurationSetNotFound
	}
	cfg.SendingEnabled = enabled
	return nil
}

func (s *Service) CreateCustomVerificationEmailTemplate(template CustomVerificationEmailTemplate) error {
	template.TemplateName = strings.TrimSpace(template.TemplateName)
	template.FromEmailAddress = strings.TrimSpace(template.FromEmailAddress)
	if template.TemplateName == "" || template.FromEmailAddress == "" || strings.TrimSpace(template.TemplateSubject) == "" || strings.TrimSpace(template.TemplateContent) == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.customVerifTemplates[template.TemplateName]; ok {
		return ErrCustomVerificationTemplateExists
	}
	now := time.Now().UTC()
	template.CreatedAt = now
	template.UpdatedAt = now
	s.customVerifTemplates[template.TemplateName] = &template
	return nil
}

func (s *Service) UpdateCustomVerificationEmailTemplate(template CustomVerificationEmailTemplate) error {
	template.TemplateName = strings.TrimSpace(template.TemplateName)
	template.FromEmailAddress = strings.TrimSpace(template.FromEmailAddress)
	if template.TemplateName == "" || template.FromEmailAddress == "" || strings.TrimSpace(template.TemplateSubject) == "" || strings.TrimSpace(template.TemplateContent) == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.customVerifTemplates[template.TemplateName]
	if !ok {
		return ErrCustomVerificationTemplateNotFound
	}
	current.FromEmailAddress = template.FromEmailAddress
	current.TemplateSubject = template.TemplateSubject
	current.TemplateContent = template.TemplateContent
	current.SuccessRedirectionURL = template.SuccessRedirectionURL
	current.FailureRedirectionURL = template.FailureRedirectionURL
	current.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) DeleteCustomVerificationEmailTemplate(templateName string) {
	templateName = strings.TrimSpace(templateName)
	if templateName == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.customVerifTemplates, templateName)
}

func (s *Service) GetCustomVerificationEmailTemplate(templateName string) (CustomVerificationEmailTemplate, error) {
	templateName = strings.TrimSpace(templateName)
	if templateName == "" {
		return CustomVerificationEmailTemplate{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	tpl, ok := s.customVerifTemplates[templateName]
	if !ok {
		return CustomVerificationEmailTemplate{}, ErrCustomVerificationTemplateNotFound
	}
	return *tpl, nil
}

func (s *Service) ListCustomVerificationEmailTemplates(nextToken string, maxItems int) ([]CustomVerificationEmailTemplate, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if maxItems <= 0 {
		maxItems = 10
	}

	items := make([]CustomVerificationEmailTemplate, 0, len(s.customVerifTemplates))
	for _, tpl := range s.customVerifTemplates {
		items = append(items, *tpl)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].TemplateName < items[j].TemplateName })

	start, err := parseToken(nextToken)
	if err != nil || start < 0 || start > len(items) {
		return nil, "", ErrInvalidParameter
	}
	end := start + maxItems
	if end > len(items) {
		end = len(items)
	}
	page := append([]CustomVerificationEmailTemplate(nil), items[start:end]...)
	next := ""
	if end < len(items) {
		next = fmt.Sprintf("%d", end)
	}
	return page, next, nil
}

func (s *Service) SendCustomVerificationEmail(templateName, emailAddress string) (string, error) {
	if !s.GetAccountSendingEnabled() {
		return "", ErrMessageRejected
	}
	templateName = strings.TrimSpace(templateName)
	emailAddress = strings.TrimSpace(emailAddress)
	if templateName == "" || emailAddress == "" || !strings.Contains(emailAddress, "@") {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	_, ok := s.customVerifTemplates[templateName]
	s.mu.Unlock()
	if !ok {
		return "", ErrCustomVerificationTemplateNotFound
	}

	return s.recordSend(), nil
}

func (s *Service) PutIdentityPolicy(identity, policyName, policy string) error {
	identity = normalizeIdentity(identity)
	policyName = strings.TrimSpace(policyName)
	if identity == "" || policyName == "" || strings.TrimSpace(policy) == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.identities[identity]; !ok {
		return ErrIdentityNotFound
	}
	if s.identityPolicies[identity] == nil {
		s.identityPolicies[identity] = map[string]string{}
	}
	s.identityPolicies[identity][policyName] = policy
	return nil
}

func (s *Service) DeleteIdentityPolicy(identity, policyName string) error {
	identity = normalizeIdentity(identity)
	policyName = strings.TrimSpace(policyName)
	if identity == "" || policyName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.identities[identity]; !ok {
		return ErrIdentityNotFound
	}
	policies := s.identityPolicies[identity]
	if policies == nil {
		return ErrIdentityPolicyNotFound
	}
	if _, ok := policies[policyName]; !ok {
		return ErrIdentityPolicyNotFound
	}
	delete(policies, policyName)
	if len(policies) == 0 {
		delete(s.identityPolicies, identity)
	}
	return nil
}

func (s *Service) ListIdentityPolicies(identity string) ([]string, error) {
	identity = normalizeIdentity(identity)
	if identity == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.identities[identity]; !ok {
		return nil, ErrIdentityNotFound
	}
	policies := s.identityPolicies[identity]
	out := make([]string, 0, len(policies))
	for name := range policies {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Service) GetIdentityPolicies(identity string, policyNames []string) (map[string]string, error) {
	identity = normalizeIdentity(identity)
	if identity == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.identities[identity]; !ok {
		return nil, ErrIdentityNotFound
	}

	policies := s.identityPolicies[identity]
	out := map[string]string{}
	if len(policyNames) == 0 {
		for key, value := range policies {
			out[key] = value
		}
		return out, nil
	}
	for _, name := range policyNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if value, ok := policies[name]; ok {
			out[name] = value
		}
	}
	return out, nil
}

func (s *Service) CreateReceiptFilter(filter ReceiptFilter) error {
	filter.Name = strings.TrimSpace(filter.Name)
	filter.Policy = strings.TrimSpace(filter.Policy)
	filter.CIDR = strings.TrimSpace(filter.CIDR)
	if filter.Name == "" || filter.Policy == "" || filter.CIDR == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.receiptFilters[filter.Name]; ok {
		return ErrReceiptFilterExists
	}
	copied := filter
	s.receiptFilters[filter.Name] = &copied
	return nil
}

func (s *Service) DeleteReceiptFilter(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.receiptFilters[name]; !ok {
		return ErrReceiptFilterNotFound
	}
	delete(s.receiptFilters, name)
	return nil
}

func (s *Service) ListReceiptFilters() []ReceiptFilter {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]ReceiptFilter, 0, len(s.receiptFilters))
	for _, filter := range s.receiptFilters {
		items = append(items, *filter)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (s *Service) CreateReceiptRuleSet(ruleSetName string) error {
	ruleSetName = strings.TrimSpace(ruleSetName)
	if ruleSetName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.receiptRuleSets[ruleSetName]; ok {
		return ErrReceiptRuleSetExists
	}
	s.receiptRuleSets[ruleSetName] = &ReceiptRuleSet{
		Name:      ruleSetName,
		CreatedAt: time.Now().UTC(),
		Rules:     make([]*ReceiptRule, 0),
	}
	return nil
}

func (s *Service) DeleteReceiptRuleSet(ruleSetName string) error {
	ruleSetName = strings.TrimSpace(ruleSetName)
	if ruleSetName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.receiptRuleSets[ruleSetName]; !ok {
		return ErrReceiptRuleSetNotFound
	}
	delete(s.receiptRuleSets, ruleSetName)
	if s.activeReceiptRuleSet == ruleSetName {
		s.activeReceiptRuleSet = ""
	}
	return nil
}

func (s *Service) ListReceiptRuleSets(nextToken string, maxItems int) ([]ReceiptRuleSet, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if maxItems <= 0 {
		maxItems = 100
	}

	items := make([]ReceiptRuleSet, 0, len(s.receiptRuleSets))
	for _, ruleSet := range s.receiptRuleSets {
		items = append(items, ReceiptRuleSet{Name: ruleSet.Name, CreatedAt: ruleSet.CreatedAt})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	start, err := parseToken(nextToken)
	if err != nil || start < 0 || start > len(items) {
		return nil, "", ErrInvalidParameter
	}
	end := start + maxItems
	if end > len(items) {
		end = len(items)
	}
	page := append([]ReceiptRuleSet(nil), items[start:end]...)
	next := ""
	if end < len(items) {
		next = fmt.Sprintf("%d", end)
	}
	return page, next, nil
}

func (s *Service) SetActiveReceiptRuleSet(ruleSetName string) error {
	ruleSetName = strings.TrimSpace(ruleSetName)

	s.mu.Lock()
	defer s.mu.Unlock()
	if ruleSetName == "" {
		s.activeReceiptRuleSet = ""
		return nil
	}
	if _, ok := s.receiptRuleSets[ruleSetName]; !ok {
		return ErrReceiptRuleSetNotFound
	}
	s.activeReceiptRuleSet = ruleSetName
	return nil
}

func (s *Service) DescribeActiveReceiptRuleSet() (*ReceiptRuleSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeReceiptRuleSet == "" {
		return nil, nil
	}
	ruleSet, ok := s.receiptRuleSets[s.activeReceiptRuleSet]
	if !ok {
		return nil, nil
	}
	copied := copyReceiptRuleSet(ruleSet)
	return &copied, nil
}

func (s *Service) DescribeReceiptRuleSet(ruleSetName string) (ReceiptRuleSet, error) {
	ruleSetName = strings.TrimSpace(ruleSetName)
	if ruleSetName == "" {
		return ReceiptRuleSet{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	ruleSet, ok := s.receiptRuleSets[ruleSetName]
	if !ok {
		return ReceiptRuleSet{}, ErrReceiptRuleSetNotFound
	}
	return copyReceiptRuleSet(ruleSet), nil
}

func (s *Service) CloneReceiptRuleSet(sourceRuleSetName, ruleSetName string) error {
	sourceRuleSetName = strings.TrimSpace(sourceRuleSetName)
	ruleSetName = strings.TrimSpace(ruleSetName)
	if sourceRuleSetName == "" || ruleSetName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	source, ok := s.receiptRuleSets[sourceRuleSetName]
	if !ok {
		return ErrReceiptRuleSetNotFound
	}
	if _, exists := s.receiptRuleSets[ruleSetName]; exists {
		return ErrReceiptRuleSetExists
	}
	copied := copyReceiptRuleSet(source)
	copied.Name = ruleSetName
	copied.CreatedAt = time.Now().UTC()
	s.receiptRuleSets[ruleSetName] = &copied
	return nil
}

func (s *Service) CreateReceiptRule(ruleSetName, after string, rule ReceiptRule) error {
	ruleSetName = strings.TrimSpace(ruleSetName)
	after = strings.TrimSpace(after)
	rule.Name = strings.TrimSpace(rule.Name)
	if ruleSetName == "" || rule.Name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ruleSet, ok := s.receiptRuleSets[ruleSetName]
	if !ok {
		return ErrReceiptRuleSetNotFound
	}
	if receiptRuleIndex(ruleSet.Rules, rule.Name) >= 0 {
		return ErrReceiptRuleExists
	}
	ruleCopy := normalizeReceiptRule(rule)
	if after == "" {
		ruleSet.Rules = append(ruleSet.Rules, &ruleCopy)
		return nil
	}
	afterIndex := receiptRuleIndex(ruleSet.Rules, after)
	if afterIndex < 0 {
		return ErrReceiptRuleNotFound
	}
	ruleSet.Rules = append(ruleSet.Rules, nil)
	copy(ruleSet.Rules[afterIndex+2:], ruleSet.Rules[afterIndex+1:])
	ruleSet.Rules[afterIndex+1] = &ruleCopy
	return nil
}

func (s *Service) UpdateReceiptRule(ruleSetName string, rule ReceiptRule) error {
	ruleSetName = strings.TrimSpace(ruleSetName)
	rule.Name = strings.TrimSpace(rule.Name)
	if ruleSetName == "" || rule.Name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ruleSet, ok := s.receiptRuleSets[ruleSetName]
	if !ok {
		return ErrReceiptRuleSetNotFound
	}
	idx := receiptRuleIndex(ruleSet.Rules, rule.Name)
	if idx < 0 {
		return ErrReceiptRuleNotFound
	}
	ruleCopy := normalizeReceiptRule(rule)
	ruleSet.Rules[idx] = &ruleCopy
	return nil
}

func (s *Service) DeleteReceiptRule(ruleSetName, ruleName string) error {
	ruleSetName = strings.TrimSpace(ruleSetName)
	ruleName = strings.TrimSpace(ruleName)
	if ruleSetName == "" || ruleName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ruleSet, ok := s.receiptRuleSets[ruleSetName]
	if !ok {
		return ErrReceiptRuleSetNotFound
	}
	idx := receiptRuleIndex(ruleSet.Rules, ruleName)
	if idx < 0 {
		return ErrReceiptRuleNotFound
	}
	ruleSet.Rules = append(ruleSet.Rules[:idx], ruleSet.Rules[idx+1:]...)
	return nil
}

func (s *Service) DescribeReceiptRule(ruleSetName, ruleName string) (ReceiptRule, error) {
	ruleSetName = strings.TrimSpace(ruleSetName)
	ruleName = strings.TrimSpace(ruleName)
	if ruleSetName == "" || ruleName == "" {
		return ReceiptRule{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ruleSet, ok := s.receiptRuleSets[ruleSetName]
	if !ok {
		return ReceiptRule{}, ErrReceiptRuleSetNotFound
	}
	idx := receiptRuleIndex(ruleSet.Rules, ruleName)
	if idx < 0 {
		return ReceiptRule{}, ErrReceiptRuleNotFound
	}
	return *ruleSet.Rules[idx], nil
}

func (s *Service) ReorderReceiptRuleSet(ruleSetName string, ruleNames []string) error {
	ruleSetName = strings.TrimSpace(ruleSetName)
	if ruleSetName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ruleSet, ok := s.receiptRuleSets[ruleSetName]
	if !ok {
		return ErrReceiptRuleSetNotFound
	}
	if len(ruleNames) != len(ruleSet.Rules) {
		return ErrInvalidParameter
	}
	lookup := make(map[string]*ReceiptRule, len(ruleSet.Rules))
	for _, rule := range ruleSet.Rules {
		lookup[rule.Name] = rule
	}
	newRules := make([]*ReceiptRule, 0, len(ruleSet.Rules))
	seen := map[string]struct{}{}
	for _, name := range ruleNames {
		name = strings.TrimSpace(name)
		rule, exists := lookup[name]
		if !exists {
			return ErrReceiptRuleNotFound
		}
		if _, duplicate := seen[name]; duplicate {
			return ErrInvalidParameter
		}
		seen[name] = struct{}{}
		newRules = append(newRules, rule)
	}
	ruleSet.Rules = newRules
	return nil
}

func (s *Service) SetReceiptRulePosition(ruleSetName, ruleName, after string) error {
	ruleSetName = strings.TrimSpace(ruleSetName)
	ruleName = strings.TrimSpace(ruleName)
	after = strings.TrimSpace(after)
	if ruleSetName == "" || ruleName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ruleSet, ok := s.receiptRuleSets[ruleSetName]
	if !ok {
		return ErrReceiptRuleSetNotFound
	}
	idx := receiptRuleIndex(ruleSet.Rules, ruleName)
	if idx < 0 {
		return ErrReceiptRuleNotFound
	}
	rule := ruleSet.Rules[idx]
	ruleSet.Rules = append(ruleSet.Rules[:idx], ruleSet.Rules[idx+1:]...)

	if after == "" {
		ruleSet.Rules = append([]*ReceiptRule{rule}, ruleSet.Rules...)
		return nil
	}
	afterIdx := receiptRuleIndex(ruleSet.Rules, after)
	if afterIdx < 0 {
		return ErrReceiptRuleNotFound
	}
	ruleSet.Rules = append(ruleSet.Rules, nil)
	copy(ruleSet.Rules[afterIdx+2:], ruleSet.Rules[afterIdx+1:])
	ruleSet.Rules[afterIdx+1] = rule
	return nil
}

func (s *Service) SendBounce(originalMessageID, bounceSender string) (string, error) {
	if !s.GetAccountSendingEnabled() {
		return "", ErrMessageRejected
	}
	originalMessageID = strings.TrimSpace(originalMessageID)
	bounceSender = strings.TrimSpace(bounceSender)
	if originalMessageID == "" || bounceSender == "" || !strings.Contains(bounceSender, "@") {
		return "", ErrInvalidParameter
	}
	return s.recordSend(), nil
}

func (s *Service) SendBulkTemplatedEmail(source, templateName string, destinations []BulkTemplatedDestination) ([]BulkTemplatedEmailStatus, error) {
	if !s.GetAccountSendingEnabled() {
		return nil, ErrMessageRejected
	}
	source = strings.TrimSpace(source)
	templateName = strings.TrimSpace(templateName)
	if source == "" || !strings.Contains(source, "@") || templateName == "" || len(destinations) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	_, ok := s.templates[templateName]
	s.mu.Unlock()
	if !ok {
		return nil, ErrTemplateNotFound
	}

	statuses := make([]BulkTemplatedEmailStatus, 0, len(destinations))
	for _, destination := range destinations {
		if len(sanitizeDestinations(destination.Destinations)) == 0 {
			statuses = append(statuses, BulkTemplatedEmailStatus{Status: "Failed", Error: "MessageRejected"})
			continue
		}
		statuses = append(statuses, BulkTemplatedEmailStatus{Status: "Success", MessageID: s.recordSend()})
	}
	return statuses, nil
}

func (s *Service) TestRenderTemplate(templateName, templateData string) (string, error) {
	templateName = strings.TrimSpace(templateName)
	if templateName == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	tpl, ok := s.templates[templateName]
	s.mu.Unlock()
	if !ok {
		return "", ErrTemplateNotFound
	}

	rendered := strings.TrimSpace(tpl.TextPart)
	if rendered == "" {
		rendered = strings.TrimSpace(tpl.HTMLPart)
	}
	if rendered == "" {
		rendered = strings.TrimSpace(tpl.Subject)
	}
	if rendered == "" {
		return "", ErrInvalidParameter
	}

	return renderTemplateData(rendered, templateData), nil
}

func copyReceiptRuleSet(ruleSet *ReceiptRuleSet) ReceiptRuleSet {
	copied := ReceiptRuleSet{
		Name:      ruleSet.Name,
		CreatedAt: ruleSet.CreatedAt,
		Rules:     make([]*ReceiptRule, 0, len(ruleSet.Rules)),
	}
	for _, rule := range ruleSet.Rules {
		ruleCopy := *rule
		ruleCopy.Recipients = append([]string(nil), rule.Recipients...)
		copied.Rules = append(copied.Rules, &ruleCopy)
	}
	return copied
}

func normalizeReceiptRule(rule ReceiptRule) ReceiptRule {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.TLSPolicy = strings.TrimSpace(rule.TLSPolicy)
	rule.Recipients = sanitizeDestinations(rule.Recipients)
	return rule
}

func receiptRuleIndex(rules []*ReceiptRule, ruleName string) int {
	for idx, rule := range rules {
		if strings.EqualFold(strings.TrimSpace(rule.Name), strings.TrimSpace(ruleName)) {
			return idx
		}
	}
	return -1
}

func renderTemplateData(templateBody, rawData string) string {
	rawData = strings.TrimSpace(rawData)
	if rawData == "" {
		return templateBody
	}
	values := map[string]any{}
	if err := json.Unmarshal([]byte(rawData), &values); err != nil {
		return templateBody
	}

	rendered := templateBody
	for key, value := range values {
		placeholder := "{{" + strings.TrimSpace(key) + "}}"
		rendered = strings.ReplaceAll(rendered, placeholder, fmt.Sprint(value))
		placeholderSpaced := "{{ " + strings.TrimSpace(key) + " }}"
		rendered = strings.ReplaceAll(rendered, placeholderSpaced, fmt.Sprint(value))
	}
	return rendered
}
