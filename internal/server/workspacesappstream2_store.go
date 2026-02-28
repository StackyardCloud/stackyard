package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	workspacesAppStream2Region    = "us-east-1"
	workspacesAppStream2AccountID = "123456789012"
)

type workspacesAppStream2Store struct {
	mu sync.Mutex

	nextID int64

	fleets       map[string]map[string]any
	stacks       map[string]map[string]any
	images       map[string]map[string]any
	builders     map[string]map[string]any
	applications map[string]map[string]any

	appBlocks        map[string]map[string]any
	appBlockBuilders map[string]map[string]any
	directoryConfigs map[string]map[string]any
	entitlements     map[string]map[string]any
	users            map[string]map[string]any

	exportImageTasks map[string]map[string]any
	usageReports     map[string]map[string]any
	sessions         map[string]map[string]any
	tags             map[string]map[string]string

	stackToFleets          map[string]map[string]struct{}
	fleetToStacks          map[string]map[string]struct{}
	applicationToFleets    map[string]map[string]struct{}
	entitlementToApps      map[string]map[string]struct{}
	builderToAppBlocks     map[string]map[string]struct{}
	imageBuilderToSoftware map[string]map[string]struct{}
	userStackAssociations  []map[string]any
}

func newWorkSpacesAppStream2Store() *workspacesAppStream2Store {
	s := &workspacesAppStream2Store{
		nextID:                 2,
		fleets:                 map[string]map[string]any{},
		stacks:                 map[string]map[string]any{},
		images:                 map[string]map[string]any{},
		builders:               map[string]map[string]any{},
		applications:           map[string]map[string]any{},
		appBlocks:              map[string]map[string]any{},
		appBlockBuilders:       map[string]map[string]any{},
		directoryConfigs:       map[string]map[string]any{},
		entitlements:           map[string]map[string]any{},
		users:                  map[string]map[string]any{},
		exportImageTasks:       map[string]map[string]any{},
		usageReports:           map[string]map[string]any{},
		sessions:               map[string]map[string]any{},
		tags:                   map[string]map[string]string{},
		stackToFleets:          map[string]map[string]struct{}{},
		fleetToStacks:          map[string]map[string]struct{}{},
		applicationToFleets:    map[string]map[string]struct{}{},
		entitlementToApps:      map[string]map[string]struct{}{},
		builderToAppBlocks:     map[string]map[string]struct{}{},
		imageBuilderToSoftware: map[string]map[string]struct{}{},
	}
	s.seedLocked()
	return s
}

func (s *workspacesAppStream2Store) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	s.seedLocked()

	switch action {
	case "DescribeFleets":
		return map[string]any{"Fleets": s.listResourceMapsLocked(s.fleets), "NextToken": ""}
	case "DescribeStacks":
		return map[string]any{"Stacks": s.listResourceMapsLocked(s.stacks), "NextToken": ""}
	case "DescribeImages":
		return map[string]any{"Images": s.listResourceMapsLocked(s.images), "NextToken": ""}
	case "DescribeImageBuilders":
		return map[string]any{"ImageBuilders": s.listResourceMapsLocked(s.builders), "NextToken": ""}
	case "DescribeApplications":
		return map[string]any{"Applications": s.listResourceMapsLocked(s.applications), "NextToken": ""}
	case "DescribeAppBlocks":
		return map[string]any{"AppBlocks": s.listResourceMapsLocked(s.appBlocks), "NextToken": ""}
	case "DescribeAppBlockBuilders":
		return map[string]any{"AppBlockBuilders": s.listResourceMapsLocked(s.appBlockBuilders), "NextToken": ""}
	case "DescribeDirectoryConfigs":
		return map[string]any{"DirectoryConfigs": s.listResourceMapsLocked(s.directoryConfigs), "NextToken": ""}
	case "DescribeEntitlements":
		return map[string]any{"Entitlements": s.listResourceMapsLocked(s.entitlements), "NextToken": ""}
	case "DescribeUsers":
		return map[string]any{"Users": s.listResourceMapsLocked(s.users), "NextToken": ""}
	case "DescribeUsageReportSubscriptions":
		return map[string]any{"UsageReportSubscriptions": s.listResourceMapsLocked(s.usageReports)}
	case "DescribeSessions":
		return map[string]any{"Sessions": s.listResourceMapsLocked(s.sessions), "NextToken": ""}
	case "DescribeThemeForStack":
		stackName := workspacesAppStream2LookupString(payload, "StackName", "Name")
		if stackName == "" {
			stackName = "stackyard-stack"
		}
		return map[string]any{
			"Theme": map[string]any{
				"StackName": stackName,
				"ThemeStyling": map[string]any{
					"PrimaryColor":    "#2D4D9E",
					"BackgroundColor": "#FFFFFF",
				},
			},
		}

	case "CreateFleet":
		name := workspacesAppStream2LookupString(payload, "Name", "FleetName")
		if name == "" {
			name = s.nextTokenLocked("fleet")
		}
		resource := s.ensureFleetLocked(name, now)
		return map[string]any{"Fleet": workspacesAppStream2CloneMap(resource)}
	case "CreateStack":
		name := workspacesAppStream2LookupString(payload, "Name", "StackName")
		if name == "" {
			name = s.nextTokenLocked("stack")
		}
		resource := s.ensureStackLocked(name, now)
		return map[string]any{"Stack": workspacesAppStream2CloneMap(resource)}
	case "CreateApplication":
		name := workspacesAppStream2LookupString(payload, "Name", "ApplicationName")
		if name == "" {
			name = s.nextTokenLocked("application")
		}
		resource := s.ensureApplicationLocked(name, now)
		return map[string]any{"Application": workspacesAppStream2CloneMap(resource)}
	case "CreateAppBlock":
		name := workspacesAppStream2LookupString(payload, "Name", "AppBlockName")
		if name == "" {
			name = s.nextTokenLocked("appblock")
		}
		resource := s.ensureAppBlockLocked(name, now)
		return map[string]any{"AppBlock": workspacesAppStream2CloneMap(resource)}
	case "CreateAppBlockBuilder":
		name := workspacesAppStream2LookupString(payload, "Name", "AppBlockBuilderName")
		if name == "" {
			name = s.nextTokenLocked("appblockbuilder")
		}
		resource := s.ensureAppBlockBuilderLocked(name, now)
		return map[string]any{"AppBlockBuilder": workspacesAppStream2CloneMap(resource)}
	case "CreateDirectoryConfig":
		name := workspacesAppStream2LookupString(payload, "DirectoryName", "Name")
		if name == "" {
			name = s.nextTokenLocked("directory")
		}
		resource := s.ensureDirectoryConfigLocked(name, now)
		return map[string]any{"DirectoryConfig": workspacesAppStream2CloneMap(resource)}
	case "CreateEntitlement":
		name := workspacesAppStream2LookupString(payload, "Name", "EntitlementName")
		if name == "" {
			name = s.nextTokenLocked("entitlement")
		}
		resource := s.ensureEntitlementLocked(name, now)
		return map[string]any{"Entitlement": workspacesAppStream2CloneMap(resource)}
	case "CreateUser":
		userName := workspacesAppStream2LookupString(payload, "UserName", "Name")
		if userName == "" {
			userName = "stackyard-user"
		}
		authType := workspacesAppStream2LookupString(payload, "AuthenticationType")
		if authType == "" {
			authType = "API"
		}
		resource := s.ensureUserLocked(userName, authType, now)
		return map[string]any{"User": workspacesAppStream2CloneMap(resource)}
	case "CreateUsageReportSubscription":
		name := workspacesAppStream2LookupString(payload, "S3BucketName", "Name")
		if name == "" {
			name = "stackyard-reports"
		}
		resource := s.ensureUsageReportLocked(name, now)
		return map[string]any{"S3BucketName": resource["S3BucketName"]}
	case "CreateExportImageTask":
		taskID := s.nextTokenLocked("export-task")
		task := map[string]any{
			"ExportImageTaskId": taskID,
			"Status":            "COMPLETED",
			"CreatedTime":       now,
			"ImageName":         workspacesAppStream2LookupString(payload, "ImageName", "Name"),
		}
		if task["ImageName"] == "" {
			task["ImageName"] = "stackyard-image"
		}
		s.exportImageTasks[taskID] = task
		return map[string]any{"ExportImageTask": workspacesAppStream2CloneMap(task)}
	case "CreateImportedImage", "CreateUpdatedImage", "CopyImage":
		name := workspacesAppStream2LookupString(payload, "Name", "ImageName")
		if name == "" {
			name = s.nextTokenLocked("image")
		}
		resource := s.ensureImageLocked(name, now)
		return map[string]any{"Image": workspacesAppStream2CloneMap(resource)}
	case "CreateImageBuilder":
		name := workspacesAppStream2LookupString(payload, "Name", "ImageBuilderName")
		if name == "" {
			name = s.nextTokenLocked("builder")
		}
		resource := s.ensureImageBuilderLocked(name, now)
		return map[string]any{"ImageBuilder": workspacesAppStream2CloneMap(resource)}

	case "DeleteFleet":
		delete(s.fleets, workspacesAppStream2LookupString(payload, "Name", "FleetName"))
		return map[string]any{}
	case "DeleteStack":
		delete(s.stacks, workspacesAppStream2LookupString(payload, "Name", "StackName"))
		return map[string]any{}
	case "DeleteApplication":
		delete(s.applications, workspacesAppStream2LookupString(payload, "Name", "ApplicationName"))
		return map[string]any{}
	case "DeleteAppBlock":
		delete(s.appBlocks, workspacesAppStream2LookupString(payload, "Name", "AppBlockName"))
		return map[string]any{}
	case "DeleteAppBlockBuilder":
		delete(s.appBlockBuilders, workspacesAppStream2LookupString(payload, "Name", "AppBlockBuilderName"))
		return map[string]any{}
	case "DeleteDirectoryConfig":
		delete(s.directoryConfigs, workspacesAppStream2LookupString(payload, "DirectoryName", "Name"))
		return map[string]any{}
	case "DeleteEntitlement":
		delete(s.entitlements, workspacesAppStream2LookupString(payload, "Name", "EntitlementName"))
		return map[string]any{}
	case "DeleteImage":
		delete(s.images, workspacesAppStream2LookupString(payload, "Name", "ImageName"))
		return map[string]any{}
	case "DeleteImageBuilder":
		delete(s.builders, workspacesAppStream2LookupString(payload, "Name", "ImageBuilderName"))
		return map[string]any{}
	case "DeleteUsageReportSubscription":
		delete(s.usageReports, workspacesAppStream2LookupString(payload, "S3BucketName", "Name"))
		return map[string]any{}
	case "DeleteUser":
		userName := workspacesAppStream2LookupString(payload, "UserName", "Name")
		authType := workspacesAppStream2LookupString(payload, "AuthenticationType")
		delete(s.users, workspacesAppStream2UserKey(userName, authType))
		return map[string]any{}
	case "DeleteImagePermissions", "DeleteThemeForStack":
		return map[string]any{}

	case "UpdateFleet":
		name := workspacesAppStream2LookupString(payload, "Name", "FleetName")
		resource := s.ensureFleetLocked(name, now)
		resource["LastUpdatedTime"] = now
		return map[string]any{"Fleet": workspacesAppStream2CloneMap(resource)}
	case "UpdateStack":
		name := workspacesAppStream2LookupString(payload, "Name", "StackName")
		resource := s.ensureStackLocked(name, now)
		resource["LastUpdatedTime"] = now
		return map[string]any{"Stack": workspacesAppStream2CloneMap(resource)}
	case "UpdateApplication":
		name := workspacesAppStream2LookupString(payload, "Name", "ApplicationName")
		resource := s.ensureApplicationLocked(name, now)
		resource["LastUpdatedTime"] = now
		return map[string]any{"Application": workspacesAppStream2CloneMap(resource)}
	case "UpdateAppBlockBuilder":
		name := workspacesAppStream2LookupString(payload, "Name", "AppBlockBuilderName")
		resource := s.ensureAppBlockBuilderLocked(name, now)
		resource["LastUpdatedTime"] = now
		return map[string]any{"AppBlockBuilder": workspacesAppStream2CloneMap(resource)}
	case "UpdateDirectoryConfig":
		name := workspacesAppStream2LookupString(payload, "DirectoryName", "Name")
		resource := s.ensureDirectoryConfigLocked(name, now)
		resource["LastUpdatedTime"] = now
		return map[string]any{"DirectoryConfig": workspacesAppStream2CloneMap(resource)}
	case "UpdateEntitlement":
		name := workspacesAppStream2LookupString(payload, "Name", "EntitlementName")
		resource := s.ensureEntitlementLocked(name, now)
		resource["LastUpdatedTime"] = now
		return map[string]any{"Entitlement": workspacesAppStream2CloneMap(resource)}
	case "UpdateImagePermissions", "UpdateThemeForStack":
		return map[string]any{}

	case "AssociateFleet":
		stackName := workspacesAppStream2LookupString(payload, "StackName", "Name")
		fleetName := workspacesAppStream2LookupString(payload, "FleetName")
		s.associateStackFleetLocked(stackName, fleetName, now)
		return map[string]any{}
	case "DisassociateFleet":
		stackName := workspacesAppStream2LookupString(payload, "StackName", "Name")
		fleetName := workspacesAppStream2LookupString(payload, "FleetName")
		s.disassociateStackFleetLocked(stackName, fleetName)
		return map[string]any{}
	case "AssociateApplicationFleet":
		appName := workspacesAppStream2LookupString(payload, "ApplicationName", "Name")
		fleetName := workspacesAppStream2LookupString(payload, "FleetName")
		if s.applicationToFleets[appName] == nil {
			s.applicationToFleets[appName] = map[string]struct{}{}
		}
		s.applicationToFleets[appName][fleetName] = struct{}{}
		s.ensureApplicationLocked(appName, now)
		s.ensureFleetLocked(fleetName, now)
		return map[string]any{}
	case "DisassociateApplicationFleet":
		appName := workspacesAppStream2LookupString(payload, "ApplicationName", "Name")
		fleetName := workspacesAppStream2LookupString(payload, "FleetName")
		if s.applicationToFleets[appName] != nil {
			delete(s.applicationToFleets[appName], fleetName)
		}
		return map[string]any{}
	case "AssociateApplicationToEntitlement":
		appName := workspacesAppStream2LookupString(payload, "ApplicationIdentifier", "ApplicationName", "Name")
		entitlementName := workspacesAppStream2LookupString(payload, "EntitlementName", "Name")
		if s.entitlementToApps[entitlementName] == nil {
			s.entitlementToApps[entitlementName] = map[string]struct{}{}
		}
		s.entitlementToApps[entitlementName][appName] = struct{}{}
		s.ensureEntitlementLocked(entitlementName, now)
		s.ensureApplicationLocked(appName, now)
		return map[string]any{}
	case "DisassociateApplicationFromEntitlement":
		appName := workspacesAppStream2LookupString(payload, "ApplicationIdentifier", "ApplicationName", "Name")
		entitlementName := workspacesAppStream2LookupString(payload, "EntitlementName", "Name")
		if s.entitlementToApps[entitlementName] != nil {
			delete(s.entitlementToApps[entitlementName], appName)
		}
		return map[string]any{}
	case "AssociateAppBlockBuilderAppBlock":
		builderName := workspacesAppStream2LookupString(payload, "AppBlockBuilderName", "Name")
		appBlockName := workspacesAppStream2LookupString(payload, "AppBlockName")
		if s.builderToAppBlocks[builderName] == nil {
			s.builderToAppBlocks[builderName] = map[string]struct{}{}
		}
		s.builderToAppBlocks[builderName][appBlockName] = struct{}{}
		s.ensureAppBlockBuilderLocked(builderName, now)
		s.ensureAppBlockLocked(appBlockName, now)
		return map[string]any{}
	case "DisassociateAppBlockBuilderAppBlock":
		builderName := workspacesAppStream2LookupString(payload, "AppBlockBuilderName", "Name")
		appBlockName := workspacesAppStream2LookupString(payload, "AppBlockName")
		if s.builderToAppBlocks[builderName] != nil {
			delete(s.builderToAppBlocks[builderName], appBlockName)
		}
		return map[string]any{}
	case "AssociateSoftwareToImageBuilder":
		builderName := workspacesAppStream2LookupString(payload, "ImageBuilderName", "Name")
		software := workspacesAppStream2LookupString(payload, "Platform", "SoftwareName", "SoftwareURL")
		if software == "" {
			software = "default-software"
		}
		if s.imageBuilderToSoftware[builderName] == nil {
			s.imageBuilderToSoftware[builderName] = map[string]struct{}{}
		}
		s.imageBuilderToSoftware[builderName][software] = struct{}{}
		s.ensureImageBuilderLocked(builderName, now)
		return map[string]any{}
	case "DisassociateSoftwareFromImageBuilder":
		builderName := workspacesAppStream2LookupString(payload, "ImageBuilderName", "Name")
		software := workspacesAppStream2LookupString(payload, "Platform", "SoftwareName", "SoftwareURL")
		if s.imageBuilderToSoftware[builderName] != nil {
			delete(s.imageBuilderToSoftware[builderName], software)
		}
		return map[string]any{}

	case "BatchAssociateUserStack":
		assocs := workspacesAppStream2MapSlice(payload["UserStackAssociations"])
		if len(assocs) == 0 {
			assocs = []map[string]any{
				{
					"StackName":          "stackyard-stack",
					"UserName":           "stackyard-user",
					"AuthenticationType": "API",
				},
			}
		}
		for _, assoc := range assocs {
			assocCopy := workspacesAppStream2CloneMap(assoc)
			if assocCopy["SendEmailNotification"] == nil {
				assocCopy["SendEmailNotification"] = false
			}
			s.userStackAssociations = append(s.userStackAssociations, assocCopy)
		}
		return map[string]any{"Errors": []any{}}
	case "BatchDisassociateUserStack":
		s.userStackAssociations = []map[string]any{}
		return map[string]any{"Errors": []any{}}

	case "ListAssociatedFleets":
		stackName := workspacesAppStream2LookupString(payload, "StackName", "Name")
		if stackName == "" {
			stackName = "stackyard-stack"
		}
		names := sortedSetValues(s.stackToFleets[stackName])
		return map[string]any{"Names": names, "NextToken": ""}
	case "ListAssociatedStacks":
		fleetName := workspacesAppStream2LookupString(payload, "FleetName", "Name")
		if fleetName == "" {
			fleetName = "stackyard-fleet"
		}
		names := sortedSetValues(s.fleetToStacks[fleetName])
		return map[string]any{"Names": names, "NextToken": ""}
	case "ListEntitledApplications":
		stackName := workspacesAppStream2LookupString(payload, "StackName", "Name")
		if stackName == "" {
			stackName = "stackyard-stack"
		}
		_ = stackName
		out := []any{}
		for _, app := range s.listResourceMapsLocked(s.applications) {
			appMap, _ := app.(map[string]any)
			out = append(out, map[string]any{
				"Name":        appMap["Name"],
				"AppBlockArn": appMap["Arn"],
			})
		}
		return map[string]any{"EntitledApplications": out, "NextToken": ""}
	case "ListExportImageTasks":
		return map[string]any{"ExportImageTasks": s.listResourceMapsLocked(s.exportImageTasks), "NextToken": ""}
	case "GetExportImageTask":
		taskID := workspacesAppStream2LookupString(payload, "ExportImageTaskId")
		if taskID == "" {
			taskID = "export-task-000001"
		}
		task := s.exportImageTasks[taskID]
		if task == nil {
			task = map[string]any{
				"ExportImageTaskId": taskID,
				"Status":            "COMPLETED",
				"ImageName":         "stackyard-image",
				"CreatedTime":       now,
			}
		}
		return map[string]any{"ExportImageTask": workspacesAppStream2CloneMap(task)}

	case "DescribeApplicationFleetAssociations":
		items := []any{}
		for appName, fleets := range s.applicationToFleets {
			for fleetName := range fleets {
				items = append(items, map[string]any{
					"ApplicationName": appName,
					"FleetName":       fleetName,
				})
			}
		}
		return map[string]any{"ApplicationFleetAssociations": items, "NextToken": ""}
	case "DescribeAppBlockBuilderAppBlockAssociations":
		items := []any{}
		for builderName, appBlocks := range s.builderToAppBlocks {
			for appBlockName := range appBlocks {
				items = append(items, map[string]any{
					"AppBlockBuilderName": builderName,
					"AppBlockName":        appBlockName,
				})
			}
		}
		return map[string]any{"AppBlockBuilderAppBlockAssociations": items, "NextToken": ""}
	case "DescribeUserStackAssociations":
		out := make([]any, 0, len(s.userStackAssociations))
		for _, assoc := range s.userStackAssociations {
			out = append(out, workspacesAppStream2CloneMap(assoc))
		}
		return map[string]any{"UserStackAssociations": out, "NextToken": ""}
	case "DescribeAppLicenseUsage":
		return map[string]any{"AppLicenseUsages": []any{map[string]any{"UsageDate": now, "LicenseStatus": "IN_USE"}}}
	case "DescribeImagePermissions":
		return map[string]any{
			"Name":                       "stackyard-image",
			"SharedImagePermissionsList": []any{map[string]any{"sharedAccountId": workspacesAppStream2AccountID}},
		}
	case "DescribeSoftwareAssociations":
		associations := []any{}
		for builderName, softwareSet := range s.imageBuilderToSoftware {
			for software := range softwareSet {
				associations = append(associations, map[string]any{
					"ImageBuilderName": builderName,
					"Platform":         software,
				})
			}
		}
		return map[string]any{"SoftwareAssociations": associations}

	case "CreateStreamingURL":
		return map[string]any{
			"StreamingURL": fmt.Sprintf("https://appstream.%s.amazonaws.com/stream/%s", workspacesAppStream2Region, s.nextTokenLocked("stream")),
			"Expires":      now,
		}
	case "CreateImageBuilderStreamingURL":
		return map[string]any{
			"StreamingURL": fmt.Sprintf("https://appstream.%s.amazonaws.com/imagebuilder/%s", workspacesAppStream2Region, s.nextTokenLocked("ib-stream")),
			"Expires":      now,
		}
	case "CreateAppBlockBuilderStreamingURL":
		return map[string]any{
			"StreamingURL": fmt.Sprintf("https://appstream.%s.amazonaws.com/appblockbuilder/%s", workspacesAppStream2Region, s.nextTokenLocked("abb-stream")),
			"Expires":      now,
		}

	case "StartFleet", "StopFleet":
		name := workspacesAppStream2LookupString(payload, "Name", "FleetName")
		f := s.ensureFleetLocked(name, now)
		if strings.HasPrefix(action, "Start") {
			f["State"] = "RUNNING"
		} else {
			f["State"] = "STOPPED"
		}
		return map[string]any{}
	case "StartImageBuilder", "StopImageBuilder":
		name := workspacesAppStream2LookupString(payload, "Name", "ImageBuilderName")
		builder := s.ensureImageBuilderLocked(name, now)
		if strings.HasPrefix(action, "Start") {
			builder["State"] = "RUNNING"
		} else {
			builder["State"] = "STOPPED"
		}
		return map[string]any{}
	case "StartAppBlockBuilder", "StopAppBlockBuilder":
		name := workspacesAppStream2LookupString(payload, "Name", "AppBlockBuilderName")
		builder := s.ensureAppBlockBuilderLocked(name, now)
		if strings.HasPrefix(action, "Start") {
			builder["State"] = "RUNNING"
		} else {
			builder["State"] = "STOPPED"
		}
		return map[string]any{}
	case "StartSoftwareDeploymentToImageBuilder":
		return map[string]any{}
	case "EnableUser", "DisableUser":
		userName := workspacesAppStream2LookupString(payload, "UserName", "Name")
		authType := workspacesAppStream2LookupString(payload, "AuthenticationType")
		u := s.ensureUserLocked(userName, authType, now)
		if strings.HasPrefix(action, "Enable") {
			u["Enabled"] = true
		} else {
			u["Enabled"] = false
		}
		return map[string]any{}
	case "ExpireSession":
		sessionID := workspacesAppStream2LookupString(payload, "SessionId")
		if sessionID == "" {
			sessionID = "session-000001"
		}
		sess := s.ensureSessionLocked(sessionID, now)
		sess["State"] = "EXPIRED"
		return map[string]any{}
	case "CreateThemeForStack":
		return map[string]any{}
	case "TagResource":
		resourceArn := workspacesAppStream2LookupString(payload, "ResourceArn")
		if resourceArn == "" {
			resourceArn = "arn:aws:appstream:us-east-1:123456789012:fleet/stackyard-fleet"
		}
		tags := s.ensureTagsLocked(resourceArn)
		for k, v := range workspacesAppStream2ExtractTags(payload) {
			tags[k] = v
		}
		return map[string]any{}
	case "UntagResource":
		resourceArn := workspacesAppStream2LookupString(payload, "ResourceArn")
		if resourceArn == "" {
			resourceArn = "arn:aws:appstream:us-east-1:123456789012:fleet/stackyard-fleet"
		}
		tags := s.ensureTagsLocked(resourceArn)
		for _, key := range workspacesAppStream2ExtractTagKeys(payload) {
			delete(tags, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceArn := workspacesAppStream2LookupString(payload, "ResourceArn")
		if resourceArn == "" {
			resourceArn = "arn:aws:appstream:us-east-1:123456789012:fleet/stackyard-fleet"
		}
		return map[string]any{"Tags": workspacesAppStream2CloneStringMap(s.ensureTagsLocked(resourceArn))}
	}

	return map[string]any{}
}

func (s *workspacesAppStream2Store) seedLocked() {
	now := time.Now().UTC().Format(time.RFC3339)

	s.ensureFleetLocked("stackyard-fleet", now)
	s.ensureStackLocked("stackyard-stack", now)
	s.ensureImageLocked("stackyard-image", now)
	s.ensureImageBuilderLocked("stackyard-image-builder", now)
	s.ensureApplicationLocked("stackyard-application", now)
	s.ensureAppBlockLocked("stackyard-appblock", now)
	s.ensureAppBlockBuilderLocked("stackyard-appblockbuilder", now)
	s.ensureDirectoryConfigLocked("stackyard.example.com", now)
	s.ensureEntitlementLocked("stackyard-entitlement", now)
	s.ensureUserLocked("stackyard-user", "API", now)
	s.ensureUsageReportLocked("stackyard-reports", now)
	s.ensureSessionLocked("session-000001", now)

	if s.exportImageTasks["export-task-000001"] == nil {
		s.exportImageTasks["export-task-000001"] = map[string]any{
			"ExportImageTaskId": "export-task-000001",
			"Status":            "COMPLETED",
			"ImageName":         "stackyard-image",
			"CreatedTime":       now,
		}
	}

	s.associateStackFleetLocked("stackyard-stack", "stackyard-fleet", now)
	if s.applicationToFleets["stackyard-application"] == nil {
		s.applicationToFleets["stackyard-application"] = map[string]struct{}{}
	}
	s.applicationToFleets["stackyard-application"]["stackyard-fleet"] = struct{}{}
	if s.entitlementToApps["stackyard-entitlement"] == nil {
		s.entitlementToApps["stackyard-entitlement"] = map[string]struct{}{}
	}
	s.entitlementToApps["stackyard-entitlement"]["stackyard-application"] = struct{}{}
	if s.builderToAppBlocks["stackyard-appblockbuilder"] == nil {
		s.builderToAppBlocks["stackyard-appblockbuilder"] = map[string]struct{}{}
	}
	s.builderToAppBlocks["stackyard-appblockbuilder"]["stackyard-appblock"] = struct{}{}
	if s.imageBuilderToSoftware["stackyard-image-builder"] == nil {
		s.imageBuilderToSoftware["stackyard-image-builder"] = map[string]struct{}{}
	}
	s.imageBuilderToSoftware["stackyard-image-builder"]["AMAZON_LINUX_2"] = struct{}{}
	s.ensureTagsLocked(workspacesAppStream2FleetARN("stackyard-fleet"))["stackyard"] = "true"
}

func (s *workspacesAppStream2Store) associateStackFleetLocked(stackName, fleetName, now string) {
	if stackName == "" {
		stackName = "stackyard-stack"
	}
	if fleetName == "" {
		fleetName = "stackyard-fleet"
	}
	s.ensureStackLocked(stackName, now)
	s.ensureFleetLocked(fleetName, now)
	if s.stackToFleets[stackName] == nil {
		s.stackToFleets[stackName] = map[string]struct{}{}
	}
	if s.fleetToStacks[fleetName] == nil {
		s.fleetToStacks[fleetName] = map[string]struct{}{}
	}
	s.stackToFleets[stackName][fleetName] = struct{}{}
	s.fleetToStacks[fleetName][stackName] = struct{}{}
}

func (s *workspacesAppStream2Store) disassociateStackFleetLocked(stackName, fleetName string) {
	if set := s.stackToFleets[stackName]; set != nil {
		delete(set, fleetName)
	}
	if set := s.fleetToStacks[fleetName]; set != nil {
		delete(set, stackName)
	}
}

func (s *workspacesAppStream2Store) ensureFleetLocked(name, now string) map[string]any {
	if name == "" {
		name = "stackyard-fleet"
	}
	if existing := s.fleets[name]; existing != nil {
		return existing
	}
	fleet := map[string]any{
		"Name":            name,
		"Arn":             workspacesAppStream2FleetARN(name),
		"State":           "RUNNING",
		"CreatedTime":     now,
		"InstanceType":    "stream.standard.small",
		"DisplayName":     name,
		"LastUpdatedTime": now,
	}
	s.fleets[name] = fleet
	return fleet
}

func (s *workspacesAppStream2Store) ensureStackLocked(name, now string) map[string]any {
	if name == "" {
		name = "stackyard-stack"
	}
	if existing := s.stacks[name]; existing != nil {
		return existing
	}
	stack := map[string]any{
		"Name":            name,
		"Arn":             workspacesAppStream2StackARN(name),
		"DisplayName":     name,
		"CreatedTime":     now,
		"LastUpdatedTime": now,
	}
	s.stacks[name] = stack
	return stack
}

func (s *workspacesAppStream2Store) ensureImageLocked(name, now string) map[string]any {
	if name == "" {
		name = "stackyard-image"
	}
	if existing := s.images[name]; existing != nil {
		return existing
	}
	image := map[string]any{
		"Name":            name,
		"Arn":             workspacesAppStream2ImageARN(name),
		"State":           "AVAILABLE",
		"CreatedTime":     now,
		"Visibility":      "PRIVATE",
		"LastUpdatedTime": now,
	}
	s.images[name] = image
	return image
}

func (s *workspacesAppStream2Store) ensureImageBuilderLocked(name, now string) map[string]any {
	if name == "" {
		name = "stackyard-image-builder"
	}
	if existing := s.builders[name]; existing != nil {
		return existing
	}
	builder := map[string]any{
		"Name":            name,
		"Arn":             workspacesAppStream2ImageBuilderARN(name),
		"State":           "RUNNING",
		"CreatedTime":     now,
		"LastUpdatedTime": now,
	}
	s.builders[name] = builder
	return builder
}

func (s *workspacesAppStream2Store) ensureApplicationLocked(name, now string) map[string]any {
	if name == "" {
		name = "stackyard-application"
	}
	if existing := s.applications[name]; existing != nil {
		return existing
	}
	app := map[string]any{
		"Name":            name,
		"Arn":             workspacesAppStream2ApplicationARN(name),
		"DisplayName":     name,
		"CreatedTime":     now,
		"LastUpdatedTime": now,
	}
	s.applications[name] = app
	return app
}

func (s *workspacesAppStream2Store) ensureAppBlockLocked(name, now string) map[string]any {
	if name == "" {
		name = "stackyard-appblock"
	}
	if existing := s.appBlocks[name]; existing != nil {
		return existing
	}
	appBlock := map[string]any{
		"Name":            name,
		"Arn":             workspacesAppStream2AppBlockARN(name),
		"CreatedTime":     now,
		"LastUpdatedTime": now,
	}
	s.appBlocks[name] = appBlock
	return appBlock
}

func (s *workspacesAppStream2Store) ensureAppBlockBuilderLocked(name, now string) map[string]any {
	if name == "" {
		name = "stackyard-appblockbuilder"
	}
	if existing := s.appBlockBuilders[name]; existing != nil {
		return existing
	}
	builder := map[string]any{
		"Name":            name,
		"Arn":             workspacesAppStream2AppBlockBuilderARN(name),
		"State":           "RUNNING",
		"CreatedTime":     now,
		"LastUpdatedTime": now,
	}
	s.appBlockBuilders[name] = builder
	return builder
}

func (s *workspacesAppStream2Store) ensureDirectoryConfigLocked(name, now string) map[string]any {
	if name == "" {
		name = "stackyard.example.com"
	}
	if existing := s.directoryConfigs[name]; existing != nil {
		return existing
	}
	cfg := map[string]any{
		"DirectoryName": name,
		"CreatedTime":   now,
		"ServiceAccountCredentials": map[string]any{
			"AccountName": "stackyard-svc",
		},
	}
	s.directoryConfigs[name] = cfg
	return cfg
}

func (s *workspacesAppStream2Store) ensureEntitlementLocked(name, now string) map[string]any {
	if name == "" {
		name = "stackyard-entitlement"
	}
	if existing := s.entitlements[name]; existing != nil {
		return existing
	}
	ent := map[string]any{
		"Name":            name,
		"CreatedTime":     now,
		"Description":     "stackyard entitlement",
		"LastUpdatedTime": now,
	}
	s.entitlements[name] = ent
	return ent
}

func (s *workspacesAppStream2Store) ensureUserLocked(userName, authType, now string) map[string]any {
	if userName == "" {
		userName = "stackyard-user"
	}
	if authType == "" {
		authType = "API"
	}
	key := workspacesAppStream2UserKey(userName, authType)
	if existing := s.users[key]; existing != nil {
		return existing
	}
	user := map[string]any{
		"UserName":           userName,
		"AuthenticationType": authType,
		"Enabled":            true,
		"FirstName":          "Stackyard",
		"LastName":           "User",
		"CreatedTime":        now,
		"LastUpdatedTime":    now,
	}
	s.users[key] = user
	return user
}

func (s *workspacesAppStream2Store) ensureUsageReportLocked(bucketName, now string) map[string]any {
	if bucketName == "" {
		bucketName = "stackyard-reports"
	}
	if existing := s.usageReports[bucketName]; existing != nil {
		return existing
	}
	report := map[string]any{
		"S3BucketName":            bucketName,
		"Schedule":                "DAILY",
		"LastGeneratedReportDate": now,
	}
	s.usageReports[bucketName] = report
	return report
}

func (s *workspacesAppStream2Store) ensureSessionLocked(sessionID, now string) map[string]any {
	if sessionID == "" {
		sessionID = "session-000001"
	}
	if existing := s.sessions[sessionID]; existing != nil {
		return existing
	}
	session := map[string]any{
		"Id":                 sessionID,
		"UserId":             "stackyard-user",
		"State":              "ACTIVE",
		"AuthenticationType": "API",
		"CreatedTime":        now,
	}
	s.sessions[sessionID] = session
	return session
}

func (s *workspacesAppStream2Store) ensureTagsLocked(resourceARN string) map[string]string {
	if resourceARN == "" {
		resourceARN = workspacesAppStream2FleetARN("stackyard-fleet")
	}
	if existing := s.tags[resourceARN]; existing != nil {
		return existing
	}
	s.tags[resourceARN] = map[string]string{}
	return s.tags[resourceARN]
}

func (s *workspacesAppStream2Store) listResourceMapsLocked(src map[string]map[string]any) []any {
	keys := make([]string, 0, len(src))
	for key := range src {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, workspacesAppStream2CloneMap(src[key]))
	}
	return out
}

func (s *workspacesAppStream2Store) nextTokenLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%06d", prefix, id)
}

func workspacesAppStream2FleetARN(name string) string {
	return fmt.Sprintf("arn:aws:appstream:%s:%s:fleet/%s", workspacesAppStream2Region, workspacesAppStream2AccountID, name)
}

func workspacesAppStream2StackARN(name string) string {
	return fmt.Sprintf("arn:aws:appstream:%s:%s:stack/%s", workspacesAppStream2Region, workspacesAppStream2AccountID, name)
}

func workspacesAppStream2ImageARN(name string) string {
	return fmt.Sprintf("arn:aws:appstream:%s:%s:image/%s", workspacesAppStream2Region, workspacesAppStream2AccountID, name)
}

func workspacesAppStream2ImageBuilderARN(name string) string {
	return fmt.Sprintf("arn:aws:appstream:%s:%s:image-builder/%s", workspacesAppStream2Region, workspacesAppStream2AccountID, name)
}

func workspacesAppStream2ApplicationARN(name string) string {
	return fmt.Sprintf("arn:aws:appstream:%s:%s:application/%s", workspacesAppStream2Region, workspacesAppStream2AccountID, name)
}

func workspacesAppStream2AppBlockARN(name string) string {
	return fmt.Sprintf("arn:aws:appstream:%s:%s:app-block/%s", workspacesAppStream2Region, workspacesAppStream2AccountID, name)
}

func workspacesAppStream2AppBlockBuilderARN(name string) string {
	return fmt.Sprintf("arn:aws:appstream:%s:%s:app-block-builder/%s", workspacesAppStream2Region, workspacesAppStream2AccountID, name)
}

func workspacesAppStream2UserKey(userName, authType string) string {
	if userName == "" {
		userName = "stackyard-user"
	}
	if authType == "" {
		authType = "API"
	}
	return strings.ToUpper(authType) + "|" + userName
}

func workspacesAppStream2LookupString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		if str, ok := raw.(string); ok {
			str = strings.TrimSpace(str)
			if str != "" {
				return str
			}
		}
	}
	return ""
}

func workspacesAppStream2MapSlice(raw any) []map[string]any {
	switch v := raw.(type) {
	case []map[string]any:
		return v
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, entry)
		}
		return out
	default:
		return nil
	}
}

func workspacesAppStream2ExtractTags(payload map[string]any) map[string]string {
	raw := payload["Tags"]
	if raw == nil {
		raw = payload["tags"]
	}
	out := map[string]string{}
	switch tags := raw.(type) {
	case map[string]any:
		for k, v := range tags {
			if s, ok := v.(string); ok && strings.TrimSpace(k) != "" {
				out[k] = s
			}
		}
	case map[string]string:
		for k, v := range tags {
			if strings.TrimSpace(k) != "" {
				out[k] = v
			}
		}
	}
	return out
}

func workspacesAppStream2ExtractTagKeys(payload map[string]any) []string {
	raw := payload["TagKeys"]
	if raw == nil {
		raw = payload["tagKeys"]
	}
	switch values := raw.(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			s, ok := value.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		values = strings.TrimSpace(values)
		if values == "" {
			return nil
		}
		parts := strings.Split(values, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			key := strings.TrimSpace(part)
			if key != "" {
				out = append(out, key)
			}
		}
		return out
	default:
		return nil
	}
}

func sortedSetValues(set map[string]struct{}) []string {
	if len(set) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func workspacesAppStream2CloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func workspacesAppStream2CloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
