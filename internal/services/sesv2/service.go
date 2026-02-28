package sesv2

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrNotFound         = errors.New("resource not found")
	ErrSendingDisabled  = errors.New("account sending is disabled")
)

type Identity struct {
	EmailIdentity            string
	IdentityType             string
	VerificationStatus       string
	VerifiedForSendingStatus bool
	ConfigurationSetName     string
	FeedbackForwardingStatus bool
	DkimSigningEnabled       bool
	DkimSigningAttributes    map[string]any
	DkimStatus               string
	DkimTokens               []string
	MailFromDomain           string
	MailFromDomainStatus     string
	BehaviorOnMxFailure      string
	Tags                     map[string]string
	Policies                 map[string]string
	CreatedAt                time.Time
	LastUpdatedAt            time.Time
}

type EmailTemplate struct {
	TemplateName string
	Subject      string
	Text         string
	HTML         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ConfigurationSet struct {
	Name              string
	EventDestinations map[string]map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type SuppressedDestination struct {
	EmailAddress   string
	Reason         string
	LastUpdateTime time.Time
}

type Contact struct {
	EmailAddress     string
	AttributesData   string
	UnsubscribeAll   bool
	TopicPreferences []map[string]string
	CreatedAt        time.Time
	LastUpdatedAt    time.Time
}

type ContactList struct {
	Name          string
	Description   string
	CreatedAt     time.Time
	LastUpdatedAt time.Time
	Contacts      map[string]*Contact
}

type SendQuota struct {
	Max24HourSend   float64
	MaxSendRate     float64
	SentLast24Hours float64
}

type Account struct {
	SendingEnabled               bool
	ProductionAccessEnabled      bool
	DedicatedIpAutoWarmupEnabled bool
	SuppressedReasons            []string
	SendQuota                    SendQuota
}

type BulkEmailEntryResult struct {
	Status    string
	MessageID string
}

type Service struct {
	mu           sync.Mutex
	seq          uint64
	identities   map[string]*Identity
	templates    map[string]*EmailTemplate
	configSets   map[string]*ConfigurationSet
	suppressed   map[string]*SuppressedDestination
	resourceTags map[string]map[string]string
	contactLists map[string]*ContactList
	sent         []time.Time
	account      Account
}

func NewService() *Service {
	return &Service{
		identities:   map[string]*Identity{},
		templates:    map[string]*EmailTemplate{},
		configSets:   map[string]*ConfigurationSet{},
		suppressed:   map[string]*SuppressedDestination{},
		resourceTags: map[string]map[string]string{},
		contactLists: map[string]*ContactList{},
		sent:         []time.Time{},
		account: Account{
			SendingEnabled:               true,
			ProductionAccessEnabled:      true,
			DedicatedIpAutoWarmupEnabled: false,
			SuppressedReasons:            []string{"BOUNCE", "COMPLAINT"},
			SendQuota: SendQuota{
				Max24HourSend: 200,
				MaxSendRate:   1,
			},
		},
	}
}

func (s *Service) CreateEmailIdentity(identity, configSet string, tags map[string]string) (Identity, error) {
	norm := normalizeEmailIdentity(identity)
	if norm == "" {
		return Identity{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	existing, ok := s.identities[norm]
	if ok {
		if configSet != "" {
			existing.ConfigurationSetName = configSet
		}
		if len(tags) != 0 {
			existing.Tags = mergeTags(existing.Tags, tags)
		}
		existing.LastUpdatedAt = now
		return cloneIdentity(*existing), nil
	}

	idType := "EMAIL_ADDRESS"
	if !strings.Contains(norm, "@") {
		idType = "DOMAIN"
	}

	entry := &Identity{
		EmailIdentity:            norm,
		IdentityType:             idType,
		VerificationStatus:       "SUCCESS",
		VerifiedForSendingStatus: true,
		ConfigurationSetName:     configSet,
		FeedbackForwardingStatus: true,
		DkimSigningEnabled:       true,
		DkimStatus:               "SUCCESS",
		DkimTokens:               deterministicTokens(norm),
		MailFromDomainStatus:     "SUCCESS",
		BehaviorOnMxFailure:      "USE_DEFAULT_VALUE",
		Tags:                     cloneStringMap(tags),
		Policies:                 map[string]string{},
		CreatedAt:                now,
		LastUpdatedAt:            now,
	}
	s.identities[norm] = entry
	return cloneIdentity(*entry), nil
}

func (s *Service) GetEmailIdentity(identity string) (Identity, bool) {
	norm := normalizeEmailIdentity(identity)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.identities[norm]
	if !ok {
		return Identity{}, false
	}
	return cloneIdentity(*entry), true
}

func (s *Service) ListEmailIdentities() []Identity {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Identity, 0, len(s.identities))
	for _, entry := range s.identities {
		out = append(out, cloneIdentity(*entry))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].EmailIdentity < out[j].EmailIdentity
	})
	return out
}

func (s *Service) DeleteEmailIdentity(identity string) bool {
	norm := normalizeEmailIdentity(identity)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.identities[norm]; !ok {
		return false
	}
	delete(s.identities, norm)
	return true
}

func (s *Service) PutEmailIdentityConfigurationSet(identity, configSet string) error {
	if configSet == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.identities[normalizeEmailIdentity(identity)]
	if !ok {
		return ErrNotFound
	}
	entry.ConfigurationSetName = configSet
	entry.LastUpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) PutEmailIdentityFeedback(identity string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.identities[normalizeEmailIdentity(identity)]
	if !ok {
		return ErrNotFound
	}
	entry.FeedbackForwardingStatus = enabled
	entry.LastUpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) PutEmailIdentityDkim(identity string, signingEnabled bool, attrs map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.identities[normalizeEmailIdentity(identity)]
	if !ok {
		return ErrNotFound
	}
	entry.DkimSigningEnabled = signingEnabled
	if attrs != nil {
		entry.DkimSigningAttributes = cloneAnyMap(attrs)
	}
	entry.LastUpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) PutEmailIdentityMailFrom(identity, domain, behavior string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.identities[normalizeEmailIdentity(identity)]
	if !ok {
		return ErrNotFound
	}
	entry.MailFromDomain = strings.TrimSpace(domain)
	if strings.TrimSpace(behavior) != "" {
		entry.BehaviorOnMxFailure = strings.TrimSpace(behavior)
	}
	entry.LastUpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) CreateEmailIdentityPolicy(identity, policyName, policy string) error {
	identity = normalizeEmailIdentity(identity)
	policyName = strings.TrimSpace(policyName)
	if identity == "" || policyName == "" || strings.TrimSpace(policy) == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.identities[identity]
	if !ok {
		return ErrNotFound
	}
	entry.Policies[policyName] = policy
	entry.LastUpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) DeleteEmailIdentityPolicy(identity, policyName string) bool {
	identity = normalizeEmailIdentity(identity)
	policyName = strings.TrimSpace(policyName)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.identities[identity]
	if !ok {
		return false
	}
	if _, ok := entry.Policies[policyName]; !ok {
		return false
	}
	delete(entry.Policies, policyName)
	entry.LastUpdatedAt = time.Now().UTC()
	return true
}

func (s *Service) GetEmailIdentityPolicies(identity string) (map[string]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.identities[normalizeEmailIdentity(identity)]
	if !ok {
		return nil, false
	}
	return cloneStringMap(entry.Policies), true
}

func (s *Service) CreateEmailTemplate(name, subject, text, html string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.templates[name]; ok {
		return ErrAlreadyExists
	}
	now := time.Now().UTC()
	s.templates[name] = &EmailTemplate{
		TemplateName: name,
		Subject:      subject,
		Text:         text,
		HTML:         html,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return nil
}

func (s *Service) UpdateEmailTemplate(name, subject, text, html string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.templates[name]
	if !ok {
		return ErrNotFound
	}
	entry.Subject = subject
	entry.Text = text
	entry.HTML = html
	entry.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) GetEmailTemplate(name string) (EmailTemplate, bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.templates[name]
	if !ok {
		return EmailTemplate{}, false
	}
	return *entry, true
}

func (s *Service) DeleteEmailTemplate(name string) bool {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.templates[name]; !ok {
		return false
	}
	delete(s.templates, name)
	return true
}

func (s *Service) ListEmailTemplates() []EmailTemplate {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]EmailTemplate, 0, len(s.templates))
	for _, tpl := range s.templates {
		out = append(out, *tpl)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TemplateName < out[j].TemplateName
	})
	return out
}

func (s *Service) RenderEmailTemplate(name, templateData string) (string, error) {
	tpl, ok := s.GetEmailTemplate(name)
	if !ok {
		return "", ErrNotFound
	}
	content := tpl.Text
	if strings.TrimSpace(content) == "" {
		content = tpl.HTML
	}
	if strings.TrimSpace(content) == "" {
		content = tpl.Subject
	}
	if strings.TrimSpace(templateData) == "" {
		return content, nil
	}

	payload := map[string]string{}
	if err := json.Unmarshal([]byte(templateData), &payload); err != nil {
		return "", ErrInvalidParameter
	}
	for k, v := range payload {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
		content = strings.ReplaceAll(content, "{{ "+k+" }}", v)
	}
	return content, nil
}

func (s *Service) CreateConfigurationSet(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.configSets[name]; ok {
		return ErrAlreadyExists
	}
	now := time.Now().UTC()
	s.configSets[name] = &ConfigurationSet{
		Name:              name,
		EventDestinations: map[string]map[string]any{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	return nil
}

func (s *Service) GetConfigurationSet(name string) (ConfigurationSet, bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configSets[name]
	if !ok {
		return ConfigurationSet{}, false
	}
	return cloneConfigurationSet(*cfg), true
}

func (s *Service) ListConfigurationSets() []ConfigurationSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ConfigurationSet, 0, len(s.configSets))
	for _, cfg := range s.configSets {
		out = append(out, cloneConfigurationSet(*cfg))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) DeleteConfigurationSet(name string) bool {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.configSets[name]; !ok {
		return false
	}
	delete(s.configSets, name)
	return true
}

func (s *Service) UpsertConfigurationSetEventDestination(configSet, destination string, payload map[string]any) error {
	configSet = strings.TrimSpace(configSet)
	destination = strings.TrimSpace(destination)
	if configSet == "" || destination == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configSets[configSet]
	if !ok {
		return ErrNotFound
	}
	cfg.EventDestinations[destination] = cloneAnyMap(payload)
	cfg.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) DeleteConfigurationSetEventDestination(configSet, destination string) bool {
	configSet = strings.TrimSpace(configSet)
	destination = strings.TrimSpace(destination)
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configSets[configSet]
	if !ok {
		return false
	}
	if _, ok := cfg.EventDestinations[destination]; !ok {
		return false
	}
	delete(cfg.EventDestinations, destination)
	cfg.UpdatedAt = time.Now().UTC()
	return true
}

func (s *Service) GetConfigurationSetEventDestinations(configSet string) (map[string]map[string]any, bool) {
	configSet = strings.TrimSpace(configSet)
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configSets[configSet]
	if !ok {
		return nil, false
	}
	out := map[string]map[string]any{}
	for k, v := range cfg.EventDestinations {
		out[k] = cloneAnyMap(v)
	}
	return out, true
}

func (s *Service) PutSuppressedDestination(email, reason string) (SuppressedDestination, error) {
	email = normalizeEmailIdentity(email)
	if email == "" || !strings.Contains(email, "@") {
		return SuppressedDestination{}, ErrInvalidParameter
	}
	reason = strings.ToUpper(strings.TrimSpace(reason))
	if reason == "" {
		reason = "BOUNCE"
	}
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	entry := &SuppressedDestination{EmailAddress: email, Reason: reason, LastUpdateTime: now}
	s.suppressed[email] = entry
	return *entry, nil
}

func (s *Service) GetSuppressedDestination(email string) (SuppressedDestination, bool) {
	email = normalizeEmailIdentity(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.suppressed[email]
	if !ok {
		return SuppressedDestination{}, false
	}
	return *entry, true
}

func (s *Service) ListSuppressedDestinations() []SuppressedDestination {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SuppressedDestination, 0, len(s.suppressed))
	for _, entry := range s.suppressed {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].EmailAddress < out[j].EmailAddress
	})
	return out
}

func (s *Service) DeleteSuppressedDestination(email string) bool {
	email = normalizeEmailIdentity(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.suppressed[email]; !ok {
		return false
	}
	delete(s.suppressed, email)
	return true
}

func (s *Service) CreateContactList(name, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.contactLists[name]; ok {
		return ErrAlreadyExists
	}
	now := time.Now().UTC()
	s.contactLists[name] = &ContactList{
		Name:          name,
		Description:   description,
		CreatedAt:     now,
		LastUpdatedAt: now,
		Contacts:      map[string]*Contact{},
	}
	return nil
}

func (s *Service) UpdateContactList(name, description string) error {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	list, ok := s.contactLists[name]
	if !ok {
		return ErrNotFound
	}
	list.Description = description
	list.LastUpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) GetContactList(name string) (ContactList, bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	list, ok := s.contactLists[name]
	if !ok {
		return ContactList{}, false
	}
	return cloneContactList(*list), true
}

func (s *Service) ListContactLists() []ContactList {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ContactList, 0, len(s.contactLists))
	for _, list := range s.contactLists {
		out = append(out, cloneContactList(*list))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) DeleteContactList(name string) bool {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.contactLists[name]; !ok {
		return false
	}
	delete(s.contactLists, name)
	return true
}

func (s *Service) CreateContact(listName, email, attributesData string, unsubscribeAll bool, topicPreferences []map[string]string) error {
	listName = strings.TrimSpace(listName)
	email = normalizeEmailIdentity(email)
	if listName == "" || email == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list, ok := s.contactLists[listName]
	if !ok {
		return ErrNotFound
	}
	if _, ok := list.Contacts[email]; ok {
		return ErrAlreadyExists
	}
	now := time.Now().UTC()
	list.Contacts[email] = &Contact{
		EmailAddress:     email,
		AttributesData:   attributesData,
		UnsubscribeAll:   unsubscribeAll,
		TopicPreferences: cloneTopicPreferences(topicPreferences),
		CreatedAt:        now,
		LastUpdatedAt:    now,
	}
	list.LastUpdatedAt = now
	return nil
}

func (s *Service) UpdateContact(listName, email, attributesData string, unsubscribeAll bool, topicPreferences []map[string]string) error {
	listName = strings.TrimSpace(listName)
	email = normalizeEmailIdentity(email)
	if listName == "" || email == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list, ok := s.contactLists[listName]
	if !ok {
		return ErrNotFound
	}
	contact, ok := list.Contacts[email]
	if !ok {
		return ErrNotFound
	}
	contact.AttributesData = attributesData
	contact.UnsubscribeAll = unsubscribeAll
	contact.TopicPreferences = cloneTopicPreferences(topicPreferences)
	contact.LastUpdatedAt = time.Now().UTC()
	list.LastUpdatedAt = contact.LastUpdatedAt
	return nil
}

func (s *Service) GetContact(listName, email string) (Contact, bool) {
	listName = strings.TrimSpace(listName)
	email = normalizeEmailIdentity(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	list, ok := s.contactLists[listName]
	if !ok {
		return Contact{}, false
	}
	contact, ok := list.Contacts[email]
	if !ok {
		return Contact{}, false
	}
	return cloneContact(*contact), true
}

func (s *Service) ListContacts(listName string) ([]Contact, bool) {
	listName = strings.TrimSpace(listName)
	s.mu.Lock()
	defer s.mu.Unlock()
	list, ok := s.contactLists[listName]
	if !ok {
		return nil, false
	}
	out := make([]Contact, 0, len(list.Contacts))
	for _, contact := range list.Contacts {
		out = append(out, cloneContact(*contact))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].EmailAddress < out[j].EmailAddress
	})
	return out, true
}

func (s *Service) DeleteContact(listName, email string) bool {
	listName = strings.TrimSpace(listName)
	email = normalizeEmailIdentity(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	list, ok := s.contactLists[listName]
	if !ok {
		return false
	}
	if _, ok := list.Contacts[email]; !ok {
		return false
	}
	delete(list.Contacts, email)
	list.LastUpdatedAt = time.Now().UTC()
	return true
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) error {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.resourceTags[resourceARN]
	s.resourceTags[resourceARN] = mergeTags(existing, tags)
	return nil
}

func (s *Service) UntagResource(resourceARN string, tagKeys []string) {
	resourceARN = strings.TrimSpace(resourceARN)
	s.mu.Lock()
	defer s.mu.Unlock()
	tags := s.resourceTags[resourceARN]
	for _, key := range tagKeys {
		delete(tags, key)
	}
	if len(tags) == 0 {
		delete(s.resourceTags, resourceARN)
		return
	}
	s.resourceTags[resourceARN] = tags
}

func (s *Service) ListTags(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneStringMap(s.resourceTags[resourceARN])
}

func (s *Service) PutAccountSendingAttributes(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account.SendingEnabled = enabled
}

func (s *Service) PutAccountSuppressionAttributes(reasons []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(reasons) == 0 {
		s.account.SuppressedReasons = []string{}
		return
	}
	vals := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		reason = strings.ToUpper(strings.TrimSpace(reason))
		if reason == "" {
			continue
		}
		vals = append(vals, reason)
	}
	s.account.SuppressedReasons = vals
}

func (s *Service) GetAccount() Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc := s.account
	acc.SuppressedReasons = append([]string(nil), s.account.SuppressedReasons...)
	acc.SendQuota.SentLast24Hours = s.sentInLast24HoursLocked()
	return acc
}

func (s *Service) SendEmail(from string, recipients []string) (string, error) {
	if strings.TrimSpace(from) == "" {
		return "", ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.account.SendingEnabled {
		return "", ErrSendingDisabled
	}
	if len(recipients) == 0 {
		return "", ErrInvalidParameter
	}
	id := fmt.Sprintf("sesv2-%d", atomic.AddUint64(&s.seq, 1))
	s.sent = append(s.sent, time.Now().UTC())
	return id, nil
}

func (s *Service) SendBulkEmail(entries int) ([]BulkEmailEntryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.account.SendingEnabled {
		return nil, ErrSendingDisabled
	}
	if entries <= 0 {
		entries = 1
	}
	results := make([]BulkEmailEntryResult, 0, entries)
	for i := 0; i < entries; i++ {
		id := fmt.Sprintf("sesv2-bulk-%d", atomic.AddUint64(&s.seq, 1))
		s.sent = append(s.sent, time.Now().UTC())
		results = append(results, BulkEmailEntryResult{Status: "SUCCESS", MessageID: id})
	}
	return results, nil
}

func (s *Service) sentInLast24HoursLocked() float64 {
	if len(s.sent) == 0 {
		return 0
	}
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	idx := 0
	for idx < len(s.sent) && s.sent[idx].Before(cutoff) {
		idx++
	}
	if idx > 0 {
		s.sent = append([]time.Time(nil), s.sent[idx:]...)
	}
	return float64(len(s.sent))
}

func normalizeEmailIdentity(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func deterministicTokens(identity string) []string {
	seed := sha1.Sum([]byte(identity))
	hexSeed := hex.EncodeToString(seed[:])
	return []string{hexSeed[0:16], hexSeed[8:24], hexSeed[16:32]}
}

func cloneIdentity(v Identity) Identity {
	v.Tags = cloneStringMap(v.Tags)
	v.Policies = cloneStringMap(v.Policies)
	v.DkimTokens = append([]string(nil), v.DkimTokens...)
	v.DkimSigningAttributes = cloneAnyMap(v.DkimSigningAttributes)
	return v
}

func cloneContact(v Contact) Contact {
	v.TopicPreferences = cloneTopicPreferences(v.TopicPreferences)
	return v
}

func cloneContactList(v ContactList) ContactList {
	original := v.Contacts
	v.Contacts = map[string]*Contact{}
	for k, c := range original {
		clone := cloneContact(*c)
		v.Contacts[k] = &clone
	}
	return v
}

func cloneConfigurationSet(v ConfigurationSet) ConfigurationSet {
	v.EventDestinations = map[string]map[string]any{}
	for name, payload := range v.EventDestinations {
		v.EventDestinations[name] = cloneAnyMap(payload)
	}
	return v
}

func mergeTags(existing, incoming map[string]string) map[string]string {
	out := cloneStringMap(existing)
	for k, v := range incoming {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneTopicPreferences(in []map[string]string) []map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]string, 0, len(in))
	for _, item := range in {
		out = append(out, cloneStringMap(item))
	}
	return out
}
