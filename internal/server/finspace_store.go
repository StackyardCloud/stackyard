package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type finspaceStore struct {
	mu sync.Mutex

	nextDatasetID         int64
	nextDataViewID        int64
	nextChangesetID       int64
	nextUserID            int64
	nextPermissionGroupID int64

	datasets         map[string]*finspaceDataset
	dataViews        map[string]map[string]*finspaceDataView
	changesets       map[string]map[string]*finspaceChangeset
	users            map[string]*finspaceUser
	permissionGroups map[string]*finspacePermissionGroup
	groupUsers       map[string]map[string]struct{}
	userGroups       map[string]map[string]struct{}
}

type finspaceDataset struct {
	ID          string
	Title       string
	Kind        string
	Description string
	Status      string
	CreatedAt   string
	UpdatedAt   string
}

type finspaceDataView struct {
	ID        string
	DatasetID string
	Name      string
	Status    string
	CreatedAt string
	UpdatedAt string
}

type finspaceChangeset struct {
	ID         string
	DatasetID  string
	ChangeType string
	Status     string
	CreatedAt  string
	UpdatedAt  string
}

type finspaceUser struct {
	ID         string
	Email      string
	FirstName  string
	LastName   string
	APIAccess  string
	Status     string
	CreatedAt  string
	UpdatedAt  string
	LastPwdSet string
}

type finspacePermissionGroup struct {
	ID          string
	Name        string
	Description string
	CreatedAt   string
	UpdatedAt   string
}

func newFinSpaceStore() *finspaceStore {
	now := time.Now().UTC().Format(time.RFC3339)

	dataset := &finspaceDataset{
		ID:          "dataset-000001",
		Title:       "stackyard-finspace-dataset",
		Kind:        "TABULAR",
		Description: "stackyard dataset",
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	dataView := &finspaceDataView{
		ID:        "dataview-000001",
		DatasetID: dataset.ID,
		Name:      "stackyard-data-view",
		Status:    "SUCCESS",
		CreatedAt: now,
		UpdatedAt: now,
	}
	changeset := &finspaceChangeset{
		ID:         "changeset-000001",
		DatasetID:  dataset.ID,
		ChangeType: "REPLACE",
		Status:     "SUCCESS",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	user := &finspaceUser{
		ID:         "user-000001",
		Email:      "stackyard@example.com",
		FirstName:  "Stackyard",
		LastName:   "User",
		APIAccess:  "ENABLED",
		Status:     "ENABLED",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastPwdSet: now,
	}
	group := &finspacePermissionGroup{
		ID:          "permission-group-000001",
		Name:        "stackyard-default-group",
		Description: "Default Stackyard FinSpace permission group",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	groupUsers := map[string]map[string]struct{}{
		group.ID: {user.ID: {}},
	}
	userGroups := map[string]map[string]struct{}{
		user.ID: {group.ID: {}},
	}

	return &finspaceStore{
		nextDatasetID:         2,
		nextDataViewID:        2,
		nextChangesetID:       2,
		nextUserID:            2,
		nextPermissionGroupID: 2,
		datasets: map[string]*finspaceDataset{
			dataset.ID: dataset,
		},
		dataViews: map[string]map[string]*finspaceDataView{
			dataset.ID: {
				dataView.ID: dataView,
			},
		},
		changesets: map[string]map[string]*finspaceChangeset{
			dataset.ID: {
				changeset.ID: changeset,
			},
		},
		users: map[string]*finspaceUser{
			user.ID: user,
		},
		permissionGroups: map[string]*finspacePermissionGroup{
			group.ID: group,
		},
		groupUsers: groupUsers,
		userGroups: userGroups,
	}
}

func (s *finspaceStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := finspaceMergeMaps(payload, pathParams, query)
	datasetID := finspaceStringAny(ctx, []string{"datasetId", "id"}, "dataset-000001")
	dataViewID := finspaceStringAny(ctx, []string{"dataviewId", "dataViewId", "id"}, "dataview-000001")
	changesetID := finspaceStringAny(ctx, []string{"changesetId", "id"}, "changeset-000001")
	userID := finspaceStringAny(ctx, []string{"userId", "id"}, "user-000001")
	permissionGroupID := finspaceStringAny(ctx, []string{"permissionGroupId", "id"}, "permission-group-000001")
	now := time.Now().UTC().Format(time.RFC3339)

	s.ensureDatasetLocked(datasetID)
	s.ensureUserLocked(userID)
	s.ensurePermissionGroupLocked(permissionGroupID)

	switch action {
	case "AssociateUserToPermissionGroup":
		s.linkMembershipLocked(userID, permissionGroupID)
		return map[string]any{}

	case "DisassociateUserFromPermissionGroup":
		s.unlinkMembershipLocked(userID, permissionGroupID)
		return map[string]any{}

	case "CreateDataset":
		id := s.nextIDLocked("dataset")
		dataset := &finspaceDataset{
			ID:          id,
			Title:       finspaceStringAny(payload, []string{"datasetTitle", "title", "name"}, "stackyard-finspace-dataset"),
			Kind:        finspaceStringAny(payload, []string{"kind"}, "TABULAR"),
			Description: finspaceStringAny(payload, []string{"description"}, "stackyard dataset"),
			Status:      "ACTIVE",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.datasets[id] = dataset
		return map[string]any{"datasetId": dataset.ID}

	case "DeleteDataset":
		dataset := s.ensureDatasetLocked(datasetID)
		dataset.Status = "DELETED"
		dataset.UpdatedAt = now
		return map[string]any{}

	case "GetDataset":
		return s.datasetPayload(s.ensureDatasetLocked(datasetID))

	case "ListDatasets":
		items := make([]any, 0, len(s.datasets))
		for _, id := range s.sortedDatasetIDsLocked() {
			items = append(items, s.datasetSummaryPayload(s.datasets[id]))
		}
		return map[string]any{"datasets": items, "nextToken": ""}

	case "CreateDataView":
		dataset := s.ensureDatasetLocked(datasetID)
		id := s.nextIDLocked("dataview")
		view := &finspaceDataView{
			ID:        id,
			DatasetID: dataset.ID,
			Name:      finspaceStringAny(payload, []string{"name", "dataViewName"}, "stackyard-data-view"),
			Status:    "RUNNING",
			CreatedAt: now,
			UpdatedAt: now,
		}
		s.ensureDataViewMapLocked(dataset.ID)
		s.dataViews[dataset.ID][id] = view
		return map[string]any{"datasetId": dataset.ID, "dataViewId": view.ID}

	case "GetDataView":
		view := s.ensureDataViewLocked(datasetID, dataViewID)
		return s.dataViewPayload(view)

	case "ListDataViews":
		s.ensureDatasetLocked(datasetID)
		s.ensureDataViewMapLocked(datasetID)
		items := make([]any, 0, len(s.dataViews[datasetID]))
		for _, id := range s.sortedDataViewIDsLocked(datasetID) {
			items = append(items, s.dataViewSummaryPayload(s.dataViews[datasetID][id]))
		}
		return map[string]any{"dataViews": items, "nextToken": ""}

	case "GetExternalDataViewAccessDetails":
		view := s.ensureDataViewLocked(datasetID, dataViewID)
		return map[string]any{
			"dataViewId": view.ID,
			"datasetId":  view.DatasetID,
			"s3Path":     fmt.Sprintf("s3://stackyard-finspace/datasets/%s/dataviews/%s", view.DatasetID, view.ID),
			"credentials": map[string]any{
				"accessKeyId":     "ASIASTACKYARDFINSPACE",
				"secretAccessKey": "stackyard",
				"sessionToken":    "stackyard-token",
				"expirationTime":  time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
			},
		}

	case "CreateChangeset":
		dataset := s.ensureDatasetLocked(datasetID)
		id := s.nextIDLocked("changeset")
		changeset := &finspaceChangeset{
			ID:         id,
			DatasetID:  dataset.ID,
			ChangeType: finspaceStringAny(payload, []string{"changeType"}, "REPLACE"),
			Status:     "PENDING",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		s.ensureChangesetMapLocked(dataset.ID)
		s.changesets[dataset.ID][id] = changeset
		return map[string]any{"datasetId": dataset.ID, "changesetId": changeset.ID}

	case "GetChangeset":
		return s.changesetPayload(s.ensureChangesetLocked(datasetID, changesetID))

	case "ListChangesets":
		s.ensureDatasetLocked(datasetID)
		s.ensureChangesetMapLocked(datasetID)
		items := make([]any, 0, len(s.changesets[datasetID]))
		for _, id := range s.sortedChangesetIDsLocked(datasetID) {
			items = append(items, s.changesetSummaryPayload(s.changesets[datasetID][id]))
		}
		return map[string]any{"changesets": items, "nextToken": ""}

	case "UpdateChangeset":
		changeset := s.ensureChangesetLocked(datasetID, changesetID)
		if status := finspaceStringAny(payload, []string{"status"}, ""); status != "" {
			changeset.Status = strings.ToUpper(status)
		} else {
			changeset.Status = "SUCCESS"
		}
		changeset.UpdatedAt = now
		return s.changesetPayload(changeset)

	case "UpdateDataset":
		dataset := s.ensureDatasetLocked(datasetID)
		if title := finspaceStringAny(payload, []string{"datasetTitle", "title", "name"}, ""); title != "" {
			dataset.Title = title
		}
		if kind := finspaceStringAny(payload, []string{"kind"}, ""); kind != "" {
			dataset.Kind = kind
		}
		if desc := finspaceStringAny(payload, []string{"description"}, ""); desc != "" {
			dataset.Description = desc
		}
		dataset.UpdatedAt = now
		return s.datasetPayload(dataset)

	case "CreatePermissionGroup":
		id := s.nextIDLocked("permissiongroup")
		group := &finspacePermissionGroup{
			ID:          id,
			Name:        finspaceStringAny(payload, []string{"name"}, "stackyard-permission-group"),
			Description: finspaceStringAny(payload, []string{"description"}, "Stackyard FinSpace permission group"),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.permissionGroups[id] = group
		return map[string]any{"permissionGroupId": group.ID}

	case "DeletePermissionGroup":
		delete(s.permissionGroups, permissionGroupID)
		delete(s.groupUsers, permissionGroupID)
		for _, groups := range s.userGroups {
			delete(groups, permissionGroupID)
		}
		return map[string]any{}

	case "GetPermissionGroup":
		return s.permissionGroupPayload(s.ensurePermissionGroupLocked(permissionGroupID))

	case "ListPermissionGroups":
		items := make([]any, 0, len(s.permissionGroups))
		for _, id := range s.sortedPermissionGroupIDsLocked() {
			items = append(items, s.permissionGroupSummaryPayload(s.permissionGroups[id]))
		}
		return map[string]any{"permissionGroups": items, "nextToken": ""}

	case "ListPermissionGroupsByUser":
		s.ensureUserLocked(userID)
		groups := s.userGroups[userID]
		items := make([]any, 0, len(groups))
		for _, id := range s.sortedMembershipIDsLocked(groups) {
			items = append(items, s.permissionGroupSummaryPayload(s.ensurePermissionGroupLocked(id)))
		}
		return map[string]any{"permissionGroups": items, "nextToken": ""}

	case "ListUsersByPermissionGroup":
		s.ensurePermissionGroupLocked(permissionGroupID)
		users := s.groupUsers[permissionGroupID]
		items := make([]any, 0, len(users))
		for _, id := range s.sortedMembershipIDsLocked(users) {
			items = append(items, s.userSummaryPayload(s.ensureUserLocked(id)))
		}
		return map[string]any{"users": items, "nextToken": ""}

	case "UpdatePermissionGroup":
		group := s.ensurePermissionGroupLocked(permissionGroupID)
		if name := finspaceStringAny(payload, []string{"name"}, ""); name != "" {
			group.Name = name
		}
		if desc := finspaceStringAny(payload, []string{"description"}, ""); desc != "" {
			group.Description = desc
		}
		group.UpdatedAt = now
		return s.permissionGroupPayload(group)

	case "CreateUser":
		id := s.nextIDLocked("user")
		user := &finspaceUser{
			ID:         id,
			Email:      finspaceStringAny(payload, []string{"emailAddress", "email"}, "stackyard@example.com"),
			FirstName:  finspaceStringAny(payload, []string{"firstName"}, "Stackyard"),
			LastName:   finspaceStringAny(payload, []string{"lastName"}, "User"),
			APIAccess:  strings.ToUpper(finspaceStringAny(payload, []string{"apiAccess"}, "ENABLED")),
			Status:     "ENABLED",
			CreatedAt:  now,
			UpdatedAt:  now,
			LastPwdSet: now,
		}
		s.users[id] = user
		return map[string]any{"userId": user.ID}

	case "GetUser":
		return s.userPayload(s.ensureUserLocked(userID))

	case "ListUsers":
		items := make([]any, 0, len(s.users))
		for _, id := range s.sortedUserIDsLocked() {
			items = append(items, s.userSummaryPayload(s.users[id]))
		}
		return map[string]any{"users": items, "nextToken": ""}

	case "UpdateUser":
		user := s.ensureUserLocked(userID)
		if email := finspaceStringAny(payload, []string{"emailAddress", "email"}, ""); email != "" {
			user.Email = email
		}
		if first := finspaceStringAny(payload, []string{"firstName"}, ""); first != "" {
			user.FirstName = first
		}
		if last := finspaceStringAny(payload, []string{"lastName"}, ""); last != "" {
			user.LastName = last
		}
		if access := finspaceStringAny(payload, []string{"apiAccess"}, ""); access != "" {
			user.APIAccess = strings.ToUpper(access)
		}
		user.UpdatedAt = now
		return s.userPayload(user)

	case "DisableUser":
		user := s.ensureUserLocked(userID)
		user.Status = "DISABLED"
		user.UpdatedAt = now
		return map[string]any{}

	case "EnableUser":
		user := s.ensureUserLocked(userID)
		user.Status = "ENABLED"
		user.UpdatedAt = now
		return map[string]any{}

	case "ResetUserPassword":
		user := s.ensureUserLocked(userID)
		user.LastPwdSet = now
		user.UpdatedAt = now
		return map[string]any{}

	case "GetProgrammaticAccessCredentials":
		return map[string]any{
			"credentials": map[string]any{
				"accessKeyId":     "ASIASTACKYARDFINSPACE",
				"secretAccessKey": "stackyard",
				"sessionToken":    "stackyard-token",
				"expirationTime":  time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
			},
		}

	case "GetWorkingLocation":
		return map[string]any{
			"s3Uri":        "s3://stackyard-finspace/working-location/",
			"locationType": finspaceStringAny(payload, []string{"locationType"}, "SAGEMAKER"),
		}
	}

	if strings.HasPrefix(action, "Get") {
		return map[string]any{"status": "ACTIVE", "action": action}
	}
	if strings.HasPrefix(action, "List") {
		return map[string]any{"items": []any{}, "nextToken": ""}
	}
	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") {
		return map[string]any{"status": "SUCCESS", "action": action}
	}
	if strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Enable") || strings.HasPrefix(action, "Disable") || strings.HasPrefix(action, "Reset") || strings.HasPrefix(action, "Associate") || strings.HasPrefix(action, "Disassociate") {
		return map[string]any{}
	}
	return map[string]any{}
}

func (s *finspaceStore) nextIDLocked(kind string) string {
	switch kind {
	case "dataset":
		id := fmt.Sprintf("dataset-%06d", s.nextDatasetID)
		s.nextDatasetID++
		return id
	case "dataview":
		id := fmt.Sprintf("dataview-%06d", s.nextDataViewID)
		s.nextDataViewID++
		return id
	case "changeset":
		id := fmt.Sprintf("changeset-%06d", s.nextChangesetID)
		s.nextChangesetID++
		return id
	case "user":
		id := fmt.Sprintf("user-%06d", s.nextUserID)
		s.nextUserID++
		return id
	case "permissiongroup":
		id := fmt.Sprintf("permission-group-%06d", s.nextPermissionGroupID)
		s.nextPermissionGroupID++
		return id
	default:
		id := fmt.Sprintf("id-%06d", s.nextDatasetID)
		s.nextDatasetID++
		return id
	}
}

func (s *finspaceStore) ensureDatasetLocked(datasetID string) *finspaceDataset {
	id := strings.TrimSpace(datasetID)
	if id == "" {
		id = "dataset-000001"
	}
	if existing := s.datasets[id]; existing != nil {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &finspaceDataset{
		ID:          id,
		Title:       "stackyard-finspace-dataset",
		Kind:        "TABULAR",
		Description: "stackyard dataset",
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.datasets[id] = created
	return created
}

func (s *finspaceStore) ensureDataViewMapLocked(datasetID string) {
	if s.dataViews[datasetID] == nil {
		s.dataViews[datasetID] = map[string]*finspaceDataView{}
	}
}

func (s *finspaceStore) ensureDataViewLocked(datasetID, dataViewID string) *finspaceDataView {
	dataset := s.ensureDatasetLocked(datasetID)
	s.ensureDataViewMapLocked(dataset.ID)
	id := strings.TrimSpace(dataViewID)
	if id == "" {
		id = "dataview-000001"
	}
	if existing := s.dataViews[dataset.ID][id]; existing != nil {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &finspaceDataView{
		ID:        id,
		DatasetID: dataset.ID,
		Name:      "stackyard-data-view",
		Status:    "SUCCESS",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.dataViews[dataset.ID][id] = created
	return created
}

func (s *finspaceStore) ensureChangesetMapLocked(datasetID string) {
	if s.changesets[datasetID] == nil {
		s.changesets[datasetID] = map[string]*finspaceChangeset{}
	}
}

func (s *finspaceStore) ensureChangesetLocked(datasetID, changesetID string) *finspaceChangeset {
	dataset := s.ensureDatasetLocked(datasetID)
	s.ensureChangesetMapLocked(dataset.ID)
	id := strings.TrimSpace(changesetID)
	if id == "" {
		id = "changeset-000001"
	}
	if existing := s.changesets[dataset.ID][id]; existing != nil {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &finspaceChangeset{
		ID:         id,
		DatasetID:  dataset.ID,
		ChangeType: "REPLACE",
		Status:     "SUCCESS",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.changesets[dataset.ID][id] = created
	return created
}

func (s *finspaceStore) ensureUserLocked(userID string) *finspaceUser {
	id := strings.TrimSpace(userID)
	if id == "" {
		id = "user-000001"
	}
	if existing := s.users[id]; existing != nil {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &finspaceUser{
		ID:         id,
		Email:      "stackyard@example.com",
		FirstName:  "Stackyard",
		LastName:   "User",
		APIAccess:  "ENABLED",
		Status:     "ENABLED",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastPwdSet: now,
	}
	s.users[id] = created
	return created
}

func (s *finspaceStore) ensurePermissionGroupLocked(permissionGroupID string) *finspacePermissionGroup {
	id := strings.TrimSpace(permissionGroupID)
	if id == "" {
		id = "permission-group-000001"
	}
	if existing := s.permissionGroups[id]; existing != nil {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &finspacePermissionGroup{
		ID:          id,
		Name:        "stackyard-default-group",
		Description: "Stackyard FinSpace permission group",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.permissionGroups[id] = created
	return created
}

func (s *finspaceStore) linkMembershipLocked(userID, permissionGroupID string) {
	s.ensureUserLocked(userID)
	s.ensurePermissionGroupLocked(permissionGroupID)
	if s.groupUsers[permissionGroupID] == nil {
		s.groupUsers[permissionGroupID] = map[string]struct{}{}
	}
	s.groupUsers[permissionGroupID][userID] = struct{}{}
	if s.userGroups[userID] == nil {
		s.userGroups[userID] = map[string]struct{}{}
	}
	s.userGroups[userID][permissionGroupID] = struct{}{}
}

func (s *finspaceStore) unlinkMembershipLocked(userID, permissionGroupID string) {
	if groupMembers := s.groupUsers[permissionGroupID]; groupMembers != nil {
		delete(groupMembers, userID)
	}
	if userGroups := s.userGroups[userID]; userGroups != nil {
		delete(userGroups, permissionGroupID)
	}
}

func (s *finspaceStore) sortedDatasetIDsLocked() []string {
	out := make([]string, 0, len(s.datasets))
	for id := range s.datasets {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *finspaceStore) sortedDataViewIDsLocked(datasetID string) []string {
	views := s.dataViews[datasetID]
	out := make([]string, 0, len(views))
	for id := range views {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *finspaceStore) sortedChangesetIDsLocked(datasetID string) []string {
	changesets := s.changesets[datasetID]
	out := make([]string, 0, len(changesets))
	for id := range changesets {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *finspaceStore) sortedUserIDsLocked() []string {
	out := make([]string, 0, len(s.users))
	for id := range s.users {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *finspaceStore) sortedPermissionGroupIDsLocked() []string {
	out := make([]string, 0, len(s.permissionGroups))
	for id := range s.permissionGroups {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *finspaceStore) sortedMembershipIDsLocked(ids map[string]struct{}) []string {
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *finspaceStore) datasetPayload(dataset *finspaceDataset) map[string]any {
	return map[string]any{
		"datasetId":    dataset.ID,
		"datasetArn":   fmt.Sprintf("arn:aws:finspace:us-east-1:123456789012:dataset/%s", dataset.ID),
		"datasetTitle": dataset.Title,
		"kind":         dataset.Kind,
		"description":  dataset.Description,
		"status":       dataset.Status,
		"createTime":   dataset.CreatedAt,
		"lastModified": dataset.UpdatedAt,
	}
}

func (s *finspaceStore) datasetSummaryPayload(dataset *finspaceDataset) map[string]any {
	return map[string]any{
		"datasetId":    dataset.ID,
		"datasetTitle": dataset.Title,
		"kind":         dataset.Kind,
		"status":       dataset.Status,
		"lastModified": dataset.UpdatedAt,
	}
}

func (s *finspaceStore) dataViewPayload(view *finspaceDataView) map[string]any {
	return map[string]any{
		"dataViewId":   view.ID,
		"datasetId":    view.DatasetID,
		"status":       view.Status,
		"dataViewName": view.Name,
		"createTime":   view.CreatedAt,
		"lastModified": view.UpdatedAt,
	}
}

func (s *finspaceStore) dataViewSummaryPayload(view *finspaceDataView) map[string]any {
	return map[string]any{
		"dataViewId":   view.ID,
		"datasetId":    view.DatasetID,
		"status":       view.Status,
		"dataViewName": view.Name,
		"lastModified": view.UpdatedAt,
	}
}

func (s *finspaceStore) changesetPayload(changeset *finspaceChangeset) map[string]any {
	return map[string]any{
		"changesetId":  changeset.ID,
		"datasetId":    changeset.DatasetID,
		"changeType":   changeset.ChangeType,
		"status":       changeset.Status,
		"createTime":   changeset.CreatedAt,
		"lastModified": changeset.UpdatedAt,
	}
}

func (s *finspaceStore) changesetSummaryPayload(changeset *finspaceChangeset) map[string]any {
	return map[string]any{
		"changesetId":  changeset.ID,
		"changeType":   changeset.ChangeType,
		"status":       changeset.Status,
		"lastModified": changeset.UpdatedAt,
	}
}

func (s *finspaceStore) userPayload(user *finspaceUser) map[string]any {
	return map[string]any{
		"userId":       user.ID,
		"emailAddress": user.Email,
		"firstName":    user.FirstName,
		"lastName":     user.LastName,
		"apiAccess":    user.APIAccess,
		"status":       user.Status,
		"createTime":   user.CreatedAt,
		"lastModified": user.UpdatedAt,
	}
}

func (s *finspaceStore) userSummaryPayload(user *finspaceUser) map[string]any {
	return map[string]any{
		"userId":       user.ID,
		"emailAddress": user.Email,
		"status":       user.Status,
		"apiAccess":    user.APIAccess,
		"lastModified": user.UpdatedAt,
	}
}

func (s *finspaceStore) permissionGroupPayload(group *finspacePermissionGroup) map[string]any {
	return map[string]any{
		"permissionGroupId": group.ID,
		"name":              group.Name,
		"description":       group.Description,
		"createTime":        group.CreatedAt,
		"lastModified":      group.UpdatedAt,
	}
}

func (s *finspaceStore) permissionGroupSummaryPayload(group *finspacePermissionGroup) map[string]any {
	return map[string]any{
		"permissionGroupId": group.ID,
		"name":              group.Name,
		"description":       group.Description,
		"lastModified":      group.UpdatedAt,
	}
}

func finspaceMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			out[key] = values[0]
			continue
		}
		dup := make([]string, len(values))
		copy(dup, values)
		out[key] = dup
	}
	return out
}

func finspaceStringAny(values map[string]any, keys []string, def string) string {
	if values == nil {
		return def
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if raw, ok := finspaceLookupInsensitive(values, key); ok && raw != nil {
			text := strings.TrimSpace(fmt.Sprint(raw))
			if text != "" {
				return text
			}
		}
	}
	return def
}

func finspaceLookupInsensitive(values map[string]any, key string) (any, bool) {
	for candidate, value := range values {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(key)) {
			return value, true
		}
	}
	return nil, false
}
