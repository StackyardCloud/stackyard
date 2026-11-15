package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.Maps accounts, served by the shared ARM handler: keys and SAS
// tokens for the Maps data plane, which Stackyard does not serve.
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: "Microsoft.Maps",
		Type:      "accounts",
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"uniqueId": azureMapsUniqueID(rec)}
		},
		Validate: azureMapsValidateAccount,
		Actions:  map[string]azureARMAction{"listkeys": azureMapsListKeys, "listsas": azureMapsListSas},
	})
}

func azureMapsValidateAccount(payload map[string]any) string {
	if !slices.Contains([]string{"G2", "S0", "S1"}, azureARMString(azureARMObject(payload["sku"])["name"])) {
		return "sku.name must be G2, S0 or S1."
	}
	if kind, present := payload["kind"]; present && kind != "Gen1" && kind != "Gen2" {
		return "kind must be Gen1 or Gen2."
	}
	return ""
}

// azureMapsKey returns a URL-safe key like real Maps keys, which clients put
// in a subscription-key query parameter.
func azureMapsKey(rec *azureARMResource, which string) string {
	return strings.NewReplacer("+", "-", "/", "_", "=", "").Replace(azureARMDerivedKey(rec.ID, "maps-"+which))
}

func azureMapsListKeys(rec *azureARMResource, _ map[string]any) (int, any) {
	created := rec.CreatedAt.Format(time.RFC3339)
	return http.StatusOK, map[string]any{
		"primaryKey":              azureMapsKey(rec, "primary"),
		"secondaryKey":            azureMapsKey(rec, "secondary"),
		"primaryKeyLastUpdated":   created,
		"secondaryKeyLastUpdated": created,
	}
}

// azureMapsListSas returns an account SAS token: the base64 claims signed
// with the chosen account key.
func azureMapsListSas(rec *azureARMResource, payload map[string]any) (int, any) {
	invalid := func(msg string) (int, any) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	signingKey := azureARMString(payload["signingKey"])
	if !slices.Contains([]string{"primaryKey", "secondaryKey", "managedIdentity"}, signingKey) {
		return invalid("signingKey must be primaryKey, secondaryKey or managedIdentity.")
	}
	if azureARMString(payload["principalId"]) == "" {
		return invalid("principalId is required.")
	}
	rate, ok := payload["maxRatePerSecond"].(float64)
	if !ok || rate != float64(int(rate)) || rate < 1 || rate > 500 {
		return invalid("maxRatePerSecond must be an integer from 1 to 500.")
	}
	start, startErr := time.Parse(time.RFC3339, azureARMString(payload["start"]))
	expiry, expiryErr := time.Parse(time.RFC3339, azureARMString(payload["expiry"]))
	if startErr != nil || expiryErr != nil || !expiry.After(start) {
		return invalid("start and expiry must be RFC 3339 times with expiry after start.")
	}
	claims, _ := json.Marshal(map[string]any{
		"account": azureMapsUniqueID(rec), "principalId": payload["principalId"], "regions": payload["regions"],
		"maxRatePerSecond": int(rate), "start": start.UTC().Format(time.RFC3339), "expiry": expiry.UTC().Format(time.RFC3339),
	})
	mac := hmac.New(sha256.New, []byte(azureMapsKey(rec, signingKey)))
	mac.Write(claims)
	token := base64.RawURLEncoding.EncodeToString(claims) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return http.StatusOK, map[string]any{"accountSasToken": token}
}

func azureMapsUniqueID(rec *azureARMResource) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()
}
