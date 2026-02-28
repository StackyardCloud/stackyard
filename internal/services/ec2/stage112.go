package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type DeleteLaunchTemplateVersionsResponseSuccessItem struct {
	LaunchTemplateID   string
	LaunchTemplateName string
	VersionNumber      int64
}

type DeleteLaunchTemplateVersionsResponseError struct {
	Code    string
	Message string
}

type DeleteLaunchTemplateVersionsResponseErrorItem struct {
	LaunchTemplateID   string
	LaunchTemplateName string
	ResponseError      DeleteLaunchTemplateVersionsResponseError
	VersionNumber      *int64
}

func (s *Service) DeleteLaunchTemplateVersions(
	launchTemplateID string,
	launchTemplateName string,
	versions []string,
) ([]DeleteLaunchTemplateVersionsResponseSuccessItem, []DeleteLaunchTemplateVersionsResponseErrorItem, error) {
	launchTemplateID = strings.TrimSpace(launchTemplateID)
	launchTemplateName = strings.TrimSpace(launchTemplateName)
	versions = dedupeTrimmedStrings(versions)
	if launchTemplateID == "" && launchTemplateName == "" {
		return nil, nil, ErrInvalidParameter
	}
	if launchTemplateID != "" && launchTemplateName != "" {
		return nil, nil, ErrInvalidParameter
	}
	if len(versions) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if launchTemplateID == "" {
		launchTemplateID = s.launchTemplateNameIndex[launchTemplateName]
	}
	launchTemplate := s.launchTemplates[launchTemplateID]
	if launchTemplate == nil {
		return nil, nil, ErrNotFound
	}
	if launchTemplateName != "" && launchTemplate.LaunchTemplateName != launchTemplateName {
		return nil, nil, ErrInvalidParameter
	}

	versionSet := s.launchTemplateVersions[launchTemplateID]
	if versionSet == nil {
		versionSet = map[int64]*LaunchTemplateVersion{}
		s.launchTemplateVersions[launchTemplateID] = versionSet
	}

	successes := make([]DeleteLaunchTemplateVersionsResponseSuccessItem, 0, len(versions))
	errors := make([]DeleteLaunchTemplateVersionsResponseErrorItem, 0)
	seenVersionNumbers := make(map[int64]struct{}, len(versions))
	for _, rawVersion := range versions {
		versionNumber, ok := resolveDeleteLaunchTemplateVersion(rawVersion, launchTemplate)
		if !ok {
			errors = append(errors, DeleteLaunchTemplateVersionsResponseErrorItem{
				LaunchTemplateID:   launchTemplate.LaunchTemplateID,
				LaunchTemplateName: launchTemplate.LaunchTemplateName,
				ResponseError: DeleteLaunchTemplateVersionsResponseError{
					Code:    "InvalidLaunchTemplateVersion.Malformed",
					Message: "launch template version is malformed",
				},
			})
			continue
		}
		if _, seen := seenVersionNumbers[versionNumber]; seen {
			continue
		}
		seenVersionNumbers[versionNumber] = struct{}{}

		if versionNumber == launchTemplate.DefaultVersionNumber {
			versionNumberCopy := versionNumber
			errors = append(errors, DeleteLaunchTemplateVersionsResponseErrorItem{
				LaunchTemplateID:   launchTemplate.LaunchTemplateID,
				LaunchTemplateName: launchTemplate.LaunchTemplateName,
				ResponseError: DeleteLaunchTemplateVersionsResponseError{
					Code:    "OperationNotPermitted",
					Message: "cannot delete default launch template version",
				},
				VersionNumber: &versionNumberCopy,
			})
			continue
		}

		version := versionSet[versionNumber]
		if version == nil {
			versionNumberCopy := versionNumber
			errors = append(errors, DeleteLaunchTemplateVersionsResponseErrorItem{
				LaunchTemplateID:   launchTemplate.LaunchTemplateID,
				LaunchTemplateName: launchTemplate.LaunchTemplateName,
				ResponseError: DeleteLaunchTemplateVersionsResponseError{
					Code:    "InvalidLaunchTemplateVersion.NotFound",
					Message: "launch template version not found",
				},
				VersionNumber: &versionNumberCopy,
			})
			continue
		}

		successes = append(successes, DeleteLaunchTemplateVersionsResponseSuccessItem{
			LaunchTemplateID:   launchTemplate.LaunchTemplateID,
			LaunchTemplateName: launchTemplate.LaunchTemplateName,
			VersionNumber:      version.VersionNumber,
		})
		delete(versionSet, versionNumber)
	}

	if len(versionSet) > 0 {
		latestVersionNumber := int64(0)
		for versionNumber := range versionSet {
			if versionNumber > latestVersionNumber {
				latestVersionNumber = versionNumber
			}
		}
		launchTemplate.LatestVersionNumber = latestVersionNumber
	}

	return successes, errors, nil
}

func (s *Service) DeleteLocalGatewayRoute(
	localGatewayRouteTableID string,
	destinationCidrBlock *string,
	destinationPrefixListID *string,
) (LocalGatewayRoute, error) {
	localGatewayRouteTableID = strings.TrimSpace(localGatewayRouteTableID)
	destinationCIDR := strings.TrimSpace(derefString(destinationCidrBlock))
	destinationPrefixListIDValue := strings.TrimSpace(derefString(destinationPrefixListID))
	if localGatewayRouteTableID == "" {
		return LocalGatewayRoute{}, ErrInvalidParameter
	}
	if destinationCIDR == "" && destinationPrefixListIDValue == "" {
		return LocalGatewayRoute{}, ErrInvalidParameter
	}
	if destinationCIDR != "" && destinationPrefixListIDValue != "" {
		return LocalGatewayRoute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.localGatewayRouteTables[localGatewayRouteTableID] == nil {
		return LocalGatewayRoute{}, ErrNotFound
	}

	candidates := make([]string, 0)
	for key, route := range s.localGatewayRoutes {
		if route == nil {
			continue
		}
		if route.LocalGatewayRouteTableID != localGatewayRouteTableID {
			continue
		}
		if route.DestinationCidrBlock != destinationCIDR {
			continue
		}
		if route.DestinationPrefixListID != destinationPrefixListIDValue {
			continue
		}
		candidates = append(candidates, key)
	}
	if len(candidates) == 0 {
		return LocalGatewayRoute{}, ErrNotFound
	}
	sort.Strings(candidates)
	out := cloneStage108LocalGatewayRoute(s.localGatewayRoutes[candidates[0]])
	delete(s.localGatewayRoutes, candidates[0])
	return out, nil
}

func (s *Service) DeleteLocalGatewayRouteTable(localGatewayRouteTableID string) (LocalGatewayRouteTable, error) {
	localGatewayRouteTableID = strings.TrimSpace(localGatewayRouteTableID)
	if localGatewayRouteTableID == "" {
		return LocalGatewayRouteTable{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	routeTable := s.localGatewayRouteTables[localGatewayRouteTableID]
	if routeTable == nil {
		return LocalGatewayRouteTable{}, ErrNotFound
	}

	out := cloneStage108LocalGatewayRouteTable(routeTable)
	delete(s.localGatewayRouteTables, localGatewayRouteTableID)

	for key, route := range s.localGatewayRoutes {
		if route == nil || route.LocalGatewayRouteTableID != localGatewayRouteTableID {
			continue
		}
		delete(s.localGatewayRoutes, key)
	}
	for key, association := range s.localGatewayRouteTableVifAssociations {
		if association == nil || association.LocalGatewayRouteTableID != localGatewayRouteTableID {
			continue
		}
		delete(s.localGatewayRouteTableVifAssociations, key)
	}
	for key, association := range s.localGatewayRouteTableVpcAssociations {
		if association == nil || association.LocalGatewayRouteTableID != localGatewayRouteTableID {
			continue
		}
		delete(s.localGatewayRouteTableVpcAssociations, key)
	}

	return out, nil
}

func (s *Service) DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociation(localGatewayRouteTableVirtualInterfaceGroupAssociationID string) (LocalGatewayRouteTableVirtualInterfaceGroupAssociation, error) {
	localGatewayRouteTableVirtualInterfaceGroupAssociationID = strings.TrimSpace(localGatewayRouteTableVirtualInterfaceGroupAssociationID)
	if localGatewayRouteTableVirtualInterfaceGroupAssociationID == "" {
		return LocalGatewayRouteTableVirtualInterfaceGroupAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	foundKey := ""
	var association *LocalGatewayRouteTableVirtualInterfaceGroupAssociation
	for key, item := range s.localGatewayRouteTableVifAssociations {
		if item == nil || item.LocalGatewayRouteTableVirtualInterfaceGroupAssociationID != localGatewayRouteTableVirtualInterfaceGroupAssociationID {
			continue
		}
		foundKey = key
		association = item
		break
	}
	if foundKey == "" || association == nil {
		return LocalGatewayRouteTableVirtualInterfaceGroupAssociation{}, ErrNotFound
	}

	out := cloneStage108LocalGatewayRouteTableVirtualInterfaceGroupAssociation(association)
	delete(s.localGatewayRouteTableVifAssociations, foundKey)
	return out, nil
}

func (s *Service) DeleteLocalGatewayRouteTableVpcAssociation(localGatewayRouteTableVpcAssociationID string) (LocalGatewayRouteTableVpcAssociation, error) {
	localGatewayRouteTableVpcAssociationID = strings.TrimSpace(localGatewayRouteTableVpcAssociationID)
	if localGatewayRouteTableVpcAssociationID == "" {
		return LocalGatewayRouteTableVpcAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	foundKey := ""
	var association *LocalGatewayRouteTableVpcAssociation
	for key, item := range s.localGatewayRouteTableVpcAssociations {
		if item == nil || item.LocalGatewayRouteTableVpcAssociationID != localGatewayRouteTableVpcAssociationID {
			continue
		}
		foundKey = key
		association = item
		break
	}
	if foundKey == "" || association == nil {
		return LocalGatewayRouteTableVpcAssociation{}, ErrNotFound
	}

	out := cloneStage108LocalGatewayRouteTableVpcAssociation(association)
	delete(s.localGatewayRouteTableVpcAssociations, foundKey)
	return out, nil
}

func (s *Service) DeleteLocalGatewayVirtualInterface(localGatewayVirtualInterfaceID string) (LocalGatewayVirtualInterface, error) {
	localGatewayVirtualInterfaceID = strings.TrimSpace(localGatewayVirtualInterfaceID)
	if localGatewayVirtualInterfaceID == "" {
		return LocalGatewayVirtualInterface{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	virtualInterface := s.localGatewayVirtualInterfaces[localGatewayVirtualInterfaceID]
	if virtualInterface == nil {
		return LocalGatewayVirtualInterface{}, ErrNotFound
	}
	out := cloneStage108LocalGatewayVirtualInterface(virtualInterface)
	delete(s.localGatewayVirtualInterfaces, localGatewayVirtualInterfaceID)

	group := s.localGatewayVirtualInterfaceGroups[virtualInterface.LocalGatewayVirtualInterfaceGroupID]
	if group != nil && len(group.LocalGatewayVirtualInterfaceIDs) > 0 {
		filtered := make([]string, 0, len(group.LocalGatewayVirtualInterfaceIDs))
		for _, id := range group.LocalGatewayVirtualInterfaceIDs {
			if id == localGatewayVirtualInterfaceID {
				continue
			}
			filtered = append(filtered, id)
		}
		group.LocalGatewayVirtualInterfaceIDs = filtered
	}

	return out, nil
}

func (s *Service) DeleteLocalGatewayVirtualInterfaceGroup(localGatewayVirtualInterfaceGroupID string) (LocalGatewayVirtualInterfaceGroup, error) {
	localGatewayVirtualInterfaceGroupID = strings.TrimSpace(localGatewayVirtualInterfaceGroupID)
	if localGatewayVirtualInterfaceGroupID == "" {
		return LocalGatewayVirtualInterfaceGroup{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	group := s.localGatewayVirtualInterfaceGroups[localGatewayVirtualInterfaceGroupID]
	if group == nil {
		return LocalGatewayVirtualInterfaceGroup{}, ErrNotFound
	}
	out := cloneStage109LocalGatewayVirtualInterfaceGroup(group)
	delete(s.localGatewayVirtualInterfaceGroups, localGatewayVirtualInterfaceGroupID)

	for key, association := range s.localGatewayRouteTableVifAssociations {
		if association == nil || association.LocalGatewayVirtualInterfaceGroupID != localGatewayVirtualInterfaceGroupID {
			continue
		}
		delete(s.localGatewayRouteTableVifAssociations, key)
	}
	for key, virtualInterface := range s.localGatewayVirtualInterfaces {
		if virtualInterface == nil || virtualInterface.LocalGatewayVirtualInterfaceGroupID != localGatewayVirtualInterfaceGroupID {
			continue
		}
		delete(s.localGatewayVirtualInterfaces, key)
	}

	return out, nil
}

func (s *Service) DeleteManagedPrefixList(prefixListID string) (ManagedPrefixList, error) {
	prefixListID = strings.TrimSpace(prefixListID)
	if prefixListID == "" {
		return ManagedPrefixList{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	prefixList := s.managedPrefixLists[prefixListID]
	if prefixList == nil {
		return ManagedPrefixList{}, ErrNotFound
	}
	out := cloneStage109ManagedPrefixList(prefixList)
	delete(s.managedPrefixLists, prefixListID)
	return out, nil
}

func (s *Service) DeleteNetworkInsightsAccessScope(networkInsightsAccessScopeID string) (string, error) {
	networkInsightsAccessScopeID = strings.TrimSpace(networkInsightsAccessScopeID)
	if networkInsightsAccessScopeID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.networkInsightsAccessScopes[networkInsightsAccessScopeID] == nil {
		return "", ErrNotFound
	}
	delete(s.networkInsightsAccessScopes, networkInsightsAccessScopeID)
	return networkInsightsAccessScopeID, nil
}

func (s *Service) DeleteNetworkInsightsAccessScopeAnalysis(networkInsightsAccessScopeAnalysisID string) (string, error) {
	networkInsightsAccessScopeAnalysisID = strings.TrimSpace(networkInsightsAccessScopeAnalysisID)
	if networkInsightsAccessScopeAnalysisID == "" {
		return "", ErrInvalidParameter
	}
	return networkInsightsAccessScopeAnalysisID, nil
}

func resolveDeleteLaunchTemplateVersion(raw string, launchTemplate *LaunchTemplate) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || launchTemplate == nil {
		return 0, false
	}
	switch raw {
	case "$Default":
		return launchTemplate.DefaultVersionNumber, true
	case "$Latest":
		return launchTemplate.LatestVersionNumber, true
	default:
		versionNumber, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || versionNumber <= 0 {
			return 0, false
		}
		return versionNumber, true
	}
}
