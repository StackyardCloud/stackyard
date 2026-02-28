package ses

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrIdentityNotFound = errors.New("identity not found")
	ErrTemplateExists   = errors.New("template already exists")
	ErrTemplateNotFound = errors.New("template not found")
	ErrMessageRejected  = errors.New("message rejected")
)

const (
	IdentityTypeEmailAddress = "EmailAddress"
	IdentityTypeDomain       = "Domain"

	verificationStatusSuccess = "Success"
	dkimStatusSuccess         = "Success"
	mailFromStatusSuccess     = "Success"

	BehaviorOnMXFailureUseDefaultValue = "UseDefaultValue"
	BehaviorOnMXFailureRejectMessage   = "RejectMessage"
)

type Identity struct {
	Name         string
	IdentityType string

	VerificationToken string
	VerificationState string

	DkimEnabled           bool
	DkimTokens            []string
	DkimVerificationState string

	FeedbackForwardingEnabled              bool
	HeadersInBounceNotificationsEnabled    bool
	HeadersInComplaintNotificationsEnabled bool
	HeadersInDeliveryNotificationsEnabled  bool

	NotificationTopics map[string]string

	MailFromDomain     string
	MailFromDomainStat string
	BehaviorOnMXFail   string
}

type VerificationAttributes struct {
	VerificationStatus string
	VerificationToken  string
}

type IdentityDkimAttributes struct {
	DkimEnabled       bool
	DkimTokens        []string
	VerificationState string
}

type IdentityMailFromDomainAttributes struct {
	BehaviorOnMXFailure string
	MailFromDomain      string
	MailFromDomainState string
}

type IdentityNotificationAttributes struct {
	BounceTopic                     string
	ComplaintTopic                  string
	DeliveryTopic                   string
	ForwardingEnabled               bool
	HeadersInBounceNotifications    bool
	HeadersInComplaintNotifications bool
	HeadersInDeliveryNotifications  bool
}

type Template struct {
	Name      string
	Subject   string
	HTMLPart  string
	TextPart  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TemplateMetadata struct {
	Name      string
	CreatedAt time.Time
}

type SendEmailInput struct {
	Source       string
	Destinations []string
	Subject      string
	TextBody     string
	HTMLBody     string
	Tags         map[string]string
}

type SendRawEmailInput struct {
	Source       string
	Destinations []string
	RawData      []byte
	Tags         map[string]string
}

type SendTemplatedEmailInput struct {
	Source       string
	Destinations []string
	TemplateName string
	TemplateData string
	Tags         map[string]string
}

type SendQuota struct {
	Max24HourSend   float64
	MaxSendRate     float64
	SentLast24Hours float64
}

type SendDataPoint struct {
	Timestamp        time.Time
	DeliveryAttempts int64
	Rejects          int64
	Bounces          int64
	Complaints       int64
}

type sentMessage struct {
	Timestamp time.Time
}

type Service struct {
	mu                    sync.Mutex
	seq                   uint64
	accountSendingEnabled bool
	identities            map[string]*Identity
	templates             map[string]*Template
	sent                  []sentMessage
	configurationSets     map[string]*ConfigurationSet
	customVerifTemplates  map[string]*CustomVerificationEmailTemplate
	receiptFilters        map[string]*ReceiptFilter
	receiptRuleSets       map[string]*ReceiptRuleSet
	activeReceiptRuleSet  string
	identityPolicies      map[string]map[string]string
}

func NewService() *Service {
	return &Service{
		accountSendingEnabled: true,
		identities:            make(map[string]*Identity),
		templates:             make(map[string]*Template),
		sent:                  make([]sentMessage, 0),
		configurationSets:     make(map[string]*ConfigurationSet),
		customVerifTemplates:  make(map[string]*CustomVerificationEmailTemplate),
		receiptFilters:        make(map[string]*ReceiptFilter),
		receiptRuleSets:       make(map[string]*ReceiptRuleSet),
		identityPolicies:      make(map[string]map[string]string),
	}
}

func (s *Service) VerifyEmailIdentity(email string) error {
	email = normalizeIdentity(email)
	if email == "" || !strings.Contains(email, "@") {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry := s.ensureIdentityLocked(email, IdentityTypeEmailAddress)
	entry.VerificationState = verificationStatusSuccess
	entry.VerificationToken = ""
	return nil
}

func (s *Service) VerifyEmailAddress(email string) error {
	return s.VerifyEmailIdentity(email)
}

func (s *Service) VerifyDomainIdentity(domain string) (string, error) {
	domain = normalizeIdentity(domain)
	if domain == "" || strings.Contains(domain, "@") {
		return "", ErrInvalidParameter
	}

	token := verificationToken(domain)

	s.mu.Lock()
	defer s.mu.Unlock()

	entry := s.ensureIdentityLocked(domain, IdentityTypeDomain)
	entry.VerificationState = verificationStatusSuccess
	entry.VerificationToken = token
	entry.DkimTokens = dkimTokens(domain)
	entry.DkimVerificationState = dkimStatusSuccess
	return token, nil
}

func (s *Service) VerifyDomainDkim(domain string) ([]string, error) {
	domain = normalizeIdentity(domain)
	if domain == "" || strings.Contains(domain, "@") {
		return nil, ErrInvalidParameter
	}

	tokens := dkimTokens(domain)
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := s.ensureIdentityLocked(domain, IdentityTypeDomain)
	entry.DkimEnabled = true
	entry.DkimTokens = tokens
	entry.DkimVerificationState = dkimStatusSuccess
	if entry.VerificationState == "" {
		entry.VerificationState = verificationStatusSuccess
	}
	if entry.VerificationToken == "" {
		entry.VerificationToken = verificationToken(domain)
	}
	return append([]string(nil), tokens...), nil
}

func (s *Service) DeleteIdentity(identity string) {
	identity = normalizeIdentity(identity)
	if identity == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.identities, identity)
}

func (s *Service) DeleteVerifiedEmailAddress(email string) {
	email = normalizeIdentity(email)
	if email == "" || !strings.Contains(email, "@") {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.identities, email)
}

func (s *Service) GetIdentityVerificationAttributes(identities []string) map[string]VerificationAttributes {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[string]VerificationAttributes)
	for _, identity := range identities {
		key := normalizeIdentity(identity)
		if key == "" {
			continue
		}
		entry, ok := s.identities[key]
		if !ok {
			continue
		}
		out[entry.Name] = VerificationAttributes{
			VerificationStatus: entry.VerificationState,
			VerificationToken:  entry.VerificationToken,
		}
	}
	return out
}

func (s *Service) GetIdentityDkimAttributes(identities []string) map[string]IdentityDkimAttributes {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[string]IdentityDkimAttributes)
	for _, identity := range identities {
		key := normalizeIdentity(identity)
		entry, ok := s.identities[key]
		if !ok {
			continue
		}
		out[entry.Name] = IdentityDkimAttributes{
			DkimEnabled:       entry.DkimEnabled,
			DkimTokens:        append([]string(nil), entry.DkimTokens...),
			VerificationState: entry.DkimVerificationState,
		}
	}
	return out
}

func (s *Service) GetIdentityMailFromDomainAttributes(identities []string) map[string]IdentityMailFromDomainAttributes {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[string]IdentityMailFromDomainAttributes)
	for _, identity := range identities {
		key := normalizeIdentity(identity)
		entry, ok := s.identities[key]
		if !ok {
			continue
		}
		out[entry.Name] = IdentityMailFromDomainAttributes{
			BehaviorOnMXFailure: entry.BehaviorOnMXFail,
			MailFromDomain:      entry.MailFromDomain,
			MailFromDomainState: entry.MailFromDomainStat,
		}
	}
	return out
}

func (s *Service) GetIdentityNotificationAttributes(identities []string) map[string]IdentityNotificationAttributes {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[string]IdentityNotificationAttributes)
	for _, identity := range identities {
		key := normalizeIdentity(identity)
		entry, ok := s.identities[key]
		if !ok {
			continue
		}
		out[entry.Name] = IdentityNotificationAttributes{
			BounceTopic:                     entry.NotificationTopics["Bounce"],
			ComplaintTopic:                  entry.NotificationTopics["Complaint"],
			DeliveryTopic:                   entry.NotificationTopics["Delivery"],
			ForwardingEnabled:               entry.FeedbackForwardingEnabled,
			HeadersInBounceNotifications:    entry.HeadersInBounceNotificationsEnabled,
			HeadersInComplaintNotifications: entry.HeadersInComplaintNotificationsEnabled,
			HeadersInDeliveryNotifications:  entry.HeadersInDeliveryNotificationsEnabled,
		}
	}
	return out
}

func (s *Service) SetIdentityDkimEnabled(identity string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.identities[normalizeIdentity(identity)]
	if !ok {
		return ErrIdentityNotFound
	}
	entry.DkimEnabled = enabled
	return nil
}

func (s *Service) SetIdentityFeedbackForwardingEnabled(identity string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.identities[normalizeIdentity(identity)]
	if !ok {
		return ErrIdentityNotFound
	}
	entry.FeedbackForwardingEnabled = enabled
	return nil
}

func (s *Service) SetIdentityHeadersInNotificationsEnabled(identity, notificationType string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.identities[normalizeIdentity(identity)]
	if !ok {
		return ErrIdentityNotFound
	}
	switch notificationType {
	case "Bounce":
		entry.HeadersInBounceNotificationsEnabled = enabled
	case "Complaint":
		entry.HeadersInComplaintNotificationsEnabled = enabled
	case "Delivery":
		entry.HeadersInDeliveryNotificationsEnabled = enabled
	default:
		return ErrInvalidParameter
	}
	return nil
}

func (s *Service) SetIdentityNotificationTopic(identity, notificationType, snsTopic string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.identities[normalizeIdentity(identity)]
	if !ok {
		return ErrIdentityNotFound
	}
	switch notificationType {
	case "Bounce", "Complaint", "Delivery":
		entry.NotificationTopics[notificationType] = strings.TrimSpace(snsTopic)
		return nil
	default:
		return ErrInvalidParameter
	}
}

func (s *Service) SetIdentityMailFromDomain(identity, mailFromDomain, behaviorOnMXFailure string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.identities[normalizeIdentity(identity)]
	if !ok {
		return ErrIdentityNotFound
	}
	mailFromDomain = normalizeIdentity(mailFromDomain)
	if behaviorOnMXFailure == "" {
		behaviorOnMXFailure = BehaviorOnMXFailureUseDefaultValue
	}
	if behaviorOnMXFailure != BehaviorOnMXFailureUseDefaultValue && behaviorOnMXFailure != BehaviorOnMXFailureRejectMessage {
		return ErrInvalidParameter
	}
	entry.BehaviorOnMXFail = behaviorOnMXFailure
	entry.MailFromDomain = mailFromDomain
	if mailFromDomain == "" {
		entry.MailFromDomainStat = "Pending"
	} else {
		entry.MailFromDomainStat = mailFromStatusSuccess
	}
	return nil
}

func (s *Service) ListIdentities(identityType, nextToken string, maxItems int) ([]string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedType := strings.TrimSpace(identityType)
	if normalizedType != "" && normalizedType != IdentityTypeEmailAddress && normalizedType != IdentityTypeDomain {
		return nil, "", ErrInvalidParameter
	}
	if maxItems <= 0 {
		maxItems = 100
	}

	items := make([]string, 0, len(s.identities))
	for _, identity := range s.identities {
		if normalizedType != "" && identity.IdentityType != normalizedType {
			continue
		}
		items = append(items, identity.Name)
	}
	sort.Strings(items)

	start, err := parseToken(nextToken)
	if err != nil || start < 0 || start > len(items) {
		return nil, "", ErrInvalidParameter
	}

	end := start + maxItems
	if end > len(items) {
		end = len(items)
	}
	page := append([]string(nil), items[start:end]...)
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return page, next, nil
}

func (s *Service) ListVerifiedEmailAddresses() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]string, 0)
	for _, identity := range s.identities {
		if identity.IdentityType == IdentityTypeEmailAddress && identity.VerificationState == verificationStatusSuccess {
			items = append(items, identity.Name)
		}
	}
	sort.Strings(items)
	return items
}

func (s *Service) CreateTemplate(template Template) error {
	template.Name = strings.TrimSpace(template.Name)
	if template.Name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.templates[template.Name]; ok {
		return ErrTemplateExists
	}
	now := time.Now().UTC()
	s.templates[template.Name] = &Template{
		Name:      template.Name,
		Subject:   template.Subject,
		HTMLPart:  template.HTMLPart,
		TextPart:  template.TextPart,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return nil
}

func (s *Service) UpdateTemplate(template Template) error {
	template.Name = strings.TrimSpace(template.Name)
	if template.Name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.templates[template.Name]
	if !ok {
		return ErrTemplateNotFound
	}
	entry.Subject = template.Subject
	entry.HTMLPart = template.HTMLPart
	entry.TextPart = template.TextPart
	entry.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) DeleteTemplate(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.templates[name]; !ok {
		return ErrTemplateNotFound
	}
	delete(s.templates, name)
	return nil
}

func (s *Service) GetTemplate(name string) (Template, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Template{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	template, ok := s.templates[name]
	if !ok {
		return Template{}, ErrTemplateNotFound
	}
	return *template, nil
}

func (s *Service) ListTemplates(nextToken string, maxItems int) ([]TemplateMetadata, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if maxItems <= 0 {
		maxItems = 10
	}

	all := make([]TemplateMetadata, 0, len(s.templates))
	for _, template := range s.templates {
		all = append(all, TemplateMetadata{Name: template.Name, CreatedAt: template.CreatedAt})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })

	start, err := parseToken(nextToken)
	if err != nil || start < 0 || start > len(all) {
		return nil, "", ErrInvalidParameter
	}
	end := start + maxItems
	if end > len(all) {
		end = len(all)
	}
	page := append([]TemplateMetadata(nil), all[start:end]...)
	next := ""
	if end < len(all) {
		next = strconv.Itoa(end)
	}
	return page, next, nil
}

func (s *Service) SendEmail(input SendEmailInput) (string, error) {
	if !s.GetAccountSendingEnabled() {
		return "", ErrMessageRejected
	}
	if normalizeIdentity(input.Source) == "" || !strings.Contains(input.Source, "@") {
		return "", ErrInvalidParameter
	}
	if len(sanitizeDestinations(input.Destinations)) == 0 {
		return "", ErrInvalidParameter
	}
	if strings.TrimSpace(input.Subject) == "" && strings.TrimSpace(input.TextBody) == "" && strings.TrimSpace(input.HTMLBody) == "" {
		return "", ErrMessageRejected
	}
	return s.recordSend(), nil
}

func (s *Service) SendRawEmail(input SendRawEmailInput) (string, error) {
	if !s.GetAccountSendingEnabled() {
		return "", ErrMessageRejected
	}
	if len(input.RawData) == 0 {
		return "", ErrInvalidParameter
	}
	return s.recordSend(), nil
}

func (s *Service) SendTemplatedEmail(input SendTemplatedEmailInput) (string, error) {
	if !s.GetAccountSendingEnabled() {
		return "", ErrMessageRejected
	}
	if normalizeIdentity(input.Source) == "" || !strings.Contains(input.Source, "@") {
		return "", ErrInvalidParameter
	}
	if len(sanitizeDestinations(input.Destinations)) == 0 {
		return "", ErrInvalidParameter
	}
	templateName := strings.TrimSpace(input.TemplateName)
	if templateName == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	_, ok := s.templates[templateName]
	s.mu.Unlock()
	if !ok {
		return "", ErrTemplateNotFound
	}

	return s.recordSend(), nil
}

func (s *Service) GetAccountSendingEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accountSendingEnabled
}

func (s *Service) UpdateAccountSendingEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountSendingEnabled = enabled
}

func (s *Service) GetSendQuota() SendQuota {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)
	count := 0
	for _, msg := range s.sent {
		if msg.Timestamp.After(cutoff) {
			count++
		}
	}

	return SendQuota{
		Max24HourSend:   200,
		MaxSendRate:     1,
		SentLast24Hours: float64(count),
	}
}

func (s *Service) GetSendStatistics() []SendDataPoint {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)
	attempts := int64(0)
	for _, msg := range s.sent {
		if msg.Timestamp.After(cutoff) {
			attempts++
		}
	}
	return []SendDataPoint{{
		Timestamp:        now,
		DeliveryAttempts: attempts,
		Rejects:          0,
		Bounces:          0,
		Complaints:       0,
	}}
}

func (s *Service) recordSend() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := atomic.AddUint64(&s.seq, 1)
	s.sent = append(s.sent, sentMessage{Timestamp: time.Now().UTC()})
	return fmt.Sprintf("%d-%d@stackyard.local", time.Now().UTC().UnixNano(), id)
}

func (s *Service) ensureIdentityLocked(name, identityType string) *Identity {
	entry, ok := s.identities[name]
	if ok {
		if entry.NotificationTopics == nil {
			entry.NotificationTopics = map[string]string{}
		}
		if entry.BehaviorOnMXFail == "" {
			entry.BehaviorOnMXFail = BehaviorOnMXFailureUseDefaultValue
		}
		if entry.MailFromDomainStat == "" {
			entry.MailFromDomainStat = mailFromStatusSuccess
		}
		if entry.DkimVerificationState == "" {
			entry.DkimVerificationState = dkimStatusSuccess
		}
		return entry
	}

	domain := name
	if identityType == IdentityTypeEmailAddress {
		parts := strings.Split(name, "@")
		if len(parts) == 2 {
			domain = parts[1]
		}
	}
	entry = &Identity{
		Name:                                   name,
		IdentityType:                           identityType,
		VerificationState:                      verificationStatusSuccess,
		DkimEnabled:                            true,
		DkimTokens:                             dkimTokens(domain),
		DkimVerificationState:                  dkimStatusSuccess,
		FeedbackForwardingEnabled:              true,
		HeadersInBounceNotificationsEnabled:    false,
		HeadersInComplaintNotificationsEnabled: false,
		HeadersInDeliveryNotificationsEnabled:  false,
		NotificationTopics:                     map[string]string{},
		MailFromDomain:                         domain,
		MailFromDomainStat:                     mailFromStatusSuccess,
		BehaviorOnMXFail:                       BehaviorOnMXFailureUseDefaultValue,
	}
	if identityType == IdentityTypeDomain {
		entry.VerificationToken = verificationToken(name)
	}
	s.identities[name] = entry
	return entry
}

func verificationToken(domain string) string {
	sum := sha1.Sum([]byte(domain))
	return hex.EncodeToString(sum[:])
}

func dkimTokens(domain string) []string {
	if domain == "" {
		return nil
	}
	out := make([]string, 0, 3)
	for i := 1; i <= 3; i++ {
		sum := sha1.Sum([]byte(domain + ":dkim:" + strconv.Itoa(i)))
		out = append(out, hex.EncodeToString(sum[:8]))
	}
	return out
}

func normalizeIdentity(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sanitizeDestinations(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func parseToken(token string) (int, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(token)
	if err != nil {
		return 0, err
	}
	return value, nil
}
