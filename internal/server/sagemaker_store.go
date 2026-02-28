package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type sagemakerStore struct {
	mu sync.Mutex

	nextID int64

	trainingJobs    map[string]*sagemakerTrainingJob
	models          map[string]*sagemakerModel
	endpointConfigs map[string]*sagemakerEndpointConfig
	endpoints       map[string]*sagemakerEndpoint
	notebooks       map[string]*sagemakerNotebookInstance
	tags            map[string]map[string]string
}

type sagemakerTrainingJob struct {
	Name              string
	ARN               string
	ModelArtifactsS3  string
	Status            string
	SecondaryStatus   string
	CreationTime      string
	LastModifiedTime  string
	TrainingEndTime   string
	FailureReason     string
	TrainingImage     string
	TrainingJobOutput string
}

type sagemakerModel struct {
	Name             string
	ARN              string
	ExecutionRoleArn string
	CreationTime     string
}

type sagemakerEndpointConfig struct {
	Name         string
	ARN          string
	CreationTime string
}

type sagemakerEndpoint struct {
	Name               string
	ARN                string
	EndpointConfigName string
	Status             string
	CreationTime       string
	LastModifiedTime   string
}

type sagemakerNotebookInstance struct {
	Name             string
	ARN              string
	InstanceType     string
	Status           string
	CreationTime     string
	LastModifiedTime string
}

func newSageMakerStore() *sagemakerStore {
	now := time.Now().UTC().Format(time.RFC3339)

	trainingName := "stackyard-training-job"
	trainingARN := sagemakerTrainingJobARN(trainingName)
	modelName := "stackyard-model"
	modelARN := sagemakerModelARN(modelName)
	endpointConfigName := "stackyard-endpoint-config"
	endpointConfigARN := sagemakerEndpointConfigARN(endpointConfigName)
	endpointName := "stackyard-endpoint"
	endpointARN := sagemakerEndpointARN(endpointName)
	notebookName := "stackyard-notebook"
	notebookARN := sagemakerNotebookARN(notebookName)

	return &sagemakerStore{
		nextID: 2,
		trainingJobs: map[string]*sagemakerTrainingJob{
			trainingName: {
				Name:              trainingName,
				ARN:               trainingARN,
				ModelArtifactsS3:  "s3://stackyard-sagemaker/model.tar.gz",
				Status:            "Completed",
				SecondaryStatus:   "Completed",
				CreationTime:      now,
				LastModifiedTime:  now,
				TrainingEndTime:   now,
				TrainingImage:     "123456789012.dkr.ecr.us-east-1.amazonaws.com/sagemaker-training:latest",
				TrainingJobOutput: "s3://stackyard-sagemaker/output/",
			},
		},
		models: map[string]*sagemakerModel{
			modelName: {
				Name:             modelName,
				ARN:              modelARN,
				ExecutionRoleArn: "arn:aws:iam::123456789012:role/stackyard-sagemaker",
				CreationTime:     now,
			},
		},
		endpointConfigs: map[string]*sagemakerEndpointConfig{
			endpointConfigName: {
				Name:         endpointConfigName,
				ARN:          endpointConfigARN,
				CreationTime: now,
			},
		},
		endpoints: map[string]*sagemakerEndpoint{
			endpointName: {
				Name:               endpointName,
				ARN:                endpointARN,
				EndpointConfigName: endpointConfigName,
				Status:             "InService",
				CreationTime:       now,
				LastModifiedTime:   now,
			},
		},
		notebooks: map[string]*sagemakerNotebookInstance{
			notebookName: {
				Name:             notebookName,
				ARN:              notebookARN,
				InstanceType:     "ml.t3.medium",
				Status:           "InService",
				CreationTime:     now,
				LastModifiedTime: now,
			},
		},
		tags: map[string]map[string]string{
			trainingARN:       {"stackyard": "true"},
			modelARN:          {"stackyard": "true"},
			endpointConfigARN: {"stackyard": "true"},
			endpointARN:       {"stackyard": "true"},
			notebookARN:       {"stackyard": "true"},
		},
	}
}

func (s *sagemakerStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateTrainingJob":
		name := sagemakerPayloadString(payload, "TrainingJobName", s.nextTokenLocked("training"))
		job := s.ensureTrainingJobLocked(name)
		job.Status = "InProgress"
		job.SecondaryStatus = "Training"
		job.LastModifiedTime = now
		job.TrainingEndTime = ""
		return map[string]any{"TrainingJobArn": job.ARN}

	case "DescribeTrainingJob":
		name := sagemakerPayloadString(payload, "TrainingJobName", "stackyard-training-job")
		job := s.ensureTrainingJobLocked(name)
		if job.Status == "InProgress" {
			job.Status = "Completed"
			job.SecondaryStatus = "Completed"
			job.TrainingEndTime = now
			job.LastModifiedTime = now
		}
		return map[string]any{
			"TrainingJobName":        job.Name,
			"TrainingJobArn":         job.ARN,
			"TrainingJobStatus":      job.Status,
			"SecondaryStatus":        job.SecondaryStatus,
			"CreationTime":           job.CreationTime,
			"LastModifiedTime":       job.LastModifiedTime,
			"TrainingEndTime":        job.TrainingEndTime,
			"ModelArtifacts":         map[string]any{"S3ModelArtifacts": job.ModelArtifactsS3},
			"AlgorithmSpecification": map[string]any{"TrainingImage": job.TrainingImage, "TrainingInputMode": "File"},
			"OutputDataConfig":       map[string]any{"S3OutputPath": job.TrainingJobOutput},
		}

	case "ListTrainingJobs":
		items := make([]any, 0, len(s.trainingJobs))
		for _, job := range s.sortedTrainingJobsLocked() {
			items = append(items, map[string]any{
				"TrainingJobName":   job.Name,
				"TrainingJobArn":    job.ARN,
				"TrainingJobStatus": job.Status,
				"CreationTime":      job.CreationTime,
				"LastModifiedTime":  job.LastModifiedTime,
			})
		}
		return map[string]any{"TrainingJobSummaries": items, "NextToken": ""}

	case "StopTrainingJob":
		name := sagemakerPayloadString(payload, "TrainingJobName", "stackyard-training-job")
		job := s.ensureTrainingJobLocked(name)
		job.Status = "Stopped"
		job.SecondaryStatus = "Stopped"
		job.LastModifiedTime = now
		job.TrainingEndTime = now
		return map[string]any{}

	case "CreateModel":
		name := sagemakerPayloadString(payload, "ModelName", s.nextTokenLocked("model"))
		model := s.ensureModelLocked(name)
		if role := sagemakerPayloadString(payload, "ExecutionRoleArn", ""); role != "" {
			model.ExecutionRoleArn = role
		}
		model.CreationTime = now
		s.ensureTagsLocked(model.ARN)
		return map[string]any{"ModelArn": model.ARN}

	case "DescribeModel":
		name := sagemakerPayloadString(payload, "ModelName", "stackyard-model")
		model := s.ensureModelLocked(name)
		return map[string]any{
			"ModelName":        model.Name,
			"ModelArn":         model.ARN,
			"ExecutionRoleArn": model.ExecutionRoleArn,
			"CreationTime":     model.CreationTime,
			"PrimaryContainer": map[string]any{
				"Image":             "123456789012.dkr.ecr.us-east-1.amazonaws.com/sagemaker-inference:latest",
				"ModelDataUrl":      "s3://stackyard-sagemaker/model.tar.gz",
				"Environment":       map[string]any{},
				"ContainerHostname": "stackyard",
			},
		}

	case "ListModels":
		items := make([]any, 0, len(s.models))
		for _, model := range s.sortedModelsLocked() {
			items = append(items, map[string]any{
				"ModelName":    model.Name,
				"ModelArn":     model.ARN,
				"CreationTime": model.CreationTime,
			})
		}
		return map[string]any{"Models": items, "NextToken": ""}

	case "DeleteModel":
		name := sagemakerPayloadString(payload, "ModelName", "stackyard-model")
		if model, ok := s.models[name]; ok {
			delete(s.tags, model.ARN)
		}
		delete(s.models, name)
		return map[string]any{}

	case "CreateEndpointConfig":
		name := sagemakerPayloadString(payload, "EndpointConfigName", s.nextTokenLocked("endpoint-config"))
		ec := s.ensureEndpointConfigLocked(name)
		ec.CreationTime = now
		s.ensureTagsLocked(ec.ARN)
		return map[string]any{"EndpointConfigArn": ec.ARN}

	case "DescribeEndpointConfig":
		name := sagemakerPayloadString(payload, "EndpointConfigName", "stackyard-endpoint-config")
		ec := s.ensureEndpointConfigLocked(name)
		modelName := sagemakerPayloadString(payload, "ModelName", "stackyard-model")
		return map[string]any{
			"EndpointConfigName": ec.Name,
			"EndpointConfigArn":  ec.ARN,
			"CreationTime":       ec.CreationTime,
			"ProductionVariants": []any{map[string]any{"VariantName": "AllTraffic", "ModelName": modelName, "InitialInstanceCount": 1, "InstanceType": "ml.t3.medium"}},
		}

	case "ListEndpointConfigs":
		items := make([]any, 0, len(s.endpointConfigs))
		for _, ec := range s.sortedEndpointConfigsLocked() {
			items = append(items, map[string]any{"EndpointConfigName": ec.Name, "EndpointConfigArn": ec.ARN, "CreationTime": ec.CreationTime})
		}
		return map[string]any{"EndpointConfigs": items, "NextToken": ""}

	case "DeleteEndpointConfig":
		name := sagemakerPayloadString(payload, "EndpointConfigName", "stackyard-endpoint-config")
		if ec, ok := s.endpointConfigs[name]; ok {
			delete(s.tags, ec.ARN)
		}
		delete(s.endpointConfigs, name)
		return map[string]any{}

	case "CreateEndpoint":
		name := sagemakerPayloadString(payload, "EndpointName", s.nextTokenLocked("endpoint"))
		endpoint := s.ensureEndpointLocked(name)
		endpoint.EndpointConfigName = sagemakerPayloadString(payload, "EndpointConfigName", "stackyard-endpoint-config")
		endpoint.Status = "Creating"
		endpoint.LastModifiedTime = now
		s.ensureTagsLocked(endpoint.ARN)
		return map[string]any{"EndpointArn": endpoint.ARN}

	case "DescribeEndpoint":
		name := sagemakerPayloadString(payload, "EndpointName", "stackyard-endpoint")
		endpoint := s.ensureEndpointLocked(name)
		if endpoint.Status == "Creating" || endpoint.Status == "Updating" {
			endpoint.Status = "InService"
			endpoint.LastModifiedTime = now
		}
		return map[string]any{
			"EndpointName":       endpoint.Name,
			"EndpointArn":        endpoint.ARN,
			"EndpointStatus":     endpoint.Status,
			"EndpointConfigName": endpoint.EndpointConfigName,
			"CreationTime":       endpoint.CreationTime,
			"LastModifiedTime":   endpoint.LastModifiedTime,
		}

	case "ListEndpoints":
		items := make([]any, 0, len(s.endpoints))
		for _, endpoint := range s.sortedEndpointsLocked() {
			items = append(items, map[string]any{
				"EndpointName":       endpoint.Name,
				"EndpointArn":        endpoint.ARN,
				"EndpointStatus":     endpoint.Status,
				"EndpointConfigName": endpoint.EndpointConfigName,
				"CreationTime":       endpoint.CreationTime,
				"LastModifiedTime":   endpoint.LastModifiedTime,
			})
		}
		return map[string]any{"Endpoints": items, "NextToken": ""}

	case "UpdateEndpoint":
		name := sagemakerPayloadString(payload, "EndpointName", "stackyard-endpoint")
		endpoint := s.ensureEndpointLocked(name)
		endpoint.EndpointConfigName = sagemakerPayloadString(payload, "EndpointConfigName", endpoint.EndpointConfigName)
		endpoint.Status = "Updating"
		endpoint.LastModifiedTime = now
		return map[string]any{"EndpointArn": endpoint.ARN}

	case "DeleteEndpoint":
		name := sagemakerPayloadString(payload, "EndpointName", "stackyard-endpoint")
		if endpoint, ok := s.endpoints[name]; ok {
			delete(s.tags, endpoint.ARN)
		}
		delete(s.endpoints, name)
		return map[string]any{}

	case "CreateNotebookInstance":
		name := sagemakerPayloadString(payload, "NotebookInstanceName", s.nextTokenLocked("notebook"))
		nb := s.ensureNotebookLocked(name)
		nb.InstanceType = sagemakerPayloadString(payload, "InstanceType", "ml.t3.medium")
		nb.Status = "Pending"
		nb.LastModifiedTime = now
		s.ensureTagsLocked(nb.ARN)
		return map[string]any{}

	case "DescribeNotebookInstance":
		name := sagemakerPayloadString(payload, "NotebookInstanceName", "stackyard-notebook")
		nb := s.ensureNotebookLocked(name)
		if nb.Status == "Pending" || nb.Status == "Stopping" || nb.Status == "Starting" {
			nb.Status = "InService"
			nb.LastModifiedTime = now
		}
		return map[string]any{
			"NotebookInstanceName":   nb.Name,
			"NotebookInstanceArn":    nb.ARN,
			"NotebookInstanceStatus": nb.Status,
			"InstanceType":           nb.InstanceType,
			"CreationTime":           nb.CreationTime,
			"LastModifiedTime":       nb.LastModifiedTime,
		}

	case "ListNotebookInstances":
		items := make([]any, 0, len(s.notebooks))
		for _, nb := range s.sortedNotebooksLocked() {
			items = append(items, map[string]any{
				"NotebookInstanceName":   nb.Name,
				"NotebookInstanceArn":    nb.ARN,
				"NotebookInstanceStatus": nb.Status,
				"CreationTime":           nb.CreationTime,
				"LastModifiedTime":       nb.LastModifiedTime,
				"InstanceType":           nb.InstanceType,
			})
		}
		return map[string]any{"NotebookInstances": items, "NextToken": ""}

	case "StartNotebookInstance":
		name := sagemakerPayloadString(payload, "NotebookInstanceName", "stackyard-notebook")
		nb := s.ensureNotebookLocked(name)
		nb.Status = "Starting"
		nb.LastModifiedTime = now
		return map[string]any{}

	case "StopNotebookInstance":
		name := sagemakerPayloadString(payload, "NotebookInstanceName", "stackyard-notebook")
		nb := s.ensureNotebookLocked(name)
		nb.Status = "Stopping"
		nb.LastModifiedTime = now
		return map[string]any{}

	case "DeleteNotebookInstance":
		name := sagemakerPayloadString(payload, "NotebookInstanceName", "stackyard-notebook")
		if nb, ok := s.notebooks[name]; ok {
			delete(s.tags, nb.ARN)
		}
		delete(s.notebooks, name)
		return map[string]any{}

	case "UpdateNotebookInstance":
		name := sagemakerPayloadString(payload, "NotebookInstanceName", "stackyard-notebook")
		nb := s.ensureNotebookLocked(name)
		if instanceType := sagemakerPayloadString(payload, "InstanceType", ""); instanceType != "" {
			nb.InstanceType = instanceType
		}
		nb.LastModifiedTime = now
		return map[string]any{}

	case "AddTags", "TagResource":
		resourceARN := sagemakerPayloadString(payload, "ResourceArn", sagemakerEndpointARN("stackyard-endpoint"))
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range sagemakerPayloadTags(payload, "Tags") {
			tags[key] = value
		}
		return map[string]any{"Tags": s.tagsList(tags)}

	case "DeleteTags", "UntagResource":
		resourceARN := sagemakerPayloadString(payload, "ResourceArn", sagemakerEndpointARN("stackyard-endpoint"))
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range sagemakerPayloadStringSlice(payload, "TagKeys") {
			delete(tags, key)
		}
		return map[string]any{"Tags": s.tagsList(tags)}

	case "ListTags", "ListTagsForResource":
		resourceARN := sagemakerPayloadString(payload, "ResourceArn", sagemakerEndpointARN("stackyard-endpoint"))
		tags := s.ensureTagsLocked(resourceARN)
		return map[string]any{"Tags": s.tagsList(tags), "NextToken": ""}
	}

	return map[string]any{}
}

func (s *sagemakerStore) ensureTrainingJobLocked(name string) *sagemakerTrainingJob {
	if name = strings.TrimSpace(name); name == "" {
		name = "stackyard-training-job"
	}
	if job, ok := s.trainingJobs[name]; ok {
		return job
	}
	now := time.Now().UTC().Format(time.RFC3339)
	job := &sagemakerTrainingJob{
		Name:              name,
		ARN:               sagemakerTrainingJobARN(name),
		ModelArtifactsS3:  "s3://stackyard-sagemaker/model.tar.gz",
		Status:            "Completed",
		SecondaryStatus:   "Completed",
		CreationTime:      now,
		LastModifiedTime:  now,
		TrainingEndTime:   now,
		TrainingImage:     "123456789012.dkr.ecr.us-east-1.amazonaws.com/sagemaker-training:latest",
		TrainingJobOutput: "s3://stackyard-sagemaker/output/",
	}
	s.trainingJobs[name] = job
	s.ensureTagsLocked(job.ARN)
	return job
}

func (s *sagemakerStore) ensureModelLocked(name string) *sagemakerModel {
	if name = strings.TrimSpace(name); name == "" {
		name = "stackyard-model"
	}
	if model, ok := s.models[name]; ok {
		return model
	}
	now := time.Now().UTC().Format(time.RFC3339)
	model := &sagemakerModel{
		Name:             name,
		ARN:              sagemakerModelARN(name),
		ExecutionRoleArn: "arn:aws:iam::123456789012:role/stackyard-sagemaker",
		CreationTime:     now,
	}
	s.models[name] = model
	s.ensureTagsLocked(model.ARN)
	return model
}

func (s *sagemakerStore) ensureEndpointConfigLocked(name string) *sagemakerEndpointConfig {
	if name = strings.TrimSpace(name); name == "" {
		name = "stackyard-endpoint-config"
	}
	if ec, ok := s.endpointConfigs[name]; ok {
		return ec
	}
	now := time.Now().UTC().Format(time.RFC3339)
	ec := &sagemakerEndpointConfig{Name: name, ARN: sagemakerEndpointConfigARN(name), CreationTime: now}
	s.endpointConfigs[name] = ec
	s.ensureTagsLocked(ec.ARN)
	return ec
}

func (s *sagemakerStore) ensureEndpointLocked(name string) *sagemakerEndpoint {
	if name = strings.TrimSpace(name); name == "" {
		name = "stackyard-endpoint"
	}
	if endpoint, ok := s.endpoints[name]; ok {
		return endpoint
	}
	now := time.Now().UTC().Format(time.RFC3339)
	endpoint := &sagemakerEndpoint{
		Name:               name,
		ARN:                sagemakerEndpointARN(name),
		EndpointConfigName: "stackyard-endpoint-config",
		Status:             "InService",
		CreationTime:       now,
		LastModifiedTime:   now,
	}
	s.endpoints[name] = endpoint
	s.ensureTagsLocked(endpoint.ARN)
	return endpoint
}

func (s *sagemakerStore) ensureNotebookLocked(name string) *sagemakerNotebookInstance {
	if name = strings.TrimSpace(name); name == "" {
		name = "stackyard-notebook"
	}
	if nb, ok := s.notebooks[name]; ok {
		return nb
	}
	now := time.Now().UTC().Format(time.RFC3339)
	nb := &sagemakerNotebookInstance{
		Name:             name,
		ARN:              sagemakerNotebookARN(name),
		InstanceType:     "ml.t3.medium",
		Status:           "InService",
		CreationTime:     now,
		LastModifiedTime: now,
	}
	s.notebooks[name] = nb
	s.ensureTagsLocked(nb.ARN)
	return nb
}

func (s *sagemakerStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = sagemakerEndpointARN("stackyard-endpoint")
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{"stackyard": "true"}
	s.tags[resourceARN] = tags
	return tags
}

func (s *sagemakerStore) tagsList(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func (s *sagemakerStore) sortedTrainingJobsLocked() []*sagemakerTrainingJob {
	keys := make([]string, 0, len(s.trainingJobs))
	for key := range s.trainingJobs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*sagemakerTrainingJob, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.trainingJobs[key])
	}
	return out
}

func (s *sagemakerStore) sortedModelsLocked() []*sagemakerModel {
	keys := make([]string, 0, len(s.models))
	for key := range s.models {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*sagemakerModel, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.models[key])
	}
	return out
}

func (s *sagemakerStore) sortedEndpointConfigsLocked() []*sagemakerEndpointConfig {
	keys := make([]string, 0, len(s.endpointConfigs))
	for key := range s.endpointConfigs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*sagemakerEndpointConfig, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.endpointConfigs[key])
	}
	return out
}

func (s *sagemakerStore) sortedEndpointsLocked() []*sagemakerEndpoint {
	keys := make([]string, 0, len(s.endpoints))
	for key := range s.endpoints {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*sagemakerEndpoint, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.endpoints[key])
	}
	return out
}

func (s *sagemakerStore) sortedNotebooksLocked() []*sagemakerNotebookInstance {
	keys := make([]string, 0, len(s.notebooks))
	for key := range s.notebooks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*sagemakerNotebookInstance, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.notebooks[key])
	}
	return out
}

func (s *sagemakerStore) nextTokenLocked(prefix string) string {
	if strings.TrimSpace(prefix) == "" {
		prefix = "id"
	}
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%06d", prefix, id)
}

func sagemakerPayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	value, ok := raw.(string)
	if !ok {
		return def
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func sagemakerPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value, ok := item.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func sagemakerPayloadTags(payload map[string]any, key string) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return out
	}

	switch typed := raw.(type) {
	case map[string]any:
		for k, v := range typed {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if str, ok := v.(string); ok {
				out[k] = strings.TrimSpace(str)
			}
		}
	case []any:
		for _, item := range typed {
			tag, ok := item.(map[string]any)
			if !ok {
				continue
			}
			k := strings.TrimSpace(sagemakerPayloadString(tag, "Key", ""))
			if k == "" {
				k = strings.TrimSpace(sagemakerPayloadString(tag, "key", ""))
			}
			if k == "" {
				continue
			}
			v := sagemakerPayloadString(tag, "Value", "")
			if v == "" {
				v = sagemakerPayloadString(tag, "value", "")
			}
			out[k] = v
		}
	}

	return out
}

func sagemakerTrainingJobARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:us-east-1:123456789012:training-job/%s", strings.TrimSpace(name))
}

func sagemakerModelARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:us-east-1:123456789012:model/%s", strings.TrimSpace(name))
}

func sagemakerEndpointConfigARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:us-east-1:123456789012:endpoint-config/%s", strings.TrimSpace(name))
}

func sagemakerEndpointARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:us-east-1:123456789012:endpoint/%s", strings.TrimSpace(name))
}

func sagemakerNotebookARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:us-east-1:123456789012:notebook-instance/%s", strings.TrimSpace(name))
}
