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

	switch action {
	case "AcceptInboundConnection", "DeleteInboundConnection", "DeleteOutboundConnection", "RejectInboundConnection":
		return map[string]any{"Connection": s.connectionStatus(pathParams)}
	case "AddDataSource", "UpdateDataSource":
		return map[string]any{"Message": "updated"}
	case "AssociatePackage", "DissociatePackage":
		return map[string]any{"DomainPackageDetails": s.domainPackageDetails(domainName)}
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
			"DryRunProgressStatus": map[string]any{},
			"DryRunConfig":         s.domainStatus(domainName),
			"DryRunResults":        map[string]any{},
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
			"Status":         "ACTIVE",
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
