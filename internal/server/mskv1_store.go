package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	mskv1DefaultRegion    = "us-east-1"
	mskv1DefaultAccountID = "123456789012"

	mskv1SeedReplicatorARN    = "arn:aws:kafka:us-east-1:123456789012:replicator/stackyard-replicator/00000000-0000-0000-0000-000000000001-1"
	mskv1SeedSourceClusterARN = "arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-source/01234567-89ab-cdef-0123-456789abcdef-1"
	mskv1SeedTargetClusterARN = "arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-target/01234567-89ab-cdef-0123-456789abcdef-2"
)

type mskv1Store struct {
	mu sync.Mutex

	nextReplicatorSerial int64
	replicators          map[string]*mskv1Replicator
}

type mskv1Replicator struct {
	Arn                     string
	Name                    string
	Description             string
	State                   string
	CreationTime            string
	CurrentVersion          string
	IsReplicatorReference   bool
	ResourceARN             string
	ServiceExecutionRoleArn string
	KafkaClusters           []mskv1KafkaCluster
	ReplicationInfos        []mskv1ReplicationInfo
	Tags                    map[string]string
	StateInfoCode           string
	StateInfoMessage        string
}

type mskv1KafkaCluster struct {
	Alias            string
	MSKClusterARN    string
	SubnetIDs        []string
	SecurityGroupIDs []string
}

type mskv1ReplicationInfo struct {
	SourceKafkaClusterARN string
	TargetKafkaClusterARN string
	SourceAlias           string
	TargetAlias           string
	TargetCompressionType string
	ConsumerGroup         map[string]any
	Topic                 map[string]any
}

func newMSKV1Store() *mskv1Store {
	now := time.Now().UTC()
	s := &mskv1Store{
		nextReplicatorSerial: 2,
		replicators:          map[string]*mskv1Replicator{},
	}
	seed := s.ensureReplicatorLocked(mskv1SeedReplicatorARN, "stackyard-replicator", now)
	seed.Description = "Stackyard MSK Replicator seed"
	seed.State = "RUNNING"
	seed.CurrentVersion = "1"
	seed.ServiceExecutionRoleArn = "arn:aws:iam::123456789012:role/stackyard-msk-replicator"
	seed.Tags["env"] = "coverage"
	seed.Tags["service"] = "mskv1"
	seed.StateInfoCode = "NONE"
	seed.StateInfoMessage = "replicator is healthy"
	return s
}

func (s *mskv1Store) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := mskv1MergeMaps(payload, pathParams, query)
	now := time.Now().UTC()

	replicatorARN := mskv1StringAny(ctx, []string{"replicatorArn", "ReplicatorArn"}, mskv1SeedReplicatorARN)
	replicatorName := mskv1StringAny(ctx, []string{"replicatorName", "ReplicatorName"}, "stackyard-replicator")
	replicatorNameFilter := mskv1StringAny(ctx, []string{"replicatorNameFilter", "ReplicatorNameFilter"}, "")
	replicator := s.ensureReplicatorLocked(replicatorARN, replicatorName, now)

	switch action {
	case "CreateReplicator":
		created := s.createReplicatorLocked(payload, now)
		return map[string]any{
			"replicatorArn":   created.Arn,
			"replicatorName":  created.Name,
			"replicatorState": created.State,
		}
	case "DeleteReplicator":
		replicator.State = "DELETING"
		replicator.CurrentVersion = mskv1NextVersion(replicator.CurrentVersion)
		replicator.StateInfoCode = "NONE"
		replicator.StateInfoMessage = "replicator deletion in progress"
		return map[string]any{
			"replicatorArn":   replicator.Arn,
			"replicatorState": replicator.State,
		}
	case "DescribeReplicator":
		return s.describeReplicatorPayload(replicator)
	case "ListReplicators":
		items := make([]any, 0, len(s.replicators))
		for _, arn := range s.sortedReplicatorARNsLocked() {
			current := s.replicators[arn]
			if current == nil {
				continue
			}
			if replicatorNameFilter != "" && !strings.HasPrefix(strings.ToLower(current.Name), strings.ToLower(replicatorNameFilter)) {
				continue
			}
			items = append(items, s.replicatorSummaryPayload(current))
		}
		return map[string]any{
			"replicators": items,
			"nextToken":   "",
		}
	case "UpdateReplicationInfo":
		s.applyReplicationInfoUpdateLocked(replicator, payload)
		replicator.State = "RUNNING"
		replicator.CurrentVersion = mskv1NextVersion(replicator.CurrentVersion)
		replicator.StateInfoCode = "NONE"
		replicator.StateInfoMessage = "replication info updated"
		return map[string]any{
			"replicatorArn":   replicator.Arn,
			"replicatorState": replicator.State,
		}
	default:
		return map[string]any{}
	}
}

func (s *mskv1Store) createReplicatorLocked(payload map[string]any, now time.Time) *mskv1Replicator {
	name := mskv1StringAny(payload, []string{"replicatorName", "ReplicatorName"}, "")
	if name == "" {
		name = fmt.Sprintf("stackyard-replicator-%06d", s.nextReplicatorSerial)
	}
	arn := mskv1ReplicatorARN(name, s.nextReplicatorSerial)
	s.nextReplicatorSerial++

	created := s.ensureReplicatorLocked(arn, name, now)
	created.Description = mskv1StringAny(payload, []string{"description", "Description"}, created.Description)
	created.State = "RUNNING"
	created.CurrentVersion = "1"
	role := mskv1StringAny(payload, []string{"serviceExecutionRoleArn", "ServiceExecutionRoleArn"}, "")
	if role != "" {
		created.ServiceExecutionRoleArn = role
	}
	if clusters := mskv1KafkaClustersFromAny(payload["kafkaClusters"], payload["KafkaClusters"]); len(clusters) > 0 {
		created.KafkaClusters = clusters
	}
	if infos := mskv1ReplicationInfosFromAny(payload["replicationInfoList"], payload["ReplicationInfoList"], created.KafkaClusters); len(infos) > 0 {
		created.ReplicationInfos = infos
	}
	for key, value := range mskv1TagsFromAny(payload["tags"]) {
		created.Tags[key] = value
	}
	for key, value := range mskv1TagsFromAny(payload["Tags"]) {
		created.Tags[key] = value
	}
	created.StateInfoCode = "NONE"
	created.StateInfoMessage = "replicator created"
	return created
}

func (s *mskv1Store) ensureReplicatorLocked(arn, name string, now time.Time) *mskv1Replicator {
	resolvedARN := strings.TrimSpace(arn)
	resolvedName := strings.TrimSpace(name)
	if resolvedName == "" {
		resolvedName = mskv1ReplicatorNameFromARN(resolvedARN)
	}
	if resolvedName == "" {
		resolvedName = "stackyard-replicator"
	}
	if resolvedARN == "" {
		resolvedARN = mskv1ReplicatorARN(resolvedName, s.nextReplicatorSerial)
		s.nextReplicatorSerial++
	}

	if existing := s.replicators[resolvedARN]; existing != nil {
		if existing.Name == "" {
			existing.Name = resolvedName
		}
		if existing.ResourceARN == "" {
			existing.ResourceARN = resolvedARN
		}
		return existing
	}

	created := &mskv1Replicator{
		Arn:                     resolvedARN,
		Name:                    resolvedName,
		Description:             "Stackyard MSK Replicator",
		State:                   "RUNNING",
		CreationTime:            now.Format(time.RFC3339),
		CurrentVersion:          "1",
		IsReplicatorReference:   false,
		ResourceARN:             resolvedARN,
		ServiceExecutionRoleArn: "arn:aws:iam::123456789012:role/stackyard-msk-replicator",
		KafkaClusters: []mskv1KafkaCluster{
			{
				Alias:            "source",
				MSKClusterARN:    mskv1SeedSourceClusterARN,
				SubnetIDs:        []string{"subnet-0123456789abcdef0"},
				SecurityGroupIDs: []string{"sg-0123456789abcdef0"},
			},
			{
				Alias:            "target",
				MSKClusterARN:    mskv1SeedTargetClusterARN,
				SubnetIDs:        []string{"subnet-0123456789abcdef0"},
				SecurityGroupIDs: []string{"sg-0123456789abcdef0"},
			},
		},
		ReplicationInfos: []mskv1ReplicationInfo{
			{
				SourceKafkaClusterARN: mskv1SeedSourceClusterARN,
				TargetKafkaClusterARN: mskv1SeedTargetClusterARN,
				SourceAlias:           "source",
				TargetAlias:           "target",
				TargetCompressionType: "NONE",
				ConsumerGroup: map[string]any{
					"consumerGroupsToReplicate":       []any{".*"},
					"consumerGroupsToExclude":         []any{},
					"detectAndCopyNewConsumerGroups":  true,
					"synchroniseConsumerGroupOffsets": true,
				},
				Topic: map[string]any{
					"copyAccessControlListsForTopics": true,
					"copyTopicConfigurations":         true,
					"detectAndCopyNewTopics":          true,
					"startingPosition":                map[string]any{"type": "LATEST"},
					"topicNameConfiguration":          map[string]any{"type": "PREFIXED_WITH_SOURCE_CLUSTER_ALIAS"},
					"topicsToReplicate":               []any{".*"},
					"topicsToExclude":                 []any{},
				},
			},
		},
		Tags:             map[string]string{"stackyard": "true"},
		StateInfoCode:    "NONE",
		StateInfoMessage: "replicator is healthy",
	}
	s.replicators[resolvedARN] = created
	return created
}

func (s *mskv1Store) applyReplicationInfoUpdateLocked(replicator *mskv1Replicator, payload map[string]any) {
	if replicator == nil {
		return
	}
	sourceARN := mskv1StringAny(payload, []string{"sourceKafkaClusterArn", "SourceKafkaClusterArn"}, "")
	targetARN := mskv1StringAny(payload, []string{"targetKafkaClusterArn", "TargetKafkaClusterArn"}, "")
	if sourceARN == "" && len(replicator.KafkaClusters) > 0 {
		sourceARN = replicator.KafkaClusters[0].MSKClusterARN
	}
	if targetARN == "" && len(replicator.KafkaClusters) > 1 {
		targetARN = replicator.KafkaClusters[1].MSKClusterARN
	}
	sourceAlias := s.aliasForClusterARN(replicator, sourceARN, "source")
	targetAlias := s.aliasForClusterARN(replicator, targetARN, "target")
	targetCompression := mskv1StringAny(payload, []string{"targetCompressionType", "TargetCompressionType"}, "")
	consumerGroup := mskv1ReplicationConsumerGroupFromAny(payload["consumerGroupReplication"], payload["ConsumerGroupReplication"])
	topic := mskv1ReplicationTopicFromAny(payload["topicReplication"], payload["TopicReplication"])

	updated := false
	for i := range replicator.ReplicationInfos {
		item := &replicator.ReplicationInfos[i]
		if item.SourceKafkaClusterARN == sourceARN && item.TargetKafkaClusterARN == targetARN {
			if targetCompression != "" {
				item.TargetCompressionType = targetCompression
			}
			if len(consumerGroup) > 0 {
				item.ConsumerGroup = consumerGroup
			}
			if len(topic) > 0 {
				item.Topic = topic
			}
			item.SourceAlias = sourceAlias
			item.TargetAlias = targetAlias
			updated = true
			break
		}
	}
	if updated {
		return
	}

	if targetCompression == "" {
		targetCompression = "NONE"
	}
	if len(consumerGroup) == 0 {
		consumerGroup = map[string]any{"consumerGroupsToReplicate": []any{".*"}}
	}
	if len(topic) == 0 {
		topic = map[string]any{"topicsToReplicate": []any{".*"}}
	}
	replicator.ReplicationInfos = append(replicator.ReplicationInfos, mskv1ReplicationInfo{
		SourceKafkaClusterARN: sourceARN,
		TargetKafkaClusterARN: targetARN,
		SourceAlias:           sourceAlias,
		TargetAlias:           targetAlias,
		TargetCompressionType: targetCompression,
		ConsumerGroup:         consumerGroup,
		Topic:                 topic,
	})
}

func (s *mskv1Store) aliasForClusterARN(replicator *mskv1Replicator, arn, fallback string) string {
	for _, cluster := range replicator.KafkaClusters {
		if strings.EqualFold(strings.TrimSpace(cluster.MSKClusterARN), strings.TrimSpace(arn)) {
			if cluster.Alias != "" {
				return cluster.Alias
			}
		}
	}
	return fallback
}

func (s *mskv1Store) sortedReplicatorARNsLocked() []string {
	keys := make([]string, 0, len(s.replicators))
	for key := range s.replicators {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *mskv1Store) describeReplicatorPayload(replicator *mskv1Replicator) map[string]any {
	if replicator == nil {
		return map[string]any{}
	}
	kafkaClusters := make([]any, 0, len(replicator.KafkaClusters))
	for _, cluster := range replicator.KafkaClusters {
		kafkaClusters = append(kafkaClusters, map[string]any{
			"amazonMskCluster": map[string]any{
				"mskClusterArn": cluster.MSKClusterARN,
			},
			"kafkaClusterAlias": cluster.Alias,
			"vpcConfig": map[string]any{
				"securityGroupIds": mskv1AnyArrayFromStrings(cluster.SecurityGroupIDs),
				"subnetIds":        mskv1AnyArrayFromStrings(cluster.SubnetIDs),
			},
		})
	}

	replicationInfos := make([]any, 0, len(replicator.ReplicationInfos))
	for _, info := range replicator.ReplicationInfos {
		replicationInfos = append(replicationInfos, map[string]any{
			"consumerGroupReplication": mskv1CloneAnyMap(info.ConsumerGroup),
			"sourceKafkaClusterAlias":  info.SourceAlias,
			"targetCompressionType":    info.TargetCompressionType,
			"targetKafkaClusterAlias":  info.TargetAlias,
			"topicReplication":         mskv1CloneAnyMap(info.Topic),
		})
	}

	return map[string]any{
		"creationTime":            replicator.CreationTime,
		"currentVersion":          replicator.CurrentVersion,
		"isReplicatorReference":   replicator.IsReplicatorReference,
		"kafkaClusters":           kafkaClusters,
		"replicationInfoList":     replicationInfos,
		"replicatorArn":           replicator.Arn,
		"replicatorDescription":   replicator.Description,
		"replicatorName":          replicator.Name,
		"replicatorResourceArn":   replicator.ResourceARN,
		"replicatorState":         replicator.State,
		"serviceExecutionRoleArn": replicator.ServiceExecutionRoleArn,
		"stateInfo": map[string]any{
			"code":    replicator.StateInfoCode,
			"message": replicator.StateInfoMessage,
		},
		"tags": mskv1CloneStringMap(replicator.Tags),
	}
}

func (s *mskv1Store) replicatorSummaryPayload(replicator *mskv1Replicator) map[string]any {
	if replicator == nil {
		return map[string]any{}
	}
	kafkaClustersSummary := make([]any, 0, len(replicator.KafkaClusters))
	for _, cluster := range replicator.KafkaClusters {
		kafkaClustersSummary = append(kafkaClustersSummary, map[string]any{
			"amazonMskCluster": map[string]any{
				"mskClusterArn": cluster.MSKClusterARN,
			},
			"kafkaClusterAlias": cluster.Alias,
		})
	}

	replicationInfoSummaryList := make([]any, 0, len(replicator.ReplicationInfos))
	for _, info := range replicator.ReplicationInfos {
		replicationInfoSummaryList = append(replicationInfoSummaryList, map[string]any{
			"sourceKafkaClusterAlias": info.SourceAlias,
			"targetKafkaClusterAlias": info.TargetAlias,
		})
	}

	return map[string]any{
		"creationTime":               replicator.CreationTime,
		"currentVersion":             replicator.CurrentVersion,
		"isReplicatorReference":      replicator.IsReplicatorReference,
		"kafkaClustersSummary":       kafkaClustersSummary,
		"replicationInfoSummaryList": replicationInfoSummaryList,
		"replicatorArn":              replicator.Arn,
		"replicatorName":             replicator.Name,
		"replicatorResourceArn":      replicator.ResourceARN,
		"replicatorState":            replicator.State,
	}
}

func mskv1ReplicatorARN(name string, serial int64) string {
	return fmt.Sprintf(
		"arn:aws:kafka:%s:%s:replicator/%s/%012d-1",
		mskv1DefaultRegion,
		mskv1DefaultAccountID,
		strings.TrimSpace(name),
		serial,
	)
}

func mskv1ReplicatorNameFromARN(arn string) string {
	parts := strings.Split(strings.TrimSpace(arn), ":")
	if len(parts) < 6 {
		return ""
	}
	resource := strings.TrimSpace(parts[5])
	segments := strings.Split(resource, "/")
	if len(segments) < 2 {
		return ""
	}
	return strings.TrimSpace(segments[1])
}

func mskv1NextVersion(current string) string {
	value := strings.TrimSpace(current)
	if value == "" {
		return "1"
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return "1"
	}
	return strconv.Itoa(parsed + 1)
}

func mskv1MergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
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
			for _, value := range values {
				arr = append(arr, value)
			}
			out[key] = arr
		}
	}
	return out
}

func mskv1StringAny(in map[string]any, keys []string, fallback string) string {
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

func mskv1KafkaClustersFromAny(primary any, secondary any) []mskv1KafkaCluster {
	var raw []any
	switch value := primary.(type) {
	case []any:
		raw = value
	}
	if len(raw) == 0 {
		if value, ok := secondary.([]any); ok {
			raw = value
		}
	}
	if len(raw) == 0 {
		return nil
	}

	clusters := make([]mskv1KafkaCluster, 0, len(raw))
	for idx, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		amazonMskCluster, _ := entry["amazonMskCluster"].(map[string]any)
		mskClusterARN := mskv1StringAny(amazonMskCluster, []string{"mskClusterArn", "MskClusterArn"}, "")
		if mskClusterARN == "" {
			if idx == 0 {
				mskClusterARN = mskv1SeedSourceClusterARN
			} else {
				mskClusterARN = mskv1SeedTargetClusterARN
			}
		}

		vpcConfig, _ := entry["vpcConfig"].(map[string]any)
		alias := mskv1StringAny(entry, []string{"kafkaClusterAlias", "KafkaClusterAlias"}, "")
		if alias == "" {
			if idx == 0 {
				alias = "source"
			} else if idx == 1 {
				alias = "target"
			} else {
				alias = fmt.Sprintf("cluster-%d", idx+1)
			}
		}

		subnets := mskv1StringArrayAny(vpcConfig["subnetIds"], []string{"subnet-0123456789abcdef0"})
		securityGroups := mskv1StringArrayAny(vpcConfig["securityGroupIds"], []string{"sg-0123456789abcdef0"})

		clusters = append(clusters, mskv1KafkaCluster{
			Alias:            alias,
			MSKClusterARN:    mskClusterARN,
			SubnetIDs:        subnets,
			SecurityGroupIDs: securityGroups,
		})
	}
	return clusters
}

func mskv1ReplicationInfosFromAny(primary any, secondary any, kafkaClusters []mskv1KafkaCluster) []mskv1ReplicationInfo {
	var raw []any
	switch value := primary.(type) {
	case []any:
		raw = value
	}
	if len(raw) == 0 {
		if value, ok := secondary.([]any); ok {
			raw = value
		}
	}
	if len(raw) == 0 {
		return nil
	}

	out := make([]mskv1ReplicationInfo, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sourceARN := mskv1StringAny(entry, []string{"sourceKafkaClusterArn", "SourceKafkaClusterArn"}, "")
		targetARN := mskv1StringAny(entry, []string{"targetKafkaClusterArn", "TargetKafkaClusterArn"}, "")
		if sourceARN == "" {
			sourceARN = mskv1SeedSourceClusterARN
		}
		if targetARN == "" {
			targetARN = mskv1SeedTargetClusterARN
		}
		sourceAlias := mskv1ClusterAliasForARN(kafkaClusters, sourceARN, "source")
		targetAlias := mskv1ClusterAliasForARN(kafkaClusters, targetARN, "target")
		targetCompression := mskv1StringAny(entry, []string{"targetCompressionType", "TargetCompressionType"}, "NONE")

		out = append(out, mskv1ReplicationInfo{
			SourceKafkaClusterARN: sourceARN,
			TargetKafkaClusterARN: targetARN,
			SourceAlias:           sourceAlias,
			TargetAlias:           targetAlias,
			TargetCompressionType: targetCompression,
			ConsumerGroup: mskv1ReplicationConsumerGroupFromAny(
				entry["consumerGroupReplication"],
				entry["ConsumerGroupReplication"],
			),
			Topic: mskv1ReplicationTopicFromAny(
				entry["topicReplication"],
				entry["TopicReplication"],
			),
		})
	}
	return out
}

func mskv1ReplicationConsumerGroupFromAny(primary any, secondary any) map[string]any {
	out := map[string]any{}
	var src map[string]any
	if value, ok := primary.(map[string]any); ok {
		src = value
	}
	if len(src) == 0 {
		if value, ok := secondary.(map[string]any); ok {
			src = value
		}
	}
	if len(src) == 0 {
		return out
	}

	if values := mskv1AnyArrayAny(src["consumerGroupsToReplicate"]); len(values) > 0 {
		out["consumerGroupsToReplicate"] = values
	}
	if values := mskv1AnyArrayAny(src["consumerGroupsToExclude"]); len(values) > 0 {
		out["consumerGroupsToExclude"] = values
	}
	if value, ok := src["detectAndCopyNewConsumerGroups"].(bool); ok {
		out["detectAndCopyNewConsumerGroups"] = value
	}
	if value, ok := src["synchroniseConsumerGroupOffsets"].(bool); ok {
		out["synchroniseConsumerGroupOffsets"] = value
	}
	return out
}

func mskv1ReplicationTopicFromAny(primary any, secondary any) map[string]any {
	out := map[string]any{}
	var src map[string]any
	if value, ok := primary.(map[string]any); ok {
		src = value
	}
	if len(src) == 0 {
		if value, ok := secondary.(map[string]any); ok {
			src = value
		}
	}
	if len(src) == 0 {
		return out
	}

	if value, ok := src["copyAccessControlListsForTopics"].(bool); ok {
		out["copyAccessControlListsForTopics"] = value
	}
	if value, ok := src["copyTopicConfigurations"].(bool); ok {
		out["copyTopicConfigurations"] = value
	}
	if value, ok := src["detectAndCopyNewTopics"].(bool); ok {
		out["detectAndCopyNewTopics"] = value
	}
	if values := mskv1AnyArrayAny(src["topicsToReplicate"]); len(values) > 0 {
		out["topicsToReplicate"] = values
	}
	if values := mskv1AnyArrayAny(src["topicsToExclude"]); len(values) > 0 {
		out["topicsToExclude"] = values
	}
	if startingPosition, ok := src["startingPosition"].(map[string]any); ok {
		positionType := mskv1StringAny(startingPosition, []string{"type", "Type"}, "")
		if positionType != "" {
			out["startingPosition"] = map[string]any{"type": positionType}
		}
	}
	if topicNameConfig, ok := src["topicNameConfiguration"].(map[string]any); ok {
		configType := mskv1StringAny(topicNameConfig, []string{"type", "Type"}, "")
		if configType != "" {
			out["topicNameConfiguration"] = map[string]any{"type": configType}
		}
	}
	return out
}

func mskv1ClusterAliasForARN(clusters []mskv1KafkaCluster, arn, fallback string) string {
	for _, cluster := range clusters {
		if strings.EqualFold(strings.TrimSpace(cluster.MSKClusterARN), strings.TrimSpace(arn)) {
			if cluster.Alias != "" {
				return cluster.Alias
			}
		}
	}
	return fallback
}

func mskv1AnyArrayAny(raw any) []any {
	switch value := raw.(type) {
	case []any:
		out := make([]any, 0, len(value))
		for _, item := range value {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	case []string:
		return mskv1AnyArrayFromStrings(value)
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil
		}
		return []any{trimmed}
	default:
		return nil
	}
}

func mskv1AnyArrayFromStrings(in []string) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func mskv1StringArrayAny(raw any, fallback []string) []string {
	switch value := raw.(type) {
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			trimmed := strings.TrimSpace(fmt.Sprintf("%v", item))
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) > 0 {
			return out
		}
	case []string:
		out := make([]string, 0, len(value))
		for _, item := range value {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return append([]string{}, fallback...)
}

func mskv1TagsFromAny(raw any) map[string]string {
	out := map[string]string{}
	switch value := raw.(type) {
	case map[string]string:
		for key, item := range value {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(item)
		}
	case map[string]any:
		for key, item := range value {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(fmt.Sprintf("%v", item))
		}
	}
	return out
}

func mskv1CloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func mskv1CloneAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
