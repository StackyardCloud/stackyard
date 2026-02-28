package server

import "sync"

type computeOptimizerStore struct {
	mu sync.Mutex
}

func newComputeOptimizerStore() *computeOptimizerStore {
	return &computeOptimizerStore{}
}

func (s *computeOptimizerStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, known := computeOptimizerOperationByName[action]; !known {
		return map[string]any{}
	}

	_ = payload
	return map[string]any{}
}
