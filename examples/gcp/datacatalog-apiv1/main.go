package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	datacatalog "cloud.google.com/go/datacatalog/apiv1"
	"cloud.google.com/go/datacatalog/apiv1/datacatalogpb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *datacatalog.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	entryGroupID := getenv("STACKYARD_GCP_DATACATALOG_ENTRY_GROUP_ID", "team_group")
	entryID := getenv("STACKYARD_GCP_DATACATALOG_ENTRY_ID", "orders")
	tagTemplateID := getenv("STACKYARD_GCP_DATACATALOG_TAG_TEMPLATE_ID", "dataset_template")
	tagTemplateFieldID := getenv("STACKYARD_GCP_DATACATALOG_TAG_TEMPLATE_FIELD_ID", "domain")
	renamedTagTemplateFieldID := getenv("STACKYARD_GCP_DATACATALOG_RENAMED_FIELD_ID", "domain_v2")
	tagID := getenv("STACKYARD_GCP_DATACATALOG_TAG_ID", "tag-1")
	organizationID := getenv("STACKYARD_GCP_DATACATALOG_ORGANIZATION_ID", "123456789")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	entryGroupName := locationName + "/entryGroups/" + entryGroupID
	entryName := entryGroupName + "/entries/" + entryID
	tagTemplateName := locationName + "/tagTemplates/" + tagTemplateID
	tagTemplateFieldName := tagTemplateName + "/fields/" + tagTemplateFieldID
	renamedTagTemplateFieldName := tagTemplateName + "/fields/" + renamedTagTemplateFieldID
	tagName := entryName + "/tags/" + tagID
	organizationName := "organizations/" + organizationID

	fmt.Printf("Stackyard GCP Data Catalog apiv1 client using %s\n", apiEndpoint)

	client, err := datacatalog.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create datacatalog client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "SearchCatalog",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				it := c.SearchCatalog(ctx, &datacatalogpb.SearchCatalogRequest{
					Scope: &datacatalogpb.SearchCatalogRequest_Scope{
						IncludeProjectIds: []string{projectID},
					},
					Query:    "orders",
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "ListEntryGroups",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				it := c.ListEntryGroups(ctx, &datacatalogpb.ListEntryGroupsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetEntryGroup",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.GetEntryGroup(ctx, &datacatalogpb.GetEntryGroupRequest{Name: entryGroupName})
				return err
			},
		},
		{
			name: "CreateEntryGroup",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.CreateEntryGroup(ctx, &datacatalogpb.CreateEntryGroupRequest{
					Parent:       locationName,
					EntryGroupId: entryGroupID,
					EntryGroup: &datacatalogpb.EntryGroup{
						DisplayName: "Team Group",
					},
				})
				return err
			},
		},
		{
			name: "UpdateEntryGroup",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.UpdateEntryGroup(ctx, &datacatalogpb.UpdateEntryGroupRequest{
					EntryGroup: &datacatalogpb.EntryGroup{
						Name:        entryGroupName,
						DisplayName: "Team Group Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "ListEntries",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				it := c.ListEntries(ctx, &datacatalogpb.ListEntriesRequest{
					Parent:   entryGroupName,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetEntry",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.GetEntry(ctx, &datacatalogpb.GetEntryRequest{Name: entryName})
				return err
			},
		},
		{
			name: "CreateEntry",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.CreateEntry(ctx, &datacatalogpb.CreateEntryRequest{
					Parent:  entryGroupName,
					EntryId: entryID,
					Entry: &datacatalogpb.Entry{
						DisplayName:        "Orders Entry",
						LinkedResource:     "//stackyard.local/orders",
						EntryType:          &datacatalogpb.Entry_UserSpecifiedType{UserSpecifiedType: "orders_dataset"},
						System:             &datacatalogpb.Entry_UserSpecifiedSystem{UserSpecifiedSystem: "stackyard"},
						Description:        "Orders dataset metadata",
						FullyQualifiedName: "stackyard.orders",
					},
				})
				return err
			},
		},
		{
			name: "UpdateEntry",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.UpdateEntry(ctx, &datacatalogpb.UpdateEntryRequest{
					Entry: &datacatalogpb.Entry{
						Name:               entryName,
						DisplayName:        "Orders Entry Updated",
						LinkedResource:     "//stackyard.local/orders",
						EntryType:          &datacatalogpb.Entry_UserSpecifiedType{UserSpecifiedType: "orders_dataset"},
						System:             &datacatalogpb.Entry_UserSpecifiedSystem{UserSpecifiedSystem: "stackyard"},
						Description:        "Updated metadata",
						FullyQualifiedName: "stackyard.orders",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "LookupEntry",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.LookupEntry(ctx, &datacatalogpb.LookupEntryRequest{
					TargetName: &datacatalogpb.LookupEntryRequest_FullyQualifiedName{
						FullyQualifiedName: "stackyard.orders",
					},
					Project:  projectID,
					Location: locationID,
				})
				return err
			},
		},
		{
			name: "ModifyEntryOverview",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.ModifyEntryOverview(ctx, &datacatalogpb.ModifyEntryOverviewRequest{
					Name:          entryName,
					EntryOverview: &datacatalogpb.EntryOverview{Overview: "<p>Orders metadata overview</p>"},
				})
				return err
			},
		},
		{
			name: "ModifyEntryContacts",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.ModifyEntryContacts(ctx, &datacatalogpb.ModifyEntryContactsRequest{
					Name: entryName,
					Contacts: &datacatalogpb.Contacts{
						People: []*datacatalogpb.Contacts_Person{
							{Designation: "Data Steward", Email: "owner@example.com"},
						},
					},
				})
				return err
			},
		},
		{
			name: "CreateTagTemplate",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.CreateTagTemplate(ctx, &datacatalogpb.CreateTagTemplateRequest{
					Parent:        locationName,
					TagTemplateId: tagTemplateID,
					TagTemplate: &datacatalogpb.TagTemplate{
						DisplayName: "Dataset Template",
						Fields: map[string]*datacatalogpb.TagTemplateField{
							"owner": {
								DisplayName: "Owner",
								Type: &datacatalogpb.FieldType{
									TypeDecl: &datacatalogpb.FieldType_PrimitiveType_{
										PrimitiveType: datacatalogpb.FieldType_STRING,
									},
								},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetTagTemplate",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.GetTagTemplate(ctx, &datacatalogpb.GetTagTemplateRequest{Name: tagTemplateName})
				return err
			},
		},
		{
			name: "UpdateTagTemplate",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.UpdateTagTemplate(ctx, &datacatalogpb.UpdateTagTemplateRequest{
					TagTemplate: &datacatalogpb.TagTemplate{
						Name:        tagTemplateName,
						DisplayName: "Dataset Template Updated",
						Fields: map[string]*datacatalogpb.TagTemplateField{
							"owner": {
								DisplayName: "Owner",
								Type: &datacatalogpb.FieldType{
									TypeDecl: &datacatalogpb.FieldType_PrimitiveType_{
										PrimitiveType: datacatalogpb.FieldType_STRING,
									},
								},
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "CreateTagTemplateField",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.CreateTagTemplateField(ctx, &datacatalogpb.CreateTagTemplateFieldRequest{
					Parent:             tagTemplateName,
					TagTemplateFieldId: tagTemplateFieldID,
					TagTemplateField: &datacatalogpb.TagTemplateField{
						DisplayName: "Domain",
						Type: &datacatalogpb.FieldType{
							TypeDecl: &datacatalogpb.FieldType_PrimitiveType_{
								PrimitiveType: datacatalogpb.FieldType_STRING,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateTagTemplateField",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.UpdateTagTemplateField(ctx, &datacatalogpb.UpdateTagTemplateFieldRequest{
					Name: tagTemplateFieldName,
					TagTemplateField: &datacatalogpb.TagTemplateField{
						DisplayName: "Domain Updated",
						Type: &datacatalogpb.FieldType{
							TypeDecl: &datacatalogpb.FieldType_PrimitiveType_{
								PrimitiveType: datacatalogpb.FieldType_STRING,
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "RenameTagTemplateField",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.RenameTagTemplateField(ctx, &datacatalogpb.RenameTagTemplateFieldRequest{
					Name:                  tagTemplateFieldName,
					NewTagTemplateFieldId: renamedTagTemplateFieldID,
				})
				return err
			},
		},
		{
			name: "CreateTag",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.CreateTag(ctx, &datacatalogpb.CreateTagRequest{
					Parent: entryName,
					Tag: &datacatalogpb.Tag{
						Template: tagTemplateName,
						Fields: map[string]*datacatalogpb.TagField{
							"owner": {
								Kind: &datacatalogpb.TagField_StringValue{StringValue: "team-a"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "ListTags",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				it := c.ListTags(ctx, &datacatalogpb.ListTagsRequest{
					Parent:   entryName,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "UpdateTag",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.UpdateTag(ctx, &datacatalogpb.UpdateTagRequest{
					Tag: &datacatalogpb.Tag{
						Name:     tagName,
						Template: tagTemplateName,
						Fields: map[string]*datacatalogpb.TagField{
							"owner": {
								Kind: &datacatalogpb.TagField_StringValue{StringValue: "team-b"},
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"fields"}},
				})
				return err
			},
		},
		{
			name: "ReconcileTags",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.ReconcileTags(ctx, &datacatalogpb.ReconcileTagsRequest{
					Parent:      entryName,
					TagTemplate: tagTemplateName,
					Tags: []*datacatalogpb.Tag{
						{
							Template: tagTemplateName,
							Fields: map[string]*datacatalogpb.TagField{
								"owner": {
									Kind: &datacatalogpb.TagField_StringValue{StringValue: "team-b"},
								},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "StarEntry",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.StarEntry(ctx, &datacatalogpb.StarEntryRequest{Name: entryName})
				return err
			},
		},
		{
			name: "UnstarEntry",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.UnstarEntry(ctx, &datacatalogpb.UnstarEntryRequest{Name: entryName})
				return err
			},
		},
		{
			name: "GetIAMPolicy",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: entryGroupName})
				return err
			},
		},
		{
			name: "SetIAMPolicy",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: entryGroupName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "TestIAMPermissions",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    entryGroupName,
					Permissions: []string{"datacatalog.entries.get"},
				})
				return err
			},
		},
		{
			name: "ImportEntries",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.ImportEntries(ctx, &datacatalogpb.ImportEntriesRequest{
					Parent: entryGroupName,
					Source: &datacatalogpb.ImportEntriesRequest_GcsBucketPath{
						GcsBucketPath: "gs://stackyard-imports/orders",
					},
				})
				return err
			},
		},
		{
			name: "SetConfig",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.SetConfig(ctx, &datacatalogpb.SetConfigRequest{
					Name: organizationName,
					Configuration: &datacatalogpb.SetConfigRequest_TagTemplateMigration{
						TagTemplateMigration: datacatalogpb.TagTemplateMigration_TAG_TEMPLATE_MIGRATION_ENABLED,
					},
				})
				return err
			},
		},
		{
			name: "RetrieveConfig",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.RetrieveConfig(ctx, &datacatalogpb.RetrieveConfigRequest{Name: organizationName})
				return err
			},
		},
		{
			name: "RetrieveEffectiveConfig",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				_, err := c.RetrieveEffectiveConfig(ctx, &datacatalogpb.RetrieveEffectiveConfigRequest{Name: "projects/" + projectID})
				return err
			},
		},
		{
			name: "DeleteTag",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				return c.DeleteTag(ctx, &datacatalogpb.DeleteTagRequest{Name: tagName})
			},
		},
		{
			name: "DeleteTagTemplateField",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				return c.DeleteTagTemplateField(ctx, &datacatalogpb.DeleteTagTemplateFieldRequest{
					Name:  renamedTagTemplateFieldName,
					Force: true,
				})
			},
		},
		{
			name: "DeleteEntry",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				return c.DeleteEntry(ctx, &datacatalogpb.DeleteEntryRequest{Name: entryName})
			},
		},
		{
			name: "DeleteTagTemplate",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				return c.DeleteTagTemplate(ctx, &datacatalogpb.DeleteTagTemplateRequest{
					Name:  tagTemplateName,
					Force: true,
				})
			},
		},
		{
			name: "DeleteEntryGroup",
			call: func(ctx context.Context, c *datacatalog.Client) error {
				return c.DeleteEntryGroup(ctx, &datacatalogpb.DeleteEntryGroupRequest{
					Name:  entryGroupName,
					Force: true,
				})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close datacatalog client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
