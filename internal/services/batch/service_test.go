package batch

import "testing"

func TestServiceLifecycle(t *testing.T) {
	svc := NewService()

	ce, err := svc.CreateComputeEnvironment("ce-1", "UNMANAGED", "ENABLED", 8, "", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create compute environment: %v", err)
	}
	if ce.ARN == "" {
		t.Fatalf("expected compute environment arn")
	}

	jq, err := svc.CreateJobQueue("jq-1", 1, "ENABLED", []ComputeEnvironmentOrder{{Order: 1, ComputeEnvironment: ce.Name}}, "", nil)
	if err != nil {
		t.Fatalf("create job queue: %v", err)
	}
	if jq.ARN == "" {
		t.Fatalf("expected job queue arn")
	}

	jd, err := svc.RegisterJobDefinition("jd-1", "container", map[string]string{"k": "v"}, map[string]string{"team": "platform"})
	if err != nil {
		t.Fatalf("register job definition: %v", err)
	}
	if jd.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", jd.Revision)
	}

	job, err := svc.SubmitJob("job-1", jq.Name, jd.Name, map[string]string{"attempt": "1"}, map[string]string{"env": "dev"})
	if err != nil {
		t.Fatalf("submit job: %v", err)
	}
	if job.ID == "" {
		t.Fatalf("expected job id")
	}

	jobs := svc.ListJobs(jq.Name, "")
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	desc := svc.DescribeJobs([]string{job.ID})
	if len(desc) != 1 {
		t.Fatalf("expected 1 described job, got %d", len(desc))
	}

	if err := svc.CancelJob(job.ID, "cancelled"); err != nil {
		t.Fatalf("cancel job: %v", err)
	}
	desc = svc.DescribeJobs([]string{job.ID})
	if len(desc) != 1 || desc[0].Status != "FAILED" {
		t.Fatalf("expected FAILED status after cancel")
	}

	if err := svc.TagResource(jd.ARN, map[string]string{"owner": "ops"}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}
	tags, ok := svc.ListTagsForResource(jd.ARN)
	if !ok {
		t.Fatalf("expected tags to exist")
	}
	if tags["owner"] != "ops" {
		t.Fatalf("expected owner tag")
	}
	if err := svc.UntagResource(jd.ARN, []string{"owner"}); err != nil {
		t.Fatalf("untag resource: %v", err)
	}
	tags, ok = svc.ListTagsForResource(jd.ARN)
	if !ok {
		t.Fatalf("expected tags to exist")
	}
	if _, exists := tags["owner"]; exists {
		t.Fatalf("expected owner tag removed")
	}

	if err := svc.DeregisterJobDefinition(jd.ARN); err != nil {
		t.Fatalf("deregister job definition: %v", err)
	}
	defs := svc.DescribeJobDefinitions([]string{jd.ARN}, "INACTIVE")
	if len(defs) != 1 {
		t.Fatalf("expected 1 inactive job definition, got %d", len(defs))
	}

	if err := svc.DeleteJobQueue(jq.Name); err != nil {
		t.Fatalf("delete job queue: %v", err)
	}
	if err := svc.DeleteComputeEnvironment(ce.Name); err != nil {
		t.Fatalf("delete compute environment: %v", err)
	}
}

func TestServiceSchedulingPolicyLifecycle(t *testing.T) {
	svc := NewService()

	sp, err := svc.CreateSchedulingPolicy("sp-1", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create scheduling policy: %v", err)
	}
	if sp.ARN == "" {
		t.Fatalf("expected arn")
	}

	updated, err := svc.UpdateSchedulingPolicy(sp.ARN, map[string]string{"team": "platform"})
	if err != nil {
		t.Fatalf("update scheduling policy: %v", err)
	}
	if updated.Tags["team"] != "platform" {
		t.Fatalf("expected updated tag")
	}

	list := svc.DescribeSchedulingPolicies(nil)
	if len(list) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(list))
	}

	if err := svc.DeleteSchedulingPolicy(sp.Name); err != nil {
		t.Fatalf("delete scheduling policy: %v", err)
	}
	if len(svc.DescribeSchedulingPolicies(nil)) != 0 {
		t.Fatalf("expected policies to be empty")
	}
}

func TestServiceErrors(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateComputeEnvironment("", "", "", 0, "", nil); err != ErrInvalidParameter {
		t.Fatalf("expected invalid parameter, got %v", err)
	}

	if _, err := svc.CreateComputeEnvironment("ce-1", "", "", 0, "", nil); err != nil {
		t.Fatalf("create compute environment: %v", err)
	}
	if _, err := svc.CreateComputeEnvironment("ce-1", "", "", 0, "", nil); err != ErrAlreadyExists {
		t.Fatalf("expected already exists, got %v", err)
	}

	if _, err := svc.CreateJobQueue("jq-1", 1, "", []ComputeEnvironmentOrder{{Order: 1, ComputeEnvironment: "missing"}}, "", nil); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}

	if _, err := svc.SubmitJob("job-1", "missing", "missing", nil, nil); err != ErrNotFound {
		t.Fatalf("expected not found on submit, got %v", err)
	}

	if err := svc.CancelJob("missing", "x"); err != ErrNotFound {
		t.Fatalf("expected not found for cancel, got %v", err)
	}

	if _, ok := svc.ListTagsForResource("arn:aws:batch:us-east-1:123456789012:job-definition/missing:1"); ok {
		t.Fatalf("expected tags lookup to fail")
	}
}

func TestServiceExtendedOperations(t *testing.T) {
	svc := NewService()

	ce, err := svc.CreateComputeEnvironment("ce-ext", "UNMANAGED", "ENABLED", 8, "", nil)
	if err != nil {
		t.Fatalf("create compute environment: %v", err)
	}
	jq, err := svc.CreateJobQueue("jq-ext", 1, "ENABLED", []ComputeEnvironmentOrder{{Order: 1, ComputeEnvironment: ce.Name}}, "", nil)
	if err != nil {
		t.Fatalf("create job queue: %v", err)
	}
	jd, err := svc.RegisterJobDefinition("jd-ext", "container", nil, nil)
	if err != nil {
		t.Fatalf("register job definition: %v", err)
	}

	cr, err := svc.CreateConsumableResource("gpu-hours", "REPLENISHABLE", 100, map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create consumable resource: %v", err)
	}
	if cr.ARN == "" {
		t.Fatalf("expected consumable resource arn")
	}
	if _, err := svc.DescribeConsumableResource(cr.Name); err != nil {
		t.Fatalf("describe consumable resource: %v", err)
	}
	if resources, _ := svc.ListConsumableResources(10, 0); len(resources) != 1 {
		t.Fatalf("expected one consumable resource, got %d", len(resources))
	}
	updatedCR, err := svc.UpdateConsumableResource(cr.Name, "ADD", 5)
	if err != nil {
		t.Fatalf("update consumable resource: %v", err)
	}
	if updatedCR.TotalQuantity != 105 {
		t.Fatalf("expected total quantity 105, got %d", updatedCR.TotalQuantity)
	}

	if _, err := svc.SubmitJobWithOptions(
		"job-ext",
		jq.Name,
		jd.Name,
		nil,
		nil,
		[]ConsumableResourceRequirement{{ConsumableResource: cr.Name, Quantity: 2}},
		"team-a",
		2,
	); err != nil {
		t.Fatalf("submit job with consumable requirements: %v", err)
	}

	listByConsumable, _, err := svc.ListJobsByConsumableResource(cr.Name, 10, 0)
	if err != nil {
		t.Fatalf("list jobs by consumable resource: %v", err)
	}
	if len(listByConsumable) != 1 {
		t.Fatalf("expected one job by consumable resource, got %d", len(listByConsumable))
	}

	snapshot, err := svc.GetJobQueueSnapshot(jq.Name)
	if err != nil {
		t.Fatalf("get job queue snapshot: %v", err)
	}
	if len(snapshot.Jobs) == 0 {
		t.Fatalf("expected snapshot jobs")
	}

	se, err := svc.CreateServiceEnvironment("svc-env", "SAGEMAKER_TRAINING", "ENABLED", []ServiceEnvironmentCapacity{{CapacityUnit: "GPU", MaxCapacity: 10}}, map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create service environment: %v", err)
	}
	if se.ARN == "" {
		t.Fatalf("expected service environment arn")
	}
	serviceEnvs, _ := svc.DescribeServiceEnvironments([]string{se.Name}, 10, 0)
	if len(serviceEnvs) != 1 {
		t.Fatalf("expected one service environment, got %d", len(serviceEnvs))
	}
	state := "DISABLED"
	updatedSE, err := svc.UpdateServiceEnvironment(se.Name, &state, []ServiceEnvironmentCapacity{{CapacityUnit: "GPU", MaxCapacity: 5}})
	if err != nil {
		t.Fatalf("update service environment: %v", err)
	}
	if updatedSE.State != "DISABLED" {
		t.Fatalf("expected updated service environment state")
	}

	serviceJob, err := svc.SubmitServiceJob(
		"svc-job",
		jq.Name,
		"SAGEMAKER_TRAINING",
		`{"foo":"bar"}`,
		1,
		"share-team",
		ServiceJobRetryStrategy{Attempts: 2},
		ServiceJobTimeout{AttemptDurationSeconds: 60},
		nil,
	)
	if err != nil {
		t.Fatalf("submit service job: %v", err)
	}
	if serviceJob.ID == "" {
		t.Fatalf("expected service job id")
	}
	if _, err := svc.DescribeServiceJob(serviceJob.ID); err != nil {
		t.Fatalf("describe service job: %v", err)
	}
	serviceJobs, _ := svc.ListServiceJobs(jq.Name, "", 10, 0)
	if len(serviceJobs) != 1 {
		t.Fatalf("expected one service job, got %d", len(serviceJobs))
	}
	if err := svc.TerminateServiceJob(serviceJob.ID, "cleanup"); err != nil {
		t.Fatalf("terminate service job: %v", err)
	}
	describedServiceJob, err := svc.DescribeServiceJob(serviceJob.ID)
	if err != nil {
		t.Fatalf("describe terminated service job: %v", err)
	}
	if !describedServiceJob.IsTerminated {
		t.Fatalf("expected service job to be terminated")
	}

	if err := svc.DeleteServiceEnvironment(se.Name); err != nil {
		t.Fatalf("delete service environment: %v", err)
	}
	if err := svc.DeleteConsumableResource(cr.Name); err != nil {
		t.Fatalf("delete consumable resource: %v", err)
	}
}
