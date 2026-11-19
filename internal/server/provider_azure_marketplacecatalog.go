package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Azure Marketplace Catalog, Resource Manager API (Microsoft.Marketplace):
// the product catalog at tenant, subscription, billing account and billing
// profile scope, the tenant-level search, and the subscription's Discovery
// API keys. Products are a fixed seed; keys are stored.

const (
	azureMarketplaceCatalogNamespace = "Microsoft.Marketplace"
	azureMarketplaceKeyType          = azureMarketplaceCatalogNamespace + "/keys"
	azureMarketplaceTenantHeader     = "x-ms-client-tenant-id"
)

// azureMarketplaceKeyAlias is Learn's alias rule: alphanumeric characters
// separated by dashes.
var azureMarketplaceKeyAlias = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)*$`)

// azureMarketplaceScope is where a Microsoft.Marketplace request is made.
type azureMarketplaceScope struct {
	kind string // "tenant", "subscription" or "billing"
	sub  string
}

// handleAzureMarketplaceCatalogRouter serves the Marketplace Catalog
// Resource Manager operations and leaves other Microsoft.Marketplace paths
// to later routers.
func (s *Server) handleAzureMarketplaceCatalogRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	i := len(seg) - 2
	for i >= 0 && !(strings.EqualFold(seg[i], "providers") && strings.EqualFold(seg[i+1], azureMarketplaceCatalogNamespace)) {
		i--
	}
	if i < 0 {
		return false
	}
	scope, ok := azureMarketplaceScopeOf(seg[:i])
	rest := seg[i+2:]
	if !ok || len(rest) == 0 {
		return false
	}
	switch {
	case strings.EqualFold(rest[0], "products") && len(rest) <= 2:
		if !azureMarketplaceMethod(w, r, http.MethodGet) {
			return true
		}
		if len(rest) == 1 {
			s.listAzureMarketplaceProducts(w, scope, r.URL.Query())
		} else {
			getAzureMarketplaceProduct(w, scope, rest[1], r.URL.Query())
		}
	case strings.EqualFold(rest[0], "search") && len(rest) == 1 && scope.kind == "tenant":
		if azureMarketplaceMethod(w, r, http.MethodPost) && azureMarketplaceTenant(w, r) {
			if payload, ok := azureARMReadBody(w, r); ok {
				status, body := searchAzureMarketplace(payload)
				respondAzureARMAction(w, status, body)
			}
		}
	case strings.EqualFold(rest[0], "keys") && scope.kind == "subscription" && len(rest) == 1:
		if azureMarketplaceMethod(w, r, http.MethodGet) && azureMarketplaceTenant(w, r) {
			respondJSON(w, http.StatusOK, map[string]any{"value": s.azureMarketplaceKeys(scope.sub), "nextLink": nil})
		}
	case strings.EqualFold(rest[0], "keys") && scope.kind == "subscription" && len(rest) == 3 &&
		(strings.EqualFold(rest[2], "create") || strings.EqualFold(rest[2], "revoke")):
		if azureMarketplaceMethod(w, r, http.MethodPost) && azureMarketplaceTenant(w, r) {
			if strings.EqualFold(rest[2], "create") {
				s.createAzureMarketplaceKey(w, scope.sub, rest[1])
			} else {
				s.revokeAzureMarketplaceKey(w, scope.sub, rest[1])
			}
		}
	default:
		return false
	}
	return true
}

// azureMarketplaceScopeOf parses the segments before
// providers/Microsoft.Marketplace.
func azureMarketplaceScopeOf(seg []string) (azureMarketplaceScope, bool) {
	switch {
	case len(seg) == 0:
		return azureMarketplaceScope{kind: "tenant"}, true
	case len(seg) == 2 && strings.EqualFold(seg[0], "subscriptions"):
		return azureMarketplaceScope{kind: "subscription", sub: seg[1]}, true
	case (len(seg) == 4 || len(seg) == 6 && strings.EqualFold(seg[4], "billingProfiles")) &&
		strings.EqualFold(seg[0], "providers") && strings.EqualFold(seg[1], "Microsoft.Billing") && strings.EqualFold(seg[2], "billingAccounts"):
		return azureMarketplaceScope{kind: "billing"}, true
	}
	return azureMarketplaceScope{}, false
}

// azureMarketplaceMethod checks the method and the api-version.
func azureMarketplaceMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return false
	}
	return true
}

// azureMarketplaceTenant requires the tenant header Learn marks required
// for search and the key operations.
func azureMarketplaceTenant(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get(azureMarketplaceTenantHeader)) == "" {
		azureARMError(w, http.StatusBadRequest, "BadRequest", "The "+azureMarketplaceTenantHeader+" header is missing.")
		return false
	}
	return true
}

func (s *Server) listAzureMarketplaceProducts(w http.ResponseWriter, scope azureMarketplaceScope, query url.Values) {
	q, msg := azureMarketplaceListQuery(scope, query)
	if msg != "" {
		azureARMError(w, http.StatusBadRequest, "BadRequest", msg)
		return
	}
	value := []any{}
	for _, product := range azureMarketplaceProducts(q) {
		value = append(value, q.project(product))
	}
	// ponytail: the seed catalog fits one page, so nextLink is always null.
	respondJSON(w, http.StatusOK, map[string]any{"value": value, "nextLink": nil})
}

func getAzureMarketplaceProduct(w http.ResponseWriter, scope azureMarketplaceScope, id string, query url.Values) {
	opts, msg := azureMarketplaceGetQuery(scope, query)
	if msg != "" {
		azureARMError(w, http.StatusBadRequest, "BadRequest", msg)
		return
	}
	for _, product := range azureMarketplaceCatalog {
		if !strings.EqualFold(azureARMString(product["uniqueProductId"]), id) {
			continue
		}
		if opts.market != "" && !azureMarketplaceHas(product["markets"], opts.market) {
			break
		}
		out := azureARMMerge(product, nil)
		delete(out, "markets")
		plans := []any{}
		for _, plan := range azureARMSlice(product["plans"]) {
			if opts.keepPlan(azureARMObject(plan)) {
				plans = append(plans, plan)
			}
		}
		out["plans"] = plans
		if !opts.templates {
			delete(out, "serviceInstructionTemplates")
		}
		if opts.pricingAudience != "" {
			out["pricingAudience"] = opts.pricingAudience
		}
		respondJSON(w, http.StatusOK, out)
		return
	}
	azureARMError(w, http.StatusNotFound, "NotFound", "Product '"+id+"' was not found.")
}

// azureMarketplaceKeys lists the subscription's active keys by alias.
func (s *Server) azureMarketplaceKeys(sub string) []any {
	records := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return rec.Type == azureMarketplaceKeyType && strings.EqualFold(rec.SubscriptionID, sub)
	})
	slices.SortFunc(records, func(a, b *azureARMResource) int { return strings.Compare(a.Name, b.Name) })
	value := make([]any, 0, len(records))
	for _, rec := range records {
		value = append(value, map[string]any{"keyAlias": rec.Name, "key": rec.Properties["key"]})
	}
	return value
}

func azureMarketplaceKeyID(sub, alias string) string {
	return "/subscriptions/" + sub + "/providers/" + azureMarketplaceKeyType + "/" + alias
}

func (s *Server) createAzureMarketplaceKey(w http.ResponseWriter, sub, alias string) {
	if !azureMarketplaceKeyAlias.MatchString(alias) {
		azureARMError(w, http.StatusBadRequest, "BadRequest", "The API key alias must contain only alphanumeric characters separated by dashes.")
		return
	}
	id := azureMarketplaceKeyID(sub, alias)
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if s.azureARMResources[azureARMKey(id)] != nil {
		azureARMError(w, http.StatusConflict, "Conflict", "An API key with alias '"+alias+"' already exists.")
		return
	}
	key := azureARMDerivedKey(id, now.Format(time.RFC3339Nano))
	s.azureARMResources[azureARMKey(id)] = &azureARMResource{
		ID: id, SubscriptionID: sub, Name: alias, Type: azureMarketplaceKeyType,
		Properties: map[string]any{"key": key}, CreatedAt: now, UpdatedAt: now,
	}
	respondJSON(w, http.StatusOK, map[string]any{"keyAlias": alias, "key": key})
}

func (s *Server) revokeAzureMarketplaceKey(w http.ResponseWriter, sub, alias string) {
	key := azureARMKey(azureMarketplaceKeyID(sub, alias))
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if s.azureARMResources[key] == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "API key '"+alias+"' was not found.")
		return
	}
	delete(s.azureARMResources, key)
	w.WriteHeader(http.StatusOK)
}

// azureMarketplaceCatalog is the seed catalog, shaped like Learn's samples.
// markets, which the API doesn't return, holds where a product is sold.
var azureMarketplaceCatalog = func() []map[string]any {
	var out []map[string]any
	if err := json.Unmarshal([]byte(azureMarketplaceCatalogJSON), &out); err != nil {
		panic("azure marketplace catalog: " + err.Error())
	}
	return out
}()

const azureMarketplaceCatalogJSON = `[
{"uniqueProductId":"securescan-enterprise","productId":"securescan-enterprise","displayName":"Example Security Scanner","publisherId":"example-security","publisherDisplayName":"Example Security, Inc.","publisherType":"ThirdParty","productType":"SaaS","productFamily":"Security","productSubType":"SecurityAnalytics","popularity":4.8,"categoryIds":["SecurityAndIdentity"],"industryIds":["Technology"],"operatingSystems":["Linux"],"pricingTypes":["Payg"],"azureBenefit":"Eligible","badges":["PreferredSolution","PowerBICertified","AdditionalPurchaseRequirement"],"publishingStage":"Public","summary":"Enterprise cloud security scanner with continuous monitoring, compliance checking, and automated remediation guidance","longSummary":"Continuously monitors cloud infrastructure for vulnerabilities, misconfigurations, and compliance violations.","description":"Comprehensive cloud security monitoring and compliance management for enterprises.","lastModifiedDateTime":"2025-04-02T10:00:00Z","locations":["global"],"markets":["US","GB","DE"],"ratingAverage":4.8,"ratingCount":245,"ratingBuckets":["AboveFour"],"hasRiPlans":true,"hasMarketplaceFootprint":true,"smallIconUri":"https://www.example.com/icons/small.png",
 "serviceInstructionTemplates":[{"id":"onboarding","displayName":"Onboarding instructions"}],
 "plans":[
  {"planId":"pro","uniquePlanId":"securescan-enterprisepro","skuId":"pro-001","displayName":"Professional","planType":"SaaS","cspState":"OptIn","pricingTypes":["Payg"],"summary":"Advanced security features for growing organizations","description":"Professional-grade security scanning with priority support","displayRank":"2","isPrivate":false,"isHidden":false,"isStopSell":false,"hasRi":true,"stackType":"Gen2","metadata":{"generation":"2023","altStackReference":"basic"},"altStackReference":"basic","categoryIds":["SecurityAndIdentity"],"hasProtectedArtifacts":false},
  {"planId":"basic","uniquePlanId":"securescan-enterprisebasic","skuId":"basic-001","displayName":"Basic","planType":"SaaS","cspState":"OptIn","pricingTypes":["Free"],"summary":"Core scanning","description":"Core scanning for small teams","displayRank":"1","isPrivate":false,"isHidden":true,"isStopSell":false,"hasRi":false,"categoryIds":["SecurityAndIdentity"],"hasProtectedArtifacts":false},
  {"planId":"legacy","uniquePlanId":"securescan-enterpriselegacy","skuId":"legacy-001","displayName":"Legacy","planType":"SaaS","cspState":"OptOut","pricingTypes":["Payg"],"summary":"Retired plan","description":"No longer sold","displayRank":"3","isPrivate":false,"isHidden":false,"isStopSell":true,"hasRi":false,"categoryIds":["SecurityAndIdentity"],"hasProtectedArtifacts":false}]},
{"uniqueProductId":"microsoft.managedinstance_standard","productId":"managedinstance_standard","displayName":"Azure SQL Managed Instance","publisherId":"microsoft","publisherDisplayName":"Microsoft","publisherType":"Microsoft","productType":"SaaS","productFamily":"Databases","popularity":4.6,"categoryIds":["Databases"],"industryIds":[],"operatingSystems":[],"pricingTypes":["Payg","Ri"],"azureBenefit":"Eligible","badges":[],"publishingStage":"Public","summary":"Fully managed SQL Server database engine","longSummary":"Azure SQL Managed Instance combines broad SQL Server compatibility with a fully managed platform.","description":"Managed SQL Server instances on Azure.","lastModifiedDateTime":"2025-03-15T08:00:00Z","locations":["eastus","westeurope"],"markets":["US","GB","DE"],"ratingAverage":4.5,"ratingCount":980,"ratingBuckets":["AboveFour"],"hasRiPlans":true,"hasMarketplaceFootprint":true,"service":"SQL Managed Instance","serviceFamily":"Databases",
 "plans":[{"planId":"standard","uniquePlanId":"microsoft.managedinstance_standardstandard","skuId":"standard","displayName":"General Purpose","planType":"AzureServices","cspState":"OptIn","pricingTypes":["Payg","Ri"],"summary":"General purpose tier","description":"Balanced compute and storage","displayRank":"1","isPrivate":false,"isHidden":false,"isStopSell":false,"hasRi":true,"categoryIds":["Databases"],"hasProtectedArtifacts":false}]},
{"uniqueProductId":"canonical.ubuntu-24_04-lts","productId":"ubuntu-24_04-lts","displayName":"Ubuntu Server 24.04 LTS","publisherId":"canonical","publisherDisplayName":"Canonical","publisherType":"ThirdParty","productType":"VirtualMachine","productFamily":"Compute","popularity":4.9,"categoryIds":["Compute","OperatingSystems"],"industryIds":[],"operatingSystems":["Linux"],"pricingTypes":["Free"],"azureBenefit":"NotEligible","badges":[],"publishingStage":"Public","summary":"Ubuntu Server 24.04 LTS virtual machine image","longSummary":"Long-term support release of Ubuntu Server for Azure virtual machines.","description":"Ubuntu Server images optimized for Azure.","lastModifiedDateTime":"2025-05-01T00:00:00Z","locations":["eastus","westus2","westeurope"],"markets":["US","GB","DE","JP"],"ratingAverage":4.7,"ratingCount":3120,"ratingBuckets":["AboveFour"],"hasRiPlans":false,"hasMarketplaceFootprint":true,
 "plans":[
  {"planId":"server","uniquePlanId":"canonical.ubuntu-24_04-ltsserver","skuId":"server","displayName":"Ubuntu Server 24.04 LTS - x64 Gen2","planType":"VirtualMachine","cspState":"OptIn","pricingTypes":["Free"],"summary":"x64 Gen2 image","description":"Generation 2 image","displayRank":"1","isPrivate":false,"isHidden":false,"isStopSell":false,"hasRi":false,"stackType":"Gen2","vmArchitectureType":"X64Gen2","vmSecuritytypes":["Trusted","Confidential"],"metadata":{"generation":"2","altStackReference":"server-gen1"},"altStackReference":"server-gen1","categoryIds":["Compute"],"hasProtectedArtifacts":false},
  {"planId":"server-arm64","uniquePlanId":"canonical.ubuntu-24_04-ltsserver-arm64","skuId":"server-arm64","displayName":"Ubuntu Server 24.04 LTS - Arm64","planType":"VirtualMachine","cspState":"OptIn","pricingTypes":["Free"],"summary":"Arm64 image","description":"Arm64 image","displayRank":"2","isPrivate":false,"isHidden":false,"isStopSell":false,"hasRi":false,"stackType":"Gen2","vmArchitectureType":"Arm64","vmSecuritytypes":["Trusted"],"metadata":{"generation":"2"},"categoryIds":["Compute"],"hasProtectedArtifacts":false}]},
{"uniqueProductId":"contoso.analytics-suite","productId":"analytics-suite","displayName":"Contoso's Analytics Suite","publisherId":"contoso","publisherDisplayName":"Contoso","publisherType":"ThirdParty","productType":"AzureApplication","productFamily":"Analytics","popularity":3.9,"categoryIds":["Analytics"],"industryIds":["Retail"],"operatingSystems":[],"pricingTypes":["Byol","Payg"],"azureBenefit":"Eligible","badges":[],"publishingStage":"Public","summary":"Retail analytics deployed into your subscription","longSummary":"A managed analytics application with a private plan for negotiated customers.","description":"Analytics for retail data.","lastModifiedDateTime":"2024-11-20T12:00:00Z","locations":["eastus"],"markets":["US"],"ratingAverage":3.9,"ratingCount":41,"ratingBuckets":["AboveThree"],"hasRiPlans":false,"hasMarketplaceFootprint":true,
 "plans":[
  {"planId":"standard","uniquePlanId":"contoso.analytics-suitestandard","skuId":"standard","displayName":"Standard","planType":"ManagedApplication","cspState":"SelectiveOptIn","pricingTypes":["Payg"],"summary":"Public plan","description":"Pay as you go","displayRank":"1","isPrivate":false,"isHidden":false,"isStopSell":false,"hasRi":false,"categoryIds":["Analytics"],"hasProtectedArtifacts":false},
  {"planId":"negotiated","uniquePlanId":"contoso.analytics-suitenegotiated","skuId":"negotiated","displayName":"Negotiated","planType":"ManagedApplication","cspState":"OptOut","pricingTypes":["Byol"],"summary":"Private plan","description":"Bring your own license under a private offer","displayRank":"2","isPrivate":true,"isHidden":false,"isStopSell":false,"hasRi":false,"categoryIds":["Analytics"],"hasProtectedArtifacts":true}]},
{"uniqueProductId":"fabrikam.inventory-api","productId":"inventory-api","displayName":"Fabrikam Inventory API","publisherId":"fabrikam","publisherDisplayName":"Fabrikam","publisherType":"ThirdParty","productType":"ContainerApps","productFamily":"Containers","popularity":2.7,"categoryIds":["Containers","Integration"],"industryIds":["Manufacturing"],"operatingSystems":["Linux"],"pricingTypes":["Byol"],"azureBenefit":"NotEligible","badges":[],"publishingStage":"Preview","summary":"Inventory API packaged for Azure Container Apps","longSummary":"A containerized inventory service for Azure Container Apps.","description":"Inventory tracking API.","lastModifiedDateTime":"2025-01-10T09:30:00Z","locations":["westeurope"],"markets":["GB","DE"],"ratingAverage":0,"ratingCount":0,"ratingBuckets":[],"hasRiPlans":false,"hasMarketplaceFootprint":false,
 "plans":[{"planId":"byol","uniquePlanId":"fabrikam.inventory-apibyol","skuId":"byol","displayName":"Bring your own license","planType":"ContainerApps","cspState":"OptOut","pricingTypes":["Byol"],"summary":"BYOL","description":"Use an existing license","displayRank":"1","isPrivate":false,"isHidden":false,"isStopSell":false,"hasRi":false,"categoryIds":["Containers"],"hasProtectedArtifacts":false}]}
]`
