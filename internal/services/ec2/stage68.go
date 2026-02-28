package ec2

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type VerifiedAccessInstance struct {
	ID                           string
	CidrEndpointsCustomSubDomain string
	Description                  string
	FIPSEnabled                  bool
	CreationTime                 string
	LastUpdatedTime              string
	Tags                         map[string]string
	TrustProviderIDs             []string
}

type VerifiedAccessGroup struct {
	ID               string
	ARN              string
	VerifiedInstance string
	Description      string
	PolicyDocument   string
	Owner            string
	CreationTime     string
	LastUpdatedTime  string
	DeletionTime     *string
	Tags             map[string]string
}

type VerifiedAccessEndpoint struct {
	ID                string
	VerifiedGroup     string
	VerifiedInstance  string
	ApplicationDomain string
	AttachmentType    string
	EndpointType      string
	Description       string
	DomainCertARN     string
	EndpointDomain    string
	SecurityGroupIDs  []string
	StatusCode        string
	StatusMessage     string
	CreationTime      string
	LastUpdatedTime   string
	DeletionTime      *string
	Tags              map[string]string
}

type VerifiedAccessTrustProvider struct {
	ID                      string
	PolicyReferenceName     string
	TrustProviderType       string
	UserTrustProviderType   string
	DeviceTrustProviderType string
	Description             string
	CreationTime            string
	LastUpdatedTime         string
	Tags                    map[string]string
}

func (s *Service) CreateVerifiedAccessInstance(
	cidrEndpointsCustomSubDomain *string,
	description *string,
	fipsEnabled *bool,
	tags []Tag,
) (VerifiedAccessInstance, error) {
	customSubDomain := ""
	if cidrEndpointsCustomSubDomain != nil {
		customSubDomain = strings.TrimSpace(*cidrEndpointsCustomSubDomain)
	}
	desc := ""
	if description != nil {
		desc = strings.TrimSpace(*description)
	}
	fips := false
	if fipsEnabled != nil {
		fips = *fipsEnabled
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	instance := &VerifiedAccessInstance{
		ID:                           s.nextIDLocked("vai"),
		CidrEndpointsCustomSubDomain: customSubDomain,
		Description:                  desc,
		FIPSEnabled:                  fips,
		CreationTime:                 now,
		LastUpdatedTime:              now,
		Tags:                         tagsToMap(tags),
		TrustProviderIDs:             []string{},
	}
	s.verifiedAccessInstances[instance.ID] = instance
	return cloneVerifiedAccessInstance(instance), nil
}

func (s *Service) DeleteVerifiedAccessInstance(verifiedAccessInstanceID string) (VerifiedAccessInstance, error) {
	verifiedAccessInstanceID = strings.TrimSpace(verifiedAccessInstanceID)
	if verifiedAccessInstanceID == "" {
		return VerifiedAccessInstance{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.verifiedAccessInstances[verifiedAccessInstanceID]
	if instance == nil {
		return VerifiedAccessInstance{}, ErrNotFound
	}
	for _, group := range s.verifiedAccessGroups {
		if group != nil && group.VerifiedInstance == verifiedAccessInstanceID {
			return VerifiedAccessInstance{}, ErrConflict
		}
	}
	for _, endpoint := range s.verifiedAccessEndpoints {
		if endpoint != nil && endpoint.VerifiedInstance == verifiedAccessInstanceID {
			return VerifiedAccessInstance{}, ErrConflict
		}
	}

	out := cloneVerifiedAccessInstance(instance)
	delete(s.verifiedAccessInstances, verifiedAccessInstanceID)
	return out, nil
}

func (s *Service) DescribeVerifiedAccessInstances(
	verifiedAccessInstanceIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VerifiedAccessInstance, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(verifiedAccessInstanceIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterIDSet := toStringSet(standardFilters["verified-access-instance-id"])
	filterDescriptionSet := toStringSet(standardFilters["description"])
	filterFipsEnabledSet := toLowerStringSet(standardFilters["fips-enabled"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VerifiedAccessInstance, 0, len(s.verifiedAccessInstances))
	for _, instance := range s.verifiedAccessInstances {
		if instance == nil {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[instance.ID]; !ok {
				continue
			}
		}
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[instance.ID]; !ok {
				continue
			}
		}
		if len(filterDescriptionSet) > 0 {
			if _, ok := filterDescriptionSet[instance.Description]; !ok {
				continue
			}
		}
		if len(filterFipsEnabledSet) > 0 {
			if _, ok := filterFipsEnabledSet[strings.ToLower(fmt.Sprintf("%t", instance.FIPSEnabled))]; !ok {
				continue
			}
		}
		if !matchesTagFilters(instance.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneVerifiedAccessInstance(instance))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VerifiedAccessInstance(nil), items[start:end]...), outputToken, nil
}

func (s *Service) CreateVerifiedAccessGroup(
	verifiedAccessInstanceID string,
	description *string,
	policyDocument *string,
	tags []Tag,
) (VerifiedAccessGroup, error) {
	verifiedAccessInstanceID = strings.TrimSpace(verifiedAccessInstanceID)
	if verifiedAccessInstanceID == "" {
		return VerifiedAccessGroup{}, ErrInvalidParameter
	}
	desc := ""
	if description != nil {
		desc = strings.TrimSpace(*description)
	}
	policy := ""
	if policyDocument != nil {
		policy = strings.TrimSpace(*policyDocument)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.verifiedAccessInstances[verifiedAccessInstanceID] == nil {
		return VerifiedAccessGroup{}, ErrNotFound
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id := s.nextIDLocked("vag")
	group := &VerifiedAccessGroup{
		ID:               id,
		ARN:              fmt.Sprintf("arn:aws:ec2:%s:%s:verified-access-group/%s", DefaultRegion, DefaultAccountID, id),
		VerifiedInstance: verifiedAccessInstanceID,
		Description:      desc,
		PolicyDocument:   policy,
		Owner:            DefaultAccountID,
		CreationTime:     now,
		LastUpdatedTime:  now,
		Tags:             tagsToMap(tags),
	}
	s.verifiedAccessGroups[group.ID] = group
	return cloneVerifiedAccessGroup(group), nil
}

func (s *Service) DeleteVerifiedAccessGroup(verifiedAccessGroupID string) (VerifiedAccessGroup, error) {
	verifiedAccessGroupID = strings.TrimSpace(verifiedAccessGroupID)
	if verifiedAccessGroupID == "" {
		return VerifiedAccessGroup{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	group := s.verifiedAccessGroups[verifiedAccessGroupID]
	if group == nil {
		return VerifiedAccessGroup{}, ErrNotFound
	}
	for _, endpoint := range s.verifiedAccessEndpoints {
		if endpoint != nil && endpoint.VerifiedGroup == verifiedAccessGroupID {
			return VerifiedAccessGroup{}, ErrConflict
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	group.DeletionTime = &now
	out := cloneVerifiedAccessGroup(group)
	delete(s.verifiedAccessGroups, verifiedAccessGroupID)
	return out, nil
}

func (s *Service) DescribeVerifiedAccessGroups(
	verifiedAccessGroupIDs []string,
	verifiedAccessInstanceID *string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VerifiedAccessGroup, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(verifiedAccessGroupIDs))
	instanceIDFilter := ""
	if verifiedAccessInstanceID != nil {
		instanceIDFilter = strings.TrimSpace(*verifiedAccessInstanceID)
	}
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterIDSet := toStringSet(standardFilters["verified-access-group-id"])
	filterInstanceIDSet := toStringSet(standardFilters["verified-access-instance-id"])
	filterOwnerSet := toStringSet(standardFilters["owner"])
	filterDescriptionSet := toStringSet(standardFilters["description"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VerifiedAccessGroup, 0, len(s.verifiedAccessGroups))
	for _, group := range s.verifiedAccessGroups {
		if group == nil {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[group.ID]; !ok {
				continue
			}
		}
		if instanceIDFilter != "" && group.VerifiedInstance != instanceIDFilter {
			continue
		}
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[group.ID]; !ok {
				continue
			}
		}
		if len(filterInstanceIDSet) > 0 {
			if _, ok := filterInstanceIDSet[group.VerifiedInstance]; !ok {
				continue
			}
		}
		if len(filterOwnerSet) > 0 {
			if _, ok := filterOwnerSet[group.Owner]; !ok {
				continue
			}
		}
		if len(filterDescriptionSet) > 0 {
			if _, ok := filterDescriptionSet[group.Description]; !ok {
				continue
			}
		}
		if !matchesTagFilters(group.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneVerifiedAccessGroup(group))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VerifiedAccessGroup(nil), items[start:end]...), outputToken, nil
}

func (s *Service) CreateVerifiedAccessEndpoint(
	verifiedAccessGroupID, attachmentType, endpointType string,
	applicationDomain, description, domainCertificateARN, endpointDomainPrefix *string,
	securityGroupIDs []string,
	tags []Tag,
) (VerifiedAccessEndpoint, error) {
	verifiedAccessGroupID = strings.TrimSpace(verifiedAccessGroupID)
	attachmentType = strings.ToLower(strings.TrimSpace(attachmentType))
	endpointType = strings.ToLower(strings.TrimSpace(endpointType))
	if verifiedAccessGroupID == "" || attachmentType == "" || endpointType == "" {
		return VerifiedAccessEndpoint{}, ErrInvalidParameter
	}
	if attachmentType != "vpc" {
		return VerifiedAccessEndpoint{}, ErrInvalidParameter
	}
	switch endpointType {
	case "cidr", "load-balancer", "network-interface", "rds":
	default:
		return VerifiedAccessEndpoint{}, ErrInvalidParameter
	}

	appDomain := ""
	if applicationDomain != nil {
		appDomain = strings.TrimSpace(*applicationDomain)
	}
	desc := ""
	if description != nil {
		desc = strings.TrimSpace(*description)
	}
	certARN := ""
	if domainCertificateARN != nil {
		certARN = strings.TrimSpace(*domainCertificateARN)
	}
	prefix := "stackyard"
	if endpointDomainPrefix != nil && strings.TrimSpace(*endpointDomainPrefix) != "" {
		prefix = strings.TrimSpace(*endpointDomainPrefix)
	}
	sgIDs := dedupeTrimmedStrings(securityGroupIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	group := s.verifiedAccessGroups[verifiedAccessGroupID]
	if group == nil {
		return VerifiedAccessEndpoint{}, ErrNotFound
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id := s.nextIDLocked("vae")
	endpoint := &VerifiedAccessEndpoint{
		ID:                id,
		VerifiedGroup:     verifiedAccessGroupID,
		VerifiedInstance:  group.VerifiedInstance,
		ApplicationDomain: appDomain,
		AttachmentType:    attachmentType,
		EndpointType:      endpointType,
		Description:       desc,
		DomainCertARN:     certARN,
		EndpointDomain:    fmt.Sprintf("%s.%s.%s.verified-access.local", prefix, id, DefaultRegion),
		SecurityGroupIDs:  sgIDs,
		StatusCode:        "active",
		StatusMessage:     "endpoint is active",
		CreationTime:      now,
		LastUpdatedTime:   now,
		Tags:              tagsToMap(tags),
	}
	s.verifiedAccessEndpoints[endpoint.ID] = endpoint
	return cloneVerifiedAccessEndpoint(endpoint), nil
}

func (s *Service) DeleteVerifiedAccessEndpoint(verifiedAccessEndpointID string) (VerifiedAccessEndpoint, error) {
	verifiedAccessEndpointID = strings.TrimSpace(verifiedAccessEndpointID)
	if verifiedAccessEndpointID == "" {
		return VerifiedAccessEndpoint{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.verifiedAccessEndpoints[verifiedAccessEndpointID]
	if endpoint == nil {
		return VerifiedAccessEndpoint{}, ErrNotFound
	}

	now := time.Now().UTC().Format(time.RFC3339)
	endpoint.DeletionTime = &now
	endpoint.StatusCode = "deleting"
	endpoint.StatusMessage = "endpoint is deleting"
	endpoint.LastUpdatedTime = now

	out := cloneVerifiedAccessEndpoint(endpoint)
	delete(s.verifiedAccessEndpoints, verifiedAccessEndpointID)
	return out, nil
}

func (s *Service) DescribeVerifiedAccessEndpoints(
	verifiedAccessEndpointIDs []string,
	verifiedAccessGroupID *string,
	verifiedAccessInstanceID *string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VerifiedAccessEndpoint, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(verifiedAccessEndpointIDs))
	groupIDFilter := ""
	if verifiedAccessGroupID != nil {
		groupIDFilter = strings.TrimSpace(*verifiedAccessGroupID)
	}
	instanceIDFilter := ""
	if verifiedAccessInstanceID != nil {
		instanceIDFilter = strings.TrimSpace(*verifiedAccessInstanceID)
	}
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterIDSet := toStringSet(standardFilters["verified-access-endpoint-id"])
	filterGroupIDSet := toStringSet(standardFilters["verified-access-group-id"])
	filterInstanceIDSet := toStringSet(standardFilters["verified-access-instance-id"])
	filterAttachmentSet := toLowerStringSet(standardFilters["attachment-type"])
	filterEndpointTypeSet := toLowerStringSet(standardFilters["endpoint-type"])
	filterAppDomainSet := toStringSet(standardFilters["application-domain"])
	filterStatusCodeSet := toLowerStringSet(standardFilters["status.code"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VerifiedAccessEndpoint, 0, len(s.verifiedAccessEndpoints))
	for _, endpoint := range s.verifiedAccessEndpoints {
		if endpoint == nil {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[endpoint.ID]; !ok {
				continue
			}
		}
		if groupIDFilter != "" && endpoint.VerifiedGroup != groupIDFilter {
			continue
		}
		if instanceIDFilter != "" && endpoint.VerifiedInstance != instanceIDFilter {
			continue
		}
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[endpoint.ID]; !ok {
				continue
			}
		}
		if len(filterGroupIDSet) > 0 {
			if _, ok := filterGroupIDSet[endpoint.VerifiedGroup]; !ok {
				continue
			}
		}
		if len(filterInstanceIDSet) > 0 {
			if _, ok := filterInstanceIDSet[endpoint.VerifiedInstance]; !ok {
				continue
			}
		}
		if len(filterAttachmentSet) > 0 {
			if _, ok := filterAttachmentSet[strings.ToLower(endpoint.AttachmentType)]; !ok {
				continue
			}
		}
		if len(filterEndpointTypeSet) > 0 {
			if _, ok := filterEndpointTypeSet[strings.ToLower(endpoint.EndpointType)]; !ok {
				continue
			}
		}
		if len(filterAppDomainSet) > 0 {
			if _, ok := filterAppDomainSet[endpoint.ApplicationDomain]; !ok {
				continue
			}
		}
		if len(filterStatusCodeSet) > 0 {
			if _, ok := filterStatusCodeSet[strings.ToLower(endpoint.StatusCode)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(endpoint.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneVerifiedAccessEndpoint(endpoint))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VerifiedAccessEndpoint(nil), items[start:end]...), outputToken, nil
}

func (s *Service) CreateVerifiedAccessTrustProvider(
	policyReferenceName, trustProviderType, userTrustProviderType, deviceTrustProviderType string,
	description *string,
	tags []Tag,
) (VerifiedAccessTrustProvider, error) {
	policyReferenceName = strings.TrimSpace(policyReferenceName)
	trustProviderType = strings.ToLower(strings.TrimSpace(trustProviderType))
	userTrustProviderType = strings.ToLower(strings.TrimSpace(userTrustProviderType))
	deviceTrustProviderType = strings.ToLower(strings.TrimSpace(deviceTrustProviderType))
	if policyReferenceName == "" || trustProviderType == "" {
		return VerifiedAccessTrustProvider{}, ErrInvalidParameter
	}
	switch trustProviderType {
	case "user":
		if userTrustProviderType == "" {
			userTrustProviderType = "iam-identity-center"
		}
	case "device":
		if deviceTrustProviderType == "" {
			deviceTrustProviderType = "jamf"
		}
	default:
		return VerifiedAccessTrustProvider{}, ErrInvalidParameter
	}
	desc := ""
	if description != nil {
		desc = strings.TrimSpace(*description)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	provider := &VerifiedAccessTrustProvider{
		ID:                      s.nextIDLocked("vatp"),
		PolicyReferenceName:     policyReferenceName,
		TrustProviderType:       trustProviderType,
		UserTrustProviderType:   userTrustProviderType,
		DeviceTrustProviderType: deviceTrustProviderType,
		Description:             desc,
		CreationTime:            now,
		LastUpdatedTime:         now,
		Tags:                    tagsToMap(tags),
	}
	s.verifiedAccessTrustProviders[provider.ID] = provider
	return cloneVerifiedAccessTrustProvider(provider), nil
}

func (s *Service) DeleteVerifiedAccessTrustProvider(verifiedAccessTrustProviderID string) (VerifiedAccessTrustProvider, error) {
	verifiedAccessTrustProviderID = strings.TrimSpace(verifiedAccessTrustProviderID)
	if verifiedAccessTrustProviderID == "" {
		return VerifiedAccessTrustProvider{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	provider := s.verifiedAccessTrustProviders[verifiedAccessTrustProviderID]
	if provider == nil {
		return VerifiedAccessTrustProvider{}, ErrNotFound
	}

	out := cloneVerifiedAccessTrustProvider(provider)
	delete(s.verifiedAccessTrustProviders, verifiedAccessTrustProviderID)
	return out, nil
}

func (s *Service) DescribeVerifiedAccessTrustProviders(
	verifiedAccessTrustProviderIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VerifiedAccessTrustProvider, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(verifiedAccessTrustProviderIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterIDSet := toStringSet(standardFilters["verified-access-trust-provider-id"])
	filterPolicyRefSet := toStringSet(standardFilters["policy-reference-name"])
	filterTrustProviderTypeSet := toLowerStringSet(standardFilters["trust-provider-type"])
	filterUserTrustProviderTypeSet := toLowerStringSet(standardFilters["user-trust-provider-type"])
	filterDeviceTrustProviderTypeSet := toLowerStringSet(standardFilters["device-trust-provider-type"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VerifiedAccessTrustProvider, 0, len(s.verifiedAccessTrustProviders))
	for _, provider := range s.verifiedAccessTrustProviders {
		if provider == nil {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[provider.ID]; !ok {
				continue
			}
		}
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[provider.ID]; !ok {
				continue
			}
		}
		if len(filterPolicyRefSet) > 0 {
			if _, ok := filterPolicyRefSet[provider.PolicyReferenceName]; !ok {
				continue
			}
		}
		if len(filterTrustProviderTypeSet) > 0 {
			if _, ok := filterTrustProviderTypeSet[strings.ToLower(provider.TrustProviderType)]; !ok {
				continue
			}
		}
		if len(filterUserTrustProviderTypeSet) > 0 {
			if _, ok := filterUserTrustProviderTypeSet[strings.ToLower(provider.UserTrustProviderType)]; !ok {
				continue
			}
		}
		if len(filterDeviceTrustProviderTypeSet) > 0 {
			if _, ok := filterDeviceTrustProviderTypeSet[strings.ToLower(provider.DeviceTrustProviderType)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(provider.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneVerifiedAccessTrustProvider(provider))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VerifiedAccessTrustProvider(nil), items[start:end]...), outputToken, nil
}

func cloneVerifiedAccessInstance(in *VerifiedAccessInstance) VerifiedAccessInstance {
	if in == nil {
		return VerifiedAccessInstance{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	out.TrustProviderIDs = append([]string(nil), in.TrustProviderIDs...)
	return out
}

func cloneVerifiedAccessGroup(in *VerifiedAccessGroup) VerifiedAccessGroup {
	if in == nil {
		return VerifiedAccessGroup{}
	}
	out := *in
	if in.DeletionTime != nil {
		deletionTime := *in.DeletionTime
		out.DeletionTime = &deletionTime
	}
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneVerifiedAccessEndpoint(in *VerifiedAccessEndpoint) VerifiedAccessEndpoint {
	if in == nil {
		return VerifiedAccessEndpoint{}
	}
	out := *in
	if in.DeletionTime != nil {
		deletionTime := *in.DeletionTime
		out.DeletionTime = &deletionTime
	}
	out.SecurityGroupIDs = append([]string(nil), in.SecurityGroupIDs...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneVerifiedAccessTrustProvider(in *VerifiedAccessTrustProvider) VerifiedAccessTrustProvider {
	if in == nil {
		return VerifiedAccessTrustProvider{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}
