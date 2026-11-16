package server

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Microsoft.StorageActions storage tasks, served by the shared ARM
// handler. No task runs against any blob; conditions are checked for shape
// only.
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: "Microsoft.StorageActions", Type: "storageTasks",
		ValidateName:       azureStorageTaskValidateName,
		Validate:           azureStorageTaskValidate,
		ReadOnlyProperties: azureStorageTaskProperties,
	})
}

var (
	azureStorageTaskNamePattern = regexp.MustCompile(`^[a-z0-9]{3,18}$`)
	azureStorageTaskOperations  = []string{"SetBlobTier", "SetBlobLegalHold", "SetBlobImmutabilityPolicy", "SetBlobExpiry", "SetBlobTags", "DeleteBlob", "UndeleteBlob"}
)

func azureStorageTaskValidateName(name string) string {
	if !azureStorageTaskNamePattern.MatchString(name) {
		return "Storage task names are 3-18 lowercase letters or digits."
	}
	return ""
}

func azureStorageTaskValidate(payload map[string]any) string {
	if identityType := azureARMString(azureARMObject(payload["identity"])["type"]); identityType == "" || identityType == "None" {
		return "A storage task needs a managed identity."
	}
	props := azureARMObject(payload["properties"])
	if _, ok := props["enabled"].(bool); !ok {
		return "properties.enabled is required."
	}
	if strings.TrimSpace(azureARMString(props["description"])) == "" {
		return "properties.description is required."
	}
	action := azureARMObject(props["action"])
	if msg := azureStorageTaskBranch(action["if"], "properties.action.if", true); msg != "" {
		return msg
	}
	if elseBranch := action["else"]; elseBranch != nil {
		return azureStorageTaskBranch(elseBranch, "properties.action.else", false)
	}
	return ""
}

// azureStorageTaskBranch checks an if or else branch: a condition (if only)
// and at least one known operation.
func azureStorageTaskBranch(raw any, at string, needsCondition bool) string {
	branch := azureARMObject(raw)
	condition := strings.TrimSpace(azureARMString(branch["condition"]))
	if needsCondition && (!strings.HasPrefix(condition, "[[") || !strings.HasSuffix(condition, "]]")) {
		return at + ".condition must be an expression in [[...]]."
	}
	operations, _ := branch["operations"].([]any)
	if len(operations) == 0 {
		return at + ".operations must list at least one operation."
	}
	for i, rawOperation := range operations {
		operation := azureARMObject(rawOperation)
		if !slices.Contains(azureStorageTaskOperations, azureARMString(operation["name"])) {
			return fmt.Sprintf("%s.operations[%d].name must be one of %s.", at, i, strings.Join(azureStorageTaskOperations, ", "))
		}
		if v, present := operation["onSuccess"]; present && v != "continue" {
			return fmt.Sprintf("%s.operations[%d].onSuccess must be continue.", at, i)
		}
		if v, present := operation["onFailure"]; present && v != "break" {
			return fmt.Sprintf("%s.operations[%d].onFailure must be break.", at, i)
		}
	}
	return ""
}

// azureStorageTaskProperties reports when the task was created.
// ponytail: taskVersion stays 1; count versions in StateProperties if a
// client relies on it changing.
func azureStorageTaskProperties(rec *azureARMResource) map[string]any {
	return map[string]any{"creationTimeInUtc": rec.CreatedAt.Format("2006-01-02T15:04:05Z"), "taskVersion": 1}
}
