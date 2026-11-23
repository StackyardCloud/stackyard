package sesv2

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Bounds on caller-controlled growth of the tenant and multi-region endpoint
// stores.
const (
	maxTenants             = 1000
	maxResourcesPerTenant  = 1000
	maxMultiRegionEndpoint = 100
)

// ErrLimitExceeded reports that a store bound was reached.
var ErrLimitExceeded = errors.New("limit exceeded")

type Tenant struct {
	Name              string
	ID                string
	ARN               string
	CreatedAt         time.Time
	Tags              map[string]string
	SuppressedReasons []string
	SuppressionScope  string
	// Resources maps an associated resource ARN to when it was associated.
	Resources map[string]time.Time
}

type TenantResource struct {
	ResourceArn  string
	ResourceType string
}

type ResourceTenant struct {
	TenantName   string
	TenantID     string
	ResourceArn  string
	AssociatedAt time.Time
}

type MultiRegionEndpoint struct {
	Name      string
	ID        string
	Regions   []string
	CreatedAt time.Time
	Tags      map[string]string
}

func cloneTenant(t Tenant) Tenant {
	out := t
	out.Tags = mergeTags(nil, t.Tags)
	out.SuppressedReasons = append([]string(nil), t.SuppressedReasons...)
	out.Resources = make(map[string]time.Time, len(t.Resources))
	for arn, at := range t.Resources {
		out.Resources[arn] = at
	}
	return out
}

func (s *Service) CreateTenant(name string, tags map[string]string, suppressedReasons []string, scope string) (Tenant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Tenant{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[name]; ok {
		return Tenant{}, ErrAlreadyExists
	}
	if len(s.tenants) >= maxTenants {
		return Tenant{}, ErrLimitExceeded
	}
	s.seq++
	id := fmt.Sprintf("tenant-%012d", s.seq)
	t := &Tenant{
		Name:              name,
		ID:                id,
		ARN:               "arn:aws:ses:us-east-1:123456789012:tenant/" + name + "/" + id,
		CreatedAt:         time.Now().UTC(),
		Tags:              mergeTags(nil, tags),
		SuppressedReasons: append([]string(nil), suppressedReasons...),
		SuppressionScope:  scope,
		Resources:         map[string]time.Time{},
	}
	s.tenants[name] = t
	return cloneTenant(*t), nil
}

func (s *Service) GetTenant(name string) (Tenant, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[strings.TrimSpace(name)]
	if !ok {
		return Tenant{}, false
	}
	return cloneTenant(*t), true
}

func (s *Service) ListTenants() []Tenant {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		out = append(out, cloneTenant(*t))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeleteTenant(name string) bool {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[name]; !ok {
		return false
	}
	delete(s.tenants, name)
	return true
}

func (s *Service) AssociateTenantResource(name, resourceArn string) error {
	name, resourceArn = strings.TrimSpace(name), strings.TrimSpace(resourceArn)
	if name == "" || tenantResourceType(resourceArn) == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[name]
	if !ok {
		return ErrNotFound
	}
	if _, ok := t.Resources[resourceArn]; ok {
		return ErrAlreadyExists
	}
	if len(t.Resources) >= maxResourcesPerTenant {
		return ErrLimitExceeded
	}
	t.Resources[resourceArn] = time.Now().UTC()
	return nil
}

func (s *Service) DisassociateTenantResource(name, resourceArn string) error {
	name, resourceArn = strings.TrimSpace(name), strings.TrimSpace(resourceArn)
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[name]
	if !ok {
		return ErrNotFound
	}
	if _, ok := t.Resources[resourceArn]; !ok {
		return ErrNotFound
	}
	delete(t.Resources, resourceArn)
	return nil
}

// ListTenantResources returns the tenant's resources, optionally limited to
// one ResourceType.
func (s *Service) ListTenantResources(name, resourceType string) ([]TenantResource, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[strings.TrimSpace(name)]
	if !ok {
		return nil, false
	}
	out := make([]TenantResource, 0, len(t.Resources))
	for arn := range t.Resources {
		typ := tenantResourceType(arn)
		if resourceType != "" && typ != resourceType {
			continue
		}
		out = append(out, TenantResource{ResourceArn: arn, ResourceType: typ})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ResourceArn < out[j].ResourceArn })
	return out, true
}

func (s *Service) ListResourceTenants(resourceArn string) []ResourceTenant {
	resourceArn = strings.TrimSpace(resourceArn)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ResourceTenant{}
	for _, t := range s.tenants {
		if at, ok := t.Resources[resourceArn]; ok {
			out = append(out, ResourceTenant{TenantName: t.Name, TenantID: t.ID, ResourceArn: resourceArn, AssociatedAt: at})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TenantName < out[j].TenantName })
	return out
}

// tenantResourceType maps an SES resource ARN to its tenant ResourceType, or
// "" when the ARN is not an associable resource.
func tenantResourceType(arn string) string {
	switch {
	case !strings.HasPrefix(arn, "arn:"):
		return ""
	case strings.Contains(arn, ":identity/"):
		return "EMAIL_IDENTITY"
	case strings.Contains(arn, ":configuration-set/"):
		return "CONFIGURATION_SET"
	case strings.Contains(arn, ":template/"):
		return "EMAIL_TEMPLATE"
	}
	return ""
}

func cloneMultiRegionEndpoint(e MultiRegionEndpoint) MultiRegionEndpoint {
	out := e
	out.Regions = append([]string(nil), e.Regions...)
	out.Tags = mergeTags(nil, e.Tags)
	return out
}

func (s *Service) CreateMultiRegionEndpoint(name string, regions []string, tags map[string]string) (MultiRegionEndpoint, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(regions) == 0 {
		return MultiRegionEndpoint{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.multiRegion[name]; ok {
		return MultiRegionEndpoint{}, ErrAlreadyExists
	}
	if len(s.multiRegion) >= maxMultiRegionEndpoint {
		return MultiRegionEndpoint{}, ErrLimitExceeded
	}
	s.seq++
	e := &MultiRegionEndpoint{
		Name:      name,
		ID:        fmt.Sprintf("mre-%012d", s.seq),
		Regions:   append([]string(nil), regions...),
		CreatedAt: time.Now().UTC(),
		Tags:      mergeTags(nil, tags),
	}
	s.multiRegion[name] = e
	return cloneMultiRegionEndpoint(*e), nil
}

func (s *Service) GetMultiRegionEndpoint(name string) (MultiRegionEndpoint, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.multiRegion[strings.TrimSpace(name)]
	if !ok {
		return MultiRegionEndpoint{}, false
	}
	return cloneMultiRegionEndpoint(*e), true
}

func (s *Service) DeleteMultiRegionEndpoint(name string) bool {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.multiRegion[name]; !ok {
		return false
	}
	delete(s.multiRegion, name)
	return true
}

func (s *Service) ListMultiRegionEndpoints() []MultiRegionEndpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]MultiRegionEndpoint, 0, len(s.multiRegion))
	for _, e := range s.multiRegion {
		out = append(out, cloneMultiRegionEndpoint(*e))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
