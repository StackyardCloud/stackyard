package server

import (
	"cmp"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// HorizonDB settings: parameter groups with the regional default groups
// and capability catalog, the cluster's two authentication methods, its
// Microsoft Entra administrators and its log capture configuration.

// azureHZParameter is an engine parameter of the parameter group catalog.
type azureHZParameter struct {
	Name, Description, DataType, AllowedValues, Default string
	Dynamic, ReadOnly                                   bool
	Unit, Doc                                           string
}

// ponytail: a subset of the default group's parameters, from the Learn
// examples, sorted by name; add more as clients need them.
var azureHZParameters = []azureHZParameter{
	{"allow_alter_system", "Allows running the ALTER SYSTEM command. Can be set to off for environments where global configuration changes should be made using a different method.", "Boolean", "on,off", "on", true, true, "", "runtime-config-compatible.html#GUC-ALLOW-ALTER-SYSTEM"},
	{"application_name", "Sets the application name to be reported in statistics and logs.", "String", "[A-Za-z0-9._-]*", "", true, false, "", "libpq-connect.html#LIBPQ-CONNECT-APPLICATION-NAME"},
	{"archive_mode", "Allows archiving of WAL files using \"archive_command\".", "Enumeration", "always,on,off", "off", false, true, "", "runtime-config-wal.html#GUC-ARCHIVE-MODE"},
	{"array_nulls", "Enable input of NULL elements in arrays. When turned on, unquoted NULL in an array input value means a null value; otherwise it is taken literally.", "Boolean", "on,off", "on", true, false, "", "runtime-config-compatible.html#GUC-ARRAY-NULLS"},
	{"authentication_timeout", "Sets the maximum allowed time to complete client authentication.", "Integer", "1-600", "60", true, true, "s", "runtime-config-connection.html#GUC-AUTHENTICATION-TIMEOUT"},
	{"autovacuum", "Starts the autovacuum subprocess.", "Boolean", "on,off", "on", true, false, "", "runtime-config-autovacuum.html#GUC-AUTOVACUUM"},
	{"autovacuum_analyze_scale_factor", "Number of tuple inserts, updates, or deletes prior to analyze as a fraction of reltuples.", "Numeric", "0-100", "0.1", true, false, "", "runtime-config-autovacuum.html#GUC-AUTOVACUUM-ANALYZE-SCALE-FACTOR"},
	{"autovacuum_freeze_max_age", "Age at which to autovacuum a table to prevent transaction ID wraparound.", "Integer", "100000-2000000000", "200000000", false, false, "", "runtime-config-autovacuum.html#GUC-AUTOVACUUM-FREEZE-MAX-AGE"},
	{"wal_receiver_timeout", "Sets the maximum wait time to receive data from the sending server.", "Integer", "0-2147483647", "60000", true, false, "ms", "runtime-config-replication.html#GUC-WAL-RECEIVER-TIMEOUT"},
	{"work_mem", "Sets the maximum memory to be used for query workspaces. This much memory can be used by each internal sort operation and hash table before switching to temporary disk files.", "Integer", "4096-2097151", "4096", true, false, "kB", "runtime-config-resource.html#GUC-WORK-MEM"},
	{"xmlbinary", "Sets how binary values are to be encoded in XML.", "Enumeration", "base64,hex", "base64", true, false, "", "runtime-config-client.html#GUC-XMLBINARY"},
	{"xmloption", "Sets whether XML data in implicit parsing and serialization operations is to be considered as documents or content fragments.", "Enumeration", "content,document", "content", true, false, "", "runtime-config-client.html#GUC-XMLOPTION"},
}

var (
	azureHZGroupName     = regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9]*[a-zA-Z0-9]$`)
	azureHZPrincipalKind = []string{"Unknown", "User", "Group", "ServicePrincipal"}
	// azureHZVCores are the pool sizes the capability catalog offers.
	azureHZVCores = []int{2, 4, 8, 16, 32, 48, 64, 96, 128, 192}
)

// azureHZPatchParameters carries a PATCH's parameter overrides to
// Reconcile, which overlays them on the stored ones by name; a PUT's
// parameters replace them.
const azureHZPatchParameters = "_patchParameters"

func azureHZRegisterSettings() {
	ns := azureHZNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "parameterGroups",
		ValidateName: func(name string) string {
			if len(name) > 63 || !azureHZGroupName.MatchString(name) {
				return "The parameter group name '" + name + "' must be 2 to 63 letters, digits and hyphens, starting and ending with a letter or digit."
			}
			return ""
		},
		Validate: func(payload map[string]any) string {
			return cmp.Or(azureDataBoxObjects(payload, "", "properties"), azureHZGroupShape(azureARMObject(payload["properties"])))
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			if msg := cmp.Or(
				azureOracleOnly(payload, "", []string{"tags", "properties"}),
				azureOracleOnly(props, "properties.", []string{"parameters", "description"}),
				azureHZGroupShape(props),
			); msg != "" {
				return nil, nil, msg
			}
			if sent, ok := props["parameters"]; ok {
				props = maps.Clone(props)
				delete(props, "parameters")
				props[azureHZPatchParameters] = sent
				payload = maps.Clone(payload)
				payload["properties"] = props
			}
			return payload, nil, ""
		},
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			parameters := rec.Properties["parameters"]
			if patch, ok := rec.Properties[azureHZPatchParameters]; ok {
				delete(rec.Properties, azureHZPatchParameters)
				parameters = append(slices.Clone(azureARMSlice(parameters)), azureARMSlice(patch)...)
			}
			version := rec.Properties["postgreSqlVersion"]
			switch {
			case existing != nil && version == nil:
				version = existing.Properties["postgreSqlVersion"]
			case existing != nil && version != existing.Properties["postgreSqlVersion"]:
				return nil, "properties.postgreSqlVersion can't be changed after the parameter group is created."
			case version == nil:
				return nil, "properties.postgreSqlVersion is required."
			}
			return map[string]any{"postgreSqlVersion": version, "parameters": azureHZParameterList(azureARMString(version), parameters)}, ""
		},
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			if len(s.azureHZGroupPoolsLocked(existing.ID)) > 0 {
				return "The parameter group is associated with one or more pools."
			}
			return ""
		},
		DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMView(ns, "parameterGroups/connections", func(s *Server, scope string) []map[string]any {
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		var out []map[string]any
		for _, pool := range s.azureHZGroupPoolsLocked(scope) {
			out = append(out, map[string]any{"poolResourceId": pool.ID})
		}
		return out
	})
	registerAzureARMView(ns, "locations/parameterGroups", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, version := range azureHZVersions {
			name := azureHZDefaultGroupName(version)
			out = append(out, map[string]any{
				"id": scope + "/parameterGroups/" + name, "name": name, "type": ns + "/locations/parameterGroups",
				"properties": map[string]any{
					"provisioningState": "Succeeded", "postgreSqlVersion": version,
					"description": "Default parameters for PostgreSQL " + version + " provisioned pools",
					"parameters":  azureHZParameterList(version, nil),
				},
			})
		}
		return out
	})
	registerAzureARMView(ns, "locations/capabilities", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/capabilities/default", "name": "default", "type": ns + "/locations/capabilities",
			"properties": azureHZCapabilities(),
		}}
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/authentications", Proxy: true,
		ValidateName: func(name string) string {
			if name != "passwordMethod" && name != "microsoftEntra" {
				return "The authentication method must be passwordMethod or microsoftEntra."
			}
			return ""
		},
		Validate: func(payload map[string]any) string {
			return cmp.Or(azureDataBoxObjects(payload, "", "properties"),
				azureWorkloadsEnums(azureARMObject(payload["properties"]), "properties.", []string{"state"}, map[string][]string{"state": azureHZStates}))
		},
		ValidateInParent: func(_ map[string]any, name string, payload map[string]any) string {
			return azureHZAuthentication(name, azureARMObject(payload["properties"]), true)
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			if props["state"] == "Disabled" && props["administratorLoginPassword"] != nil {
				return nil, nil, "Disabling password authentication can't include properties.administratorLoginPassword."
			}
			return payload, nil, cmp.Or(
				azureOracleOnly(payload, "", []string{"properties"}),
				azureOracleOnly(props, "properties.", []string{"state", "administratorLoginPassword"}),
				azureLabEnum(props, "properties.", "state", azureHZStates),
				azureLabLength(props, "properties.", "administratorLoginPassword", 8, 128),
			)
		},
		ValidatePatch: func(_ *Server, existing *azureARMResource, payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			if props["state"] == nil && props["administratorLoginPassword"] != nil {
				props = azureARMMerge(props, map[string]any{"state": existing.Properties["state"]})
			}
			return azureHZAuthentication(existing.Name, props, false)
		},
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			if rec.Name != "passwordMethod" {
				return nil, ""
			}
			var stored any
			if existing != nil {
				stored = existing.Properties["administratorLogin"]
			}
			login := rec.Properties["administratorLogin"]
			switch {
			case stored != nil && login != nil && login != stored:
				return nil, "properties.administratorLogin can't be changed once it's set."
			case stored != nil:
				return map[string]any{"administratorLogin": stored}, ""
			case login == nil && rec.Properties["state"] == "Enabled":
				return nil, "properties.administratorLogin and administratorLoginPassword are required to enable password authentication."
			}
			return nil, ""
		},
		WriteOnlyProperties: []string{"administratorLoginPassword"},
		NoDelete:            true,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/microsoftEntraAdministrators", Proxy: true,
		ValidateName: func(name string) string {
			if !azureOracleGUID.MatchString(name) {
				return "The administrator name must be the principal's object ID, a GUID."
			}
			return ""
		},
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			if raw, present := props["tenantId"]; present && !azureOracleGUID.MatchString(azureARMString(raw)) {
				return "properties.tenantId must be a GUID."
			}
			return cmp.Or(
				azureDataBoxObjects(payload, "", "properties"),
				azureWorkloadsEnums(props, "properties.", []string{"principalName", "principalType"}, map[string][]string{"principalType": azureHZPrincipalKind}),
				azureLabLength(props, "properties.", "principalName", 1, 256),
			)
		},
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			out := map[string]any{"objectId": rec.Name, "tenantId": cmp.Or(azureARMString(rec.Properties["tenantId"]), azureDefaultTenantID)}
			if existing != nil {
				for _, key := range []string{"principalName", "principalType", "tenantId"} {
					want, stored := cmp.Or(rec.Properties[key], out[key]), existing.Properties[key]
					if want != stored && (key != "tenantId" || !strings.EqualFold(azureARMString(want), azureARMString(stored))) {
						return nil, "properties." + key + " can't be changed after the administrator is added."
					}
				}
				out["tenantId"] = existing.Properties["tenantId"]
			}
			return out, ""
		},
		NoPatch: true, DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/logCaptureConfigurations", Proxy: true,
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			return payload, nil, cmp.Or(
				azureOracleOnly(payload, "", []string{"properties"}),
				azureOracleOnly(props, "properties.", []string{"state", "retentionDays"}),
				azureLabEnum(props, "properties.", "state", azureHZStates),
				azureOracleNumber(props, "properties.", "retentionDays", 1, 35, true),
			)
		},
		NoPut: true, NoDelete: true,
	})
}

// azureHZAuthentication checks an authentication method's properties:
// Microsoft Entra has only a state; password authentication sets its
// login (PUT only) and password only while enabling, never disabling.
func azureHZAuthentication(name string, props map[string]any, put bool) string {
	if name != "passwordMethod" {
		return azureOracleOnly(props, "properties.", []string{"state", "provisioningState"})
	}
	_, login := props["administratorLogin"]
	_, password := props["administratorLoginPassword"]
	switch {
	case !put && login:
		return "properties.administratorLogin can't be updated."
	case (login || password) && props["state"] != "Enabled":
		return "Setting the administrator login or password requires properties.state Enabled in the same request."
	case put && login && !password:
		// The password is write-only, so only the request can show it.
		return "properties.administratorLoginPassword is required with properties.administratorLogin."
	}
	return cmp.Or(
		azureOracleOnly(props, "properties.", []string{"state", "administratorLogin", "administratorLoginPassword", "provisioningState"}),
		azureLabLength(props, "properties.", "administratorLogin", 1, 63),
		azureLabLength(props, "properties.", "administratorLoginPassword", 8, 128),
	)
}

// azureHZGroupShape checks a parameter group's properties: the version,
// the description and parameter overrides of the catalog's writable
// parameters, each named once with a valid value.
func azureHZGroupShape(props map[string]any) string {
	msg := cmp.Or(
		azureLabEnum(props, "properties.", "postgreSqlVersion", azureHZVersions),
		azureLabLength(props, "properties.", "description", 0, 1024),
	)
	raw, present := props["parameters"]
	if msg != "" || !present {
		return msg
	}
	items, ok := raw.([]any)
	if !ok {
		return "properties.parameters must be a list."
	}
	seen := map[string]bool{}
	for _, item := range items {
		entry := azureARMObject(item)
		name, value := azureARMString(entry["name"]), entry["value"]
		i := slices.IndexFunc(azureHZParameters, func(p azureHZParameter) bool { return p.Name == name })
		switch {
		case i < 0:
			return "Parameter '" + name + "' isn't a HorizonDB engine parameter."
		case seen[name]:
			return "Parameter '" + name + "' is listed more than once."
		}
		seen[name] = true
		switch {
		case !azureHZOverride(entry):
			continue
		case azureHZParameters[i].ReadOnly:
			return "Parameter '" + name + "' is read-only."
		}
		text, ok := value.(string)
		if !ok {
			return "Parameter '" + name + "' needs a string value."
		}
		if msg := azureHZCheckValue(azureHZParameters[i], text); msg != "" {
			return msg
		}
	}
	return ""
}

// azureHZOverride reports whether a parameters entry sets a value: one a
// client sends, or one a previous write overrode, but not a default that
// a GET response listed.
func azureHZOverride(entry map[string]any) bool {
	return entry["value"] != nil && entry["valueSource"] != "SystemDefault"
}

func azureHZCheckValue(param azureHZParameter, value string) string {
	switch param.DataType {
	case "String":
		if !regexp.MustCompile(`^(?:` + param.AllowedValues + `)$`).MatchString(value) {
			return "Value '" + value + "' of " + param.Name + " must match " + param.AllowedValues + "."
		}
	case "Numeric":
		lo, hi, _ := strings.Cut(param.AllowedValues, "-")
		n, err := strconv.ParseFloat(value, 64)
		low, _ := strconv.ParseFloat(lo, 64)
		high, _ := strconv.ParseFloat(hi, 64)
		if err != nil || n < low || n > high {
			return "Value '" + value + "' of " + param.Name + " must be a number from " + param.AllowedValues + "."
		}
	default:
		return azurePGCheckValue(azurePGParameter{Name: param.Name, DataType: param.DataType, AllowedValues: param.AllowedValues}, value)
	}
	return ""
}

// azureHZParameterList is the full parameter catalog for the version with
// the overrides in raw applied.
func azureHZParameterList(version string, raw any) []any {
	overrides := map[string]string{}
	for _, item := range azureARMSlice(raw) {
		if entry := azureARMObject(item); azureHZOverride(entry) {
			overrides[azureARMString(entry["name"])] = azureARMString(entry["value"])
		}
	}
	out := make([]any, 0, len(azureHZParameters))
	for _, p := range azureHZParameters {
		value, source := p.Default, "SystemDefault"
		if v, ok := overrides[p.Name]; ok {
			value, source = v, "UserOverride"
		}
		entry := map[string]any{
			"name": p.Name, "description": p.Description, "value": value, "defaultValue": p.Default, "valueSource": source,
			"dataType": p.DataType, "allowedValues": p.AllowedValues, "isDynamic": p.Dynamic, "isReadOnly": p.ReadOnly,
			"documentationLink": "https://www.postgresql.org/docs/" + version + "/" + p.Doc,
		}
		if p.Unit != "" {
			entry["unit"] = p.Unit
		}
		out = append(out, entry)
	}
	return out
}

// azureHZGroupPoolsLocked returns the pools associated with the parameter
// group. Caller holds providerStorageMu.
func (s *Server) azureHZGroupPoolsLocked(groupID string) []*azureARMResource {
	return s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureHZPoolType) &&
			strings.EqualFold(azureARMString(azureARMObject(rec.Properties["parameterGroup"])["parameterGroupResourceId"]), groupID)
	})
}

// azureHZCapabilities is the regional capability catalog: both PostgreSQL
// versions, and provisioned compute on AMD and Intel generations 1 to 3
// (azureHZMaxGeneration), which pools are checked against.
func azureHZCapabilities() map[string]any {
	var sizes []any
	for _, vCores := range azureHZVCores {
		sizes = append(sizes, map[string]any{"vCores": vCores, "memoryPerVCoreInMb": 8192})
	}
	var processors []any
	for _, name := range azureHZProcessors {
		var generations []any
		for gen := 1; gen <= int(azureHZMaxGeneration); gen++ {
			generations = append(generations, map[string]any{
				"computeGeneration": gen, "recommendedComputeModelType": "Provisioned",
				"supportedComputeModels": []any{map[string]any{"type": "Provisioned", "recommendedVCoreCount": 2, "supportedComputeSizes": sizes}},
			})
		}
		processors = append(processors, map[string]any{"name": name, "supportedGenerations": generations})
	}
	var versions []any
	for _, v := range azureHZVersions {
		versions = append(versions, map[string]any{"name": v, "supportedVersionsToUpgrade": []any{}})
	}
	return map[string]any{
		"provisioningState": "Succeeded",
		"clusterCapabilities": map[string]any{
			"recommendedPostgreSqlVersion": "17", "supportedPostgreSqlVersions": versions, "supportedFeatures": []any{},
		},
		"poolCapabilities": map[string]any{
			"recommendedComputeProcessor": "AMD", "supportedProcessors": processors, "supportedFeatures": []any{},
		},
	}
}
