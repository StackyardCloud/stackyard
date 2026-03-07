package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCommerceConsumerProcurementRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPCommerceConsumerProcurementPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPCommerceConsumerProcurementListOrders(w, r, path) {
			return true
		}
		if handleGCPCommerceConsumerProcurementGetOrder(w, path) {
			return true
		}
		if handleGCPCommerceConsumerProcurementGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPCommerceConsumerProcurementPlaceOrder(w, r, path) {
			return true
		}
		if handleGCPCommerceConsumerProcurementModifyOrder(w, r, path) {
			return true
		}
		if handleGCPCommerceConsumerProcurementCancelOrder(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPCommerceConsumerProcurementPath(path string) bool {
	_, tail, ok := parseGCPCommerceConsumerProcurementBillingAccountTail(path)
	if !ok || len(tail) == 0 {
		return false
	}
	return isGCPCommerceConsumerProcurementOrdersCollectionTail(tail) ||
		isGCPCommerceConsumerProcurementOrderTail(tail) ||
		isGCPCommerceConsumerProcurementOrdersActionTail(tail, "place") ||
		isGCPCommerceConsumerProcurementOrderActionTail(tail, "modify") ||
		isGCPCommerceConsumerProcurementOrderActionTail(tail, "cancel") ||
		isGCPCommerceConsumerProcurementOperationTail(tail)
}

func handleGCPCommerceConsumerProcurementListOrders(w http.ResponseWriter, r *http.Request, path string) bool {
	billingAccount, tail, ok := parseGCPCommerceConsumerProcurementBillingAccountTail(path)
	if !ok || !isGCPCommerceConsumerProcurementOrdersCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPCommerceConsumerProcurementPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCommerceConsumerProcurementOrder(billingAccount, "order-1", "ACTIVE")}
	return respondGCPCommerceConsumerProcurementList(w, "orders", items, pageSize, start, path)
}

func handleGCPCommerceConsumerProcurementGetOrder(w http.ResponseWriter, path string) bool {
	billingAccount, tail, ok := parseGCPCommerceConsumerProcurementBillingAccountTail(path)
	if !ok || !isGCPCommerceConsumerProcurementOrderTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCommerceConsumerProcurementOrder(billingAccount, tail[1], "ACTIVE"))
	return true
}

func handleGCPCommerceConsumerProcurementPlaceOrder(w http.ResponseWriter, r *http.Request, path string) bool {
	billingAccount, tail, ok := parseGCPCommerceConsumerProcurementBillingAccountTail(path)
	if !ok || !isGCPCommerceConsumerProcurementOrdersActionTail(tail, "place") {
		return false
	}
	body, valid := decodeGCPCommerceConsumerProcurementJSONBody(w, r, path)
	if !valid {
		return true
	}
	displayName := strings.TrimSpace(gcpCommerceConsumerProcurementString(body, "displayName"))
	if displayName == "" {
		respondGCPCommerceConsumerProcurementInvalidArgument(w, path, "displayName is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpCommerceConsumerProcurementOperation(billingAccount, "order-1", "placeOrder"))
	return true
}

func handleGCPCommerceConsumerProcurementModifyOrder(w http.ResponseWriter, r *http.Request, path string) bool {
	billingAccount, tail, ok := parseGCPCommerceConsumerProcurementBillingAccountTail(path)
	if !ok || !isGCPCommerceConsumerProcurementOrderActionTail(tail, "modify") {
		return false
	}
	body, valid := decodeGCPCommerceConsumerProcurementJSONBody(w, r, path)
	if !valid {
		return true
	}
	displayName := strings.TrimSpace(gcpCommerceConsumerProcurementString(body, "displayName"))
	if displayName == "" {
		respondGCPCommerceConsumerProcurementInvalidArgument(w, path, "displayName is required")
		return true
	}
	modifications, _ := body["modifications"].([]any)
	if len(modifications) == 0 {
		respondGCPCommerceConsumerProcurementInvalidArgument(w, path, "modifications must include at least one entry")
		return true
	}
	respondJSON(w, http.StatusOK, gcpCommerceConsumerProcurementOperation(billingAccount, tail[1], "modifyOrder"))
	return true
}

func handleGCPCommerceConsumerProcurementCancelOrder(w http.ResponseWriter, r *http.Request, path string) bool {
	billingAccount, tail, ok := parseGCPCommerceConsumerProcurementBillingAccountTail(path)
	if !ok || !isGCPCommerceConsumerProcurementOrderActionTail(tail, "cancel") {
		return false
	}
	body, valid := decodeGCPCommerceConsumerProcurementJSONBody(w, r, path)
	if !valid {
		return true
	}
	if _, exists := body["cancellationPolicy"]; !exists {
		respondGCPCommerceConsumerProcurementInvalidArgument(w, path, "cancellationPolicy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpCommerceConsumerProcurementOperation(billingAccount, tail[1], "cancelOrder"))
	return true
}

func handleGCPCommerceConsumerProcurementGetOperation(w http.ResponseWriter, path string) bool {
	billingAccount, tail, ok := parseGCPCommerceConsumerProcurementBillingAccountTail(path)
	if !ok || !isGCPCommerceConsumerProcurementOperationTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("billingAccounts/%s/orders/%s/operations/%s", billingAccount, tail[1], tail[3]),
		"done": true,
		"response": map[string]any{
			"@type": "type.googleapis.com/google.cloud.commerce.consumer.procurement.v1.Order",
			"name":  fmt.Sprintf("billingAccounts/%s/orders/%s", billingAccount, tail[1]),
		},
	})
	return true
}

func parseGCPCommerceConsumerProcurementBillingAccountTail(path string) (billingAccount string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "billingAccounts" {
		return "", nil, false
	}
	billingAccount = strings.TrimSpace(parts[3])
	if billingAccount == "" {
		return "", nil, false
	}
	return billingAccount, parts[4:], true
}

func isGCPCommerceConsumerProcurementOrdersCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "orders"
}

func isGCPCommerceConsumerProcurementOrderTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "orders" && strings.TrimSpace(tail[1]) != ""
}

func isGCPCommerceConsumerProcurementOrdersActionTail(tail []string, action string) bool {
	if len(tail) != 1 {
		return false
	}
	resource, parsedAction, found := strings.Cut(tail[0], ":")
	return found && resource == "orders" && parsedAction == action
}

func isGCPCommerceConsumerProcurementOrderActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "orders" {
		return false
	}
	orderID, parsedAction, found := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return found && strings.TrimSpace(orderID) != "" && parsedAction == action
}

func isGCPCommerceConsumerProcurementOperationTail(tail []string) bool {
	return len(tail) == 4 &&
		tail[0] == "orders" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "operations" &&
		strings.TrimSpace(tail[3]) != ""
}

func parseGCPCommerceConsumerProcurementPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPCommerceConsumerProcurementInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPCommerceConsumerProcurementInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPCommerceConsumerProcurementList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPCommerceConsumerProcurementInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPCommerceConsumerProcurementJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPCommerceConsumerProcurementInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpCommerceConsumerProcurementString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpCommerceConsumerProcurementOrder(billingAccount, orderID, state string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("billingAccounts/%s/orders/%s", billingAccount, orderID),
		"displayName": "Team commitment order",
		"state":       state,
	}
}

func gcpCommerceConsumerProcurementOperation(billingAccount, orderID, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("billingAccounts/%s/orders/%s/operations/%s", billingAccount, orderID, operationID),
		"done": false,
	}
}

func respondGCPCommerceConsumerProcurementInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
