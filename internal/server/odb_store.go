package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	odbDefaultRegion    = "us-east-1"
	odbDefaultAccountID = "123456789012"
)

type odbStore struct {
	mu sync.Mutex

	nextID int64

	initialized bool
	onboarding  string

	networks             map[string]map[string]any
	peerings             map[string]map[string]any
	exadataInfras        map[string]map[string]any
	vmClusters           map[string]map[string]any
	autonomousVmClusters map[string]map[string]any
	autonomousVms        map[string]map[string]any
	dbNodes              map[string]map[string]any
	dbServers            map[string]map[string]any

	iamRoles map[string]map[string]bool
	tags     map[string]map[string]string
}

func newODBStore() *odbStore {
	s := &odbStore{
		nextID: 2,

		initialized: true,
		onboarding:  "ONBOARDED",

		networks:             map[string]map[string]any{},
		peerings:             map[string]map[string]any{},
		exadataInfras:        map[string]map[string]any{},
		vmClusters:           map[string]map[string]any{},
		autonomousVmClusters: map[string]map[string]any{},
		autonomousVms:        map[string]map[string]any{},
		dbNodes:              map[string]map[string]any{},
		dbServers:            map[string]map[string]any{},

		iamRoles: map[string]map[string]bool{},
		tags:     map[string]map[string]string{},
	}
	s.ensureSeedDataLocked()
	return s
}

func (s *odbStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSeedDataLocked()
	now := time.Now().UTC().Format(time.RFC3339)
	reqID := s.nextTokenLocked("req")

	input := odbCloneMap(payload)
	for k, v := range pathParams {
		input[k] = v
	}
	for k, values := range query {
		if len(values) == 0 {
			continue
		}
		input[k] = values[len(values)-1]
	}

	defaultNetworkID := s.firstIDLocked(s.networks)
	defaultPeeringID := s.firstIDLocked(s.peerings)
	defaultExadataID := s.firstIDLocked(s.exadataInfras)
	defaultVMClusterID := s.firstIDLocked(s.vmClusters)
	defaultAutonomousClusterID := s.firstIDLocked(s.autonomousVmClusters)
	defaultDbNodeID := s.firstIDLocked(s.dbNodes)
	defaultDbServerID := s.firstIDLocked(s.dbServers)

	switch action {
	case "InitializeService":
		s.initialized = true
		s.onboarding = "ONBOARDED"
		return map[string]any{"requestId": reqID, "status": s.onboarding}

	case "GetOciOnboardingStatus":
		status := "NOT_INITIALIZED"
		if s.initialized {
			status = s.onboarding
		}
		return map[string]any{"requestId": reqID, "status": status}

	case "AcceptMarketplaceRegistration":
		return map[string]any{"requestId": reqID, "status": "ACCEPTED"}

	case "AssociateIamRoleToResource":
		resourceARN := odbString(input, []string{"resourceArn", "ResourceArn"}, odbARN("resource", "stackyard"))
		roleARN := odbString(input, []string{"iamRoleArn", "IamRoleArn", "roleArn", "RoleArn"}, odbARN("iam-role", "stackyard-odb-role"))
		roles := s.ensureRolesForResourceLocked(resourceARN)
		roles[roleARN] = true
		return map[string]any{"requestId": reqID, "resourceArn": resourceARN, "iamRoleArn": roleARN}

	case "DisassociateIamRoleFromResource":
		resourceARN := odbString(input, []string{"resourceArn", "ResourceArn"}, odbARN("resource", "stackyard"))
		roleARN := odbString(input, []string{"iamRoleArn", "IamRoleArn", "roleArn", "RoleArn"}, "")
		roles := s.ensureRolesForResourceLocked(resourceARN)
		if roleARN != "" {
			delete(roles, roleARN)
		}
		return map[string]any{"requestId": reqID}

	case "CreateOdbNetwork":
		id := odbString(input, []string{"odbNetworkId", "OdbNetworkId", "networkId", "id", "name"}, s.nextTokenLocked("odb-network"))
		resource := s.ensureResourceLocked(s.networks, id, "odb-network", now)
		resource["displayName"] = odbString(input, []string{"displayName", "name", "Name"}, odbStringAny(resource["displayName"]))
		resource["status"] = "AVAILABLE"
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "odbNetworkId": id, "odbNetwork": odbCloneMap(resource)}

	case "GetOdbNetwork":
		id := odbString(input, []string{"odbNetworkId", "OdbNetworkId", "networkId", "id"}, defaultNetworkID)
		resource := s.ensureResourceLocked(s.networks, id, "odb-network", now)
		return map[string]any{"requestId": reqID, "odbNetwork": odbCloneMap(resource)}

	case "ListOdbNetworks":
		return map[string]any{"requestId": reqID, "odbNetworks": s.listResourcesLocked(s.networks), "nextToken": ""}

	case "UpdateOdbNetwork":
		id := odbString(input, []string{"odbNetworkId", "OdbNetworkId", "networkId", "id"}, defaultNetworkID)
		resource := s.ensureResourceLocked(s.networks, id, "odb-network", now)
		if name := odbString(input, []string{"displayName", "name", "Name"}, ""); name != "" {
			resource["displayName"] = name
		}
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "odbNetwork": odbCloneMap(resource)}

	case "DeleteOdbNetwork":
		id := odbString(input, []string{"odbNetworkId", "OdbNetworkId", "networkId", "id"}, defaultNetworkID)
		resource := s.ensureResourceLocked(s.networks, id, "odb-network", now)
		arn := odbString(resource, []string{"resourceArn"}, "")
		delete(s.networks, id)
		delete(s.tags, arn)
		delete(s.iamRoles, arn)
		return map[string]any{"requestId": reqID}

	case "CreateOdbPeeringConnection":
		id := odbString(input, []string{"odbPeeringConnectionId", "OdbPeeringConnectionId", "peeringConnectionId", "id", "name"}, s.nextTokenLocked("odb-peering"))
		resource := s.ensureResourceLocked(s.peerings, id, "odb-peering-connection", now)
		resource["displayName"] = odbString(input, []string{"displayName", "name", "Name"}, odbStringAny(resource["displayName"]))
		resource["status"] = "ACTIVE"
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "odbPeeringConnectionId": id, "odbPeeringConnection": odbCloneMap(resource)}

	case "GetOdbPeeringConnection":
		id := odbString(input, []string{"odbPeeringConnectionId", "OdbPeeringConnectionId", "peeringConnectionId", "id"}, defaultPeeringID)
		resource := s.ensureResourceLocked(s.peerings, id, "odb-peering-connection", now)
		return map[string]any{"requestId": reqID, "odbPeeringConnection": odbCloneMap(resource)}

	case "ListOdbPeeringConnections":
		return map[string]any{"requestId": reqID, "odbPeeringConnections": s.listResourcesLocked(s.peerings), "nextToken": ""}

	case "UpdateOdbPeeringConnection":
		id := odbString(input, []string{"odbPeeringConnectionId", "OdbPeeringConnectionId", "peeringConnectionId", "id"}, defaultPeeringID)
		resource := s.ensureResourceLocked(s.peerings, id, "odb-peering-connection", now)
		if status := odbString(input, []string{"status", "Status"}, ""); status != "" {
			resource["status"] = status
		}
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "odbPeeringConnection": odbCloneMap(resource)}

	case "DeleteOdbPeeringConnection":
		id := odbString(input, []string{"odbPeeringConnectionId", "OdbPeeringConnectionId", "peeringConnectionId", "id"}, defaultPeeringID)
		resource := s.ensureResourceLocked(s.peerings, id, "odb-peering-connection", now)
		arn := odbString(resource, []string{"resourceArn"}, "")
		delete(s.peerings, id)
		delete(s.tags, arn)
		delete(s.iamRoles, arn)
		return map[string]any{"requestId": reqID}

	case "CreateCloudExadataInfrastructure":
		id := odbString(input, []string{"cloudExadataInfrastructureId", "CloudExadataInfrastructureId", "id", "name"}, s.nextTokenLocked("exadata-infra"))
		resource := s.ensureResourceLocked(s.exadataInfras, id, "cloud-exadata-infrastructure", now)
		resource["displayName"] = odbString(input, []string{"displayName", "name", "Name"}, odbStringAny(resource["displayName"]))
		resource["status"] = "AVAILABLE"
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "cloudExadataInfrastructureId": id, "cloudExadataInfrastructure": odbCloneMap(resource)}

	case "GetCloudExadataInfrastructure":
		id := odbString(input, []string{"cloudExadataInfrastructureId", "CloudExadataInfrastructureId", "id"}, defaultExadataID)
		resource := s.ensureResourceLocked(s.exadataInfras, id, "cloud-exadata-infrastructure", now)
		return map[string]any{"requestId": reqID, "cloudExadataInfrastructure": odbCloneMap(resource)}

	case "ListCloudExadataInfrastructures":
		return map[string]any{"requestId": reqID, "cloudExadataInfrastructures": s.listResourcesLocked(s.exadataInfras), "nextToken": ""}

	case "UpdateCloudExadataInfrastructure":
		id := odbString(input, []string{"cloudExadataInfrastructureId", "CloudExadataInfrastructureId", "id"}, defaultExadataID)
		resource := s.ensureResourceLocked(s.exadataInfras, id, "cloud-exadata-infrastructure", now)
		if displayName := odbString(input, []string{"displayName", "name", "Name"}, ""); displayName != "" {
			resource["displayName"] = displayName
		}
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "cloudExadataInfrastructure": odbCloneMap(resource)}

	case "DeleteCloudExadataInfrastructure":
		id := odbString(input, []string{"cloudExadataInfrastructureId", "CloudExadataInfrastructureId", "id"}, defaultExadataID)
		resource := s.ensureResourceLocked(s.exadataInfras, id, "cloud-exadata-infrastructure", now)
		arn := odbString(resource, []string{"resourceArn"}, "")
		delete(s.exadataInfras, id)
		delete(s.tags, arn)
		delete(s.iamRoles, arn)
		return map[string]any{"requestId": reqID}

	case "GetCloudExadataInfrastructureUnallocatedResources":
		id := odbString(input, []string{"cloudExadataInfrastructureId", "CloudExadataInfrastructureId", "id"}, defaultExadataID)
		_ = s.ensureResourceLocked(s.exadataInfras, id, "cloud-exadata-infrastructure", now)
		return map[string]any{
			"requestId": reqID,
			"cloudExadataInfrastructureUnallocatedResources": map[string]any{
				"availableStorageInTBs": 10,
				"availableMemoryInGBs":  256,
				"availableCpuCount":     32,
			},
		}

	case "CreateCloudVmCluster":
		id := odbString(input, []string{"cloudVmClusterId", "CloudVmClusterId", "id", "name"}, s.nextTokenLocked("vm-cluster"))
		resource := s.ensureResourceLocked(s.vmClusters, id, "cloud-vm-cluster", now)
		resource["displayName"] = odbString(input, []string{"displayName", "name", "Name"}, odbStringAny(resource["displayName"]))
		resource["status"] = "AVAILABLE"
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "cloudVmClusterId": id, "cloudVmCluster": odbCloneMap(resource)}

	case "GetCloudVmCluster":
		id := odbString(input, []string{"cloudVmClusterId", "CloudVmClusterId", "id"}, defaultVMClusterID)
		resource := s.ensureResourceLocked(s.vmClusters, id, "cloud-vm-cluster", now)
		return map[string]any{"requestId": reqID, "cloudVmCluster": odbCloneMap(resource)}

	case "ListCloudVmClusters":
		return map[string]any{"requestId": reqID, "cloudVmClusters": s.listResourcesLocked(s.vmClusters), "nextToken": ""}

	case "DeleteCloudVmCluster":
		id := odbString(input, []string{"cloudVmClusterId", "CloudVmClusterId", "id"}, defaultVMClusterID)
		resource := s.ensureResourceLocked(s.vmClusters, id, "cloud-vm-cluster", now)
		arn := odbString(resource, []string{"resourceArn"}, "")
		delete(s.vmClusters, id)
		delete(s.tags, arn)
		delete(s.iamRoles, arn)
		return map[string]any{"requestId": reqID}

	case "CreateCloudAutonomousVmCluster":
		id := odbString(input, []string{"cloudAutonomousVmClusterId", "CloudAutonomousVmClusterId", "id", "name"}, s.nextTokenLocked("autonomous-cluster"))
		resource := s.ensureResourceLocked(s.autonomousVmClusters, id, "cloud-autonomous-vm-cluster", now)
		resource["displayName"] = odbString(input, []string{"displayName", "name", "Name"}, odbStringAny(resource["displayName"]))
		resource["status"] = "AVAILABLE"
		resource["updatedAt"] = now
		s.ensureAutonomousVMForClusterLocked(id, now)
		return map[string]any{"requestId": reqID, "cloudAutonomousVmClusterId": id, "cloudAutonomousVmCluster": odbCloneMap(resource)}

	case "GetCloudAutonomousVmCluster":
		id := odbString(input, []string{"cloudAutonomousVmClusterId", "CloudAutonomousVmClusterId", "id"}, defaultAutonomousClusterID)
		resource := s.ensureResourceLocked(s.autonomousVmClusters, id, "cloud-autonomous-vm-cluster", now)
		return map[string]any{"requestId": reqID, "cloudAutonomousVmCluster": odbCloneMap(resource)}

	case "ListCloudAutonomousVmClusters":
		return map[string]any{"requestId": reqID, "cloudAutonomousVmClusters": s.listResourcesLocked(s.autonomousVmClusters), "nextToken": ""}

	case "DeleteCloudAutonomousVmCluster":
		id := odbString(input, []string{"cloudAutonomousVmClusterId", "CloudAutonomousVmClusterId", "id"}, defaultAutonomousClusterID)
		resource := s.ensureResourceLocked(s.autonomousVmClusters, id, "cloud-autonomous-vm-cluster", now)
		arn := odbString(resource, []string{"resourceArn"}, "")
		delete(s.autonomousVmClusters, id)
		delete(s.tags, arn)
		delete(s.iamRoles, arn)
		for vmID, vm := range s.autonomousVms {
			if strings.EqualFold(odbString(vm, []string{"cloudAutonomousVmClusterId"}, ""), id) {
				delete(s.autonomousVms, vmID)
			}
		}
		return map[string]any{"requestId": reqID}

	case "ListAutonomousVirtualMachines":
		return map[string]any{"requestId": reqID, "autonomousVirtualMachines": s.listResourcesLocked(s.autonomousVms), "nextToken": ""}

	case "GetDbNode":
		id := odbString(input, []string{"dbNodeId", "DbNodeId", "id"}, defaultDbNodeID)
		resource := s.ensureResourceLocked(s.dbNodes, id, "db-node", now)
		return map[string]any{"requestId": reqID, "dbNode": odbCloneMap(resource)}

	case "ListDbNodes":
		return map[string]any{"requestId": reqID, "dbNodes": s.listResourcesLocked(s.dbNodes), "nextToken": ""}

	case "RebootDbNode":
		id := odbString(input, []string{"dbNodeId", "DbNodeId", "id"}, defaultDbNodeID)
		resource := s.ensureResourceLocked(s.dbNodes, id, "db-node", now)
		resource["status"] = "REBOOTING"
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "dbNode": odbCloneMap(resource)}

	case "StartDbNode":
		id := odbString(input, []string{"dbNodeId", "DbNodeId", "id"}, defaultDbNodeID)
		resource := s.ensureResourceLocked(s.dbNodes, id, "db-node", now)
		resource["status"] = "AVAILABLE"
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "dbNode": odbCloneMap(resource)}

	case "StopDbNode":
		id := odbString(input, []string{"dbNodeId", "DbNodeId", "id"}, defaultDbNodeID)
		resource := s.ensureResourceLocked(s.dbNodes, id, "db-node", now)
		resource["status"] = "STOPPED"
		resource["updatedAt"] = now
		return map[string]any{"requestId": reqID, "dbNode": odbCloneMap(resource)}

	case "GetDbServer":
		id := odbString(input, []string{"dbServerId", "DbServerId", "id"}, defaultDbServerID)
		resource := s.ensureResourceLocked(s.dbServers, id, "db-server", now)
		return map[string]any{"requestId": reqID, "dbServer": odbCloneMap(resource)}

	case "ListDbServers":
		return map[string]any{"requestId": reqID, "dbServers": s.listResourcesLocked(s.dbServers), "nextToken": ""}

	case "ListDbSystemShapes":
		return map[string]any{
			"requestId": reqID,
			"dbSystemShapes": []any{
				map[string]any{"name": "odb.x8m", "coreCount": 32, "memoryInGBs": 512},
				map[string]any{"name": "odb.x9m", "coreCount": 64, "memoryInGBs": 1024},
			},
			"nextToken": "",
		}

	case "ListGiVersions":
		return map[string]any{
			"requestId": reqID,
			"giVersions": []any{
				map[string]any{"version": "23.0.0.0", "status": "AVAILABLE"},
				map[string]any{"version": "21.0.0.0", "status": "AVAILABLE"},
			},
			"nextToken": "",
		}

	case "ListSystemVersions":
		return map[string]any{
			"requestId": reqID,
			"systemVersions": []any{
				map[string]any{"version": "v1", "status": "CURRENT"},
			},
			"nextToken": "",
		}

	case "TagResource":
		resourceARN := odbString(input, []string{"resourceArn", "ResourceArn"}, odbARN("resource", "stackyard"))
		tags := odbTagsFromAny(input["tags"])
		if len(tags) == 0 {
			tags = odbTagsFromAny(input["Tags"])
		}
		existing := s.ensureTagsLocked(resourceARN)
		for k, v := range tags {
			existing[k] = v
		}
		return map[string]any{"requestId": reqID}

	case "UntagResource":
		resourceARN := odbString(input, []string{"resourceArn", "ResourceArn"}, odbARN("resource", "stackyard"))
		tagKeys := odbStringSlice(input["tagKeys"])
		if len(tagKeys) == 0 {
			tagKeys = odbStringSlice(input["TagKeys"])
		}
		existing := s.ensureTagsLocked(resourceARN)
		for _, key := range tagKeys {
			delete(existing, key)
		}
		return map[string]any{"requestId": reqID}

	case "ListTagsForResource":
		resourceARN := odbString(input, []string{"resourceArn", "ResourceArn"}, odbARN("resource", "stackyard"))
		return map[string]any{"requestId": reqID, "tags": odbCloneStringMap(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{"requestId": reqID}
}

func (s *odbStore) ensureSeedDataLocked() {
	now := time.Now().UTC().Format(time.RFC3339)

	network := s.ensureResourceLocked(s.networks, "odb-network-000001", "odb-network", now)
	network["displayName"] = "stackyard-network"

	peering := s.ensureResourceLocked(s.peerings, "odb-peering-000001", "odb-peering-connection", now)
	peering["displayName"] = "stackyard-peering"
	peering["status"] = "ACTIVE"

	exadata := s.ensureResourceLocked(s.exadataInfras, "exadata-infra-000001", "cloud-exadata-infrastructure", now)
	exadata["displayName"] = "stackyard-exadata-infra"

	vmCluster := s.ensureResourceLocked(s.vmClusters, "vm-cluster-000001", "cloud-vm-cluster", now)
	vmCluster["displayName"] = "stackyard-vm-cluster"
	vmCluster["cloudExadataInfrastructureId"] = "exadata-infra-000001"

	autonomousCluster := s.ensureResourceLocked(s.autonomousVmClusters, "autonomous-cluster-000001", "cloud-autonomous-vm-cluster", now)
	autonomousCluster["displayName"] = "stackyard-autonomous-cluster"
	autonomousCluster["cloudExadataInfrastructureId"] = "exadata-infra-000001"
	s.ensureAutonomousVMForClusterLocked("autonomous-cluster-000001", now)

	dbNode := s.ensureResourceLocked(s.dbNodes, "db-node-000001", "db-node", now)
	dbNode["displayName"] = "stackyard-db-node"
	dbNode["status"] = "AVAILABLE"

	dbServer := s.ensureResourceLocked(s.dbServers, "db-server-000001", "db-server", now)
	dbServer["displayName"] = "stackyard-db-server"
	dbServer["status"] = "AVAILABLE"

	s.ensureTagsLocked(odbString(network, []string{"resourceArn"}, ""))["service"] = "odb"
	s.ensureTagsLocked(odbString(exadata, []string{"resourceArn"}, ""))["service"] = "odb"
}

func (s *odbStore) ensureResourceLocked(target map[string]map[string]any, id, kind, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.nextTokenLocked(kind)
	}

	if item, ok := target[id]; ok {
		item["id"] = id
		item["resourceId"] = id
		if _, exists := item["resourceArn"]; !exists {
			item["resourceArn"] = odbARN(kind, id)
		}
		if _, exists := item["createdAt"]; !exists {
			item["createdAt"] = now
		}
		item["updatedAt"] = now
		return item
	}

	item := map[string]any{
		"id":          id,
		"resourceId":  id,
		"resourceArn": odbARN(kind, id),
		"displayName": id,
		"status":      "AVAILABLE",
		"type":        kind,
		"createdAt":   now,
		"updatedAt":   now,
	}
	target[id] = item
	return item
}

func (s *odbStore) ensureAutonomousVMForClusterLocked(clusterID, now string) {
	vmID := "autonomous-vm-" + strings.TrimPrefix(clusterID, "autonomous-cluster-")
	resource := s.ensureResourceLocked(s.autonomousVms, vmID, "autonomous-virtual-machine", now)
	resource["cloudAutonomousVmClusterId"] = clusterID
	resource["displayName"] = "stackyard-autonomous-vm"
	resource["status"] = "AVAILABLE"
}

func (s *odbStore) ensureRolesForResourceLocked(resourceARN string) map[string]bool {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = odbARN("resource", "stackyard")
	}
	existing, ok := s.iamRoles[resourceARN]
	if ok {
		return existing
	}
	existing = map[string]bool{}
	s.iamRoles[resourceARN] = existing
	return existing
}

func (s *odbStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = odbARN("resource", "stackyard")
	}
	existing, ok := s.tags[resourceARN]
	if ok {
		return existing
	}
	existing = map[string]string{}
	s.tags[resourceARN] = existing
	return existing
}

func (s *odbStore) listResourcesLocked(target map[string]map[string]any) []any {
	ids := make([]string, 0, len(target))
	for id := range target {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, odbCloneMap(target[id]))
	}
	return out
}

func (s *odbStore) firstIDLocked(target map[string]map[string]any) string {
	if len(target) == 0 {
		return ""
	}
	ids := make([]string, 0, len(target))
	for id := range target {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids[0]
}

func (s *odbStore) nextTokenLocked(prefix string) string {
	clean := strings.Trim(prefix, "-_ ")
	if clean == "" {
		clean = "odb"
	}
	token := fmt.Sprintf("%s-%06d", clean, s.nextID)
	s.nextID++
	return token
}

func odbARN(kind, id string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "resource"
	}
	id = strings.TrimSpace(id)
	if id == "" {
		id = "stackyard"
	}
	return fmt.Sprintf("arn:aws:odb:%s:%s:%s/%s", odbDefaultRegion, odbDefaultAccountID, kind, id)
}

func odbString(payload map[string]any, keys []string, def string) string {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if value, ok := payload[key]; ok {
			if s := odbStringAny(value); s != "" {
				return s
			}
		}
		for existingKey, value := range payload {
			if strings.EqualFold(strings.TrimSpace(existingKey), key) {
				if s := odbStringAny(value); s != "" {
					return s
				}
			}
		}
	}
	return strings.TrimSpace(def)
}

func odbStringAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func odbTagsFromAny(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]any:
		for k, v := range typed {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = odbStringAny(v)
		}
	case map[string]string:
		for k, v := range typed {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(v)
		}
	case []any:
		for _, item := range typed {
			asMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := odbString(asMap, []string{"key", "Key"}, "")
			if key == "" {
				continue
			}
			out[key] = odbString(asMap, []string{"value", "Value"}, "")
		}
	}
	return out
}

func odbStringSlice(value any) []string {
	out := []string{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if s := odbStringAny(item); s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, item := range typed {
			if s := strings.TrimSpace(item); s != "" {
				out = append(out, s)
			}
		}
	case string:
		if s := strings.TrimSpace(typed); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func odbCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func odbCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
