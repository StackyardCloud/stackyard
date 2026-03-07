package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	folderParent := getenv("STACKYARD_GCP_FOLDER_PARENT", "organizations/123456")
	folderName := getenv("STACKYARD_GCP_FOLDER_NAME", "folders/1001")
	destinationParent := getenv("STACKYARD_GCP_DESTINATION_PARENT", "folders/2000")
	projectName := getenv("STACKYARD_GCP_PROJECT_NAME", "projects/415104041262")
	projectParent := getenv("STACKYARD_GCP_PROJECT_PARENT", "organizations/123456")
	tagKeyName := getenv("STACKYARD_GCP_TAG_KEY_NAME", "tagKeys/2001")
	tagValueName := getenv("STACKYARD_GCP_TAG_VALUE_NAME", "tagValues/3001")
	tagBindingParent := getenv("STACKYARD_GCP_TAG_BINDING_PARENT", "//cloudresourcemanager.googleapis.com/projects/415104041262")
	tagKeyNamespaced := getenv("STACKYARD_GCP_TAG_KEY_NAMESPACED", "123456/env")
	tagValueNamespaced := getenv("STACKYARD_GCP_TAG_VALUE_NAMESPACED", "123456/env/prod")

	fmt.Printf("Stackyard GCP Resource Manager apiv3 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "resourcemanager",
		},
	}

	clientOpts := []option.ClientOption{
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	}

	foldersClient, err := resourcemanager.NewFoldersRESTClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create folders client: %v", err)
	}
	defer closeClient("folders", foldersClient.Close)

	projectsClient, err := resourcemanager.NewProjectsRESTClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create projects client: %v", err)
	}
	defer closeClient("projects", projectsClient.Close)

	organizationsClient, err := resourcemanager.NewOrganizationsRESTClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create organizations client: %v", err)
	}
	defer closeClient("organizations", organizationsClient.Close)

	tagKeysClient, err := resourcemanager.NewTagKeysRESTClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create tagKeys client: %v", err)
	}
	defer closeClient("tagKeys", tagKeysClient.Close)

	tagValuesClient, err := resourcemanager.NewTagValuesRESTClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create tagValues client: %v", err)
	}
	defer closeClient("tagValues", tagValuesClient.Close)

	tagBindingsClient, err := resourcemanager.NewTagBindingsRESTClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create tagBindings client: %v", err)
	}
	defer closeClient("tagBindings", tagBindingsClient.Close)

	tagHoldsClient, err := resourcemanager.NewTagHoldsRESTClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create tagHolds client: %v", err)
	}
	defer closeClient("tagHolds", tagHoldsClient.Close)

	folderOpName := "operations/create-folder-1001"
	projectOpName := "operations/create-project-stackyard-prod"
	tagKeyOpName := "operations/create-tagkey-2001"
	tagValueOpName := "operations/create-tagvalue-3001"
	tagBindingOpName := "operations/create-tagbinding-3001"
	tagHoldOpName := "operations/create-taghold-3001"

	calls := []callSpec{
		{
			name: "Folders.ListFolders",
			call: func(ctx context.Context) error {
				it := foldersClient.ListFolders(ctx, &resourcemanagerpb.ListFoldersRequest{Parent: folderParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "Folders.SearchFolders",
			call: func(ctx context.Context) error {
				it := foldersClient.SearchFolders(ctx, &resourcemanagerpb.SearchFoldersRequest{Query: "lifecyclestate=active", PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "Folders.GetFolder",
			call: func(ctx context.Context) error {
				_, err := foldersClient.GetFolder(ctx, &resourcemanagerpb.GetFolderRequest{Name: folderName})
				return err
			},
		},
		{
			name: "Folders.CreateFolder",
			call: func(ctx context.Context) error {
				op, err := foldersClient.CreateFolder(ctx, &resourcemanagerpb.CreateFolderRequest{
					Folder: &resourcemanagerpb.Folder{
						Parent:      folderParent,
						DisplayName: "Team Folder",
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					folderOpName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "Folders.UpdateFolder",
			call: func(ctx context.Context) error {
				op, err := foldersClient.UpdateFolder(ctx, &resourcemanagerpb.UpdateFolderRequest{
					Folder:     &resourcemanagerpb.Folder{Name: folderName, DisplayName: "Team Folder Updated"},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				if err != nil {
					return err
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "Folders.MoveFolder",
			call: func(ctx context.Context) error {
				op, err := foldersClient.MoveFolder(ctx, &resourcemanagerpb.MoveFolderRequest{Name: folderName, DestinationParent: destinationParent})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					folderOpName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "Folders.DeleteFolder",
			call: func(ctx context.Context) error {
				op, err := foldersClient.DeleteFolder(ctx, &resourcemanagerpb.DeleteFolderRequest{Name: folderName})
				if err != nil {
					return err
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "Folders.UndeleteFolder",
			call: func(ctx context.Context) error {
				op, err := foldersClient.UndeleteFolder(ctx, &resourcemanagerpb.UndeleteFolderRequest{Name: folderName})
				if err != nil {
					return err
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "Folders.GetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := foldersClient.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: folderName})
				return err
			},
		},
		{
			name: "Folders.SetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := foldersClient.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: folderName,
					Policy:   &iampb.Policy{Bindings: []*iampb.Binding{{Role: "roles/viewer", Members: []string{"user:bob@example.com"}}}},
				})
				return err
			},
		},
		{
			name: "Folders.TestIamPermissions",
			call: func(ctx context.Context) error {
				_, err := foldersClient.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{Resource: folderName, Permissions: []string{"resourcemanager.folders.get"}})
				return err
			},
		},
		{
			name: "Folders.PollCreateFolderOperation",
			call: func(ctx context.Context) error {
				_, err := foldersClient.CreateFolderOperation(folderOpName).Poll(ctx)
				return err
			},
		},
		{
			name: "Projects.ListProjects",
			call: func(ctx context.Context) error {
				it := projectsClient.ListProjects(ctx, &resourcemanagerpb.ListProjectsRequest{Parent: projectParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "Projects.SearchProjects",
			call: func(ctx context.Context) error {
				it := projectsClient.SearchProjects(ctx, &resourcemanagerpb.SearchProjectsRequest{Query: "state=active", PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "Projects.GetProject",
			call: func(ctx context.Context) error {
				_, err := projectsClient.GetProject(ctx, &resourcemanagerpb.GetProjectRequest{Name: projectName})
				return err
			},
		},
		{
			name: "Projects.CreateProject",
			call: func(ctx context.Context) error {
				op, err := projectsClient.CreateProject(ctx, &resourcemanagerpb.CreateProjectRequest{Project: &resourcemanagerpb.Project{
					ProjectId:   "stackyard-prod",
					DisplayName: "Stackyard Prod",
					Parent:      projectParent,
				}})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					projectOpName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "Projects.MoveProject",
			call: func(ctx context.Context) error {
				op, err := projectsClient.MoveProject(ctx, &resourcemanagerpb.MoveProjectRequest{Name: projectName, DestinationParent: destinationParent})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					projectOpName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "Projects.PollCreateProjectOperation",
			call: func(ctx context.Context) error {
				_, err := projectsClient.CreateProjectOperation(projectOpName).Poll(ctx)
				return err
			},
		},
		{
			name: "Organizations.SearchOrganizations",
			call: func(ctx context.Context) error {
				it := organizationsClient.SearchOrganizations(ctx, &resourcemanagerpb.SearchOrganizationsRequest{Query: "domain:example.com", PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "Organizations.GetOrganization",
			call: func(ctx context.Context) error {
				_, err := organizationsClient.GetOrganization(ctx, &resourcemanagerpb.GetOrganizationRequest{Name: "organizations/123456"})
				return err
			},
		},
		{
			name: "TagKeys.ListTagKeys",
			call: func(ctx context.Context) error {
				it := tagKeysClient.ListTagKeys(ctx, &resourcemanagerpb.ListTagKeysRequest{Parent: projectParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "TagKeys.GetTagKey",
			call: func(ctx context.Context) error {
				_, err := tagKeysClient.GetTagKey(ctx, &resourcemanagerpb.GetTagKeyRequest{Name: tagKeyName})
				return err
			},
		},
		{
			name: "TagKeys.GetNamespacedTagKey",
			call: func(ctx context.Context) error {
				_, err := tagKeysClient.GetNamespacedTagKey(ctx, &resourcemanagerpb.GetNamespacedTagKeyRequest{Name: tagKeyNamespaced})
				return err
			},
		},
		{
			name: "TagKeys.CreateTagKey",
			call: func(ctx context.Context) error {
				op, err := tagKeysClient.CreateTagKey(ctx, &resourcemanagerpb.CreateTagKeyRequest{TagKey: &resourcemanagerpb.TagKey{Parent: projectParent, ShortName: "env", Description: "environment"}})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					tagKeyOpName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "TagKeys.PollCreateTagKeyOperation",
			call: func(ctx context.Context) error {
				_, err := tagKeysClient.CreateTagKeyOperation(tagKeyOpName).Poll(ctx)
				return err
			},
		},
		{
			name: "TagValues.ListTagValues",
			call: func(ctx context.Context) error {
				it := tagValuesClient.ListTagValues(ctx, &resourcemanagerpb.ListTagValuesRequest{Parent: tagKeyName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "TagValues.GetTagValue",
			call: func(ctx context.Context) error {
				_, err := tagValuesClient.GetTagValue(ctx, &resourcemanagerpb.GetTagValueRequest{Name: tagValueName})
				return err
			},
		},
		{
			name: "TagValues.GetNamespacedTagValue",
			call: func(ctx context.Context) error {
				_, err := tagValuesClient.GetNamespacedTagValue(ctx, &resourcemanagerpb.GetNamespacedTagValueRequest{Name: tagValueNamespaced})
				return err
			},
		},
		{
			name: "TagValues.CreateTagValue",
			call: func(ctx context.Context) error {
				op, err := tagValuesClient.CreateTagValue(ctx, &resourcemanagerpb.CreateTagValueRequest{TagValue: &resourcemanagerpb.TagValue{Parent: tagKeyName, ShortName: "prod", Description: "production"}})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					tagValueOpName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "TagValues.PollCreateTagValueOperation",
			call: func(ctx context.Context) error {
				_, err := tagValuesClient.CreateTagValueOperation(tagValueOpName).Poll(ctx)
				return err
			},
		},
		{
			name: "TagBindings.ListTagBindings",
			call: func(ctx context.Context) error {
				it := tagBindingsClient.ListTagBindings(ctx, &resourcemanagerpb.ListTagBindingsRequest{Parent: tagBindingParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "TagBindings.ListEffectiveTags",
			call: func(ctx context.Context) error {
				it := tagBindingsClient.ListEffectiveTags(ctx, &resourcemanagerpb.ListEffectiveTagsRequest{Parent: tagBindingParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "TagBindings.CreateTagBinding",
			call: func(ctx context.Context) error {
				op, err := tagBindingsClient.CreateTagBinding(ctx, &resourcemanagerpb.CreateTagBindingRequest{TagBinding: &resourcemanagerpb.TagBinding{Parent: tagBindingParent, TagValue: tagValueName}})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					tagBindingOpName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "TagBindings.PollCreateTagBindingOperation",
			call: func(ctx context.Context) error {
				_, err := tagBindingsClient.CreateTagBindingOperation(tagBindingOpName).Poll(ctx)
				return err
			},
		},
		{
			name: "TagHolds.ListTagHolds",
			call: func(ctx context.Context) error {
				it := tagHoldsClient.ListTagHolds(ctx, &resourcemanagerpb.ListTagHoldsRequest{Parent: tagValueName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "TagHolds.CreateTagHold",
			call: func(ctx context.Context) error {
				op, err := tagHoldsClient.CreateTagHold(ctx, &resourcemanagerpb.CreateTagHoldRequest{
					Parent: tagValueName,
					TagHold: &resourcemanagerpb.TagHold{
						Holder: "//cloudresourcemanager.googleapis.com/projects/415104041262",
						Origin: "stackyard",
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					tagHoldOpName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "TagHolds.PollCreateTagHoldOperation",
			call: func(ctx context.Context) error {
				_, err := tagHoldsClient.CreateTagHoldOperation(tagHoldOpName).Poll(ctx)
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
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

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	if strings.TrimSpace(clone.Header.Get("X-Stackyard-GCP-Service")) == "" {
		clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	}
	return base.RoundTrip(clone)
}
