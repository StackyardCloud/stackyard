package elasticbeanstalk

import "testing"

func TestServiceLifecycle(t *testing.T) {
	svc := NewService()

	app, err := svc.CreateApplication("demo-app", "demo", nil)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	if app.Name != "demo-app" {
		t.Fatalf("unexpected app name: %s", app.Name)
	}

	version, err := svc.CreateApplicationVersion("demo-app", "v1", "version 1", S3Location{S3Bucket: "bucket", S3Key: "key.zip"}, nil)
	if err != nil {
		t.Fatalf("create app version: %v", err)
	}
	if version.VersionLabel != "v1" {
		t.Fatalf("unexpected version label: %s", version.VersionLabel)
	}

	tpl, err := svc.CreateConfigurationTemplate("demo-app", "default", "template", "", "", nil)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	if tpl.TemplateName != "default" {
		t.Fatalf("unexpected template name: %s", tpl.TemplateName)
	}

	env, err := svc.CreateEnvironment("demo-app", "demo-env", "", "env", "", "default", "v1", nil, nil)
	if err != nil {
		t.Fatalf("create env: %v", err)
	}
	if env.Name != "demo-env" || env.ID == "" {
		t.Fatalf("unexpected env: %+v", env)
	}

	envs := svc.DescribeEnvironments("demo-app", nil, nil, false)
	if len(envs) != 1 {
		t.Fatalf("expected one environment, got %d", len(envs))
	}

	events := svc.DescribeEvents("demo-app", "demo-env", 10)
	if len(events) == 0 {
		t.Fatalf("expected events for environment")
	}

	if _, err := svc.UpdateEnvironment("", "demo-env", "v1", "default", "updated", []OptionSetting{{Namespace: "aws:elasticbeanstalk:application:environment", OptionName: "KEY", Value: "VALUE"}}); err != nil {
		t.Fatalf("update env: %v", err)
	}

	resources, err := svc.DescribeEnvironmentResources("", "demo-env")
	if err != nil {
		t.Fatalf("describe resources: %v", err)
	}
	if resources.EnvironmentName != "demo-env" {
		t.Fatalf("unexpected resources env name: %s", resources.EnvironmentName)
	}

	if err := svc.RequestEnvironmentInfo("", "demo-env", "tail"); err != nil {
		t.Fatalf("request env info: %v", err)
	}
	infos, err := svc.RetrieveEnvironmentInfo("", "demo-env", "tail")
	if err != nil {
		t.Fatalf("retrieve env info: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected one env info entry, got %d", len(infos))
	}

	if _, err := svc.TerminateEnvironment("", "demo-env"); err != nil {
		t.Fatalf("terminate env: %v", err)
	}

	if err := svc.DeleteApplicationVersion("demo-app", "v1"); err != nil {
		t.Fatalf("delete app version: %v", err)
	}
	if err := svc.DeleteConfigurationTemplate("demo-app", "default"); err != nil {
		t.Fatalf("delete template: %v", err)
	}
	if err := svc.DeleteApplication("demo-app", true); err != nil {
		t.Fatalf("delete app: %v", err)
	}
}

func TestCheckDNSAvailability(t *testing.T) {
	svc := NewService()
	if _, err := svc.CreateApplication("demo", "", nil); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if _, err := svc.CreateEnvironment("demo", "demo-env", "demo-prefix", "", "", "", "", nil, nil); err != nil {
		t.Fatalf("create env: %v", err)
	}

	available, fqdn, err := svc.CheckDNSAvailability("demo-prefix")
	if err != nil {
		t.Fatalf("check dns: %v", err)
	}
	if available {
		t.Fatalf("expected prefix to be unavailable")
	}
	if fqdn == "" {
		t.Fatalf("expected fully qualified cname")
	}
}

func TestValidateConfigurationSettings(t *testing.T) {
	svc := NewService()
	if _, err := svc.CreateApplication("demo", "", nil); err != nil {
		t.Fatalf("create app: %v", err)
	}

	messages := svc.ValidateConfigurationSettings("demo", []OptionSetting{{Namespace: "", OptionName: ""}})
	if len(messages) == 0 {
		t.Fatalf("expected validation messages")
	}
	if messages[0].Severity != "error" {
		t.Fatalf("expected error severity, got %q", messages[0].Severity)
	}
}

func TestStage1RoleAndAccountActions(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateApplication("demo", "", nil); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if _, err := svc.CreateApplicationVersion("demo", "v1", "", S3Location{}, nil); err != nil {
		t.Fatalf("create app version: %v", err)
	}
	if _, err := svc.CreateEnvironment("demo", "demo-env", "", "", "", "", "v1", nil, nil); err != nil {
		t.Fatalf("create env: %v", err)
	}

	quotas := svc.DescribeAccountAttributes()
	if quotas.ApplicationQuota.Maximum == 0 || quotas.EnvironmentQuota.Maximum == 0 {
		t.Fatalf("expected non-zero account quotas")
	}

	roleARN := "arn:aws:iam::123456789012:role/AWSElasticBeanstalkOperationsRole"
	if err := svc.AssociateEnvironmentOperationsRole("demo-env", roleARN); err != nil {
		t.Fatalf("associate operations role: %v", err)
	}
	envs := svc.DescribeEnvironments("demo", nil, []string{"demo-env"}, false)
	if len(envs) != 1 {
		t.Fatalf("expected one environment")
	}
	if envs[0].OperationsRole != roleARN {
		t.Fatalf("expected operations role %q, got %q", roleARN, envs[0].OperationsRole)
	}

	composed, err := svc.ComposeEnvironments("demo", "", []string{"v1"})
	if err != nil {
		t.Fatalf("compose environments: %v", err)
	}
	if len(composed) != 1 || composed[0].Name != "demo-env" {
		t.Fatalf("expected composed environment demo-env, got %+v", composed)
	}

	if err := svc.DisassociateEnvironmentOperationsRole("demo-env"); err != nil {
		t.Fatalf("disassociate operations role: %v", err)
	}
	envs = svc.DescribeEnvironments("demo", nil, []string{"demo-env"}, false)
	if len(envs) != 1 {
		t.Fatalf("expected one environment after disassociate")
	}
	if envs[0].OperationsRole != "" {
		t.Fatalf("expected empty operations role after disassociate, got %q", envs[0].OperationsRole)
	}
}

func TestStage2PlatformLifecycle(t *testing.T) {
	svc := NewService()

	platform, err := svc.CreatePlatformVersion(
		"demo-platform",
		"1.0.0",
		"builder-env",
		S3Location{S3Bucket: "bundle-bucket", S3Key: "platform.zip"},
		nil,
		[]Tag{{Key: "stage", Value: "2"}},
	)
	if err != nil {
		t.Fatalf("create platform version: %v", err)
	}
	if platform.PlatformARN == "" || platform.PlatformStatus != "Ready" {
		t.Fatalf("unexpected created platform: %+v", platform)
	}

	desc, err := svc.DescribePlatformVersion(platform.PlatformARN)
	if err != nil {
		t.Fatalf("describe platform version: %v", err)
	}
	if desc.PlatformName != "demo-platform" || desc.PlatformVersion != "1.0.0" {
		t.Fatalf("unexpected described platform: %+v", desc)
	}

	items, next, err := svc.ListPlatformVersions([]PlatformFilter{{
		Type:     "PlatformName",
		Operator: "=",
		Values:   []string{"demo-platform"},
	}}, 10, "")
	if err != nil {
		t.Fatalf("list platform versions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one listed platform, got %d", len(items))
	}
	if next != "" {
		t.Fatalf("unexpected next token: %q", next)
	}

	deleted, err := svc.DeletePlatformVersion(platform.PlatformARN)
	if err != nil {
		t.Fatalf("delete platform version: %v", err)
	}
	if deleted.PlatformStatus != "Deleted" {
		t.Fatalf("expected deleted status, got %q", deleted.PlatformStatus)
	}

	if _, err := svc.DescribePlatformVersion(platform.PlatformARN); err == nil {
		t.Fatalf("expected describe after delete to fail")
	}
}

func TestStage3HealthAndConfigActions(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateApplication("health-app", "", nil); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if _, err := svc.CreateApplicationVersion("health-app", "v1", "", S3Location{}, nil); err != nil {
		t.Fatalf("create app version: %v", err)
	}
	if _, err := svc.CreateEnvironment("health-app", "health-env", "", "", "", "", "v1", []OptionSetting{{
		Namespace:  "aws:elasticbeanstalk:application:environment",
		OptionName: "K",
		Value:      "V",
	}}, nil); err != nil {
		t.Fatalf("create env: %v", err)
	}

	health, err := svc.DescribeEnvironmentHealth("", "health-env")
	if err != nil {
		t.Fatalf("describe environment health: %v", err)
	}
	if health.EnvironmentName != "health-env" {
		t.Fatalf("unexpected environment health name: %s", health.EnvironmentName)
	}

	instances, nextToken, refreshedAt, err := svc.DescribeInstancesHealth("", "health-env", "")
	if err != nil {
		t.Fatalf("describe instances health: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance health entry, got %d", len(instances))
	}
	if nextToken != "" {
		t.Fatalf("unexpected next token: %q", nextToken)
	}
	if refreshedAt.IsZero() {
		t.Fatalf("expected refreshed timestamp")
	}

	if err := svc.DeleteEnvironmentConfiguration("health-app", "health-env"); err != nil {
		t.Fatalf("delete environment configuration: %v", err)
	}
	envs := svc.DescribeEnvironments("health-app", nil, []string{"health-env"}, false)
	if len(envs) != 1 {
		t.Fatalf("expected one environment after delete configuration")
	}
	if len(envs[0].OptionSettings) != 0 {
		t.Fatalf("expected option settings to be cleared")
	}

	if _, err := svc.CreatePlatformVersion("health-platform", "1.0.0", "", S3Location{S3Bucket: "b", S3Key: "k"}, nil, nil); err != nil {
		t.Fatalf("create platform for branch listing: %v", err)
	}
	branches, _, err := svc.ListPlatformBranches([]SearchFilter{{
		Attribute: "PlatformName",
		Operator:  "=",
		Values:    []string{"health-platform"},
	}}, 10, "")
	if err != nil {
		t.Fatalf("list platform branches: %v", err)
	}
	if len(branches) != 1 {
		t.Fatalf("expected one platform branch, got %d", len(branches))
	}
}

func TestStage4ManagedActionsAndApplicationLifecycle(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateApplication("stage4-app", "", nil); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if _, err := svc.CreateApplicationVersion("stage4-app", "v1", "", S3Location{}, nil); err != nil {
		t.Fatalf("create app version: %v", err)
	}
	if _, err := svc.CreateEnvironment("stage4-app", "stage4-env", "", "", "", "", "v1", nil, nil); err != nil {
		t.Fatalf("create env: %v", err)
	}

	actions, err := svc.DescribeEnvironmentManagedActions("", "stage4-env", "Scheduled")
	if err != nil {
		t.Fatalf("describe managed actions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected one scheduled managed action, got %d", len(actions))
	}

	applied, err := svc.ApplyEnvironmentManagedAction(actions[0].ActionID, "", "stage4-env")
	if err != nil {
		t.Fatalf("apply managed action: %v", err)
	}
	if applied.Status != "Running" {
		t.Fatalf("expected applied managed action status Running, got %q", applied.Status)
	}

	actions, err = svc.DescribeEnvironmentManagedActions("", "stage4-env", "")
	if err != nil {
		t.Fatalf("describe managed actions after apply: %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected no pending managed actions after apply")
	}

	history, nextToken, err := svc.DescribeEnvironmentManagedActionHistory("", "stage4-env", 10, "")
	if err != nil {
		t.Fatalf("describe managed action history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected one managed action history item, got %d", len(history))
	}
	if history[0].Status != "Completed" {
		t.Fatalf("expected history status Completed, got %q", history[0].Status)
	}
	if nextToken != "" {
		t.Fatalf("unexpected next token %q", nextToken)
	}

	firstUpdate, err := svc.UpdateApplicationResourceLifecycle("stage4-app", ApplicationResourceLifecycleConfig{
		ServiceRole: "arn:aws:iam::123456789012:role/eb-service-role",
		VersionLifecycleConfig: &ApplicationVersionLifecycleConfig{
			MaxAgeRule: &MaxAgeRule{
				Enabled:            true,
				DeleteSourceFromS3: true,
				MaxAgeInDays:       30,
			},
		},
	})
	if err != nil {
		t.Fatalf("update application resource lifecycle: %v", err)
	}
	if firstUpdate.ServiceRole == "" || firstUpdate.VersionLifecycleConfig == nil || firstUpdate.VersionLifecycleConfig.MaxAgeRule == nil {
		t.Fatalf("unexpected first lifecycle update result: %+v", firstUpdate)
	}

	secondUpdate, err := svc.UpdateApplicationResourceLifecycle("stage4-app", ApplicationResourceLifecycleConfig{
		VersionLifecycleConfig: &ApplicationVersionLifecycleConfig{
			MaxCountRule: &MaxCountRule{
				Enabled:            true,
				DeleteSourceFromS3: false,
				MaxCount:           25,
			},
		},
	})
	if err != nil {
		t.Fatalf("update application lifecycle with partial config: %v", err)
	}
	if secondUpdate.ServiceRole != firstUpdate.ServiceRole {
		t.Fatalf("expected service role to persist across partial updates")
	}
	if secondUpdate.VersionLifecycleConfig == nil || secondUpdate.VersionLifecycleConfig.MaxAgeRule == nil || secondUpdate.VersionLifecycleConfig.MaxCountRule == nil {
		t.Fatalf("expected lifecycle rules to merge across updates: %+v", secondUpdate.VersionLifecycleConfig)
	}
}
