package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	pcsDefaultRegion    = "us-east-1"
	pcsDefaultAccountID = "123456789012"
)

type pcsStore struct {
	mu sync.Mutex

	nextID int64

	clusters          map[string]*pcsCluster
	computeNodeGroups map[string]*pcsComputeNodeGroup
	queues            map[string]*pcsQueue
	tags              map[string]map[string]string

	clusterCreateTokens          map[string]string
	computeNodeGroupCreateTokens map[string]string
	queueCreateTokens            map[string]string
}

type pcsCluster struct {
	ID                 string
	Name               string
	ARN                string
	Size               string
	Status             string
	CreatedAt          string
	ModifiedAt         string
	Networking         map[string]any
	Scheduler          map[string]any
	SlurmConfiguration map[string]any
	Endpoints          []any
}

type pcsComputeNodeGroup struct {
	ID                 string
	Name               string
	ARN                string
	ClusterID          string
	Status             string
	CreatedAt          string
	ModifiedAt         string
	AMIID              string
	IAMInstanceProfile string
	PurchaseOption     string
	InstanceConfigs    []any
	ScalingConfig      map[string]any
	SubnetIDs          []any
	SlurmConfiguration map[string]any
	CustomTemplate     map[string]any
	SpotOptions        map[string]any
}

type pcsQueue struct {
	ID                     string
	Name                   string
	ARN                    string
	ClusterID              string
	Status                 string
	CreatedAt              string
	ModifiedAt             string
	ComputeNodeGroupConfig []any
	SlurmConfiguration     map[string]any
}

func newPCSStore() *pcsStore {
	now := time.Now().UTC().Format(time.RFC3339)

	s := &pcsStore{
		nextID:                       2,
		clusters:                     map[string]*pcsCluster{},
		computeNodeGroups:            map[string]*pcsComputeNodeGroup{},
		queues:                       map[string]*pcsQueue{},
		tags:                         map[string]map[string]string{},
		clusterCreateTokens:          map[string]string{},
		computeNodeGroupCreateTokens: map[string]string{},
		queueCreateTokens:            map[string]string{},
	}

	cluster := s.ensureClusterLocked("pcs_cluster_000001", "stackyard-cluster", now)
	cluster.Status = "ACTIVE"

	nodeGroup := s.ensureComputeNodeGroupLocked("pcs_cng_000001", cluster.ID, "stackyard-node-group", now)
	nodeGroup.Status = "ACTIVE"

	queue := s.ensureQueueLocked("pcs_queue_000001", cluster.ID, "stackyard-queue", now)
	queue.Status = "ACTIVE"
	queue.ComputeNodeGroupConfig = []any{
		map[string]any{"computeNodeGroupId": nodeGroup.ID},
	}

	s.ensureTagsLocked(cluster.ARN)["stackyard"] = "true"
	s.ensureTagsLocked(nodeGroup.ARN)["stackyard"] = "true"
	s.ensureTagsLocked(queue.ARN)["stackyard"] = "true"

	return s
}

func (s *pcsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateCluster":
		return s.pcsCreateClusterLocked(payload, now)
	case "GetCluster":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "id", "name"))
		return map[string]any{"cluster": pcsSvcClusterMap(cluster)}
	case "ListClusters":
		return map[string]any{"clusters": s.listClusterSummariesLocked(), "nextToken": ""}
	case "UpdateCluster":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "id", "name"))
		if size := pcsSvcPayloadString(payload, "size"); size != "" {
			cluster.Size = size
		}
		if slurm := pcsSvcPayloadMap(payload, "slurmConfiguration"); len(slurm) > 0 {
			cluster.SlurmConfiguration = pcsSvcCloneAnyMap(slurm)
		}
		cluster.Status = "UPDATING"
		cluster.ModifiedAt = now
		cluster.Status = "ACTIVE"
		return map[string]any{"cluster": pcsSvcClusterMap(cluster)}
	case "DeleteCluster":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "id", "name"))
		cluster.Status = "DELETING"
		cluster.ModifiedAt = now
		cluster.Status = "DELETED"
		return map[string]any{"cluster": pcsSvcClusterMap(cluster)}

	case "CreateComputeNodeGroup":
		return s.pcsCreateComputeNodeGroupLocked(payload, now)
	case "GetComputeNodeGroup":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName"))
		nodeGroup := s.resolveComputeNodeGroupLocked(cluster.ID, pcsSvcPayloadString(payload, "computeNodeGroupIdentifier", "computeNodeGroupId", "id", "name"))
		return map[string]any{"computeNodeGroup": pcsSvcComputeNodeGroupMap(nodeGroup)}
	case "ListComputeNodeGroups":
		clusterIdentifier := pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName")
		var clusterID string
		if clusterIdentifier != "" {
			clusterID = s.resolveClusterLocked(clusterIdentifier).ID
		}
		return map[string]any{"computeNodeGroups": s.listComputeNodeGroupSummariesLocked(clusterID), "nextToken": ""}
	case "UpdateComputeNodeGroup":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName"))
		nodeGroup := s.resolveComputeNodeGroupLocked(cluster.ID, pcsSvcPayloadString(payload, "computeNodeGroupIdentifier", "computeNodeGroupId", "id", "name"))
		if value := pcsSvcPayloadString(payload, "amiId"); value != "" {
			nodeGroup.AMIID = value
		}
		if value := pcsSvcPayloadString(payload, "iamInstanceProfileArn"); value != "" {
			nodeGroup.IAMInstanceProfile = value
		}
		if value := pcsSvcPayloadString(payload, "purchaseOption"); value != "" {
			nodeGroup.PurchaseOption = value
		}
		if cfg := pcsSvcPayloadMap(payload, "scalingConfiguration"); len(cfg) > 0 {
			nodeGroup.ScalingConfig = pcsSvcCloneAnyMap(cfg)
		}
		if slurm := pcsSvcPayloadMap(payload, "slurmConfiguration"); len(slurm) > 0 {
			nodeGroup.SlurmConfiguration = pcsSvcCloneAnyMap(slurm)
		}
		if template := pcsSvcPayloadMap(payload, "customLaunchTemplate"); len(template) > 0 {
			nodeGroup.CustomTemplate = pcsSvcCloneAnyMap(template)
		}
		if spot := pcsSvcPayloadMap(payload, "spotOptions"); len(spot) > 0 {
			nodeGroup.SpotOptions = pcsSvcCloneAnyMap(spot)
		}
		if subnets := pcsSvcPayloadList(payload, "subnetIds"); len(subnets) > 0 {
			nodeGroup.SubnetIDs = pcsSvcCloneAnyList(subnets)
		}
		nodeGroup.Status = "UPDATING"
		nodeGroup.ModifiedAt = now
		nodeGroup.Status = "ACTIVE"
		return map[string]any{"computeNodeGroup": pcsSvcComputeNodeGroupMap(nodeGroup)}
	case "DeleteComputeNodeGroup":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName"))
		nodeGroup := s.resolveComputeNodeGroupLocked(cluster.ID, pcsSvcPayloadString(payload, "computeNodeGroupIdentifier", "computeNodeGroupId", "id", "name"))
		nodeGroup.Status = "DELETING"
		nodeGroup.ModifiedAt = now
		nodeGroup.Status = "DELETED"
		return map[string]any{"computeNodeGroup": pcsSvcComputeNodeGroupMap(nodeGroup)}
	case "RegisterComputeNodeGroupInstance":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName"))
		bootstrapID := pcsSvcPayloadString(payload, "bootstrapId", "instanceId")
		if bootstrapID == "" {
			bootstrapID = fmt.Sprintf("i-%08d", s.nextID)
		}
		return map[string]any{
			"endpoints":    pcsSvcCloneAnyList(cluster.Endpoints),
			"nodeID":       bootstrapID,
			"sharedSecret": fmt.Sprintf("pcs-secret-%06d", s.nextID),
		}

	case "CreateQueue":
		return s.pcsCreateQueueLocked(payload, now)
	case "GetQueue":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName"))
		queue := s.resolveQueueLocked(cluster.ID, pcsSvcPayloadString(payload, "queueIdentifier", "queueId", "id", "name"))
		return map[string]any{"queue": pcsSvcQueueMap(queue)}
	case "ListQueues":
		clusterIdentifier := pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName")
		var clusterID string
		if clusterIdentifier != "" {
			clusterID = s.resolveClusterLocked(clusterIdentifier).ID
		}
		return map[string]any{"queues": s.listQueueSummariesLocked(clusterID), "nextToken": ""}
	case "UpdateQueue":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName"))
		queue := s.resolveQueueLocked(cluster.ID, pcsSvcPayloadString(payload, "queueIdentifier", "queueId", "id", "name"))
		if cng := pcsSvcPayloadList(payload, "computeNodeGroupConfigurations"); len(cng) > 0 {
			queue.ComputeNodeGroupConfig = pcsSvcCloneAnyList(cng)
		}
		if slurm := pcsSvcPayloadMap(payload, "slurmConfiguration"); len(slurm) > 0 {
			queue.SlurmConfiguration = pcsSvcCloneAnyMap(slurm)
		}
		queue.Status = "UPDATING"
		queue.ModifiedAt = now
		queue.Status = "ACTIVE"
		return map[string]any{"queue": pcsSvcQueueMap(queue)}
	case "DeleteQueue":
		cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName"))
		queue := s.resolveQueueLocked(cluster.ID, pcsSvcPayloadString(payload, "queueIdentifier", "queueId", "id", "name"))
		queue.Status = "DELETING"
		queue.ModifiedAt = now
		queue.Status = "DELETED"
		return map[string]any{"queue": pcsSvcQueueMap(queue)}

	case "TagResource":
		resourceARN := pcsSvcPayloadString(payload, "resourceArn", "resourceARN")
		if resourceARN == "" {
			resourceARN = s.defaultClusterARNLocked()
		}
		target := s.ensureTagsLocked(resourceARN)
		for key, value := range pcsSvcPayloadStringMap(payload, "tags", "Tags") {
			target[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		resourceARN := pcsSvcPayloadString(payload, "resourceArn", "resourceARN")
		if resourceARN == "" {
			resourceARN = s.defaultClusterARNLocked()
		}
		target := s.ensureTagsLocked(resourceARN)
		for _, key := range pcsSvcPayloadStringSlice(payload, "tagKeys", "TagKeys") {
			delete(target, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := pcsSvcPayloadString(payload, "resourceArn", "resourceARN")
		if resourceARN == "" {
			resourceARN = s.defaultClusterARNLocked()
		}
		return map[string]any{"tags": pcsSvcStringMapToAny(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *pcsStore) pcsCreateClusterLocked(payload map[string]any, now string) map[string]any {
	token := strings.TrimSpace(pcsSvcPayloadString(payload, "clientToken"))
	if token != "" {
		if clusterID, ok := s.clusterCreateTokens[token]; ok {
			cluster := s.resolveClusterLocked(clusterID)
			return map[string]any{"cluster": pcsSvcClusterMap(cluster)}
		}
	}

	name := pcsSvcPayloadString(payload, "clusterName", "name")
	if name == "" {
		name = s.nextNameLocked("cluster")
	}
	cluster := s.findClusterByNameLocked(name)
	if cluster == nil {
		clusterID := s.nextIdentifierLocked("cluster")
		cluster = s.ensureClusterLocked(clusterID, name, now)
	}
	if size := pcsSvcPayloadString(payload, "size"); size != "" {
		cluster.Size = size
	}
	if networking := pcsSvcPayloadMap(payload, "networking"); len(networking) > 0 {
		cluster.Networking = pcsSvcCloneAnyMap(networking)
	}
	if scheduler := pcsSvcPayloadMap(payload, "scheduler"); len(scheduler) > 0 {
		cluster.Scheduler = pcsSvcCloneAnyMap(scheduler)
	}
	if slurm := pcsSvcPayloadMap(payload, "slurmConfiguration"); len(slurm) > 0 {
		cluster.SlurmConfiguration = pcsSvcCloneAnyMap(slurm)
	}
	cluster.Status = "CREATING"
	cluster.ModifiedAt = now
	cluster.Status = "ACTIVE"

	for key, value := range pcsSvcPayloadStringMap(payload, "tags", "Tags") {
		s.ensureTagsLocked(cluster.ARN)[key] = value
	}
	if token != "" {
		s.clusterCreateTokens[token] = cluster.ID
	}

	return map[string]any{"cluster": pcsSvcClusterMap(cluster)}
}

func (s *pcsStore) pcsCreateComputeNodeGroupLocked(payload map[string]any, now string) map[string]any {
	token := strings.TrimSpace(pcsSvcPayloadString(payload, "clientToken"))
	if token != "" {
		if nodeGroupID, ok := s.computeNodeGroupCreateTokens[token]; ok {
			nodeGroup := s.resolveComputeNodeGroupLocked("", nodeGroupID)
			return map[string]any{"computeNodeGroup": pcsSvcComputeNodeGroupMap(nodeGroup)}
		}
	}

	cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName"))
	name := pcsSvcPayloadString(payload, "computeNodeGroupName", "name")
	if name == "" {
		name = s.nextNameLocked("compute-node-group")
	}

	nodeGroup := s.findComputeNodeGroupByNameLocked(cluster.ID, name)
	if nodeGroup == nil {
		nodeGroupID := s.nextIdentifierLocked("computenodegroup")
		nodeGroup = s.ensureComputeNodeGroupLocked(nodeGroupID, cluster.ID, name, now)
	}
	if value := pcsSvcPayloadString(payload, "amiId"); value != "" {
		nodeGroup.AMIID = value
	}
	if value := pcsSvcPayloadString(payload, "iamInstanceProfileArn"); value != "" {
		nodeGroup.IAMInstanceProfile = value
	}
	if value := pcsSvcPayloadString(payload, "purchaseOption"); value != "" {
		nodeGroup.PurchaseOption = value
	}
	if cfg := pcsSvcPayloadMap(payload, "scalingConfiguration"); len(cfg) > 0 {
		nodeGroup.ScalingConfig = pcsSvcCloneAnyMap(cfg)
	}
	if slurm := pcsSvcPayloadMap(payload, "slurmConfiguration"); len(slurm) > 0 {
		nodeGroup.SlurmConfiguration = pcsSvcCloneAnyMap(slurm)
	}
	if template := pcsSvcPayloadMap(payload, "customLaunchTemplate"); len(template) > 0 {
		nodeGroup.CustomTemplate = pcsSvcCloneAnyMap(template)
	}
	if spot := pcsSvcPayloadMap(payload, "spotOptions"); len(spot) > 0 {
		nodeGroup.SpotOptions = pcsSvcCloneAnyMap(spot)
	}
	if subnets := pcsSvcPayloadList(payload, "subnetIds"); len(subnets) > 0 {
		nodeGroup.SubnetIDs = pcsSvcCloneAnyList(subnets)
	}
	if instances := pcsSvcPayloadList(payload, "instanceConfigs"); len(instances) > 0 {
		nodeGroup.InstanceConfigs = pcsSvcCloneAnyList(instances)
	}
	nodeGroup.Status = "CREATING"
	nodeGroup.ModifiedAt = now
	nodeGroup.Status = "ACTIVE"

	for key, value := range pcsSvcPayloadStringMap(payload, "tags", "Tags") {
		s.ensureTagsLocked(nodeGroup.ARN)[key] = value
	}
	if token != "" {
		s.computeNodeGroupCreateTokens[token] = nodeGroup.ID
	}

	return map[string]any{"computeNodeGroup": pcsSvcComputeNodeGroupMap(nodeGroup)}
}

func (s *pcsStore) pcsCreateQueueLocked(payload map[string]any, now string) map[string]any {
	token := strings.TrimSpace(pcsSvcPayloadString(payload, "clientToken"))
	if token != "" {
		if queueID, ok := s.queueCreateTokens[token]; ok {
			queue := s.resolveQueueLocked("", queueID)
			return map[string]any{"queue": pcsSvcQueueMap(queue)}
		}
	}

	cluster := s.resolveClusterLocked(pcsSvcPayloadString(payload, "clusterIdentifier", "clusterId", "clusterName"))
	name := pcsSvcPayloadString(payload, "queueName", "name")
	if name == "" {
		name = s.nextNameLocked("queue")
	}

	queue := s.findQueueByNameLocked(cluster.ID, name)
	if queue == nil {
		queueID := s.nextIdentifierLocked("queue")
		queue = s.ensureQueueLocked(queueID, cluster.ID, name, now)
	}
	if cfg := pcsSvcPayloadList(payload, "computeNodeGroupConfigurations"); len(cfg) > 0 {
		queue.ComputeNodeGroupConfig = pcsSvcCloneAnyList(cfg)
	}
	if len(queue.ComputeNodeGroupConfig) == 0 {
		defaultNodeGroup := s.resolveComputeNodeGroupLocked(cluster.ID, "")
		queue.ComputeNodeGroupConfig = []any{
			map[string]any{"computeNodeGroupId": defaultNodeGroup.ID},
		}
	}
	if slurm := pcsSvcPayloadMap(payload, "slurmConfiguration"); len(slurm) > 0 {
		queue.SlurmConfiguration = pcsSvcCloneAnyMap(slurm)
	}
	queue.Status = "CREATING"
	queue.ModifiedAt = now
	queue.Status = "ACTIVE"

	for key, value := range pcsSvcPayloadStringMap(payload, "tags", "Tags") {
		s.ensureTagsLocked(queue.ARN)[key] = value
	}
	if token != "" {
		s.queueCreateTokens[token] = queue.ID
	}

	return map[string]any{"queue": pcsSvcQueueMap(queue)}
}

func (s *pcsStore) resolveClusterLocked(identifier string) *pcsCluster {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		for _, cluster := range s.sortedClustersLocked() {
			return cluster
		}
		return s.ensureClusterLocked("pcs_cluster_000001", "stackyard-cluster", time.Now().UTC().Format(time.RFC3339))
	}

	if cluster, ok := s.clusters[identifier]; ok {
		return cluster
	}
	for _, cluster := range s.clusters {
		if strings.EqualFold(cluster.Name, identifier) || strings.EqualFold(cluster.ARN, identifier) {
			return cluster
		}
	}

	id := identifier
	if strings.HasPrefix(identifier, "arn:aws:") {
		if seg := strings.Split(identifier, "/"); len(seg) > 1 {
			id = seg[len(seg)-1]
		}
	}
	if !strings.HasPrefix(id, "pcs_cluster_") {
		id = s.nextIdentifierLocked("cluster")
	}
	return s.ensureClusterLocked(id, "stackyard-"+id, time.Now().UTC().Format(time.RFC3339))
}

func (s *pcsStore) resolveComputeNodeGroupLocked(clusterID, identifier string) *pcsComputeNodeGroup {
	identifier = strings.TrimSpace(identifier)
	if identifier != "" {
		if nodeGroup, ok := s.computeNodeGroups[identifier]; ok {
			return nodeGroup
		}
		for _, nodeGroup := range s.computeNodeGroups {
			if (clusterID == "" || nodeGroup.ClusterID == clusterID) &&
				(strings.EqualFold(nodeGroup.Name, identifier) || strings.EqualFold(nodeGroup.ARN, identifier)) {
				return nodeGroup
			}
		}
	}

	for _, nodeGroup := range s.sortedComputeNodeGroupsLocked() {
		if clusterID == "" || nodeGroup.ClusterID == clusterID {
			return nodeGroup
		}
	}

	cluster := s.resolveClusterLocked(clusterID)
	id := identifier
	if id == "" || !strings.HasPrefix(id, "pcs_cng_") {
		id = s.nextIdentifierLocked("computenodegroup")
	}
	return s.ensureComputeNodeGroupLocked(id, cluster.ID, "stackyard-"+id, time.Now().UTC().Format(time.RFC3339))
}

func (s *pcsStore) resolveQueueLocked(clusterID, identifier string) *pcsQueue {
	identifier = strings.TrimSpace(identifier)
	if identifier != "" {
		if queue, ok := s.queues[identifier]; ok {
			return queue
		}
		for _, queue := range s.queues {
			if (clusterID == "" || queue.ClusterID == clusterID) &&
				(strings.EqualFold(queue.Name, identifier) || strings.EqualFold(queue.ARN, identifier)) {
				return queue
			}
		}
	}

	for _, queue := range s.sortedQueuesLocked() {
		if clusterID == "" || queue.ClusterID == clusterID {
			return queue
		}
	}

	cluster := s.resolveClusterLocked(clusterID)
	id := identifier
	if id == "" || !strings.HasPrefix(id, "pcs_queue_") {
		id = s.nextIdentifierLocked("queue")
	}
	return s.ensureQueueLocked(id, cluster.ID, "stackyard-"+id, time.Now().UTC().Format(time.RFC3339))
}

func (s *pcsStore) ensureClusterLocked(id, name, now string) *pcsCluster {
	if existing, ok := s.clusters[id]; ok {
		return existing
	}
	if strings.TrimSpace(name) == "" {
		name = "stackyard-cluster-" + id
	}

	cluster := &pcsCluster{
		ID:         id,
		Name:       name,
		ARN:        pcsSvcResourceARN("cluster", id),
		Size:       "SMALL",
		Status:     "ACTIVE",
		CreatedAt:  now,
		ModifiedAt: now,
		Networking: map[string]any{
			"networkType":      "VPC",
			"subnetIds":        []any{"subnet-00000001", "subnet-00000002"},
			"securityGroupIds": []any{"sg-00000001"},
		},
		Scheduler: map[string]any{
			"type":    "SLURM",
			"version": "23.11",
		},
		SlurmConfiguration: map[string]any{
			"scaleDownIdleTimeInSeconds": 600,
		},
		Endpoints: []any{
			map[string]any{
				"type":             "CLUSTER_CONTROLLER",
				"privateIpAddress": "10.0.0.10",
				"publicIpAddress":  "54.0.0.10",
				"port":             "6817",
			},
		},
	}
	s.clusters[id] = cluster
	s.ensureTagsLocked(cluster.ARN)
	return cluster
}

func (s *pcsStore) ensureComputeNodeGroupLocked(id, clusterID, name, now string) *pcsComputeNodeGroup {
	if existing, ok := s.computeNodeGroups[id]; ok {
		return existing
	}
	if strings.TrimSpace(clusterID) == "" {
		clusterID = s.resolveClusterLocked("").ID
	}
	if strings.TrimSpace(name) == "" {
		name = "stackyard-compute-node-group-" + id
	}

	nodeGroup := &pcsComputeNodeGroup{
		ID:                 id,
		Name:               name,
		ARN:                pcsSvcResourceARN("computenodegroup", id),
		ClusterID:          clusterID,
		Status:             "ACTIVE",
		CreatedAt:          now,
		ModifiedAt:         now,
		AMIID:              "ami-00000000000000001",
		IAMInstanceProfile: "arn:aws:iam::123456789012:instance-profile/AWSPCSDefault",
		PurchaseOption:     "ONDEMAND",
		InstanceConfigs: []any{
			map[string]any{"instanceType": "c7i.large"},
		},
		ScalingConfig: map[string]any{
			"minInstanceCount": 0,
			"maxInstanceCount": 4,
		},
		SubnetIDs: []any{"subnet-00000001"},
		SlurmConfiguration: map[string]any{
			"slurmCustomSettings": []any{},
		},
		CustomTemplate: map[string]any{
			"id":      "lt-00000001",
			"version": "1",
		},
		SpotOptions: map[string]any{
			"allocationStrategy": "lowest-price",
		},
	}
	s.computeNodeGroups[id] = nodeGroup
	s.ensureTagsLocked(nodeGroup.ARN)
	return nodeGroup
}

func (s *pcsStore) ensureQueueLocked(id, clusterID, name, now string) *pcsQueue {
	if existing, ok := s.queues[id]; ok {
		return existing
	}
	if strings.TrimSpace(clusterID) == "" {
		clusterID = s.resolveClusterLocked("").ID
	}
	if strings.TrimSpace(name) == "" {
		name = "stackyard-queue-" + id
	}

	queue := &pcsQueue{
		ID:                     id,
		Name:                   name,
		ARN:                    pcsSvcResourceARN("queue", id),
		ClusterID:              clusterID,
		Status:                 "ACTIVE",
		CreatedAt:              now,
		ModifiedAt:             now,
		ComputeNodeGroupConfig: []any{},
		SlurmConfiguration: map[string]any{
			"slurmCustomSettings": []any{},
		},
	}
	s.queues[id] = queue
	s.ensureTagsLocked(queue.ARN)
	return queue
}

func (s *pcsStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = s.defaultClusterARNLocked()
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	created := map[string]string{}
	s.tags[resourceARN] = created
	return created
}

func (s *pcsStore) defaultClusterARNLocked() string {
	return s.resolveClusterLocked("").ARN
}

func (s *pcsStore) findClusterByNameLocked(name string) *pcsCluster {
	name = strings.TrimSpace(name)
	for _, cluster := range s.clusters {
		if strings.EqualFold(cluster.Name, name) {
			return cluster
		}
	}
	return nil
}

func (s *pcsStore) findComputeNodeGroupByNameLocked(clusterID, name string) *pcsComputeNodeGroup {
	name = strings.TrimSpace(name)
	for _, nodeGroup := range s.computeNodeGroups {
		if nodeGroup.ClusterID == clusterID && strings.EqualFold(nodeGroup.Name, name) {
			return nodeGroup
		}
	}
	return nil
}

func (s *pcsStore) findQueueByNameLocked(clusterID, name string) *pcsQueue {
	name = strings.TrimSpace(name)
	for _, queue := range s.queues {
		if queue.ClusterID == clusterID && strings.EqualFold(queue.Name, name) {
			return queue
		}
	}
	return nil
}

func (s *pcsStore) listClusterSummariesLocked() []any {
	items := make([]any, 0, len(s.clusters))
	for _, cluster := range s.sortedClustersLocked() {
		items = append(items, map[string]any{
			"id":         cluster.ID,
			"name":       cluster.Name,
			"arn":        cluster.ARN,
			"status":     cluster.Status,
			"createdAt":  cluster.CreatedAt,
			"modifiedAt": cluster.ModifiedAt,
		})
	}
	return items
}

func (s *pcsStore) listComputeNodeGroupSummariesLocked(clusterID string) []any {
	items := []any{}
	for _, nodeGroup := range s.sortedComputeNodeGroupsLocked() {
		if clusterID != "" && nodeGroup.ClusterID != clusterID {
			continue
		}
		items = append(items, map[string]any{
			"id":         nodeGroup.ID,
			"name":       nodeGroup.Name,
			"arn":        nodeGroup.ARN,
			"clusterId":  nodeGroup.ClusterID,
			"status":     nodeGroup.Status,
			"createdAt":  nodeGroup.CreatedAt,
			"modifiedAt": nodeGroup.ModifiedAt,
		})
	}
	return items
}

func (s *pcsStore) listQueueSummariesLocked(clusterID string) []any {
	items := []any{}
	for _, queue := range s.sortedQueuesLocked() {
		if clusterID != "" && queue.ClusterID != clusterID {
			continue
		}
		items = append(items, map[string]any{
			"id":         queue.ID,
			"name":       queue.Name,
			"arn":        queue.ARN,
			"clusterId":  queue.ClusterID,
			"status":     queue.Status,
			"createdAt":  queue.CreatedAt,
			"modifiedAt": queue.ModifiedAt,
		})
	}
	return items
}

func (s *pcsStore) sortedClustersLocked() []*pcsCluster {
	keys := make([]string, 0, len(s.clusters))
	for key := range s.clusters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*pcsCluster, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.clusters[key])
	}
	return out
}

func (s *pcsStore) sortedComputeNodeGroupsLocked() []*pcsComputeNodeGroup {
	keys := make([]string, 0, len(s.computeNodeGroups))
	for key := range s.computeNodeGroups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*pcsComputeNodeGroup, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.computeNodeGroups[key])
	}
	return out
}

func (s *pcsStore) sortedQueuesLocked() []*pcsQueue {
	keys := make([]string, 0, len(s.queues))
	for key := range s.queues {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*pcsQueue, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.queues[key])
	}
	return out
}

func (s *pcsStore) nextIdentifierLocked(kind string) string {
	id := s.nextID
	s.nextID++

	switch kind {
	case "cluster":
		return fmt.Sprintf("pcs_cluster_%06d", id)
	case "computenodegroup":
		return fmt.Sprintf("pcs_cng_%06d", id)
	default:
		return fmt.Sprintf("pcs_queue_%06d", id)
	}
}

func (s *pcsStore) nextNameLocked(kind string) string {
	return fmt.Sprintf("stackyard-%s-%06d", kind, s.nextID)
}

func pcsSvcResourceARN(resourceType, id string) string {
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		resourceType = "cluster"
	}
	id = strings.TrimSpace(id)
	if id == "" {
		id = "pcs_cluster_000001"
	}
	return fmt.Sprintf("arn:aws:pcs:%s:%s:%s/%s", pcsDefaultRegion, pcsDefaultAccountID, resourceType, id)
}

func pcsSvcClusterMap(cluster *pcsCluster) map[string]any {
	if cluster == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":                 cluster.ID,
		"name":               cluster.Name,
		"arn":                cluster.ARN,
		"size":               cluster.Size,
		"status":             cluster.Status,
		"createdAt":          cluster.CreatedAt,
		"modifiedAt":         cluster.ModifiedAt,
		"networking":         pcsSvcCloneAnyMap(cluster.Networking),
		"scheduler":          pcsSvcCloneAnyMap(cluster.Scheduler),
		"slurmConfiguration": pcsSvcCloneAnyMap(cluster.SlurmConfiguration),
		"endpoints":          pcsSvcCloneAnyList(cluster.Endpoints),
		"errorInfo":          []any{},
	}
}

func pcsSvcComputeNodeGroupMap(nodeGroup *pcsComputeNodeGroup) map[string]any {
	if nodeGroup == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":                    nodeGroup.ID,
		"name":                  nodeGroup.Name,
		"arn":                   nodeGroup.ARN,
		"clusterId":             nodeGroup.ClusterID,
		"status":                nodeGroup.Status,
		"createdAt":             nodeGroup.CreatedAt,
		"modifiedAt":            nodeGroup.ModifiedAt,
		"amiId":                 nodeGroup.AMIID,
		"iamInstanceProfileArn": nodeGroup.IAMInstanceProfile,
		"purchaseOption":        nodeGroup.PurchaseOption,
		"instanceConfigs":       pcsSvcCloneAnyList(nodeGroup.InstanceConfigs),
		"scalingConfiguration":  pcsSvcCloneAnyMap(nodeGroup.ScalingConfig),
		"subnetIds":             pcsSvcCloneAnyList(nodeGroup.SubnetIDs),
		"slurmConfiguration":    pcsSvcCloneAnyMap(nodeGroup.SlurmConfiguration),
		"customLaunchTemplate":  pcsSvcCloneAnyMap(nodeGroup.CustomTemplate),
		"spotOptions":           pcsSvcCloneAnyMap(nodeGroup.SpotOptions),
		"errorInfo":             []any{},
	}
}

func pcsSvcQueueMap(queue *pcsQueue) map[string]any {
	if queue == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":                             queue.ID,
		"name":                           queue.Name,
		"arn":                            queue.ARN,
		"clusterId":                      queue.ClusterID,
		"status":                         queue.Status,
		"createdAt":                      queue.CreatedAt,
		"modifiedAt":                     queue.ModifiedAt,
		"computeNodeGroupConfigurations": pcsSvcCloneAnyList(queue.ComputeNodeGroupConfig),
		"slurmConfiguration":             pcsSvcCloneAnyMap(queue.SlurmConfiguration),
		"errorInfo":                      []any{},
	}
}

func pcsSvcPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		if str, ok := raw.(string); ok && strings.TrimSpace(str) != "" {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

func pcsSvcPayloadMap(payload map[string]any, key string) map[string]any {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return map[string]any{}
	}
	if out, ok := raw.(map[string]any); ok {
		return out
	}
	return map[string]any{}
}

func pcsSvcPayloadList(payload map[string]any, key string) []any {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	if list, ok := raw.([]any); ok {
		return list
	}
	return nil
}

func pcsSvcPayloadStringMap(payload map[string]any, keys ...string) map[string]string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case map[string]any:
			out := map[string]string{}
			for k, v := range typed {
				if str, ok := v.(string); ok {
					out[k] = str
				}
			}
			return out
		case map[string]string:
			out := map[string]string{}
			for k, v := range typed {
				out[k] = v
			}
			return out
		}
	}
	return map[string]string{}
}

func pcsSvcPayloadStringSlice(payload map[string]any, keys ...string) []string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case []any:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				if str, ok := item.(string); ok {
					str = strings.TrimSpace(str)
					if str != "" {
						out = append(out, str)
					}
				}
			}
			return out
		case []string:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			return out
		}
	}
	return nil
}

func pcsSvcStringMapToAny(in map[string]string) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func pcsSvcCloneAnyMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		out[key] = pcsSvcCloneAny(value)
	}
	return out
}

func pcsSvcCloneAnyList(in []any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, pcsSvcCloneAny(item))
	}
	return out
}

func pcsSvcCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return pcsSvcCloneAnyMap(typed)
	case []any:
		return pcsSvcCloneAnyList(typed)
	default:
		return typed
	}
}
