package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	mskDefaultRegion    = "us-east-1"
	mskDefaultAccountID = "123456789012"
)

type mskStore struct {
	mu sync.Mutex

	nextClusterSerial   int64
	nextOperationSerial int64

	clusters        map[string]*mskCluster
	clusterByARN    map[string]string
	operations      map[string]*mskClusterOperation
	operationByARN  map[string]string
	clusterOpByARNs map[string][]string
}

type mskCluster struct {
	Arn                 string
	Name                string
	ClusterType         string
	State               string
	CreationTime        string
	CurrentVersion      string
	NumberOfBrokerNodes int
	KafkaVersion        string
}

type mskClusterOperation struct {
	Arn         string
	ClusterArn  string
	ClusterType string
	State       string
	Type        string
	StartTime   string
	EndTime     string
}

func newMSKStore() *mskStore {
	s := &mskStore{
		nextClusterSerial:   2,
		nextOperationSerial: 2,
		clusters:            map[string]*mskCluster{},
		clusterByARN:        map[string]string{},
		operations:          map[string]*mskClusterOperation{},
		operationByARN:      map[string]string{},
		clusterOpByARNs:     map[string][]string{},
	}

	now := time.Now().UTC()
	clusterArn := "arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-msk-v2-cluster/01234567-89ab-cdef-0123-456789abcdef-7"
	cluster := s.ensureClusterLocked(clusterArn, "stackyard-msk-v2-cluster", now)
	s.ensureOperationLocked(
		"arn:aws:kafka:us-east-1:123456789012:cluster-operation/stackyard-msk-v2-cluster/01234567-89ab-cdef-0123-456789abcdef-7",
		cluster.Arn,
		"CREATE_CLUSTER",
		"SUCCEEDED",
		now,
	)
	return s
}

func (s *mskStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := mskMergeMaps(payload, pathParams, query)
	clusterArn := mskStringAny(ctx, []string{"clusterArn", "ClusterArn"}, "arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-msk-v2-cluster/01234567-89ab-cdef-0123-456789abcdef-7")
	clusterOpArn := mskStringAny(ctx, []string{"clusterOperationArn", "ClusterOperationArn"}, "arn:aws:kafka:us-east-1:123456789012:cluster-operation/stackyard-msk-v2-cluster/01234567-89ab-cdef-0123-456789abcdef-7")
	clusterName := mskStringAny(ctx, []string{"clusterName", "ClusterName"}, "")

	now := time.Now().UTC()
	cluster := s.ensureClusterLocked(clusterArn, clusterName, now)

	switch action {
	case "CreateClusterV2":
		created := s.createClusterFromPayloadLocked(payload, now)
		op := s.ensureOperationLocked("", created.Arn, "CREATE_CLUSTER", "SUCCEEDED", now)
		return map[string]any{
			"clusterArn":          created.Arn,
			"clusterName":         created.Name,
			"state":               created.State,
			"clusterOperationArn": op.Arn,
		}
	case "DescribeClusterV2":
		return map[string]any{
			"clusterInfo": s.clusterInfo(createdOrFallback(cluster, s.ensureClusterLocked(clusterArn, clusterName, now))),
		}
	case "ListClustersV2":
		items := make([]any, 0, len(s.clusters))
		for _, arn := range s.sortedClusterARNsLocked() {
			items = append(items, s.clusterInfo(s.clusters[arn]))
		}
		return map[string]any{
			"clusterInfoList": items,
			"nextToken":       "",
		}
	case "DescribeClusterOperationV2":
		op := s.ensureOperationLocked(clusterOpArn, cluster.Arn, "CREATE_CLUSTER", "SUCCEEDED", now)
		return map[string]any{
			"clusterOperationInfo": s.clusterOperationInfo(op),
		}
	case "ListClusterOperationsV2":
		ops := make([]any, 0)
		for _, opArn := range s.clusterOpByARNs[cluster.Arn] {
			if op := s.operations[opArn]; op != nil {
				ops = append(ops, s.clusterOperationInfo(op))
			}
		}
		if len(ops) == 0 {
			op := s.ensureOperationLocked("", cluster.Arn, "CREATE_CLUSTER", "SUCCEEDED", now)
			ops = append(ops, s.clusterOperationInfo(op))
		}
		return map[string]any{
			"clusterOperationInfoList": ops,
			"nextToken":                "",
		}
	default:
		return map[string]any{}
	}
}

func createdOrFallback(primary, fallback *mskCluster) *mskCluster {
	if primary != nil {
		return primary
	}
	return fallback
}

func (s *mskStore) createClusterFromPayloadLocked(payload map[string]any, now time.Time) *mskCluster {
	name := mskStringAny(payload, []string{"clusterName", "ClusterName"}, "")
	if name == "" {
		name = fmt.Sprintf("stackyard-msk-v2-cluster-%06d", s.nextClusterSerial)
	}

	serial := s.nextClusterSerial
	s.nextClusterSerial++
	id := fmt.Sprintf("%012d", serial)
	clusterArn := fmt.Sprintf("arn:aws:kafka:%s:%s:cluster/%s/%s", mskDefaultRegion, mskDefaultAccountID, name, id)
	cluster := s.ensureClusterLocked(clusterArn, name, now)

	if provisioned, ok := payload["provisioned"].(map[string]any); ok {
		if n := mskIntAny(provisioned, []string{"numberOfBrokerNodes"}, 0); n > 0 {
			cluster.NumberOfBrokerNodes = n
		}
		if v := mskStringAny(provisioned, []string{"kafkaVersion"}, ""); v != "" {
			cluster.KafkaVersion = v
		}
	}
	if provisioned, ok := payload["Provisioned"].(map[string]any); ok {
		if n := mskIntAny(provisioned, []string{"NumberOfBrokerNodes"}, 0); n > 0 {
			cluster.NumberOfBrokerNodes = n
		}
		if v := mskStringAny(provisioned, []string{"KafkaVersion"}, ""); v != "" {
			cluster.KafkaVersion = v
		}
	}

	cluster.State = "ACTIVE"
	cluster.CurrentVersion = mskVersion(now)
	return cluster
}

func (s *mskStore) ensureClusterLocked(clusterArn, clusterName string, now time.Time) *mskCluster {
	arn := strings.TrimSpace(clusterArn)
	name := strings.TrimSpace(clusterName)

	if arn != "" {
		if existing := s.clusters[arn]; existing != nil {
			if name != "" {
				existing.Name = name
			}
			return existing
		}
		if mapped := strings.TrimSpace(s.clusterByARN[arn]); mapped != "" {
			if existing := s.clusters[mapped]; existing != nil {
				return existing
			}
		}
		if parsed := mskClusterNameFromARN(arn); parsed != "" && name == "" {
			name = parsed
		}
	}
	if name == "" {
		name = "stackyard-msk-v2-cluster"
	}
	if arn == "" {
		arn = fmt.Sprintf(
			"arn:aws:kafka:%s:%s:cluster/%s/%012d",
			mskDefaultRegion,
			mskDefaultAccountID,
			name,
			1,
		)
	}

	cluster := &mskCluster{
		Arn:                 arn,
		Name:                name,
		ClusterType:         "PROVISIONED",
		State:               "ACTIVE",
		CreationTime:        now.Format(time.RFC3339),
		CurrentVersion:      mskVersion(now),
		NumberOfBrokerNodes: 3,
		KafkaVersion:        "3.7.x",
	}
	s.clusters[arn] = cluster
	s.clusterByARN[arn] = arn
	if _, exists := s.clusterOpByARNs[arn]; !exists {
		s.clusterOpByARNs[arn] = []string{}
	}
	return cluster
}

func (s *mskStore) ensureOperationLocked(operationArn, clusterArn, operationType, operationState string, now time.Time) *mskClusterOperation {
	arn := strings.TrimSpace(operationArn)
	if arn != "" {
		if existing := s.operations[arn]; existing != nil {
			return existing
		}
	}
	clusterArn = strings.TrimSpace(clusterArn)
	if clusterArn == "" {
		clusterArn = "arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-msk-v2-cluster/01234567-89ab-cdef-0123-456789abcdef-7"
	}
	if operationType == "" {
		operationType = "CREATE_CLUSTER"
	}
	if operationState == "" {
		operationState = "SUCCEEDED"
	}

	if arn == "" {
		serial := s.nextOperationSerial
		s.nextOperationSerial++
		arn = fmt.Sprintf(
			"arn:aws:kafka:%s:%s:cluster-operation/%s/%012d",
			mskDefaultRegion,
			mskDefaultAccountID,
			mskClusterNameFromARN(clusterArn),
			serial,
		)
	}

	op := &mskClusterOperation{
		Arn:         arn,
		ClusterArn:  clusterArn,
		ClusterType: "PROVISIONED",
		State:       operationState,
		Type:        operationType,
		StartTime:   now.Format(time.RFC3339),
		EndTime:     now.Format(time.RFC3339),
	}
	s.operations[arn] = op
	s.operationByARN[arn] = arn
	s.clusterOpByARNs[clusterArn] = appendUniqueString(s.clusterOpByARNs[clusterArn], arn)
	return op
}

func (s *mskStore) sortedClusterARNsLocked() []string {
	keys := make([]string, 0, len(s.clusters))
	for key := range s.clusters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *mskStore) clusterInfo(c *mskCluster) map[string]any {
	if c == nil {
		return map[string]any{}
	}
	return map[string]any{
		"clusterArn":     c.Arn,
		"clusterName":    c.Name,
		"clusterType":    c.ClusterType,
		"state":          c.State,
		"creationTime":   c.CreationTime,
		"currentVersion": c.CurrentVersion,
		"provisioned": map[string]any{
			"numberOfBrokerNodes": c.NumberOfBrokerNodes,
			"kafkaVersion":        c.KafkaVersion,
		},
	}
}

func (s *mskStore) clusterOperationInfo(op *mskClusterOperation) map[string]any {
	if op == nil {
		return map[string]any{}
	}
	return map[string]any{
		"clusterArn":     op.ClusterArn,
		"clusterType":    op.ClusterType,
		"operationArn":   op.Arn,
		"operationState": op.State,
		"operationType":  op.Type,
		"startTime":      op.StartTime,
		"endTime":        op.EndTime,
	}
}

func mskVersion(now time.Time) string {
	return fmt.Sprintf("K%s", now.UTC().Format("20060102150405"))
}

func mskClusterNameFromARN(arn string) string {
	parts := strings.Split(strings.TrimSpace(arn), ":")
	if len(parts) < 6 {
		return ""
	}
	resource := parts[5]
	segments := strings.Split(resource, "/")
	if len(segments) < 2 {
		return ""
	}
	return strings.TrimSpace(segments[1])
}

func mskMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 1 {
			out[key] = values[0]
		} else if len(values) > 1 {
			arr := make([]any, 0, len(values))
			for _, v := range values {
				arr = append(arr, v)
			}
			out[key] = arr
		}
	}
	return out
}

func mskStringAny(in map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		raw, ok := in[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		case json.Number:
			trimmed := strings.TrimSpace(value.String())
			if trimmed != "" {
				return trimmed
			}
		default:
			trimmed := strings.TrimSpace(fmt.Sprintf("%v", raw))
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func mskIntAny(in map[string]any, keys []string, fallback int) int {
	for _, key := range keys {
		raw, ok := in[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case int:
			return value
		case int64:
			return int(value)
		case float64:
			return int(value)
		case json.Number:
			if parsed, err := value.Int64(); err == nil {
				return int(parsed)
			}
			if parsed, err := value.Float64(); err == nil {
				return int(parsed)
			}
		case string:
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			var parsed int
			if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func appendUniqueString(in []string, value string) []string {
	val := strings.TrimSpace(value)
	if val == "" {
		return in
	}
	for _, existing := range in {
		if strings.TrimSpace(existing) == val {
			return in
		}
	}
	return append(in, val)
}
