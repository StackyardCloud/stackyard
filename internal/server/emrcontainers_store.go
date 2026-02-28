package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type emrContainersStore struct {
	mu sync.Mutex

	nextID int64

	virtualClusters        map[string]*emrContainersVirtualCluster
	jobRuns                map[string]*emrContainersJobRun
	managedEndpoints       map[string]*emrContainersManagedEndpoint
	jobTemplates           map[string]*emrContainersJobTemplate
	securityConfigurations map[string]*emrContainersSecurityConfiguration
	tags                   map[string]map[string]string
}

type emrContainersVirtualCluster struct {
	ID             string
	Name           string
	ARN            string
	State          string
	ContainerInfo  map[string]any
	SecurityConfig string
	CreatedAt      string
	LastModifiedAt string
}

type emrContainersJobRun struct {
	ID               string
	Name             string
	ARN              string
	VirtualClusterID string
	ExecutionRoleARN string
	ReleaseLabel     string
	State            string
	CreatedAt        string
	FinishedAt       string
}

type emrContainersManagedEndpoint struct {
	ID               string
	Name             string
	ARN              string
	VirtualClusterID string
	ExecutionRoleARN string
	ReleaseLabel     string
	State            string
	CreatedAt        string
}

type emrContainersJobTemplate struct {
	ID        string
	Name      string
	ARN       string
	CreatedAt string
}

type emrContainersSecurityConfiguration struct {
	ID        string
	Name      string
	ARN       string
	CreatedAt string
}

func newEMRContainersStore() *emrContainersStore {
	now := time.Now().UTC().Format(time.RFC3339)
	seedVC := &emrContainersVirtualCluster{
		ID:             "vc-000001",
		Name:           "stackyard-emr-on-eks",
		ARN:            emrContainersVirtualClusterARN("vc-000001"),
		State:          "RUNNING",
		ContainerInfo:  map[string]any{"eksInfo": map[string]any{"namespace": "default"}},
		SecurityConfig: "sc-000001",
		CreatedAt:      now,
		LastModifiedAt: now,
	}
	seedJob := &emrContainersJobRun{
		ID:               "jr-000001",
		Name:             "stackyard-job-run",
		ARN:              emrContainersJobRunARN(seedVC.ID, "jr-000001"),
		VirtualClusterID: seedVC.ID,
		ExecutionRoleARN: "arn:aws:iam::123456789012:role/stackyard-emr-containers-execution-role",
		ReleaseLabel:     "emr-7.0.0-latest",
		State:            "COMPLETED",
		CreatedAt:        now,
		FinishedAt:       now,
	}
	seedEndpoint := &emrContainersManagedEndpoint{
		ID:               "me-000001",
		Name:             "stackyard-managed-endpoint",
		ARN:              emrContainersManagedEndpointARN(seedVC.ID, "me-000001"),
		VirtualClusterID: seedVC.ID,
		ExecutionRoleARN: "arn:aws:iam::123456789012:role/stackyard-emr-containers-execution-role",
		ReleaseLabel:     "emr-7.0.0-latest",
		State:            "ACTIVE",
		CreatedAt:        now,
	}
	seedTemplate := &emrContainersJobTemplate{
		ID:        "jt-000001",
		Name:      "stackyard-job-template",
		ARN:       emrContainersJobTemplateARN("jt-000001"),
		CreatedAt: now,
	}
	seedSecurityConfig := &emrContainersSecurityConfiguration{
		ID:        "sc-000001",
		Name:      "stackyard-security-configuration",
		ARN:       emrContainersSecurityConfigurationARN("sc-000001"),
		CreatedAt: now,
	}

	tags := map[string]map[string]string{}
	tags[seedVC.ARN] = map[string]string{"stackyard": "true", "service": "emrcontainers"}
	tags[seedJob.ARN] = map[string]string{"stackyard": "true", "service": "emrcontainers"}
	tags[seedEndpoint.ARN] = map[string]string{"stackyard": "true", "service": "emrcontainers"}
	tags[seedTemplate.ARN] = map[string]string{"stackyard": "true", "service": "emrcontainers"}
	tags[seedSecurityConfig.ARN] = map[string]string{"stackyard": "true", "service": "emrcontainers"}

	return &emrContainersStore{
		nextID: 2,
		virtualClusters: map[string]*emrContainersVirtualCluster{
			seedVC.ID: seedVC,
		},
		jobRuns: map[string]*emrContainersJobRun{
			seedJob.ID: seedJob,
		},
		managedEndpoints: map[string]*emrContainersManagedEndpoint{
			seedEndpoint.ID: seedEndpoint,
		},
		jobTemplates: map[string]*emrContainersJobTemplate{
			seedTemplate.ID: seedTemplate,
		},
		securityConfigurations: map[string]*emrContainersSecurityConfiguration{
			seedSecurityConfig.ID: seedSecurityConfig,
		},
		tags: tags,
	}
}

func (s *emrContainersStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	virtualClusterID := emrContainersPayloadString(payload, "virtualClusterId", "vc-000001")
	if virtualClusterID == "" {
		virtualClusterID = "vc-000001"
	}
	jobRunID := emrContainersPayloadString(payload, "id", "jr-000001")
	if jobRunID == "" {
		jobRunID = "jr-000001"
	}
	managedEndpointID := emrContainersPayloadString(payload, "managedEndpointId", "me-000001")
	if managedEndpointID == "" {
		managedEndpointID = emrContainersPayloadString(payload, "id", "me-000001")
	}
	jobTemplateID := emrContainersPayloadString(payload, "jobTemplateId", "jt-000001")
	if jobTemplateID == "" {
		jobTemplateID = emrContainersPayloadString(payload, "id", "jt-000001")
	}
	securityConfigID := emrContainersPayloadString(payload, "id", "sc-000001")
	if securityConfigID == "" {
		securityConfigID = emrContainersPayloadString(payload, "name", "sc-000001")
	}

	s.ensureVirtualClusterLocked(virtualClusterID)

	switch action {
	case "CreateVirtualCluster":
		name := emrContainersPayloadString(payload, "name", s.nextTokenLocked("stackyard-emr-on-eks", 6))
		id := s.nextTokenLocked("vc", 6)
		vc := &emrContainersVirtualCluster{
			ID:             id,
			Name:           name,
			ARN:            emrContainersVirtualClusterARN(id),
			State:          "RUNNING",
			ContainerInfo:  map[string]any{"eksInfo": map[string]any{"namespace": "default"}},
			SecurityConfig: "sc-000001",
			CreatedAt:      now,
			LastModifiedAt: now,
		}
		s.virtualClusters[id] = vc
		s.ensureTagSetLocked(vc.ARN)
		return map[string]any{"id": vc.ID, "name": vc.Name, "arn": vc.ARN}

	case "DeleteVirtualCluster":
		vc := s.ensureVirtualClusterLocked(virtualClusterID)
		vc.State = "TERMINATING"
		vc.LastModifiedAt = now
		return map[string]any{"id": vc.ID}

	case "DescribeVirtualCluster":
		vc := s.ensureVirtualClusterLocked(virtualClusterID)
		return map[string]any{"virtualCluster": s.virtualClusterPayload(vc)}

	case "ListVirtualClusters":
		items := make([]any, 0, len(s.virtualClusters))
		for _, id := range s.sortedKeysVirtualClustersLocked() {
			vc := s.virtualClusters[id]
			items = append(items, map[string]any{
				"id":                vc.ID,
				"name":              vc.Name,
				"arn":               vc.ARN,
				"state":             vc.State,
				"containerProvider": vc.ContainerInfo,
				"createdAt":         vc.CreatedAt,
			})
		}
		return map[string]any{"virtualClusters": items, "nextToken": ""}

	case "StartJobRun":
		vc := s.ensureVirtualClusterLocked(virtualClusterID)
		name := emrContainersPayloadString(payload, "name", s.nextTokenLocked("stackyard-job", 6))
		id := s.nextTokenLocked("jr", 6)
		jr := &emrContainersJobRun{
			ID:               id,
			Name:             name,
			ARN:              emrContainersJobRunARN(vc.ID, id),
			VirtualClusterID: vc.ID,
			ExecutionRoleARN: emrContainersPayloadString(payload, "executionRoleArn", "arn:aws:iam::123456789012:role/stackyard-emr-containers-execution-role"),
			ReleaseLabel:     emrContainersPayloadString(payload, "releaseLabel", "emr-7.0.0-latest"),
			State:            "PENDING",
			CreatedAt:        now,
		}
		s.jobRuns[id] = jr
		s.ensureTagSetLocked(jr.ARN)
		return map[string]any{"id": jr.ID, "name": jr.Name, "arn": jr.ARN, "virtualClusterId": jr.VirtualClusterID}

	case "CancelJobRun":
		jr := s.ensureJobRunLocked(virtualClusterID, jobRunID)
		jr.State = "CANCELLED"
		jr.FinishedAt = now
		return map[string]any{"id": jr.ID, "virtualClusterId": jr.VirtualClusterID}

	case "DescribeJobRun":
		jr := s.ensureJobRunLocked(virtualClusterID, jobRunID)
		return map[string]any{"jobRun": s.jobRunPayload(jr)}

	case "ListJobRuns":
		items := make([]any, 0, len(s.jobRuns))
		for _, id := range s.sortedKeysJobRunsLocked() {
			jr := s.jobRuns[id]
			if virtualClusterID != "" && jr.VirtualClusterID != virtualClusterID {
				continue
			}
			items = append(items, map[string]any{
				"id":               jr.ID,
				"name":             jr.Name,
				"arn":              jr.ARN,
				"virtualClusterId": jr.VirtualClusterID,
				"state":            jr.State,
				"createdAt":        jr.CreatedAt,
			})
		}
		return map[string]any{"jobRuns": items, "nextToken": ""}

	case "CreateManagedEndpoint":
		vc := s.ensureVirtualClusterLocked(virtualClusterID)
		name := emrContainersPayloadString(payload, "name", s.nextTokenLocked("stackyard-endpoint", 6))
		id := s.nextTokenLocked("me", 6)
		ep := &emrContainersManagedEndpoint{
			ID:               id,
			Name:             name,
			ARN:              emrContainersManagedEndpointARN(vc.ID, id),
			VirtualClusterID: vc.ID,
			ExecutionRoleARN: emrContainersPayloadString(payload, "executionRoleArn", "arn:aws:iam::123456789012:role/stackyard-emr-containers-execution-role"),
			ReleaseLabel:     emrContainersPayloadString(payload, "releaseLabel", "emr-7.0.0-latest"),
			State:            "ACTIVE",
			CreatedAt:        now,
		}
		s.managedEndpoints[id] = ep
		s.ensureTagSetLocked(ep.ARN)
		return map[string]any{"id": ep.ID, "name": ep.Name, "arn": ep.ARN, "virtualClusterId": ep.VirtualClusterID}

	case "DeleteManagedEndpoint":
		ep := s.ensureManagedEndpointLocked(virtualClusterID, managedEndpointID)
		ep.State = "TERMINATING"
		return map[string]any{"id": ep.ID, "virtualClusterId": ep.VirtualClusterID}

	case "DescribeManagedEndpoint":
		ep := s.ensureManagedEndpointLocked(virtualClusterID, managedEndpointID)
		return map[string]any{"endpoint": s.managedEndpointPayload(ep)}

	case "ListManagedEndpoints":
		items := make([]any, 0, len(s.managedEndpoints))
		for _, id := range s.sortedKeysManagedEndpointsLocked() {
			ep := s.managedEndpoints[id]
			if virtualClusterID != "" && ep.VirtualClusterID != virtualClusterID {
				continue
			}
			items = append(items, map[string]any{
				"id":               ep.ID,
				"name":             ep.Name,
				"arn":              ep.ARN,
				"virtualClusterId": ep.VirtualClusterID,
				"state":            ep.State,
				"createdAt":        ep.CreatedAt,
			})
		}
		return map[string]any{"endpoints": items, "nextToken": ""}

	case "CreateJobTemplate":
		name := emrContainersPayloadString(payload, "name", s.nextTokenLocked("stackyard-job-template", 6))
		id := s.nextTokenLocked("jt", 6)
		jt := &emrContainersJobTemplate{ID: id, Name: name, ARN: emrContainersJobTemplateARN(id), CreatedAt: now}
		s.jobTemplates[id] = jt
		s.ensureTagSetLocked(jt.ARN)
		return map[string]any{"id": jt.ID, "name": jt.Name, "arn": jt.ARN}

	case "DeleteJobTemplate":
		jt := s.ensureJobTemplateLocked(jobTemplateID)
		delete(s.jobTemplates, jt.ID)
		return map[string]any{"id": jt.ID}

	case "DescribeJobTemplate":
		jt := s.ensureJobTemplateLocked(jobTemplateID)
		return map[string]any{"jobTemplate": map[string]any{"id": jt.ID, "name": jt.Name, "arn": jt.ARN, "createdAt": jt.CreatedAt}}

	case "ListJobTemplates":
		items := make([]any, 0, len(s.jobTemplates))
		for _, id := range s.sortedKeysJobTemplatesLocked() {
			jt := s.jobTemplates[id]
			items = append(items, map[string]any{"id": jt.ID, "name": jt.Name, "arn": jt.ARN, "createdAt": jt.CreatedAt})
		}
		return map[string]any{"templates": items, "nextToken": ""}

	case "CreateSecurityConfiguration":
		id := emrContainersPayloadString(payload, "name", "")
		if id == "" {
			id = s.nextTokenLocked("sc", 6)
		}
		sc := &emrContainersSecurityConfiguration{
			ID:        id,
			Name:      emrContainersPayloadString(payload, "name", id),
			ARN:       emrContainersSecurityConfigurationARN(id),
			CreatedAt: now,
		}
		s.securityConfigurations[id] = sc
		s.ensureTagSetLocked(sc.ARN)
		return map[string]any{"id": sc.ID, "name": sc.Name, "arn": sc.ARN}

	case "DescribeSecurityConfiguration":
		sc := s.ensureSecurityConfigurationLocked(securityConfigID)
		return map[string]any{"securityConfiguration": map[string]any{"id": sc.ID, "name": sc.Name, "arn": sc.ARN, "createdAt": sc.CreatedAt}}

	case "ListSecurityConfigurations":
		items := make([]any, 0, len(s.securityConfigurations))
		for _, id := range s.sortedKeysSecurityConfigurationsLocked() {
			sc := s.securityConfigurations[id]
			items = append(items, map[string]any{"id": sc.ID, "name": sc.Name, "arn": sc.ARN, "createdAt": sc.CreatedAt})
		}
		return map[string]any{"securityConfigurations": items, "nextToken": ""}

	case "GetManagedEndpointSessionCredentials":
		ep := s.ensureManagedEndpointLocked(virtualClusterID, managedEndpointID)
		return map[string]any{
			"id": ep.ID,
			"credentials": map[string]any{
				"accessKeyId":     "ASIASTACKYARDEMRONEXAMPLE",
				"secretAccessKey": "stackyard-emr-containers-secret",
				"sessionToken":    "stackyard-emr-containers-session-token",
				"expiration":      time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
			},
		}

	case "TagResource":
		resourceARN := emrContainersPayloadString(payload, "resourceArn", emrContainersVirtualClusterARN(virtualClusterID))
		tagSet := s.ensureTagSetLocked(resourceARN)
		for k, v := range emrContainersPayloadTags(payload, "tags") {
			tagSet[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := emrContainersPayloadString(payload, "resourceArn", emrContainersVirtualClusterARN(virtualClusterID))
		tagSet := s.ensureTagSetLocked(resourceARN)
		for _, key := range emrContainersPayloadStringSlice(payload, "tagKeys") {
			delete(tagSet, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := emrContainersPayloadString(payload, "resourceArn", emrContainersVirtualClusterARN(virtualClusterID))
		return map[string]any{"tags": emrContainersCloneStringMap(s.ensureTagSetLocked(resourceARN))}
	}

	switch {
	case strings.HasPrefix(action, "List"):
		return map[string]any{"items": []any{}, "nextToken": ""}
	case strings.HasPrefix(action, "Describe"), strings.HasPrefix(action, "Get"):
		return map[string]any{"action": action, "status": "ACTIVE", "timestamp": now}
	case strings.HasPrefix(action, "Create"), strings.HasPrefix(action, "Start"), strings.HasPrefix(action, "Tag"):
		return map[string]any{"action": action, "status": "OK", "id": s.nextTokenLocked("req", 10)}
	case strings.HasPrefix(action, "Delete"), strings.HasPrefix(action, "Cancel"), strings.HasPrefix(action, "Untag"):
		return map[string]any{"action": action, "status": "OK"}
	}

	return map[string]any{}
}

func (s *emrContainersStore) ensureVirtualClusterLocked(id string) *emrContainersVirtualCluster {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "vc-000001"
	}
	if existing, ok := s.virtualClusters[id]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	vc := &emrContainersVirtualCluster{
		ID:             id,
		Name:           "stackyard-emr-on-eks",
		ARN:            emrContainersVirtualClusterARN(id),
		State:          "RUNNING",
		ContainerInfo:  map[string]any{"eksInfo": map[string]any{"namespace": "default"}},
		SecurityConfig: "sc-000001",
		CreatedAt:      now,
		LastModifiedAt: now,
	}
	s.virtualClusters[id] = vc
	s.ensureTagSetLocked(vc.ARN)
	return vc
}

func (s *emrContainersStore) ensureJobRunLocked(virtualClusterID, id string) *emrContainersJobRun {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "jr-000001"
	}
	if existing, ok := s.jobRuns[id]; ok {
		return existing
	}
	vc := s.ensureVirtualClusterLocked(virtualClusterID)
	now := time.Now().UTC().Format(time.RFC3339)
	jr := &emrContainersJobRun{
		ID:               id,
		Name:             "stackyard-job-run",
		ARN:              emrContainersJobRunARN(vc.ID, id),
		VirtualClusterID: vc.ID,
		ExecutionRoleARN: "arn:aws:iam::123456789012:role/stackyard-emr-containers-execution-role",
		ReleaseLabel:     "emr-7.0.0-latest",
		State:            "RUNNING",
		CreatedAt:        now,
	}
	s.jobRuns[id] = jr
	s.ensureTagSetLocked(jr.ARN)
	return jr
}

func (s *emrContainersStore) ensureManagedEndpointLocked(virtualClusterID, id string) *emrContainersManagedEndpoint {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "me-000001"
	}
	if existing, ok := s.managedEndpoints[id]; ok {
		return existing
	}
	vc := s.ensureVirtualClusterLocked(virtualClusterID)
	now := time.Now().UTC().Format(time.RFC3339)
	ep := &emrContainersManagedEndpoint{
		ID:               id,
		Name:             "stackyard-managed-endpoint",
		ARN:              emrContainersManagedEndpointARN(vc.ID, id),
		VirtualClusterID: vc.ID,
		ExecutionRoleARN: "arn:aws:iam::123456789012:role/stackyard-emr-containers-execution-role",
		ReleaseLabel:     "emr-7.0.0-latest",
		State:            "ACTIVE",
		CreatedAt:        now,
	}
	s.managedEndpoints[id] = ep
	s.ensureTagSetLocked(ep.ARN)
	return ep
}

func (s *emrContainersStore) ensureJobTemplateLocked(id string) *emrContainersJobTemplate {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "jt-000001"
	}
	if existing, ok := s.jobTemplates[id]; ok {
		return existing
	}
	jt := &emrContainersJobTemplate{
		ID:        id,
		Name:      "stackyard-job-template",
		ARN:       emrContainersJobTemplateARN(id),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.jobTemplates[id] = jt
	s.ensureTagSetLocked(jt.ARN)
	return jt
}

func (s *emrContainersStore) ensureSecurityConfigurationLocked(id string) *emrContainersSecurityConfiguration {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "sc-000001"
	}
	if existing, ok := s.securityConfigurations[id]; ok {
		return existing
	}
	sc := &emrContainersSecurityConfiguration{
		ID:        id,
		Name:      id,
		ARN:       emrContainersSecurityConfigurationARN(id),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.securityConfigurations[id] = sc
	s.ensureTagSetLocked(sc.ARN)
	return sc
}

func (s *emrContainersStore) ensureTagSetLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = emrContainersVirtualClusterARN("vc-000001")
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	s.tags[resourceARN] = map[string]string{
		"stackyard": "true",
		"service":   "emrcontainers",
	}
	return s.tags[resourceARN]
}

func (s *emrContainersStore) nextTokenLocked(prefix string, width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%0*d", prefix, width, id)
}

func (s *emrContainersStore) sortedKeysVirtualClustersLocked() []string {
	keys := make([]string, 0, len(s.virtualClusters))
	for key := range s.virtualClusters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *emrContainersStore) sortedKeysJobRunsLocked() []string {
	keys := make([]string, 0, len(s.jobRuns))
	for key := range s.jobRuns {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *emrContainersStore) sortedKeysManagedEndpointsLocked() []string {
	keys := make([]string, 0, len(s.managedEndpoints))
	for key := range s.managedEndpoints {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *emrContainersStore) sortedKeysJobTemplatesLocked() []string {
	keys := make([]string, 0, len(s.jobTemplates))
	for key := range s.jobTemplates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *emrContainersStore) sortedKeysSecurityConfigurationsLocked() []string {
	keys := make([]string, 0, len(s.securityConfigurations))
	for key := range s.securityConfigurations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *emrContainersStore) virtualClusterPayload(vc *emrContainersVirtualCluster) map[string]any {
	if vc == nil {
		vc = s.ensureVirtualClusterLocked("vc-000001")
	}
	return map[string]any{
		"id":                      vc.ID,
		"name":                    vc.Name,
		"arn":                     vc.ARN,
		"state":                   vc.State,
		"containerProvider":       vc.ContainerInfo,
		"securityConfigurationId": vc.SecurityConfig,
		"createdAt":               vc.CreatedAt,
		"lastModifiedAt":          vc.LastModifiedAt,
	}
}

func (s *emrContainersStore) jobRunPayload(jr *emrContainersJobRun) map[string]any {
	if jr == nil {
		jr = s.ensureJobRunLocked("vc-000001", "jr-000001")
	}
	payload := map[string]any{
		"id":               jr.ID,
		"name":             jr.Name,
		"arn":              jr.ARN,
		"virtualClusterId": jr.VirtualClusterID,
		"executionRoleArn": jr.ExecutionRoleARN,
		"releaseLabel":     jr.ReleaseLabel,
		"state":            jr.State,
		"createdAt":        jr.CreatedAt,
	}
	if strings.TrimSpace(jr.FinishedAt) != "" {
		payload["finishedAt"] = jr.FinishedAt
	}
	return payload
}

func (s *emrContainersStore) managedEndpointPayload(ep *emrContainersManagedEndpoint) map[string]any {
	if ep == nil {
		ep = s.ensureManagedEndpointLocked("vc-000001", "me-000001")
	}
	return map[string]any{
		"id":               ep.ID,
		"name":             ep.Name,
		"arn":              ep.ARN,
		"virtualClusterId": ep.VirtualClusterID,
		"executionRoleArn": ep.ExecutionRoleARN,
		"releaseLabel":     ep.ReleaseLabel,
		"state":            ep.State,
		"createdAt":        ep.CreatedAt,
	}
}

func emrContainersPayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func emrContainersPayloadString(payload map[string]any, key, fallback string) string {
	value, ok := emrContainersPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	out := strings.TrimSpace(fmt.Sprintf("%v", value))
	if out == "" {
		return fallback
	}
	return out
}

func emrContainersPayloadStringSlice(payload map[string]any, key string) []string {
	value, ok := emrContainersPayloadValue(payload, key)
	if !ok {
		return nil
	}

	switch raw := value.(type) {
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			s := strings.TrimSpace(item)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", value))
		if s == "" {
			return nil
		}
		return []string{s}
	}
}

func emrContainersPayloadTags(payload map[string]any, key string) map[string]string {
	value, ok := emrContainersPayloadValue(payload, key)
	if !ok {
		return nil
	}
	result := map[string]string{}

	switch raw := value.(type) {
	case map[string]any:
		for k, v := range raw {
			keyName := strings.TrimSpace(k)
			if keyName == "" {
				continue
			}
			result[keyName] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	case map[string]string:
		for k, v := range raw {
			keyName := strings.TrimSpace(k)
			if keyName == "" {
				continue
			}
			result[keyName] = strings.TrimSpace(v)
		}
	case []any:
		for _, item := range raw {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			keyName := strings.TrimSpace(fmt.Sprintf("%v", entry["Key"]))
			if keyName == "" {
				keyName = strings.TrimSpace(fmt.Sprintf("%v", entry["key"]))
			}
			if keyName == "" {
				continue
			}
			valueName := strings.TrimSpace(fmt.Sprintf("%v", entry["Value"]))
			if valueName == "" {
				valueName = strings.TrimSpace(fmt.Sprintf("%v", entry["value"]))
			}
			result[keyName] = valueName
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func emrContainersCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func emrContainersVirtualClusterARN(id string) string {
	if strings.TrimSpace(id) == "" {
		id = "vc-000001"
	}
	return "arn:aws:emr-containers:us-east-1:123456789012:/virtualclusters/" + id
}

func emrContainersJobRunARN(virtualClusterID, id string) string {
	if strings.TrimSpace(virtualClusterID) == "" {
		virtualClusterID = "vc-000001"
	}
	if strings.TrimSpace(id) == "" {
		id = "jr-000001"
	}
	return "arn:aws:emr-containers:us-east-1:123456789012:/virtualclusters/" + virtualClusterID + "/jobruns/" + id
}

func emrContainersManagedEndpointARN(virtualClusterID, id string) string {
	if strings.TrimSpace(virtualClusterID) == "" {
		virtualClusterID = "vc-000001"
	}
	if strings.TrimSpace(id) == "" {
		id = "me-000001"
	}
	return "arn:aws:emr-containers:us-east-1:123456789012:/virtualclusters/" + virtualClusterID + "/managedendpoints/" + id
}

func emrContainersJobTemplateARN(id string) string {
	if strings.TrimSpace(id) == "" {
		id = "jt-000001"
	}
	return "arn:aws:emr-containers:us-east-1:123456789012:/jobtemplates/" + id
}

func emrContainersSecurityConfigurationARN(id string) string {
	if strings.TrimSpace(id) == "" {
		id = "sc-000001"
	}
	return "arn:aws:emr-containers:us-east-1:123456789012:/securityconfigurations/" + id
}
