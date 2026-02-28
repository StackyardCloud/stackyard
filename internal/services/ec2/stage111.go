package ec2

import "strings"

type InstanceEventWindowStateChange struct {
	InstanceEventWindowID string
	State                 string
}

func (s *Service) DeleteFlowLogs(flowLogIDs []string) ([]UnsuccessfulItem, error) {
	flowLogIDs = dedupeTrimmedStrings(flowLogIDs)
	if len(flowLogIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	unsuccessful := make([]UnsuccessfulItem, 0)
	for _, flowLogID := range flowLogIDs {
		if _, ok := s.flowLogs[flowLogID]; !ok {
			unsuccessful = append(unsuccessful, UnsuccessfulItem{
				ResourceID: flowLogID,
				Code:       "InvalidFlowLogId.NotFound",
				Message:    "flow log not found",
			})
			continue
		}
		delete(s.flowLogs, flowLogID)
	}
	return unsuccessful, nil
}

func (s *Service) DeleteFpgaImage(fpgaImageID string) (bool, error) {
	fpgaImageID = strings.TrimSpace(fpgaImageID)
	if fpgaImageID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fpgaImages[fpgaImageID] == nil {
		return false, ErrNotFound
	}
	delete(s.fpgaImages, fpgaImageID)
	return true, nil
}

func (s *Service) DeleteInstanceConnectEndpoint(instanceConnectEndpointID string) (InstanceConnectEndpoint, error) {
	instanceConnectEndpointID = strings.TrimSpace(instanceConnectEndpointID)
	if instanceConnectEndpointID == "" {
		return InstanceConnectEndpoint{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.instanceConnectEndpoints[instanceConnectEndpointID]
	if endpoint == nil {
		return InstanceConnectEndpoint{}, ErrNotFound
	}
	out := cloneStage107InstanceConnectEndpoint(endpoint)
	out.State = "delete-complete"
	out.StateMessage = "deleted"
	delete(s.instanceConnectEndpoints, instanceConnectEndpointID)
	return out, nil
}

func (s *Service) DeleteInstanceEventWindow(instanceEventWindowID string, forceDelete *bool) (InstanceEventWindowStateChange, error) {
	instanceEventWindowID = strings.TrimSpace(instanceEventWindowID)
	if instanceEventWindowID == "" {
		return InstanceEventWindowStateChange{}, ErrInvalidParameter
	}
	_ = forceDelete

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instanceEventWindows[instanceEventWindowID] == nil {
		return InstanceEventWindowStateChange{}, ErrNotFound
	}
	delete(s.instanceEventWindows, instanceEventWindowID)
	delete(s.instanceEventWindowAssociations, instanceEventWindowID)

	return InstanceEventWindowStateChange{
		InstanceEventWindowID: instanceEventWindowID,
		State:                 "deleted",
	}, nil
}

func (s *Service) DeleteIpam(ipamID string, cascade *bool) (Ipam, error) {
	ipamID = strings.TrimSpace(ipamID)
	if ipamID == "" {
		return Ipam{}, ErrInvalidParameter
	}
	_ = cascade

	s.mu.Lock()
	defer s.mu.Unlock()

	ipam := s.ipams[ipamID]
	if ipam == nil {
		return Ipam{}, ErrNotFound
	}
	out := cloneStage107Ipam(ipam)
	delete(s.ipams, ipamID)

	scopeIDs := make(map[string]struct{})
	for scopeID, scope := range s.ipamScopes {
		if scope == nil || scope.IpamID != ipamID {
			continue
		}
		scopeIDs[scopeID] = struct{}{}
		delete(s.ipamScopes, scopeID)
	}
	for poolID, pool := range s.ipamPools {
		if pool == nil {
			continue
		}
		if _, ok := scopeIDs[pool.IpamScopeID]; ok {
			delete(s.ipamPools, poolID)
		}
	}
	for tokenID, token := range s.ipamExternalResourceVerificationTokens {
		if token == nil || token.IpamID != ipamID {
			continue
		}
		delete(s.ipamExternalResourceVerificationTokens, tokenID)
	}
	for associationID, association := range s.ipamResourceDiscoveryAssociations {
		if association == nil || association.IpamID != ipamID {
			continue
		}
		delete(s.ipamResourceDiscoveryAssociations, associationID)
		delete(s.ipamResourceDiscoveryAssociationByPair, association.IpamID+"|"+association.IpamResourceDiscoveryID)
	}

	return out, nil
}

func (s *Service) DeleteIpamExternalResourceVerificationToken(ipamExternalResourceVerificationTokenID string) (IpamExternalResourceVerificationToken, error) {
	ipamExternalResourceVerificationTokenID = strings.TrimSpace(ipamExternalResourceVerificationTokenID)
	if ipamExternalResourceVerificationTokenID == "" {
		return IpamExternalResourceVerificationToken{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	token := s.ipamExternalResourceVerificationTokens[ipamExternalResourceVerificationTokenID]
	if token == nil {
		return IpamExternalResourceVerificationToken{}, ErrNotFound
	}
	out := cloneStage107IpamExternalResourceVerificationToken(token)
	delete(s.ipamExternalResourceVerificationTokens, ipamExternalResourceVerificationTokenID)
	return out, nil
}

func (s *Service) DeleteIpamPool(ipamPoolID string, cascade *bool) (IpamPool, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	if ipamPoolID == "" {
		return IpamPool{}, ErrInvalidParameter
	}
	_ = cascade

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ipamPools[ipamPoolID]
	if pool == nil {
		return IpamPool{}, ErrNotFound
	}
	out := cloneStage108IpamPool(pool)
	delete(s.ipamPools, ipamPoolID)
	if scope := s.ipamScopes[pool.IpamScopeID]; scope != nil && scope.PoolCount > 0 {
		scope.PoolCount--
	}
	return out, nil
}

func (s *Service) DeleteIpamResourceDiscovery(ipamResourceDiscoveryID string) (IpamResourceDiscovery, error) {
	ipamResourceDiscoveryID = strings.TrimSpace(ipamResourceDiscoveryID)
	if ipamResourceDiscoveryID == "" {
		return IpamResourceDiscovery{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	discovery := s.ipamResourceDiscoveries[ipamResourceDiscoveryID]
	if discovery == nil {
		return IpamResourceDiscovery{}, ErrNotFound
	}
	out := cloneStage108IpamResourceDiscovery(discovery)
	delete(s.ipamResourceDiscoveries, ipamResourceDiscoveryID)

	for associationID, association := range s.ipamResourceDiscoveryAssociations {
		if association == nil || association.IpamResourceDiscoveryID != ipamResourceDiscoveryID {
			continue
		}
		delete(s.ipamResourceDiscoveryAssociations, associationID)
		delete(s.ipamResourceDiscoveryAssociationByPair, association.IpamID+"|"+association.IpamResourceDiscoveryID)
	}

	return out, nil
}

func (s *Service) DeleteIpamScope(ipamScopeID string) (IpamScope, error) {
	ipamScopeID = strings.TrimSpace(ipamScopeID)
	if ipamScopeID == "" {
		return IpamScope{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	scope := s.ipamScopes[ipamScopeID]
	if scope == nil {
		return IpamScope{}, ErrNotFound
	}
	out := cloneStage108IpamScope(scope)
	delete(s.ipamScopes, ipamScopeID)

	for poolID, pool := range s.ipamPools {
		if pool == nil || pool.IpamScopeID != ipamScopeID {
			continue
		}
		delete(s.ipamPools, poolID)
	}

	return out, nil
}

func (s *Service) DeleteLaunchTemplate(launchTemplateID string, launchTemplateName string) (LaunchTemplate, error) {
	launchTemplateID = strings.TrimSpace(launchTemplateID)
	launchTemplateName = strings.TrimSpace(launchTemplateName)
	if launchTemplateID == "" && launchTemplateName == "" {
		return LaunchTemplate{}, ErrInvalidParameter
	}
	if launchTemplateID != "" && launchTemplateName != "" {
		return LaunchTemplate{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if launchTemplateID == "" {
		launchTemplateID = s.launchTemplateNameIndex[launchTemplateName]
	}
	launchTemplate := s.launchTemplates[launchTemplateID]
	if launchTemplate == nil {
		return LaunchTemplate{}, ErrNotFound
	}
	if launchTemplateName != "" && launchTemplate.LaunchTemplateName != launchTemplateName {
		return LaunchTemplate{}, ErrInvalidParameter
	}

	out := cloneStage108LaunchTemplate(launchTemplate)
	delete(s.launchTemplates, launchTemplateID)
	delete(s.launchTemplateNameIndex, launchTemplate.LaunchTemplateName)
	delete(s.launchTemplateVersions, launchTemplateID)
	return out, nil
}
