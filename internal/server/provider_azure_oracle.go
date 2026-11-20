package server

import (
	"maps"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Oracle Database@Azure (Oracle.Database 2026-06-01): the marketplace
// Oracle subscription, Exadata infrastructure with its database servers,
// Exadata VM clusters with their database nodes and virtual network
// addresses, and Autonomous Databases with their backups, plus the
// read-only regional catalogs (shapes, Grid Infrastructure and system
// versions, character sets, database versions, DNS views and zones).
// Nothing runs in OCI: lifecycle states settle immediately.

const azureOracleNamespace = "Oracle.Database"

var (
	azureOracleVnetID   = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.Network/virtualNetworks/[^/]+$`)
	azureOracleSubnetID = regexp.MustCompile(`(?i)^(/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.Network/virtualNetworks/[^/]+)/subnets/[^/]+$`)
	azureOracleGUID     = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	azureOracleDuration = regexp.MustCompile(`^P[1-9][0-9]*[YM]$`)
	azureOracleEmail    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// azureOracleShape is a seeded Exadata shape. The per-server capacities
// are the emulator's catalog values, not Learn's; they bound what VM
// clusters may allocate.
type azureOracleShape struct {
	name, model                         string
	coresPerNode, memoryPerNode, nodeGB int
	tbPerStorageServer                  float64
}

// azureOracleShapes are the Exadata infrastructure shapes Learn's
// SystemShapes enum lists (ExaDbXS is Exascale, which has no
// infrastructure).
var azureOracleShapes = []azureOracleShape{
	{name: "Exadata.X9M", model: "OCPU", coresPerNode: 126, memoryPerNode: 1390, nodeGB: 2243, tbPerStorageServer: 63.6},
	{name: "Exadata.X11M", model: "ECPU", coresPerNode: 760, memoryPerNode: 1390, nodeGB: 2243, tbPerStorageServer: 80},
}

func azureOracleShapeNamed(name string) (azureOracleShape, bool) {
	for _, shape := range azureOracleShapes {
		if strings.EqualFold(shape.name, name) {
			return shape, true
		}
	}
	return azureOracleShape{}, false
}

// azureOracleGIVersions are the seeded Grid Infrastructure versions and
// the shapes they run on.
var azureOracleGIVersions = map[string][]string{
	"19.0.0.0": {"Exadata.X9M", "Exadata.X11M"},
	"23.0.0.0": {"Exadata.X9M", "Exadata.X11M", "ExaDbXS"},
}

var (
	azureOracleSystemVersions = []string{"22.1.7.0.0.230113", "24.1.0.0.0.240118"}
	azureOracleCharacterSets  = []string{"AL32UTF8", "UTF8", "WE8ISO8859P1"}
	azureOracleNCharSets      = []string{"AL16UTF16", "UTF8"}
	azureOracleADBVersions    = []string{"19c", "23ai"}
)

func init() {
	ns := azureOracleNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "oracleSubscriptions", Subscription: true,
		ValidateName: func(name string) string {
			if name != "default" {
				return "The Oracle subscription must be named default."
			}
			return ""
		},
		Validate: azureOracleSubscriptionValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"saasSubscriptionId": azureWorkloadsHash(rec.ID+"#saas", 32), "cloudAccountId": azureOracleOCID("tenancy", rec.ID),
				// ponytail: the OCI account is treated as linked; nothing
				// here can complete the OCI console's linking step.
				"cloudAccountState": "Available",
			}
		},
		StateProperties: map[string]any{"azureSubscriptionIds": nil, "addSubscriptionOperationState": nil},
		PatchBody:       azureOracleSubscriptionPatch,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"listcloudaccountdetails":     azureOracleCloudAccount,
			"listsaassubscriptiondetails": azureOracleSaaSDetails,
			"listactivationlinks":         azureOracleActivationLinks,
			"addazuresubscriptions":       azureOracleAddSubscriptions,
		},
	})
	azureOracleRegisterExadata()
	azureOracleRegisterADB()
	azureOracleRegisterCatalogs()
}

// azureOracleOCID is a deterministic OCI identifier for a resource.
func azureOracleOCID(kind, id string) string {
	return "ocid1." + kind + ".oc1..aaaa" + azureWorkloadsHash(id, 40)
}

func azureOracleBad(msg string) (int, any) {
	return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
}

func azureOracleConflict(msg string) (int, any) {
	return http.StatusConflict, azureARMErrorBody("Conflict", msg)
}

// azureOracleUpdateLocked stores a copy of the record at id with props
// merged in. Caller holds providerStorageMu.
func (s *Server) azureOracleUpdateLocked(id string, props map[string]any) *azureARMResource {
	stored := s.azureARMResources[azureARMKey(id)]
	if stored == nil {
		return nil
	}
	updated := *stored
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMMerge(stored.Properties, props)
	s.azureARMResources[azureARMKey(id)] = &updated
	return &updated
}

// azureOracleEntitledLocked checks that the subscription may provision
// Oracle resources: it holds the purchased Oracle subscription, or an
// Oracle subscription elsewhere added it. Caller holds providerStorageMu.
func (s *Server) azureOracleEntitledLocked(sub string) string {
	found := s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
		if !strings.EqualFold(r.Type, azureOracleNamespace+"/oracleSubscriptions") {
			return false
		}
		return strings.EqualFold(r.SubscriptionID, sub) || slices.ContainsFunc(azureARMSlice(r.Properties["azureSubscriptionIds"]), func(v any) bool {
			return strings.EqualFold(azureARMString(v), sub)
		})
	})
	if len(found) == 0 {
		return "Subscription " + sub + " has no Oracle Database@Azure offer: purchase it (oracleSubscriptions/default) or add this subscription to an existing one first."
	}
	return ""
}

// azureOracleNumber checks that m[key], when present, is a number from
// least to most, and a whole number when whole is set.
func azureOracleNumber(m map[string]any, path, key string, least, most float64, whole bool) string {
	raw, present := m[key]
	if !present {
		return ""
	}
	n, ok := raw.(float64)
	if !ok || n < least || n > most || (whole && n != float64(int64(n))) {
		kind := "a number"
		if whole {
			kind = "a whole number"
		}
		return path + key + " must be " + kind + " from " + strconv.FormatFloat(least, 'f', -1, 64) + " to " + strconv.FormatFloat(most, 'f', -1, 64) + "."
	}
	return ""
}

// azureOracleStrings checks that m[key], when present, is a list of
// least to most non-empty strings of at most 255 characters.
func azureOracleStrings(m map[string]any, path, key string, least, most int) string {
	raw, present := m[key]
	if !present {
		return ""
	}
	items, ok := raw.([]any)
	if !ok || len(items) < least || len(items) > most {
		return path + key + " must be a list of " + strconv.Itoa(least) + " to " + strconv.Itoa(most) + " strings."
	}
	for _, item := range items {
		if text, ok := item.(string); !ok || text == "" || len(text) > 255 {
			return path + key + " must hold non-empty strings of at most 255 characters."
		}
	}
	return ""
}

// azureOracleBools checks that the keys of m, when present, are booleans.
func azureOracleBools(m map[string]any, path string, keys ...string) string {
	for _, key := range keys {
		if raw, present := m[key]; present {
			if _, ok := raw.(bool); !ok {
				return path + key + " must be true or false."
			}
		}
	}
	return ""
}

// azureOracleNetwork checks a resource's optional vnetId and subnetId: a
// virtual network and one of its subnets.
func azureOracleNetwork(props map[string]any, required bool) string {
	vnet, subnet := azureARMString(props["vnetId"]), azureARMString(props["subnetId"])
	if required && (vnet == "" || subnet == "") {
		return "properties.vnetId and properties.subnetId are required."
	}
	if vnet == "" && subnet == "" {
		return ""
	}
	if !azureOracleVnetID.MatchString(vnet) {
		return "properties.vnetId must be a virtual network ID: /subscriptions/{id}/resourceGroups/{group}/providers/Microsoft.Network/virtualNetworks/{name}."
	}
	m := azureOracleSubnetID.FindStringSubmatch(subnet)
	if m == nil || !strings.EqualFold(m[1], vnet) {
		return "properties.subnetId must be a subnet of properties.vnetId, delegated to Oracle.Database/networkAttachment."
	}
	return ""
}

// azureOracleContacts checks customerContacts: objects with an email, at
// most limit of them (0 for no limit).
func azureOracleContacts(props map[string]any, limit int) string {
	raw, present := props["customerContacts"]
	if !present {
		return ""
	}
	items, ok := raw.([]any)
	if !ok || (limit > 0 && len(items) > limit) {
		return "properties.customerContacts must be a list of at most " + strconv.Itoa(limit) + " contacts."
	}
	for _, item := range items {
		if !azureOracleEmail.MatchString(azureARMString(azureARMObject(item)["email"])) {
			return "properties.customerContacts entries need a valid email."
		}
	}
	return ""
}

// azureOracleWhitelist checks an IP access list of addresses and CIDRs.
func azureOracleWhitelist(props map[string]any) string {
	if msg := azureOracleStrings(props, "properties.", "whitelistedIps", 1, 1024); msg != "" {
		return msg
	}
	for _, item := range azureARMSlice(props["whitelistedIps"]) {
		text := azureARMString(item)
		if net.ParseIP(text) == nil {
			if _, _, err := net.ParseCIDR(text); err != nil {
				return "properties.whitelistedIps entry '" + text + "' isn't an IP address or CIDR."
			}
		}
	}
	return ""
}

// azureOracleOnly rejects keys outside allowed, for update models.
func azureOracleOnly(m map[string]any, path string, allowed []string) string {
	for key := range m {
		if !slices.Contains(allowed, key) {
			return path + key + " can't be updated."
		}
	}
	return ""
}

func azureOracleSubscriptionValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties", "plan"); msg != "" {
		return msg
	}
	if msg := azureOraclePlan(azureARMObject(payload["plan"])); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if term, present := props["termUnit"]; present && !azureOracleDuration.MatchString(azureARMString(term)) {
		return "properties.termUnit must be an ISO 8601 duration such as P1Y or P3Y."
	}
	return azureOracleSubscriptionProps(props)
}

func azureOraclePlan(plan map[string]any) string {
	for key, value := range plan {
		if _, ok := value.(string); !ok || !slices.Contains([]string{"name", "product", "promotionCode", "publisher", "version"}, key) {
			return "plan." + key + " isn't a plan field; plan holds the strings name, product, promotionCode, publisher and version."
		}
	}
	return ""
}

func azureOracleSubscriptionProps(props map[string]any) string {
	if msg := azureLabEnum(props, "properties.", "intent", []string{"Retain", "Reset"}); msg != "" {
		return msg
	}
	return azureLabLength(props, "properties.", "productCode", 1, 255)
}

// azureOracleSubscriptionPatch takes the update model: tags, the plan,
// and properties.intent and productCode.
func azureOracleSubscriptionPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureOracleOnly(payload, "", []string{"tags", "plan", "properties"}); msg != "" {
		return nil, nil, msg
	}
	if msg := azureDataBoxObjects(payload, "", "tags", "plan", "properties"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureOracleOnly(props, "properties.", []string{"intent", "productCode"}); msg != "" {
		return nil, nil, msg
	}
	if msg := azureOraclePlan(azureARMObject(payload["plan"])); msg != "" {
		return nil, nil, msg
	}
	return payload, nil, azureOracleSubscriptionProps(props)
}

func azureOracleCloudAccount(_ *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, map[string]any{"cloudAccountName": "stackyard-" + azureWorkloadsHash(rec.ID, 8), "cloudAccountHomeRegion": "eastus"}
}

func azureOracleSaaSDetails(_ *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	term := azureARMString(rec.Properties["termUnit"])
	if term == "" {
		term = "P1M"
	}
	return http.StatusOK, map[string]any{
		"id": rec.Properties["saasSubscriptionId"], "subscriptionName": "Oracle Database@Azure", "timeCreated": rec.CreatedAt.Format(time.RFC3339),
		"offerId": azureARMString(rec.Plan["product"]), "planId": azureARMString(rec.Plan["name"]), "publisherId": "Oracle",
		"saasSubscriptionStatus": "Subscribed", "termUnit": term, "isAutoRenew": true, "isFreeTrial": false,
	}
}

func azureOracleActivationLinks(_ *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	q := url.Values{"saas": {azureARMString(rec.Properties["saasSubscriptionId"])}}.Encode()
	return http.StatusOK, map[string]any{
		"newCloudAccountActivationLink":      "https://cloud.oracle.com/azure/activate/new?" + q,
		"existingCloudAccountActivationLink": "https://cloud.oracle.com/azure/activate/existing?" + q,
	}
}

// azureOracleAddSubscriptions lets other Azure subscriptions provision
// against this Oracle subscription; Learn answers 202.
func azureOracleAddSubscriptions(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureOracleStrings(payload, "", "azureSubscriptionIds", 1, 1000); msg != "" || payload["azureSubscriptionIds"] == nil {
		return azureOracleBad("azureSubscriptionIds must be a non-empty list of subscription IDs.")
	}
	added := azureARMSlice(payload["azureSubscriptionIds"])
	for _, id := range added {
		if !azureOracleGUID.MatchString(azureARMString(id)) {
			return azureOracleBad("azureSubscriptionIds entry '" + azureARMString(id) + "' isn't a subscription ID.")
		}
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	stored := s.azureARMResources[azureARMKey(rec.ID)]
	if stored == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	ids := slices.Clone(azureARMSlice(stored.Properties["azureSubscriptionIds"]))
	for _, id := range added {
		if !slices.ContainsFunc(ids, func(v any) bool { return strings.EqualFold(azureARMString(v), azureARMString(id)) }) {
			ids = append(ids, strings.ToLower(azureARMString(id)))
		}
	}
	s.azureOracleUpdateLocked(rec.ID, map[string]any{"azureSubscriptionIds": ids, "addSubscriptionOperationState": "Succeeded"})
	return http.StatusAccepted, nil
}

// azureOracleRegisterCatalogs serves the read-only regional catalogs at
// /subscriptions/{id}/providers/Oracle.Database/locations/{location}/...
func azureOracleRegisterCatalogs() {
	ns := azureOracleNamespace
	catalog := func(collection string, items func(scope string, query url.Values) []map[string]any) {
		registerAzureARMQueryView(ns, "locations/"+collection, func(_ *Server, scope string, query url.Values) []map[string]any {
			return items(scope+"/"+collection, query)
		})
	}
	item := func(base, typ, name string, props map[string]any) map[string]any {
		return map[string]any{"id": base + "/" + name, "name": name, "type": ns + "/locations/" + typ, "properties": props}
	}
	catalog("dbSystemShapes", func(base string, _ url.Values) []map[string]any {
		out := []map[string]any{}
		for _, shape := range azureOracleShapes {
			out = append(out, item(base, "dbSystemShapes", shape.name, map[string]any{
				"shapeFamily": "EXADATA", "shapeName": shape.name, "computeModel": shape.model,
				"availableCoreCount": shape.coresPerNode * 32, "availableCoreCountPerNode": shape.coresPerNode, "minimumCoreCount": 2,
				"runtimeMinimumCoreCount": 2, "coreCountIncrement": 1, "minimumNodeCount": 2, "maximumNodeCount": 32,
				"minStorageCount": 3, "maxStorageCount": 64, "availableMemoryPerNodeInGbs": shape.memoryPerNode,
				"availableDbNodePerNodeInGbs": shape.nodeGB, "availableDataStoragePerServerInTbs": shape.tbPerStorageServer,
				"areServerTypesSupported": shape.model == "ECPU",
			}))
		}
		return out
	})
	catalog("giVersions", func(base string, query url.Values) []map[string]any {
		out := []map[string]any{}
		for _, version := range slices.Sorted(maps.Keys(azureOracleGIVersions)) {
			shape := query.Get("shape")
			if shape != "" && !slices.ContainsFunc(azureOracleGIVersions[version], func(v string) bool { return strings.EqualFold(v, shape) }) {
				continue
			}
			out = append(out, item(base, "giVersions", version, map[string]any{"version": version}))
		}
		return out
	})
	list := func(collection, key string, values []string) {
		catalog(collection, func(base string, _ url.Values) []map[string]any {
			out := []map[string]any{}
			for _, v := range values {
				out = append(out, item(base, collection, v, map[string]any{key: v}))
			}
			return out
		})
	}
	list("systemVersions", "systemVersion", azureOracleSystemVersions)
	list("autonomousDatabaseCharacterSets", "characterSet", azureOracleCharacterSets)
	list("autonomousDatabaseNationalCharacterSets", "characterSet", azureOracleNCharSets)
	list("autonomousDbVersions", "version", azureOracleADBVersions)
	dns := func(collection, kind string, props func(location, ocid string) map[string]any) {
		catalog(collection, func(base string, _ url.Values) []map[string]any {
			location := base[strings.LastIndex(base[:strings.LastIndex(base, "/")], "/")+1 : strings.LastIndex(base, "/")]
			ocid := azureOracleOCID(kind, strings.ToLower(location))
			return []map[string]any{item(base, collection, ocid, azureARMMerge(map[string]any{
				"ocid": ocid, "isProtected": true, "lifecycleState": "Active", "timeCreated": "2024-01-01T00:00:00Z",
				"self": "https://dns." + strings.ToLower(location) + ".oci.oraclecloud.com/20180115/" + collection + "/" + ocid,
			}, props(location, ocid)))}
		})
	}
	dns("dnsPrivateViews", "dnsview", func(location, _ string) map[string]any {
		return map[string]any{"displayName": "default-view-" + strings.ToLower(location), "timeUpdated": "2024-01-01T00:00:00Z"}
	})
	dns("dnsPrivateZones", "dnszone", func(location, _ string) map[string]any {
		return map[string]any{"zoneType": "Primary", "serial": 1, "version": "1", "viewId": azureOracleOCID("dnsview", strings.ToLower(location))}
	})
}
