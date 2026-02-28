package ec2

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type VerifiedAccessPolicy struct {
	PolicyDocument string
	PolicyEnabled  bool
}

type VerifiedAccessLoggingOptions struct {
	CloudWatchLogGroup     string
	CloudWatchLogsEnabled  bool
	IncludeTrustContext    bool
	KinesisDeliveryStream  string
	KinesisFirehoseEnabled bool
	LogVersion             string
	S3BucketName           string
	S3BucketOwner          string
	S3Enabled              bool
	S3Prefix               string
}

type VerifiedAccessInstanceLoggingConfiguration struct {
	AccessLogs               VerifiedAccessLoggingOptions
	VerifiedAccessInstanceID string
}

type VerifiedAccessEndpointTarget struct {
	VerifiedAccessEndpointID              string
	VerifiedAccessEndpointTargetDNS       string
	VerifiedAccessEndpointTargetIPAddress string
}

type VerifiedAccessInstanceOpenVPNClientConfigurationRoute struct {
	CIDR string
}

type VerifiedAccessInstanceOpenVPNClientConfiguration struct {
	Config string
	Routes []VerifiedAccessInstanceOpenVPNClientConfigurationRoute
}

type VerifiedAccessInstanceUserTrustProviderClientConfiguration struct {
	AuthorizationEndpoint    string
	ClientID                 string
	ClientSecret             string
	Issuer                   string
	PKCEEnabled              bool
	PublicSigningKeyEndpoint string
	Scopes                   string
	TokenEndpoint            string
	Type                     string
	UserInfoEndpoint         string
}

type VerifiedAccessInstanceClientConfigurationExport struct {
	DeviceTrustProviders     []string
	OpenVPNConfigurations    []VerifiedAccessInstanceOpenVPNClientConfiguration
	Region                   string
	UserTrustProvider        *VerifiedAccessInstanceUserTrustProviderClientConfiguration
	VerifiedAccessInstanceID string
	Version                  string
}

func (s *Service) AttachVerifiedAccessTrustProvider(verifiedAccessInstanceID, verifiedAccessTrustProviderID string) (VerifiedAccessInstance, VerifiedAccessTrustProvider, error) {
	verifiedAccessInstanceID = strings.TrimSpace(verifiedAccessInstanceID)
	verifiedAccessTrustProviderID = strings.TrimSpace(verifiedAccessTrustProviderID)
	if verifiedAccessInstanceID == "" || verifiedAccessTrustProviderID == "" {
		return VerifiedAccessInstance{}, VerifiedAccessTrustProvider{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.verifiedAccessInstances[verifiedAccessInstanceID]
	if instance == nil {
		return VerifiedAccessInstance{}, VerifiedAccessTrustProvider{}, ErrNotFound
	}
	provider := s.verifiedAccessTrustProviders[verifiedAccessTrustProviderID]
	if provider == nil {
		return VerifiedAccessInstance{}, VerifiedAccessTrustProvider{}, ErrNotFound
	}
	found := false
	for _, id := range instance.TrustProviderIDs {
		if id == verifiedAccessTrustProviderID {
			found = true
			break
		}
	}
	if !found {
		instance.TrustProviderIDs = append(instance.TrustProviderIDs, verifiedAccessTrustProviderID)
	}
	instance.LastUpdatedTime = time.Now().UTC().Format(time.RFC3339)

	return cloneVerifiedAccessInstance(instance), cloneVerifiedAccessTrustProvider(provider), nil
}

func (s *Service) DetachVerifiedAccessTrustProvider(verifiedAccessInstanceID, verifiedAccessTrustProviderID string) (VerifiedAccessInstance, VerifiedAccessTrustProvider, error) {
	verifiedAccessInstanceID = strings.TrimSpace(verifiedAccessInstanceID)
	verifiedAccessTrustProviderID = strings.TrimSpace(verifiedAccessTrustProviderID)
	if verifiedAccessInstanceID == "" || verifiedAccessTrustProviderID == "" {
		return VerifiedAccessInstance{}, VerifiedAccessTrustProvider{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.verifiedAccessInstances[verifiedAccessInstanceID]
	if instance == nil {
		return VerifiedAccessInstance{}, VerifiedAccessTrustProvider{}, ErrNotFound
	}
	provider := s.verifiedAccessTrustProviders[verifiedAccessTrustProviderID]
	if provider == nil {
		return VerifiedAccessInstance{}, VerifiedAccessTrustProvider{}, ErrNotFound
	}
	index := -1
	for i, id := range instance.TrustProviderIDs {
		if id == verifiedAccessTrustProviderID {
			index = i
			break
		}
	}
	if index < 0 {
		return VerifiedAccessInstance{}, VerifiedAccessTrustProvider{}, ErrNotFound
	}
	instance.TrustProviderIDs = append(instance.TrustProviderIDs[:index], instance.TrustProviderIDs[index+1:]...)
	instance.LastUpdatedTime = time.Now().UTC().Format(time.RFC3339)

	return cloneVerifiedAccessInstance(instance), cloneVerifiedAccessTrustProvider(provider), nil
}

func (s *Service) DescribeVerifiedAccessInstanceLoggingConfigurations(
	verifiedAccessInstanceIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VerifiedAccessInstanceLoggingConfiguration, *string, error) {
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
	filterIncludeTrustSet := toLowerStringSet(standardFilters["access-logs.include-trust-context"])
	filterLogVersionSet := toLowerStringSet(standardFilters["access-logs.log-version"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VerifiedAccessInstanceLoggingConfiguration, 0, len(s.verifiedAccessInstances))
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
		if !matchesTagFilters(instance.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		cfg := s.loggingConfigForInstanceLocked(instance.ID)
		if len(filterIncludeTrustSet) > 0 {
			includeTrustKey := strings.ToLower(fmt.Sprintf("%t", cfg.AccessLogs.IncludeTrustContext))
			if _, ok := filterIncludeTrustSet[includeTrustKey]; !ok {
				continue
			}
		}
		if len(filterLogVersionSet) > 0 {
			if _, ok := filterLogVersionSet[strings.ToLower(cfg.AccessLogs.LogVersion)]; !ok {
				continue
			}
		}
		items = append(items, cfg)
	}

	sort.Slice(items, func(i, j int) bool { return items[i].VerifiedAccessInstanceID < items[j].VerifiedAccessInstanceID })
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VerifiedAccessInstanceLoggingConfiguration(nil), items[start:end]...), outputToken, nil
}

func (s *Service) ExportVerifiedAccessInstanceClientConfiguration(verifiedAccessInstanceID string) (VerifiedAccessInstanceClientConfigurationExport, error) {
	verifiedAccessInstanceID = strings.TrimSpace(verifiedAccessInstanceID)
	if verifiedAccessInstanceID == "" {
		return VerifiedAccessInstanceClientConfigurationExport{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.verifiedAccessInstances[verifiedAccessInstanceID]
	if instance == nil {
		return VerifiedAccessInstanceClientConfigurationExport{}, ErrNotFound
	}

	deviceProviderSet := map[string]struct{}{}
	var userProvider *VerifiedAccessInstanceUserTrustProviderClientConfiguration
	for _, providerID := range instance.TrustProviderIDs {
		provider := s.verifiedAccessTrustProviders[providerID]
		if provider == nil {
			continue
		}
		switch strings.ToLower(provider.TrustProviderType) {
		case "device":
			providerType := strings.TrimSpace(provider.DeviceTrustProviderType)
			if providerType == "" {
				providerType = "jamf"
			}
			deviceProviderSet[providerType] = struct{}{}
		case "user":
			if userProvider != nil {
				continue
			}
			reference := strings.TrimSpace(provider.PolicyReferenceName)
			if reference == "" {
				reference = "stackyard"
			}
			providerType := strings.TrimSpace(provider.UserTrustProviderType)
			if providerType == "" {
				providerType = "iam-identity-center"
			}
			userProvider = &VerifiedAccessInstanceUserTrustProviderClientConfiguration{
				AuthorizationEndpoint:    fmt.Sprintf("https://%s.auth.example.com/oauth2/authorize", reference),
				ClientID:                 "stackyard-client",
				ClientSecret:             "stackyard-secret",
				Issuer:                   fmt.Sprintf("https://%s.auth.example.com", reference),
				PKCEEnabled:              true,
				PublicSigningKeyEndpoint: fmt.Sprintf("https://%s.auth.example.com/.well-known/jwks.json", reference),
				Scopes:                   "openid email profile",
				TokenEndpoint:            fmt.Sprintf("https://%s.auth.example.com/oauth2/token", reference),
				Type:                     providerType,
				UserInfoEndpoint:         fmt.Sprintf("https://%s.auth.example.com/oauth2/userinfo", reference),
			}
		}
	}

	deviceProviders := make([]string, 0, len(deviceProviderSet))
	for providerType := range deviceProviderSet {
		deviceProviders = append(deviceProviders, providerType)
	}
	sort.Strings(deviceProviders)

	config := VerifiedAccessInstanceClientConfigurationExport{
		DeviceTrustProviders: deviceProviders,
		OpenVPNConfigurations: []VerifiedAccessInstanceOpenVPNClientConfiguration{
			{
				Config: "client\nproto tcp\nremote verified-access.local 443\n",
				Routes: []VerifiedAccessInstanceOpenVPNClientConfigurationRoute{
					{CIDR: "0.0.0.0/0"},
				},
			},
		},
		Region:                   DefaultRegion,
		UserTrustProvider:        userProvider,
		VerifiedAccessInstanceID: instance.ID,
		Version:                  "1",
	}

	return cloneVerifiedAccessInstanceClientConfigurationExport(config), nil
}

func (s *Service) GetVerifiedAccessEndpointPolicy(verifiedAccessEndpointID string) (VerifiedAccessPolicy, error) {
	verifiedAccessEndpointID = strings.TrimSpace(verifiedAccessEndpointID)
	if verifiedAccessEndpointID == "" {
		return VerifiedAccessPolicy{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.verifiedAccessEndpoints[verifiedAccessEndpointID] == nil {
		return VerifiedAccessPolicy{}, ErrNotFound
	}
	return cloneVerifiedAccessPolicy(s.endpointPolicyLocked(verifiedAccessEndpointID)), nil
}

func (s *Service) GetVerifiedAccessEndpointTargets(
	verifiedAccessEndpointID string,
	maxResults *int32,
	nextToken *string,
) ([]VerifiedAccessEndpointTarget, *string, error) {
	verifiedAccessEndpointID = strings.TrimSpace(verifiedAccessEndpointID)
	if verifiedAccessEndpointID == "" {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.verifiedAccessEndpoints[verifiedAccessEndpointID]
	if endpoint == nil {
		return nil, nil, ErrNotFound
	}

	targetDNS := strings.TrimSpace(endpoint.ApplicationDomain)
	if targetDNS == "" {
		targetDNS = endpoint.EndpointDomain
	}
	items := []VerifiedAccessEndpointTarget{
		{
			VerifiedAccessEndpointID:              endpoint.ID,
			VerifiedAccessEndpointTargetDNS:       targetDNS,
			VerifiedAccessEndpointTargetIPAddress: "10.0.0.10",
		},
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VerifiedAccessEndpointTarget(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetVerifiedAccessGroupPolicy(verifiedAccessGroupID string) (VerifiedAccessPolicy, error) {
	verifiedAccessGroupID = strings.TrimSpace(verifiedAccessGroupID)
	if verifiedAccessGroupID == "" {
		return VerifiedAccessPolicy{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	group := s.verifiedAccessGroups[verifiedAccessGroupID]
	if group == nil {
		return VerifiedAccessPolicy{}, ErrNotFound
	}
	return cloneVerifiedAccessPolicy(s.groupPolicyLocked(group.ID, group)), nil
}

func (s *Service) ModifyVerifiedAccessEndpoint(
	verifiedAccessEndpointID string,
	verifiedAccessGroupID *string,
	description *string,
) (VerifiedAccessEndpoint, error) {
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
	if verifiedAccessGroupID != nil {
		groupID := strings.TrimSpace(*verifiedAccessGroupID)
		if groupID == "" {
			return VerifiedAccessEndpoint{}, ErrInvalidParameter
		}
		group := s.verifiedAccessGroups[groupID]
		if group == nil {
			return VerifiedAccessEndpoint{}, ErrNotFound
		}
		endpoint.VerifiedGroup = groupID
		endpoint.VerifiedInstance = group.VerifiedInstance
	}
	if description != nil {
		endpoint.Description = strings.TrimSpace(*description)
	}
	endpoint.LastUpdatedTime = time.Now().UTC().Format(time.RFC3339)

	return cloneVerifiedAccessEndpoint(endpoint), nil
}

func (s *Service) ModifyVerifiedAccessEndpointPolicy(
	verifiedAccessEndpointID string,
	policyDocument *string,
	policyEnabled *bool,
) (VerifiedAccessPolicy, error) {
	verifiedAccessEndpointID = strings.TrimSpace(verifiedAccessEndpointID)
	if verifiedAccessEndpointID == "" {
		return VerifiedAccessPolicy{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.verifiedAccessEndpoints[verifiedAccessEndpointID]
	if endpoint == nil {
		return VerifiedAccessPolicy{}, ErrNotFound
	}

	policy := s.endpointPolicyLocked(verifiedAccessEndpointID)
	if policyDocument != nil {
		policy.PolicyDocument = strings.TrimSpace(*policyDocument)
	}
	if policyEnabled != nil {
		policy.PolicyEnabled = *policyEnabled
	}
	endpoint.LastUpdatedTime = time.Now().UTC().Format(time.RFC3339)

	return cloneVerifiedAccessPolicy(policy), nil
}

func (s *Service) ModifyVerifiedAccessGroup(
	verifiedAccessGroupID string,
	verifiedAccessInstanceID *string,
	description *string,
) (VerifiedAccessGroup, error) {
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
	now := time.Now().UTC().Format(time.RFC3339)

	if verifiedAccessInstanceID != nil {
		instanceID := strings.TrimSpace(*verifiedAccessInstanceID)
		if instanceID == "" {
			return VerifiedAccessGroup{}, ErrInvalidParameter
		}
		if s.verifiedAccessInstances[instanceID] == nil {
			return VerifiedAccessGroup{}, ErrNotFound
		}
		group.VerifiedInstance = instanceID
		for _, endpoint := range s.verifiedAccessEndpoints {
			if endpoint == nil || endpoint.VerifiedGroup != group.ID {
				continue
			}
			endpoint.VerifiedInstance = instanceID
			endpoint.LastUpdatedTime = now
		}
	}
	if description != nil {
		group.Description = strings.TrimSpace(*description)
	}
	group.LastUpdatedTime = now

	return cloneVerifiedAccessGroup(group), nil
}

func (s *Service) ModifyVerifiedAccessGroupPolicy(
	verifiedAccessGroupID string,
	policyDocument *string,
	policyEnabled *bool,
) (VerifiedAccessPolicy, error) {
	verifiedAccessGroupID = strings.TrimSpace(verifiedAccessGroupID)
	if verifiedAccessGroupID == "" {
		return VerifiedAccessPolicy{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	group := s.verifiedAccessGroups[verifiedAccessGroupID]
	if group == nil {
		return VerifiedAccessPolicy{}, ErrNotFound
	}

	policy := s.groupPolicyLocked(group.ID, group)
	if policyDocument != nil {
		policy.PolicyDocument = strings.TrimSpace(*policyDocument)
		group.PolicyDocument = policy.PolicyDocument
	}
	if policyEnabled != nil {
		policy.PolicyEnabled = *policyEnabled
	}
	group.LastUpdatedTime = time.Now().UTC().Format(time.RFC3339)

	return cloneVerifiedAccessPolicy(policy), nil
}

func (s *Service) ModifyVerifiedAccessInstance(
	verifiedAccessInstanceID string,
	cidrEndpointsCustomSubDomain *string,
	description *string,
) (VerifiedAccessInstance, error) {
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
	if cidrEndpointsCustomSubDomain != nil {
		instance.CidrEndpointsCustomSubDomain = strings.TrimSpace(*cidrEndpointsCustomSubDomain)
	}
	if description != nil {
		instance.Description = strings.TrimSpace(*description)
	}
	instance.LastUpdatedTime = time.Now().UTC().Format(time.RFC3339)

	return cloneVerifiedAccessInstance(instance), nil
}

func (s *Service) ModifyVerifiedAccessInstanceLoggingConfiguration(
	verifiedAccessInstanceID string,
	accessLogs VerifiedAccessLoggingOptions,
) (VerifiedAccessInstanceLoggingConfiguration, error) {
	verifiedAccessInstanceID = strings.TrimSpace(verifiedAccessInstanceID)
	if verifiedAccessInstanceID == "" {
		return VerifiedAccessInstanceLoggingConfiguration{}, ErrInvalidParameter
	}

	if strings.TrimSpace(accessLogs.LogVersion) == "" {
		accessLogs.LogVersion = "ocsf-1.0.0-rc.2"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.verifiedAccessInstances[verifiedAccessInstanceID]
	if instance == nil {
		return VerifiedAccessInstanceLoggingConfiguration{}, ErrNotFound
	}

	cfg := &VerifiedAccessInstanceLoggingConfiguration{
		AccessLogs:               cloneVerifiedAccessLoggingOptions(accessLogs),
		VerifiedAccessInstanceID: verifiedAccessInstanceID,
	}
	s.verifiedAccessInstanceLoggingConfigs[verifiedAccessInstanceID] = cfg
	instance.LastUpdatedTime = time.Now().UTC().Format(time.RFC3339)
	return cloneVerifiedAccessInstanceLoggingConfiguration(cfg), nil
}

func (s *Service) ModifyVerifiedAccessTrustProvider(
	verifiedAccessTrustProviderID string,
	description *string,
) (VerifiedAccessTrustProvider, error) {
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
	if description != nil {
		provider.Description = strings.TrimSpace(*description)
	}
	provider.LastUpdatedTime = time.Now().UTC().Format(time.RFC3339)

	return cloneVerifiedAccessTrustProvider(provider), nil
}

func cloneVerifiedAccessPolicy(in *VerifiedAccessPolicy) VerifiedAccessPolicy {
	if in == nil {
		return VerifiedAccessPolicy{}
	}
	return *in
}

func cloneVerifiedAccessLoggingOptions(in VerifiedAccessLoggingOptions) VerifiedAccessLoggingOptions {
	return in
}

func cloneVerifiedAccessInstanceLoggingConfiguration(in *VerifiedAccessInstanceLoggingConfiguration) VerifiedAccessInstanceLoggingConfiguration {
	if in == nil {
		return VerifiedAccessInstanceLoggingConfiguration{}
	}
	out := *in
	out.AccessLogs = cloneVerifiedAccessLoggingOptions(in.AccessLogs)
	return out
}

func cloneVerifiedAccessInstanceOpenVPNClientConfiguration(in VerifiedAccessInstanceOpenVPNClientConfiguration) VerifiedAccessInstanceOpenVPNClientConfiguration {
	out := in
	out.Routes = append([]VerifiedAccessInstanceOpenVPNClientConfigurationRoute(nil), in.Routes...)
	return out
}

func cloneVerifiedAccessInstanceClientConfigurationExport(in VerifiedAccessInstanceClientConfigurationExport) VerifiedAccessInstanceClientConfigurationExport {
	out := in
	out.DeviceTrustProviders = append([]string(nil), in.DeviceTrustProviders...)
	out.OpenVPNConfigurations = make([]VerifiedAccessInstanceOpenVPNClientConfiguration, 0, len(in.OpenVPNConfigurations))
	for _, cfg := range in.OpenVPNConfigurations {
		out.OpenVPNConfigurations = append(out.OpenVPNConfigurations, cloneVerifiedAccessInstanceOpenVPNClientConfiguration(cfg))
	}
	if in.UserTrustProvider != nil {
		userProvider := *in.UserTrustProvider
		out.UserTrustProvider = &userProvider
	}
	return out
}

func (s *Service) endpointPolicyLocked(verifiedAccessEndpointID string) *VerifiedAccessPolicy {
	if policy := s.verifiedAccessEndpointPolicies[verifiedAccessEndpointID]; policy != nil {
		return policy
	}
	policy := &VerifiedAccessPolicy{PolicyEnabled: false}
	s.verifiedAccessEndpointPolicies[verifiedAccessEndpointID] = policy
	return policy
}

func (s *Service) groupPolicyLocked(verifiedAccessGroupID string, group *VerifiedAccessGroup) *VerifiedAccessPolicy {
	if policy := s.verifiedAccessGroupPolicies[verifiedAccessGroupID]; policy != nil {
		return policy
	}
	initialDoc := ""
	initialEnabled := false
	if group != nil {
		initialDoc = strings.TrimSpace(group.PolicyDocument)
		initialEnabled = initialDoc != ""
	}
	policy := &VerifiedAccessPolicy{
		PolicyDocument: initialDoc,
		PolicyEnabled:  initialEnabled,
	}
	s.verifiedAccessGroupPolicies[verifiedAccessGroupID] = policy
	return policy
}

func (s *Service) loggingConfigForInstanceLocked(verifiedAccessInstanceID string) VerifiedAccessInstanceLoggingConfiguration {
	if cfg := s.verifiedAccessInstanceLoggingConfigs[verifiedAccessInstanceID]; cfg != nil {
		return cloneVerifiedAccessInstanceLoggingConfiguration(cfg)
	}
	return VerifiedAccessInstanceLoggingConfiguration{
		VerifiedAccessInstanceID: verifiedAccessInstanceID,
		AccessLogs: VerifiedAccessLoggingOptions{
			LogVersion: "ocsf-1.0.0-rc.2",
		},
	}
}
