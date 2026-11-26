package keyspaces

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Change data capture (CDC): a table's cdcSpecification opens a stream that
// Keyspaces Streams (internal/server/keyspacesstreams*.go) serves. Enabling
// CDC on a table without an enabled stream opens a new stream; disabling it
// marks the latest stream DISABLED. Older streams stay readable until the
// history cap drops them.

// maxStreamHistory bounds the streams kept per table.
const maxStreamHistory = 10

// streamLabelLayout is the label (and ARN suffix) format AWS uses.
const streamLabelLayout = "2006-01-02T15:04:05.000"

// Stream is one CDC stream of a table.
type Stream struct {
	ARN          string
	Label        string
	Status       string
	ViewType     string
	Created      time.Time
	KeyspaceName string
	TableName    string
}

var cdcViewTypes = map[string]bool{"NEW_IMAGE": true, "OLD_IMAGE": true, "KEYS_ONLY": true, "NEW_AND_OLD_IMAGES": true}

// applyCDC applies a requested cdcSpecification to table. A nil spec leaves
// CDC unchanged. It validates everything before changing the table.
func applyCDC(table *Table, spec map[string]any, now time.Time) error {
	if spec == nil {
		return nil
	}
	status, _ := spec["status"].(string)
	viewType, viewGiven := spec["viewType"].(string)
	if status != "ENABLED" && status != "DISABLED" {
		return fmt.Errorf("%w: cdcSpecification.status must be ENABLED or DISABLED", ErrInvalidParameter)
	}
	if viewGiven && !cdcViewTypes[viewType] {
		return fmt.Errorf("%w: cdcSpecification.viewType must be one of NEW_IMAGE, OLD_IMAGE, KEYS_ONLY, NEW_AND_OLD_IMAGES", ErrInvalidParameter)
	}
	var latest *Stream
	if n := len(table.Streams); n > 0 {
		latest = &table.Streams[n-1]
	}
	enabled := latest != nil && latest.Status == "ENABLED"
	if status == "DISABLED" {
		if enabled {
			latest.Status = "DISABLED"
		}
		return nil
	}
	if enabled {
		if viewGiven && viewType != latest.ViewType {
			return fmt.Errorf("%w: the view type of an enabled stream can't be changed", ErrInvalidParameter)
		}
		return nil
	}
	if !viewGiven {
		viewType = "NEW_AND_OLD_IMAGES"
	}
	created := now.UTC().Truncate(time.Millisecond)
	if latest != nil && !created.After(latest.Created) {
		created = latest.Created.Add(time.Millisecond) // labels stay unique
	}
	label := created.Format(streamLabelLayout)
	stream := Stream{
		ARN:          table.ResourceARN + "/stream/" + label,
		Label:        label,
		Status:       "ENABLED",
		ViewType:     viewType,
		Created:      created,
		KeyspaceName: table.KeyspaceName,
		TableName:    table.TableName,
	}
	streams := append(append([]Stream(nil), table.Streams...), stream)
	if len(streams) > maxStreamHistory {
		streams = streams[len(streams)-maxStreamHistory:]
	}
	table.Streams = streams
	return nil
}

// LatestStream returns a table's newest stream, if it has one.
func (t Table) LatestStream() (Stream, bool) {
	if len(t.Streams) == 0 {
		return Stream{}, false
	}
	return t.Streams[len(t.Streams)-1], true
}

// Streams lists every CDC stream, optionally limited to one keyspace (and
// table), ordered by keyspace, table and creation time.
func (s *Service) Streams(keyspaceName, tableName string) []Stream {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Stream
	for ks, tables := range s.tables {
		if keyspaceName != "" && ks != keyspaceName {
			continue
		}
		for name, table := range tables {
			if tableName != "" && name != tableName {
				continue
			}
			out = append(out, table.Streams...)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.KeyspaceName != b.KeyspaceName {
			return a.KeyspaceName < b.KeyspaceName
		}
		if a.TableName != b.TableName {
			return a.TableName < b.TableName
		}
		return a.Created.Before(b.Created)
	})
	return out
}

// StreamByARN finds a stream by its ARN.
func (s *Service) StreamByARN(arn string) (Stream, error) {
	idx := strings.Index(arn, "/stream/")
	if idx < 0 {
		return Stream{}, ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, tables := range s.tables {
		for _, table := range tables {
			if table.ResourceARN != arn[:idx] {
				continue
			}
			for _, st := range table.Streams {
				if st.ARN == arn {
					return st, nil
				}
			}
		}
	}
	return Stream{}, ErrNotFound
}
