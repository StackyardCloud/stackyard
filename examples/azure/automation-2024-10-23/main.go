package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/automation/armautomation"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_AUTOMATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AUTOMATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-automation")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	automationAccountName := getenv("STACKYARD_AZURE_AUTOMATION_ACCOUNT_NAME", "aa-example")

	variableName := getenv("STACKYARD_AZURE_AUTOMATION_VARIABLE", "var-example")
	credentialName := getenv("STACKYARD_AZURE_AUTOMATION_CREDENTIAL", "cred-example")
	certificateName := getenv("STACKYARD_AZURE_AUTOMATION_CERTIFICATE", "cert-example")
	runbookName := getenv("STACKYARD_AZURE_AUTOMATION_RUNBOOK", "rb-example")
	scheduleName := getenv("STACKYARD_AZURE_AUTOMATION_SCHEDULE", "sched-example")
	jobScheduleID := getenv("STACKYARD_AZURE_AUTOMATION_JOB_SCHEDULE", "job-sched-example")
	webhookName := getenv("STACKYARD_AZURE_AUTOMATION_WEBHOOK", "webhook-example")
	jobName := getenv("STACKYARD_AZURE_AUTOMATION_JOB", "job-example")
	sourceControlName := getenv("STACKYARD_AZURE_AUTOMATION_SOURCE_CONTROL", "sc-example")
	syncJobID := getenv("STACKYARD_AZURE_AUTOMATION_SYNC_JOB", "sync-example")

	fmt.Printf("Stackyard Azure Automation typed SDK example using %s\n", endpoint)

	credential := azsdkshim.StaticTokenCredential{}
	armOptions := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			InsecureAllowCredentialWithHTTP: true,
			Cloud: cloud.Configuration{
				ActiveDirectoryAuthorityHost: endpoint,
				Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
					cloud.ResourceManager: {
						Endpoint: endpoint,
						Audience: endpoint,
					},
				},
			},
			Transport: azsdkshim.NewTransport("/azure", account, subscriptionKey),
		},
	}

	factory, err := armautomation.NewClientFactory(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create automation client factory failed: %v", err)
	}

	operationsClient := factory.NewOperationsClient()
	accountClient := factory.NewAccountClient()
	variableClient := factory.NewVariableClient()
	credentialClient := factory.NewCredentialClient()
	certificateClient := factory.NewCertificateClient()
	runbookClient := factory.NewRunbookClient()
	runbookDraftClient := factory.NewRunbookDraftClient()
	scheduleClient := factory.NewScheduleClient()
	jobScheduleClient := factory.NewJobScheduleClient()
	webhookClient := factory.NewWebhookClient()
	jobClient := factory.NewJobClient()
	jobStreamClient := factory.NewJobStreamClient()
	sourceControlClient := factory.NewSourceControlClient()
	sourceControlSyncJobClient := factory.NewSourceControlSyncJobClient()
	sourceControlSyncJobStreamsClient := factory.NewSourceControlSyncJobStreamsClient()
	statisticsClient := factory.NewStatisticsClient()
	usagesClient := factory.NewUsagesClient()
	privateLinkResourcesClient := factory.NewPrivateLinkResourcesClient()

	var generatedWebhookURI *string
	var firstJobStreamID string
	var firstSyncJobStreamID string

	expiry := time.Now().UTC().Add(24 * time.Hour)
	startTime := time.Now().UTC().Add(15 * time.Minute)

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "ListOperations",
			run: func() error {
				return drainPager(ctx, operationsClient.NewListPager(nil))
			},
		},
		{
			name: "CreateAutomationAccount",
			run: func() error {
				_, err := accountClient.CreateOrUpdate(ctx, resourceGroup, automationAccountName, armautomation.AccountCreateOrUpdateParameters{
					Location: to.Ptr(region),
					Tags: map[string]*string{
						"example": to.Ptr("stackyard"),
					},
					Properties: &armautomation.AccountCreateOrUpdateProperties{
						DisableLocalAuth:    to.Ptr(false),
						PublicNetworkAccess: to.Ptr(true),
						SKU: &armautomation.SKU{
							Name: to.Ptr(armautomation.SKUNameEnumBasic),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetAutomationAccount",
			run: func() error {
				_, err := accountClient.Get(ctx, resourceGroup, automationAccountName, nil)
				return err
			},
		},
		{
			name: "ListAutomationAccountsByResourceGroup",
			run: func() error {
				return drainPager(ctx, accountClient.NewListByResourceGroupPager(resourceGroup, nil))
			},
		},
		{
			name: "ListAutomationAccountsBySubscription",
			run: func() error {
				return drainPager(ctx, accountClient.NewListPager(nil))
			},
		},
		{
			name: "UpdateAutomationAccount",
			run: func() error {
				_, err := accountClient.Update(ctx, resourceGroup, automationAccountName, armautomation.AccountUpdateParameters{
					Tags: map[string]*string{
						"example": to.Ptr("stackyard"),
						"stage":   to.Ptr("demo"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateVariable",
			run: func() error {
				_, err := variableClient.CreateOrUpdate(ctx, resourceGroup, automationAccountName, variableName, armautomation.VariableCreateOrUpdateParameters{
					Name: to.Ptr(variableName),
					Properties: &armautomation.VariableCreateOrUpdateProperties{
						Value:       to.Ptr("stackyard"),
						IsEncrypted: to.Ptr(false),
						Description: to.Ptr("automation example variable"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetVariable",
			run: func() error {
				_, err := variableClient.Get(ctx, resourceGroup, automationAccountName, variableName, nil)
				return err
			},
		},
		{
			name: "ListVariables",
			run: func() error {
				return drainPager(ctx, variableClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "CreateCredential",
			run: func() error {
				_, err := credentialClient.CreateOrUpdate(ctx, resourceGroup, automationAccountName, credentialName, armautomation.CredentialCreateOrUpdateParameters{
					Name: to.Ptr(credentialName),
					Properties: &armautomation.CredentialCreateOrUpdateProperties{
						UserName:    to.Ptr("stackyard-user"),
						Password:    to.Ptr("secret-password"),
						Description: to.Ptr("automation example credential"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetCredential",
			run: func() error {
				_, err := credentialClient.Get(ctx, resourceGroup, automationAccountName, credentialName, nil)
				return err
			},
		},
		{
			name: "ListCredentials",
			run: func() error {
				return drainPager(ctx, credentialClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "CreateCertificate",
			run: func() error {
				_, err := certificateClient.CreateOrUpdate(ctx, resourceGroup, automationAccountName, certificateName, armautomation.CertificateCreateOrUpdateParameters{
					Name: to.Ptr(certificateName),
					Properties: &armautomation.CertificateCreateOrUpdateProperties{
						Base64Value:  to.Ptr("c3RhY2t5YXJk"),
						Description:  to.Ptr("automation example certificate"),
						IsExportable: to.Ptr(true),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetCertificate",
			run: func() error {
				_, err := certificateClient.Get(ctx, resourceGroup, automationAccountName, certificateName, nil)
				return err
			},
		},
		{
			name: "ListCertificates",
			run: func() error {
				return drainPager(ctx, certificateClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "CreateRunbook",
			run: func() error {
				_, err := runbookClient.CreateOrUpdate(ctx, resourceGroup, automationAccountName, runbookName, armautomation.RunbookCreateOrUpdateParameters{
					Name:     to.Ptr(runbookName),
					Location: to.Ptr(region),
					Properties: &armautomation.RunbookCreateOrUpdateProperties{
						RunbookType: to.Ptr(armautomation.RunbookTypeEnumPowerShellWorkflow),
						LogVerbose:  to.Ptr(true),
						Draft: &armautomation.RunbookDraft{
							DraftContentLink: &armautomation.ContentLink{
								URI: to.Ptr("https://example.invalid/runbook.ps1"),
							},
						},
						PublishContentLink: &armautomation.ContentLink{
							URI: to.Ptr("https://example.invalid/runbook.ps1"),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetRunbook",
			run: func() error {
				_, err := runbookClient.Get(ctx, resourceGroup, automationAccountName, runbookName, nil)
				return err
			},
		},
		{
			name: "GetRunbookContent",
			run: func() error {
				_, err := runbookClient.GetContent(ctx, resourceGroup, automationAccountName, runbookName, nil)
				return err
			},
		},
		{
			name: "ListRunbooks",
			run: func() error {
				return drainPager(ctx, runbookClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "GetRunbookDraft",
			run: func() error {
				_, err := runbookDraftClient.Get(ctx, resourceGroup, automationAccountName, runbookName, nil)
				return err
			},
		},
		{
			name: "GetRunbookDraftContent",
			run: func() error {
				_, err := runbookDraftClient.GetContent(ctx, resourceGroup, automationAccountName, runbookName, nil)
				return err
			},
		},
		{
			name: "UndoRunbookDraftEdit",
			run: func() error {
				_, err := runbookDraftClient.UndoEdit(ctx, resourceGroup, automationAccountName, runbookName, nil)
				return err
			},
		},
		{
			name: "CreateSchedule",
			run: func() error {
				_, err := scheduleClient.CreateOrUpdate(ctx, resourceGroup, automationAccountName, scheduleName, armautomation.ScheduleCreateOrUpdateParameters{
					Name: to.Ptr(scheduleName),
					Properties: &armautomation.ScheduleCreateOrUpdateProperties{
						Frequency: to.Ptr(armautomation.ScheduleFrequencyHour),
						StartTime: &startTime,
						TimeZone:  to.Ptr("UTC"),
						Interval:  int64(1),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetSchedule",
			run: func() error {
				_, err := scheduleClient.Get(ctx, resourceGroup, automationAccountName, scheduleName, nil)
				return err
			},
		},
		{
			name: "ListSchedules",
			run: func() error {
				return drainPager(ctx, scheduleClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "CreateJobSchedule",
			run: func() error {
				_, err := jobScheduleClient.Create(ctx, resourceGroup, automationAccountName, jobScheduleID, armautomation.JobScheduleCreateParameters{
					Properties: &armautomation.JobScheduleCreateProperties{
						Runbook: &armautomation.RunbookAssociationProperty{Name: to.Ptr(runbookName)},
						Schedule: &armautomation.ScheduleAssociationProperty{
							Name: to.Ptr(scheduleName),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetJobSchedule",
			run: func() error {
				_, err := jobScheduleClient.Get(ctx, resourceGroup, automationAccountName, jobScheduleID, nil)
				return err
			},
		},
		{
			name: "ListJobSchedules",
			run: func() error {
				return drainPager(ctx, jobScheduleClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "GenerateWebhookURI",
			run: func() error {
				resp, err := webhookClient.GenerateURI(ctx, resourceGroup, automationAccountName, nil)
				if err != nil {
					return err
				}
				generatedWebhookURI = resp.Value
				if generatedWebhookURI == nil || *generatedWebhookURI == "" {
					return errors.New("generated webhook uri was empty")
				}
				return nil
			},
		},
		{
			name: "CreateWebhook",
			run: func() error {
				_, err := webhookClient.CreateOrUpdate(ctx, resourceGroup, automationAccountName, webhookName, armautomation.WebhookCreateOrUpdateParameters{
					Name: to.Ptr(webhookName),
					Properties: &armautomation.WebhookCreateOrUpdateProperties{
						Runbook: &armautomation.RunbookAssociationProperty{Name: to.Ptr(runbookName)},
						IsEnabled: to.Ptr(true),
						ExpiryTime: &expiry,
						URI: generatedWebhookURI,
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetWebhook",
			run: func() error {
				_, err := webhookClient.Get(ctx, resourceGroup, automationAccountName, webhookName, nil)
				return err
			},
		},
		{
			name: "ListWebhooks",
			run: func() error {
				return drainPager(ctx, webhookClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "CreateJob",
			run: func() error {
				_, err := jobClient.Create(ctx, resourceGroup, automationAccountName, jobName, armautomation.JobCreateParameters{
					Properties: &armautomation.JobCreateProperties{
						Runbook: &armautomation.RunbookAssociationProperty{Name: to.Ptr(runbookName)},
						Parameters: map[string]*string{
							"message": to.Ptr("hello from stackyard"),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetJob",
			run: func() error {
				_, err := jobClient.Get(ctx, resourceGroup, automationAccountName, jobName, nil)
				return err
			},
		},
		{
			name: "ListJobs",
			run: func() error {
				return drainPager(ctx, jobClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "GetJobOutput",
			run: func() error {
				_, err := jobClient.GetOutput(ctx, resourceGroup, automationAccountName, jobName, nil)
				return err
			},
		},
		{
			name: "GetJobRunbookContent",
			run: func() error {
				_, err := jobClient.GetRunbookContent(ctx, resourceGroup, automationAccountName, jobName, nil)
				return err
			},
		},
		{
			name: "ListJobStreams",
			run: func() error {
				pager := jobStreamClient.NewListByJobPager(resourceGroup, automationAccountName, jobName, nil)
				page, err := pager.NextPage(ctx)
				if err != nil {
					return err
				}
				if len(page.Value) == 0 || page.Value[0] == nil || page.Value[0].Properties == nil || page.Value[0].Properties.JobStreamID == nil {
					return errors.New("job stream list was empty")
				}
				firstJobStreamID = *page.Value[0].Properties.JobStreamID
				return nil
			},
		},
		{
			name: "GetJobStream",
			run: func() error {
				_, err := jobStreamClient.Get(ctx, resourceGroup, automationAccountName, jobName, firstJobStreamID, nil)
				return err
			},
		},
		{
			name: "SuspendJob",
			run: func() error {
				_, err := jobClient.Suspend(ctx, resourceGroup, automationAccountName, jobName, nil)
				return err
			},
		},
		{
			name: "ResumeJob",
			run: func() error {
				_, err := jobClient.Resume(ctx, resourceGroup, automationAccountName, jobName, nil)
				return err
			},
		},
		{
			name: "StopJob",
			run: func() error {
				_, err := jobClient.Stop(ctx, resourceGroup, automationAccountName, jobName, nil)
				return err
			},
		},
		{
			name: "CreateSourceControl",
			run: func() error {
				_, err := sourceControlClient.CreateOrUpdate(ctx, resourceGroup, automationAccountName, sourceControlName, armautomation.SourceControlCreateOrUpdateParameters{
					Properties: &armautomation.SourceControlCreateOrUpdateProperties{
						RepoURL:        to.Ptr("https://github.com/stackyard/example"),
						Branch:         to.Ptr("main"),
						FolderPath:     to.Ptr("/"),
						AutoSync:       to.Ptr(true),
						PublishRunbook: to.Ptr(true),
						SourceType:     to.Ptr(armautomation.SourceTypeGitHub),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetSourceControl",
			run: func() error {
				_, err := sourceControlClient.Get(ctx, resourceGroup, automationAccountName, sourceControlName, nil)
				return err
			},
		},
		{
			name: "ListSourceControls",
			run: func() error {
				return drainPager(ctx, sourceControlClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "CreateSourceControlSyncJob",
			run: func() error {
				_, err := sourceControlSyncJobClient.Create(ctx, resourceGroup, automationAccountName, sourceControlName, syncJobID, armautomation.SourceControlSyncJobCreateParameters{
					Properties: &armautomation.SourceControlSyncJobCreateProperties{
						CommitID: to.Ptr(""),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetSourceControlSyncJob",
			run: func() error {
				_, err := sourceControlSyncJobClient.Get(ctx, resourceGroup, automationAccountName, sourceControlName, syncJobID, nil)
				return err
			},
		},
		{
			name: "ListSourceControlSyncJobs",
			run: func() error {
				return drainPager(ctx, sourceControlSyncJobClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, sourceControlName, nil))
			},
		},
		{
			name: "ListSourceControlSyncJobStreams",
			run: func() error {
				pager := sourceControlSyncJobStreamsClient.NewListBySyncJobPager(resourceGroup, automationAccountName, sourceControlName, syncJobID, nil)
				page, err := pager.NextPage(ctx)
				if err != nil {
					return err
				}
				if len(page.Value) == 0 || page.Value[0] == nil || page.Value[0].Properties == nil || page.Value[0].Properties.SourceControlSyncJobStreamID == nil {
					return errors.New("source control sync job stream list was empty")
				}
				firstSyncJobStreamID = *page.Value[0].Properties.SourceControlSyncJobStreamID
				return nil
			},
		},
		{
			name: "GetSourceControlSyncJobStream",
			run: func() error {
				_, err := sourceControlSyncJobStreamsClient.Get(ctx, resourceGroup, automationAccountName, sourceControlName, syncJobID, firstSyncJobStreamID, nil)
				return err
			},
		},
		{
			name: "ListStatistics",
			run: func() error {
				return drainPager(ctx, statisticsClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "ListUsages",
			run: func() error {
				return drainPager(ctx, usagesClient.NewListByAutomationAccountPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "ListPrivateLinkResources",
			run: func() error {
				return drainPager(ctx, privateLinkResourcesClient.NewAutomationPager(resourceGroup, automationAccountName, nil))
			},
		},
		{
			name: "DeleteWebhook",
			run: func() error {
				_, err := webhookClient.Delete(ctx, resourceGroup, automationAccountName, webhookName, nil)
				return err
			},
		},
		{
			name: "DeleteJobSchedule",
			run: func() error {
				_, err := jobScheduleClient.Delete(ctx, resourceGroup, automationAccountName, jobScheduleID, nil)
				return err
			},
		},
		{
			name: "DeleteSchedule",
			run: func() error {
				_, err := scheduleClient.Delete(ctx, resourceGroup, automationAccountName, scheduleName, nil)
				return err
			},
		},
		{
			name: "DeleteRunbook",
			run: func() error {
				_, err := runbookClient.Delete(ctx, resourceGroup, automationAccountName, runbookName, nil)
				return err
			},
		},
		{
			name: "DeleteCertificate",
			run: func() error {
				_, err := certificateClient.Delete(ctx, resourceGroup, automationAccountName, certificateName, nil)
				return err
			},
		},
		{
			name: "DeleteCredential",
			run: func() error {
				_, err := credentialClient.Delete(ctx, resourceGroup, automationAccountName, credentialName, nil)
				return err
			},
		},
		{
			name: "DeleteVariable",
			run: func() error {
				_, err := variableClient.Delete(ctx, resourceGroup, automationAccountName, variableName, nil)
				return err
			},
		},
		{
			name: "DeleteSourceControl",
			run: func() error {
				_, err := sourceControlClient.Delete(ctx, resourceGroup, automationAccountName, sourceControlName, nil)
				return err
			},
		},
		{
			name: "DeleteAutomationAccount",
			run: func() error {
				_, err := accountClient.Delete(ctx, resourceGroup, automationAccountName, nil)
				return err
			},
		},
	}

	for _, call := range calls {
		if err := call.run(); err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: ok\n", call.name)
	}
}

func drainPager[T any](ctx context.Context, pager *runtime.Pager[T]) error {
	for pager.More() {
		if _, err := pager.NextPage(ctx); err != nil {
			return err
		}
	}
	return nil
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
