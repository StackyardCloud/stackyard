package ec2

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type LocalGateway struct {
	LocalGatewayID string
	OutpostARN     string
	OwnerID        string
	State          string
	Tags           map[string]string
}

type LockedSnapshotInfo struct {
	CoolOffPeriod          *int32
	CoolOffPeriodExpiresOn *time.Time
	LockCreatedOn          *time.Time
	LockDuration           *int32
	LockDurationStartTime  *time.Time
	LockExpiresOn          *time.Time
	LockState              string
	OwnerID                string
	SnapshotID             string
}

type MacHost struct {
	HostID                      string
	MacOSLatestSupportedVersion []string
}

func (s *Service) DescribeLaunchTemplates(launchTemplateIDs, launchTemplateNames []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]LaunchTemplate, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(launchTemplateIDs))
	requestedNameSet := toStringSet(dedupeTrimmedStrings(launchTemplateNames))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	launchTemplateIDFilterSet := toStringSet(standardFilters["launch-template-id"])
	launchTemplateNameFilterSet := toStringSet(standardFilters["launch-template-name"])
	createdByFilterSet := toStringSet(standardFilters["created-by"])
	createTimeFilterSet := toStringSet(standardFilters["create-time"])
	defaultVersionFilterSet := toStringSet(standardFilters["default-version-number"])
	latestVersionFilterSet := toStringSet(standardFilters["latest-version-number"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.launchTemplates))
	for launchTemplateID := range s.launchTemplates {
		ids = append(ids, launchTemplateID)
	}
	sort.Strings(ids)

	items := make([]LaunchTemplate, 0, len(ids))
	for _, launchTemplateID := range ids {
		launchTemplate := s.launchTemplates[launchTemplateID]
		if launchTemplate == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[launchTemplateID]; !ok {
				continue
			}
		}
		if len(requestedNameSet) > 0 {
			if _, ok := requestedNameSet[launchTemplate.LaunchTemplateName]; !ok {
				continue
			}
		}
		if len(launchTemplateIDFilterSet) > 0 {
			if _, ok := launchTemplateIDFilterSet[launchTemplateID]; !ok {
				continue
			}
		}
		if len(launchTemplateNameFilterSet) > 0 {
			if _, ok := launchTemplateNameFilterSet[launchTemplate.LaunchTemplateName]; !ok {
				continue
			}
		}
		if len(createdByFilterSet) > 0 {
			if _, ok := createdByFilterSet[launchTemplate.CreatedBy]; !ok {
				continue
			}
		}
		if len(createTimeFilterSet) > 0 {
			createTime := launchTemplate.CreateTime.UTC().Format(time.RFC3339)
			if _, ok := createTimeFilterSet[createTime]; !ok {
				continue
			}
		}
		if len(defaultVersionFilterSet) > 0 {
			if _, ok := defaultVersionFilterSet[strconv.FormatInt(launchTemplate.DefaultVersionNumber, 10)]; !ok {
				continue
			}
		}
		if len(latestVersionFilterSet) > 0 {
			if _, ok := latestVersionFilterSet[strconv.FormatInt(launchTemplate.LatestVersionNumber, 10)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(launchTemplate.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage108LaunchTemplate(launchTemplate))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LaunchTemplate(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociations(associationIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]LocalGatewayRouteTableVirtualInterfaceGroupAssociation, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(associationIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	associationIDFilterSet := toStringSet(standardFilters["local-gateway-route-table-virtual-interface-group-association-id"])
	localGatewayIDFilterSet := toStringSet(standardFilters["local-gateway-id"])
	localGatewayRouteTableIDFilterSet := toStringSet(standardFilters["local-gateway-route-table-id"])
	localGatewayVirtualInterfaceGroupIDFilterSet := toStringSet(standardFilters["local-gateway-virtual-interface-group-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])

	s.mu.Lock()
	items := make([]LocalGatewayRouteTableVirtualInterfaceGroupAssociation, 0, len(s.localGatewayRouteTableVifAssociations))
	for _, association := range s.localGatewayRouteTableVifAssociations {
		if association == nil {
			continue
		}
		associationID := association.LocalGatewayRouteTableVirtualInterfaceGroupAssociationID
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[associationID]; !ok {
				continue
			}
		}
		if len(associationIDFilterSet) > 0 {
			if _, ok := associationIDFilterSet[associationID]; !ok {
				continue
			}
		}
		if len(localGatewayIDFilterSet) > 0 {
			if _, ok := localGatewayIDFilterSet[association.LocalGatewayID]; !ok {
				continue
			}
		}
		if len(localGatewayRouteTableIDFilterSet) > 0 {
			if _, ok := localGatewayRouteTableIDFilterSet[association.LocalGatewayRouteTableID]; !ok {
				continue
			}
		}
		if len(localGatewayVirtualInterfaceGroupIDFilterSet) > 0 {
			if _, ok := localGatewayVirtualInterfaceGroupIDFilterSet[association.LocalGatewayVirtualInterfaceGroupID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[association.OwnerID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(association.State)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(association.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage108LocalGatewayRouteTableVirtualInterfaceGroupAssociation(association))
	}
	s.mu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].LocalGatewayRouteTableVirtualInterfaceGroupAssociationID < items[j].LocalGatewayRouteTableVirtualInterfaceGroupAssociationID
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LocalGatewayRouteTableVirtualInterfaceGroupAssociation(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeLocalGatewayRouteTableVpcAssociations(associationIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]LocalGatewayRouteTableVpcAssociation, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(associationIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	associationIDFilterSet := toStringSet(standardFilters["local-gateway-route-table-vpc-association-id"])
	localGatewayIDFilterSet := toStringSet(standardFilters["local-gateway-id"])
	localGatewayRouteTableIDFilterSet := toStringSet(standardFilters["local-gateway-route-table-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	vpcIDFilterSet := toStringSet(standardFilters["vpc-id"])

	s.mu.Lock()
	items := make([]LocalGatewayRouteTableVpcAssociation, 0, len(s.localGatewayRouteTableVpcAssociations))
	for _, association := range s.localGatewayRouteTableVpcAssociations {
		if association == nil {
			continue
		}
		associationID := association.LocalGatewayRouteTableVpcAssociationID
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[associationID]; !ok {
				continue
			}
		}
		if len(associationIDFilterSet) > 0 {
			if _, ok := associationIDFilterSet[associationID]; !ok {
				continue
			}
		}
		if len(localGatewayIDFilterSet) > 0 {
			if _, ok := localGatewayIDFilterSet[association.LocalGatewayID]; !ok {
				continue
			}
		}
		if len(localGatewayRouteTableIDFilterSet) > 0 {
			if _, ok := localGatewayRouteTableIDFilterSet[association.LocalGatewayRouteTableID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[association.OwnerID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(association.State)]; !ok {
				continue
			}
		}
		if len(vpcIDFilterSet) > 0 {
			if _, ok := vpcIDFilterSet[association.VpcID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(association.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage108LocalGatewayRouteTableVpcAssociation(association))
	}
	s.mu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].LocalGatewayRouteTableVpcAssociationID < items[j].LocalGatewayRouteTableVpcAssociationID
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LocalGatewayRouteTableVpcAssociation(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeLocalGatewayRouteTables(routeTableIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]LocalGatewayRouteTable, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(routeTableIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	routeTableIDFilterSet := toStringSet(standardFilters["local-gateway-route-table-id"])
	localGatewayIDFilterSet := toStringSet(standardFilters["local-gateway-id"])
	modeFilterSet := toLowerStringSet(standardFilters["mode"])
	outpostARNFilterSet := toStringSet(standardFilters["outpost-arn"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.localGatewayRouteTables))
	for routeTableID := range s.localGatewayRouteTables {
		ids = append(ids, routeTableID)
	}
	sort.Strings(ids)

	items := make([]LocalGatewayRouteTable, 0, len(ids))
	for _, routeTableID := range ids {
		routeTable := s.localGatewayRouteTables[routeTableID]
		if routeTable == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[routeTableID]; !ok {
				continue
			}
		}
		if len(routeTableIDFilterSet) > 0 {
			if _, ok := routeTableIDFilterSet[routeTableID]; !ok {
				continue
			}
		}
		if len(localGatewayIDFilterSet) > 0 {
			if _, ok := localGatewayIDFilterSet[routeTable.LocalGatewayID]; !ok {
				continue
			}
		}
		if len(modeFilterSet) > 0 {
			if _, ok := modeFilterSet[strings.ToLower(routeTable.Mode)]; !ok {
				continue
			}
		}
		if len(outpostARNFilterSet) > 0 {
			if _, ok := outpostARNFilterSet[routeTable.OutpostARN]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[routeTable.OwnerID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(routeTable.State)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(routeTable.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage108LocalGatewayRouteTable(routeTable))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LocalGatewayRouteTable(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeLocalGatewayVirtualInterfaceGroups(groupIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]LocalGatewayVirtualInterfaceGroup, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(groupIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	groupIDFilterSet := toStringSet(standardFilters["local-gateway-virtual-interface-group-id"])
	localGatewayIDFilterSet := toStringSet(standardFilters["local-gateway-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(append(append([]string{}, standardFilters["state"]...), standardFilters["configuration-state"]...))
	localBgpASNFilterSet := toStringSet(standardFilters["local-bgp-asn"])
	localBgpASNExtendedFilterSet := toStringSet(standardFilters["local-bgp-asn-extended"])
	virtualInterfaceIDFilterSet := toStringSet(standardFilters["local-gateway-virtual-interface-id"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.localGatewayVirtualInterfaceGroups))
	for groupID := range s.localGatewayVirtualInterfaceGroups {
		ids = append(ids, groupID)
	}
	sort.Strings(ids)

	items := make([]LocalGatewayVirtualInterfaceGroup, 0, len(ids))
	for _, groupID := range ids {
		group := s.localGatewayVirtualInterfaceGroups[groupID]
		if group == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[groupID]; !ok {
				continue
			}
		}
		if len(groupIDFilterSet) > 0 {
			if _, ok := groupIDFilterSet[groupID]; !ok {
				continue
			}
		}
		if len(localGatewayIDFilterSet) > 0 {
			if _, ok := localGatewayIDFilterSet[group.LocalGatewayID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[group.OwnerID]; !ok {
				continue
			}
		}
		state := strings.TrimSpace(group.ConfigurationState)
		if state == "" {
			state = "available"
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(state)]; !ok {
				continue
			}
		}
		if len(localBgpASNFilterSet) > 0 {
			localASN := ""
			if group.LocalBgpASN != nil {
				localASN = strconv.FormatInt(int64(*group.LocalBgpASN), 10)
			}
			if _, ok := localBgpASNFilterSet[localASN]; !ok {
				continue
			}
		}
		if len(localBgpASNExtendedFilterSet) > 0 {
			localASNExtended := ""
			if group.LocalBgpASNExtended != nil {
				localASNExtended = strconv.FormatInt(*group.LocalBgpASNExtended, 10)
			}
			if _, ok := localBgpASNExtendedFilterSet[localASNExtended]; !ok {
				continue
			}
		}

		item := cloneStage109LocalGatewayVirtualInterfaceGroup(group)
		if len(item.LocalGatewayVirtualInterfaceIDs) == 0 {
			derivedIDs := make([]string, 0)
			for _, virtualInterface := range s.localGatewayVirtualInterfaces {
				if virtualInterface == nil {
					continue
				}
				if virtualInterface.LocalGatewayVirtualInterfaceGroupID != groupID {
					continue
				}
				derivedIDs = append(derivedIDs, virtualInterface.LocalGatewayVirtualInterfaceID)
			}
			sort.Strings(derivedIDs)
			item.LocalGatewayVirtualInterfaceIDs = dedupeTrimmedStrings(derivedIDs)
		}
		if len(virtualInterfaceIDFilterSet) > 0 {
			if !containsAnyString(item.LocalGatewayVirtualInterfaceIDs, virtualInterfaceIDFilterSet) {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LocalGatewayVirtualInterfaceGroup(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeLocalGatewayVirtualInterfaces(virtualInterfaceIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]LocalGatewayVirtualInterface, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(virtualInterfaceIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	virtualInterfaceIDFilterSet := toStringSet(standardFilters["local-gateway-virtual-interface-id"])
	virtualInterfaceGroupIDFilterSet := toStringSet(standardFilters["local-gateway-virtual-interface-group-id"])
	localGatewayIDFilterSet := toStringSet(standardFilters["local-gateway-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	outpostLagIDFilterSet := toStringSet(standardFilters["outpost-lag-id"])
	stateFilterSet := toLowerStringSet(append(append([]string{}, standardFilters["state"]...), standardFilters["configuration-state"]...))
	vlanFilterSet := toStringSet(standardFilters["vlan"])
	localBgpASNFilterSet := toStringSet(standardFilters["local-bgp-asn"])
	peerBgpASNFilterSet := toStringSet(standardFilters["peer-bgp-asn"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.localGatewayVirtualInterfaces))
	for virtualInterfaceID := range s.localGatewayVirtualInterfaces {
		ids = append(ids, virtualInterfaceID)
	}
	sort.Strings(ids)

	items := make([]LocalGatewayVirtualInterface, 0, len(ids))
	for _, virtualInterfaceID := range ids {
		virtualInterface := s.localGatewayVirtualInterfaces[virtualInterfaceID]
		if virtualInterface == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[virtualInterfaceID]; !ok {
				continue
			}
		}
		if len(virtualInterfaceIDFilterSet) > 0 {
			if _, ok := virtualInterfaceIDFilterSet[virtualInterfaceID]; !ok {
				continue
			}
		}
		if len(virtualInterfaceGroupIDFilterSet) > 0 {
			if _, ok := virtualInterfaceGroupIDFilterSet[virtualInterface.LocalGatewayVirtualInterfaceGroupID]; !ok {
				continue
			}
		}
		if len(localGatewayIDFilterSet) > 0 {
			if _, ok := localGatewayIDFilterSet[virtualInterface.LocalGatewayID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[virtualInterface.OwnerID]; !ok {
				continue
			}
		}
		if len(outpostLagIDFilterSet) > 0 {
			if _, ok := outpostLagIDFilterSet[virtualInterface.OutpostLagID]; !ok {
				continue
			}
		}
		state := strings.TrimSpace(virtualInterface.ConfigurationState)
		if state == "" {
			state = "pending"
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(state)]; !ok {
				continue
			}
		}
		if len(vlanFilterSet) > 0 {
			vlan := ""
			if virtualInterface.VLAN != nil {
				vlan = strconv.FormatInt(int64(*virtualInterface.VLAN), 10)
			}
			if _, ok := vlanFilterSet[vlan]; !ok {
				continue
			}
		}
		if len(localBgpASNFilterSet) > 0 {
			localASN := ""
			if virtualInterface.LocalBgpASN != nil {
				localASN = strconv.FormatInt(int64(*virtualInterface.LocalBgpASN), 10)
			}
			if _, ok := localBgpASNFilterSet[localASN]; !ok {
				continue
			}
		}
		if len(peerBgpASNFilterSet) > 0 {
			peerASN := ""
			if virtualInterface.PeerBgpASN != nil {
				peerASN = strconv.FormatInt(int64(*virtualInterface.PeerBgpASN), 10)
			}
			if _, ok := peerBgpASNFilterSet[peerASN]; !ok {
				continue
			}
		}
		if !matchesTagFilters(virtualInterface.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, cloneStage108LocalGatewayVirtualInterface(virtualInterface))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LocalGatewayVirtualInterface(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeLocalGateways(localGatewayIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]LocalGateway, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(localGatewayIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	localGatewayIDFilterSet := toStringSet(standardFilters["local-gateway-id"])
	outpostARNFilterSet := toStringSet(standardFilters["outpost-arn"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])

	s.mu.Lock()
	gatewayByID := map[string]*LocalGateway{}
	for _, routeTable := range s.localGatewayRouteTables {
		if routeTable == nil {
			continue
		}
		localGatewayID := strings.TrimSpace(routeTable.LocalGatewayID)
		if localGatewayID == "" {
			continue
		}
		gateway := gatewayByID[localGatewayID]
		if gateway == nil {
			gateway = &LocalGateway{
				LocalGatewayID: localGatewayID,
				OutpostARN:     strings.TrimSpace(routeTable.OutpostARN),
				OwnerID:        strings.TrimSpace(routeTable.OwnerID),
				State:          "available",
				Tags:           map[string]string{},
			}
			gatewayByID[localGatewayID] = gateway
		}
		if gateway.OutpostARN == "" {
			gateway.OutpostARN = strings.TrimSpace(routeTable.OutpostARN)
		}
	}
	for _, group := range s.localGatewayVirtualInterfaceGroups {
		if group == nil {
			continue
		}
		localGatewayID := strings.TrimSpace(group.LocalGatewayID)
		if localGatewayID == "" {
			continue
		}
		if gatewayByID[localGatewayID] == nil {
			gatewayByID[localGatewayID] = &LocalGateway{
				LocalGatewayID: localGatewayID,
				OutpostARN:     stage119DefaultLocalGatewayOutpostARN(),
				OwnerID:        firstNonEmptyString(group.OwnerID, DefaultAccountID),
				State:          "available",
				Tags:           map[string]string{},
			}
		}
	}
	for _, virtualInterface := range s.localGatewayVirtualInterfaces {
		if virtualInterface == nil {
			continue
		}
		localGatewayID := strings.TrimSpace(virtualInterface.LocalGatewayID)
		if localGatewayID == "" {
			continue
		}
		if gatewayByID[localGatewayID] == nil {
			gatewayByID[localGatewayID] = &LocalGateway{
				LocalGatewayID: localGatewayID,
				OutpostARN:     stage119DefaultLocalGatewayOutpostARN(),
				OwnerID:        firstNonEmptyString(virtualInterface.OwnerID, DefaultAccountID),
				State:          "available",
				Tags:           map[string]string{},
			}
		}
	}

	ids := make([]string, 0, len(gatewayByID))
	for localGatewayID := range gatewayByID {
		ids = append(ids, localGatewayID)
	}
	sort.Strings(ids)

	items := make([]LocalGateway, 0, len(ids))
	for _, localGatewayID := range ids {
		gateway := gatewayByID[localGatewayID]
		if gateway == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[localGatewayID]; !ok {
				continue
			}
		}
		if len(localGatewayIDFilterSet) > 0 {
			if _, ok := localGatewayIDFilterSet[localGatewayID]; !ok {
				continue
			}
		}
		outpostARN := firstNonEmptyString(gateway.OutpostARN, stage119DefaultLocalGatewayOutpostARN())
		if len(outpostARNFilterSet) > 0 {
			if _, ok := outpostARNFilterSet[outpostARN]; !ok {
				continue
			}
		}
		ownerID := firstNonEmptyString(gateway.OwnerID, DefaultAccountID)
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[ownerID]; !ok {
				continue
			}
		}
		state := firstNonEmptyString(gateway.State, "available")
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(state)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(gateway.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, LocalGateway{
			LocalGatewayID: localGatewayID,
			OutpostARN:     outpostARN,
			OwnerID:        ownerID,
			State:          state,
			Tags:           cloneStringMap(gateway.Tags),
		})
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LocalGateway(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeLockedSnapshots(snapshotIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]LockedSnapshotInfo, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(snapshotIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, _, _ := splitEC2Filters(filters)
	snapshotIDFilterSet := toStringSet(standardFilters["snapshot-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	lockStateFilterSet := toLowerStringSet(standardFilters["lock-state"])

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		for snapshotID := range s.snapshots {
			candidateIDs = append(candidateIDs, snapshotID)
		}
		sort.Strings(candidateIDs)
	}

	items := make([]LockedSnapshotInfo, 0, len(candidateIDs))
	for _, snapshotID := range candidateIDs {
		snapshot := s.snapshots[snapshotID]
		if snapshot == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[snapshotID]; !ok {
				continue
			}
		}
		if len(snapshotIDFilterSet) > 0 {
			if _, ok := snapshotIDFilterSet[snapshotID]; !ok {
				continue
			}
		}

		item := stage119LockedSnapshotInfoFromSnapshot(snapshot)
		if locked := s.lockedSnapshots[snapshotID]; locked != nil {
			item = cloneStage119LockedSnapshotInfo(*locked)
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[item.OwnerID]; !ok {
				continue
			}
		}
		if len(lockStateFilterSet) > 0 {
			if _, ok := lockStateFilterSet[strings.ToLower(item.LockState)]; !ok {
				continue
			}
		}

		items = append(items, cloneStage119LockedSnapshotInfo(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LockedSnapshotInfo(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeMacHosts(hostIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]MacHost, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(hostIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, _, _ := splitEC2Filters(filters)
	hostIDFilterSet := toStringSet(standardFilters["host-id"])
	macOSVersionFilterSet := toLowerStringSet(append(append([]string{}, standardFilters["mac-os-latest-supported-version"]...), standardFilters["macos-latest-supported-version"]...))

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		for hostID := range s.dedicatedHosts {
			candidateIDs = append(candidateIDs, hostID)
		}
		sort.Strings(candidateIDs)
	}

	items := make([]MacHost, 0, len(candidateIDs))
	for _, hostID := range candidateIDs {
		host := s.dedicatedHosts[hostID]
		if host == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[hostID]; !ok {
				continue
			}
		}
		if len(hostIDFilterSet) > 0 {
			if _, ok := hostIDFilterSet[hostID]; !ok {
				continue
			}
		}

		item := MacHost{HostID: hostID, MacOSLatestSupportedVersion: stage119MacHostVersions(host)}
		if len(macOSVersionFilterSet) > 0 {
			loweredVersions := make([]string, 0, len(item.MacOSLatestSupportedVersion))
			for _, version := range item.MacOSLatestSupportedVersion {
				loweredVersions = append(loweredVersions, strings.ToLower(strings.TrimSpace(version)))
			}
			if !containsAnyString(loweredVersions, macOSVersionFilterSet) {
				continue
			}
		}
		items = append(items, cloneStage119MacHost(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]MacHost(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeMacModificationTasks(taskIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]MacModificationTask, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(taskIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	taskIDFilterSet := toStringSet(standardFilters["mac-modification-task-id"])
	instanceIDFilterSet := toStringSet(standardFilters["instance-id"])
	taskStateFilterSet := toLowerStringSet(standardFilters["task-state"])
	taskTypeFilterSet := toLowerStringSet(standardFilters["task-type"])

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		for taskID := range s.macModificationTasks {
			candidateIDs = append(candidateIDs, taskID)
		}
		sort.Strings(candidateIDs)
	}

	items := make([]MacModificationTask, 0, len(candidateIDs))
	for _, taskID := range candidateIDs {
		task := s.macModificationTasks[taskID]
		if task == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[taskID]; !ok {
				continue
			}
		}
		if len(taskIDFilterSet) > 0 {
			if _, ok := taskIDFilterSet[taskID]; !ok {
				continue
			}
		}

		item := cloneStage107MacModificationTask(task)
		if strings.EqualFold(item.TaskState, "completed") {
			item.TaskState = "successful"
		}
		if strings.EqualFold(item.TaskType, "restore-volume-permissions") {
			item.TaskType = "volume-ownership-delegation"
		}

		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[item.InstanceID]; !ok {
				continue
			}
		}
		if len(taskStateFilterSet) > 0 {
			if _, ok := taskStateFilterSet[strings.ToLower(item.TaskState)]; !ok {
				continue
			}
		}
		if len(taskTypeFilterSet) > 0 {
			if _, ok := taskTypeFilterSet[strings.ToLower(item.TaskType)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]MacModificationTask(nil), items[start:end]...), outputToken, nil
}

func stage119DefaultLocalGatewayOutpostARN() string {
	return fmt.Sprintf("arn:aws:outposts:%s:%s:outpost/op-00000000", DefaultRegion, DefaultAccountID)
}

func stage119LockedSnapshotInfoFromSnapshot(snapshot *Snapshot) LockedSnapshotInfo {
	createdOn := time.Now().UTC()
	if snapshot != nil && !snapshot.StartTime.IsZero() {
		createdOn = snapshot.StartTime.UTC()
	}
	lockDuration := int32(24)
	coolOffPeriod := int32(0)
	lockExpiresOn := createdOn.Add(24 * time.Hour)
	coolOffPeriodExpiresOn := lockExpiresOn
	return LockedSnapshotInfo{
		CoolOffPeriod:          &coolOffPeriod,
		CoolOffPeriodExpiresOn: &coolOffPeriodExpiresOn,
		LockCreatedOn:          &createdOn,
		LockDuration:           &lockDuration,
		LockDurationStartTime:  &createdOn,
		LockExpiresOn:          &lockExpiresOn,
		LockState:              "compliance",
		OwnerID:                DefaultAccountID,
		SnapshotID:             snapshot.ID,
	}
}

func stage119MacHostVersions(host *DedicatedHost) []string {
	instanceType := strings.ToLower(strings.TrimSpace(host.InstanceType))
	switch {
	case strings.Contains(instanceType, "mac2"):
		return []string{"macos-14", "macos-13"}
	case strings.Contains(instanceType, "mac1"):
		return []string{"macos-13", "macos-12"}
	default:
		return []string{"macos-14"}
	}
}

func cloneStage119LocalGateway(in *LocalGateway) LocalGateway {
	if in == nil {
		return LocalGateway{}
	}
	return LocalGateway{
		LocalGatewayID: in.LocalGatewayID,
		OutpostARN:     in.OutpostARN,
		OwnerID:        in.OwnerID,
		State:          in.State,
		Tags:           cloneStringMap(in.Tags),
	}
}

func cloneStage119LockedSnapshotInfo(in LockedSnapshotInfo) LockedSnapshotInfo {
	return LockedSnapshotInfo{
		CoolOffPeriod:          cloneInt32Pointer(in.CoolOffPeriod),
		CoolOffPeriodExpiresOn: cloneTimePointer(in.CoolOffPeriodExpiresOn),
		LockCreatedOn:          cloneTimePointer(in.LockCreatedOn),
		LockDuration:           cloneInt32Pointer(in.LockDuration),
		LockDurationStartTime:  cloneTimePointer(in.LockDurationStartTime),
		LockExpiresOn:          cloneTimePointer(in.LockExpiresOn),
		LockState:              in.LockState,
		OwnerID:                in.OwnerID,
		SnapshotID:             in.SnapshotID,
	}
}

func cloneStage119MacHost(in MacHost) MacHost {
	return MacHost{
		HostID:                      in.HostID,
		MacOSLatestSupportedVersion: append([]string(nil), in.MacOSLatestSupportedVersion...),
	}
}
