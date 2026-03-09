package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type azureQueueMessage struct {
	ID         string
	PopReceipt string
	Body       []byte
	EnqueuedAt time.Time
}

func (s *Server) handleAzureQueueRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/queue/") {
		return false
	}
	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/queue/"))
	if len(segments) < 2 {
		respondAzureImplemented(w, path)
		return true
	}

	account := segments[0]
	queue := segments[1]

	if len(segments) == 2 && r.Method == http.MethodPut {
		s.handleAzureQueueCreate(w, account, queue)
		return true
	}
	if len(segments) == 3 && segments[2] == "messages" && r.Method == http.MethodPost {
		s.handleAzureQueueEnqueue(w, r, account, queue)
		return true
	}
	if len(segments) == 4 && segments[2] == "messages" && segments[3] == "dequeue" && r.Method == http.MethodPost {
		s.handleAzureQueueDequeue(w, r, account, queue)
		return true
	}
	if len(segments) == 4 && segments[2] == "messages" && r.Method == http.MethodDelete {
		s.handleAzureQueueDeleteMessage(w, r, account, queue, segments[3])
		return true
	}
	respondAzureImplemented(w, path)
	return true
}

func (s *Server) handleAzureQueueCreate(w http.ResponseWriter, account, queue string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	queues := s.azureQueues[account]
	if queues == nil {
		queues = map[string][]azureQueueMessage{}
		s.azureQueues[account] = queues
	}
	if _, exists := queues[queue]; exists {
		respondJSON(w, http.StatusConflict, map[string]any{"error": "QueueAlreadyExists", "message": "queue already exists"})
		return
	}
	queues[queue] = []azureQueueMessage{}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleAzureQueueEnqueue(w http.ResponseWriter, r *http.Request, account, queue string) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body"})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	queues := s.azureQueues[account]
	if queues == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "QueueNotFound", "message": "queue not found"})
		return
	}
	messages, ok := queues[queue]
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "QueueNotFound", "message": "queue not found"})
		return
	}

	id := s.nextAzureQueueMessageID()
	msg := azureQueueMessage{
		ID:         id,
		PopReceipt: "receipt-" + id,
		Body:       append([]byte(nil), body...),
		EnqueuedAt: time.Now().UTC(),
	}
	queues[queue] = append(messages, msg)
	respondJSON(w, http.StatusCreated, map[string]any{
		"messageId":     msg.ID,
		"popReceipt":    msg.PopReceipt,
		"insertionTime": msg.EnqueuedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleAzureQueueDequeue(w http.ResponseWriter, r *http.Request, account, queue string) {
	num := 1
	if raw := strings.TrimSpace(r.URL.Query().Get("numofmessages")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidQueryParameterValue", "message": "numofmessages must be a positive integer"})
			return
		}
		num = value
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	queues := s.azureQueues[account]
	if queues == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "QueueNotFound", "message": "queue not found"})
		return
	}
	messages, ok := queues[queue]
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "QueueNotFound", "message": "queue not found"})
		return
	}
	if num > len(messages) {
		num = len(messages)
	}
	selected := make([]azureQueueMessage, 0, num)
	now := time.Now().UTC()
	for i := 0; i < num; i++ {
		msg := messages[i]
		msg.PopReceipt = fmt.Sprintf("receipt-%s-%d", msg.ID, now.UnixNano()+int64(i))
		messages[i] = msg
		selected = append(selected, msg)
	}
	queues[queue] = messages

	items := make([]map[string]any, 0, len(selected))
	for _, msg := range selected {
		items = append(items, map[string]any{
			"messageId":     msg.ID,
			"popReceipt":    msg.PopReceipt,
			"messageText":   string(msg.Body),
			"insertionTime": msg.EnqueuedAt.Format(time.RFC3339),
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{"messages": items})
}

func (s *Server) handleAzureQueueDeleteMessage(w http.ResponseWriter, r *http.Request, account, queue, messageID string) {
	expectedPopReceipt := strings.TrimSpace(r.URL.Query().Get("popreceipt"))

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	queues := s.azureQueues[account]
	if queues == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "QueueNotFound", "message": "queue not found"})
		return
	}
	messages, ok := queues[queue]
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "QueueNotFound", "message": "queue not found"})
		return
	}

	index := -1
	for i, msg := range messages {
		if msg.ID == messageID {
			if expectedPopReceipt != "" && expectedPopReceipt != msg.PopReceipt {
				respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidQueryParameterValue", "message": "popreceipt does not match message"})
				return
			}
			index = i
			break
		}
	}
	if index < 0 {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "MessageNotFound", "message": "message not found"})
		return
	}

	queues[queue] = append(messages[:index], messages[index+1:]...)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) nextAzureQueueMessageID() string {
	s.azureQueueNextID++
	return fmt.Sprintf("%d", s.azureQueueNextID)
}
