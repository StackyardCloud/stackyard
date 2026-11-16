package server

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/netip"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Nginx.NginxPlus (NGINXaaS for Azure) deployments with their configuration,
// certificates, dataplane API keys and WAF policies, served by the shared
// ARM handler. No NGINX runs: configuration analysis only checks that file
// contents are base64 and the root file is one of them, and certificates
// are never fetched from Key Vault.
func init() {
	const ns = "Nginx.NginxPlus"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "nginxDeployments",
		ValidateName:       azureNginxName,
		Validate:           azureNginxDeploymentValidate,
		ReadOnlyProperties: azureNginxDeploymentProperties,
		Actions: map[string]azureARMAction{
			"listdefaultwafpolicies": func(*azureARMResource, map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"value": azureNginxDefaultWafPolicies}
			},
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "nginxDeployments/configurations", Proxy: true, NoPatch: true,
		ValidateName: azureSingletonName("default"),
		Validate: func(payload map[string]any) string {
			return azureNginxConfigValidate(azureARMObject(payload["properties"]), "properties")
		},
		ReadOnlyProperties:  azureNginxConfigurationProperties,
		WriteOnlyProperties: []string{"protectedFiles[].content"},
		Actions: map[string]azureARMAction{
			"analyze": func(_ *azureARMResource, payload map[string]any) (int, any) {
				config, ok := payload["config"].(map[string]any)
				if !ok {
					return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "config is required.")
				}
				return http.StatusOK, azureNginxAnalysis(azureNginxConfigValidate(config, "config"))
			},
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "nginxDeployments/certificates", Proxy: true, KeepLocation: true, NoPatch: true,
		ValidateName:       azureNginxName,
		Validate:           azureNginxCertificateValidate,
		ReadOnlyProperties: azureNginxCertificateProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "nginxDeployments/apiKeys", Proxy: true, NoPatch: true,
		ValidateName:        azureNginxName,
		Validate:            azureNginxAPIKeyValidate,
		WriteOnlyProperties: []string{"secretText"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			// The hint is the secret's first three characters.
			secret := []rune(azureARMString(rec.Properties["secretText"]))
			return map[string]any{"hint": string(secret[:min(3, len(secret))])}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "nginxDeployments/wafPolicies", Proxy: true, NoPatch: true,
		ValidateName: azureNginxName,
		Validate:     azureNginxWafPolicyValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			at := rec.UpdatedAt.Format("2006-01-02T15:04:05Z")
			return map[string]any{
				"compilingState": map[string]any{"code": "Succeeded", "displayStatus": "Compiled", "time": at},
				"applyingState":  map[string]any{"code": "Succeeded", "displayStatus": "Applied", "time": at},
			}
		},
		Actions: map[string]azureARMAction{
			"analyzewafpolicy": func(_ *azureARMResource, payload map[string]any) (int, any) {
				errors := []any{}
				if _, err := base64.StdEncoding.DecodeString(azureARMString(payload["content"])); err != nil || azureARMString(payload["content"]) == "" {
					errors = append(errors, map[string]any{"code": "InvalidContent", "field": "content", "message": "content must be base64-encoded JSON."})
				}
				status := "SUCCEEDED"
				if len(errors) > 0 {
					status = "FAILED"
				}
				return http.StatusOK, map[string]any{"status": status, "data": map[string]any{"errors": errors}}
			},
		},
	})
}

var (
	azureNginxNamePattern  = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9-]{0,28}[a-zA-Z0-9]|[a-zA-Z0-9])$`)
	azureNginxEmailPattern = regexp.MustCompile(`^$|^[A-Za-z0-9._%+-]+@(?:[A-Za-z0-9-]+\.)+[A-Za-z]{2,}$`)
	// azureNginxDefaultWafPolicies are the policies NGINX App Protect ships.
	azureNginxDefaultWafPolicies = []any{
		map[string]any{"filepath": "/etc/app_protect/conf/NginxDefaultPolicy.json", "content": base64.StdEncoding.EncodeToString([]byte(`{"policy":{"name":"app_protect_default_policy","template":{"name":"POLICY_TEMPLATE_NGINX_BASE"}}}`))},
		map[string]any{"filepath": "/etc/app_protect/conf/NginxStrictPolicy.json", "content": base64.StdEncoding.EncodeToString([]byte(`{"policy":{"name":"app_protect_strict_policy","template":{"name":"POLICY_TEMPLATE_NGINX_BASE"},"enforcementMode":"blocking"}}`))},
	}
)

func azureNginxName(name string) string {
	if !azureNginxNamePattern.MatchString(name) {
		return "Names must be 1 to 30 letters, digits or hyphens, starting and ending with a letter or digit."
	}
	return ""
}

func azureNginxDeploymentValidate(payload map[string]any) string {
	if strings.TrimSpace(azureARMString(azureARMObject(payload["sku"])["name"])) == "" {
		return "sku.name is required, such as standardv3_Monthly."
	}
	props := azureARMObject(payload["properties"])
	network := azureARMObject(props["networkProfile"])
	if !azureNetworkIDOf(map[string]any{"id": azureARMObject(network["networkInterfaceConfiguration"])["subnetId"]}, "virtualnetworks/subnets") {
		return "properties.networkProfile.networkInterfaceConfiguration.subnetId must be a subnet ID."
	}
	frontend := azureARMObject(network["frontEndIPConfiguration"])
	public, private := azureARMSlice(frontend["publicIPAddresses"]), azureARMSlice(frontend["privateIPAddresses"])
	if len(public)+len(private) == 0 {
		return "properties.networkProfile.frontEndIPConfiguration needs a public or private IP address."
	}
	for i, raw := range public {
		if !azureNetworkIDOf(raw, "publicipaddresses") {
			return fmt.Sprintf("properties.networkProfile.frontEndIPConfiguration.publicIPAddresses[%d].id must be a public IP address ID.", i)
		}
	}
	for i, raw := range private {
		address := azureARMObject(raw)
		method := azureARMString(address["privateIPAllocationMethod"])
		_, err := netip.ParseAddr(azureARMString(address["privateIPAddress"]))
		switch {
		case method != "" && method != "Static" && method != "Dynamic":
			return fmt.Sprintf("properties.networkProfile.frontEndIPConfiguration.privateIPAddresses[%d].privateIPAllocationMethod must be Static or Dynamic.", i)
		case method == "Static" && err != nil:
			return fmt.Sprintf("properties.networkProfile.frontEndIPConfiguration.privateIPAddresses[%d].privateIPAddress must be an IP address for Static allocation.", i)
		}
	}
	scaling := azureARMObject(props["scalingProperties"])
	profiles := azureARMSlice(azureARMObject(scaling["autoScaleSettings"])["profiles"])
	if _, fixed := scaling["capacity"]; fixed && len(profiles) > 0 {
		return "properties.scalingProperties sets either capacity or autoScaleSettings, not both."
	}
	if raw, fixed := scaling["capacity"]; fixed && !azureWholeNumber(raw, 0, 1<<31-1) {
		return "properties.scalingProperties.capacity must be a non-negative integer."
	}
	for i, raw := range profiles {
		profile := azureARMObject(raw)
		capacity := azureARMObject(profile["capacity"])
		low, high := capacity["min"], capacity["max"]
		if azureARMString(profile["name"]) == "" || !azureWholeNumber(low, 0, 1<<31-1) || !azureWholeNumber(high, 0, 1<<31-1) || low.(float64) > high.(float64) {
			return fmt.Sprintf("properties.scalingProperties.autoScaleSettings.profiles[%d] needs a name and integer capacity min no greater than max.", i)
		}
	}
	if raw, present := azureARMObject(props["autoUpgradeProfile"])["upgradeChannel"]; present && raw != "stable" && raw != "preview" {
		return "properties.autoUpgradeProfile.upgradeChannel must be stable or preview."
	}
	if raw, present := azureARMObject(props["userProfile"])["preferredEmail"]; present && !azureNginxEmailPattern.MatchString(azureARMString(raw)) {
		return "properties.userProfile.preferredEmail must be an email address."
	}
	waf := azureARMObject(azureARMObject(props["nginxAppProtect"])["webApplicationFirewallSettings"])
	if raw, present := waf["activationState"]; present && raw != "Enabled" && raw != "Disabled" {
		return "properties.nginxAppProtect.webApplicationFirewallSettings.activationState must be Enabled or Disabled."
	}
	return ""
}

// azureNginxDeploymentProperties reports the NGINX Plus version, the
// frontend address (the first private one, or a stable public one) and the
// dataplane endpoint.
func azureNginxDeploymentProperties(rec *azureARMResource) map[string]any {
	sum := sha256.Sum256([]byte(azureARMKey(rec.ID)))
	frontend := azureARMObject(azureARMObject(rec.Properties["networkProfile"])["frontEndIPConfiguration"])
	address := fmt.Sprintf("20.%d.%d.%d", sum[0], sum[1], 1+int(sum[2])%254)
	if private := azureARMSlice(frontend["privateIPAddresses"]); len(private) > 0 {
		if ip := azureARMString(azureARMObject(private[0])["privateIPAddress"]); ip != "" {
			address = ip
		}
	}
	props := map[string]any{
		"nginxVersion":         "1.27.4 (nginx-plus-r34-p1)",
		"ipAddress":            address,
		"dataplaneApiEndpoint": fmt.Sprintf("https://%s-%x.%s.nginxaas.net", strings.ToLower(rec.Name), sum[3:6], azureLocationHost(rec.Location)),
	}
	if appProtect, present := rec.Properties["nginxAppProtect"]; present {
		props["nginxAppProtect"] = azureARMMerge(azureARMObject(appProtect), map[string]any{"webApplicationFirewallStatus": map[string]any{
			"wafRelease":        "5.6.0",
			"componentVersions": map[string]any{"wafEngineVersion": "11.4.0", "wafNginxVersion": "5.6.0"},
		}})
	}
	return props
}

// azureNginxConfigValidate checks configuration properties at path: a root
// file, and files and protected files with a virtual path and base64
// content, or a package.
func azureNginxConfigValidate(props map[string]any, path string) string {
	files, protected := azureARMSlice(props["files"]), azureARMSlice(props["protectedFiles"])
	_, hasPackage := azureARMObject(props["package"])["data"]
	if len(files)+len(protected) == 0 && !hasPackage {
		return path + " needs files, protectedFiles or a package."
	}
	if hasPackage {
		if _, err := base64.StdEncoding.DecodeString(azureARMString(azureARMObject(props["package"])["data"])); err != nil {
			return path + ".package.data must be base64."
		}
	}
	root := azureARMString(props["rootFile"])
	if root == "" {
		return path + ".rootFile is required."
	}
	var paths []string
	for i, raw := range slices.Concat(files, protected) {
		file := azureARMObject(raw)
		virtual := azureARMString(file["virtualPath"])
		if !strings.HasPrefix(virtual, "/") {
			return fmt.Sprintf("%s file %d needs an absolute virtualPath.", path, i)
		}
		if _, err := base64.StdEncoding.DecodeString(azureARMString(file["content"])); err != nil {
			return fmt.Sprintf("%s file %s must have base64 content.", path, virtual)
		}
		paths = append(paths, virtual)
	}
	if !hasPackage && !slices.Contains(paths, root) {
		return path + ".rootFile must be one of the files' virtual paths."
	}
	return ""
}

// azureNginxConfigurationProperties records each protected file's content
// hash; the content itself is write-only.
func azureNginxConfigurationProperties(rec *azureARMResource) map[string]any {
	protected := azureARMSlice(rec.Properties["protectedFiles"])
	if protected == nil {
		return nil
	}
	out := make([]any, 0, len(protected))
	for _, raw := range protected {
		file := azureARMObject(raw)
		if content, ok := file["content"].(string); ok {
			sum := sha256.Sum256([]byte(content))
			file = azureARMMerge(file, map[string]any{"contentHash": hex.EncodeToString(sum[:])})
		}
		out = append(out, file)
	}
	return map[string]any{"protectedFiles": out}
}

func azureNginxAnalysis(msg string) map[string]any {
	if msg == "" {
		return map[string]any{"status": "SUCCEEDED", "data": map[string]any{"errors": []any{}, "diagnostics": []any{}}}
	}
	return map[string]any{"status": "FAILED", "data": map[string]any{"errors": []any{map[string]any{"id": "config", "directive": "", "description": msg, "file": "", "line": 0, "message": msg, "rule": "syntax"}}, "diagnostics": []any{}}}
}

func azureNginxCertificateValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"keyVirtualPath", "certificateVirtualPath"} {
		if !strings.HasPrefix(azureARMString(props[key]), "/") {
			return "properties." + key + " must be an absolute path."
		}
	}
	if !strings.Contains(strings.ToLower(azureARMString(props["keyVaultSecretId"])), ".vault.azure.net/secrets/") {
		return "properties.keyVaultSecretId must be a Key Vault secret URL."
	}
	return ""
}

// azureNginxCertificateProperties reports a stable thumbprint and the
// secret version, from the URL when it names one.
func azureNginxCertificateProperties(rec *azureARMResource) map[string]any {
	secret := azureARMString(rec.Properties["keyVaultSecretId"])
	sum := sha1.Sum([]byte(strings.ToLower(secret)))
	version := hex.EncodeToString(sum[:16])
	if parts := strings.Split(strings.TrimRight(secret, "/"), "/secrets/"); len(parts) == 2 && strings.Contains(parts[1], "/") {
		version = parts[1][strings.Index(parts[1], "/")+1:]
	}
	return map[string]any{
		"sha1Thumbprint":        strings.ToUpper(hex.EncodeToString(sum[:])),
		"keyVaultSecretVersion": version,
		"keyVaultSecretCreated": rec.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func azureNginxAPIKeyValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if len(azureARMString(props["secretText"])) < 12 {
		return "properties.secretText must be at least 12 characters."
	}
	if raw, present := props["endDateTime"]; present && !azureNginxDateTime(raw) {
		return "properties.endDateTime must be an RFC 3339 date-time."
	}
	return ""
}

func azureNginxDateTime(raw any) bool {
	_, err := time.Parse(time.RFC3339, azureARMString(raw))
	return err == nil
}

func azureNginxWafPolicyValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !strings.HasPrefix(azureARMString(props["filepath"]), "/") {
		return "properties.filepath must be an absolute path."
	}
	if _, err := base64.StdEncoding.DecodeString(azureARMString(props["content"])); err != nil || azureARMString(props["content"]) == "" {
		return "properties.content must be the base64-encoded policy."
	}
	return ""
}
