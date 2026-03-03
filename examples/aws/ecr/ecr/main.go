package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	awsecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	repositoryName := getenv("STACKYARD_REPOSITORY_NAME", "ecr-repo")
	cachePrefix := getenv("STACKYARD_CACHE_PREFIX", "ecr-cache")
	templatePrefix := getenv("STACKYARD_TEMPLATE_PREFIX", "ecr-template")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		exitf("load aws config: %v", err)
	}

	client := ecr.NewFromConfig(cfg, func(o *ecr.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	fmt.Printf("Stackyard ECR advanced client using %s\n", endpoint)

	createRepoOut, err := client.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(repositoryName),
		Tags: []awsecrtypes.Tag{
			{Key: aws.String("env"), Value: aws.String("dev")},
		},
	})
	if err != nil {
		exitf("create repository: %v", err)
	}
	repositoryArn := aws.ToString(createRepoOut.Repository.RepositoryArn)
	logf("created repository: %s", repositoryName)

	if _, err := client.PutImage(ctx, &ecr.PutImageInput{
		RepositoryName: aws.String(repositoryName),
		ImageManifest:  aws.String(`{"schemaVersion":2}`),
		ImageTag:       aws.String("latest"),
	}); err != nil {
		exitf("put image: %v", err)
	}

	const policyText = `{"Version":"2012-10-17","Statement":[]}`
	if _, err := client.SetRepositoryPolicy(ctx, &ecr.SetRepositoryPolicyInput{
		RepositoryName: aws.String(repositoryName),
		PolicyText:     aws.String(policyText),
	}); err != nil {
		exitf("set repository policy: %v", err)
	}
	getPolicyOut, err := client.GetRepositoryPolicy(ctx, &ecr.GetRepositoryPolicyInput{
		RepositoryName: aws.String(repositoryName),
	})
	if err != nil {
		exitf("get repository policy: %v", err)
	}
	logf("repository policy length: %d", len(aws.ToString(getPolicyOut.PolicyText)))

	if _, err := client.PutImageScanningConfiguration(ctx, &ecr.PutImageScanningConfigurationInput{
		RepositoryName: aws.String(repositoryName),
		ImageScanningConfiguration: &awsecrtypes.ImageScanningConfiguration{
			ScanOnPush: true,
		},
	}); err != nil {
		exitf("put image scanning configuration: %v", err)
	}
	scanningOut, err := client.BatchGetRepositoryScanningConfiguration(ctx, &ecr.BatchGetRepositoryScanningConfigurationInput{
		RepositoryNames: []string{repositoryName},
	})
	if err != nil {
		exitf("batch get repository scanning configuration: %v", err)
	}
	logf("repository scanning configs: %d", len(scanningOut.ScanningConfigurations))

	startScanOut, err := client.StartImageScan(ctx, &ecr.StartImageScanInput{
		RepositoryName: aws.String(repositoryName),
		ImageId: &awsecrtypes.ImageIdentifier{
			ImageTag: aws.String("latest"),
		},
	})
	if err != nil {
		exitf("start image scan: %v", err)
	}
	logf("scan status: %s", startScanOut.ImageScanStatus.Status)

	scanFindingsOut, err := client.DescribeImageScanFindings(ctx, &ecr.DescribeImageScanFindingsInput{
		RepositoryName: aws.String(repositoryName),
		ImageId: &awsecrtypes.ImageIdentifier{
			ImageTag: aws.String("latest"),
		},
	})
	if err != nil {
		exitf("describe image scan findings: %v", err)
	}
	logf("scan findings: %d", len(scanFindingsOut.ImageScanFindings.Findings))

	const lifecyclePolicyText = `{"rules":[{"rulePriority":1,"description":"expire untagged","selection":{"tagStatus":"untagged","countType":"imageCountMoreThan","countNumber":1},"action":{"type":"expire"}}]}`
	if _, err := client.PutLifecyclePolicy(ctx, &ecr.PutLifecyclePolicyInput{
		RepositoryName:      aws.String(repositoryName),
		LifecyclePolicyText: aws.String(lifecyclePolicyText),
	}); err != nil {
		exitf("put lifecycle policy: %v", err)
	}
	if _, err := client.StartLifecyclePolicyPreview(ctx, &ecr.StartLifecyclePolicyPreviewInput{
		RepositoryName: aws.String(repositoryName),
	}); err != nil {
		exitf("start lifecycle policy preview: %v", err)
	}
	lifecyclePreviewOut, err := client.GetLifecyclePolicyPreview(ctx, &ecr.GetLifecyclePolicyPreviewInput{
		RepositoryName: aws.String(repositoryName),
	})
	if err != nil {
		exitf("get lifecycle policy preview: %v", err)
	}
	logf("lifecycle preview results: %d", len(lifecyclePreviewOut.PreviewResults))

	if _, err := client.CreatePullThroughCacheRule(ctx, &ecr.CreatePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(cachePrefix),
		UpstreamRegistryUrl: aws.String("registry-1.docker.io"),
	}); err != nil {
		exitf("create pull-through cache rule: %v", err)
	}
	cacheRulesOut, err := client.DescribePullThroughCacheRules(ctx, &ecr.DescribePullThroughCacheRulesInput{
		EcrRepositoryPrefixes: []string{cachePrefix},
	})
	if err != nil {
		exitf("describe pull-through cache rules: %v", err)
	}
	logf("pull-through cache rules: %d", len(cacheRulesOut.PullThroughCacheRules))

	if _, err := client.UpdatePullThroughCacheRule(ctx, &ecr.UpdatePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(cachePrefix),
		CredentialArn:       aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:ecr"),
	}); err != nil {
		exitf("update pull-through cache rule: %v", err)
	}
	validateRuleOut, err := client.ValidatePullThroughCacheRule(ctx, &ecr.ValidatePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(cachePrefix),
	})
	if err != nil {
		exitf("validate pull-through cache rule: %v", err)
	}
	logf("pull-through cache rule valid: %t", validateRuleOut.IsValid)

	if _, err := client.CreateRepositoryCreationTemplate(ctx, &ecr.CreateRepositoryCreationTemplateInput{
		Prefix:     aws.String(templatePrefix),
		AppliedFor: []awsecrtypes.RCTAppliedFor{awsecrtypes.RCTAppliedForReplication},
	}); err != nil {
		exitf("create repository creation template: %v", err)
	}
	templateDescribeOut, err := client.DescribeRepositoryCreationTemplates(ctx, &ecr.DescribeRepositoryCreationTemplatesInput{
		Prefixes: []string{templatePrefix},
	})
	if err != nil {
		exitf("describe repository creation templates: %v", err)
	}
	logf("repository creation templates: %d", len(templateDescribeOut.RepositoryCreationTemplates))

	if _, err := client.UpdateRepositoryCreationTemplate(ctx, &ecr.UpdateRepositoryCreationTemplateInput{
		Prefix:             aws.String(templatePrefix),
		Description:        aws.String("stackyard template"),
		ImageTagMutability: awsecrtypes.ImageTagMutabilityImmutable,
	}); err != nil {
		exitf("update repository creation template: %v", err)
	}

	if _, err := client.TagResource(ctx, &ecr.TagResourceInput{
		ResourceArn: aws.String(repositoryArn),
		Tags: []awsecrtypes.Tag{
			{Key: aws.String("team"), Value: aws.String("platform")},
		},
	}); err != nil {
		exitf("tag resource: %v", err)
	}
	tagOut, err := client.ListTagsForResource(ctx, &ecr.ListTagsForResourceInput{
		ResourceArn: aws.String(repositoryArn),
	})
	if err != nil {
		exitf("list tags for resource: %v", err)
	}
	logf("resource tags: %d", len(tagOut.Tags))
	if _, err := client.UntagResource(ctx, &ecr.UntagResourceInput{
		ResourceArn: aws.String(repositoryArn),
		TagKeys:     []string{"team"},
	}); err != nil {
		exitf("untag resource: %v", err)
	}

	putSigningOut, err := ecrRawJSONRequest(ctx, endpoint, region, creds, "PutSigningConfiguration", map[string]any{
		"signingConfiguration": map[string]any{
			"rules": []map[string]any{
				{
					"signingProfileArn": "arn:aws:signer:us-east-1:123456789012:/signing-profiles/demo",
					"repositoryFilters": []map[string]any{
						{"filter": repositoryName, "filterType": "WILDCARD_MATCH"},
					},
				},
			},
		},
	})
	if err != nil {
		exitf("put signing configuration: %v", err)
	}
	logf("put signing configuration keys: %d", len(putSigningOut))

	if _, err := ecrRawJSONRequest(ctx, endpoint, region, creds, "GetSigningConfiguration", map[string]any{}); err != nil {
		exitf("get signing configuration: %v", err)
	}
	signingStatusOut, err := ecrRawJSONRequest(ctx, endpoint, region, creds, "DescribeImageSigningStatus", map[string]any{
		"repositoryName": repositoryName,
		"imageId": map[string]any{
			"imageTag": "latest",
		},
	})
	if err != nil {
		exitf("describe image signing status: %v", err)
	}
	logf("image signing fields: %d", len(signingStatusOut))
	if _, err := ecrRawJSONRequest(ctx, endpoint, region, creds, "DeleteSigningConfiguration", map[string]any{}); err != nil {
		exitf("delete signing configuration: %v", err)
	}

	if _, err := client.DeleteRepositoryCreationTemplate(ctx, &ecr.DeleteRepositoryCreationTemplateInput{
		Prefix: aws.String(templatePrefix),
	}); err != nil {
		exitf("delete repository creation template: %v", err)
	}
	if _, err := client.DeletePullThroughCacheRule(ctx, &ecr.DeletePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(cachePrefix),
	}); err != nil {
		exitf("delete pull-through cache rule: %v", err)
	}
	if _, err := client.DeleteLifecyclePolicy(ctx, &ecr.DeleteLifecyclePolicyInput{
		RepositoryName: aws.String(repositoryName),
	}); err != nil {
		exitf("delete lifecycle policy: %v", err)
	}
	if _, err := client.DeleteRepositoryPolicy(ctx, &ecr.DeleteRepositoryPolicyInput{
		RepositoryName: aws.String(repositoryName),
	}); err != nil {
		exitf("delete repository policy: %v", err)
	}
	if _, err := client.DeleteRepository(ctx, &ecr.DeleteRepositoryInput{
		RepositoryName: aws.String(repositoryName),
		Force:          true,
	}); err != nil {
		exitf("delete repository: %v", err)
	}

	fmt.Println("Done.")
}

func ecrRawJSONRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, action string, payload map[string]any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerRegistry_V20150921."+action)

	credentialsValue, err := creds.Retrieve(ctx)
	if err != nil {
		return nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialsValue, req, hashSHA256(body), "ecr", region, time.Now()); err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("%s failed (%d): %s", action, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if len(respBody) == 0 {
		return map[string]any{}, nil
	}

	var out map[string]any
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
