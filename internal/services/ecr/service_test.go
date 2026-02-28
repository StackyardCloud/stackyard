package ecr

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
)

func TestServiceStage1RepositoryCoreAuthTags(t *testing.T) {
	svc := NewService()

	auth, err := svc.GetAuthorizationToken(nil)
	if err != nil {
		t.Fatalf("get authorization token: %v", err)
	}
	if len(auth) != 1 || auth[0].AuthorizationToken == "" {
		t.Fatalf("expected one authorization token entry")
	}

	repo, err := svc.CreateRepository("stage1-repo", "", nil, "", "", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	repos, _, err := svc.DescribeRepositories([]string{"stage1-repo"}, "", 10)
	if err != nil {
		t.Fatalf("describe repositories: %v", err)
	}
	if len(repos) != 1 || repos[0].RepositoryArn != repo.RepositoryArn {
		t.Fatalf("unexpected describe repositories output")
	}

	if err := svc.TagResource(repo.RepositoryArn, map[string]string{"team": "platform"}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}
	tags, err := svc.ListTagsForResource(repo.RepositoryArn)
	if err != nil {
		t.Fatalf("list tags for resource: %v", err)
	}
	if tags["env"] != "test" || tags["team"] != "platform" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
	if err := svc.UntagResource(repo.RepositoryArn, []string{"team"}); err != nil {
		t.Fatalf("untag resource: %v", err)
	}
	tags, err = svc.ListTagsForResource(repo.RepositoryArn)
	if err != nil {
		t.Fatalf("list tags for resource after untag: %v", err)
	}
	if _, ok := tags["team"]; ok {
		t.Fatalf("expected team tag to be removed")
	}

	if _, err := svc.DeleteRepository("stage1-repo", true); err != nil {
		t.Fatalf("delete repository: %v", err)
	}
}

func TestServiceStage2RepositoryPolicyMutabilityScanning(t *testing.T) {
	svc := NewService()
	repo := createRepositoryForTest(t, svc, "stage2-repo")

	const policyText = `{"Version":"2012-10-17","Statement":[]}`
	if _, gotPolicy, err := svc.SetRepositoryPolicy(repo.RepositoryName, policyText, false); err != nil {
		t.Fatalf("set repository policy: %v", err)
	} else if gotPolicy != policyText {
		t.Fatalf("unexpected repository policy text: %q", gotPolicy)
	}
	if _, gotPolicy, err := svc.GetRepositoryPolicy(repo.RepositoryName); err != nil {
		t.Fatalf("get repository policy: %v", err)
	} else if gotPolicy != policyText {
		t.Fatalf("unexpected repository policy text from get: %q", gotPolicy)
	}
	if _, _, err := svc.DeleteRepositoryPolicy(repo.RepositoryName); err != nil {
		t.Fatalf("delete repository policy: %v", err)
	}
	if _, _, err := svc.GetRepositoryPolicy(repo.RepositoryName); !errors.Is(err, ErrRepositoryPolicyNotFound) {
		t.Fatalf("expected ErrRepositoryPolicyNotFound, got %v", err)
	}

	if _, mutability, err := svc.PutImageTagMutability(repo.RepositoryName, "IMMUTABLE"); err != nil {
		t.Fatalf("put image tag mutability: %v", err)
	} else if mutability != "IMMUTABLE" {
		t.Fatalf("expected IMMUTABLE mutability, got %q", mutability)
	}

	if _, scanOnPush, err := svc.PutImageScanningConfiguration(repo.RepositoryName, true); err != nil {
		t.Fatalf("put image scanning configuration: %v", err)
	} else if !scanOnPush {
		t.Fatalf("expected scanOnPush=true")
	}

	configs, failures, err := svc.BatchGetRepositoryScanningConfiguration([]string{repo.RepositoryName, "missing-repo"})
	if err != nil {
		t.Fatalf("batch get repository scanning configuration: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected one scanning configuration, got %d", len(configs))
	}
	if len(failures) != 1 {
		t.Fatalf("expected one scanning failure for missing repo, got %d", len(failures))
	}
}

func TestServiceStage3PushPullPrimitives(t *testing.T) {
	svc := NewService()
	repo := createRepositoryForTest(t, svc, "stage3-repo")

	uploadID, partSize, err := svc.InitiateLayerUpload(repo.RepositoryName)
	if err != nil {
		t.Fatalf("initiate layer upload: %v", err)
	}
	if uploadID == "" || partSize <= 0 {
		t.Fatalf("unexpected initiate layer upload output: uploadID=%q partSize=%d", uploadID, partSize)
	}

	layerBlob := []byte("stage3-layer-data")
	lastByte, err := svc.UploadLayerPart(repo.RepositoryName, uploadID, 0, int64(len(layerBlob)-1), layerBlob)
	if err != nil {
		t.Fatalf("upload layer part: %v", err)
	}
	if lastByte != int64(len(layerBlob)-1) {
		t.Fatalf("unexpected last byte received: %d", lastByte)
	}

	layerDigest := sha256DigestForTest(layerBlob)
	completedLayerDigest, err := svc.CompleteLayerUpload(repo.RepositoryName, uploadID, []string{layerDigest})
	if err != nil {
		t.Fatalf("complete layer upload: %v", err)
	}
	if completedLayerDigest != layerDigest {
		t.Fatalf("expected layer digest %q, got %q", layerDigest, completedLayerDigest)
	}

	layers, layerFailures, err := svc.BatchCheckLayerAvailability(repo.RepositoryName, []string{layerDigest})
	if err != nil {
		t.Fatalf("batch check layer availability: %v", err)
	}
	if len(layerFailures) != 0 {
		t.Fatalf("expected no layer failures, got %d", len(layerFailures))
	}
	if len(layers) != 1 || layers[0].LayerAvailability != "AVAILABLE" {
		t.Fatalf("unexpected layer availability output: %#v", layers)
	}

	image := putImageForTest(t, svc, repo.RepositoryName, "latest", `{"schemaVersion":2}`)
	images, imageFailures, err := svc.BatchGetImage(repo.RepositoryName, []ImageIdentifier{{ImageTag: "latest"}}, nil)
	if err != nil {
		t.Fatalf("batch get image: %v", err)
	}
	if len(imageFailures) != 0 {
		t.Fatalf("expected no image failures, got %d", len(imageFailures))
	}
	if len(images) != 1 || images[0].ImageID.ImageDigest != image.ImageID.ImageDigest {
		t.Fatalf("unexpected batch get image output: %#v", images)
	}

	downloadURL, outDigest, err := svc.GetDownloadURLForLayer(repo.RepositoryName, layerDigest)
	if err != nil {
		t.Fatalf("get download url for layer: %v", err)
	}
	if downloadURL == "" || outDigest != layerDigest {
		t.Fatalf("unexpected get download url output: url=%q digest=%q", downloadURL, outDigest)
	}

	imageIDs, _, err := svc.ListImages(repo.RepositoryName, "ANY", "", 100)
	if err != nil {
		t.Fatalf("list images: %v", err)
	}
	if len(imageIDs) != 1 {
		t.Fatalf("expected one image id, got %d", len(imageIDs))
	}
}

func TestServiceStage4ImageCatalogAndDeletion(t *testing.T) {
	svc := NewService()
	repo := createRepositoryForTest(t, svc, "stage4-repo")

	imageV1 := putImageForTest(t, svc, repo.RepositoryName, "v1", `{"schemaVersion":2}`)
	_ = putImageForTest(t, svc, repo.RepositoryName, "v2", `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)

	details, _, err := svc.DescribeImages(repo.RepositoryName, nil, "ANY", "", 100)
	if err != nil {
		t.Fatalf("describe images: %v", err)
	}
	if len(details) != 2 {
		t.Fatalf("expected two image details, got %d", len(details))
	}

	deleted, failures, err := svc.BatchDeleteImage(repo.RepositoryName, []ImageIdentifier{{ImageTag: "v1"}})
	if err != nil {
		t.Fatalf("batch delete image: %v", err)
	}
	if len(failures) != 0 || len(deleted) != 1 || deleted[0].ImageDigest != imageV1.ImageID.ImageDigest {
		t.Fatalf("unexpected batch delete output: deleted=%#v failures=%#v", deleted, failures)
	}
}

func TestServiceStage5LifecyclePolicy(t *testing.T) {
	svc := NewService()
	repo := createRepositoryForTest(t, svc, "stage5-repo")
	putImageForTest(t, svc, repo.RepositoryName, "latest", `{"schemaVersion":2}`)

	const lifecyclePolicyText = `{"rules":[{"rulePriority":1,"selection":{"tagStatus":"untagged","countType":"imageCountMoreThan","countNumber":1},"action":{"type":"expire"}}]}`
	if _, err := svc.PutLifecyclePolicy(repo.RepositoryName, lifecyclePolicyText); err != nil {
		t.Fatalf("put lifecycle policy: %v", err)
	}
	policy, err := svc.GetLifecyclePolicy(repo.RepositoryName)
	if err != nil {
		t.Fatalf("get lifecycle policy: %v", err)
	}
	if policy.LifecyclePolicyText != lifecyclePolicyText {
		t.Fatalf("unexpected lifecycle policy text")
	}

	if _, status, err := svc.StartLifecyclePolicyPreview(repo.RepositoryName, ""); err != nil {
		t.Fatalf("start lifecycle policy preview: %v", err)
	} else if status == "" {
		t.Fatalf("expected lifecycle preview status")
	}

	_, _, previewResults, summary, _, err := svc.GetLifecyclePolicyPreview(repo.RepositoryName, nil, "ANY", "", 100)
	if err != nil {
		t.Fatalf("get lifecycle policy preview: %v", err)
	}
	if int32(len(previewResults)) != summary.ExpiringImageTotalCount {
		t.Fatalf("expected summary count to match preview results")
	}

	if _, err := svc.DeleteLifecyclePolicy(repo.RepositoryName); err != nil {
		t.Fatalf("delete lifecycle policy: %v", err)
	}
	if _, err := svc.GetLifecyclePolicy(repo.RepositoryName); !errors.Is(err, ErrLifecyclePolicyNotFound) {
		t.Fatalf("expected ErrLifecyclePolicyNotFound, got %v", err)
	}
}

func TestServiceStage6RegistryGlobalSettings(t *testing.T) {
	svc := NewService()
	repo := createRepositoryForTest(t, svc, "stage6-repo")
	putImageForTest(t, svc, repo.RepositoryName, "latest", `{"schemaVersion":2}`)

	if _, _, err := svc.DescribeRegistry(); err != nil {
		t.Fatalf("describe registry: %v", err)
	}

	const registryPolicyText = `{"Version":"2012-10-17","Statement":[]}`
	if _, gotPolicy, err := svc.PutRegistryPolicy(registryPolicyText); err != nil {
		t.Fatalf("put registry policy: %v", err)
	} else if gotPolicy != registryPolicyText {
		t.Fatalf("unexpected registry policy text from put")
	}
	if _, gotPolicy, err := svc.GetRegistryPolicy(); err != nil {
		t.Fatalf("get registry policy: %v", err)
	} else if gotPolicy != registryPolicyText {
		t.Fatalf("unexpected registry policy text from get")
	}
	if _, _, err := svc.DeleteRegistryPolicy(); err != nil {
		t.Fatalf("delete registry policy: %v", err)
	}
	if _, _, err := svc.GetRegistryPolicy(); !errors.Is(err, ErrRegistryPolicyNotFound) {
		t.Fatalf("expected ErrRegistryPolicyNotFound, got %v", err)
	}

	rules := []RegistryScanningRule{
		{
			RepositoryFilters: []ScanningRepositoryFilter{
				{Filter: "*", FilterType: "WILDCARD"},
			},
			ScanFrequency: "SCAN_ON_PUSH",
		},
	}
	if _, err := svc.PutRegistryScanningConfiguration("BASIC", rules); err != nil {
		t.Fatalf("put registry scanning configuration: %v", err)
	}
	if _, cfg, err := svc.GetRegistryScanningConfiguration(); err != nil {
		t.Fatalf("get registry scanning configuration: %v", err)
	} else if cfg.ScanType != "BASIC" || len(cfg.Rules) != 1 {
		t.Fatalf("unexpected registry scanning configuration: %#v", cfg)
	}

	replicationConfig := ReplicationConfiguration{
		Rules: []ReplicationRule{
			{
				Destinations: []ReplicationDestination{
					{Region: "us-west-2", RegistryID: DefaultAccountID},
				},
				RepositoryFilters: []RepositoryFilter{
					{Filter: "stage6", FilterType: "PREFIX_MATCH"},
				},
			},
		},
	}
	if _, err := svc.PutReplicationConfiguration(replicationConfig); err != nil {
		t.Fatalf("put replication configuration: %v", err)
	}

	if _, err := svc.PutAccountSetting("REGISTRY_POLICY_SCOPE", "V2"); err != nil {
		t.Fatalf("put account setting: %v", err)
	}
	if setting, err := svc.GetAccountSetting("REGISTRY_POLICY_SCOPE"); err != nil {
		t.Fatalf("get account setting: %v", err)
	} else if setting.Value != "V2" {
		t.Fatalf("unexpected account setting value: %q", setting.Value)
	}

	if _, statuses, _, err := svc.DescribeImageReplicationStatus(repo.RepositoryName, ImageIdentifier{ImageTag: "latest"}); err != nil {
		t.Fatalf("describe image replication status: %v", err)
	} else if len(statuses) != 1 || statuses[0].Status != "COMPLETE" {
		t.Fatalf("unexpected replication status output: %#v", statuses)
	}
}

func TestServiceStage7PullThroughCacheRules(t *testing.T) {
	svc := NewService()

	rule, err := svc.CreatePullThroughCacheRule("stage7", "registry-1.docker.io", "", "")
	if err != nil {
		t.Fatalf("create pull-through cache rule: %v", err)
	}
	if rule.UpstreamRegistry == "" {
		t.Fatalf("expected inferred upstream registry")
	}

	rules, _, err := svc.DescribePullThroughCacheRules([]string{"stage7"}, "", 100)
	if err != nil {
		t.Fatalf("describe pull-through cache rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected one pull-through cache rule, got %d", len(rules))
	}

	updated, err := svc.UpdatePullThroughCacheRule("stage7", "arn:aws:secretsmanager:us-east-1:123456789012:secret:ecr")
	if err != nil {
		t.Fatalf("update pull-through cache rule: %v", err)
	}
	if updated.CredentialArn == "" {
		t.Fatalf("expected credential arn after update")
	}

	if _, valid, _, err := svc.ValidatePullThroughCacheRule("stage7"); err != nil {
		t.Fatalf("validate pull-through cache rule: %v", err)
	} else if !valid {
		t.Fatalf("expected pull-through cache rule to be valid")
	}

	if _, err := svc.DeletePullThroughCacheRule("stage7"); err != nil {
		t.Fatalf("delete pull-through cache rule: %v", err)
	}
	if _, err := svc.UpdatePullThroughCacheRule("stage7", "arn:aws:secretsmanager:us-east-1:123456789012:secret:ecr"); !errors.Is(err, ErrPullThroughRuleNotFound) {
		t.Fatalf("expected ErrPullThroughRuleNotFound, got %v", err)
	}
}

func TestServiceStage8RepositoryCreationTemplates(t *testing.T) {
	svc := NewService()

	template, err := svc.CreateRepositoryCreationTemplate(RepositoryCreationTemplateInput{
		Prefix:        "stage8",
		AppliedFor:    []string{"REPLICATION"},
		AppliedForSet: true,
	})
	if err != nil {
		t.Fatalf("create repository creation template: %v", err)
	}
	if template.Prefix != "stage8" {
		t.Fatalf("unexpected template prefix: %q", template.Prefix)
	}

	templates, _, err := svc.DescribeRepositoryCreationTemplates([]string{"stage8"}, "", 100)
	if err != nil {
		t.Fatalf("describe repository creation templates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected one template, got %d", len(templates))
	}

	description := "updated template"
	imageTagMutability := "IMMUTABLE"
	updated, err := svc.UpdateRepositoryCreationTemplate(RepositoryCreationTemplateInput{
		Prefix:             "stage8",
		Description:        &description,
		ImageTagMutability: &imageTagMutability,
	})
	if err != nil {
		t.Fatalf("update repository creation template: %v", err)
	}
	if updated.Description != description || updated.ImageTagMutability != imageTagMutability {
		t.Fatalf("unexpected updated template output: %#v", updated)
	}

	if _, err := svc.DeleteRepositoryCreationTemplate("stage8"); err != nil {
		t.Fatalf("delete repository creation template: %v", err)
	}
	if _, err := svc.UpdateRepositoryCreationTemplate(RepositoryCreationTemplateInput{
		Prefix:      "stage8",
		Description: &description,
	}); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestServiceStage9SigningConfiguration(t *testing.T) {
	svc := NewService()
	repo := createRepositoryForTest(t, svc, "stage9-repo")
	putImageForTest(t, svc, repo.RepositoryName, "v1", `{"schemaVersion":2}`)

	signingConfig := SigningConfiguration{
		Rules: []SigningRule{
			{
				SigningProfileArn: "arn:aws:signer:us-east-1:123456789012:/signing-profiles/demo",
				RepositoryFilters: []SigningRepositoryFilter{
					{Filter: "stage9*", FilterType: "WILDCARD_MATCH"},
				},
			},
		},
	}
	if _, err := svc.PutSigningConfiguration(signingConfig); err != nil {
		t.Fatalf("put signing configuration: %v", err)
	}
	if _, gotConfig, err := svc.GetSigningConfiguration(); err != nil {
		t.Fatalf("get signing configuration: %v", err)
	} else if len(gotConfig.Rules) != 1 {
		t.Fatalf("expected one signing rule, got %d", len(gotConfig.Rules))
	}

	_, _, _, statuses, err := svc.DescribeImageSigningStatus(repo.RepositoryName, ImageIdentifier{ImageTag: "v1"})
	if err != nil {
		t.Fatalf("describe image signing status: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Status != "COMPLETE" {
		t.Fatalf("unexpected image signing statuses: %#v", statuses)
	}

	if _, _, err := svc.DeleteSigningConfiguration(); err != nil {
		t.Fatalf("delete signing configuration: %v", err)
	}
	if _, _, err := svc.GetSigningConfiguration(); !errors.Is(err, ErrSigningConfigNotFound) {
		t.Fatalf("expected ErrSigningConfigNotFound, got %v", err)
	}
}

func TestServiceStage10ScanExecution(t *testing.T) {
	svc := NewService()
	repo := createRepositoryForTest(t, svc, "stage10-repo")
	putImageForTest(t, svc, repo.RepositoryName, "latest", `{"schemaVersion":2}`)

	if _, status, _, _, err := svc.StartImageScan(repo.RepositoryName, ImageIdentifier{ImageTag: "latest"}); err != nil {
		t.Fatalf("start image scan: %v", err)
	} else if status.Status != "COMPLETE" {
		t.Fatalf("expected COMPLETE scan status, got %s", status.Status)
	}

	_, findings, status, _, _, _, err := svc.DescribeImageScanFindings(repo.RepositoryName, ImageIdentifier{ImageTag: "latest"}, "", 100)
	if err != nil {
		t.Fatalf("describe image scan findings: %v", err)
	}
	if status.Status != "COMPLETE" {
		t.Fatalf("expected COMPLETE findings status, got %s", status.Status)
	}
	if len(findings.Findings) == 0 {
		t.Fatalf("expected scan findings")
	}
}

func TestServiceStage11PullTimeReferrersStorageClass(t *testing.T) {
	svc := NewService()
	repo := createRepositoryForTest(t, svc, "stage11-repo")

	subjectImage := putImageForTest(t, svc, repo.RepositoryName, "subject", `{"schemaVersion":2}`)
	referrerManifest := fmt.Sprintf(
		`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","artifactType":"application/vnd.example.signature","subject":{"digest":"%s"}}`,
		subjectImage.ImageID.ImageDigest,
	)
	putImageForTest(t, svc, repo.RepositoryName, "signature", referrerManifest)

	principalARN := "arn:aws:iam::123456789012:role/example"
	if _, err := svc.RegisterPullTimeUpdateExclusion(principalARN); err != nil {
		t.Fatalf("register pull-time update exclusion: %v", err)
	}
	exclusions, _, err := svc.ListPullTimeUpdateExclusions("", 100)
	if err != nil {
		t.Fatalf("list pull-time update exclusions: %v", err)
	}
	if len(exclusions) != 1 || exclusions[0] != principalARN {
		t.Fatalf("unexpected exclusions output: %#v", exclusions)
	}

	referrers, _, err := svc.ListImageReferrers(repo.RepositoryName, subjectImage.ImageID.ImageDigest, "ANY", nil, "", 100)
	if err != nil {
		t.Fatalf("list image referrers: %v", err)
	}
	if len(referrers) != 1 || referrers[0].Digest == "" {
		t.Fatalf("unexpected referrers output: %#v", referrers)
	}

	_, imageStatus, _, _, err := svc.UpdateImageStorageClass(repo.RepositoryName, ImageIdentifier{ImageTag: "subject"}, "ARCHIVE")
	if err != nil {
		t.Fatalf("update image storage class: %v", err)
	}
	if imageStatus != "ARCHIVED" {
		t.Fatalf("expected ARCHIVED image status, got %s", imageStatus)
	}

	if _, err := svc.DeregisterPullTimeUpdateExclusion(principalARN); err != nil {
		t.Fatalf("deregister pull-time update exclusion: %v", err)
	}
}

func createRepositoryForTest(t *testing.T, svc *Service, name string) Repository {
	t.Helper()
	repo, err := svc.CreateRepository(name, "", nil, "", "", nil)
	if err != nil {
		t.Fatalf("create repository %s: %v", name, err)
	}
	return repo
}

func putImageForTest(t *testing.T, svc *Service, repositoryName, tag, manifest string) Image {
	t.Helper()
	image, err := svc.PutImage(repositoryName, manifest, "", tag, "")
	if err != nil {
		t.Fatalf("put image repo=%s tag=%s: %v", repositoryName, tag, err)
	}
	return image
}

func sha256DigestForTest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}
