package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type workspacesStore struct {
	mu sync.Mutex

	nextID int64

	workspaces        map[string]*workspacesWorkspace
	directories       map[string]*workspacesDirectory
	bundles           map[string]*workspacesBundle
	images            map[string]*workspacesImage
	pools             map[string]*workspacesPool
	applications      map[string]*workspacesApplication
	connectionAliases map[string]*workspacesConnectionAlias
	ipGroups          map[string]*workspacesIPGroup
	accountLinks      map[string]*workspacesAccountLink
	connectAddIns     map[string]*workspacesConnectAddIn
	tags              map[string]map[string]string
}

type workspacesWorkspace struct {
	ID          string
	DirectoryID string
	BundleID    string
	UserName    string
	State       string
	CreatedAt   string
}

type workspacesDirectory struct {
	ID        string
	Alias     string
	State     string
	CreatedAt string
}

type workspacesBundle struct {
	ID   string
	Name string
}

type workspacesImage struct {
	ID   string
	Name string
}

type workspacesPool struct {
	ID    string
	Name  string
	State string
}

type workspacesApplication struct {
	ID   string
	Name string
}

type workspacesConnectionAlias struct {
	ID    string
	Alias string
}

type workspacesIPGroup struct {
	ID   string
	Name string
}

type workspacesAccountLink struct {
	ID     string
	Status string
}

type workspacesConnectAddIn struct {
	ID   string
	Name string
}

func newWorkSpacesStore() *workspacesStore {
	now := time.Now().UTC().Format(time.RFC3339)

	seedWorkspace := &workspacesWorkspace{ID: "ws-000001", DirectoryID: "d-000001", BundleID: "wsb-000001", UserName: "stackyard-user", State: "AVAILABLE", CreatedAt: now}
	seedDirectory := &workspacesDirectory{ID: "d-000001", Alias: "stackyard-directory", State: "REGISTERED", CreatedAt: now}
	seedBundle := &workspacesBundle{ID: "wsb-000001", Name: "stackyard-bundle"}
	seedImage := &workspacesImage{ID: "wsi-000001", Name: "stackyard-image"}
	seedPool := &workspacesPool{ID: "wspool-000001", Name: "stackyard-pool", State: "AVAILABLE"}
	seedApp := &workspacesApplication{ID: "wsapp-000001", Name: "stackyard-application"}
	seedAlias := &workspacesConnectionAlias{ID: "wsca-000001", Alias: "stackyard-alias"}
	seedIPGroup := &workspacesIPGroup{ID: "wsipg-000001", Name: "stackyard-ip-group"}
	seedAccountLink := &workspacesAccountLink{ID: "al-000001", Status: "ACCEPTED"}
	seedAddIn := &workspacesConnectAddIn{ID: "addin-000001", Name: "stackyard-addin"}

	return &workspacesStore{
		nextID: 2,
		workspaces: map[string]*workspacesWorkspace{
			seedWorkspace.ID: seedWorkspace,
		},
		directories: map[string]*workspacesDirectory{
			seedDirectory.ID: seedDirectory,
		},
		bundles: map[string]*workspacesBundle{
			seedBundle.ID: seedBundle,
		},
		images: map[string]*workspacesImage{
			seedImage.ID: seedImage,
		},
		pools: map[string]*workspacesPool{
			seedPool.ID: seedPool,
		},
		applications: map[string]*workspacesApplication{
			seedApp.ID: seedApp,
		},
		connectionAliases: map[string]*workspacesConnectionAlias{
			seedAlias.ID: seedAlias,
		},
		ipGroups: map[string]*workspacesIPGroup{
			seedIPGroup.ID: seedIPGroup,
		},
		accountLinks: map[string]*workspacesAccountLink{
			seedAccountLink.ID: seedAccountLink,
		},
		connectAddIns: map[string]*workspacesConnectAddIn{
			seedAddIn.ID: seedAddIn,
		},
		tags: map[string]map[string]string{
			seedWorkspace.ID: {"stackyard": "true"},
		},
	}
}

func (s *workspacesStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "DescribeAccount":
		return map[string]any{
			"DedicatedTenancySupport": "ENABLED",
			"EncryptionKey":           "alias/aws/workspaces",
			"WorkspacesDefaultRole":   "arn:aws:iam::123456789012:role/workspaces_DefaultRole",
		}

	case "ModifyAccount", "ModifyCertificateBasedAuthProperties", "ModifyClientProperties", "ModifyEndpointEncryptionMode", "ModifySamlProperties", "ModifySelfservicePermissions", "ModifyStreamingProperties", "ModifyWorkspaceAccessProperties", "ModifyWorkspaceCreationProperties", "ModifyWorkspaceProperties", "ModifyWorkspaceState", "RebootWorkspaces", "RebuildWorkspaces", "RestoreWorkspace", "MigrateWorkspace", "StartWorkspaces", "StopWorkspaces", "TerminateWorkspaces", "UpdateConnectionAliasPermission", "UpdateRulesOfIpGroup", "UpdateWorkspaceBundle", "UpdateWorkspaceImagePermission", "UpdateWorkspacesPool", "AuthorizeIpRules", "RevokeIpRules", "AssociateConnectionAlias", "DisassociateConnectionAlias", "AssociateIpGroups", "DisassociateIpGroups", "AssociateWorkspaceApplication", "DisassociateWorkspaceApplication", "DeployWorkspaceApplications", "DeleteWorkspaceBundle", "DeleteWorkspaceImage", "DeleteConnectionAlias", "DeleteClientBranding", "DeleteConnectClientAddIn", "DeregisterWorkspaceDirectory", "ImportClientBranding", "StopWorkspacesPool", "StartWorkspacesPool", "TerminateWorkspacesPool", "TerminateWorkspacesPoolSession":
		if action == "TerminateWorkspaces" {
			for _, id := range s.extractWorkspaceIDs(payload) {
				if ws, ok := s.workspaces[id]; ok {
					ws.State = "TERMINATED"
				}
			}
		}
		if action == "StartWorkspaces" {
			for _, id := range s.extractWorkspaceIDs(payload) {
				if ws, ok := s.workspaces[id]; ok {
					ws.State = "AVAILABLE"
				}
			}
		}
		if action == "StopWorkspaces" {
			for _, id := range s.extractWorkspaceIDs(payload) {
				if ws, ok := s.workspaces[id]; ok {
					ws.State = "STOPPED"
				}
			}
		}
		return map[string]any{}

	case "DescribeAccountModifications":
		return map[string]any{
			"AccountModifications": []any{
				map[string]any{"ModificationState": "UPDATE_INITIATED", "DedicatedTenancySupport": "ENABLED"},
			},
		}

	case "CreateWorkspaces":
		pending := []any{}
		requests := workspacesPayloadSlice(payload, "Workspaces")
		if len(requests) == 0 {
			requests = []any{map[string]any{}}
		}
		for _, raw := range requests {
			req, _ := raw.(map[string]any)
			id := s.nextTokenLocked("ws")
			directoryID := workspacesPayloadString(req, "DirectoryId", s.firstDirectoryIDLocked())
			bundleID := workspacesPayloadString(req, "BundleId", s.firstBundleIDLocked())
			userName := workspacesPayloadString(req, "UserName", "stackyard-user")
			s.workspaces[id] = &workspacesWorkspace{ID: id, DirectoryID: directoryID, BundleID: bundleID, UserName: userName, State: "PENDING", CreatedAt: now}
			s.ensureTagMapLocked(id)
			pending = append(pending, map[string]any{"WorkspaceId": id, "UserName": userName, "State": "PENDING"})
		}
		return map[string]any{"FailedRequests": []any{}, "PendingRequests": pending}

	case "DescribeWorkspaces":
		ids := workspacesPayloadStringList(payload, "WorkspaceIds")
		items := []any{}
		for _, ws := range s.sortedWorkspacesLocked() {
			if len(ids) > 0 && !workspacesContainsString(ids, ws.ID) {
				continue
			}
			items = append(items, map[string]any{
				"WorkspaceId":         ws.ID,
				"DirectoryId":         ws.DirectoryID,
				"BundleId":            ws.BundleID,
				"UserName":            ws.UserName,
				"State":               ws.State,
				"WorkspaceProperties": map[string]any{"RunningMode": "AUTO_STOP"},
				"WorkspaceName":       fmt.Sprintf("workspace-%s", ws.ID),
			})
		}
		return map[string]any{"Workspaces": items, "NextToken": ""}

	case "DescribeWorkspacesConnectionStatus":
		ids := workspacesPayloadStringList(payload, "WorkspaceIds")
		if len(ids) == 0 {
			ids = []string{s.firstWorkspaceIDLocked()}
		}
		statuses := make([]any, 0, len(ids))
		for _, id := range ids {
			statuses = append(statuses, map[string]any{
				"WorkspaceId":                      id,
				"ConnectionState":                  "CONNECTED",
				"LastKnownUserConnectionTimestamp": now,
			})
		}
		return map[string]any{"WorkspacesConnectionStatus": statuses}

	case "CreateStandbyWorkspaces":
		return map[string]any{"PendingStandbyRequests": []any{}, "FailedStandbyRequests": []any{}}

	case "RegisterWorkspaceDirectory":
		id := workspacesPayloadString(payload, "DirectoryId", s.nextTokenLocked("d"))
		s.directories[id] = &workspacesDirectory{ID: id, Alias: fmt.Sprintf("directory-%s", id), State: "REGISTERED", CreatedAt: now}
		return map[string]any{}

	case "DescribeWorkspaceDirectories":
		items := make([]any, 0, len(s.directories))
		for _, dir := range s.sortedDirectoriesLocked() {
			items = append(items, map[string]any{
				"DirectoryId":              dir.ID,
				"Alias":                    dir.Alias,
				"State":                    dir.State,
				"WorkspaceSecurityGroupId": "sg-000001",
			})
		}
		return map[string]any{"Directories": items, "NextToken": ""}

	case "DescribeClientProperties":
		ids := workspacesPayloadStringList(payload, "ResourceIds")
		if len(ids) == 0 {
			ids = []string{s.firstDirectoryIDLocked()}
		}
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			out = append(out, map[string]any{
				"ResourceId": id,
				"ClientProperties": map[string]any{
					"ReconnectEnabled": "ENABLED",
				},
			})
		}
		return map[string]any{"ClientPropertiesList": out}

	case "CreateWorkspaceBundle":
		id := s.nextTokenLocked("wsb")
		name := workspacesPayloadString(payload, "BundleName", fmt.Sprintf("bundle-%s", id))
		s.bundles[id] = &workspacesBundle{ID: id, Name: name}
		return map[string]any{"WorkspaceBundle": map[string]any{"BundleId": id, "Name": name}}

	case "DescribeWorkspaceBundles":
		items := make([]any, 0, len(s.bundles))
		for _, b := range s.sortedBundlesLocked() {
			items = append(items, map[string]any{"BundleId": b.ID, "Name": b.Name})
		}
		return map[string]any{"Bundles": items, "NextToken": ""}

	case "CreateWorkspaceImage", "CopyWorkspaceImage", "CreateUpdatedWorkspaceImage", "ImportWorkspaceImage", "ImportCustomWorkspaceImage":
		id := s.nextTokenLocked("wsi")
		name := workspacesPayloadString(payload, "Name", fmt.Sprintf("image-%s", id))
		s.images[id] = &workspacesImage{ID: id, Name: name}
		return map[string]any{"ImageId": id}

	case "DescribeWorkspaceImages":
		items := make([]any, 0, len(s.images))
		for _, img := range s.sortedImagesLocked() {
			items = append(items, map[string]any{"ImageId": img.ID, "Name": img.Name, "State": "AVAILABLE"})
		}
		return map[string]any{"Images": items, "NextToken": ""}

	case "DescribeCustomWorkspaceImageImport":
		id := workspacesPayloadString(payload, "ImageId", s.firstImageIDLocked())
		return map[string]any{"ImageId": id, "Status": "COMPLETED"}

	case "DescribeWorkspaceImagePermissions":
		return map[string]any{"ImagePermissions": []any{map[string]any{"SharedAccountId": "123456789012"}}}

	case "DescribeImageAssociations":
		return map[string]any{"ImageAssociations": []any{map[string]any{"ImageId": s.firstImageIDLocked(), "AssociatedResourceId": s.firstWorkspaceIDLocked(), "AssociationState": "ASSOCIATED"}}}

	case "DescribeBundleAssociations":
		return map[string]any{"BundleAssociations": []any{map[string]any{"BundleId": s.firstBundleIDLocked(), "AssociatedResourceId": s.firstWorkspaceIDLocked(), "AssociationState": "ASSOCIATED"}}}

	case "DescribeWorkspaceSnapshots":
		wsID := workspacesPayloadString(payload, "WorkspaceId", s.firstWorkspaceIDLocked())
		return map[string]any{"RebuildSnapshots": []any{map[string]any{"WorkspaceId": wsID, "SnapshotTime": now}}}

	case "CreateConnectionAlias":
		id := s.nextTokenLocked("wsca")
		alias := workspacesPayloadString(payload, "ConnectionString", fmt.Sprintf("alias-%s", id))
		s.connectionAliases[id] = &workspacesConnectionAlias{ID: id, Alias: alias}
		return map[string]any{"ConnectionAlias": map[string]any{"ConnectionAliasId": id, "Alias": alias, "State": "CREATED"}}

	case "DescribeConnectionAliases":
		items := make([]any, 0, len(s.connectionAliases))
		for _, alias := range s.sortedConnectionAliasesLocked() {
			items = append(items, map[string]any{"ConnectionAliasId": alias.ID, "Alias": alias.Alias, "State": "ASSOCIATED"})
		}
		return map[string]any{"ConnectionAliases": items, "NextToken": ""}

	case "DescribeConnectionAliasPermissions":
		return map[string]any{"ConnectionAliasPermissions": []any{map[string]any{"SharedAccountId": "123456789012", "AllowAssociation": true}}}

	case "CreateConnectClientAddIn":
		id := s.nextTokenLocked("addin")
		name := workspacesPayloadString(payload, "Name", fmt.Sprintf("addin-%s", id))
		s.connectAddIns[id] = &workspacesConnectAddIn{ID: id, Name: name}
		return map[string]any{"AddIn": map[string]any{"AddInId": id, "Name": name}}

	case "DescribeConnectClientAddIns":
		items := make([]any, 0, len(s.connectAddIns))
		for _, addIn := range s.sortedConnectAddInsLocked() {
			items = append(items, map[string]any{"AddInId": addIn.ID, "Name": addIn.Name})
		}
		return map[string]any{"AddIns": items, "NextToken": ""}

	case "DescribeApplications":
		items := make([]any, 0, len(s.applications))
		for _, app := range s.sortedApplicationsLocked() {
			items = append(items, map[string]any{"ApplicationId": app.ID, "Name": app.Name, "State": "AVAILABLE"})
		}
		return map[string]any{"Applications": items, "NextToken": ""}

	case "DescribeApplicationAssociations":
		return map[string]any{"Associations": []any{map[string]any{"ApplicationId": s.firstApplicationIDLocked(), "AssociatedResourceId": s.firstWorkspaceIDLocked(), "State": "ASSOCIATED"}}}

	case "DescribeWorkspaceAssociations":
		return map[string]any{"Associations": []any{map[string]any{"WorkspaceId": s.firstWorkspaceIDLocked(), "AssociatedResourceId": s.firstApplicationIDLocked(), "AssociationState": "ASSOCIATED"}}}

	case "CreateWorkspacesPool":
		id := s.nextTokenLocked("wspool")
		name := workspacesPayloadString(payload, "PoolName", fmt.Sprintf("pool-%s", id))
		s.pools[id] = &workspacesPool{ID: id, Name: name, State: "AVAILABLE"}
		return map[string]any{"WorkspacesPool": map[string]any{"WorkspacesPoolId": id, "PoolName": name, "State": "AVAILABLE"}}

	case "DescribeWorkspacesPools":
		items := make([]any, 0, len(s.pools))
		for _, pool := range s.sortedPoolsLocked() {
			items = append(items, map[string]any{"WorkspacesPoolId": pool.ID, "PoolName": pool.Name, "State": pool.State})
		}
		return map[string]any{"WorkspacesPools": items, "NextToken": ""}

	case "DescribeWorkspacesPoolSessions":
		poolID := workspacesPayloadString(payload, "WorkspacesPoolId", s.firstPoolIDLocked())
		return map[string]any{"Sessions": []any{map[string]any{"WorkspacesPoolId": poolID, "SessionId": "session-000001", "State": "ACTIVE"}}, "NextToken": ""}

	case "CreateIpGroup":
		id := s.nextTokenLocked("wsipg")
		name := workspacesPayloadString(payload, "GroupName", fmt.Sprintf("ip-group-%s", id))
		s.ipGroups[id] = &workspacesIPGroup{ID: id, Name: name}
		return map[string]any{"GroupId": id}

	case "DescribeIpGroups":
		items := make([]any, 0, len(s.ipGroups))
		for _, group := range s.sortedIPGroupsLocked() {
			items = append(items, map[string]any{"groupId": group.ID, "groupName": group.Name})
		}
		return map[string]any{"Result": items, "NextToken": ""}

	case "DeleteIpGroup":
		groupID := workspacesPayloadString(payload, "GroupId", "")
		if groupID != "" {
			delete(s.ipGroups, groupID)
		}
		return map[string]any{}

	case "CreateAccountLinkInvitation":
		id := s.nextTokenLocked("al")
		s.accountLinks[id] = &workspacesAccountLink{ID: id, Status: "PENDING_ACCEPTANCE"}
		return map[string]any{"AccountLink": map[string]any{"AccountLinkId": id, "Status": "PENDING_ACCEPTANCE"}}

	case "AcceptAccountLinkInvitation":
		id := workspacesPayloadString(payload, "AccountLinkId", s.firstAccountLinkIDLocked())
		link := s.ensureAccountLinkLocked(id)
		link.Status = "ACCEPTED"
		return map[string]any{}

	case "RejectAccountLinkInvitation":
		id := workspacesPayloadString(payload, "AccountLinkId", s.firstAccountLinkIDLocked())
		link := s.ensureAccountLinkLocked(id)
		link.Status = "REJECTED"
		return map[string]any{}

	case "DeleteAccountLinkInvitation":
		id := workspacesPayloadString(payload, "AccountLinkId", s.firstAccountLinkIDLocked())
		delete(s.accountLinks, id)
		return map[string]any{}

	case "ListAccountLinks":
		items := make([]any, 0, len(s.accountLinks))
		for _, link := range s.sortedAccountLinksLocked() {
			items = append(items, map[string]any{"AccountLinkId": link.ID, "Status": link.Status})
		}
		return map[string]any{"AccountLinks": items, "NextToken": ""}

	case "GetAccountLink":
		id := workspacesPayloadString(payload, "AccountLinkId", s.firstAccountLinkIDLocked())
		link := s.ensureAccountLinkLocked(id)
		return map[string]any{"AccountLink": map[string]any{"AccountLinkId": link.ID, "Status": link.Status}}

	case "CreateTags":
		resourceID := workspacesPayloadString(payload, "ResourceId", s.firstWorkspaceIDLocked())
		s.ensureTagMapLocked(resourceID)
		for k, v := range workspacesExtractTags(payload) {
			s.tags[resourceID][k] = v
		}
		return map[string]any{}

	case "DeleteTags":
		resourceID := workspacesPayloadString(payload, "ResourceId", s.firstWorkspaceIDLocked())
		s.ensureTagMapLocked(resourceID)
		for _, key := range workspacesPayloadStringList(payload, "TagKeys") {
			delete(s.tags[resourceID], key)
		}
		return map[string]any{}

	case "DescribeTags":
		resourceID := workspacesPayloadString(payload, "ResourceId", s.firstWorkspaceIDLocked())
		s.ensureTagMapLocked(resourceID)
		keys := make([]string, 0, len(s.tags[resourceID]))
		for k := range s.tags[resourceID] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		tagList := make([]any, 0, len(keys))
		for _, k := range keys {
			tagList = append(tagList, map[string]any{"Key": k, "Value": s.tags[resourceID][k]})
		}
		return map[string]any{"TagList": tagList}

	case "ListAvailableManagementCidrRanges":
		return map[string]any{"ManagementCidrRanges": []any{"10.0.0.0/16", "10.10.0.0/16"}}
	}

	return map[string]any{}
}

func (s *workspacesStore) nextTokenLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%06d", prefix, id)
}

func (s *workspacesStore) firstWorkspaceIDLocked() string {
	for _, ws := range s.sortedWorkspacesLocked() {
		return ws.ID
	}
	id := "ws-000001"
	s.workspaces[id] = &workspacesWorkspace{ID: id, DirectoryID: s.firstDirectoryIDLocked(), BundleID: s.firstBundleIDLocked(), UserName: "stackyard-user", State: "AVAILABLE", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	return id
}

func (s *workspacesStore) firstDirectoryIDLocked() string {
	for _, dir := range s.sortedDirectoriesLocked() {
		return dir.ID
	}
	id := "d-000001"
	s.directories[id] = &workspacesDirectory{ID: id, Alias: "stackyard-directory", State: "REGISTERED", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	return id
}

func (s *workspacesStore) firstBundleIDLocked() string {
	for _, b := range s.sortedBundlesLocked() {
		return b.ID
	}
	id := "wsb-000001"
	s.bundles[id] = &workspacesBundle{ID: id, Name: "stackyard-bundle"}
	return id
}

func (s *workspacesStore) firstImageIDLocked() string {
	for _, img := range s.sortedImagesLocked() {
		return img.ID
	}
	id := "wsi-000001"
	s.images[id] = &workspacesImage{ID: id, Name: "stackyard-image"}
	return id
}

func (s *workspacesStore) firstPoolIDLocked() string {
	for _, pool := range s.sortedPoolsLocked() {
		return pool.ID
	}
	id := "wspool-000001"
	s.pools[id] = &workspacesPool{ID: id, Name: "stackyard-pool", State: "AVAILABLE"}
	return id
}

func (s *workspacesStore) firstApplicationIDLocked() string {
	for _, app := range s.sortedApplicationsLocked() {
		return app.ID
	}
	id := "wsapp-000001"
	s.applications[id] = &workspacesApplication{ID: id, Name: "stackyard-application"}
	return id
}

func (s *workspacesStore) firstAccountLinkIDLocked() string {
	for _, link := range s.sortedAccountLinksLocked() {
		return link.ID
	}
	id := "al-000001"
	s.accountLinks[id] = &workspacesAccountLink{ID: id, Status: "ACCEPTED"}
	return id
}

func (s *workspacesStore) ensureAccountLinkLocked(id string) *workspacesAccountLink {
	if existing, ok := s.accountLinks[id]; ok {
		return existing
	}
	link := &workspacesAccountLink{ID: id, Status: "PENDING_ACCEPTANCE"}
	s.accountLinks[id] = link
	return link
}

func (s *workspacesStore) sortedWorkspacesLocked() []*workspacesWorkspace {
	items := make([]*workspacesWorkspace, 0, len(s.workspaces))
	for _, ws := range s.workspaces {
		items = append(items, ws)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) sortedDirectoriesLocked() []*workspacesDirectory {
	items := make([]*workspacesDirectory, 0, len(s.directories))
	for _, dir := range s.directories {
		items = append(items, dir)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) sortedBundlesLocked() []*workspacesBundle {
	items := make([]*workspacesBundle, 0, len(s.bundles))
	for _, bundle := range s.bundles {
		items = append(items, bundle)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) sortedImagesLocked() []*workspacesImage {
	items := make([]*workspacesImage, 0, len(s.images))
	for _, img := range s.images {
		items = append(items, img)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) sortedPoolsLocked() []*workspacesPool {
	items := make([]*workspacesPool, 0, len(s.pools))
	for _, pool := range s.pools {
		items = append(items, pool)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) sortedApplicationsLocked() []*workspacesApplication {
	items := make([]*workspacesApplication, 0, len(s.applications))
	for _, app := range s.applications {
		items = append(items, app)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) sortedConnectionAliasesLocked() []*workspacesConnectionAlias {
	items := make([]*workspacesConnectionAlias, 0, len(s.connectionAliases))
	for _, alias := range s.connectionAliases {
		items = append(items, alias)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) sortedIPGroupsLocked() []*workspacesIPGroup {
	items := make([]*workspacesIPGroup, 0, len(s.ipGroups))
	for _, group := range s.ipGroups {
		items = append(items, group)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) sortedAccountLinksLocked() []*workspacesAccountLink {
	items := make([]*workspacesAccountLink, 0, len(s.accountLinks))
	for _, link := range s.accountLinks {
		items = append(items, link)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) sortedConnectAddInsLocked() []*workspacesConnectAddIn {
	items := make([]*workspacesConnectAddIn, 0, len(s.connectAddIns))
	for _, addIn := range s.connectAddIns {
		items = append(items, addIn)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workspacesStore) ensureTagMapLocked(resourceID string) {
	if _, ok := s.tags[resourceID]; ok {
		return
	}
	s.tags[resourceID] = map[string]string{}
}

func (s *workspacesStore) extractWorkspaceIDs(payload map[string]any) []string {
	ids := workspacesPayloadStringList(payload, "WorkspaceIds")
	if len(ids) > 0 {
		return ids
	}

	requestKeys := []string{"StartWorkspaceRequests", "StopWorkspaceRequests", "RebootWorkspaceRequests", "RebuildWorkspaceRequests", "TerminateWorkspaceRequests"}
	for _, key := range requestKeys {
		for _, item := range workspacesPayloadSlice(payload, key) {
			if m, ok := item.(map[string]any); ok {
				if id := workspacesPayloadString(m, "WorkspaceId", ""); id != "" {
					ids = append(ids, id)
				}
			}
		}
	}

	if len(ids) == 0 {
		ids = append(ids, s.firstWorkspaceIDLocked())
	}
	return ids
}

func workspacesPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return fallback
	}
	value, ok := raw.(string)
	if !ok {
		return fallback
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func workspacesPayloadSlice(payload map[string]any, key string) []any {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	slice, ok := raw.([]any)
	if !ok {
		return nil
	}
	return slice
}

func workspacesPayloadStringList(payload map[string]any, key string) []string {
	items := workspacesPayloadSlice(payload, key)
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value, ok := item.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func workspacesExtractTags(payload map[string]any) map[string]string {
	result := map[string]string{}
	for _, rawTag := range workspacesPayloadSlice(payload, "Tags") {
		tagMap, ok := rawTag.(map[string]any)
		if !ok {
			continue
		}
		key := workspacesPayloadString(tagMap, "Key", "")
		value := workspacesPayloadString(tagMap, "Value", "")
		if key == "" {
			continue
		}
		result[key] = value
	}
	return result
}

func workspacesContainsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
