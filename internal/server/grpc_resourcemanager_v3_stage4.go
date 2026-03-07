package server

import (
	"strconv"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	resourcemanagerv3pb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpResourceManagerV3ListFoldersMethod        = "/google.cloud.resourcemanager.v3.Folders/ListFolders"
	gcpResourceManagerV3SearchFoldersMethod      = "/google.cloud.resourcemanager.v3.Folders/SearchFolders"
	gcpResourceManagerV3GetFolderMethod          = "/google.cloud.resourcemanager.v3.Folders/GetFolder"
	gcpResourceManagerV3CreateFolderMethod       = "/google.cloud.resourcemanager.v3.Folders/CreateFolder"
	gcpResourceManagerV3UpdateFolderMethod       = "/google.cloud.resourcemanager.v3.Folders/UpdateFolder"
	gcpResourceManagerV3MoveFolderMethod         = "/google.cloud.resourcemanager.v3.Folders/MoveFolder"
	gcpResourceManagerV3DeleteFolderMethod       = "/google.cloud.resourcemanager.v3.Folders/DeleteFolder"
	gcpResourceManagerV3UndeleteFolderMethod     = "/google.cloud.resourcemanager.v3.Folders/UndeleteFolder"
	gcpResourceManagerV3FoldersGetIAMPolicy      = "/google.cloud.resourcemanager.v3.Folders/GetIamPolicy"
	gcpResourceManagerV3FoldersSetIAMPolicy      = "/google.cloud.resourcemanager.v3.Folders/SetIamPolicy"
	gcpResourceManagerV3FoldersTestIAMPermission = "/google.cloud.resourcemanager.v3.Folders/TestIamPermissions"

	gcpResourceManagerV3ListProjectsMethod        = "/google.cloud.resourcemanager.v3.Projects/ListProjects"
	gcpResourceManagerV3SearchProjectsMethod      = "/google.cloud.resourcemanager.v3.Projects/SearchProjects"
	gcpResourceManagerV3GetProjectMethod          = "/google.cloud.resourcemanager.v3.Projects/GetProject"
	gcpResourceManagerV3CreateProjectMethod       = "/google.cloud.resourcemanager.v3.Projects/CreateProject"
	gcpResourceManagerV3UpdateProjectMethod       = "/google.cloud.resourcemanager.v3.Projects/UpdateProject"
	gcpResourceManagerV3MoveProjectMethod         = "/google.cloud.resourcemanager.v3.Projects/MoveProject"
	gcpResourceManagerV3DeleteProjectMethod       = "/google.cloud.resourcemanager.v3.Projects/DeleteProject"
	gcpResourceManagerV3UndeleteProjectMethod     = "/google.cloud.resourcemanager.v3.Projects/UndeleteProject"
	gcpResourceManagerV3ProjectsGetIAMPolicy      = "/google.cloud.resourcemanager.v3.Projects/GetIamPolicy"
	gcpResourceManagerV3ProjectsSetIAMPolicy      = "/google.cloud.resourcemanager.v3.Projects/SetIamPolicy"
	gcpResourceManagerV3ProjectsTestIAMPermission = "/google.cloud.resourcemanager.v3.Projects/TestIamPermissions"

	gcpResourceManagerV3GetOrganizationMethod          = "/google.cloud.resourcemanager.v3.Organizations/GetOrganization"
	gcpResourceManagerV3SearchOrganizationsMethod      = "/google.cloud.resourcemanager.v3.Organizations/SearchOrganizations"
	gcpResourceManagerV3OrganizationsGetIAMPolicy      = "/google.cloud.resourcemanager.v3.Organizations/GetIamPolicy"
	gcpResourceManagerV3OrganizationsSetIAMPolicy      = "/google.cloud.resourcemanager.v3.Organizations/SetIamPolicy"
	gcpResourceManagerV3OrganizationsTestIAMPermission = "/google.cloud.resourcemanager.v3.Organizations/TestIamPermissions"

	gcpResourceManagerV3ListTagKeysMethod        = "/google.cloud.resourcemanager.v3.TagKeys/ListTagKeys"
	gcpResourceManagerV3GetTagKeyMethod          = "/google.cloud.resourcemanager.v3.TagKeys/GetTagKey"
	gcpResourceManagerV3GetNamespacedTagKey      = "/google.cloud.resourcemanager.v3.TagKeys/GetNamespacedTagKey"
	gcpResourceManagerV3CreateTagKeyMethod       = "/google.cloud.resourcemanager.v3.TagKeys/CreateTagKey"
	gcpResourceManagerV3UpdateTagKeyMethod       = "/google.cloud.resourcemanager.v3.TagKeys/UpdateTagKey"
	gcpResourceManagerV3DeleteTagKeyMethod       = "/google.cloud.resourcemanager.v3.TagKeys/DeleteTagKey"
	gcpResourceManagerV3TagKeysGetIAMPolicy      = "/google.cloud.resourcemanager.v3.TagKeys/GetIamPolicy"
	gcpResourceManagerV3TagKeysSetIAMPolicy      = "/google.cloud.resourcemanager.v3.TagKeys/SetIamPolicy"
	gcpResourceManagerV3TagKeysTestIAMPermission = "/google.cloud.resourcemanager.v3.TagKeys/TestIamPermissions"

	gcpResourceManagerV3ListTagValuesMethod        = "/google.cloud.resourcemanager.v3.TagValues/ListTagValues"
	gcpResourceManagerV3GetTagValueMethod          = "/google.cloud.resourcemanager.v3.TagValues/GetTagValue"
	gcpResourceManagerV3GetNamespacedTagValue      = "/google.cloud.resourcemanager.v3.TagValues/GetNamespacedTagValue"
	gcpResourceManagerV3CreateTagValueMethod       = "/google.cloud.resourcemanager.v3.TagValues/CreateTagValue"
	gcpResourceManagerV3UpdateTagValueMethod       = "/google.cloud.resourcemanager.v3.TagValues/UpdateTagValue"
	gcpResourceManagerV3DeleteTagValueMethod       = "/google.cloud.resourcemanager.v3.TagValues/DeleteTagValue"
	gcpResourceManagerV3TagValuesGetIAMPolicy      = "/google.cloud.resourcemanager.v3.TagValues/GetIamPolicy"
	gcpResourceManagerV3TagValuesSetIAMPolicy      = "/google.cloud.resourcemanager.v3.TagValues/SetIamPolicy"
	gcpResourceManagerV3TagValuesTestIAMPermission = "/google.cloud.resourcemanager.v3.TagValues/TestIamPermissions"

	gcpResourceManagerV3ListTagBindingsMethod  = "/google.cloud.resourcemanager.v3.TagBindings/ListTagBindings"
	gcpResourceManagerV3CreateTagBindingMethod = "/google.cloud.resourcemanager.v3.TagBindings/CreateTagBinding"
	gcpResourceManagerV3DeleteTagBindingMethod = "/google.cloud.resourcemanager.v3.TagBindings/DeleteTagBinding"
	gcpResourceManagerV3ListEffectiveTags      = "/google.cloud.resourcemanager.v3.TagBindings/ListEffectiveTags"

	gcpResourceManagerV3ListTagHoldsMethod  = "/google.cloud.resourcemanager.v3.TagHolds/ListTagHolds"
	gcpResourceManagerV3CreateTagHoldMethod = "/google.cloud.resourcemanager.v3.TagHolds/CreateTagHold"
	gcpResourceManagerV3DeleteTagHoldMethod = "/google.cloud.resourcemanager.v3.TagHolds/DeleteTagHold"
)

func gcpStage4GRPCResourceManagerV3ListFolders(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.ListFoldersRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPResourceManagerParent(parent) {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.Folder{
		gcpStage4ResourceManagerV3Folder("1001", parent, "Team Folder", resourcemanagerv3pb.Folder_ACTIVE),
		gcpStage4ResourceManagerV3Folder("1002", parent, "Archive Folder", resourcemanagerv3pb.Folder_DELETE_REQUESTED),
	}
	if !req.GetShowDeleted() {
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.ListFoldersResponse{Folders: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3SearchFolders(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.SearchFoldersRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetQuery()) == "" {
		return grpcInvalidArgument("query-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.Folder{
		gcpStage4ResourceManagerV3Folder("1001", "organizations/123456", "Team Folder", resourcemanagerv3pb.Folder_ACTIVE),
		gcpStage4ResourceManagerV3Folder("1002", "folders/1001", "Archive Folder", resourcemanagerv3pb.Folder_DELETE_REQUESTED),
	}
	lower := strings.ToLower(req.GetQuery())
	switch {
	case strings.Contains(lower, "lifecyclestate=active"):
		items = items[:1]
	case strings.Contains(lower, "lifecyclestate=delete_requested"):
		items = items[1:]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.SearchFoldersResponse{Folders: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3GetFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.GetFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "folders")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	state := resourcemanagerv3pb.Folder_ACTIVE
	if strings.Contains(strings.ToLower(folderID), "deleted") {
		state = resourcemanagerv3pb.Folder_DELETE_REQUESTED
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Folder(folderID, "organizations/123456", "Folder "+folderID, state))
}

func gcpStage4GRPCResourceManagerV3CreateFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.CreateFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetFolder() == nil {
		return grpcInvalidArgument("folder-required")
	}
	if !isGCPResourceManagerParent(strings.TrimSpace(req.GetFolder().GetParent())) {
		return grpcInvalidArgument("folder.parent-required")
	}
	if !isGCPResourceManagerDisplayName(strings.TrimSpace(req.GetFolder().GetDisplayName())) {
		return grpcInvalidArgument("folder.display_name-invalid")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("create-folder-1001", false))
}

func gcpStage4GRPCResourceManagerV3UpdateFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.UpdateFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetFolder() == nil {
		return grpcInvalidArgument("folder-required")
	}
	folderID, ok := parseGCPStage4ResourceManagerV3Name(req.GetFolder().GetName(), "folders")
	if !ok {
		return grpcInvalidArgument("folder.name-required")
	}
	if !isGCPResourceManagerDisplayName(strings.TrimSpace(req.GetFolder().GetDisplayName())) {
		return grpcInvalidArgument("folder.display_name-invalid")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	for _, path := range req.GetUpdateMask().GetPaths() {
		switch strings.TrimSpace(path) {
		case "display_name", "displayName":
		default:
			return grpcInvalidArgument("update_mask-invalid")
		}
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("update-folder-"+folderID, false))
}

func gcpStage4GRPCResourceManagerV3MoveFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.MoveFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "folders")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if !isGCPResourceManagerParent(strings.TrimSpace(req.GetDestinationParent())) {
		return grpcInvalidArgument("destination_parent-required")
	}
	if strings.TrimSpace(req.GetDestinationParent()) == "folders/"+folderID {
		return grpcFailedPrecondition("destination_parent-cannot-equal-folder")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("move-folder-"+folderID, false))
}

func gcpStage4GRPCResourceManagerV3DeleteFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.DeleteFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "folders")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("delete-folder-"+folderID, false))
}

func gcpStage4GRPCResourceManagerV3UndeleteFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.UndeleteFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "folders")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(folderID), "active") {
		return grpcFailedPrecondition("folder-not-delete-requested")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("undelete-folder-"+folderID, false))
}

func gcpStage4GRPCResourceManagerV3ListProjects(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.ListProjectsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPResourceManagerParent(parent) {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.Project{
		gcpStage4ResourceManagerV3Project("415104041262", parent, "stackyard-prod", "Stackyard Prod", resourcemanagerv3pb.Project_ACTIVE),
		gcpStage4ResourceManagerV3Project("415104041263", parent, "stackyard-archive", "Stackyard Archive", resourcemanagerv3pb.Project_DELETE_REQUESTED),
	}
	if !req.GetShowDeleted() {
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.ListProjectsResponse{Projects: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3SearchProjects(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.SearchProjectsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetQuery()) == "" {
		return grpcInvalidArgument("query-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.Project{
		gcpStage4ResourceManagerV3Project("415104041262", "organizations/123456", "stackyard-prod", "Stackyard Prod", resourcemanagerv3pb.Project_ACTIVE),
		gcpStage4ResourceManagerV3Project("415104041263", "folders/1001", "stackyard-archive", "Stackyard Archive", resourcemanagerv3pb.Project_DELETE_REQUESTED),
	}
	lower := strings.ToLower(req.GetQuery())
	switch {
	case strings.Contains(lower, "state=active"):
		items = items[:1]
	case strings.Contains(lower, "state=delete_requested"):
		items = items[1:]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.SearchProjectsResponse{Projects: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3GetProject(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.GetProjectRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	projectID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "projects")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	state := resourcemanagerv3pb.Project_ACTIVE
	if strings.Contains(strings.ToLower(projectID), "deleted") {
		state = resourcemanagerv3pb.Project_DELETE_REQUESTED
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Project(projectID, "organizations/123456", "stackyard-prod", "Project "+projectID, state))
}

func gcpStage4GRPCResourceManagerV3CreateProject(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.CreateProjectRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetProject() == nil {
		return grpcInvalidArgument("project-required")
	}
	projectID := strings.TrimSpace(req.GetProject().GetProjectId())
	if !isGCPResourceManagerV3ProjectID(projectID) {
		return grpcInvalidArgument("project.project_id-invalid")
	}
	displayName := strings.TrimSpace(req.GetProject().GetDisplayName())
	if !isGCPResourceManagerV3ProjectDisplayName(displayName) {
		return grpcInvalidArgument("project.display_name-invalid")
	}
	if parent := strings.TrimSpace(req.GetProject().GetParent()); parent != "" && !isGCPResourceManagerParent(parent) {
		return grpcInvalidArgument("project.parent-invalid")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("create-project-"+projectID, false))
}

func gcpStage4GRPCResourceManagerV3UpdateProject(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.UpdateProjectRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetProject() == nil {
		return grpcInvalidArgument("project-required")
	}
	projectID, ok := parseGCPStage4ResourceManagerV3Name(req.GetProject().GetName(), "projects")
	if !ok {
		return grpcInvalidArgument("project.name-required")
	}
	if !isGCPResourceManagerV3ProjectDisplayName(strings.TrimSpace(req.GetProject().GetDisplayName())) {
		return grpcInvalidArgument("project.display_name-invalid")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	for _, path := range req.GetUpdateMask().GetPaths() {
		switch strings.TrimSpace(path) {
		case "display_name", "displayName", "labels":
		default:
			return grpcInvalidArgument("update_mask-invalid")
		}
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("update-project-"+projectID, false))
}

func gcpStage4GRPCResourceManagerV3MoveProject(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.MoveProjectRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	projectID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "projects")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if !isGCPResourceManagerParent(strings.TrimSpace(req.GetDestinationParent())) {
		return grpcInvalidArgument("destination_parent-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("move-project-"+projectID, false))
}

func gcpStage4GRPCResourceManagerV3DeleteProject(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.DeleteProjectRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	projectID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "projects")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("delete-project-"+projectID, false))
}

func gcpStage4GRPCResourceManagerV3UndeleteProject(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.UndeleteProjectRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	projectID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "projects")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(projectID), "active") {
		return grpcFailedPrecondition("project-not-delete-requested")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("undelete-project-"+projectID, false))
}

func gcpStage4GRPCResourceManagerV3SearchOrganizations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.SearchOrganizationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetQuery()) == "" {
		return grpcInvalidArgument("query-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.Organization{
		gcpStage4ResourceManagerV3Organization("123456", "example.com"),
		gcpStage4ResourceManagerV3Organization("123457", "example.org"),
	}
	if strings.Contains(strings.ToLower(req.GetQuery()), "example.com") {
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.SearchOrganizationsResponse{Organizations: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3GetOrganization(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.GetOrganizationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "organizations")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Organization(orgID, "example.com"))
}

func gcpStage4GRPCResourceManagerV3ListTagKeys(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.ListTagKeysRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !isGCPResourceManagerV3TagKeyParent(strings.TrimSpace(req.GetParent())) {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.TagKey{
		gcpStage4ResourceManagerV3TagKey("2001", req.GetParent(), "env"),
		gcpStage4ResourceManagerV3TagKey("2002", req.GetParent(), "owner"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.ListTagKeysResponse{TagKeys: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3GetTagKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.GetTagKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	tagKeyID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "tagKeys")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3TagKey(tagKeyID, "organizations/123456", "env"))
}

func gcpStage4GRPCResourceManagerV3GetNamespacedTagKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.GetNamespacedTagKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return grpcInvalidArgument("name-required")
	}
	shortName := "env"
	if idx := strings.LastIndex(req.GetName(), "/"); idx >= 0 && idx+1 < len(req.GetName()) {
		shortName = req.GetName()[idx+1:]
	}
	tagKey := gcpStage4ResourceManagerV3TagKey("2001", "organizations/123456", shortName)
	tagKey.NamespacedName = req.GetName()
	return grpcProtoSuccess(tagKey)
}

func gcpStage4GRPCResourceManagerV3CreateTagKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.CreateTagKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetTagKey() == nil {
		return grpcInvalidArgument("tag_key-required")
	}
	if !isGCPResourceManagerV3TagKeyParent(strings.TrimSpace(req.GetTagKey().GetParent())) {
		return grpcInvalidArgument("tag_key.parent-invalid")
	}
	if !isGCPResourceManagerV3TagShortName(strings.TrimSpace(req.GetTagKey().GetShortName())) {
		return grpcInvalidArgument("tag_key.short_name-invalid")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("create-tagkey-2001", false))
}

func gcpStage4GRPCResourceManagerV3UpdateTagKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.UpdateTagKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetTagKey() == nil {
		return grpcInvalidArgument("tag_key-required")
	}
	tagKeyID, ok := parseGCPStage4ResourceManagerV3Name(req.GetTagKey().GetName(), "tagKeys")
	if !ok {
		return grpcInvalidArgument("tag_key.name-required")
	}
	if strings.TrimSpace(req.GetTagKey().GetDescription()) == "" {
		return grpcInvalidArgument("tag_key.description-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	for _, path := range req.GetUpdateMask().GetPaths() {
		switch strings.TrimSpace(path) {
		case "description", "etag":
		default:
			return grpcInvalidArgument("update_mask-invalid")
		}
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("update-tagkey-"+tagKeyID, false))
}

func gcpStage4GRPCResourceManagerV3DeleteTagKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.DeleteTagKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	tagKeyID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "tagKeys")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("delete-tagkey-"+tagKeyID, false))
}

func gcpStage4GRPCResourceManagerV3ListTagValues(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.ListTagValuesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !isGCPResourceManagerV3TagValueParent(strings.TrimSpace(req.GetParent())) {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.TagValue{
		gcpStage4ResourceManagerV3TagValue("3001", req.GetParent(), "prod"),
		gcpStage4ResourceManagerV3TagValue("3002", req.GetParent(), "dev"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.ListTagValuesResponse{TagValues: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3GetTagValue(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.GetTagValueRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	tagValueID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "tagValues")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3TagValue(tagValueID, "tagKeys/2001", "prod"))
}

func gcpStage4GRPCResourceManagerV3GetNamespacedTagValue(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.GetNamespacedTagValueRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return grpcInvalidArgument("name-required")
	}
	shortName := "prod"
	if idx := strings.LastIndex(req.GetName(), "/"); idx >= 0 && idx+1 < len(req.GetName()) {
		shortName = req.GetName()[idx+1:]
	}
	tagValue := gcpStage4ResourceManagerV3TagValue("3001", "tagKeys/2001", shortName)
	tagValue.NamespacedName = req.GetName()
	return grpcProtoSuccess(tagValue)
}

func gcpStage4GRPCResourceManagerV3CreateTagValue(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.CreateTagValueRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetTagValue() == nil {
		return grpcInvalidArgument("tag_value-required")
	}
	if !isGCPResourceManagerV3TagValueParent(strings.TrimSpace(req.GetTagValue().GetParent())) {
		return grpcInvalidArgument("tag_value.parent-invalid")
	}
	if !isGCPResourceManagerV3TagShortName(strings.TrimSpace(req.GetTagValue().GetShortName())) {
		return grpcInvalidArgument("tag_value.short_name-invalid")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("create-tagvalue-3001", false))
}

func gcpStage4GRPCResourceManagerV3UpdateTagValue(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.UpdateTagValueRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetTagValue() == nil {
		return grpcInvalidArgument("tag_value-required")
	}
	tagValueID, ok := parseGCPStage4ResourceManagerV3Name(req.GetTagValue().GetName(), "tagValues")
	if !ok {
		return grpcInvalidArgument("tag_value.name-required")
	}
	if strings.TrimSpace(req.GetTagValue().GetDescription()) == "" {
		return grpcInvalidArgument("tag_value.description-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	for _, path := range req.GetUpdateMask().GetPaths() {
		switch strings.TrimSpace(path) {
		case "description", "etag":
		default:
			return grpcInvalidArgument("update_mask-invalid")
		}
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("update-tagvalue-"+tagValueID, false))
}

func gcpStage4GRPCResourceManagerV3DeleteTagValue(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.DeleteTagValueRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	tagValueID, ok := parseGCPStage4ResourceManagerV3Name(req.GetName(), "tagValues")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("delete-tagvalue-"+tagValueID, false))
}

func gcpStage4GRPCResourceManagerV3ListTagBindings(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.ListTagBindingsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetParent()) == "" {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.TagBinding{gcpStage4ResourceManagerV3TagBinding(req.GetParent(), "tagValues/3001")}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.ListTagBindingsResponse{TagBindings: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3CreateTagBinding(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.CreateTagBindingRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	tagBinding := req.GetTagBinding()
	if tagBinding == nil {
		return grpcInvalidArgument("tag_binding-required")
	}
	if strings.TrimSpace(tagBinding.GetParent()) == "" {
		return grpcInvalidArgument("tag_binding.parent-required")
	}
	tagValue := strings.TrimSpace(tagBinding.GetTagValue())
	namespaced := strings.TrimSpace(tagBinding.GetTagValueNamespacedName())
	if tagValue == "" && namespaced == "" {
		return grpcInvalidArgument("tag_binding.tag_value-required")
	}
	if tagValue != "" && namespaced != "" {
		return grpcInvalidArgument("tag_binding.tag_value-mutually-exclusive")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("create-tagbinding-3001", false))
}

func gcpStage4GRPCResourceManagerV3DeleteTagBinding(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.DeleteTagBindingRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("delete-tagbinding-3001", false))
}

func gcpStage4GRPCResourceManagerV3ListEffectiveTags(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.ListEffectiveTagsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetParent()) == "" {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.EffectiveTag{gcpStage4ResourceManagerV3EffectiveTag(req.GetParent(), "tagKeys/2001", "tagValues/3001")}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.ListEffectiveTagsResponse{EffectiveTags: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3ListTagHolds(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.ListTagHoldsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	tagValueID, ok := parseGCPStage4ResourceManagerV3Name(req.GetParent(), "tagValues")
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerv3pb.TagHold{gcpStage4ResourceManagerV3TagHold(tagValueID, "hold-1")}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerv3pb.ListTagHoldsResponse{TagHolds: items[start:end], NextPageToken: next})
}

func gcpStage4GRPCResourceManagerV3CreateTagHold(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.CreateTagHoldRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, ok := parseGCPStage4ResourceManagerV3Name(req.GetParent(), "tagValues"); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetTagHold() == nil {
		return grpcInvalidArgument("tag_hold-required")
	}
	if strings.TrimSpace(req.GetTagHold().GetHolder()) == "" {
		return grpcInvalidArgument("tag_hold.holder-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("create-taghold-3001", false))
}

func gcpStage4GRPCResourceManagerV3DeleteTagHold(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerv3pb.DeleteTagHoldRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPStage4ResourceManagerV3TagHoldName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Operation("delete-taghold-3001-hold-1", false))
}

func gcpStage4GRPCResourceManagerV3GetIAMPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.GetIamPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !isGCPResourceManagerV3IAMResource(strings.TrimSpace(req.GetResource())) {
		return grpcInvalidArgument("resource-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Policy(req.GetResource(), nil))
}

func gcpStage4GRPCResourceManagerV3SetIAMPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.SetIamPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !isGCPResourceManagerV3IAMResource(strings.TrimSpace(req.GetResource())) {
		return grpcInvalidArgument("resource-required")
	}
	if req.GetPolicy() == nil {
		return grpcInvalidArgument("policy-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerV3Policy(req.GetResource(), req.GetPolicy()))
}

func gcpStage4GRPCResourceManagerV3TestIAMPermissions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.TestIamPermissionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !isGCPResourceManagerV3IAMResource(strings.TrimSpace(req.GetResource())) {
		return grpcInvalidArgument("resource-required")
	}
	if len(req.GetPermissions()) == 0 {
		return grpcInvalidArgument("permissions-required")
	}
	return grpcProtoSuccess(&iampb.TestIamPermissionsResponse{Permissions: req.GetPermissions()})
}

func gcpStage4ResourceManagerV3Folder(folderID, parent, displayName string, state resourcemanagerv3pb.Folder_State) *resourcemanagerv3pb.Folder {
	return &resourcemanagerv3pb.Folder{
		Name:        "folders/" + folderID,
		Parent:      parent,
		DisplayName: displayName,
		State:       state,
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:  timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ResourceManagerV3Project(projectNumber, parent, projectID, displayName string, state resourcemanagerv3pb.Project_State) *resourcemanagerv3pb.Project {
	return &resourcemanagerv3pb.Project{
		Name:        "projects/" + projectNumber,
		Parent:      parent,
		ProjectId:   projectID,
		DisplayName: displayName,
		State:       state,
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:  timestamppb.New(gcpStage4ReferenceTime),
		Etag:        "resourcemanager-project-etag",
		Labels: map[string]string{
			"environment": "test",
			"owner":       "stackyard",
		},
	}
}

func gcpStage4ResourceManagerV3Organization(orgID, displayName string) *resourcemanagerv3pb.Organization {
	return &resourcemanagerv3pb.Organization{
		Name:        "organizations/" + orgID,
		DisplayName: displayName,
		Owner: &resourcemanagerv3pb.Organization_DirectoryCustomerId{
			DirectoryCustomerId: "C0123abc",
		},
		State:      resourcemanagerv3pb.Organization_ACTIVE,
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
		Etag:       "resourcemanager-org-etag",
	}
}

func gcpStage4ResourceManagerV3TagKey(tagKeyID, parent, shortName string) *resourcemanagerv3pb.TagKey {
	return &resourcemanagerv3pb.TagKey{
		Name:           "tagKeys/" + tagKeyID,
		Parent:         parent,
		ShortName:      shortName,
		NamespacedName: "123456/" + shortName,
		Description:    "Tag key for " + shortName,
		CreateTime:     timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:     timestamppb.New(gcpStage4ReferenceTime),
		Etag:           "resourcemanager-tagkey-etag",
		Purpose:        resourcemanagerv3pb.Purpose_GCE_FIREWALL,
		PurposeData: map[string]string{
			"network": "default",
		},
	}
}

func gcpStage4ResourceManagerV3TagValue(tagValueID, parent, shortName string) *resourcemanagerv3pb.TagValue {
	return &resourcemanagerv3pb.TagValue{
		Name:           "tagValues/" + tagValueID,
		Parent:         parent,
		ShortName:      shortName,
		NamespacedName: "123456/env/" + shortName,
		Description:    "Tag value for " + shortName,
		CreateTime:     timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:     timestamppb.New(gcpStage4ReferenceTime),
		Etag:           "resourcemanager-tagvalue-etag",
	}
}

func gcpStage4ResourceManagerV3TagBinding(parent, tagValue string) *resourcemanagerv3pb.TagBinding {
	encodedParent := strings.NewReplacer("/", "%2F", ":", "%3A").Replace(parent)
	return &resourcemanagerv3pb.TagBinding{
		Name:                   "tagBindings/" + encodedParent + "/" + tagValue,
		Parent:                 parent,
		TagValue:               tagValue,
		TagValueNamespacedName: "123456/env/prod",
	}
}

func gcpStage4ResourceManagerV3EffectiveTag(parent, tagKey, tagValue string) *resourcemanagerv3pb.EffectiveTag {
	return &resourcemanagerv3pb.EffectiveTag{
		TagValue:           tagValue,
		NamespacedTagValue: "123456/env/prod",
		TagKey:             tagKey,
		NamespacedTagKey:   "123456/env",
		TagKeyParentName:   "organizations/123456",
		Inherited:          !strings.Contains(parent, "projects/"),
	}
}

func gcpStage4ResourceManagerV3TagHold(tagValueID, holdID string) *resourcemanagerv3pb.TagHold {
	return &resourcemanagerv3pb.TagHold{
		Name:       "tagValues/" + tagValueID + "/tagHolds/" + holdID,
		Holder:     "//cloudresourcemanager.googleapis.com/projects/415104041262",
		Origin:     "stackyard",
		HelpLink:   "https://cloud.google.com/resource-manager/docs/tags/tags-creating-and-managing",
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ResourceManagerV3Operation(operationID string, done bool) *longrunningpb.Operation {
	return &longrunningpb.Operation{Name: "operations/" + operationID, Done: done}
}

func gcpStage4ResourceManagerV3Policy(resource string, policy *iampb.Policy) *iampb.Policy {
	if policy == nil {
		role := "roles/viewer"
		switch {
		case strings.HasPrefix(resource, "folders/"):
			role = "roles/resourcemanager.folderViewer"
		case strings.HasPrefix(resource, "projects/"):
			role = "roles/resourcemanager.projectViewer"
		case strings.HasPrefix(resource, "organizations/"):
			role = "roles/resourcemanager.organizationViewer"
		case strings.HasPrefix(resource, "tagKeys/"):
			role = "roles/resourcemanager.tagAdmin"
		case strings.HasPrefix(resource, "tagValues/"):
			role = "roles/resourcemanager.tagUser"
		}
		return &iampb.Policy{
			Version: 1,
			Etag:    []byte("resourcemanager-v3-etag"),
			Bindings: []*iampb.Binding{{
				Role:    role,
				Members: []string{"user:alice@example.com"},
			}},
		}
	}
	cloned, ok := proto.Clone(policy).(*iampb.Policy)
	if !ok || cloned == nil {
		return policy
	}
	if len(cloned.GetEtag()) == 0 {
		cloned.Etag = []byte("resourcemanager-v3-etag")
	}
	if cloned.GetVersion() == 0 {
		cloned.Version = 1
	}
	return cloned
}

func parseGCPStage4ResourceManagerV3Name(name, kind string) (id string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 2 || parts[0] != kind {
		return "", false
	}
	id = strings.TrimSpace(parts[1])
	if id == "" {
		return "", false
	}
	return id, true
}

func parseGCPStage4ResourceManagerV3TagHoldName(name string) (tagValueID, holdID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "tagValues" || parts[2] != "tagHolds" {
		return "", "", false
	}
	tagValueID = strings.TrimSpace(parts[1])
	holdID = strings.TrimSpace(parts[3])
	if tagValueID == "" || holdID == "" {
		return "", "", false
	}
	return tagValueID, holdID, true
}
