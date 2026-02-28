package sqs

import (
	"encoding/json"
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
	ErrQueueExists    = errors.New("queue already exists")
	ErrQueueNotFound  = errors.New("queue not found")
	ErrReceiptInvalid = errors.New("receipt handle invalid")
)

type Queue struct {
	Name       string            `json:"name"`
	CreatedAt  time.Time         `json:"created_at"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type MessageAttributeValue struct {
	DataType    string `json:"data_type"`
	StringValue string `json:"string_value,omitempty"`
	BinaryValue string `json:"binary_value,omitempty"`
}

type Message struct {
	ID                string                           `json:"id"`
	Queue             string                           `json:"queue"`
	Body              string                           `json:"body"`
	Attributes        map[string]string                `json:"attributes,omitempty"`
	MessageAttributes map[string]MessageAttributeValue `json:"message_attributes,omitempty"`
	SentAt            time.Time                        `json:"sent_at"`
	ReceiptHandle     string                           `json:"receipt_handle"`
	ReceiveCount      int                              `json:"receive_count,omitempty"`
	SequenceNumber    string                           `json:"sequence_number,omitempty"`
	FirstReceivedAt   time.Time                        `json:"first_received_at,omitempty"`
}

type Service struct {
	mu     sync.Mutex
	seq    uint64
	queues map[string]*queueState
	tasks  map[string]*moveTask
}

type queueState struct {
	queue    Queue
	messages []*messageState
	inflight map[string]*messageState
}

type messageState struct {
	id                string
	queue             string
	body              string
	attributes        map[string]string
	messageAttributes map[string]MessageAttributeValue
	sentAt            time.Time
	visibleAt         time.Time
	receiveCount      int
	receipt           string
	firstReceivedAt   time.Time
	sequenceNumber    string
}

type moveTask struct {
	Handle    string
	Source    string
	Dest      string
	Status    string
	CreatedAt time.Time
}

func NewService() *Service {
	return &Service{queues: make(map[string]*queueState), tasks: make(map[string]*moveTask)}
}

func (s *Service) CreateQueue(name string, attributes map[string]string) (Queue, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Queue{}, ErrQueueNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.queues[name]; ok {
		if attrsEqual(existing.queue.Attributes, attributes) {
			return existing.queue, nil
		}
		return Queue{}, ErrQueueExists
	}

	q := Queue{Name: name, CreatedAt: time.Now().UTC(), Attributes: cloneAttrs(attributes), Tags: map[string]string{}}
	s.queues[name] = &queueState{queue: q, inflight: make(map[string]*messageState)}
	return q, nil
}

func (s *Service) GetQueue(name string) (Queue, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Queue{}, ErrQueueNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.queues[name]
	if !ok {
		return Queue{}, ErrQueueNotFound
	}
	return state.queue, nil
}

func (s *Service) DeleteQueue(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrQueueNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.queues[name]; !ok {
		return ErrQueueNotFound
	}
	delete(s.queues, name)
	return nil
}

func (s *Service) PurgeQueue(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrQueueNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.queues[name]
	if !ok {
		return ErrQueueNotFound
	}
	state.messages = nil
	state.inflight = make(map[string]*messageState)
	return nil
}

func (s *Service) GetQueueAttributes(name string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.queues[name]
	if !ok {
		return nil, ErrQueueNotFound
	}
	return cloneAttrs(state.queue.Attributes), nil
}

func (s *Service) SetQueueAttributes(name string, attrs map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.queues[name]
	if !ok {
		return ErrQueueNotFound
	}
	if state.queue.Attributes == nil {
		state.queue.Attributes = map[string]string{}
	}
	for k, v := range attrs {
		state.queue.Attributes[k] = v
	}
	return nil
}

func (s *Service) ListQueueTags(name string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.queues[name]
	if !ok {
		return nil, ErrQueueNotFound
	}
	return cloneAttrs(state.queue.Tags), nil
}

func (s *Service) TagQueue(name string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.queues[name]
	if !ok {
		return ErrQueueNotFound
	}
	if state.queue.Tags == nil {
		state.queue.Tags = map[string]string{}
	}
	for k, v := range tags {
		state.queue.Tags[k] = v
	}
	return nil
}

func (s *Service) UntagQueue(name string, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.queues[name]
	if !ok {
		return ErrQueueNotFound
	}
	for _, key := range keys {
		delete(state.queue.Tags, key)
	}
	return nil
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

func cloneMessageAttributes(attrs map[string]MessageAttributeValue) map[string]MessageAttributeValue {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]MessageAttributeValue, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	return out
}

func attrsEqual(a, b map[string]string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func (s *Service) ListQueues() []Queue {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Queue, 0, len(s.queues))
	for _, state := range s.queues {
		out = append(out, state.queue)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) SendMessage(queue, body string, attributes map[string]string, messageAttributes map[string]MessageAttributeValue) (Message, error) {
	return s.SendMessageWithDelay(queue, body, attributes, messageAttributes, 0)
}

func (s *Service) SendMessageWithDelay(queue, body string, attributes map[string]string, messageAttributes map[string]MessageAttributeValue, delaySeconds int) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.queues[queue]
	if !ok {
		return Message{}, ErrQueueNotFound
	}

	msg := s.addMessageLocked(state, body, attributes, messageAttributes, delaySeconds)
	return Message{
		ID:                msg.id,
		Queue:             msg.queue,
		Body:              msg.body,
		Attributes:        cloneAttrs(msg.attributes),
		MessageAttributes: cloneMessageAttributes(msg.messageAttributes),
		SentAt:            msg.sentAt,
		SequenceNumber:    msg.sequenceNumber,
	}, nil
}

func (s *Service) ReceiveMessages(queue string, maxMessages int) ([]Message, error) {
	return s.ReceiveMessagesWithVisibility(queue, maxMessages, 30)
}

func (s *Service) ReceiveMessagesWithVisibility(queue string, maxMessages int, visibilityTimeout int) ([]Message, error) {
	if maxMessages <= 0 {
		maxMessages = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.queues[queue]
	if !ok {
		return nil, ErrQueueNotFound
	}

	now := time.Now().UTC()
	picked := make([]Message, 0, maxMessages)
	for _, msg := range state.messages {
		if len(picked) >= maxMessages {
			break
		}
		if msg.visibleAt.After(now) {
			continue
		}
		msg.receiveCount++
		if msg.receiveCount == 1 {
			msg.firstReceivedAt = now
		}
		if dlqName, maxCount := parseRedrivePolicy(state.queue.Attributes); dlqName != "" && maxCount > 0 {
			if msg.receiveCount > maxCount {
				if dlqState, ok := s.queues[dlqName]; ok {
					s.addMessageLocked(dlqState, msg.body, msg.attributes, msg.messageAttributes, 0)
				}
				s.dropMessageLocked(state, msg)
				continue
			}
		}
		msg.receipt = fmt.Sprintf("rh-%s-%d", msg.id, msg.receiveCount)
		state.inflight[msg.receipt] = msg
		vt := visibilityTimeout
		if vt <= 0 {
			vt = 0
		}
		msg.visibleAt = now.Add(time.Duration(vt) * time.Second)
		picked = append(picked, Message{
			ID:                msg.id,
			Queue:             msg.queue,
			Body:              msg.body,
			Attributes:        cloneAttrs(msg.attributes),
			MessageAttributes: cloneMessageAttributes(msg.messageAttributes),
			SentAt:            msg.sentAt,
			ReceiptHandle:     msg.receipt,
			ReceiveCount:      msg.receiveCount,
			SequenceNumber:    msg.sequenceNumber,
			FirstReceivedAt:   msg.firstReceivedAt,
		})
	}
	return picked, nil
}

func (s *Service) DeleteMessage(queue, receipt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.queues[queue]
	if !ok {
		return ErrQueueNotFound
	}
	msg := state.inflight[receipt]
	if msg == nil {
		return ErrReceiptInvalid
	}
	delete(state.inflight, receipt)
	for i, candidate := range state.messages {
		if candidate == msg {
			state.messages = append(state.messages[:i], state.messages[i+1:]...)
			break
		}
	}
	return nil
}

func (s *Service) ChangeMessageVisibility(queue, receipt string, visibilityTimeout int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.queues[queue]
	if !ok {
		return ErrQueueNotFound
	}
	msg := state.inflight[receipt]
	if msg == nil {
		return ErrReceiptInvalid
	}
	now := time.Now().UTC()
	msg.visibleAt = now.Add(time.Duration(visibilityTimeout) * time.Second)
	return nil
}

func (s *Service) StartMessageMoveTask(source, dest string, maxMessages int) (moveTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.queues[source]; !ok {
		return moveTask{}, ErrQueueNotFound
	}
	if dest != "" {
		if _, ok := s.queues[dest]; !ok {
			return moveTask{}, ErrQueueNotFound
		}
	}
	id := atomic.AddUint64(&s.seq, 1)
	task := &moveTask{
		Handle:    fmt.Sprintf("mt-%d", id),
		Source:    source,
		Dest:      dest,
		Status:    "COMPLETED",
		CreatedAt: time.Now().UTC(),
	}
	s.tasks[task.Handle] = task
	return *task, nil
}

func (s *Service) CancelMessageMoveTask(handle string) (moveTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task := s.tasks[handle]
	if task == nil {
		return moveTask{}, ErrReceiptInvalid
	}
	if task.Status == "RUNNING" {
		task.Status = "CANCELED"
	}
	return *task, nil
}

func (s *Service) ListMessageMoveTasks(source string) []moveTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []moveTask{}
	for _, task := range s.tasks {
		if source != "" && task.Source != source {
			continue
		}
		out = append(out, *task)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

type redrivePolicy struct {
	DeadLetterTargetArn string `json:"deadLetterTargetArn"`
	MaxReceiveCount     string `json:"maxReceiveCount"`
}

func parseRedrivePolicy(attrs map[string]string) (string, int) {
	if len(attrs) == 0 {
		return "", 0
	}
	raw := strings.TrimSpace(attrs["RedrivePolicy"])
	if raw == "" {
		return "", 0
	}
	var policy redrivePolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return "", 0
	}
	maxCount := 0
	if policy.MaxReceiveCount != "" {
		if v, err := strconv.Atoi(policy.MaxReceiveCount); err == nil {
			maxCount = v
		}
	}
	if policy.DeadLetterTargetArn == "" {
		return "", 0
	}
	name := queueNameFromArn(policy.DeadLetterTargetArn)
	if name == "" {
		return "", 0
	}
	return name, maxCount
}

func queueNameFromArn(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func (s *Service) addMessageLocked(state *queueState, body string, attributes map[string]string, messageAttributes map[string]MessageAttributeValue, delaySeconds int) *messageState {
	id := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	visibleAt := now
	if delaySeconds > 0 {
		visibleAt = now.Add(time.Duration(delaySeconds) * time.Second)
	}
	msg := &messageState{
		id:                fmt.Sprintf("msg-%d", id),
		queue:             state.queue.Name,
		body:              body,
		attributes:        cloneAttrs(attributes),
		messageAttributes: cloneMessageAttributes(messageAttributes),
		sentAt:            now,
		visibleAt:         visibleAt,
	}
	if strings.HasSuffix(state.queue.Name, ".fifo") {
		msg.sequenceNumber = fmt.Sprintf("%d", id)
	}
	state.messages = append(state.messages, msg)
	return msg
}

func (s *Service) dropMessageLocked(state *queueState, msg *messageState) {
	for receipt, inflight := range state.inflight {
		if inflight == msg {
			delete(state.inflight, receipt)
		}
	}
	for i, candidate := range state.messages {
		if candidate == msg {
			state.messages = append(state.messages[:i], state.messages[i+1:]...)
			break
		}
	}
}
