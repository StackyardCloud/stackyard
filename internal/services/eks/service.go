package eks

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidParameter         = errors.New("invalid parameter")
	ErrClusterNotFound          = errors.New("cluster not found")
	ErrClusterAlreadyExists     = errors.New("cluster already exists")
	ErrNodegroupNotFound        = errors.New("nodegroup not found")
	ErrNodegroupAlreadyExists   = errors.New("nodegroup already exists")
	ErrFargateProfileNotFound   = errors.New("fargate profile not found")
	ErrFargateProfileExists     = errors.New("fargate profile already exists")
	ErrUpdateNotFound           = errors.New("update not found")
	ErrAddonNotFound            = errors.New("addon not found")
	ErrAddonAlreadyExists       = errors.New("addon already exists")
	ErrCapabilityNotFound       = errors.New("capability not found")
	ErrCapabilityAlreadyExists  = errors.New("capability already exists")
	ErrIdentityProviderNotFound = errors.New("identity provider config not found")
	ErrIdentityProviderExists   = errors.New("identity provider config already exists")
	ErrAccessEntryNotFound      = errors.New("access entry not found")
	ErrAccessEntryAlreadyExists = errors.New("access entry already exists")
	ErrAccessPolicyNotFound     = errors.New("access policy not found")
	ErrAssociatedPolicyNotFound = errors.New("associated access policy not found")
	ErrInsightNotFound          = errors.New("insight not found")
	ErrPodIdentityNotFound      = errors.New("pod identity association not found")
	ErrPodIdentityExists        = errors.New("pod identity association already exists")
	ErrSubscriptionNotFound     = errors.New("eks anywhere subscription not found")
	ErrSubscriptionExists       = errors.New("eks anywhere subscription already exists")
	ErrTagNotFound              = errors.New("resource tags not found")
)

const (
	DefaultRegion         = "us-east-1"
	DefaultAccountID      = "123456789012"
	DefaultClusterVersion = "1.29"
)

type ResourcesVpcConfig struct {
	SubnetIDs            []string `json:"subnetIds,omitempty"`
	EndpointPublicAccess bool     `json:"endpointPublicAccess"`
}

type ResourcesVpcConfigInput struct {
	SubnetIDs            []string
	EndpointPublicAccess *bool
}

type Cluster struct {
	Name               string             `json:"name"`
	Arn                string             `json:"arn"`
	CreatedAt          time.Time          `json:"createdAt"`
	Version            string             `json:"version,omitempty"`
	RoleArn            string             `json:"roleArn,omitempty"`
	Status             string             `json:"status"`
	ConnectorConfig    *ConnectorConfig   `json:"connectorConfig,omitempty"`
	ResourcesVpcConfig ResourcesVpcConfig `json:"resourcesVpcConfig"`
	Tags               map[string]string  `json:"tags,omitempty"`
}

type CreateClusterInput struct {
	Name               string
	Version            string
	RoleArn            string
	ResourcesVpcConfig *ResourcesVpcConfigInput
	Tags               map[string]string
}

type UpdateClusterConfigInput struct {
	ResourcesVpcConfig *ResourcesVpcConfigInput
}

type UpdateClusterVersionInput struct {
	Version string
}

type Nodegroup struct {
	NodegroupName  string            `json:"nodegroupName"`
	NodegroupArn   string            `json:"nodegroupArn"`
	ClusterName    string            `json:"clusterName"`
	Version        string            `json:"version,omitempty"`
	ReleaseVersion string            `json:"releaseVersion,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	ModifiedAt     time.Time         `json:"modifiedAt"`
	Status         string            `json:"status"`
	NodeRole       string            `json:"nodeRole,omitempty"`
	Subnets        []string          `json:"subnets,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

type CreateNodegroupInput struct {
	NodegroupName string
	NodeRole      string
	Subnets       []string
	Version       string
	Tags          map[string]string
}

type UpdateNodegroupConfigInput struct {
	Labels map[string]string
}

type UpdateNodegroupVersionInput struct {
	Version string
}

type FargateProfileSelector struct {
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type FargateProfile struct {
	FargateProfileName  string                   `json:"fargateProfileName"`
	FargateProfileArn   string                   `json:"fargateProfileArn"`
	ClusterName         string                   `json:"clusterName"`
	CreatedAt           time.Time                `json:"createdAt"`
	Status              string                   `json:"status"`
	PodExecutionRoleArn string                   `json:"podExecutionRoleArn,omitempty"`
	Subnets             []string                 `json:"subnets,omitempty"`
	Selectors           []FargateProfileSelector `json:"selectors,omitempty"`
	Tags                map[string]string        `json:"tags,omitempty"`
}

type Addon struct {
	AddonName             string            `json:"addonName"`
	AddonArn              string            `json:"addonArn"`
	ClusterName           string            `json:"clusterName"`
	Status                string            `json:"status"`
	AddonVersion          string            `json:"addonVersion,omitempty"`
	ServiceAccountRoleArn string            `json:"serviceAccountRoleArn,omitempty"`
	ConfigurationValues   string            `json:"configurationValues,omitempty"`
	CreatedAt             time.Time         `json:"createdAt"`
	ModifiedAt            time.Time         `json:"modifiedAt"`
	Tags                  map[string]string `json:"tags,omitempty"`
}

type CreateAddonInput struct {
	AddonName             string
	AddonVersion          string
	ServiceAccountRoleArn string
	ConfigurationValues   string
	Tags                  map[string]string
}

type UpdateAddonInput struct {
	AddonVersion          string
	ServiceAccountRoleArn string
	ConfigurationValues   string
}

type AddonVersionInfo struct {
	AddonName    string   `json:"addonName"`
	AddonVersion string   `json:"addonVersion"`
	Architecture []string `json:"architecture"`
}

type OIDCIdentityProviderConfig struct {
	IdentityProviderConfigName string            `json:"identityProviderConfigName"`
	IssuerURL                  string            `json:"issuerUrl,omitempty"`
	ClientID                   string            `json:"clientId,omitempty"`
	UsernameClaim              string            `json:"usernameClaim,omitempty"`
	UsernamePrefix             string            `json:"usernamePrefix,omitempty"`
	GroupsClaim                string            `json:"groupsClaim,omitempty"`
	GroupsPrefix               string            `json:"groupsPrefix,omitempty"`
	RequiredClaims             map[string]string `json:"requiredClaims,omitempty"`
	Tags                       map[string]string `json:"tags,omitempty"`
	Status                     string            `json:"status"`
}

type AssociateIdentityProviderConfigInput struct {
	OIDC *OIDCIdentityProviderConfig
	Tags map[string]string
}

type IdentityProviderConfigRef struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type AccessEntry struct {
	ClusterName      string            `json:"clusterName"`
	PrincipalArn     string            `json:"principalArn"`
	AccessEntryArn   string            `json:"accessEntryArn"`
	Type             string            `json:"type"`
	Username         string            `json:"username,omitempty"`
	KubernetesGroups []string          `json:"kubernetesGroups,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
	CreatedAt        time.Time         `json:"createdAt"`
	ModifiedAt       time.Time         `json:"modifiedAt"`
}

type CreateAccessEntryInput struct {
	PrincipalArn     string
	Type             string
	Username         string
	KubernetesGroups []string
	Tags             map[string]string
}

type UpdateAccessEntryInput struct {
	Username         *string
	KubernetesGroups []string
}

type AccessScope struct {
	Type       string   `json:"type"`
	Namespaces []string `json:"namespaces,omitempty"`
}

type AssociatedAccessPolicy struct {
	PolicyArn    string      `json:"policyArn"`
	AccessScope  AccessScope `json:"accessScope"`
	AssociatedAt time.Time   `json:"associatedAt"`
	ModifiedAt   time.Time   `json:"modifiedAt"`
}

type AssociateAccessPolicyInput struct {
	PolicyArn   string
	AccessScope AccessScope
}

type AccessPolicy struct {
	Name string `json:"name"`
	Arn  string `json:"arn"`
}

type ConnectorConfig struct {
	ActivationID     string    `json:"activationId"`
	ActivationCode   string    `json:"activationCode"`
	ActivationExpiry time.Time `json:"activationExpiry"`
	Provider         string    `json:"provider"`
	RoleArn          string    `json:"roleArn"`
}

type RegisterClusterInput struct {
	Name           string
	ConnectorRole  string
	ConnectorCloud string
	Tags           map[string]string
}

type Capability struct {
	ClusterName    string            `json:"clusterName"`
	CapabilityName string            `json:"capabilityName"`
	CapabilityArn  string            `json:"capabilityArn"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"createdAt"`
	ModifiedAt     time.Time         `json:"modifiedAt"`
	Tags           map[string]string `json:"tags,omitempty"`
}

type CreateCapabilityInput struct {
	CapabilityName string
	Tags           map[string]string
}

type UpdateCapabilityInput struct {
	Tags map[string]string
}

type InsightStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type Insight struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Category           string            `json:"category"`
	KubernetesVersion  string            `json:"kubernetesVersion,omitempty"`
	LastRefreshTime    time.Time         `json:"lastRefreshTime"`
	LastTransitionTime time.Time         `json:"lastTransitionTime"`
	Description        string            `json:"description,omitempty"`
	InsightStatus      InsightStatus     `json:"insightStatus"`
	Recommendation     string            `json:"recommendation,omitempty"`
	AdditionalInfo     map[string]string `json:"additionalInfo,omitempty"`
}

type InsightSummary struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Category           string        `json:"category"`
	KubernetesVersion  string        `json:"kubernetesVersion,omitempty"`
	LastRefreshTime    time.Time     `json:"lastRefreshTime"`
	LastTransitionTime time.Time     `json:"lastTransitionTime"`
	Description        string        `json:"description,omitempty"`
	InsightStatus      InsightStatus `json:"insightStatus"`
}

type ListInsightsInput struct {
	Categories         []string
	KubernetesVersions []string
	Statuses           []string
}

type InsightsRefresh struct {
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
}

type PodIdentityAssociation struct {
	ClusterName    string            `json:"clusterName"`
	Namespace      string            `json:"namespace"`
	ServiceAccount string            `json:"serviceAccount"`
	RoleArn        string            `json:"roleArn"`
	AssociationArn string            `json:"associationArn"`
	AssociationID  string            `json:"associationId"`
	Tags           map[string]string `json:"tags,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	ModifiedAt     time.Time         `json:"modifiedAt"`
	OwnerArn       string            `json:"ownerArn,omitempty"`
}

type PodIdentityAssociationSummary struct {
	ClusterName    string `json:"clusterName"`
	Namespace      string `json:"namespace"`
	ServiceAccount string `json:"serviceAccount"`
	AssociationArn string `json:"associationArn"`
	AssociationID  string `json:"associationId"`
	OwnerArn       string `json:"ownerArn,omitempty"`
}

type CreatePodIdentityAssociationInput struct {
	Namespace      string
	ServiceAccount string
	RoleArn        string
	Tags           map[string]string
}

type UpdatePodIdentityAssociationInput struct {
	RoleArn string
}

type EksAnywhereSubscriptionTerm struct {
	Duration int    `json:"duration"`
	Unit     string `json:"unit"`
}

type EksAnywhereSubscription struct {
	Name            string                      `json:"name,omitempty"`
	ID              string                      `json:"id"`
	Arn             string                      `json:"arn"`
	CreatedAt       time.Time                   `json:"createdAt"`
	EffectiveDate   time.Time                   `json:"effectiveDate"`
	ExpirationDate  time.Time                   `json:"expirationDate"`
	LicenseQuantity int                         `json:"licenseQuantity"`
	LicenseType     string                      `json:"licenseType"`
	Term            EksAnywhereSubscriptionTerm `json:"term"`
	Status          string                      `json:"status"`
	AutoRenew       bool                        `json:"autoRenew"`
	LicenseArns     []string                    `json:"licenseArns,omitempty"`
	Tags            map[string]string           `json:"tags,omitempty"`
}

type CreateEksAnywhereSubscriptionInput struct {
	Name            string
	Term            EksAnywhereSubscriptionTerm
	LicenseQuantity int
	LicenseType     string
	AutoRenew       bool
	Tags            map[string]string
}

type UpdateEksAnywhereSubscriptionInput struct {
	AutoRenew *bool
	Tags      map[string]string
}

type EncryptionProvider struct {
	KeyArn string `json:"keyArn,omitempty"`
}

type EncryptionConfigEntry struct {
	Resources []string           `json:"resources,omitempty"`
	Provider  EncryptionProvider `json:"provider,omitempty"`
}

var defaultAccessPolicies = []AccessPolicy{
	{Name: "AmazonEKSClusterAdminPolicy", Arn: "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"},
	{Name: "AmazonEKSAdminPolicy", Arn: "arn:aws:eks::aws:cluster-access-policy/AmazonEKSAdminPolicy"},
	{Name: "AmazonEKSEditPolicy", Arn: "arn:aws:eks::aws:cluster-access-policy/AmazonEKSEditPolicy"},
	{Name: "AmazonEKSViewPolicy", Arn: "arn:aws:eks::aws:cluster-access-policy/AmazonEKSViewPolicy"},
}

type CreateFargateProfileInput struct {
	FargateProfileName  string
	PodExecutionRoleArn string
	Subnets             []string
	Selectors           []FargateProfileSelector
	Tags                map[string]string
}

type UpdateParam struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type UpdateError struct {
	ErrorCode    string   `json:"errorCode,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	ResourceIDs  []string `json:"resourceIds,omitempty"`
}

type Update struct {
	ID            string        `json:"id"`
	Status        string        `json:"status"`
	Type          string        `json:"type"`
	Params        []UpdateParam `json:"params,omitempty"`
	Errors        []UpdateError `json:"errors,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
	NodegroupName string        `json:"-"`
	AddonName     string        `json:"-"`
}

type Service struct {
	mu                 sync.RWMutex
	clusters           map[string]*Cluster
	nodegroups         map[string]map[string]*Nodegroup
	fargateProfiles    map[string]map[string]*FargateProfile
	addons             map[string]map[string]*Addon
	capabilities       map[string]map[string]*Capability
	identityProviders  map[string]map[string]*OIDCIdentityProviderConfig
	accessEntries      map[string]map[string]*AccessEntry
	associatedPolicies map[string]map[string]map[string]*AssociatedAccessPolicy
	insights           map[string]map[string]*Insight
	insightsRefresh    map[string]*InsightsRefresh
	podIdentity        map[string]map[string]*PodIdentityAssociation
	subscriptions      map[string]*EksAnywhereSubscription
	encryptionConfig   map[string][]EncryptionConfigEntry
	tagsByARN          map[string]map[string]string
	updates            map[string]map[string]*Update
	updateCounter      uint64
}

func NewService() *Service {
	return &Service{
		clusters:           map[string]*Cluster{},
		nodegroups:         map[string]map[string]*Nodegroup{},
		fargateProfiles:    map[string]map[string]*FargateProfile{},
		addons:             map[string]map[string]*Addon{},
		capabilities:       map[string]map[string]*Capability{},
		identityProviders:  map[string]map[string]*OIDCIdentityProviderConfig{},
		accessEntries:      map[string]map[string]*AccessEntry{},
		associatedPolicies: map[string]map[string]map[string]*AssociatedAccessPolicy{},
		insights:           map[string]map[string]*Insight{},
		insightsRefresh:    map[string]*InsightsRefresh{},
		podIdentity:        map[string]map[string]*PodIdentityAssociation{},
		subscriptions:      map[string]*EksAnywhereSubscription{},
		encryptionConfig:   map[string][]EncryptionConfigEntry{},
		tagsByARN:          map[string]map[string]string{},
		updates:            map[string]map[string]*Update{},
	}
}

func (s *Service) CreateCluster(in CreateClusterInput) (*Cluster, error) {
	if in.Name == "" || in.RoleArn == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clusters[in.Name]; exists {
		return nil, ErrClusterAlreadyExists
	}

	version := in.Version
	if version == "" {
		version = DefaultClusterVersion
	}

	vpc := ResourcesVpcConfig{EndpointPublicAccess: true}
	if in.ResourcesVpcConfig != nil {
		vpc.SubnetIDs = cloneStrings(in.ResourcesVpcConfig.SubnetIDs)
		if in.ResourcesVpcConfig.EndpointPublicAccess != nil {
			vpc.EndpointPublicAccess = *in.ResourcesVpcConfig.EndpointPublicAccess
		}
	}

	cluster := &Cluster{
		Name:               in.Name,
		Arn:                clusterARN(in.Name),
		CreatedAt:          time.Now().UTC(),
		Version:            version,
		RoleArn:            in.RoleArn,
		Status:             "ACTIVE",
		ResourcesVpcConfig: vpc,
		Tags:               cloneStringMap(in.Tags),
	}
	s.clusters[in.Name] = cluster
	s.nodegroups[in.Name] = map[string]*Nodegroup{}
	s.fargateProfiles[in.Name] = map[string]*FargateProfile{}
	s.addons[in.Name] = map[string]*Addon{}
	s.capabilities[in.Name] = map[string]*Capability{}
	s.identityProviders[in.Name] = map[string]*OIDCIdentityProviderConfig{}
	s.accessEntries[in.Name] = map[string]*AccessEntry{}
	s.associatedPolicies[in.Name] = map[string]map[string]*AssociatedAccessPolicy{}
	s.insights[in.Name] = defaultInsightsForCluster(cluster.Name, cluster.Version)
	s.insightsRefresh[in.Name] = &InsightsRefresh{
		Status:      "COMPLETED",
		StartedAt:   cluster.CreatedAt,
		CompletedAt: cluster.CreatedAt,
	}
	s.podIdentity[in.Name] = map[string]*PodIdentityAssociation{}
	s.encryptionConfig[in.Name] = nil
	s.updates[in.Name] = map[string]*Update{}
	s.tagsByARN[cluster.Arn] = cloneStringMap(cluster.Tags)
	return cloneCluster(cluster), nil
}

func (s *Service) DescribeCluster(name string) (*Cluster, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cluster, ok := s.clusters[name]
	if !ok {
		return nil, ErrClusterNotFound
	}
	return cloneCluster(cluster), nil
}

func (s *Service) ListClusters() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]string, 0, len(s.clusters))
	for name := range s.clusters {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (s *Service) DeleteCluster(name string) (*Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[name]
	if !ok {
		return nil, ErrClusterNotFound
	}
	out := cloneCluster(cluster)
	out.Status = "DELETING"
	delete(s.clusters, name)
	delete(s.nodegroups, name)
	delete(s.fargateProfiles, name)
	delete(s.addons, name)
	delete(s.capabilities, name)
	delete(s.identityProviders, name)
	delete(s.accessEntries, name)
	delete(s.associatedPolicies, name)
	delete(s.insights, name)
	delete(s.insightsRefresh, name)
	delete(s.podIdentity, name)
	delete(s.encryptionConfig, name)
	delete(s.updates, name)
	delete(s.tagsByARN, cluster.Arn)
	return out, nil
}

func (s *Service) UpdateClusterConfig(name string, in UpdateClusterConfigInput) (*Update, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[name]
	if !ok {
		return nil, ErrClusterNotFound
	}

	params := make([]UpdateParam, 0, 2)
	if in.ResourcesVpcConfig != nil {
		if len(in.ResourcesVpcConfig.SubnetIDs) > 0 {
			cluster.ResourcesVpcConfig.SubnetIDs = cloneStrings(in.ResourcesVpcConfig.SubnetIDs)
			params = append(params, UpdateParam{Type: "SubnetIds", Value: fmt.Sprintf("%d", len(in.ResourcesVpcConfig.SubnetIDs))})
		}
		if in.ResourcesVpcConfig.EndpointPublicAccess != nil {
			cluster.ResourcesVpcConfig.EndpointPublicAccess = *in.ResourcesVpcConfig.EndpointPublicAccess
			params = append(params, UpdateParam{Type: "EndpointPublicAccess", Value: fmt.Sprintf("%t", *in.ResourcesVpcConfig.EndpointPublicAccess)})
		}
	}

	update := s.createUpdateLocked(name, "ConfigUpdate", params, "", "")
	return cloneUpdate(update), nil
}

func (s *Service) UpdateClusterVersion(name string, in UpdateClusterVersionInput) (*Update, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[name]
	if !ok {
		return nil, ErrClusterNotFound
	}
	version := in.Version
	if version == "" {
		version = cluster.Version
	}
	cluster.Version = version

	update := s.createUpdateLocked(name, "VersionUpdate", []UpdateParam{{Type: "Version", Value: version}}, "", "")
	return cloneUpdate(update), nil
}

func (s *Service) CreateNodegroup(clusterName string, in CreateNodegroupInput) (*Nodegroup, error) {
	if in.NodegroupName == "" || in.NodeRole == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, ErrClusterNotFound
	}
	clusterNodegroups := s.nodegroups[clusterName]
	if _, exists := clusterNodegroups[in.NodegroupName]; exists {
		return nil, ErrNodegroupAlreadyExists
	}

	version := in.Version
	if version == "" {
		version = cluster.Version
	}
	now := time.Now().UTC()
	nodegroup := &Nodegroup{
		NodegroupName: in.NodegroupName,
		NodegroupArn:  nodegroupARN(clusterName, in.NodegroupName),
		ClusterName:   clusterName,
		Version:       version,
		CreatedAt:     now,
		ModifiedAt:    now,
		Status:        "ACTIVE",
		NodeRole:      in.NodeRole,
		Subnets:       cloneStrings(in.Subnets),
		Tags:          cloneStringMap(in.Tags),
	}
	clusterNodegroups[in.NodegroupName] = nodegroup
	s.tagsByARN[nodegroup.NodegroupArn] = cloneStringMap(nodegroup.Tags)
	return cloneNodegroup(nodegroup), nil
}

func (s *Service) DescribeNodegroup(clusterName, nodegroupName string) (*Nodegroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterNodegroups, ok := s.nodegroups[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrNodegroupNotFound
	}
	nodegroup, ok := clusterNodegroups[nodegroupName]
	if !ok {
		return nil, ErrNodegroupNotFound
	}
	return cloneNodegroup(nodegroup), nil
}

func (s *Service) ListNodegroups(clusterName string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterNodegroups, ok := s.nodegroups[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return []string{}, nil
	}
	out := make([]string, 0, len(clusterNodegroups))
	for name := range clusterNodegroups {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Service) DeleteNodegroup(clusterName, nodegroupName string) (*Nodegroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterNodegroups, ok := s.nodegroups[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrNodegroupNotFound
	}
	nodegroup, ok := clusterNodegroups[nodegroupName]
	if !ok {
		return nil, ErrNodegroupNotFound
	}
	out := cloneNodegroup(nodegroup)
	out.Status = "DELETING"
	delete(clusterNodegroups, nodegroupName)
	delete(s.tagsByARN, nodegroup.NodegroupArn)
	return out, nil
}

func (s *Service) UpdateNodegroupConfig(clusterName, nodegroupName string, _ UpdateNodegroupConfigInput) (*Update, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterNodegroups, ok := s.nodegroups[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrNodegroupNotFound
	}
	if _, ok := clusterNodegroups[nodegroupName]; !ok {
		return nil, ErrNodegroupNotFound
	}

	update := s.createUpdateLocked(clusterName, "ConfigUpdate", nil, nodegroupName, "")
	return cloneUpdate(update), nil
}

func (s *Service) UpdateNodegroupVersion(clusterName, nodegroupName string, in UpdateNodegroupVersionInput) (*Update, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterNodegroups, ok := s.nodegroups[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrNodegroupNotFound
	}
	nodegroup, ok := clusterNodegroups[nodegroupName]
	if !ok {
		return nil, ErrNodegroupNotFound
	}
	version := in.Version
	if version == "" {
		version = nodegroup.Version
	}
	nodegroup.Version = version
	nodegroup.ModifiedAt = time.Now().UTC()

	update := s.createUpdateLocked(clusterName, "VersionUpdate", []UpdateParam{{Type: "Version", Value: version}}, nodegroupName, "")
	return cloneUpdate(update), nil
}

func (s *Service) CreateFargateProfile(clusterName string, in CreateFargateProfileInput) (*FargateProfile, error) {
	if in.FargateProfileName == "" || in.PodExecutionRoleArn == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	clusterProfiles, ok := s.fargateProfiles[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		clusterProfiles = map[string]*FargateProfile{}
		s.fargateProfiles[clusterName] = clusterProfiles
	}
	if _, exists := clusterProfiles[in.FargateProfileName]; exists {
		return nil, ErrFargateProfileExists
	}

	profile := &FargateProfile{
		FargateProfileName:  in.FargateProfileName,
		FargateProfileArn:   fargateProfileARN(clusterName, in.FargateProfileName),
		ClusterName:         clusterName,
		CreatedAt:           time.Now().UTC(),
		Status:              "ACTIVE",
		PodExecutionRoleArn: in.PodExecutionRoleArn,
		Subnets:             cloneStrings(in.Subnets),
		Selectors:           cloneSelectors(in.Selectors),
		Tags:                cloneStringMap(in.Tags),
	}
	clusterProfiles[in.FargateProfileName] = profile
	s.tagsByARN[profile.FargateProfileArn] = cloneStringMap(profile.Tags)
	return cloneFargateProfile(profile), nil
}

func (s *Service) DescribeFargateProfile(clusterName, profileName string) (*FargateProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterProfiles, ok := s.fargateProfiles[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrFargateProfileNotFound
	}
	profile, ok := clusterProfiles[profileName]
	if !ok {
		return nil, ErrFargateProfileNotFound
	}
	return cloneFargateProfile(profile), nil
}

func (s *Service) ListFargateProfiles(clusterName string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterProfiles, ok := s.fargateProfiles[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return []string{}, nil
	}
	out := make([]string, 0, len(clusterProfiles))
	for name := range clusterProfiles {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Service) DeleteFargateProfile(clusterName, profileName string) (*FargateProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterProfiles, ok := s.fargateProfiles[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrFargateProfileNotFound
	}
	profile, ok := clusterProfiles[profileName]
	if !ok {
		return nil, ErrFargateProfileNotFound
	}
	out := cloneFargateProfile(profile)
	out.Status = "DELETING"
	delete(clusterProfiles, profileName)
	delete(s.tagsByARN, profile.FargateProfileArn)
	return out, nil
}

func (s *Service) ListUpdates(clusterName, nodegroupName, addonName string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterUpdates, ok := s.updates[clusterName]
	if !ok {
		return []string{}, nil
	}
	out := make([]string, 0, len(clusterUpdates))
	for id, update := range clusterUpdates {
		if nodegroupName != "" && update.NodegroupName != nodegroupName {
			continue
		}
		if addonName != "" && update.AddonName != addonName {
			continue
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Service) DescribeUpdate(clusterName, updateID, nodegroupName, addonName string) (*Update, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterUpdates, ok := s.updates[clusterName]
	if !ok {
		return nil, ErrUpdateNotFound
	}
	update, ok := clusterUpdates[updateID]
	if !ok {
		return nil, ErrUpdateNotFound
	}
	if nodegroupName != "" && update.NodegroupName != nodegroupName {
		return nil, ErrUpdateNotFound
	}
	if addonName != "" && update.AddonName != addonName {
		return nil, ErrUpdateNotFound
	}
	return cloneUpdate(update), nil
}

func (s *Service) CreateAddon(clusterName string, in CreateAddonInput) (*Addon, error) {
	if strings.TrimSpace(in.AddonName) == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterAddons, ok := s.addons[clusterName]
	if !ok {
		clusterAddons = map[string]*Addon{}
		s.addons[clusterName] = clusterAddons
	}
	if _, exists := clusterAddons[in.AddonName]; exists {
		return nil, ErrAddonAlreadyExists
	}

	addonVersion := strings.TrimSpace(in.AddonVersion)
	if addonVersion == "" {
		addonVersion = "latest"
	}
	now := time.Now().UTC()
	addon := &Addon{
		AddonName:             in.AddonName,
		AddonArn:              addonARN(clusterName, in.AddonName),
		ClusterName:           clusterName,
		Status:                "ACTIVE",
		AddonVersion:          addonVersion,
		ServiceAccountRoleArn: strings.TrimSpace(in.ServiceAccountRoleArn),
		ConfigurationValues:   in.ConfigurationValues,
		CreatedAt:             now,
		ModifiedAt:            now,
		Tags:                  cloneStringMap(in.Tags),
	}
	clusterAddons[in.AddonName] = addon
	s.tagsByARN[addon.AddonArn] = cloneStringMap(addon.Tags)
	return cloneAddon(addon), nil
}

func (s *Service) DescribeAddon(clusterName, addonName string) (*Addon, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterAddons, ok := s.addons[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrAddonNotFound
	}
	addon, ok := clusterAddons[addonName]
	if !ok {
		return nil, ErrAddonNotFound
	}
	return cloneAddon(addon), nil
}

func (s *Service) ListAddons(clusterName string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterAddons, ok := s.addons[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return []string{}, nil
	}
	out := make([]string, 0, len(clusterAddons))
	for name := range clusterAddons {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Service) UpdateAddon(clusterName, addonName string, in UpdateAddonInput) (*Update, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterAddons, ok := s.addons[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrAddonNotFound
	}
	addon, ok := clusterAddons[addonName]
	if !ok {
		return nil, ErrAddonNotFound
	}

	params := []UpdateParam{}
	if strings.TrimSpace(in.AddonVersion) != "" {
		addon.AddonVersion = strings.TrimSpace(in.AddonVersion)
		params = append(params, UpdateParam{Type: "AddonVersion", Value: addon.AddonVersion})
	}
	if strings.TrimSpace(in.ServiceAccountRoleArn) != "" {
		addon.ServiceAccountRoleArn = strings.TrimSpace(in.ServiceAccountRoleArn)
		params = append(params, UpdateParam{Type: "ServiceAccountRoleArn", Value: addon.ServiceAccountRoleArn})
	}
	if strings.TrimSpace(in.ConfigurationValues) != "" {
		addon.ConfigurationValues = in.ConfigurationValues
		params = append(params, UpdateParam{Type: "ConfigurationValues", Value: "updated"})
	}
	addon.ModifiedAt = time.Now().UTC()

	update := s.createUpdateLocked(clusterName, "AddonUpdate", params, "", addonName)
	return cloneUpdate(update), nil
}

func (s *Service) DeleteAddon(clusterName, addonName string) (*Addon, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterAddons, ok := s.addons[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrAddonNotFound
	}
	addon, ok := clusterAddons[addonName]
	if !ok {
		return nil, ErrAddonNotFound
	}
	out := cloneAddon(addon)
	out.Status = "DELETING"
	delete(clusterAddons, addonName)
	delete(s.tagsByARN, addon.AddonArn)
	return out, nil
}

func (s *Service) DescribeAddonVersions(addonName string) []AddonVersionInfo {
	if strings.TrimSpace(addonName) != "" {
		return []AddonVersionInfo{
			{
				AddonName:    addonName,
				AddonVersion: "latest",
				Architecture: []string{"amd64", "arm64"},
			},
		}
	}

	return []AddonVersionInfo{
		{AddonName: "vpc-cni", AddonVersion: "latest", Architecture: []string{"amd64", "arm64"}},
		{AddonName: "coredns", AddonVersion: "latest", Architecture: []string{"amd64", "arm64"}},
		{AddonName: "kube-proxy", AddonVersion: "latest", Architecture: []string{"amd64", "arm64"}},
	}
}

func (s *Service) DescribeAddonConfiguration(addonName, addonVersion string) (string, string) {
	name := strings.TrimSpace(addonName)
	if name == "" {
		name = "vpc-cni"
	}
	version := strings.TrimSpace(addonVersion)
	if version == "" {
		version = "latest"
	}
	return name, version
}

func (s *Service) AssociateIdentityProviderConfig(clusterName string, in AssociateIdentityProviderConfigInput) (*Update, error) {
	if in.OIDC == nil || strings.TrimSpace(in.OIDC.IdentityProviderConfigName) == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterIDPs, ok := s.identityProviders[clusterName]
	if !ok {
		clusterIDPs = map[string]*OIDCIdentityProviderConfig{}
		s.identityProviders[clusterName] = clusterIDPs
	}
	key := identityProviderKey("oidc", in.OIDC.IdentityProviderConfigName)
	if _, exists := clusterIDPs[key]; exists {
		return nil, ErrIdentityProviderExists
	}

	idp := *in.OIDC
	idp.Status = "ACTIVE"
	if len(idp.Tags) == 0 {
		idp.Tags = cloneStringMap(in.Tags)
	} else {
		idp.Tags = cloneStringMap(idp.Tags)
	}
	clusterIDPs[key] = &idp

	update := s.createUpdateLocked(clusterName, "AssociateIdentityProviderConfig", []UpdateParam{
		{Type: "IdentityProviderConfigName", Value: idp.IdentityProviderConfigName},
	}, "", "")
	return cloneUpdate(update), nil
}

func (s *Service) DisassociateIdentityProviderConfig(clusterName, configType, configName string) (*Update, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterIDPs, ok := s.identityProviders[clusterName]
	if !ok {
		return nil, ErrIdentityProviderNotFound
	}
	key := identityProviderKey(configType, configName)
	if _, ok := clusterIDPs[key]; !ok {
		return nil, ErrIdentityProviderNotFound
	}
	delete(clusterIDPs, key)

	update := s.createUpdateLocked(clusterName, "DisassociateIdentityProviderConfig", []UpdateParam{
		{Type: "IdentityProviderConfigName", Value: configName},
	}, "", "")
	return cloneUpdate(update), nil
}

func (s *Service) DescribeIdentityProviderConfig(clusterName, configType, configName string) (*OIDCIdentityProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterIDPs, ok := s.identityProviders[clusterName]
	if !ok {
		return nil, ErrIdentityProviderNotFound
	}
	key := identityProviderKey(configType, configName)
	idp, ok := clusterIDPs[key]
	if !ok {
		return nil, ErrIdentityProviderNotFound
	}
	return cloneOIDCIdentityProviderConfig(idp), nil
}

func (s *Service) ListIdentityProviderConfigs(clusterName string) ([]IdentityProviderConfigRef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterIDPs, ok := s.identityProviders[clusterName]
	if !ok {
		return []IdentityProviderConfigRef{}, nil
	}

	out := make([]IdentityProviderConfigRef, 0, len(clusterIDPs))
	for _, idp := range clusterIDPs {
		out = append(out, IdentityProviderConfigRef{
			Name: idp.IdentityProviderConfigName,
			Type: "oidc",
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (s *Service) CreateAccessEntry(clusterName string, in CreateAccessEntryInput) (*AccessEntry, error) {
	if strings.TrimSpace(in.PrincipalArn) == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterEntries, ok := s.accessEntries[clusterName]
	if !ok {
		clusterEntries = map[string]*AccessEntry{}
		s.accessEntries[clusterName] = clusterEntries
	}
	if _, exists := clusterEntries[in.PrincipalArn]; exists {
		return nil, ErrAccessEntryAlreadyExists
	}

	entryType := strings.TrimSpace(in.Type)
	if entryType == "" {
		entryType = "STANDARD"
	}
	now := time.Now().UTC()
	entry := &AccessEntry{
		ClusterName:      clusterName,
		PrincipalArn:     in.PrincipalArn,
		AccessEntryArn:   accessEntryARN(clusterName, in.PrincipalArn),
		Type:             entryType,
		Username:         strings.TrimSpace(in.Username),
		KubernetesGroups: cloneStrings(in.KubernetesGroups),
		Tags:             cloneStringMap(in.Tags),
		CreatedAt:        now,
		ModifiedAt:       now,
	}
	clusterEntries[in.PrincipalArn] = entry
	if _, ok := s.associatedPolicies[clusterName]; !ok {
		s.associatedPolicies[clusterName] = map[string]map[string]*AssociatedAccessPolicy{}
	}
	if _, ok := s.associatedPolicies[clusterName][in.PrincipalArn]; !ok {
		s.associatedPolicies[clusterName][in.PrincipalArn] = map[string]*AssociatedAccessPolicy{}
	}
	s.tagsByARN[entry.AccessEntryArn] = cloneStringMap(entry.Tags)

	return cloneAccessEntry(entry), nil
}

func (s *Service) DescribeAccessEntry(clusterName, principalArn string) (*AccessEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterEntries, ok := s.accessEntries[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrAccessEntryNotFound
	}
	entry, ok := clusterEntries[principalArn]
	if !ok {
		return nil, ErrAccessEntryNotFound
	}
	return cloneAccessEntry(entry), nil
}

func (s *Service) ListAccessEntries(clusterName, associatedPolicyArn string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterEntries, ok := s.accessEntries[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return []string{}, nil
	}

	out := make([]string, 0, len(clusterEntries))
	for principalArn := range clusterEntries {
		if strings.TrimSpace(associatedPolicyArn) != "" {
			policiesByPrincipal, ok := s.associatedPolicies[clusterName]
			if !ok {
				continue
			}
			policies, ok := policiesByPrincipal[principalArn]
			if !ok {
				continue
			}
			if _, ok := policies[associatedPolicyArn]; !ok {
				continue
			}
		}
		out = append(out, principalArn)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Service) UpdateAccessEntry(clusterName, principalArn string, in UpdateAccessEntryInput) (*AccessEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterEntries, ok := s.accessEntries[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrAccessEntryNotFound
	}
	entry, ok := clusterEntries[principalArn]
	if !ok {
		return nil, ErrAccessEntryNotFound
	}
	if in.Username != nil {
		entry.Username = strings.TrimSpace(*in.Username)
	}
	if in.KubernetesGroups != nil {
		entry.KubernetesGroups = cloneStrings(in.KubernetesGroups)
	}
	entry.ModifiedAt = time.Now().UTC()
	return cloneAccessEntry(entry), nil
}

func (s *Service) DeleteAccessEntry(clusterName, principalArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterEntries, ok := s.accessEntries[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return ErrClusterNotFound
		}
		return ErrAccessEntryNotFound
	}
	entry, ok := clusterEntries[principalArn]
	if !ok {
		return ErrAccessEntryNotFound
	}
	delete(clusterEntries, principalArn)
	delete(s.tagsByARN, entry.AccessEntryArn)
	if policiesByPrincipal, ok := s.associatedPolicies[clusterName]; ok {
		delete(policiesByPrincipal, principalArn)
	}
	return nil
}

func (s *Service) AssociateAccessPolicy(clusterName, principalArn string, in AssociateAccessPolicyInput) (*AssociatedAccessPolicy, error) {
	if strings.TrimSpace(in.PolicyArn) == "" {
		return nil, ErrInvalidParameter
	}
	if !isKnownAccessPolicy(in.PolicyArn) {
		return nil, ErrAccessPolicyNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	clusterEntries, ok := s.accessEntries[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrAccessEntryNotFound
	}
	if _, ok := clusterEntries[principalArn]; !ok {
		return nil, ErrAccessEntryNotFound
	}

	scope := in.AccessScope
	scope.Type = strings.TrimSpace(scope.Type)
	if scope.Type == "" {
		scope.Type = "cluster"
	}
	now := time.Now().UTC()
	policy := &AssociatedAccessPolicy{
		PolicyArn:    in.PolicyArn,
		AccessScope:  AccessScope{Type: scope.Type, Namespaces: cloneStrings(scope.Namespaces)},
		AssociatedAt: now,
		ModifiedAt:   now,
	}

	clusterPolicies, ok := s.associatedPolicies[clusterName]
	if !ok {
		clusterPolicies = map[string]map[string]*AssociatedAccessPolicy{}
		s.associatedPolicies[clusterName] = clusterPolicies
	}
	principalPolicies, ok := clusterPolicies[principalArn]
	if !ok {
		principalPolicies = map[string]*AssociatedAccessPolicy{}
		clusterPolicies[principalArn] = principalPolicies
	}
	if existing, ok := principalPolicies[in.PolicyArn]; ok {
		policy.AssociatedAt = existing.AssociatedAt
	}
	principalPolicies[in.PolicyArn] = policy
	return cloneAssociatedAccessPolicy(policy), nil
}

func (s *Service) DisassociateAccessPolicy(clusterName, principalArn, policyArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterEntries, ok := s.accessEntries[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return ErrClusterNotFound
		}
		return ErrAccessEntryNotFound
	}
	if _, ok := clusterEntries[principalArn]; !ok {
		return ErrAccessEntryNotFound
	}

	clusterPolicies, ok := s.associatedPolicies[clusterName]
	if !ok {
		return ErrAssociatedPolicyNotFound
	}
	principalPolicies, ok := clusterPolicies[principalArn]
	if !ok {
		return ErrAssociatedPolicyNotFound
	}
	if _, ok := principalPolicies[policyArn]; !ok {
		return ErrAssociatedPolicyNotFound
	}
	delete(principalPolicies, policyArn)
	return nil
}

func (s *Service) ListAssociatedAccessPolicies(clusterName, principalArn string) ([]AssociatedAccessPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterEntries, ok := s.accessEntries[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrAccessEntryNotFound
	}
	if _, ok := clusterEntries[principalArn]; !ok {
		return nil, ErrAccessEntryNotFound
	}

	clusterPolicies, ok := s.associatedPolicies[clusterName]
	if !ok {
		return []AssociatedAccessPolicy{}, nil
	}
	principalPolicies, ok := clusterPolicies[principalArn]
	if !ok {
		return []AssociatedAccessPolicy{}, nil
	}

	out := make([]AssociatedAccessPolicy, 0, len(principalPolicies))
	for _, policy := range principalPolicies {
		out = append(out, *cloneAssociatedAccessPolicy(policy))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PolicyArn < out[j].PolicyArn
	})
	return out, nil
}

func (s *Service) ListAccessPolicies() []AccessPolicy {
	out := make([]AccessPolicy, len(defaultAccessPolicies))
	copy(out, defaultAccessPolicies)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) RegisterCluster(in RegisterClusterInput) (*Cluster, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.ConnectorRole) == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clusters[in.Name]; exists {
		return nil, ErrClusterAlreadyExists
	}

	now := time.Now().UTC()
	provider := strings.ToUpper(strings.TrimSpace(in.ConnectorCloud))
	if provider == "" {
		provider = "OTHER"
	}
	cluster := &Cluster{
		Name:      in.Name,
		Arn:       clusterARN(in.Name),
		CreatedAt: now,
		Version:   DefaultClusterVersion,
		RoleArn:   strings.TrimSpace(in.ConnectorRole),
		Status:    "ACTIVE",
		ConnectorConfig: &ConnectorConfig{
			ActivationID:     fmt.Sprintf("act-%012d", atomic.AddUint64(&s.updateCounter, 1)),
			ActivationCode:   fmt.Sprintf("code-%012d", atomic.AddUint64(&s.updateCounter, 1)),
			ActivationExpiry: now.Add(72 * time.Hour),
			Provider:         provider,
			RoleArn:          strings.TrimSpace(in.ConnectorRole),
		},
		ResourcesVpcConfig: ResourcesVpcConfig{EndpointPublicAccess: true},
		Tags:               cloneStringMap(in.Tags),
	}
	s.clusters[in.Name] = cluster
	s.nodegroups[in.Name] = map[string]*Nodegroup{}
	s.fargateProfiles[in.Name] = map[string]*FargateProfile{}
	s.addons[in.Name] = map[string]*Addon{}
	s.capabilities[in.Name] = map[string]*Capability{}
	s.identityProviders[in.Name] = map[string]*OIDCIdentityProviderConfig{}
	s.accessEntries[in.Name] = map[string]*AccessEntry{}
	s.associatedPolicies[in.Name] = map[string]map[string]*AssociatedAccessPolicy{}
	s.insights[in.Name] = defaultInsightsForCluster(cluster.Name, cluster.Version)
	s.insightsRefresh[in.Name] = &InsightsRefresh{
		Status:      "COMPLETED",
		StartedAt:   now,
		CompletedAt: now,
	}
	s.podIdentity[in.Name] = map[string]*PodIdentityAssociation{}
	s.encryptionConfig[in.Name] = nil
	s.updates[in.Name] = map[string]*Update{}
	s.tagsByARN[cluster.Arn] = cloneStringMap(cluster.Tags)
	return cloneCluster(cluster), nil
}

func (s *Service) DeregisterCluster(name string) (*Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[name]
	if !ok || cluster.ConnectorConfig == nil {
		return nil, ErrClusterNotFound
	}
	out := cloneCluster(cluster)
	out.Status = "DELETING"
	delete(s.clusters, name)
	delete(s.nodegroups, name)
	delete(s.fargateProfiles, name)
	delete(s.addons, name)
	delete(s.capabilities, name)
	delete(s.identityProviders, name)
	delete(s.accessEntries, name)
	delete(s.associatedPolicies, name)
	delete(s.insights, name)
	delete(s.insightsRefresh, name)
	delete(s.podIdentity, name)
	delete(s.encryptionConfig, name)
	delete(s.updates, name)
	delete(s.tagsByARN, cluster.Arn)
	return out, nil
}

func (s *Service) AssociateEncryptionConfig(clusterName string, config []EncryptionConfigEntry) (*Update, error) {
	if len(config) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	s.encryptionConfig[clusterName] = cloneEncryptionConfigEntries(config)
	update := s.createUpdateLocked(clusterName, "AssociateEncryptionConfig", []UpdateParam{
		{Type: "EncryptionConfig", Value: fmt.Sprintf("%d", len(config))},
	}, "", "")
	return cloneUpdate(update), nil
}

func (s *Service) CreateCapability(clusterName string, in CreateCapabilityInput) (*Capability, error) {
	if strings.TrimSpace(in.CapabilityName) == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterCapabilities, ok := s.capabilities[clusterName]
	if !ok {
		clusterCapabilities = map[string]*Capability{}
		s.capabilities[clusterName] = clusterCapabilities
	}
	if _, exists := clusterCapabilities[in.CapabilityName]; exists {
		return nil, ErrCapabilityAlreadyExists
	}

	now := time.Now().UTC()
	capability := &Capability{
		ClusterName:    clusterName,
		CapabilityName: in.CapabilityName,
		CapabilityArn:  capabilityARN(clusterName, in.CapabilityName),
		Status:         "ACTIVE",
		CreatedAt:      now,
		ModifiedAt:     now,
		Tags:           cloneStringMap(in.Tags),
	}
	clusterCapabilities[in.CapabilityName] = capability
	s.tagsByARN[capability.CapabilityArn] = cloneStringMap(capability.Tags)
	return cloneCapability(capability), nil
}

func (s *Service) ListCapabilities(clusterName string) ([]Capability, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterCapabilities, ok := s.capabilities[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return []Capability{}, nil
	}
	out := make([]Capability, 0, len(clusterCapabilities))
	for _, capability := range clusterCapabilities {
		out = append(out, *cloneCapability(capability))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CapabilityName < out[j].CapabilityName
	})
	return out, nil
}

func (s *Service) DescribeCapability(clusterName, capabilityName string) (*Capability, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterCapabilities, ok := s.capabilities[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrCapabilityNotFound
	}
	capability, ok := clusterCapabilities[capabilityName]
	if !ok {
		return nil, ErrCapabilityNotFound
	}
	return cloneCapability(capability), nil
}

func (s *Service) UpdateCapability(clusterName, capabilityName string, in UpdateCapabilityInput) (*Update, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterCapabilities, ok := s.capabilities[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrCapabilityNotFound
	}
	capability, ok := clusterCapabilities[capabilityName]
	if !ok {
		return nil, ErrCapabilityNotFound
	}
	if in.Tags != nil {
		capability.Tags = cloneStringMap(in.Tags)
		s.tagsByARN[capability.CapabilityArn] = cloneStringMap(in.Tags)
	}
	capability.ModifiedAt = time.Now().UTC()

	update := s.createUpdateLocked(clusterName, "AutoModeUpdate", []UpdateParam{
		{Type: "ComputeConfig", Value: capabilityName},
	}, "", "")
	return cloneUpdate(update), nil
}

func (s *Service) DeleteCapability(clusterName, capabilityName string) (*Capability, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterCapabilities, ok := s.capabilities[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrCapabilityNotFound
	}
	capability, ok := clusterCapabilities[capabilityName]
	if !ok {
		return nil, ErrCapabilityNotFound
	}
	out := cloneCapability(capability)
	out.Status = "DELETING"
	delete(clusterCapabilities, capabilityName)
	delete(s.tagsByARN, capability.CapabilityArn)
	return out, nil
}

func (s *Service) ListInsights(clusterName string, in ListInsightsInput) ([]InsightSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, ErrClusterNotFound
	}
	clusterInsights, ok := s.insights[clusterName]
	if !ok {
		clusterInsights = defaultInsightsForCluster(clusterName, cluster.Version)
	}

	categories := buildMembershipSet(in.Categories)
	versions := buildMembershipSet(in.KubernetesVersions)
	statuses := buildMembershipSet(in.Statuses)

	out := make([]InsightSummary, 0, len(clusterInsights))
	for _, insight := range clusterInsights {
		if len(categories) > 0 && !categories[insight.Category] {
			continue
		}
		if len(versions) > 0 && !versions[insight.KubernetesVersion] {
			continue
		}
		if len(statuses) > 0 && !statuses[insight.InsightStatus.Status] {
			continue
		}
		out = append(out, insightToSummary(insight))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (s *Service) DescribeInsight(clusterName, insightID string) (*Insight, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, ErrClusterNotFound
	}
	clusterInsights, ok := s.insights[clusterName]
	if !ok {
		clusterInsights = defaultInsightsForCluster(clusterName, cluster.Version)
	}
	insight, ok := clusterInsights[insightID]
	if !ok {
		return nil, ErrInsightNotFound
	}
	return cloneInsight(insight), nil
}

func (s *Service) StartInsightsRefresh(clusterName string) (*InsightsRefresh, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, ErrClusterNotFound
	}
	clusterInsights, ok := s.insights[clusterName]
	if !ok {
		clusterInsights = defaultInsightsForCluster(clusterName, cluster.Version)
		s.insights[clusterName] = clusterInsights
	}
	now := time.Now().UTC()
	for _, insight := range clusterInsights {
		insight.LastRefreshTime = now
	}
	refresh := &InsightsRefresh{
		Status:      "COMPLETED",
		StartedAt:   now,
		CompletedAt: now,
	}
	s.insightsRefresh[clusterName] = refresh
	return cloneInsightsRefresh(refresh), nil
}

func (s *Service) DescribeInsightsRefresh(clusterName string) (*InsightsRefresh, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, ErrClusterNotFound
	}
	refresh, ok := s.insightsRefresh[clusterName]
	if !ok {
		return &InsightsRefresh{
			Status:      "COMPLETED",
			StartedAt:   cluster.CreatedAt,
			CompletedAt: cluster.CreatedAt,
		}, nil
	}
	return cloneInsightsRefresh(refresh), nil
}

func (s *Service) CreatePodIdentityAssociation(clusterName string, in CreatePodIdentityAssociationInput) (*PodIdentityAssociation, error) {
	if strings.TrimSpace(in.Namespace) == "" || strings.TrimSpace(in.ServiceAccount) == "" || strings.TrimSpace(in.RoleArn) == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, ErrClusterNotFound
	}
	clusterAssociations, ok := s.podIdentity[clusterName]
	if !ok {
		clusterAssociations = map[string]*PodIdentityAssociation{}
		s.podIdentity[clusterName] = clusterAssociations
	}
	for _, association := range clusterAssociations {
		if association.Namespace == in.Namespace && association.ServiceAccount == in.ServiceAccount {
			return nil, ErrPodIdentityExists
		}
	}
	associationID := fmt.Sprintf("pia-%012d", atomic.AddUint64(&s.updateCounter, 1))
	now := time.Now().UTC()
	association := &PodIdentityAssociation{
		ClusterName:    clusterName,
		Namespace:      in.Namespace,
		ServiceAccount: in.ServiceAccount,
		RoleArn:        in.RoleArn,
		AssociationArn: podIdentityAssociationARN(clusterName, associationID),
		AssociationID:  associationID,
		Tags:           cloneStringMap(in.Tags),
		CreatedAt:      now,
		ModifiedAt:     now,
	}
	clusterAssociations[associationID] = association
	s.tagsByARN[association.AssociationArn] = cloneStringMap(association.Tags)
	return clonePodIdentityAssociation(association), nil
}

func (s *Service) DescribePodIdentityAssociation(clusterName, associationID string) (*PodIdentityAssociation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterAssociations, ok := s.podIdentity[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrPodIdentityNotFound
	}
	association, ok := clusterAssociations[associationID]
	if !ok {
		return nil, ErrPodIdentityNotFound
	}
	return clonePodIdentityAssociation(association), nil
}

func (s *Service) ListPodIdentityAssociations(clusterName, namespace, serviceAccount string) ([]PodIdentityAssociationSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterAssociations, ok := s.podIdentity[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return []PodIdentityAssociationSummary{}, nil
	}
	out := make([]PodIdentityAssociationSummary, 0, len(clusterAssociations))
	for _, association := range clusterAssociations {
		if strings.TrimSpace(namespace) != "" && association.Namespace != namespace {
			continue
		}
		if strings.TrimSpace(serviceAccount) != "" && association.ServiceAccount != serviceAccount {
			continue
		}
		out = append(out, PodIdentityAssociationSummary{
			ClusterName:    association.ClusterName,
			Namespace:      association.Namespace,
			ServiceAccount: association.ServiceAccount,
			AssociationArn: association.AssociationArn,
			AssociationID:  association.AssociationID,
			OwnerArn:       association.OwnerArn,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].AssociationID < out[j].AssociationID
	})
	return out, nil
}

func (s *Service) UpdatePodIdentityAssociation(clusterName, associationID string, in UpdatePodIdentityAssociationInput) (*PodIdentityAssociation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterAssociations, ok := s.podIdentity[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrPodIdentityNotFound
	}
	association, ok := clusterAssociations[associationID]
	if !ok {
		return nil, ErrPodIdentityNotFound
	}
	if strings.TrimSpace(in.RoleArn) != "" {
		association.RoleArn = strings.TrimSpace(in.RoleArn)
	}
	association.ModifiedAt = time.Now().UTC()
	return clonePodIdentityAssociation(association), nil
}

func (s *Service) DeletePodIdentityAssociation(clusterName, associationID string) (*PodIdentityAssociation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterAssociations, ok := s.podIdentity[clusterName]
	if !ok {
		if _, clusterExists := s.clusters[clusterName]; !clusterExists {
			return nil, ErrClusterNotFound
		}
		return nil, ErrPodIdentityNotFound
	}
	association, ok := clusterAssociations[associationID]
	if !ok {
		return nil, ErrPodIdentityNotFound
	}
	out := clonePodIdentityAssociation(association)
	delete(clusterAssociations, associationID)
	delete(s.tagsByARN, association.AssociationArn)
	return out, nil
}

func (s *Service) CreateEksAnywhereSubscription(in CreateEksAnywhereSubscriptionInput) (*EksAnywhereSubscription, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.subscriptions {
		if strings.EqualFold(existing.Name, strings.TrimSpace(in.Name)) {
			return nil, ErrSubscriptionExists
		}
	}
	now := time.Now().UTC()
	duration := in.Term.Duration
	if duration <= 0 {
		duration = 12
	}
	unit := strings.ToUpper(strings.TrimSpace(in.Term.Unit))
	if unit == "" {
		unit = "MONTHS"
	}
	licenseQuantity := in.LicenseQuantity
	if licenseQuantity <= 0 {
		licenseQuantity = 1
	}
	licenseType := strings.TrimSpace(in.LicenseType)
	if licenseType == "" {
		licenseType = "Cluster"
	}
	id := fmt.Sprintf("sub-%012d", atomic.AddUint64(&s.updateCounter, 1))
	subscription := &EksAnywhereSubscription{
		Name:            strings.TrimSpace(in.Name),
		ID:              id,
		Arn:             subscriptionARN(id),
		CreatedAt:       now,
		EffectiveDate:   now,
		ExpirationDate:  now.Add(time.Duration(duration) * 30 * 24 * time.Hour),
		LicenseQuantity: licenseQuantity,
		LicenseType:     licenseType,
		Term: EksAnywhereSubscriptionTerm{
			Duration: duration,
			Unit:     unit,
		},
		Status:      "ACTIVE",
		AutoRenew:   in.AutoRenew,
		LicenseArns: []string{fmt.Sprintf("arn:aws:license-manager:%s:%s:license/%s", DefaultRegion, DefaultAccountID, id)},
		Tags:        cloneStringMap(in.Tags),
	}
	s.subscriptions[id] = subscription
	s.tagsByARN[subscription.Arn] = cloneStringMap(subscription.Tags)
	return cloneEksAnywhereSubscription(subscription), nil
}

func (s *Service) DescribeEksAnywhereSubscription(subscriptionID string) (*EksAnywhereSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subscription, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	return cloneEksAnywhereSubscription(subscription), nil
}

func (s *Service) ListEksAnywhereSubscriptions() []EksAnywhereSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]EksAnywhereSubscription, 0, len(s.subscriptions))
	for _, subscription := range s.subscriptions {
		out = append(out, *cloneEksAnywhereSubscription(subscription))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *Service) UpdateEksAnywhereSubscription(subscriptionID string, in UpdateEksAnywhereSubscriptionInput) (*EksAnywhereSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscription, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	if in.AutoRenew != nil {
		subscription.AutoRenew = *in.AutoRenew
	}
	if in.Tags != nil {
		subscription.Tags = cloneStringMap(in.Tags)
		s.tagsByARN[subscription.Arn] = cloneStringMap(in.Tags)
	}
	return cloneEksAnywhereSubscription(subscription), nil
}

func (s *Service) DeleteEksAnywhereSubscription(subscriptionID string) (*EksAnywhereSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscription, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	out := cloneEksAnywhereSubscription(subscription)
	out.Status = "DELETING"
	delete(s.subscriptions, subscriptionID)
	delete(s.tagsByARN, subscription.Arn)
	return out, nil
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) error {
	if strings.TrimSpace(resourceARN) == "" || len(tags) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.resourceExistsLocked(resourceARN) && s.tagsByARN[resourceARN] == nil {
		return ErrTagNotFound
	}
	current := cloneStringMap(s.tagsByARN[resourceARN])
	if current == nil {
		current = map[string]string{}
	}
	for key, value := range tags {
		current[key] = value
	}
	s.tagsByARN[resourceARN] = current
	s.applyTagsToResourceLocked(resourceARN, current)
	return nil
}

func (s *Service) ListTagsForResource(resourceARN string) (map[string]string, error) {
	if strings.TrimSpace(resourceARN) == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if tags, ok := s.tagsByARN[resourceARN]; ok {
		return cloneStringMap(tags), nil
	}
	if s.resourceExistsLocked(resourceARN) {
		return map[string]string{}, nil
	}
	return nil, ErrTagNotFound
}

func (s *Service) UntagResource(resourceARN string, tagKeys []string) error {
	if strings.TrimSpace(resourceARN) == "" || len(tagKeys) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := cloneStringMap(s.tagsByARN[resourceARN])
	if current == nil {
		if !s.resourceExistsLocked(resourceARN) {
			return ErrTagNotFound
		}
		current = map[string]string{}
	}
	for _, key := range tagKeys {
		delete(current, key)
	}
	if len(current) == 0 {
		delete(s.tagsByARN, resourceARN)
		s.applyTagsToResourceLocked(resourceARN, nil)
		return nil
	}
	s.tagsByARN[resourceARN] = current
	s.applyTagsToResourceLocked(resourceARN, current)
	return nil
}

func (s *Service) resourceExistsLocked(resourceARN string) bool {
	for _, cluster := range s.clusters {
		if cluster.Arn == resourceARN {
			return true
		}
	}
	for _, clusterNodegroups := range s.nodegroups {
		for _, nodegroup := range clusterNodegroups {
			if nodegroup.NodegroupArn == resourceARN {
				return true
			}
		}
	}
	for _, clusterProfiles := range s.fargateProfiles {
		for _, profile := range clusterProfiles {
			if profile.FargateProfileArn == resourceARN {
				return true
			}
		}
	}
	for _, clusterAddons := range s.addons {
		for _, addon := range clusterAddons {
			if addon.AddonArn == resourceARN {
				return true
			}
		}
	}
	for _, clusterEntries := range s.accessEntries {
		for _, entry := range clusterEntries {
			if entry.AccessEntryArn == resourceARN {
				return true
			}
		}
	}
	for _, clusterCapabilities := range s.capabilities {
		for _, capability := range clusterCapabilities {
			if capability.CapabilityArn == resourceARN {
				return true
			}
		}
	}
	for _, clusterAssociations := range s.podIdentity {
		for _, association := range clusterAssociations {
			if association.AssociationArn == resourceARN {
				return true
			}
		}
	}
	for _, subscription := range s.subscriptions {
		if subscription.Arn == resourceARN {
			return true
		}
	}
	return false
}

func (s *Service) applyTagsToResourceLocked(resourceARN string, tags map[string]string) {
	cloned := cloneStringMap(tags)
	for _, cluster := range s.clusters {
		if cluster.Arn == resourceARN {
			cluster.Tags = cloned
			return
		}
	}
	for _, clusterNodegroups := range s.nodegroups {
		for _, nodegroup := range clusterNodegroups {
			if nodegroup.NodegroupArn == resourceARN {
				nodegroup.Tags = cloned
				return
			}
		}
	}
	for _, clusterProfiles := range s.fargateProfiles {
		for _, profile := range clusterProfiles {
			if profile.FargateProfileArn == resourceARN {
				profile.Tags = cloned
				return
			}
		}
	}
	for _, clusterAddons := range s.addons {
		for _, addon := range clusterAddons {
			if addon.AddonArn == resourceARN {
				addon.Tags = cloned
				return
			}
		}
	}
	for _, clusterEntries := range s.accessEntries {
		for _, entry := range clusterEntries {
			if entry.AccessEntryArn == resourceARN {
				entry.Tags = cloned
				return
			}
		}
	}
	for _, clusterCapabilities := range s.capabilities {
		for _, capability := range clusterCapabilities {
			if capability.CapabilityArn == resourceARN {
				capability.Tags = cloned
				return
			}
		}
	}
	for _, clusterAssociations := range s.podIdentity {
		for _, association := range clusterAssociations {
			if association.AssociationArn == resourceARN {
				association.Tags = cloned
				return
			}
		}
	}
	for _, subscription := range s.subscriptions {
		if subscription.Arn == resourceARN {
			subscription.Tags = cloned
			return
		}
	}
}

func buildMembershipSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out[trimmed] = true
	}
	return out
}

func insightToSummary(insight *Insight) InsightSummary {
	return InsightSummary{
		ID:                 insight.ID,
		Name:               insight.Name,
		Category:           insight.Category,
		KubernetesVersion:  insight.KubernetesVersion,
		LastRefreshTime:    insight.LastRefreshTime,
		LastTransitionTime: insight.LastTransitionTime,
		Description:        insight.Description,
		InsightStatus:      insight.InsightStatus,
	}
}

func defaultInsightsForCluster(clusterName, version string) map[string]*Insight {
	now := time.Now().UTC()
	if strings.TrimSpace(version) == "" {
		version = DefaultClusterVersion
	}
	return map[string]*Insight{
		"insight-upgrade-readiness": {
			ID:                 "insight-upgrade-readiness",
			Name:               "UpgradeReadiness",
			Category:           "UPGRADE_READINESS",
			KubernetesVersion:  version,
			LastRefreshTime:    now,
			LastTransitionTime: now,
			Description:        "Cluster has no blocking upgrade issues.",
			InsightStatus: InsightStatus{
				Status: "PASSING",
				Reason: "No deprecated APIs detected",
			},
			Recommendation: "Continue regular upgrade checks.",
			AdditionalInfo: map[string]string{
				"cluster": clusterName,
			},
		},
	}
}

func (s *Service) createUpdateLocked(clusterName, updateType string, params []UpdateParam, nodegroupName, addonName string) *Update {
	clusterUpdates, ok := s.updates[clusterName]
	if !ok {
		clusterUpdates = map[string]*Update{}
		s.updates[clusterName] = clusterUpdates
	}
	updateID := fmt.Sprintf("upd-%012d", atomic.AddUint64(&s.updateCounter, 1))
	update := &Update{
		ID:            updateID,
		Status:        "Successful",
		Type:          updateType,
		Params:        cloneUpdateParams(params),
		CreatedAt:     time.Now().UTC(),
		NodegroupName: nodegroupName,
		AddonName:     addonName,
	}
	clusterUpdates[updateID] = update
	return update
}

func clusterARN(name string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", DefaultRegion, DefaultAccountID, name)
}

func nodegroupARN(clusterName, nodegroupName string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:nodegroup/%s/%s/stackyard", DefaultRegion, DefaultAccountID, clusterName, nodegroupName)
}

func fargateProfileARN(clusterName, profileName string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:fargateprofile/%s/%s/stackyard", DefaultRegion, DefaultAccountID, clusterName, profileName)
}

func addonARN(clusterName, addonName string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:addon/%s/%s/stackyard", DefaultRegion, DefaultAccountID, clusterName, addonName)
}

func capabilityARN(clusterName, capabilityName string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:capability/%s/%s", DefaultRegion, DefaultAccountID, clusterName, capabilityName)
}

func accessEntryARN(clusterName, principalArn string) string {
	replacer := strings.NewReplacer(":", "_", "/", "_")
	return fmt.Sprintf("arn:aws:eks:%s:%s:access-entry/%s/%s", DefaultRegion, DefaultAccountID, clusterName, replacer.Replace(principalArn))
}

func podIdentityAssociationARN(clusterName, associationID string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:podidentityassociation/%s/%s", DefaultRegion, DefaultAccountID, clusterName, associationID)
}

func subscriptionARN(subscriptionID string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:eks-anywhere-subscription/%s", DefaultRegion, DefaultAccountID, subscriptionID)
}

func identityProviderKey(configType, configName string) string {
	t := strings.ToLower(strings.TrimSpace(configType))
	if t == "" {
		t = "oidc"
	}
	return t + ":" + strings.TrimSpace(configName)
}

func isKnownAccessPolicy(policyArn string) bool {
	for _, policy := range defaultAccessPolicies {
		if policy.Arn == policyArn {
			return true
		}
	}
	return false
}

func cloneCluster(in *Cluster) *Cluster {
	if in == nil {
		return nil
	}
	out := *in
	out.ConnectorConfig = cloneConnectorConfig(in.ConnectorConfig)
	out.ResourcesVpcConfig.SubnetIDs = cloneStrings(in.ResourcesVpcConfig.SubnetIDs)
	out.Tags = cloneStringMap(in.Tags)
	return &out
}

func cloneConnectorConfig(in *ConnectorConfig) *ConnectorConfig {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneNodegroup(in *Nodegroup) *Nodegroup {
	if in == nil {
		return nil
	}
	out := *in
	out.Subnets = cloneStrings(in.Subnets)
	out.Tags = cloneStringMap(in.Tags)
	return &out
}

func cloneFargateProfile(in *FargateProfile) *FargateProfile {
	if in == nil {
		return nil
	}
	out := *in
	out.Subnets = cloneStrings(in.Subnets)
	out.Selectors = cloneSelectors(in.Selectors)
	out.Tags = cloneStringMap(in.Tags)
	return &out
}

func cloneAddon(in *Addon) *Addon {
	if in == nil {
		return nil
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return &out
}

func cloneCapability(in *Capability) *Capability {
	if in == nil {
		return nil
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return &out
}

func cloneInsight(in *Insight) *Insight {
	if in == nil {
		return nil
	}
	out := *in
	out.AdditionalInfo = cloneStringMap(in.AdditionalInfo)
	return &out
}

func cloneInsightsRefresh(in *InsightsRefresh) *InsightsRefresh {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneOIDCIdentityProviderConfig(in *OIDCIdentityProviderConfig) *OIDCIdentityProviderConfig {
	if in == nil {
		return nil
	}
	out := *in
	out.RequiredClaims = cloneStringMap(in.RequiredClaims)
	out.Tags = cloneStringMap(in.Tags)
	return &out
}

func cloneAccessEntry(in *AccessEntry) *AccessEntry {
	if in == nil {
		return nil
	}
	out := *in
	out.KubernetesGroups = cloneStrings(in.KubernetesGroups)
	out.Tags = cloneStringMap(in.Tags)
	return &out
}

func cloneAssociatedAccessPolicy(in *AssociatedAccessPolicy) *AssociatedAccessPolicy {
	if in == nil {
		return nil
	}
	out := *in
	out.AccessScope = AccessScope{
		Type:       in.AccessScope.Type,
		Namespaces: cloneStrings(in.AccessScope.Namespaces),
	}
	return &out
}

func clonePodIdentityAssociation(in *PodIdentityAssociation) *PodIdentityAssociation {
	if in == nil {
		return nil
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return &out
}

func cloneEksAnywhereSubscription(in *EksAnywhereSubscription) *EksAnywhereSubscription {
	if in == nil {
		return nil
	}
	out := *in
	out.LicenseArns = cloneStrings(in.LicenseArns)
	out.Tags = cloneStringMap(in.Tags)
	return &out
}

func cloneUpdate(in *Update) *Update {
	if in == nil {
		return nil
	}
	out := *in
	out.Params = cloneUpdateParams(in.Params)
	out.Errors = cloneUpdateErrors(in.Errors)
	return &out
}

func cloneUpdateParams(in []UpdateParam) []UpdateParam {
	if len(in) == 0 {
		return nil
	}
	out := make([]UpdateParam, len(in))
	copy(out, in)
	return out
}

func cloneUpdateErrors(in []UpdateError) []UpdateError {
	if len(in) == 0 {
		return nil
	}
	out := make([]UpdateError, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].ResourceIDs = cloneStrings(in[i].ResourceIDs)
	}
	return out
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneSelectors(in []FargateProfileSelector) []FargateProfileSelector {
	if len(in) == 0 {
		return nil
	}
	out := make([]FargateProfileSelector, len(in))
	for i := range in {
		out[i] = FargateProfileSelector{
			Namespace: in[i].Namespace,
			Labels:    cloneStringMap(in[i].Labels),
		}
	}
	return out
}

func cloneEncryptionConfigEntries(in []EncryptionConfigEntry) []EncryptionConfigEntry {
	if len(in) == 0 {
		return nil
	}
	out := make([]EncryptionConfigEntry, len(in))
	for i := range in {
		out[i] = EncryptionConfigEntry{
			Resources: cloneStrings(in[i].Resources),
			Provider: EncryptionProvider{
				KeyArn: in[i].Provider.KeyArn,
			},
		}
	}
	return out
}
