package sns

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrTopicNotFound         = errors.New("topic not found")
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrPlatformNotFound      = errors.New("platform application not found")
	ErrEndpointNotFound      = errors.New("endpoint not found")
	ErrPhoneNumberNotFound   = errors.New("phone number not found")
	ErrInvalidParameter      = errors.New("invalid parameter")
	ErrSMSSandboxNotFound    = errors.New("sms sandbox phone number not found")
	ErrSMSSandboxPhoneExists = errors.New("sms sandbox phone number exists")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type Topic struct {
	ARN                    string
	Name                   string
	Attributes             map[string]string
	Tags                   map[string]string
	DataProtectionPolicy   string
	SubscriptionsConfirmed int
	SubscriptionsPending   int
}

type Subscription struct {
	ARN        string
	TopicARN   string
	Protocol   string
	Endpoint   string
	Attributes map[string]string
	Status     string
	Token      string
	Owner      string
	CreatedAt  time.Time
}

type PlatformApplication struct {
	ARN        string
	Name       string
	Platform   string
	Attributes map[string]string
	Tags       map[string]string
	CreatedAt  time.Time
}

type Endpoint struct {
	ARN            string
	ApplicationARN string
	Token          string
	CustomUserData string
	Attributes     map[string]string
	Enabled        bool
	CreatedAt      time.Time
	LastUpdatedAt  time.Time
}

type SMSSandboxPhoneNumber struct {
	PhoneNumber string
	Status      string
	CreatedAt   time.Time
}

type PublishInput struct {
	TopicARN    string
	TargetARN   string
	PhoneNumber string
	Message     string
	Subject     string
}

type PublishEntry struct {
	ID       string
	Message  string
	Subject  string
	TopicARN string
}

type PublishBatchResult struct {
	Successful []PublishBatchSuccess
	Failed     []PublishBatchFailure
}

type PublishBatchSuccess struct {
	ID        string
	MessageID string
}

type PublishBatchFailure struct {
	ID      string
	Code    string
	Message string
}

type Service struct {
	mu              sync.Mutex
	seq             uint64
	topics          map[string]*Topic
	subscriptions   map[string]*Subscription
	platformApps    map[string]*PlatformApplication
	endpoints       map[string]*Endpoint
	resourceTags    map[string]map[string]string
	smsSandbox      map[string]*SMSSandboxPhoneNumber
	optedOutNumbers map[string]bool
	smsAttributes   map[string]string
}

func NewService() *Service {
	return &Service{
		topics:          make(map[string]*Topic),
		subscriptions:   make(map[string]*Subscription),
		platformApps:    make(map[string]*PlatformApplication),
		endpoints:       make(map[string]*Endpoint),
		resourceTags:    make(map[string]map[string]string),
		smsSandbox:      make(map[string]*SMSSandboxPhoneNumber),
		optedOutNumbers: make(map[string]bool),
		smsAttributes:   make(map[string]string),
	}
}

func (s *Service) CreateTopic(name string, attributes map[string]string) (Topic, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Topic{}, ErrInvalidParameter
	}
	arn := topicARN(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.topics[arn]; ok {
		return cloneTopic(existing), nil
	}
	topic := &Topic{
		ARN:        arn,
		Name:       name,
		Attributes: cloneAttrs(attributes),
		Tags:       map[string]string{},
	}
	if topic.Attributes == nil {
		topic.Attributes = map[string]string{}
	}
	topic.Attributes["TopicArn"] = arn
	topic.Attributes["Owner"] = DefaultAccountID
	s.topics[arn] = topic
	return cloneTopic(topic), nil
}

func (s *Service) DeleteTopic(arn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.topics[arn]; !ok {
		return ErrTopicNotFound
	}
	delete(s.topics, arn)
	for subArn, sub := range s.subscriptions {
		if sub.TopicARN == arn {
			delete(s.subscriptions, subArn)
		}
	}
	delete(s.resourceTags, arn)
	return nil
}

func (s *Service) ListTopics() []Topic {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Topic, 0, len(s.topics))
	for _, topic := range s.topics {
		out = append(out, cloneTopic(topic))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ARN < out[j].ARN })
	return out
}

func (s *Service) GetTopic(arn string) (Topic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	topic, ok := s.topics[arn]
	if !ok {
		return Topic{}, ErrTopicNotFound
	}
	return cloneTopic(topic), nil
}

func (s *Service) SetTopicAttributes(arn string, attrs map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	topic, ok := s.topics[arn]
	if !ok {
		return ErrTopicNotFound
	}
	if topic.Attributes == nil {
		topic.Attributes = map[string]string{}
	}
	for k, v := range attrs {
		topic.Attributes[k] = v
	}
	if v, ok := attrs["DataProtectionPolicy"]; ok {
		topic.DataProtectionPolicy = v
	}
	return nil
}

func (s *Service) PutDataProtectionPolicy(arn, policy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	topic, ok := s.topics[arn]
	if !ok {
		return ErrTopicNotFound
	}
	topic.DataProtectionPolicy = policy
	return nil
}

func (s *Service) GetDataProtectionPolicy(arn string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	topic, ok := s.topics[arn]
	if !ok {
		return "", ErrTopicNotFound
	}
	return topic.DataProtectionPolicy, nil
}

func (s *Service) Subscribe(topicArn, protocol, endpoint string, attributes map[string]string, requireConfirm bool) (Subscription, error) {
	if topicArn == "" || protocol == "" {
		return Subscription{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.topics[topicArn]; !ok {
		return Subscription{}, ErrTopicNotFound
	}
	id := atomic.AddUint64(&s.seq, 1)
	subArn := fmt.Sprintf("%s:%d", topicArn, id)
	status := "Confirmed"
	token := ""
	if requireConfirm {
		status = "PendingConfirmation"
		token = fmt.Sprintf("token-%d", id)
	}
	sub := &Subscription{
		ARN:        subArn,
		TopicARN:   topicArn,
		Protocol:   protocol,
		Endpoint:   endpoint,
		Attributes: cloneAttrs(attributes),
		Status:     status,
		Token:      token,
		Owner:      DefaultAccountID,
		CreatedAt:  time.Now().UTC(),
	}
	if sub.Attributes == nil {
		sub.Attributes = map[string]string{}
	}
	sub.Attributes["SubscriptionArn"] = subArn
	sub.Attributes["TopicArn"] = topicArn
	sub.Attributes["Protocol"] = protocol
	sub.Attributes["Endpoint"] = endpoint
	sub.Attributes["Owner"] = DefaultAccountID
	sub.Attributes["PendingConfirmation"] = boolString(status == "PendingConfirmation")
	sub.Attributes["ConfirmationWasAuthenticated"] = "false"
	sub.Attributes["RawMessageDelivery"] = "false"
	s.subscriptions[subArn] = sub
	s.refreshTopicCountsLocked(topicArn)
	return cloneSubscription(sub), nil
}

func (s *Service) ConfirmSubscription(token string) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sub := range s.subscriptions {
		if sub.Token == token && sub.Status == "PendingConfirmation" {
			sub.Status = "Confirmed"
			sub.Attributes["PendingConfirmation"] = "false"
			s.refreshTopicCountsLocked(sub.TopicARN)
			return cloneSubscription(sub), nil
		}
	}
	return Subscription{}, ErrSubscriptionNotFound
}

func (s *Service) GetSubscription(arn string) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[arn]
	if !ok {
		return Subscription{}, ErrSubscriptionNotFound
	}
	return cloneSubscription(sub), nil
}

func (s *Service) SetSubscriptionAttributes(arn string, attrs map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[arn]
	if !ok {
		return ErrSubscriptionNotFound
	}
	if sub.Attributes == nil {
		sub.Attributes = map[string]string{}
	}
	for k, v := range attrs {
		sub.Attributes[k] = v
	}
	return nil
}

func (s *Service) ListSubscriptions(topicArn string) []Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Subscription{}
	for _, sub := range s.subscriptions {
		if topicArn != "" && sub.TopicARN != topicArn {
			continue
		}
		out = append(out, cloneSubscription(sub))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ARN < out[j].ARN })
	return out
}

func (s *Service) Unsubscribe(arn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[arn]
	if !ok {
		return ErrSubscriptionNotFound
	}
	delete(s.subscriptions, arn)
	s.refreshTopicCountsLocked(sub.TopicARN)
	return nil
}

func (s *Service) Publish(input PublishInput) (string, error) {
	if input.Message == "" {
		return "", ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if input.TopicARN != "" {
		if _, ok := s.topics[input.TopicARN]; !ok {
			return "", ErrTopicNotFound
		}
	}
	if input.TargetARN != "" {
		if _, ok := s.endpoints[input.TargetARN]; !ok {
			if _, ok := s.subscriptions[input.TargetARN]; !ok {
				if _, ok := s.topics[input.TargetARN]; !ok {
					return "", ErrEndpointNotFound
				}
			}
		}
	}
	id := atomic.AddUint64(&s.seq, 1)
	return fmt.Sprintf("msg-%d", id), nil
}

func (s *Service) PublishBatch(topicArn string, entries []PublishEntry) PublishBatchResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := PublishBatchResult{}
	if topicArn != "" {
		if _, ok := s.topics[topicArn]; !ok {
			for _, entry := range entries {
				result.Failed = append(result.Failed, PublishBatchFailure{
					ID:      entry.ID,
					Code:    "NotFound",
					Message: "topic not found",
				})
			}
			return result
		}
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.ID) == "" || strings.TrimSpace(entry.Message) == "" {
			result.Failed = append(result.Failed, PublishBatchFailure{
				ID:      entry.ID,
				Code:    "InvalidParameter",
				Message: "Id and Message are required",
			})
			continue
		}
		id := atomic.AddUint64(&s.seq, 1)
		result.Successful = append(result.Successful, PublishBatchSuccess{
			ID:        entry.ID,
			MessageID: fmt.Sprintf("msg-%d", id),
		})
	}
	return result
}

func (s *Service) CreatePlatformApplication(name, platform string, attrs map[string]string) (PlatformApplication, error) {
	name = strings.TrimSpace(name)
	platform = strings.TrimSpace(platform)
	if name == "" || platform == "" {
		return PlatformApplication{}, ErrInvalidParameter
	}
	arn := platformApplicationARN(platform, name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.platformApps[arn]; ok {
		return clonePlatformApplication(existing), nil
	}
	app := &PlatformApplication{
		ARN:        arn,
		Name:       name,
		Platform:   platform,
		Attributes: cloneAttrs(attrs),
		Tags:       map[string]string{},
		CreatedAt:  time.Now().UTC(),
	}
	s.platformApps[arn] = app
	return clonePlatformApplication(app), nil
}

func (s *Service) DeletePlatformApplication(arn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.platformApps[arn]; !ok {
		return ErrPlatformNotFound
	}
	delete(s.platformApps, arn)
	for endpointArn, endpoint := range s.endpoints {
		if endpoint.ApplicationARN == arn {
			delete(s.endpoints, endpointArn)
		}
	}
	delete(s.resourceTags, arn)
	return nil
}

func (s *Service) ListPlatformApplications() []PlatformApplication {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PlatformApplication, 0, len(s.platformApps))
	for _, app := range s.platformApps {
		out = append(out, clonePlatformApplication(app))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ARN < out[j].ARN })
	return out
}

func (s *Service) GetPlatformApplication(arn string) (PlatformApplication, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.platformApps[arn]
	if !ok {
		return PlatformApplication{}, ErrPlatformNotFound
	}
	return clonePlatformApplication(app), nil
}

func (s *Service) SetPlatformApplicationAttributes(arn string, attrs map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.platformApps[arn]
	if !ok {
		return ErrPlatformNotFound
	}
	if app.Attributes == nil {
		app.Attributes = map[string]string{}
	}
	for k, v := range attrs {
		app.Attributes[k] = v
	}
	return nil
}

func (s *Service) CreatePlatformEndpoint(appArn, token, customUserData string, attrs map[string]string) (Endpoint, error) {
	if appArn == "" || token == "" {
		return Endpoint{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.platformApps[appArn]; !ok {
		return Endpoint{}, ErrPlatformNotFound
	}
	id := atomic.AddUint64(&s.seq, 1)
	arn := platformEndpointARN(appArn, id)
	endpoint := &Endpoint{
		ARN:            arn,
		ApplicationARN: appArn,
		Token:          token,
		CustomUserData: customUserData,
		Attributes:     cloneAttrs(attrs),
		Enabled:        true,
		CreatedAt:      time.Now().UTC(),
		LastUpdatedAt:  time.Now().UTC(),
	}
	if endpoint.Attributes == nil {
		endpoint.Attributes = map[string]string{}
	}
	endpoint.Attributes["Token"] = token
	endpoint.Attributes["Enabled"] = "true"
	s.endpoints[arn] = endpoint
	return cloneEndpoint(endpoint), nil
}

func (s *Service) DeleteEndpoint(arn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.endpoints[arn]; !ok {
		return ErrEndpointNotFound
	}
	delete(s.endpoints, arn)
	delete(s.resourceTags, arn)
	return nil
}

func (s *Service) ListEndpoints(appArn string) []Endpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Endpoint{}
	for _, endpoint := range s.endpoints {
		if appArn != "" && endpoint.ApplicationARN != appArn {
			continue
		}
		out = append(out, cloneEndpoint(endpoint))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ARN < out[j].ARN })
	return out
}

func (s *Service) GetEndpoint(arn string) (Endpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	endpoint, ok := s.endpoints[arn]
	if !ok {
		return Endpoint{}, ErrEndpointNotFound
	}
	return cloneEndpoint(endpoint), nil
}

func (s *Service) SetEndpointAttributes(arn string, attrs map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	endpoint, ok := s.endpoints[arn]
	if !ok {
		return ErrEndpointNotFound
	}
	if endpoint.Attributes == nil {
		endpoint.Attributes = map[string]string{}
	}
	for k, v := range attrs {
		endpoint.Attributes[k] = v
	}
	if v, ok := attrs["Enabled"]; ok {
		endpoint.Enabled = v != "false"
	}
	endpoint.LastUpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) CreateSMSSandboxPhoneNumber(number string) (SMSSandboxPhoneNumber, error) {
	number = strings.TrimSpace(number)
	if number == "" {
		return SMSSandboxPhoneNumber{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.smsSandbox[number]; ok {
		return SMSSandboxPhoneNumber{}, ErrSMSSandboxPhoneExists
	}
	entry := &SMSSandboxPhoneNumber{
		PhoneNumber: number,
		Status:      "Pending",
		CreatedAt:   time.Now().UTC(),
	}
	s.smsSandbox[number] = entry
	return *entry, nil
}

func (s *Service) VerifySMSSandboxPhoneNumber(number string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.smsSandbox[number]
	if !ok {
		return ErrSMSSandboxNotFound
	}
	entry.Status = "Verified"
	return nil
}

func (s *Service) DeleteSMSSandboxPhoneNumber(number string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.smsSandbox[number]; !ok {
		return ErrSMSSandboxNotFound
	}
	delete(s.smsSandbox, number)
	return nil
}

func (s *Service) ListSMSSandboxPhoneNumbers() []SMSSandboxPhoneNumber {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SMSSandboxPhoneNumber, 0, len(s.smsSandbox))
	for _, entry := range s.smsSandbox {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PhoneNumber < out[j].PhoneNumber })
	return out
}

func (s *Service) IsSMSSandboxEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, entry := range s.smsSandbox {
		if entry.Status == "Verified" {
			return true
		}
	}
	return false
}

func (s *Service) SetSMSAttributes(attrs map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range attrs {
		s.smsAttributes[k] = v
	}
}

func (s *Service) GetSMSAttributes() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneAttrs(s.smsAttributes)
}

func (s *Service) CheckIfPhoneNumberIsOptedOut(number string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.optedOutNumbers[number]
}

func (s *Service) OptInPhoneNumber(number string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.optedOutNumbers, number)
}

func (s *Service) ListPhoneNumbersOptedOut() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.optedOutNumbers))
	for number := range s.optedOutNumbers {
		out = append(out, number)
	}
	sort.Strings(out)
	return out
}

func (s *Service) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = map[string]string{}
	}
	for k, v := range tags {
		s.resourceTags[arn][k] = v
	}
}

func (s *Service) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		if s.resourceTags[arn] != nil {
			delete(s.resourceTags[arn], key)
		}
	}
}

func (s *Service) ListTags(arn string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneAttrs(s.resourceTags[arn])
}

func (s *Service) refreshTopicCountsLocked(topicArn string) {
	topic, ok := s.topics[topicArn]
	if !ok {
		return
	}
	confirmed := 0
	pending := 0
	for _, sub := range s.subscriptions {
		if sub.TopicARN != topicArn {
			continue
		}
		if sub.Status == "Confirmed" {
			confirmed++
		} else if sub.Status == "PendingConfirmation" {
			pending++
		}
	}
	topic.SubscriptionsConfirmed = confirmed
	topic.SubscriptionsPending = pending
	if topic.Attributes == nil {
		topic.Attributes = map[string]string{}
	}
	topic.Attributes["SubscriptionsConfirmed"] = fmt.Sprintf("%d", confirmed)
	topic.Attributes["SubscriptionsPending"] = fmt.Sprintf("%d", pending)
	topic.Attributes["SubscriptionsDeleted"] = "0"
}

func cloneTopic(t *Topic) Topic {
	if t == nil {
		return Topic{}
	}
	cp := *t
	cp.Attributes = cloneAttrs(t.Attributes)
	cp.Tags = cloneAttrs(t.Tags)
	return cp
}

func cloneSubscription(s *Subscription) Subscription {
	if s == nil {
		return Subscription{}
	}
	cp := *s
	cp.Attributes = cloneAttrs(s.Attributes)
	return cp
}

func clonePlatformApplication(a *PlatformApplication) PlatformApplication {
	if a == nil {
		return PlatformApplication{}
	}
	cp := *a
	cp.Attributes = cloneAttrs(a.Attributes)
	cp.Tags = cloneAttrs(a.Tags)
	return cp
}

func cloneEndpoint(e *Endpoint) Endpoint {
	if e == nil {
		return Endpoint{}
	}
	cp := *e
	cp.Attributes = cloneAttrs(e.Attributes)
	return cp
}

func cloneAttrs(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]string, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	return out
}

func topicARN(name string) string {
	return fmt.Sprintf("arn:aws:sns:%s:%s:%s", DefaultRegion, DefaultAccountID, name)
}

func platformApplicationARN(platform, name string) string {
	return fmt.Sprintf("arn:aws:sns:%s:%s:app/%s/%s", DefaultRegion, DefaultAccountID, platform, name)
}

func platformEndpointARN(appArn string, id uint64) string {
	parts := strings.Split(appArn, ":")
	if len(parts) < 6 {
		return fmt.Sprintf("arn:aws:sns:%s:%s:endpoint/unknown/%d", DefaultRegion, DefaultAccountID, id)
	}
	platformParts := strings.Split(parts[5], "/")
	platform := "unknown"
	name := "unknown"
	if len(platformParts) >= 3 {
		platform = platformParts[1]
		name = platformParts[2]
	}
	return fmt.Sprintf("arn:aws:sns:%s:%s:endpoint/%s/%s/%d", DefaultRegion, DefaultAccountID, platform, name, id)
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
