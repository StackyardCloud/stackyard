package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	defaultSnapshotLoadStrategy = "on_startup"
	defaultSnapshotSaveStrategy = "on_request"
	stateJournalFileName        = "state.journal.jsonl"
	stateSnapshotsDirName       = "snapshots"
)

type persistedRequest struct {
	Method   string              `json:"method"`
	Path     string              `json:"path"`
	RawQuery string              `json:"rawQuery,omitempty"`
	Host     string              `json:"host,omitempty"`
	Header   map[string][]string `json:"header,omitempty"`
	Body     []byte              `json:"body,omitempty"`
}

type statePersistence struct {
	dir          string
	journalPath  string
	snapshotsDir string
	loadStrategy string
	saveStrategy string

	mu        sync.Mutex
	entries   []persistedRequest
	replaying bool
}

func newStatePersistence(cfg Config) *statePersistence {
	if !cfg.PersistenceEnabled {
		return nil
	}

	stateDir := strings.TrimSpace(cfg.StateDir)
	if stateDir == "" {
		stateDir = filepath.Join(os.TempDir(), "stackyard", "state")
	}
	loadStrategy, ok := normalizeSnapshotLoadStrategy(cfg.SnapshotLoadStrategy)
	if !ok {
		log.Printf("invalid snapshot load strategy %q, using %q", cfg.SnapshotLoadStrategy, defaultSnapshotLoadStrategy)
		loadStrategy = defaultSnapshotLoadStrategy
	}
	saveStrategy, ok := normalizeSnapshotSaveStrategy(cfg.SnapshotSaveStrategy)
	if !ok {
		log.Printf("invalid snapshot save strategy %q, using %q", cfg.SnapshotSaveStrategy, defaultSnapshotSaveStrategy)
		saveStrategy = defaultSnapshotSaveStrategy
	}

	journalPath := filepath.Join(stateDir, stateJournalFileName)
	snapshotsDir := filepath.Join(stateDir, stateSnapshotsDirName)
	if err := os.MkdirAll(snapshotsDir, 0o755); err != nil {
		log.Printf("failed to initialize state directory %s: %v", stateDir, err)
		return nil
	}

	entries, err := readStateJournal(journalPath)
	if err != nil {
		log.Printf("failed to read state journal %s: %v", journalPath, err)
		entries = nil
	}

	return &statePersistence{
		dir:          stateDir,
		journalPath:  journalPath,
		snapshotsDir: snapshotsDir,
		loadStrategy: loadStrategy,
		saveStrategy: saveStrategy,
		entries:      entries,
	}
}

func normalizeSnapshotLoadStrategy(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", defaultSnapshotLoadStrategy:
		return defaultSnapshotLoadStrategy, true
	case "manual":
		return "manual", true
	default:
		return "", false
	}
}

func normalizeSnapshotSaveStrategy(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", defaultSnapshotSaveStrategy:
		return defaultSnapshotSaveStrategy, true
	case "on_shutdown", "manual":
		return value, true
	default:
		return "", false
	}
}

func (p *statePersistence) middleware(next http.Handler) http.Handler {
	if p == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture := shouldCaptureStateRequest(r)
		var body []byte
		if capture {
			captured, err := readBodyBytes(r)
			if err != nil {
				log.Printf("state capture body read failed for %s %s: %v", r.Method, rawRequestPath(r), err)
			} else {
				body = captured
			}
		}

		wrapped := &stateCaptureWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		if capture {
			p.record(r, body, wrapped.status)
		}
	})
}

func shouldCaptureStateRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/_stackyard/") {
		return false
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func (p *statePersistence) record(r *http.Request, body []byte, status int) {
	if p == nil || r == nil {
		return
	}
	if status < 200 || status >= 400 {
		return
	}

	headerCopy := make(map[string][]string, len(r.Header))
	for key, values := range r.Header {
		if len(values) == 0 {
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		headerCopy[key] = copied
	}

	entry := persistedRequest{
		Method:   r.Method,
		Path:     rawRequestPath(r),
		RawQuery: r.URL.RawQuery,
		Host:     r.Host,
		Header:   headerCopy,
	}
	if len(body) > 0 {
		entry.Body = make([]byte, len(body))
		copy(entry.Body, body)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.replaying {
		return
	}
	p.entries = append(p.entries, entry)
	if p.saveStrategy == defaultSnapshotSaveStrategy {
		if err := appendStateJournalEntry(p.journalPath, entry); err != nil {
			log.Printf("state journal append failed: %v", err)
		}
	}
}

func (p *statePersistence) restore(handler http.Handler) error {
	if p == nil || handler == nil {
		return nil
	}
	if p.loadStrategy != defaultSnapshotLoadStrategy {
		return nil
	}

	p.mu.Lock()
	entries := make([]persistedRequest, len(p.entries))
	copy(entries, p.entries)
	p.replaying = true
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.replaying = false
		p.mu.Unlock()
	}()

	for i, entry := range entries {
		target := entry.Path
		if target == "" {
			target = "/"
		}
		if entry.RawQuery != "" {
			target += "?" + entry.RawQuery
		}
		req := httptest.NewRequest(entry.Method, target, bytes.NewReader(entry.Body))
		if strings.TrimSpace(entry.Host) != "" {
			req.Host = strings.TrimSpace(entry.Host)
		}
		for key, values := range entry.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		ctx := context.WithValue(req.Context(), skipAccessLogDeliveryKey, true)
		ctx = context.WithValue(ctx, skipSigV4ValidationKey, true)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code >= 400 {
			log.Printf("state replay entry %d failed: %s %s -> %d", i+1, entry.Method, target, rr.Code)
		}
	}
	return nil
}

func (p *statePersistence) close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.saveStrategy != "on_shutdown" {
		return nil
	}
	return writeStateJournal(p.journalPath, p.entries)
}

func (p *statePersistence) info() map[string]any {
	if p == nil {
		return map[string]any{"enabled": false}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshots, err := listStateSnapshots(p.snapshotsDir)
	if err != nil {
		snapshots = nil
	}
	return map[string]any{
		"enabled":              true,
		"stateDir":             p.dir,
		"snapshotLoadStrategy": p.loadStrategy,
		"snapshotSaveStrategy": p.saveStrategy,
		"journalEntries":       len(p.entries),
		"snapshots":            snapshots,
	}
}

func (p *statePersistence) createSnapshot(name string) (int, error) {
	if p == nil {
		return 0, errors.New("state persistence disabled")
	}
	if !isValidSnapshotName(name) {
		return 0, fmt.Errorf("invalid snapshot name %q", name)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := writeStateJournal(p.snapshotPath(name), p.entries); err != nil {
		return 0, err
	}
	if p.saveStrategy == "manual" {
		if err := writeStateJournal(p.journalPath, p.entries); err != nil {
			return 0, err
		}
	}
	return len(p.entries), nil
}

func (p *statePersistence) listSnapshots() ([]string, error) {
	if p == nil {
		return nil, errors.New("state persistence disabled")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return listStateSnapshots(p.snapshotsDir)
}

func (p *statePersistence) restoreSnapshot(name string) (int, error) {
	if p == nil {
		return 0, errors.New("state persistence disabled")
	}
	if !isValidSnapshotName(name) {
		return 0, fmt.Errorf("invalid snapshot name %q", name)
	}
	entries, err := readStateJournal(p.snapshotPath(name))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("snapshot %q not found", name)
		}
		return 0, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries = entries
	if err := writeStateJournal(p.journalPath, p.entries); err != nil {
		return 0, err
	}
	return len(p.entries), nil
}

func (p *statePersistence) deleteSnapshot(name string) error {
	if p == nil {
		return errors.New("state persistence disabled")
	}
	if !isValidSnapshotName(name) {
		return fmt.Errorf("invalid snapshot name %q", name)
	}
	if err := os.Remove(p.snapshotPath(name)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("snapshot %q not found", name)
		}
		return err
	}
	return nil
}

func (p *statePersistence) snapshotPath(name string) string {
	return filepath.Join(p.snapshotsDir, name+".jsonl")
}

func isValidSnapshotName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

func listStateSnapshots(dir string) ([]string, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		name := item.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		out = append(out, strings.TrimSuffix(name, ".jsonl"))
	}
	sort.Strings(out)
	return out, nil
}

func readStateJournal(path string) ([]persistedRequest, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 32<<20)
	entries := make([]persistedRequest, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry persistedRequest
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func appendStateJournalEntry(path string, entry persistedRequest) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(file).Encode(entry); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeStateJournal(path string, entries []persistedRequest) error {
	tmpPath := path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	for _, entry := range entries {
		if err := enc.Encode(entry); err != nil {
			_ = file.Close()
			return err
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

type stateCaptureWriter struct {
	http.ResponseWriter
	status int
}

func (w *stateCaptureWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) Close() error {
	var errOut error
	if s.state != nil {
		if err := s.state.close(); err != nil {
			errOut = err
		}
	}
	if s.httpServer != nil {
		if err := s.httpServer.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if errOut != nil {
				return fmt.Errorf("state close failed: %v; http close failed: %w", errOut, err)
			}
			return err
		}
	}
	if s.http2Server != nil {
		if err := s.http2Server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if errOut != nil {
				return fmt.Errorf("existing close failed: %v; http2 close failed: %w", errOut, err)
			}
			return err
		}
	}
	return errOut
}

func (s *Server) handleStateInfo(w http.ResponseWriter, _ *http.Request) {
	if s.state == nil {
		respondJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	respondJSON(w, http.StatusOK, s.state.info())
}

func (s *Server) handleStateSnapshotList(w http.ResponseWriter, _ *http.Request) {
	if s.state == nil {
		respondError(w, http.StatusConflict, "state persistence disabled")
		return
	}
	snapshots, err := s.state.listSnapshots()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"snapshots": snapshots})
}

func (s *Server) handleStateSnapshotCreate(w http.ResponseWriter, r *http.Request) {
	if s.state == nil {
		respondError(w, http.StatusConflict, "state persistence disabled")
		return
	}
	name := strings.TrimSpace(r.PathValue("name"))
	count, err := s.state.createSnapshot(name)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{
		"snapshot": name,
		"entries":  count,
	})
}

func (s *Server) handleStateSnapshotRestore(w http.ResponseWriter, r *http.Request) {
	if s.state == nil {
		respondError(w, http.StatusConflict, "state persistence disabled")
		return
	}
	name := strings.TrimSpace(r.PathValue("name"))
	count, err := s.state.restoreSnapshot(name)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"snapshot":         name,
		"entries":          count,
		"appliesOnRestart": true,
	})
}

func (s *Server) handleStateSnapshotDelete(w http.ResponseWriter, r *http.Request) {
	if s.state == nil {
		respondError(w, http.StatusConflict, "state persistence disabled")
		return
	}
	name := strings.TrimSpace(r.PathValue("name"))
	if err := s.state.deleteSnapshot(name); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": name})
}
