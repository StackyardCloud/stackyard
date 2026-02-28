package server

import "sync"

type configStore struct {
	mu sync.Mutex
}

func newConfigStore() *configStore {
	return &configStore{}
}

func (s *configStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, known := configOperationByName[action]; !known {
		return map[string]any{}
	}

	_ = payload
	return map[string]any{}
}
