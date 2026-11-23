package server

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

type openSearchStore struct {
	mu                 sync.Mutex
	defaultDomainName  string
	defaultAccountID   string
	defaultVPCEndpoint string
	defaultPackageID   string
	defaultConnection  string

	// Shared with the legacy es API (opensearch_domains.go, elasticsearchservice_store.go).
	domains map[string]*osDomain
	esl     eslState
}

func newOpenSearchStore() *openSearchStore {
	return &openSearchStore{
		defaultDomainName:  "stackyard-opensearch-domain",
		defaultAccountID:   "123456789012",
		defaultVPCEndpoint: "vpce-1234567890abcdef0",
		defaultPackageID:   "pkg-1234567890abcdef0",
		defaultConnection:  "connection-1234567890abcdef0",
	}
}

func (s *openSearchStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	domainName := s.resolveDomainName(payload, pathParams)
	principal := s.resolvePrincipal(payload)
	if out, ok := s.osHandleDomainAction(action, domainName, payload); ok {
		return out
	}

	switch action {
	case "AcceptInboundConnection", "DeleteInboundConnection", "DeleteOutboundConnection", "RejectInboundConnection":
		return map[string]any{"Connection": s.connectionStatus(pathParams)}
	case "AddDataSource", "UpdateDataSource":
		return map[string]any{"Message": "updated"}
	case "AssociatePackage", "DissociatePackage":
		return map[string]any{"DomainPackageDetails": s.domainPackageDetails(domainName)}
	case "AssociatePackages", "DissociatePackages":
		return map[string]any{"DomainPackageDetailsList": s.domainPackageDetailsList(action, domainName, payload)}
	case "CreateApplication", "GetApplication", "UpdateApplication":
		return s.application(action, payload, pathParams)
	case "ListApplications":
		app := s.application("GetApplication", payload, pathParams)
		delete(app, "iamIdentityCenterOptions")
		delete(app, "dataSources")
		delete(app, "appConfigs")
		delete(app, "kmsKeyArn")
		return map[string]any{"ApplicationSummaries": []any{app}}
	case "GetDefaultApplicationSetting", "PutDefaultApplicationSetting":
		return map[string]any{"applicationArn": opensearchFirstNonEmpty(
			opensearchPayloadString(payload, "applicationArn"), s.applicationARN("stackyard-application"))}
	case "GetDirectQueryDataSource":
		name := s.resolveDataSourceName(payload, pathParams)
		return map[string]any{
			"DataSourceName": name,
			"DataSourceType": s.directQueryDataSourceType(),
			"Description":    "stackyard direct query data source",
			"OpenSearchArns": []any{},
			"DataSourceArn":  s.directQueryDataSourceARN(name),
		}
	case "ListDirectQueryDataSources":
		name := "stackyard-datasource"
		return map[string]any{"DirectQueryDataSources": []any{map[string]any{
			"DataSourceName": name,
			"DataSourceType": s.directQueryDataSourceType(),
			"Description":    "stackyard direct query data source",
			"OpenSearchArns": []any{},
			"DataSourceArn":  s.directQueryDataSourceARN(name),
			"TagList":        []any{},
		}}}
	case "CreateIndex":
		return map[string]any{"Status": "CREATED"}
	case "UpdateIndex":
		return map[string]any{"Status": "UPDATED"}
	case "DeleteIndex":
		return map[string]any{"Status": "DELETED"}
	case "GetIndex":
		return map[string]any{"IndexSchema": map[string]any{}}
	case "UpdatePackageScope":
		users := []any{}
		if list, ok := payload["PackageUserList"].([]any); ok {
			users = list
		}
		return map[string]any{
			"PackageID":       s.resolvePackageID(payload, pathParams),
			"Operation":       opensearchFirstNonEmpty(opensearchPayloadString(payload, "Operation"), "ADD"),
			"PackageUserList": users,
		}
	case "AuthorizeVpcEndpointAccess":
		return map[string]any{
			"AuthorizedPrincipal": map[string]any{
				"PrincipalType": "AWS_ACCOUNT",
				"Principal":     principal,
			},
		}
	case "CancelDomainConfigChange":
		return map[string]any{
			"CancelledChangeIds":        []any{},
			"CancelledChangeProperties": []any{},
			"DryRun":                    false,
		}
	case "CancelServiceSoftwareUpdate", "StartServiceSoftwareUpdate":
		return map[string]any{"ServiceSoftwareOptions": map[string]any{}}
	case "CreateDomain", "DeleteDomain":
		return map[string]any{"DomainStatus": s.domainStatus(domainName)}
	case "CreatePackage", "DeletePackage", "UpdatePackage":
		return map[string]any{"PackageDetails": s.packageDetails(payload, pathParams)}
	case "CreateVpcEndpoint", "UpdateVpcEndpoint":
		return map[string]any{"VpcEndpoint": map[string]any{}}
	case "DeleteDataSource":
		return map[string]any{"Message": "deleted"}
	case "DeleteVpcEndpoint":
		return map[string]any{"VpcEndpointSummary": map[string]any{}}
	case "DescribeDomain":
		return map[string]any{"DomainStatus": s.domainStatus(domainName)}
	case "DescribeDomainAutoTunes":
		return map[string]any{"AutoTunes": []any{}, "NextToken": ""}
	case "DescribeDomainChangeProgress":
		return map[string]any{"ChangeProgressStatus": map[string]any{}}
	case "DescribeDomainConfig", "UpdateDomainConfig":
		return map[string]any{"DomainConfig": map[string]any{}}
	case "DescribeDomainHealth":
		return map[string]any{
			"DomainState":                 "Active",
			"ClusterHealth":               "Green",
			"AvailabilityZoneCount":       "1",
			"ActiveAvailabilityZoneCount": "1",
			"DataNodeCount":               "1",
		}
	case "DescribeDomainNodes":
		return map[string]any{"DomainNodesStatusList": []any{}}
	case "DescribeDomains":
		return map[string]any{"DomainStatusList": []any{s.domainStatus(domainName)}}
	case "DescribeDryRunProgress":
		return map[string]any{
			"DryRunProgressStatus": map[string]any{
				"DryRunId":     "00000000-0000-0000-0000-000000000001",
				"DryRunStatus": "SUCCEEDED",
				"CreationDate": "2026-03-13T00:00:00Z",
				"UpdateDate":   "2026-03-13T00:05:00Z",
			},
			"DryRunConfig":  s.domainStatus(domainName),
			"DryRunResults": map[string]any{},
		}
	case "DescribeInboundConnections", "DescribeOutboundConnections":
		return map[string]any{"Connections": []any{}, "NextToken": ""}
	case "DescribeInstanceTypeLimits":
		return map[string]any{"LimitsByRole": map[string]any{"data": map[string]any{}}}
	case "DescribePackages":
		return map[string]any{"PackageDetailsList": []any{}, "NextToken": ""}
	case "DescribeReservedInstanceOfferings":
		return map[string]any{"ReservedInstanceOfferings": []any{}, "NextToken": ""}
	case "DescribeReservedInstances":
		return map[string]any{"ReservedInstances": []any{}, "NextToken": ""}
	case "DescribeVpcEndpoints":
		return map[string]any{
			"VpcEndpoints":      []any{},
			"VpcEndpointErrors": []any{},
		}
	case "GetCompatibleVersions":
		return map[string]any{"CompatibleVersions": []any{}}
	case "GetDataSource":
		return map[string]any{
			"DataSourceType": s.dataSourceType(),
			"Name":           s.resolveDataSourceName(payload, pathParams),
			"Description":    "stackyard data source",
		}
	case "GetDomainMaintenanceStatus":
		return map[string]any{"Status": "COMPLETED"}
	case "GetPackageVersionHistory":
		return map[string]any{
			"PackageID":                 s.resolvePackageID(payload, pathParams),
			"PackageVersionHistoryList": []any{},
			"NextToken":                 "",
		}
	case "GetUpgradeHistory":
		return map[string]any{"UpgradeHistories": []any{}, "NextToken": ""}
	case "GetUpgradeStatus":
		return map[string]any{
			"UpgradeStep": "PRE_UPGRADE_CHECK",
			"StepStatus":  "SUCCEEDED",
			"UpgradeName": "stackyard-upgrade",
		}
	case "ListDataSources":
		return map[string]any{"DataSources": []any{}}
	case "ListDomainMaintenances":
		return map[string]any{"DomainMaintenances": []any{}, "NextToken": ""}
	case "ListDomainNames":
		return map[string]any{"DomainNames": []any{map[string]any{"DomainName": domainName}}}
	case "ListDomainsForPackage", "ListPackagesForDomain":
		return map[string]any{"DomainPackageDetailsList": []any{}, "NextToken": ""}
	case "ListInstanceTypeDetails":
		return map[string]any{"InstanceTypeDetails": []any{}, "NextToken": ""}
	case "ListScheduledActions":
		return map[string]any{"ScheduledActions": []any{}, "NextToken": ""}
	case "ListTags":
		return map[string]any{"TagList": []any{}}
	case "ListVersions":
		return map[string]any{"Versions": []any{}, "NextToken": ""}
	case "ListVpcEndpointAccess":
		return map[string]any{
			"AuthorizedPrincipalList": []any{
				map[string]any{
					"PrincipalType": "AWS_ACCOUNT",
					"Principal":     principal,
				},
			},
			"NextToken": "",
		}
	case "ListVpcEndpoints", "ListVpcEndpointsForDomain":
		return map[string]any{
			"VpcEndpointSummaryList": []any{},
			"NextToken":              "",
		}
	case "PurchaseReservedInstanceOffering":
		return map[string]any{
			"ReservedInstanceId": "ri-1234567890abcdef0",
			"ReservationName":    "stackyard-reservation",
		}
	case "RevokeVpcEndpointAccess":
		return map[string]any{}
	case "StartDomainMaintenance":
		return map[string]any{"MaintenanceId": "maintenance-1234567890abcdef0"}
	case "UpdateScheduledAction":
		return map[string]any{"ScheduledAction": s.scheduledAction(payload)}
	case "UpgradeDomain":
		return map[string]any{
			"UpgradeId":             "upgrade-1234567890abcdef0",
			"DomainName":            domainName,
			"TargetVersion":         "OpenSearch_2.11",
			"PerformCheckOnly":      false,
			"AdvancedOptions":       map[string]any{},
			"ChangeProgressDetails": map[string]any{},
		}
	default:
		return map[string]any{}
	}
}

func (s *openSearchStore) connectionStatus(pathParams map[string]string) map[string]any {
	connectionID := opensearchPathParam(pathParams, "ConnectionId")
	if connectionID == "" {
		connectionID = s.defaultConnection
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{
		"ConnectionId":    connectionID,
		"ConnectionAlias": "stackyard-connection",
		"ConnectionStatus": map[string]any{
			"StatusCode": "APPROVED",
			"Message":    "stackyard approved connection",
		},
		"CreatedAt": now,
	}
}

func (s *openSearchStore) domainPackageDetails(domainName string) map[string]any {
	return map[string]any{
		"DomainName": domainName,
		"PackageID":  s.defaultPackageID,
	}
}

// domainPackageDetailsList echoes the packages named in an Associate/DissociatePackages
// request; AssociatePackages lists objects with PackageID, DissociatePackages lists IDs.
func (s *openSearchStore) domainPackageDetailsList(action, domainName string, payload map[string]any) []any {
	status := "ACTIVE"
	if action == "DissociatePackages" {
		status = "DISSOCIATING"
	}
	out := []any{}
	list, _ := payload["PackageList"].([]any)
	for _, item := range list {
		id := opensearchAnyToString(item)
		if m, ok := item.(map[string]any); ok {
			id = opensearchPayloadString(m, "PackageID")
		}
		if id == "" {
			continue
		}
		out = append(out, map[string]any{
			"PackageID":           id,
			"DomainName":          domainName,
			"DomainPackageStatus": status,
			"LastUpdated":         time.Now().UTC().Format(time.RFC3339),
		})
	}
	return out
}

func (s *openSearchStore) applicationARN(id string) string {
	return fmt.Sprintf("arn:aws:opensearch:us-east-1:%s:application/%s", s.defaultAccountID, id)
}

func (s *openSearchStore) directQueryDataSourceARN(name string) string {
	return fmt.Sprintf("arn:aws:opensearch:us-east-1:%s:datasource/%s", s.defaultAccountID, name)
}

func (s *openSearchStore) directQueryDataSourceType() map[string]any {
	return map[string]any{"CloudWatchLog": map[string]any{
		"RoleArn": "arn:aws:iam::123456789012:role/stackyard-opensearch-direct-query",
	}}
}

// application answers Create/Get/UpdateApplication by echoing the request.
// ponytail: stateless like the rest of this store; add an app map if callers need persistence.
func (s *openSearchStore) application(action string, payload map[string]any, pathParams map[string]string) map[string]any {
	id := opensearchFirstNonEmpty(opensearchPathParam(pathParams, "id"), "stackyard-application")
	now := time.Now().UTC().Format(time.RFC3339)
	out := map[string]any{
		"id":        id,
		"arn":       s.applicationARN(id),
		"name":      opensearchFirstNonEmpty(opensearchPayloadString(payload, "name"), id),
		"createdAt": now,
	}
	for _, key := range []string{"dataSources", "appConfigs", "iamIdentityCenterOptions"} {
		if v, ok := payload[key]; ok {
			out[key] = v
		}
	}
	switch action {
	case "CreateApplication":
		if tags, ok := payload["tagList"]; ok {
			out["tagList"] = tags
		}
		if kms := opensearchPayloadString(payload, "kmsKeyArn"); kms != "" {
			out["kmsKeyArn"] = kms
		}
	case "GetApplication":
		out["status"] = "ACTIVE"
		out["endpoint"] = id + ".opensearch.stackyard.local"
		out["lastUpdatedAt"] = now
	case "UpdateApplication":
		out["lastUpdatedAt"] = now
	}
	return out
}

func (s *openSearchStore) domainStatus(domainName string) map[string]any {
	if strings.TrimSpace(domainName) == "" {
		domainName = s.defaultDomainName
	}
	return map[string]any{
		"DomainId":      fmt.Sprintf("%s/%s", s.defaultAccountID, domainName),
		"DomainName":    domainName,
		"ARN":           fmt.Sprintf("arn:aws:es:us-east-1:%s:domain/%s", s.defaultAccountID, domainName),
		"ClusterConfig": map[string]any{},
	}
}

func (s *openSearchStore) packageDetails(payload map[string]any, pathParams map[string]string) map[string]any {
	return map[string]any{
		"PackageID":          s.resolvePackageID(payload, pathParams),
		"PackageName":        opensearchFirstNonEmpty(opensearchPayloadString(payload, "PackageName"), "stackyard-package"),
		"PackageDescription": "stackyard package",
		"PackageStatus":      "AVAILABLE",
	}
}

func (s *openSearchStore) dataSourceType() map[string]any {
	return map[string]any{
		"S3GlueDataCatalog": map[string]any{
			"RoleArn": "arn:aws:iam::123456789012:role/stackyard-opensearch-data-source",
		},
	}
}

func (s *openSearchStore) scheduledAction(payload map[string]any) map[string]any {
	actionID := opensearchFirstNonEmpty(
		opensearchPayloadString(payload, "ActionID"),
		opensearchPayloadString(payload, "ActionId"),
		"scheduled-action-000001",
	)
	actionType := opensearchFirstNonEmpty(opensearchPayloadString(payload, "ActionType"), "SERVICE_SOFTWARE_UPDATE")
	return map[string]any{
		"Id":            actionID,
		"Type":          actionType,
		"Severity":      "HIGH",
		"ScheduledTime": time.Now().UTC().Add(time.Hour).UnixMilli(),
		"Description":   "stackyard scheduled action",
		"ScheduledBy":   "SYSTEM",
		"Status":        "PENDING_UPDATE",
		"Mandatory":     false,
		"Cancellable":   true,
	}
}

func (s *openSearchStore) resolveDataSourceName(payload map[string]any, pathParams map[string]string) string {
	if value := opensearchPathParam(pathParams, "DataSourceName"); value != "" {
		return value
	}
	if value := opensearchPayloadString(payload, "DataSourceName"); value != "" {
		return value
	}
	if value := opensearchPayloadString(payload, "Name"); value != "" {
		return value
	}
	return "stackyard-datasource"
}

func (s *openSearchStore) resolveDomainName(payload map[string]any, pathParams map[string]string) string {
	if value := opensearchPathParam(pathParams, "DomainName"); value != "" {
		return value
	}
	if value := opensearchPayloadString(payload, "DomainName"); value != "" {
		return value
	}
	if value := opensearchPayloadString(payload, "domainName"); value != "" {
		return value
	}
	return s.defaultDomainName
}

func (s *openSearchStore) resolvePackageID(payload map[string]any, pathParams map[string]string) string {
	if value := opensearchPathParam(pathParams, "PackageID"); value != "" {
		return value
	}
	if value := opensearchPayloadString(payload, "PackageID"); value != "" {
		return value
	}
	if value := opensearchPayloadString(payload, "PackageId"); value != "" {
		return value
	}
	return s.defaultPackageID
}

func (s *openSearchStore) resolvePrincipal(payload map[string]any) string {
	for _, key := range []string{"AuthorizedPrincipal", "Account", "Principal", "account"} {
		if value := opensearchPayloadString(payload, key); value != "" {
			return value
		}
	}
	return s.defaultAccountID
}

func opensearchPathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	if value, ok := pathParams[key]; ok {
		return strings.TrimSpace(value)
	}
	for k, value := range pathParams {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func opensearchPayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		return strings.TrimSpace(opensearchAnyToString(value))
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(opensearchAnyToString(value))
		}
	}
	return ""
}

func opensearchAnyToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func opensearchFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
