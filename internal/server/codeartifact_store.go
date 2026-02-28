package server

import (
	"fmt"
	"net/http"
	neturl "net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type codeArtifactStore struct {
	mu             sync.Mutex
	nextID         int64
	domains        map[string]*codeArtifactDomain
	repositories   map[string]*codeArtifactRepository
	packageGroups  map[string]*codeArtifactPackageGroup
	packages       map[string]*codeArtifactPackage
	resourceTags   map[string]map[string]string
	domainPolicies map[string]string
	repoPolicies   map[string]string
}

type codeArtifactDomain struct {
	Name        string
	Owner       string
	Arn         string
	Status      string
	CreatedTime time.Time
	UpdatedTime time.Time
}

type codeArtifactRepository struct {
	Domain              string
	DomainOwner         string
	Name                string
	Description         string
	Arn                 string
	ExternalConnections []string
	CreatedTime         time.Time
	UpdatedTime         time.Time
}

type codeArtifactPackageGroup struct {
	Domain      string
	DomainOwner string
	Pattern     string
	Arn         string
	Description string
	ContactInfo string
	UpdatedTime time.Time
}

type codeArtifactPackage struct {
	Domain      string
	DomainOwner string
	Repository  string
	Format      string
	Namespace   string
	Name        string
	Versions    map[string]*codeArtifactPackageVersion
}

type codeArtifactPackageVersion struct {
	Version      string
	Status       string
	Revision     string
	PublishedAt  time.Time
	Assets       map[string][]byte
	Readme       string
	Dependencies []string
}

type codeArtifactResponse struct {
	Status      int
	ContentType string
	Body        any
	RawBody     []byte
}

func newCodeArtifactStore() *codeArtifactStore {
	now := time.Now().UTC()
	domain := &codeArtifactDomain{
		Name:        "stackyard-domain",
		Owner:       "123456789012",
		Arn:         codeArtifactDomainARN("stackyard-domain"),
		Status:      "Active",
		CreatedTime: now,
		UpdatedTime: now,
	}
	repo := &codeArtifactRepository{
		Domain:              domain.Name,
		DomainOwner:         domain.Owner,
		Name:                "stackyard-repo",
		Description:         "seed repository",
		Arn:                 codeArtifactRepositoryARN(domain.Name, "stackyard-repo"),
		ExternalConnections: []string{},
		CreatedTime:         now,
		UpdatedTime:         now,
	}
	pkg := &codeArtifactPackage{
		Domain:      domain.Name,
		DomainOwner: domain.Owner,
		Repository:  repo.Name,
		Format:      "npm",
		Namespace:   "stackyard",
		Name:        "seed-package",
		Versions: map[string]*codeArtifactPackageVersion{
			"1.0.0": {
				Version:      "1.0.0",
				Status:       "Published",
				Revision:     "rev-seed",
				PublishedAt:  now,
				Assets:       map[string][]byte{"package.tgz": []byte("stackyard")},
				Readme:       "# stackyard",
				Dependencies: []string{},
			},
		},
	}
	group := &codeArtifactPackageGroup{
		Domain:      domain.Name,
		DomainOwner: domain.Owner,
		Pattern:     "/stackyard/*",
		Arn:         codeArtifactPackageGroupARN(domain.Name, "/stackyard/*"),
		Description: "seed package group",
		ContactInfo: "stackyard@example.com",
		UpdatedTime: now,
	}

	return &codeArtifactStore{
		nextID: 1,
		domains: map[string]*codeArtifactDomain{
			domain.Name: domain,
		},
		repositories: map[string]*codeArtifactRepository{
			codeArtifactRepoKey(domain.Name, repo.Name): repo,
		},
		packageGroups: map[string]*codeArtifactPackageGroup{
			codeArtifactPackageGroupKey(domain.Name, group.Pattern): group,
		},
		packages: map[string]*codeArtifactPackage{
			codeArtifactPackageKey(domain.Name, repo.Name, pkg.Format, pkg.Namespace, pkg.Name): pkg,
		},
		resourceTags: map[string]map[string]string{
			domain.Arn: {"seed": "true"},
			repo.Arn:   {"seed": "true"},
			group.Arn:  {"seed": "true"},
		},
		domainPolicies: map[string]string{
			domain.Name: `{"Version":"2012-10-17","Statement":[]}`,
		},
		repoPolicies: map[string]string{
			codeArtifactRepoKey(domain.Name, repo.Name): `{"Version":"2012-10-17","Statement":[]}`,
		},
	}
}

func (s *codeArtifactStore) Handle(r *http.Request, payload map[string]any) codeArtifactResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := strings.TrimSpace(rawRequestPath(r))
	if path == "" {
		path = "/"
	}
	method := r.Method

	switch {
	case path == "/v1/domains" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listDomainsPayload()}
	case path == "/v1/domain" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.createDomainPayload(payload)}
	case path == "/v1/domain" && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.describeDomainPayload(r, payload)}
	case path == "/v1/domain" && method == http.MethodDelete:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.deleteDomainPayload(r, payload)}

	case path == "/v1/domain/permissions/policy" && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.getDomainPermissionsPolicyPayload(r, payload)}
	case path == "/v1/domain/permissions/policy" && method == http.MethodPut:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.putDomainPermissionsPolicyPayload(r, payload)}
	case path == "/v1/domain/permissions/policy" && method == http.MethodDelete:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.deleteDomainPermissionsPolicyPayload(r, payload)}

	case path == "/v1/repositories" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listRepositoriesPayload(payload)}
	case path == "/v1/domain/repositories" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listRepositoriesInDomainPayload(r, payload)}
	case path == "/v1/repository" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.createRepositoryPayload(payload)}
	case path == "/v1/repository" && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.describeRepositoryPayload(r, payload)}
	case path == "/v1/repository" && method == http.MethodPatch:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.updateRepositoryPayload(r, payload)}
	case path == "/v1/repository" && method == http.MethodDelete:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.deleteRepositoryPayload(r, payload)}
	case path == "/v1/repository/endpoint" && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.getRepositoryEndpointPayload(r, payload)}
	case path == "/v1/repository/permissions/policy" && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.getRepositoryPermissionsPolicyPayload(r, payload)}
	case path == "/v1/repository/permissions/policy" && method == http.MethodPut:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.putRepositoryPermissionsPolicyPayload(r, payload)}
	case path == "/v1/repository/permissions/policy" && method == http.MethodDelete:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.deleteRepositoryPermissionsPolicyPayload(r, payload)}
	case path == "/v1/repository/external-connection" && method == http.MethodPut:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.associateExternalConnectionPayload(r, payload)}
	case path == "/v1/repository/external-connection" && method == http.MethodDelete:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.disassociateExternalConnectionPayload(r, payload)}

	case path == "/v1/package-groups" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listPackageGroupsPayload(r, payload)}
	case path == "/v1/package-group" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.createPackageGroupPayload(payload)}
	case path == "/v1/package-group" && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.describePackageGroupPayload(r, payload)}
	case path == "/v1/package-group" && method == http.MethodPatch:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.updatePackageGroupPayload(r, payload)}
	case path == "/v1/package-group" && method == http.MethodDelete:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.deletePackageGroupPayload(r, payload)}
	case path == "/v1/package-group/sub-groups" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listSubPackageGroupsPayload(r, payload)}
	case path == "/v1/package-group/associated-packages" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listAssociatedPackagesPayload(r, payload)}
	case path == "/v1/package-group/allowed-repositories" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listAllowedRepositoriesForGroupPayload(r, payload)}
	case path == "/v1/package-group/association" && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.getAssociatedPackageGroupPayload(r, payload)}
	case path == "/v1/package-group/origin-configuration" && method == http.MethodPatch:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.updatePackageGroupOriginConfigurationPayload(r, payload)}

	case path == "/v1/packages" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listPackagesPayload(r, payload)}
	case path == "/v1/package" && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.describePackagePayload(r, payload)}
	case path == "/v1/package" && method == http.MethodDelete:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.deletePackagePayload(r, payload)}
	case path == "/v1/package/origin-configuration" && method == http.MethodPut:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.putPackageOriginConfigurationPayload(r, payload)}
	case path == "/v1/package/versions" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listPackageVersionsPayload(r, payload)}
	case path == "/v1/package/versions" && method == http.MethodDelete:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.deletePackageVersionsPayload(r, payload)}
	case path == "/v1/package/version" && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.describePackageVersionPayload(r, payload)}
	case path == "/v1/package/version/assets" && (method == http.MethodPost || method == http.MethodGet):
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listPackageVersionAssetsPayload(r, payload)}
	case path == "/v1/package/version/asset" && method == http.MethodGet:
		return s.getPackageVersionAssetPayload(r, payload)
	case path == "/v1/package/version/readme" && method == http.MethodGet:
		return s.getPackageVersionReadmePayload(r, payload)
	case path == "/v1/package/version/dependencies" && (method == http.MethodPost || method == http.MethodGet):
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listPackageVersionDependenciesPayload(r, payload)}
	case path == "/v1/package/version/publish" && (method == http.MethodPut || method == http.MethodPost):
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.publishPackageVersionPayload(r, payload)}
	case path == "/v1/package/versions/copy" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.copyPackageVersionsPayload(r, payload)}
	case path == "/v1/package/versions/dispose" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.disposePackageVersionsPayload(r, payload)}
	case path == "/v1/package/versions/status" && (method == http.MethodPost || method == http.MethodPatch):
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.updatePackageVersionsStatusPayload(r, payload)}

	case path == "/v1/authorization-token" && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.getAuthorizationTokenPayload(r, payload)}
	case strings.HasPrefix(path, "/v1/tags") && method == http.MethodGet:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.listTagsForResourcePayload(r, payload)}
	case strings.HasPrefix(path, "/v1/tags") && method == http.MethodPost:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.tagResourcePayload(r, payload)}
	case strings.HasPrefix(path, "/v1/tags") && method == http.MethodDelete:
		return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: s.untagResourcePayload(r, payload)}
	}

	// Compatibility fallback for currently-unmapped paths.
	return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/json", Body: map[string]any{}}
}

func (s *codeArtifactStore) createDomainPayload(payload map[string]any) map[string]any {
	name := codeArtifactDefaultString(payload, "domain", fmt.Sprintf("stackyard-domain-%06d", s.nextLocked()))
	domain := s.ensureDomainLocked(name)
	return map[string]any{"domain": codeArtifactDomainPayload(domain)}
}

func (s *codeArtifactStore) describeDomainPayload(r *http.Request, payload map[string]any) map[string]any {
	name := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	domain := s.ensureDomainLocked(name)
	return map[string]any{"domain": codeArtifactDomainPayload(domain)}
}

func (s *codeArtifactStore) deleteDomainPayload(r *http.Request, payload map[string]any) map[string]any {
	name := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	domain := s.ensureDomainLocked(name)
	delete(s.domains, domain.Name)
	for key, repo := range s.repositories {
		if repo.Domain == domain.Name {
			delete(s.repositories, key)
		}
	}
	return map[string]any{"domain": codeArtifactDomainPayload(domain)}
}

func (s *codeArtifactStore) listDomainsPayload() map[string]any {
	names := make([]string, 0, len(s.domains))
	for name := range s.domains {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]map[string]any, 0, len(names))
	for _, name := range names {
		items = append(items, map[string]any{
			"name":           s.domains[name].Name,
			"owner":          s.domains[name].Owner,
			"arn":            s.domains[name].Arn,
			"status":         s.domains[name].Status,
			"createdTime":    s.domains[name].CreatedTime,
			"assetSizeBytes": int64(0),
		})
	}
	return map[string]any{"domains": items}
}

func (s *codeArtifactStore) getDomainPermissionsPolicyPayload(r *http.Request, payload map[string]any) map[string]any {
	name := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	domain := s.ensureDomainLocked(name)
	policy := s.domainPolicies[domain.Name]
	if strings.TrimSpace(policy) == "" {
		policy = `{"Version":"2012-10-17","Statement":[]}`
	}
	return map[string]any{
		"policy": map[string]any{
			"resourceArn": domain.Arn,
			"revision":    "1",
			"document":    policy,
		},
	}
}

func (s *codeArtifactStore) putDomainPermissionsPolicyPayload(r *http.Request, payload map[string]any) map[string]any {
	name := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	domain := s.ensureDomainLocked(name)
	policy := codeArtifactDefaultString(payload, "policyDocument", `{"Version":"2012-10-17","Statement":[]}`)
	s.domainPolicies[domain.Name] = policy
	return s.getDomainPermissionsPolicyPayload(r, payload)
}

func (s *codeArtifactStore) deleteDomainPermissionsPolicyPayload(r *http.Request, payload map[string]any) map[string]any {
	name := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	domain := s.ensureDomainLocked(name)
	delete(s.domainPolicies, domain.Name)
	return map[string]any{
		"policy": map[string]any{
			"resourceArn": domain.Arn,
			"revision":    "1",
			"document":    `{"Version":"2012-10-17","Statement":[]}`,
		},
	}
}

func (s *codeArtifactStore) createRepositoryPayload(payload map[string]any) map[string]any {
	domainName := codeArtifactDefaultString(payload, "domain", "stackyard-domain")
	repoName := codeArtifactDefaultString(payload, "repository", fmt.Sprintf("stackyard-repo-%06d", s.nextLocked()))
	repo := s.ensureRepositoryLocked(domainName, repoName)
	if desc := codeArtifactString(payload["description"]); desc != "" {
		repo.Description = desc
	}
	return map[string]any{"repository": codeArtifactRepositoryPayload(repo)}
}

func (s *codeArtifactStore) describeRepositoryPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	repo := s.ensureRepositoryLocked(domainName, repoName)
	return map[string]any{"repository": codeArtifactRepositoryPayload(repo)}
}

func (s *codeArtifactStore) updateRepositoryPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	repo := s.ensureRepositoryLocked(domainName, repoName)
	if desc := codeArtifactString(payload["description"]); desc != "" {
		repo.Description = desc
	}
	repo.UpdatedTime = time.Now().UTC()
	return map[string]any{"repository": codeArtifactRepositoryPayload(repo)}
}

func (s *codeArtifactStore) deleteRepositoryPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	repo := s.ensureRepositoryLocked(domainName, repoName)
	delete(s.repositories, codeArtifactRepoKey(repo.Domain, repo.Name))
	return map[string]any{"repository": codeArtifactRepositoryPayload(repo)}
}

func (s *codeArtifactStore) listRepositoriesPayload(payload map[string]any) map[string]any {
	domainFilter := codeArtifactString(payload["domain"])
	items := make([]map[string]any, 0, len(s.repositories))
	keys := make([]string, 0, len(s.repositories))
	for key := range s.repositories {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		repo := s.repositories[key]
		if domainFilter != "" && repo.Domain != domainFilter {
			continue
		}
		items = append(items, map[string]any{
			"name":                 repo.Name,
			"administratorAccount": repo.DomainOwner,
			"domainName":           repo.Domain,
			"domainOwner":          repo.DomainOwner,
			"arn":                  repo.Arn,
			"description":          repo.Description,
			"createdTime":          repo.CreatedTime,
			"upstreams":            []any{},
			"externalConnections":  codeArtifactExternalConnectionSummaries(repo.ExternalConnections),
		})
	}
	return map[string]any{"repositories": items}
}

func (s *codeArtifactStore) listRepositoriesInDomainPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	base := s.listRepositoriesPayload(map[string]any{"domain": domainName})
	base["domainName"] = domainName
	return base
}

func (s *codeArtifactStore) getRepositoryEndpointPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	format := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "format", ""), strings.TrimSpace(r.URL.Query().Get("format")), "npm")
	s.ensureRepositoryLocked(domainName, repoName)
	endpoint := fmt.Sprintf("https://%s-%s.d.codeartifact.us-east-1.amazonaws.com/%s/%s/", repoName, domainName, format, repoName)
	return map[string]any{"repositoryEndpoint": endpoint}
}

func (s *codeArtifactStore) getRepositoryPermissionsPolicyPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	repo := s.ensureRepositoryLocked(domainName, repoName)
	key := codeArtifactRepoKey(repo.Domain, repo.Name)
	policy := s.repoPolicies[key]
	if strings.TrimSpace(policy) == "" {
		policy = `{"Version":"2012-10-17","Statement":[]}`
	}
	return map[string]any{
		"policy": map[string]any{
			"resourceArn": repo.Arn,
			"revision":    "1",
			"document":    policy,
		},
	}
}

func (s *codeArtifactStore) putRepositoryPermissionsPolicyPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	repo := s.ensureRepositoryLocked(domainName, repoName)
	key := codeArtifactRepoKey(repo.Domain, repo.Name)
	s.repoPolicies[key] = codeArtifactDefaultString(payload, "policyDocument", `{"Version":"2012-10-17","Statement":[]}`)
	return s.getRepositoryPermissionsPolicyPayload(r, payload)
}

func (s *codeArtifactStore) deleteRepositoryPermissionsPolicyPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	repo := s.ensureRepositoryLocked(domainName, repoName)
	key := codeArtifactRepoKey(repo.Domain, repo.Name)
	delete(s.repoPolicies, key)
	return map[string]any{
		"policy": map[string]any{
			"resourceArn": repo.Arn,
			"revision":    "1",
			"document":    `{"Version":"2012-10-17","Statement":[]}`,
		},
	}
}

func (s *codeArtifactStore) associateExternalConnectionPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	externalConnection := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "externalConnection", ""), strings.TrimSpace(r.URL.Query().Get("externalConnection")), "public:npmjs")
	repo := s.ensureRepositoryLocked(domainName, repoName)
	repo.ExternalConnections = codeArtifactAddUnique(repo.ExternalConnections, externalConnection)
	repo.UpdatedTime = time.Now().UTC()
	return map[string]any{"repository": codeArtifactRepositoryPayload(repo)}
}

func (s *codeArtifactStore) disassociateExternalConnectionPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	externalConnection := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "externalConnection", ""), strings.TrimSpace(r.URL.Query().Get("externalConnection")), "public:npmjs")
	repo := s.ensureRepositoryLocked(domainName, repoName)
	filtered := make([]string, 0, len(repo.ExternalConnections))
	for _, item := range repo.ExternalConnections {
		if item != externalConnection {
			filtered = append(filtered, item)
		}
	}
	repo.ExternalConnections = filtered
	repo.UpdatedTime = time.Now().UTC()
	return map[string]any{"repository": codeArtifactRepositoryPayload(repo)}
}

func (s *codeArtifactStore) createPackageGroupPayload(payload map[string]any) map[string]any {
	domainName := codeArtifactDefaultString(payload, "domain", "stackyard-domain")
	pattern := codeArtifactDefaultString(payload, "packageGroup", "/stackyard/*")
	group := s.ensurePackageGroupLocked(domainName, pattern)
	if description := codeArtifactString(payload["description"]); description != "" {
		group.Description = description
	}
	if contact := codeArtifactString(payload["contactInfo"]); contact != "" {
		group.ContactInfo = contact
	}
	group.UpdatedTime = time.Now().UTC()
	return map[string]any{"packageGroup": codeArtifactPackageGroupPayload(group)}
}

func (s *codeArtifactStore) describePackageGroupPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	pattern := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageGroup", ""), strings.TrimSpace(r.URL.Query().Get("packageGroup")), "/stackyard/*")
	group := s.ensurePackageGroupLocked(domainName, pattern)
	return map[string]any{"packageGroup": codeArtifactPackageGroupPayload(group)}
}

func (s *codeArtifactStore) updatePackageGroupPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	pattern := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageGroup", ""), strings.TrimSpace(r.URL.Query().Get("packageGroup")), "/stackyard/*")
	group := s.ensurePackageGroupLocked(domainName, pattern)
	if description := codeArtifactString(payload["description"]); description != "" {
		group.Description = description
	}
	if contact := codeArtifactString(payload["contactInfo"]); contact != "" {
		group.ContactInfo = contact
	}
	group.UpdatedTime = time.Now().UTC()
	return map[string]any{"packageGroup": codeArtifactPackageGroupPayload(group)}
}

func (s *codeArtifactStore) deletePackageGroupPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	pattern := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageGroup", ""), strings.TrimSpace(r.URL.Query().Get("packageGroup")), "/stackyard/*")
	group := s.ensurePackageGroupLocked(domainName, pattern)
	delete(s.packageGroups, codeArtifactPackageGroupKey(group.Domain, group.Pattern))
	return map[string]any{"packageGroup": codeArtifactPackageGroupPayload(group)}
}

func (s *codeArtifactStore) listPackageGroupsPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "")
	keys := make([]string, 0, len(s.packageGroups))
	for key := range s.packageGroups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		group := s.packageGroups[key]
		if domainName != "" && group.Domain != domainName {
			continue
		}
		items = append(items, map[string]any{
			"arn":         group.Arn,
			"pattern":     group.Pattern,
			"domainName":  group.Domain,
			"domainOwner": group.DomainOwner,
			"createdTime": group.UpdatedTime,
		})
	}
	return map[string]any{"packageGroups": items}
}

func (s *codeArtifactStore) listSubPackageGroupsPayload(r *http.Request, payload map[string]any) map[string]any {
	out := s.listPackageGroupsPayload(r, payload)
	out["subPackageGroups"] = out["packageGroups"]
	delete(out, "packageGroups")
	return out
}

func (s *codeArtifactStore) listAssociatedPackagesPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	items := make([]map[string]any, 0, len(s.packages))
	for _, pkg := range s.packages {
		if pkg.Domain != domainName {
			continue
		}
		items = append(items, map[string]any{
			"format":                pkg.Format,
			"namespace":             pkg.Namespace,
			"package":               pkg.Name,
			"originRestrictionType": "INHERIT",
		})
	}
	sort.Slice(items, func(i, j int) bool {
		left := codeArtifactString(items[i]["package"])
		right := codeArtifactString(items[j]["package"])
		return left < right
	})
	return map[string]any{"associatedPackages": items}
}

func (s *codeArtifactStore) listAllowedRepositoriesForGroupPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	items := make([]map[string]any, 0, len(s.repositories))
	for _, repo := range s.repositories {
		if repo.Domain != domainName {
			continue
		}
		items = append(items, map[string]any{
			"repositoryName":        repo.Name,
			"originRestrictionType": "ALLOW",
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return codeArtifactString(items[i]["repositoryName"]) < codeArtifactString(items[j]["repositoryName"])
	})
	return map[string]any{"allowedRepositories": items}
}

func (s *codeArtifactStore) getAssociatedPackageGroupPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	pattern := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageGroup", ""), strings.TrimSpace(r.URL.Query().Get("packageGroup")), "/stackyard/*")
	group := s.ensurePackageGroupLocked(domainName, pattern)
	return map[string]any{"packageGroup": map[string]any{"arn": group.Arn, "pattern": group.Pattern, "domainName": group.Domain, "domainOwner": group.DomainOwner}}
}

func (s *codeArtifactStore) updatePackageGroupOriginConfigurationPayload(r *http.Request, payload map[string]any) map[string]any {
	return s.describePackageGroupPayload(r, payload)
}

func (s *codeArtifactStore) listPackagesPayload(r *http.Request, payload map[string]any) map[string]any {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	format := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "format", ""), strings.TrimSpace(r.URL.Query().Get("format")), "npm")
	items := make([]map[string]any, 0, len(s.packages))
	for _, pkg := range s.packages {
		if pkg.Domain != domainName || pkg.Repository != repoName || pkg.Format != format {
			continue
		}
		items = append(items, map[string]any{"format": pkg.Format, "namespace": pkg.Namespace, "package": pkg.Name, "originConfiguration": map[string]any{}})
	}
	sort.Slice(items, func(i, j int) bool {
		return codeArtifactString(items[i]["package"]) < codeArtifactString(items[j]["package"])
	})
	return map[string]any{"packages": items}
}

func (s *codeArtifactStore) describePackagePayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	return map[string]any{
		"package": map[string]any{
			"format":              pkg.Format,
			"namespace":           pkg.Namespace,
			"package":             pkg.Name,
			"originConfiguration": map[string]any{},
		},
	}
}

func (s *codeArtifactStore) deletePackagePayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	delete(s.packages, codeArtifactPackageKey(pkg.Domain, pkg.Repository, pkg.Format, pkg.Namespace, pkg.Name))
	return map[string]any{"deletedPackage": map[string]any{"format": pkg.Format, "namespace": pkg.Namespace, "package": pkg.Name}}
}

func (s *codeArtifactStore) putPackageOriginConfigurationPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	return map[string]any{"package": map[string]any{"format": pkg.Format, "namespace": pkg.Namespace, "package": pkg.Name, "originConfiguration": map[string]any{}}}
}

func (s *codeArtifactStore) listPackageVersionsPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	versions := make([]string, 0, len(pkg.Versions))
	for v := range pkg.Versions {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	items := make([]map[string]any, 0, len(versions))
	for _, version := range versions {
		entry := pkg.Versions[version]
		items = append(items, map[string]any{
			"version":       entry.Version,
			"revision":      entry.Revision,
			"status":        entry.Status,
			"publishedTime": entry.PublishedAt,
			"origin":        map[string]any{},
		})
	}
	return map[string]any{"defaultDisplayVersion": firstOrEmpty(versions), "versions": items}
}

func (s *codeArtifactStore) describePackageVersionPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	version := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageVersion", ""), strings.TrimSpace(r.URL.Query().Get("packageVersion")), "1.0.0")
	entry := s.ensurePackageVersionLocked(pkg, version)
	return map[string]any{
		"format":               pkg.Format,
		"namespace":            pkg.Namespace,
		"package":              pkg.Name,
		"displayName":          pkg.Name,
		"version":              entry.Version,
		"summary":              "stackyard package version",
		"homePage":             "https://example.com",
		"sourceCodeRepository": "https://example.com/source",
		"publishedTime":        entry.PublishedAt,
		"licenses":             []any{},
		"revision":             entry.Revision,
		"status":               entry.Status,
		"origin":               map[string]any{},
	}
}

func (s *codeArtifactStore) listPackageVersionAssetsPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	version := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageVersion", ""), strings.TrimSpace(r.URL.Query().Get("packageVersion")), "1.0.0")
	entry := s.ensurePackageVersionLocked(pkg, version)
	assetNames := make([]string, 0, len(entry.Assets))
	for name := range entry.Assets {
		assetNames = append(assetNames, name)
	}
	sort.Strings(assetNames)
	assets := make([]map[string]any, 0, len(assetNames))
	for _, name := range assetNames {
		assets = append(assets, map[string]any{"name": name, "size": len(entry.Assets[name]), "hashes": map[string]any{"SHA-256": "stackyard"}})
	}
	return map[string]any{"format": pkg.Format, "namespace": pkg.Namespace, "package": pkg.Name, "version": entry.Version, "versionRevision": entry.Revision, "assets": assets}
}

func (s *codeArtifactStore) getPackageVersionAssetPayload(r *http.Request, payload map[string]any) codeArtifactResponse {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	version := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageVersion", ""), strings.TrimSpace(r.URL.Query().Get("packageVersion")), "1.0.0")
	asset := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "asset", ""), strings.TrimSpace(r.URL.Query().Get("asset")), "package.tgz")
	entry := s.ensurePackageVersionLocked(pkg, version)
	content := entry.Assets[asset]
	if len(content) == 0 {
		content = []byte("stackyard")
	}
	return codeArtifactResponse{Status: http.StatusOK, ContentType: "application/octet-stream", RawBody: content}
}

func (s *codeArtifactStore) getPackageVersionReadmePayload(r *http.Request, payload map[string]any) codeArtifactResponse {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	version := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageVersion", ""), strings.TrimSpace(r.URL.Query().Get("packageVersion")), "1.0.0")
	entry := s.ensurePackageVersionLocked(pkg, version)
	readme := entry.Readme
	if strings.TrimSpace(readme) == "" {
		readme = "# stackyard"
	}
	return codeArtifactResponse{Status: http.StatusOK, ContentType: "text/plain", RawBody: []byte(readme)}
}

func (s *codeArtifactStore) listPackageVersionDependenciesPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	version := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageVersion", ""), strings.TrimSpace(r.URL.Query().Get("packageVersion")), "1.0.0")
	entry := s.ensurePackageVersionLocked(pkg, version)
	deps := make([]map[string]any, 0, len(entry.Dependencies))
	for _, dep := range entry.Dependencies {
		deps = append(deps, map[string]any{"namespace": pkg.Namespace, "package": dep, "versionRequirement": "*", "dependencyType": "runtime"})
	}
	return map[string]any{"version": entry.Version, "format": pkg.Format, "namespace": pkg.Namespace, "package": pkg.Name, "dependencies": deps}
}

func (s *codeArtifactStore) publishPackageVersionPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	version := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "packageVersion", ""), strings.TrimSpace(r.URL.Query().Get("packageVersion")), fmt.Sprintf("1.0.%d", s.nextLocked()))
	entry := s.ensurePackageVersionLocked(pkg, version)
	entry.Status = "Published"
	entry.PublishedAt = time.Now().UTC()
	if entry.Revision == "" {
		entry.Revision = fmt.Sprintf("rev-%012d", s.nextLocked())
	}
	assetName := codeArtifactDefaultString(payload, "assetName", "package.tgz")
	entry.Assets[assetName] = []byte("stackyard")
	return map[string]any{"format": pkg.Format, "namespace": pkg.Namespace, "package": pkg.Name, "version": entry.Version, "versionRevision": entry.Revision, "status": entry.Status, "asset": map[string]any{"name": assetName, "size": len(entry.Assets[assetName]), "hashes": map[string]any{"SHA-256": "stackyard"}}}
}

func (s *codeArtifactStore) copyPackageVersionsPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	versions := codeArtifactStringSlice(payload["versions"])
	if len(versions) == 0 {
		versions = []string{"1.0.0"}
	}
	successful := make(map[string]any, len(versions))
	failed := map[string]any{}
	for _, version := range versions {
		entry := s.ensurePackageVersionLocked(pkg, version)
		successful[version] = map[string]any{"revision": entry.Revision, "status": entry.Status}
	}
	return map[string]any{"successfulVersions": successful, "failedVersions": failed}
}

func (s *codeArtifactStore) deletePackageVersionsPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	versions := codeArtifactStringSlice(payload["versions"])
	if len(versions) == 0 {
		versions = []string{"1.0.0"}
	}
	successful := make(map[string]any, len(versions))
	failed := map[string]any{}
	for _, version := range versions {
		entry := s.ensurePackageVersionLocked(pkg, version)
		delete(pkg.Versions, version)
		successful[version] = map[string]any{"revision": entry.Revision, "status": "Deleted"}
	}
	return map[string]any{"successfulVersions": successful, "failedVersions": failed}
}

func (s *codeArtifactStore) disposePackageVersionsPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	versions := codeArtifactStringSlice(payload["versions"])
	if len(versions) == 0 {
		versions = []string{"1.0.0"}
	}
	successful := make(map[string]any, len(versions))
	failed := map[string]any{}
	for _, version := range versions {
		entry := s.ensurePackageVersionLocked(pkg, version)
		entry.Status = "Disposed"
		successful[version] = map[string]any{"revision": entry.Revision, "status": entry.Status}
	}
	return map[string]any{"successfulVersions": successful, "failedVersions": failed}
}

func (s *codeArtifactStore) updatePackageVersionsStatusPayload(r *http.Request, payload map[string]any) map[string]any {
	pkg := s.ensurePackageFromRequestLocked(r, payload)
	versions := codeArtifactStringSlice(payload["versions"])
	if len(versions) == 0 {
		versions = []string{"1.0.0"}
	}
	targetStatus := codeArtifactDefaultString(payload, "targetStatus", "Published")
	successful := make(map[string]any, len(versions))
	failed := map[string]any{}
	for _, version := range versions {
		entry := s.ensurePackageVersionLocked(pkg, version)
		entry.Status = targetStatus
		successful[version] = map[string]any{"revision": entry.Revision, "status": entry.Status}
	}
	return map[string]any{"successfulVersions": successful, "failedVersions": failed}
}

func (s *codeArtifactStore) getAuthorizationTokenPayload(r *http.Request, payload map[string]any) map[string]any {
	domain := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	s.ensureDomainLocked(domain)
	return map[string]any{
		"authorizationToken": "stackyard-token",
		"expiration":         time.Now().UTC().Add(12 * time.Hour),
	}
}

func (s *codeArtifactStore) listTagsForResourcePayload(r *http.Request, payload map[string]any) map[string]any {
	arn := codeArtifactResourceARN(r, payload)
	tags := codeArtifactSortedTagsPayload(s.resourceTags[arn])
	return map[string]any{"tags": tags}
}

func (s *codeArtifactStore) tagResourcePayload(r *http.Request, payload map[string]any) map[string]any {
	arn := codeArtifactResourceARN(r, payload)
	tags := codeArtifactExtractTags(payload["tags"])
	existing := s.ensureTagMapLocked(arn)
	for k, v := range tags {
		existing[k] = v
	}
	return map[string]any{}
}

func (s *codeArtifactStore) untagResourcePayload(r *http.Request, payload map[string]any) map[string]any {
	arn := codeArtifactResourceARN(r, payload)
	tagKeys := codeArtifactStringSlice(payload["tagKeys"])
	existing := s.ensureTagMapLocked(arn)
	for _, key := range tagKeys {
		delete(existing, key)
	}
	return map[string]any{}
}

func (s *codeArtifactStore) ensureDomainLocked(name string) *codeArtifactDomain {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-domain"
	}
	if existing, ok := s.domains[name]; ok {
		return existing
	}
	now := time.Now().UTC()
	domain := &codeArtifactDomain{
		Name:        name,
		Owner:       "123456789012",
		Arn:         codeArtifactDomainARN(name),
		Status:      "Active",
		CreatedTime: now,
		UpdatedTime: now,
	}
	s.domains[name] = domain
	return domain
}

func (s *codeArtifactStore) ensureRepositoryLocked(domainName, repoName string) *codeArtifactRepository {
	domain := s.ensureDomainLocked(domainName)
	repoName = strings.TrimSpace(repoName)
	if repoName == "" {
		repoName = "stackyard-repo"
	}
	key := codeArtifactRepoKey(domain.Name, repoName)
	if existing, ok := s.repositories[key]; ok {
		return existing
	}
	now := time.Now().UTC()
	repo := &codeArtifactRepository{
		Domain:              domain.Name,
		DomainOwner:         domain.Owner,
		Name:                repoName,
		Description:         "",
		Arn:                 codeArtifactRepositoryARN(domain.Name, repoName),
		ExternalConnections: []string{},
		CreatedTime:         now,
		UpdatedTime:         now,
	}
	s.repositories[key] = repo
	return repo
}

func (s *codeArtifactStore) ensurePackageGroupLocked(domainName, pattern string) *codeArtifactPackageGroup {
	domain := s.ensureDomainLocked(domainName)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		pattern = "/stackyard/*"
	}
	key := codeArtifactPackageGroupKey(domain.Name, pattern)
	if existing, ok := s.packageGroups[key]; ok {
		return existing
	}
	group := &codeArtifactPackageGroup{
		Domain:      domain.Name,
		DomainOwner: domain.Owner,
		Pattern:     pattern,
		Arn:         codeArtifactPackageGroupARN(domain.Name, pattern),
		Description: "",
		ContactInfo: "",
		UpdatedTime: time.Now().UTC(),
	}
	s.packageGroups[key] = group
	return group
}

func (s *codeArtifactStore) ensurePackageFromRequestLocked(r *http.Request, payload map[string]any) *codeArtifactPackage {
	domainName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "domain", ""), strings.TrimSpace(r.URL.Query().Get("domain")), "stackyard-domain")
	repoName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "repository", ""), strings.TrimSpace(r.URL.Query().Get("repository")), "stackyard-repo")
	format := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "format", ""), strings.TrimSpace(r.URL.Query().Get("format")), "npm")
	namespace := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "namespace", ""), strings.TrimSpace(r.URL.Query().Get("namespace")), "stackyard")
	packageName := codeArtifactFirstNonEmpty(codeArtifactDefaultString(payload, "package", ""), strings.TrimSpace(r.URL.Query().Get("package")), "seed-package")

	s.ensureRepositoryLocked(domainName, repoName)
	key := codeArtifactPackageKey(domainName, repoName, format, namespace, packageName)
	if existing, ok := s.packages[key]; ok {
		return existing
	}
	pkg := &codeArtifactPackage{
		Domain:      domainName,
		DomainOwner: "123456789012",
		Repository:  repoName,
		Format:      format,
		Namespace:   namespace,
		Name:        packageName,
		Versions: map[string]*codeArtifactPackageVersion{
			"1.0.0": {
				Version:      "1.0.0",
				Status:       "Published",
				Revision:     fmt.Sprintf("rev-%012d", s.nextLocked()),
				PublishedAt:  time.Now().UTC(),
				Assets:       map[string][]byte{"package.tgz": []byte("stackyard")},
				Readme:       "# stackyard",
				Dependencies: []string{},
			},
		},
	}
	s.packages[key] = pkg
	return pkg
}

func (s *codeArtifactStore) ensurePackageVersionLocked(pkg *codeArtifactPackage, version string) *codeArtifactPackageVersion {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "1.0.0"
	}
	if existing, ok := pkg.Versions[version]; ok {
		return existing
	}
	entry := &codeArtifactPackageVersion{
		Version:      version,
		Status:       "Published",
		Revision:     fmt.Sprintf("rev-%012d", s.nextLocked()),
		PublishedAt:  time.Now().UTC(),
		Assets:       map[string][]byte{"package.tgz": []byte("stackyard")},
		Readme:       "# stackyard",
		Dependencies: []string{},
	}
	pkg.Versions[version] = entry
	return entry
}

func (s *codeArtifactStore) ensureTagMapLocked(arn string) map[string]string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = codeArtifactDomainARN("stackyard-domain")
	}
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = map[string]string{}
	}
	return s.resourceTags[arn]
}

func (s *codeArtifactStore) nextLocked() int64 {
	next := s.nextID
	s.nextID++
	return next
}

func codeArtifactDefaultString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	if value := strings.TrimSpace(codeArtifactString(payload[key])); value != "" {
		return value
	}
	return fallback
}

func codeArtifactString(value any) string {
	if value == nil {
		return ""
	}
	if asString, ok := value.(string); ok {
		return asString
	}
	return ""
}

func codeArtifactStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(codeArtifactString(item)); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return nil
	}
}

func codeArtifactExtractTags(value any) map[string]string {
	list, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]map[string]any); ok {
			list = make([]any, 0, len(typed))
			for _, item := range typed {
				list = append(list, item)
			}
		} else {
			return map[string]string{}
		}
	}
	out := map[string]string{}
	for _, item := range list {
		tag, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := strings.TrimSpace(codeArtifactString(tag["key"]))
		if key == "" {
			key = strings.TrimSpace(codeArtifactString(tag["Key"]))
		}
		if key == "" {
			continue
		}
		value := strings.TrimSpace(codeArtifactString(tag["value"]))
		if value == "" {
			value = strings.TrimSpace(codeArtifactString(tag["Value"]))
		}
		out[key] = value
	}
	return out
}

func codeArtifactSortedTagsPayload(tags map[string]string) []map[string]string {
	if len(tags) == 0 {
		return []map[string]string{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]string{"key": key, "value": tags[key]})
	}
	return out
}

func codeArtifactResourceARN(r *http.Request, payload map[string]any) string {
	if value := strings.TrimSpace(codeArtifactString(payload["resourceArn"])); value != "" {
		return value
	}
	path := strings.TrimSpace(rawRequestPath(r))
	if strings.HasPrefix(path, "/v1/tags/") && len(path) > len("/v1/tags/") {
		encoded := strings.TrimPrefix(path, "/v1/tags/")
		decoded, err := neturl.PathUnescape(encoded)
		if err == nil && strings.TrimSpace(decoded) != "" {
			return decoded
		}
	}
	return codeArtifactDomainARN("stackyard-domain")
}

func codeArtifactAddUnique(values []string, candidate string) []string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return values
	}
	for _, existing := range values {
		if existing == candidate {
			return values
		}
	}
	return append(values, candidate)
}

func codeArtifactExternalConnectionSummaries(values []string) []map[string]any {
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		out = append(out, map[string]any{"externalConnectionName": value, "packageFormat": "npm", "status": "Available"})
	}
	return out
}

func codeArtifactDomainPayload(domain *codeArtifactDomain) map[string]any {
	return map[string]any{
		"name":            domain.Name,
		"owner":           domain.Owner,
		"arn":             domain.Arn,
		"status":          domain.Status,
		"createdTime":     domain.CreatedTime,
		"encryptionKey":   "alias/aws/codeartifact",
		"repositoryCount": 0,
		"assetSizeBytes":  int64(0),
	}
}

func codeArtifactRepositoryPayload(repo *codeArtifactRepository) map[string]any {
	return map[string]any{
		"name":                 repo.Name,
		"administratorAccount": repo.DomainOwner,
		"domainName":           repo.Domain,
		"domainOwner":          repo.DomainOwner,
		"arn":                  repo.Arn,
		"description":          repo.Description,
		"upstreams":            []any{},
		"externalConnections":  codeArtifactExternalConnectionSummaries(repo.ExternalConnections),
		"createdTime":          repo.CreatedTime,
	}
}

func codeArtifactPackageGroupPayload(group *codeArtifactPackageGroup) map[string]any {
	return map[string]any{
		"arn":                 group.Arn,
		"pattern":             group.Pattern,
		"domainName":          group.Domain,
		"domainOwner":         group.DomainOwner,
		"createdTime":         group.UpdatedTime,
		"description":         group.Description,
		"contactInfo":         group.ContactInfo,
		"originConfiguration": map[string]any{},
	}
}

func codeArtifactRepoKey(domainName, repoName string) string {
	return strings.TrimSpace(domainName) + "|" + strings.TrimSpace(repoName)
}

func codeArtifactPackageGroupKey(domainName, pattern string) string {
	return strings.TrimSpace(domainName) + "|" + strings.TrimSpace(pattern)
}

func codeArtifactPackageKey(domainName, repoName, format, namespace, packageName string) string {
	return strings.TrimSpace(domainName) + "|" + strings.TrimSpace(repoName) + "|" + strings.TrimSpace(format) + "|" + strings.TrimSpace(namespace) + "|" + strings.TrimSpace(packageName)
}

func codeArtifactDomainARN(domainName string) string {
	domainName = strings.TrimSpace(domainName)
	if domainName == "" {
		domainName = "stackyard-domain"
	}
	return "arn:aws:codeartifact:us-east-1:123456789012:domain/" + domainName
}

func codeArtifactRepositoryARN(domainName, repoName string) string {
	domainName = strings.TrimSpace(domainName)
	if domainName == "" {
		domainName = "stackyard-domain"
	}
	repoName = strings.TrimSpace(repoName)
	if repoName == "" {
		repoName = "stackyard-repo"
	}
	return "arn:aws:codeartifact:us-east-1:123456789012:repository/" + domainName + "/" + repoName
}

func codeArtifactPackageGroupARN(domainName, pattern string) string {
	domainName = strings.TrimSpace(domainName)
	if domainName == "" {
		domainName = "stackyard-domain"
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		pattern = "/stackyard/*"
	}
	return "arn:aws:codeartifact:us-east-1:123456789012:package-group/" + domainName + "/" + strings.TrimPrefix(pattern, "/")
}

func codeArtifactFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
