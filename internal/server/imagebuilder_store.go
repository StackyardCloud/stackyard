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
	imageBuilderRegion    = "us-east-1"
	imageBuilderAccountID = "123456789012"
)

type imageBuilderStore struct {
	mu sync.Mutex

	nextID int64

	components                 map[string]map[string]any
	containerRecipes           map[string]map[string]any
	distributionConfigurations map[string]map[string]any
	images                     map[string]map[string]any
	imagePipelines             map[string]map[string]any
	imageRecipes               map[string]map[string]any
	infrastructureConfigs      map[string]map[string]any
	lifecyclePolicies          map[string]map[string]any
	workflows                  map[string]map[string]any

	lifecycleExecutions    map[string]map[string]any
	workflowExecutions     map[string]map[string]any
	workflowStepExecutions map[string]map[string]any
	resourceStates         map[string]string

	policies map[string]string
	tags     map[string]map[string]string
}

func newImageBuilderStore() *imageBuilderStore {
	s := &imageBuilderStore{
		nextID: 2,

		components:                 map[string]map[string]any{},
		containerRecipes:           map[string]map[string]any{},
		distributionConfigurations: map[string]map[string]any{},
		images:                     map[string]map[string]any{},
		imagePipelines:             map[string]map[string]any{},
		imageRecipes:               map[string]map[string]any{},
		infrastructureConfigs:      map[string]map[string]any{},
		lifecyclePolicies:          map[string]map[string]any{},
		workflows:                  map[string]map[string]any{},

		lifecycleExecutions:    map[string]map[string]any{},
		workflowExecutions:     map[string]map[string]any{},
		workflowStepExecutions: map[string]map[string]any{},
		resourceStates:         map[string]string{},

		policies: map[string]string{},
		tags:     map[string]map[string]string{},
	}
	s.ensureSeedDataLocked()
	return s
}

func (s *imageBuilderStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSeedDataLocked()
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	reqID := s.nextTokenLocked("req")

	syncPayload := imageBuilderCloneMap(payload)
	for k, v := range pathParams {
		syncPayload[k] = v
	}
	for k, values := range query {
		if len(values) == 0 {
			continue
		}
		syncPayload[k] = values[len(values)-1]
	}

	defaultComponentARN := s.firstARNLocked(s.components)
	defaultContainerRecipeARN := s.firstARNLocked(s.containerRecipes)
	defaultDistributionARN := s.firstARNLocked(s.distributionConfigurations)
	defaultImageARN := s.firstARNLocked(s.images)
	defaultImagePipelineARN := s.firstARNLocked(s.imagePipelines)
	defaultImageRecipeARN := s.firstARNLocked(s.imageRecipes)
	defaultInfraARN := s.firstARNLocked(s.infrastructureConfigs)
	defaultLifecyclePolicyARN := s.firstARNLocked(s.lifecyclePolicies)
	defaultWorkflowARN := s.firstARNLocked(s.workflows)

	switch action {
	case "CreateComponent", "ImportComponent":
		name := imageBuilderString(syncPayload, []string{"name", "componentName", "clientToken"}, s.nextTokenLocked("component"))
		component := s.ensureNamedResourceLocked(s.components, "componentBuildVersionArn", "component", name, now)
		component["status"] = "AVAILABLE"
		component["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "componentBuildVersionArn": component["componentBuildVersionArn"]}

	case "GetComponent":
		arn := imageBuilderString(syncPayload, []string{"componentBuildVersionArn", "componentArn"}, defaultComponentARN)
		component := s.ensureResourceByARNLocked(s.components, "componentBuildVersionArn", "component", arn, now)
		return map[string]any{"requestId": reqID, "component": imageBuilderCloneMap(component)}

	case "DeleteComponent":
		arn := imageBuilderString(syncPayload, []string{"componentBuildVersionArn", "componentArn"}, defaultComponentARN)
		s.deleteResourceByARNLocked(s.components, arn)
		delete(s.tags, arn)
		delete(s.policies, "component:"+arn)
		return map[string]any{"requestId": reqID}

	case "PutComponentPolicy":
		arn := imageBuilderString(syncPayload, []string{"componentArn", "componentBuildVersionArn"}, defaultComponentARN)
		s.policies["component:"+arn] = imageBuilderString(syncPayload, []string{"policy", "policyDocument"}, "{}")
		return map[string]any{"requestId": reqID}

	case "GetComponentPolicy":
		arn := imageBuilderString(syncPayload, []string{"componentArn", "componentBuildVersionArn"}, defaultComponentARN)
		return map[string]any{"requestId": reqID, "policy": s.policyLocked("component:" + arn)}

	case "ListComponents":
		return map[string]any{"requestId": reqID, "componentVersionList": s.listResourcesLocked(s.components), "nextToken": ""}

	case "ListComponentBuildVersions":
		return map[string]any{"requestId": reqID, "componentSummaryList": s.listResourcesLocked(s.components), "nextToken": ""}

	case "CreateContainerRecipe":
		name := imageBuilderString(syncPayload, []string{"name", "containerRecipeName", "clientToken"}, s.nextTokenLocked("container-recipe"))
		recipe := s.ensureNamedResourceLocked(s.containerRecipes, "containerRecipeArn", "container-recipe", name, now)
		recipe["status"] = "AVAILABLE"
		recipe["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "containerRecipeArn": recipe["containerRecipeArn"]}

	case "GetContainerRecipe":
		arn := imageBuilderString(syncPayload, []string{"containerRecipeArn"}, defaultContainerRecipeARN)
		recipe := s.ensureResourceByARNLocked(s.containerRecipes, "containerRecipeArn", "container-recipe", arn, now)
		return map[string]any{"requestId": reqID, "containerRecipe": imageBuilderCloneMap(recipe)}

	case "DeleteContainerRecipe":
		arn := imageBuilderString(syncPayload, []string{"containerRecipeArn"}, defaultContainerRecipeARN)
		s.deleteResourceByARNLocked(s.containerRecipes, arn)
		delete(s.tags, arn)
		delete(s.policies, "containerRecipe:"+arn)
		return map[string]any{"requestId": reqID}

	case "PutContainerRecipePolicy":
		arn := imageBuilderString(syncPayload, []string{"containerRecipeArn"}, defaultContainerRecipeARN)
		s.policies["containerRecipe:"+arn] = imageBuilderString(syncPayload, []string{"policy", "policyDocument"}, "{}")
		return map[string]any{"requestId": reqID}

	case "GetContainerRecipePolicy":
		arn := imageBuilderString(syncPayload, []string{"containerRecipeArn"}, defaultContainerRecipeARN)
		return map[string]any{"requestId": reqID, "policy": s.policyLocked("containerRecipe:" + arn)}

	case "ListContainerRecipes":
		return map[string]any{"requestId": reqID, "containerRecipeSummaryList": s.listResourcesLocked(s.containerRecipes), "nextToken": ""}

	case "CreateDistributionConfiguration":
		name := imageBuilderString(syncPayload, []string{"name", "distributionConfigurationName", "clientToken"}, s.nextTokenLocked("distribution"))
		cfg := s.ensureNamedResourceLocked(s.distributionConfigurations, "distributionConfigurationArn", "distribution-configuration", name, now)
		cfg["status"] = "AVAILABLE"
		cfg["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "distributionConfigurationArn": cfg["distributionConfigurationArn"]}

	case "GetDistributionConfiguration":
		arn := imageBuilderString(syncPayload, []string{"distributionConfigurationArn"}, defaultDistributionARN)
		cfg := s.ensureResourceByARNLocked(s.distributionConfigurations, "distributionConfigurationArn", "distribution-configuration", arn, now)
		return map[string]any{"requestId": reqID, "distributionConfiguration": imageBuilderCloneMap(cfg)}

	case "UpdateDistributionConfiguration":
		arn := imageBuilderString(syncPayload, []string{"distributionConfigurationArn"}, defaultDistributionARN)
		cfg := s.ensureResourceByARNLocked(s.distributionConfigurations, "distributionConfigurationArn", "distribution-configuration", arn, now)
		cfg["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "distributionConfigurationArn": cfg["distributionConfigurationArn"]}

	case "DeleteDistributionConfiguration":
		arn := imageBuilderString(syncPayload, []string{"distributionConfigurationArn"}, defaultDistributionARN)
		s.deleteResourceByARNLocked(s.distributionConfigurations, arn)
		delete(s.tags, arn)
		return map[string]any{"requestId": reqID}

	case "ListDistributionConfigurations":
		return map[string]any{"requestId": reqID, "distributionConfigurationSummaryList": s.listResourcesLocked(s.distributionConfigurations), "nextToken": ""}

	case "CreateImageRecipe":
		name := imageBuilderString(syncPayload, []string{"name", "imageRecipeName", "clientToken"}, s.nextTokenLocked("image-recipe"))
		recipe := s.ensureNamedResourceLocked(s.imageRecipes, "imageRecipeArn", "image-recipe", name, now)
		recipe["status"] = "AVAILABLE"
		recipe["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "imageRecipeArn": recipe["imageRecipeArn"]}

	case "GetImageRecipe":
		arn := imageBuilderString(syncPayload, []string{"imageRecipeArn"}, defaultImageRecipeARN)
		recipe := s.ensureResourceByARNLocked(s.imageRecipes, "imageRecipeArn", "image-recipe", arn, now)
		return map[string]any{"requestId": reqID, "imageRecipe": imageBuilderCloneMap(recipe)}

	case "DeleteImageRecipe":
		arn := imageBuilderString(syncPayload, []string{"imageRecipeArn"}, defaultImageRecipeARN)
		s.deleteResourceByARNLocked(s.imageRecipes, arn)
		delete(s.tags, arn)
		delete(s.policies, "imageRecipe:"+arn)
		return map[string]any{"requestId": reqID}

	case "PutImageRecipePolicy":
		arn := imageBuilderString(syncPayload, []string{"imageRecipeArn"}, defaultImageRecipeARN)
		s.policies["imageRecipe:"+arn] = imageBuilderString(syncPayload, []string{"policy", "policyDocument"}, "{}")
		return map[string]any{"requestId": reqID}

	case "GetImageRecipePolicy":
		arn := imageBuilderString(syncPayload, []string{"imageRecipeArn"}, defaultImageRecipeARN)
		return map[string]any{"requestId": reqID, "policy": s.policyLocked("imageRecipe:" + arn)}

	case "ListImageRecipes":
		return map[string]any{"requestId": reqID, "imageRecipeSummaryList": s.listResourcesLocked(s.imageRecipes), "nextToken": ""}

	case "CreateInfrastructureConfiguration":
		name := imageBuilderString(syncPayload, []string{"name", "infrastructureConfigurationName", "clientToken"}, s.nextTokenLocked("infrastructure"))
		cfg := s.ensureNamedResourceLocked(s.infrastructureConfigs, "infrastructureConfigurationArn", "infrastructure-configuration", name, now)
		cfg["status"] = "AVAILABLE"
		cfg["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "infrastructureConfigurationArn": cfg["infrastructureConfigurationArn"]}

	case "GetInfrastructureConfiguration":
		arn := imageBuilderString(syncPayload, []string{"infrastructureConfigurationArn"}, defaultInfraARN)
		cfg := s.ensureResourceByARNLocked(s.infrastructureConfigs, "infrastructureConfigurationArn", "infrastructure-configuration", arn, now)
		return map[string]any{"requestId": reqID, "infrastructureConfiguration": imageBuilderCloneMap(cfg)}

	case "UpdateInfrastructureConfiguration":
		arn := imageBuilderString(syncPayload, []string{"infrastructureConfigurationArn"}, defaultInfraARN)
		cfg := s.ensureResourceByARNLocked(s.infrastructureConfigs, "infrastructureConfigurationArn", "infrastructure-configuration", arn, now)
		cfg["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "infrastructureConfigurationArn": cfg["infrastructureConfigurationArn"]}

	case "DeleteInfrastructureConfiguration":
		arn := imageBuilderString(syncPayload, []string{"infrastructureConfigurationArn"}, defaultInfraARN)
		s.deleteResourceByARNLocked(s.infrastructureConfigs, arn)
		delete(s.tags, arn)
		return map[string]any{"requestId": reqID}

	case "ListInfrastructureConfigurations":
		return map[string]any{"requestId": reqID, "infrastructureConfigurationSummaryList": s.listResourcesLocked(s.infrastructureConfigs), "nextToken": ""}

	case "CreateImagePipeline":
		name := imageBuilderString(syncPayload, []string{"name", "imagePipelineName", "clientToken"}, s.nextTokenLocked("image-pipeline"))
		pipeline := s.ensureNamedResourceLocked(s.imagePipelines, "imagePipelineArn", "image-pipeline", name, now)
		pipeline["status"] = "AVAILABLE"
		pipeline["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "imagePipelineArn": pipeline["imagePipelineArn"]}

	case "GetImagePipeline":
		arn := imageBuilderString(syncPayload, []string{"imagePipelineArn"}, defaultImagePipelineARN)
		pipeline := s.ensureResourceByARNLocked(s.imagePipelines, "imagePipelineArn", "image-pipeline", arn, now)
		return map[string]any{"requestId": reqID, "imagePipeline": imageBuilderCloneMap(pipeline)}

	case "UpdateImagePipeline":
		arn := imageBuilderString(syncPayload, []string{"imagePipelineArn"}, defaultImagePipelineARN)
		pipeline := s.ensureResourceByARNLocked(s.imagePipelines, "imagePipelineArn", "image-pipeline", arn, now)
		pipeline["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "imagePipelineArn": pipeline["imagePipelineArn"]}

	case "DeleteImagePipeline":
		arn := imageBuilderString(syncPayload, []string{"imagePipelineArn"}, defaultImagePipelineARN)
		s.deleteResourceByARNLocked(s.imagePipelines, arn)
		delete(s.tags, arn)
		delete(s.policies, "image:"+arn)
		return map[string]any{"requestId": reqID}

	case "StartImagePipelineExecution":
		pipelineArn := imageBuilderString(syncPayload, []string{"imagePipelineArn"}, defaultImagePipelineARN)
		pipeline := s.ensureResourceByARNLocked(s.imagePipelines, "imagePipelineArn", "image-pipeline", pipelineArn, now)
		imageName := imageBuilderString(pipeline, []string{"name"}, s.nextTokenLocked("image"))
		image := s.ensureNamedResourceLocked(s.images, "imageBuildVersionArn", "image", imageName, now)
		image["status"] = "AVAILABLE"
		execID := s.nextTokenLocked("workflow-execution")
		s.workflowExecutions[execID] = map[string]any{
			"workflowExecutionId":  execID,
			"status":               "SUCCEEDED",
			"imageBuildVersionArn": image["imageBuildVersionArn"],
			"dateCreated":          nowStr,
		}
		return map[string]any{"requestId": reqID, "imageBuildVersionArn": image["imageBuildVersionArn"], "workflowExecutionId": execID}

	case "ListImagePipelines":
		return map[string]any{"requestId": reqID, "imagePipelineList": s.listResourcesLocked(s.imagePipelines), "nextToken": ""}

	case "CreateImage", "ImportDiskImage", "ImportVmImage":
		name := imageBuilderString(syncPayload, []string{"name", "imageName", "clientToken"}, s.nextTokenLocked("image"))
		image := s.ensureNamedResourceLocked(s.images, "imageBuildVersionArn", "image", name, now)
		image["status"] = "AVAILABLE"
		image["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "imageBuildVersionArn": image["imageBuildVersionArn"]}

	case "GetImage":
		arn := imageBuilderString(syncPayload, []string{"imageBuildVersionArn", "imageArn"}, defaultImageARN)
		image := s.ensureResourceByARNLocked(s.images, "imageBuildVersionArn", "image", arn, now)
		return map[string]any{"requestId": reqID, "image": imageBuilderCloneMap(image)}

	case "DeleteImage":
		arn := imageBuilderString(syncPayload, []string{"imageBuildVersionArn", "imageArn"}, defaultImageARN)
		s.deleteResourceByARNLocked(s.images, arn)
		delete(s.tags, arn)
		delete(s.policies, "image:"+arn)
		return map[string]any{"requestId": reqID}

	case "DistributeImage":
		arn := imageBuilderString(syncPayload, []string{"imageBuildVersionArn", "imageArn"}, defaultImageARN)
		image := s.ensureResourceByARNLocked(s.images, "imageBuildVersionArn", "image", arn, now)
		image["status"] = "DISTRIBUTED"
		image["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "imageBuildVersionArn": image["imageBuildVersionArn"]}

	case "RetryImage":
		arn := imageBuilderString(syncPayload, []string{"imageBuildVersionArn", "imageArn"}, defaultImageARN)
		image := s.ensureResourceByARNLocked(s.images, "imageBuildVersionArn", "image", arn, now)
		image["status"] = "AVAILABLE"
		image["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "imageBuildVersionArn": image["imageBuildVersionArn"]}

	case "CancelImageCreation":
		arn := imageBuilderString(syncPayload, []string{"imageBuildVersionArn", "imageArn"}, defaultImageARN)
		image := s.ensureResourceByARNLocked(s.images, "imageBuildVersionArn", "image", arn, now)
		image["status"] = "CANCELLED"
		image["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "imageBuildVersionArn": image["imageBuildVersionArn"]}

	case "ListImages", "ListImageBuildVersions", "ListImagePipelineImages":
		return map[string]any{"requestId": reqID, "imageVersionList": s.listResourcesLocked(s.images), "nextToken": ""}

	case "ListImagePackages":
		return map[string]any{
			"requestId": reqID,
			"imagePackageList": []any{
				map[string]any{"packageName": "bash", "packageVersion": "5.2", "packageManager": "OS"},
			},
			"nextToken": "",
		}

	case "ListImageScanFindingAggregations":
		return map[string]any{
			"requestId": reqID,
			"nextToken": "",
			"responses": []any{
				map[string]any{"accountId": imageBuilderAccountID, "severityCounts": map[string]any{"critical": 0, "high": 0, "medium": 1, "low": 1}},
			},
		}

	case "ListImageScanFindings":
		return map[string]any{
			"requestId": reqID,
			"nextToken": "",
			"findings": []any{
				map[string]any{"title": "Sample package vulnerability", "severity": "MEDIUM", "description": "stackyard sample finding"},
			},
		}

	case "CreateLifecyclePolicy":
		name := imageBuilderString(syncPayload, []string{"name", "lifecyclePolicyName", "clientToken"}, s.nextTokenLocked("lifecycle-policy"))
		policy := s.ensureNamedResourceLocked(s.lifecyclePolicies, "lifecyclePolicyArn", "lifecycle-policy", name, now)
		policy["status"] = "ENABLED"
		policy["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "lifecyclePolicyArn": policy["lifecyclePolicyArn"]}

	case "GetLifecyclePolicy":
		arn := imageBuilderString(syncPayload, []string{"lifecyclePolicyArn"}, defaultLifecyclePolicyARN)
		policy := s.ensureResourceByARNLocked(s.lifecyclePolicies, "lifecyclePolicyArn", "lifecycle-policy", arn, now)
		return map[string]any{"requestId": reqID, "lifecyclePolicy": imageBuilderCloneMap(policy)}

	case "UpdateLifecyclePolicy":
		arn := imageBuilderString(syncPayload, []string{"lifecyclePolicyArn"}, defaultLifecyclePolicyARN)
		policy := s.ensureResourceByARNLocked(s.lifecyclePolicies, "lifecyclePolicyArn", "lifecycle-policy", arn, now)
		policy["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "lifecyclePolicyArn": policy["lifecyclePolicyArn"]}

	case "DeleteLifecyclePolicy":
		arn := imageBuilderString(syncPayload, []string{"lifecyclePolicyArn"}, defaultLifecyclePolicyARN)
		s.deleteResourceByARNLocked(s.lifecyclePolicies, arn)
		delete(s.tags, arn)
		return map[string]any{"requestId": reqID}

	case "ListLifecyclePolicies":
		return map[string]any{"requestId": reqID, "lifecyclePolicySummaryList": s.listResourcesLocked(s.lifecyclePolicies), "nextToken": ""}

	case "GetLifecycleExecution":
		execID := imageBuilderString(syncPayload, []string{"lifecycleExecutionId"}, "lifecycle-execution-000001")
		exec := s.ensureLifecycleExecutionLocked(execID, now)
		return map[string]any{"requestId": reqID, "lifecycleExecution": imageBuilderCloneMap(exec)}

	case "ListLifecycleExecutions":
		return map[string]any{"requestId": reqID, "lifecycleExecutions": s.listMapValuesLocked(s.lifecycleExecutions, "lifecycleExecutionId"), "nextToken": ""}

	case "ListLifecycleExecutionResources":
		execID := imageBuilderString(syncPayload, []string{"lifecycleExecutionId"}, "lifecycle-execution-000001")
		s.ensureLifecycleExecutionLocked(execID, now)
		return map[string]any{
			"requestId":            reqID,
			"lifecycleExecutionId": execID,
			"resources": []any{
				map[string]any{"resourceArn": defaultImageARN, "state": "AVAILABLE", "action": "KEEP"},
			},
			"nextToken": "",
		}

	case "CancelLifecycleExecution":
		execID := imageBuilderString(syncPayload, []string{"lifecycleExecutionId"}, "lifecycle-execution-000001")
		exec := s.ensureLifecycleExecutionLocked(execID, now)
		exec["status"] = "CANCELLED"
		exec["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "lifecycleExecutionId": execID}

	case "CreateWorkflow":
		name := imageBuilderString(syncPayload, []string{"name", "workflowName", "clientToken"}, s.nextTokenLocked("workflow"))
		workflow := s.ensureNamedResourceLocked(s.workflows, "workflowBuildVersionArn", "workflow", name, now)
		workflow["status"] = "AVAILABLE"
		workflow["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "workflowBuildVersionArn": workflow["workflowBuildVersionArn"]}

	case "GetWorkflow":
		arn := imageBuilderString(syncPayload, []string{"workflowBuildVersionArn", "workflowArn"}, defaultWorkflowARN)
		workflow := s.ensureResourceByARNLocked(s.workflows, "workflowBuildVersionArn", "workflow", arn, now)
		return map[string]any{"requestId": reqID, "workflow": imageBuilderCloneMap(workflow)}

	case "DeleteWorkflow":
		arn := imageBuilderString(syncPayload, []string{"workflowBuildVersionArn", "workflowArn"}, defaultWorkflowARN)
		s.deleteResourceByARNLocked(s.workflows, arn)
		delete(s.tags, arn)
		return map[string]any{"requestId": reqID}

	case "ListWorkflows", "ListWorkflowBuildVersions":
		return map[string]any{"requestId": reqID, "workflowVersionList": s.listResourcesLocked(s.workflows), "nextToken": ""}

	case "GetWorkflowExecution":
		execID := imageBuilderString(syncPayload, []string{"workflowExecutionId"}, "workflow-execution-000001")
		exec := s.ensureWorkflowExecutionLocked(execID, now)
		return map[string]any{"requestId": reqID, "workflowExecution": imageBuilderCloneMap(exec)}

	case "ListWorkflowExecutions":
		return map[string]any{"requestId": reqID, "workflowExecutions": s.listMapValuesLocked(s.workflowExecutions, "workflowExecutionId"), "nextToken": ""}

	case "GetWorkflowStepExecution":
		stepID := imageBuilderString(syncPayload, []string{"stepExecutionId"}, "workflow-step-000001")
		step := s.ensureWorkflowStepExecutionLocked(stepID, now)
		return map[string]any{"requestId": reqID, "stepExecution": imageBuilderCloneMap(step)}

	case "ListWorkflowStepExecutions", "ListWaitingWorkflowSteps":
		return map[string]any{"requestId": reqID, "steps": s.listMapValuesLocked(s.workflowStepExecutions, "stepExecutionId"), "nextToken": ""}

	case "SendWorkflowStepAction":
		stepID := imageBuilderString(syncPayload, []string{"stepExecutionId"}, "workflow-step-000001")
		step := s.ensureWorkflowStepExecutionLocked(stepID, now)
		step["status"] = "COMPLETED"
		step["dateUpdated"] = nowStr
		return map[string]any{"requestId": reqID, "stepExecutionId": stepID, "status": "COMPLETED"}

	case "StartResourceStateUpdate":
		resourceARN := imageBuilderString(syncPayload, []string{"resourceArn", "ResourceArn"}, defaultImageARN)
		state := imageBuilderString(syncPayload, []string{"state", "resourceState"}, "DEPRECATED")
		s.resourceStates[resourceARN] = state
		return map[string]any{"requestId": reqID, "resourceArn": resourceARN, "state": state}

	case "GetMarketplaceResource":
		resourceARN := imageBuilderString(syncPayload, []string{"resourceArn", "ResourceArn"}, imageBuilderARN("marketplace-resource", "stackyard"))
		return map[string]any{"requestId": reqID, "resourceArn": resourceARN, "data": "stackyard-marketplace-resource"}

	case "PutImagePolicy":
		arn := imageBuilderString(syncPayload, []string{"imageArn", "imageBuildVersionArn"}, defaultImageARN)
		s.policies["image:"+arn] = imageBuilderString(syncPayload, []string{"policy", "policyDocument"}, "{}")
		return map[string]any{"requestId": reqID}

	case "GetImagePolicy":
		arn := imageBuilderString(syncPayload, []string{"imageArn", "imageBuildVersionArn"}, defaultImageARN)
		return map[string]any{"requestId": reqID, "policy": s.policyLocked("image:" + arn)}

	case "TagResource":
		resourceARN := imageBuilderString(syncPayload, []string{"resourceArn", "ResourceArn"}, defaultImageARN)
		tags := imageBuilderTagsFromAny(syncPayload["tags"])
		if len(tags) == 0 {
			tags = imageBuilderTagsFromAny(syncPayload["Tags"])
		}
		existing := s.ensureTagsLocked(resourceARN)
		for k, v := range tags {
			existing[k] = v
		}
		return map[string]any{"requestId": reqID}

	case "UntagResource":
		resourceARN := imageBuilderString(syncPayload, []string{"resourceArn", "ResourceArn"}, defaultImageARN)
		tagKeys := imageBuilderStringSlice(syncPayload["tagKeys"])
		if len(tagKeys) == 0 {
			tagKeys = imageBuilderStringSlice(syncPayload["TagKeys"])
		}
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range tagKeys {
			delete(tags, key)
		}
		return map[string]any{"requestId": reqID}

	case "ListTagsForResource":
		resourceARN := imageBuilderString(syncPayload, []string{"resourceArn", "ResourceArn"}, defaultImageARN)
		return map[string]any{"requestId": reqID, "tags": imageBuilderCloneStringMap(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{"requestId": reqID}
}

func (s *imageBuilderStore) ensureSeedDataLocked() {
	now := time.Now().UTC()
	component := s.ensureNamedResourceLocked(s.components, "componentBuildVersionArn", "component", "stackyard-component", now)
	containerRecipe := s.ensureNamedResourceLocked(s.containerRecipes, "containerRecipeArn", "container-recipe", "stackyard-container-recipe", now)
	distributionCfg := s.ensureNamedResourceLocked(s.distributionConfigurations, "distributionConfigurationArn", "distribution-configuration", "stackyard-distribution-config", now)
	imageRecipe := s.ensureNamedResourceLocked(s.imageRecipes, "imageRecipeArn", "image-recipe", "stackyard-image-recipe", now)
	infraCfg := s.ensureNamedResourceLocked(s.infrastructureConfigs, "infrastructureConfigurationArn", "infrastructure-configuration", "stackyard-infra-config", now)
	pipeline := s.ensureNamedResourceLocked(s.imagePipelines, "imagePipelineArn", "image-pipeline", "stackyard-image-pipeline", now)
	image := s.ensureNamedResourceLocked(s.images, "imageBuildVersionArn", "image", "stackyard-image", now)
	lifecyclePolicy := s.ensureNamedResourceLocked(s.lifecyclePolicies, "lifecyclePolicyArn", "lifecycle-policy", "stackyard-lifecycle-policy", now)
	workflow := s.ensureNamedResourceLocked(s.workflows, "workflowBuildVersionArn", "workflow", "stackyard-workflow", now)

	s.ensureLifecycleExecutionLocked("lifecycle-execution-000001", now)
	s.ensureWorkflowExecutionLocked("workflow-execution-000001", now)
	s.ensureWorkflowStepExecutionLocked("workflow-step-000001", now)

	s.ensureTagsLocked(imageBuilderString(component, []string{"componentBuildVersionArn"}, ""))["stackyard"] = "true"
	s.ensureTagsLocked(imageBuilderString(containerRecipe, []string{"containerRecipeArn"}, ""))["stackyard"] = "true"
	s.ensureTagsLocked(imageBuilderString(distributionCfg, []string{"distributionConfigurationArn"}, ""))["stackyard"] = "true"
	s.ensureTagsLocked(imageBuilderString(imageRecipe, []string{"imageRecipeArn"}, ""))["stackyard"] = "true"
	s.ensureTagsLocked(imageBuilderString(infraCfg, []string{"infrastructureConfigurationArn"}, ""))["stackyard"] = "true"
	s.ensureTagsLocked(imageBuilderString(pipeline, []string{"imagePipelineArn"}, ""))["stackyard"] = "true"
	s.ensureTagsLocked(imageBuilderString(image, []string{"imageBuildVersionArn"}, ""))["stackyard"] = "true"
	s.ensureTagsLocked(imageBuilderString(lifecyclePolicy, []string{"lifecyclePolicyArn"}, ""))["stackyard"] = "true"
	s.ensureTagsLocked(imageBuilderString(workflow, []string{"workflowBuildVersionArn"}, ""))["stackyard"] = "true"
}

func (s *imageBuilderStore) ensureNamedResourceLocked(target map[string]map[string]any, arnField, kind, name string, now time.Time) map[string]any {
	for arn, item := range target {
		if strings.EqualFold(imageBuilderString(item, []string{"name"}, ""), name) {
			item[arnField] = arn
			item["arn"] = arn
			if _, ok := item["dateCreated"]; !ok {
				item["dateCreated"] = now.Format(time.RFC3339)
			}
			item["dateUpdated"] = now.Format(time.RFC3339)
			return item
		}
	}
	arn := imageBuilderARN(kind, name)
	item := map[string]any{
		"name":        name,
		"arn":         arn,
		arnField:      arn,
		"owner":       "stackyard",
		"platform":    "Linux",
		"type":        kind,
		"status":      "AVAILABLE",
		"dateCreated": now.Format(time.RFC3339),
		"dateUpdated": now.Format(time.RFC3339),
	}
	target[arn] = item
	return item
}

func (s *imageBuilderStore) ensureResourceByARNLocked(target map[string]map[string]any, arnField, kind, arn string, now time.Time) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return s.ensureNamedResourceLocked(target, arnField, kind, s.nextTokenLocked(kind), now)
	}
	if item, ok := target[arn]; ok {
		item[arnField] = arn
		item["arn"] = arn
		if _, ok := item["dateCreated"]; !ok {
			item["dateCreated"] = now.Format(time.RFC3339)
		}
		item["dateUpdated"] = now.Format(time.RFC3339)
		return item
	}
	name := imageBuilderNameFromARN(arn)
	if name == "" {
		name = s.nextTokenLocked(kind)
	}
	item := map[string]any{
		"name":        name,
		"arn":         arn,
		arnField:      arn,
		"owner":       "stackyard",
		"platform":    "Linux",
		"type":        kind,
		"status":      "AVAILABLE",
		"dateCreated": now.Format(time.RFC3339),
		"dateUpdated": now.Format(time.RFC3339),
	}
	target[arn] = item
	return item
}

func (s *imageBuilderStore) ensureLifecycleExecutionLocked(id string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.nextTokenLocked("lifecycle-execution")
	}
	if exec, ok := s.lifecycleExecutions[id]; ok {
		return exec
	}
	exec := map[string]any{
		"lifecycleExecutionId": id,
		"status":               "SUCCEEDED",
		"dateCreated":          now.Format(time.RFC3339),
		"dateUpdated":          now.Format(time.RFC3339),
	}
	s.lifecycleExecutions[id] = exec
	return exec
}

func (s *imageBuilderStore) ensureWorkflowExecutionLocked(id string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.nextTokenLocked("workflow-execution")
	}
	if exec, ok := s.workflowExecutions[id]; ok {
		return exec
	}
	exec := map[string]any{
		"workflowExecutionId": id,
		"status":              "SUCCEEDED",
		"dateCreated":         now.Format(time.RFC3339),
	}
	s.workflowExecutions[id] = exec
	return exec
}

func (s *imageBuilderStore) ensureWorkflowStepExecutionLocked(id string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.nextTokenLocked("workflow-step")
	}
	if step, ok := s.workflowStepExecutions[id]; ok {
		return step
	}
	step := map[string]any{
		"stepExecutionId": id,
		"status":          "WAITING",
		"dateCreated":     now.Format(time.RFC3339),
		"dateUpdated":     now.Format(time.RFC3339),
	}
	s.workflowStepExecutions[id] = step
	return step
}

func (s *imageBuilderStore) deleteResourceByARNLocked(target map[string]map[string]any, arn string) {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return
	}
	delete(target, arn)
}

func (s *imageBuilderStore) firstARNLocked(target map[string]map[string]any) string {
	if len(target) == 0 {
		return ""
	}
	arns := make([]string, 0, len(target))
	for arn := range target {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	return arns[0]
}

func (s *imageBuilderStore) listResourcesLocked(target map[string]map[string]any) []any {
	arns := make([]string, 0, len(target))
	for arn := range target {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, imageBuilderCloneMap(target[arn]))
	}
	return out
}

func (s *imageBuilderStore) listMapValuesLocked(target map[string]map[string]any, key string) []any {
	ids := make([]string, 0, len(target))
	for id := range target {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		item := imageBuilderCloneMap(target[id])
		if _, ok := item[key]; !ok {
			item[key] = id
		}
		out = append(out, item)
	}
	return out
}

func (s *imageBuilderStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = imageBuilderARN("resource", "stackyard")
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func (s *imageBuilderStore) policyLocked(key string) string {
	if policy, ok := s.policies[key]; ok && strings.TrimSpace(policy) != "" {
		return policy
	}
	return "{}"
}

func (s *imageBuilderStore) nextTokenLocked(prefix string) string {
	token := fmt.Sprintf("%s-%06d", strings.Trim(prefix, " -_"), s.nextID)
	s.nextID++
	return token
}

func imageBuilderARN(kind, name string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "resource"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard"
	}
	return fmt.Sprintf("arn:aws:imagebuilder:%s:%s:%s/%s", imageBuilderRegion, imageBuilderAccountID, kind, name)
}

func imageBuilderNameFromARN(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ""
	}
	parts := strings.Split(arn, "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func imageBuilderString(payload map[string]any, keys []string, def string) string {
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if value, ok := payload[key]; ok {
			if s := imageBuilderStringAny(value); s != "" {
				return s
			}
		}
		for existingKey, raw := range payload {
			if strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
				if s := imageBuilderStringAny(raw); s != "" {
					return s
				}
			}
		}
	}
	return strings.TrimSpace(def)
}

func imageBuilderStringAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func imageBuilderTagsFromAny(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]any:
		for k, v := range typed {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = imageBuilderStringAny(v)
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
			key := imageBuilderString(asMap, []string{"key", "Key"}, "")
			if key == "" {
				continue
			}
			out[key] = imageBuilderString(asMap, []string{"value", "Value"}, "")
		}
	case []map[string]string:
		for _, item := range typed {
			key := strings.TrimSpace(item["Key"])
			if key == "" {
				key = strings.TrimSpace(item["key"])
			}
			if key == "" {
				continue
			}
			value := strings.TrimSpace(item["Value"])
			if value == "" {
				value = strings.TrimSpace(item["value"])
			}
			out[key] = value
		}
	}
	return out
}

func imageBuilderStringSlice(value any) []string {
	out := []string{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if s := imageBuilderStringAny(item); s != "" {
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

func imageBuilderCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func imageBuilderCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
