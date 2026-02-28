package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type redshiftServerlessStore struct {
	mu sync.Mutex

	nextID int64

	namespaces               map[string]*redshiftServerlessNamespace
	workgroups               map[string]*redshiftServerlessWorkgroup
	snapshots                map[string]*redshiftServerlessSnapshot
	recoveryPoints           map[string]*redshiftServerlessRecoveryPoint
	tableRestores            map[string]*redshiftServerlessTableRestoreStatus
	endpointAccesses         map[string]*redshiftServerlessEndpointAccess
	scheduledActions         map[string]*redshiftServerlessScheduledAction
	usageLimits              map[string]*redshiftServerlessUsageLimit
	reservations             map[string]*redshiftServerlessReservation
	snapshotCopyConfigs      map[string]*redshiftServerlessSnapshotCopyConfiguration
	customDomainAssociations map[string]*redshiftServerlessCustomDomainAssociation
	resourcePolicies         map[string]string
	tags                     map[string]map[string]string
}

type redshiftServerlessNamespace struct {
	Name         string
	ARN          string
	AdminUser    string
	DBName       string
	DefaultIAM   string
	Status       string
	CreatedAtRFC string
}

type redshiftServerlessWorkgroup struct {
	Name         string
	ARN          string
	Namespace    string
	BaseCapacity int
	Status       string
	Address      string
	Port         int
	CreatedAtRFC string
}

type redshiftServerlessSnapshot struct {
	Name         string
	ARN          string
	Namespace    string
	Status       string
	CreatedAtRFC string
}

type redshiftServerlessRecoveryPoint struct {
	ID           string
	ARN          string
	Namespace    string
	Workgroup    string
	Status       string
	CreatedAtRFC string
}

type redshiftServerlessTableRestoreStatus struct {
	ID           string
	Namespace    string
	Workgroup    string
	SourceTable  string
	TargetTable  string
	Status       string
	CreatedAtRFC string
}

type redshiftServerlessEndpointAccess struct {
	Name         string
	ARN          string
	Workgroup    string
	Address      string
	Port         int
	Status       string
	CreatedAtRFC string
}

type redshiftServerlessScheduledAction struct {
	Name         string
	ARN          string
	Namespace    string
	Workgroup    string
	Schedule     string
	State        string
	CreatedAtRFC string
}

type redshiftServerlessUsageLimit struct {
	ID           string
	ARN          string
	Namespace    string
	Workgroup    string
	Amount       int
	UsageType    string
	Period       string
	BreachAction string
	CreatedAtRFC string
}

type redshiftServerlessReservation struct {
	Name           string
	ARN            string
	OfferingID     string
	NodeType       string
	NumberOfNodes  int
	TotalAmount    float64
	RecurringHours int
	State          string
	CreatedAtRFC   string
}

type redshiftServerlessSnapshotCopyConfiguration struct {
	Name          string
	Namespace     string
	Destination   string
	RetentionDays int
	CreatedAtRFC  string
}

type redshiftServerlessCustomDomainAssociation struct {
	Name           string
	ARN            string
	Workgroup      string
	CertificateARN string
	Status         string
	CreatedAtRFC   string
}

func newRedshiftServerlessStore() *redshiftServerlessStore {
	s := &redshiftServerlessStore{
		nextID:                   2,
		namespaces:               map[string]*redshiftServerlessNamespace{},
		workgroups:               map[string]*redshiftServerlessWorkgroup{},
		snapshots:                map[string]*redshiftServerlessSnapshot{},
		recoveryPoints:           map[string]*redshiftServerlessRecoveryPoint{},
		tableRestores:            map[string]*redshiftServerlessTableRestoreStatus{},
		endpointAccesses:         map[string]*redshiftServerlessEndpointAccess{},
		scheduledActions:         map[string]*redshiftServerlessScheduledAction{},
		usageLimits:              map[string]*redshiftServerlessUsageLimit{},
		reservations:             map[string]*redshiftServerlessReservation{},
		snapshotCopyConfigs:      map[string]*redshiftServerlessSnapshotCopyConfiguration{},
		customDomainAssociations: map[string]*redshiftServerlessCustomDomainAssociation{},
		resourcePolicies:         map[string]string{},
		tags:                     map[string]map[string]string{},
	}
	s.seedLocked()
	return s
}

func (s *redshiftServerlessStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seedLocked()

	namespaceName := redshiftServerlessPayloadString(payload, "namespaceName", "stackyard-namespace")
	workgroupName := redshiftServerlessPayloadString(payload, "workgroupName", "stackyard-workgroup")
	snapshotName := redshiftServerlessPayloadString(payload, "snapshotName", "stackyard-snapshot")
	recoveryPointID := redshiftServerlessPayloadString(payload, "recoveryPointId", "rp-000001")
	usageLimitID := redshiftServerlessPayloadString(payload, "usageLimitId", "ul-000001")
	scheduledActionName := redshiftServerlessPayloadString(payload, "scheduledActionName", "stackyard-scheduled-action")
	endpointAccessName := redshiftServerlessPayloadString(payload, "endpointName", "stackyard-endpoint")
	customDomainName := redshiftServerlessPayloadString(payload, "customDomainName", "stackyard.example.com")
	reservationName := redshiftServerlessPayloadString(payload, "reservationName", "stackyard-reservation")
	snapshotCopyConfigName := redshiftServerlessPayloadString(payload, "snapshotCopyConfigurationName", "stackyard-snapshot-copy")

	namespace := s.ensureNamespaceLocked(namespaceName)
	workgroup := s.ensureWorkgroupLocked(workgroupName, namespace.Name)
	snapshot := s.ensureSnapshotLocked(snapshotName, namespace.Name)
	recoveryPoint := s.ensureRecoveryPointLocked(recoveryPointID, namespace.Name, workgroup.Name)
	usageLimit := s.ensureUsageLimitLocked(usageLimitID, namespace.Name, workgroup.Name)
	scheduled := s.ensureScheduledActionLocked(scheduledActionName, namespace.Name, workgroup.Name)
	endpointAccess := s.ensureEndpointAccessLocked(endpointAccessName, workgroup.Name)
	customDomain := s.ensureCustomDomainAssociationLocked(customDomainName, workgroup.Name)
	reservation := s.ensureReservationLocked(reservationName)
	snapshotCopyConfig := s.ensureSnapshotCopyConfigurationLocked(snapshotCopyConfigName, namespace.Name)

	switch action {
	case "CreateNamespace":
		namespace = s.ensureNamespaceLocked(redshiftServerlessPayloadString(payload, "namespaceName", namespace.Name))
		if user := redshiftServerlessPayloadString(payload, "adminUsername", ""); user != "" {
			namespace.AdminUser = user
		}
		if db := redshiftServerlessPayloadString(payload, "dbName", ""); db != "" {
			namespace.DBName = db
		}
		namespace.Status = "AVAILABLE"
		s.mergeTagsLocked(namespace.ARN, redshiftServerlessPayloadTags(payload, "tags"))
		return map[string]any{"namespace": redshiftServerlessNamespacePayload(namespace)}

	case "GetNamespace":
		return map[string]any{"namespace": redshiftServerlessNamespacePayload(namespace)}

	case "ListNamespaces":
		return map[string]any{"namespaces": s.listNamespacesPayloadLocked(), "nextToken": ""}

	case "UpdateNamespace":
		if user := redshiftServerlessPayloadString(payload, "adminUsername", ""); user != "" {
			namespace.AdminUser = user
		}
		if db := redshiftServerlessPayloadString(payload, "dbName", ""); db != "" {
			namespace.DBName = db
		}
		namespace.Status = "AVAILABLE"
		return map[string]any{"namespace": redshiftServerlessNamespacePayload(namespace)}

	case "DeleteNamespace":
		namespace.Status = "DELETING"
		return map[string]any{}

	case "CreateWorkgroup":
		wgName := redshiftServerlessPayloadString(payload, "workgroupName", workgroup.Name)
		nsName := redshiftServerlessPayloadString(payload, "namespaceName", namespace.Name)
		workgroup = s.ensureWorkgroupLocked(wgName, nsName)
		if capacity := redshiftServerlessPayloadInt(payload, "baseCapacity", workgroup.BaseCapacity); capacity > 0 {
			workgroup.BaseCapacity = capacity
		}
		workgroup.Status = "AVAILABLE"
		s.mergeTagsLocked(workgroup.ARN, redshiftServerlessPayloadTags(payload, "tags"))
		return map[string]any{"workgroup": redshiftServerlessWorkgroupPayload(workgroup)}

	case "GetWorkgroup":
		return map[string]any{"workgroup": redshiftServerlessWorkgroupPayload(workgroup)}

	case "ListWorkgroups":
		return map[string]any{"workgroups": s.listWorkgroupsPayloadLocked(), "nextToken": ""}

	case "ListManagedWorkgroups":
		return map[string]any{"managedWorkgroups": s.listManagedWorkgroupsPayloadLocked(), "nextToken": ""}

	case "UpdateWorkgroup":
		if capacity := redshiftServerlessPayloadInt(payload, "baseCapacity", workgroup.BaseCapacity); capacity > 0 {
			workgroup.BaseCapacity = capacity
		}
		workgroup.Status = "AVAILABLE"
		return map[string]any{"workgroup": redshiftServerlessWorkgroupPayload(workgroup)}

	case "DeleteWorkgroup":
		workgroup.Status = "DELETING"
		return map[string]any{}

	case "CreateSnapshot":
		snapshot = s.ensureSnapshotLocked(redshiftServerlessPayloadString(payload, "snapshotName", snapshot.Name), namespace.Name)
		snapshot.Status = "AVAILABLE"
		s.mergeTagsLocked(snapshot.ARN, redshiftServerlessPayloadTags(payload, "tags"))
		return map[string]any{"snapshot": redshiftServerlessSnapshotPayload(snapshot)}

	case "GetSnapshot":
		return map[string]any{"snapshot": redshiftServerlessSnapshotPayload(snapshot)}

	case "ListSnapshots":
		return map[string]any{"snapshots": s.listSnapshotsPayloadLocked(), "nextToken": ""}

	case "UpdateSnapshot":
		snapshot.Status = "AVAILABLE"
		return map[string]any{"snapshot": redshiftServerlessSnapshotPayload(snapshot)}

	case "DeleteSnapshot":
		snapshot.Status = "DELETING"
		return map[string]any{}

	case "ListRecoveryPoints":
		return map[string]any{"recoveryPoints": s.listRecoveryPointsPayloadLocked(), "nextToken": ""}

	case "GetRecoveryPoint":
		return map[string]any{"recoveryPoint": redshiftServerlessRecoveryPointPayload(recoveryPoint)}

	case "RestoreFromRecoveryPoint":
		return map[string]any{"namespace": redshiftServerlessNamespacePayload(namespace), "recoveryPointId": recoveryPoint.ID}

	case "RestoreFromSnapshot":
		return map[string]any{"namespace": redshiftServerlessNamespacePayload(namespace), "snapshotName": snapshot.Name}

	case "RestoreTableFromRecoveryPoint":
		t := s.createTableRestoreLocked(namespace.Name, workgroup.Name, redshiftServerlessPayloadString(payload, "sourceTableName", "source_table"), redshiftServerlessPayloadString(payload, "targetTableName", "target_table"))
		return map[string]any{"tableRestoreStatus": redshiftServerlessTableRestorePayload(t)}

	case "RestoreTableFromSnapshot":
		t := s.createTableRestoreLocked(namespace.Name, workgroup.Name, redshiftServerlessPayloadString(payload, "sourceTableName", "source_table"), redshiftServerlessPayloadString(payload, "targetTableName", "target_table"))
		return map[string]any{"tableRestoreStatus": redshiftServerlessTableRestorePayload(t)}

	case "GetTableRestoreStatus":
		tableID := redshiftServerlessPayloadString(payload, "tableRestoreRequestId", "trs-000001")
		t := s.ensureTableRestoreLocked(tableID, namespace.Name, workgroup.Name)
		return map[string]any{"tableRestoreStatus": redshiftServerlessTableRestorePayload(t)}

	case "ListTableRestoreStatus":
		return map[string]any{"tableRestoreStatuses": s.listTableRestoresPayloadLocked(), "nextToken": ""}

	case "CreateEndpointAccess":
		endpointAccess = s.ensureEndpointAccessLocked(redshiftServerlessPayloadString(payload, "endpointName", endpointAccess.Name), workgroup.Name)
		endpointAccess.Status = "AVAILABLE"
		return map[string]any{"endpoint": redshiftServerlessEndpointAccessPayload(endpointAccess)}

	case "GetEndpointAccess":
		return map[string]any{"endpoint": redshiftServerlessEndpointAccessPayload(endpointAccess)}

	case "ListEndpointAccess":
		return map[string]any{"endpoints": s.listEndpointAccessesPayloadLocked(), "nextToken": ""}

	case "UpdateEndpointAccess":
		endpointAccess.Status = "AVAILABLE"
		return map[string]any{"endpoint": redshiftServerlessEndpointAccessPayload(endpointAccess)}

	case "DeleteEndpointAccess":
		endpointAccess.Status = "DELETING"
		return map[string]any{}

	case "CreateScheduledAction":
		scheduled = s.ensureScheduledActionLocked(redshiftServerlessPayloadString(payload, "scheduledActionName", scheduled.Name), namespace.Name, workgroup.Name)
		scheduled.Schedule = redshiftServerlessPayloadString(payload, "schedule", scheduled.Schedule)
		scheduled.State = "ACTIVE"
		return map[string]any{"scheduledAction": redshiftServerlessScheduledActionPayload(scheduled)}

	case "GetScheduledAction":
		return map[string]any{"scheduledAction": redshiftServerlessScheduledActionPayload(scheduled)}

	case "ListScheduledActions":
		return map[string]any{"scheduledActions": s.listScheduledActionsPayloadLocked(), "nextToken": ""}

	case "UpdateScheduledAction":
		scheduled.Schedule = redshiftServerlessPayloadString(payload, "schedule", scheduled.Schedule)
		scheduled.State = "ACTIVE"
		return map[string]any{"scheduledAction": redshiftServerlessScheduledActionPayload(scheduled)}

	case "DeleteScheduledAction":
		scheduled.State = "DELETING"
		return map[string]any{}

	case "CreateUsageLimit":
		usageLimit = s.ensureUsageLimitLocked(redshiftServerlessPayloadString(payload, "usageLimitId", usageLimit.ID), namespace.Name, workgroup.Name)
		usageLimit.UsageType = redshiftServerlessPayloadString(payload, "usageType", usageLimit.UsageType)
		usageLimit.Period = redshiftServerlessPayloadString(payload, "period", usageLimit.Period)
		usageLimit.BreachAction = redshiftServerlessPayloadString(payload, "breachAction", usageLimit.BreachAction)
		usageLimit.Amount = redshiftServerlessPayloadInt(payload, "amount", usageLimit.Amount)
		return map[string]any{"usageLimit": redshiftServerlessUsageLimitPayload(usageLimit)}

	case "GetUsageLimit":
		return map[string]any{"usageLimit": redshiftServerlessUsageLimitPayload(usageLimit)}

	case "ListUsageLimits":
		return map[string]any{"usageLimits": s.listUsageLimitsPayloadLocked(), "nextToken": ""}

	case "UpdateUsageLimit":
		usageLimit.Amount = redshiftServerlessPayloadInt(payload, "amount", usageLimit.Amount)
		usageLimit.BreachAction = redshiftServerlessPayloadString(payload, "breachAction", usageLimit.BreachAction)
		return map[string]any{"usageLimit": redshiftServerlessUsageLimitPayload(usageLimit)}

	case "DeleteUsageLimit":
		return map[string]any{}

	case "CreateReservation":
		reservation = s.ensureReservationLocked(redshiftServerlessPayloadString(payload, "reservationName", reservation.Name))
		return map[string]any{"reservation": redshiftServerlessReservationPayload(reservation)}

	case "GetReservation":
		return map[string]any{"reservation": redshiftServerlessReservationPayload(reservation)}

	case "ListReservations":
		return map[string]any{"reservations": s.listReservationsPayloadLocked(), "nextToken": ""}

	case "ListReservationOfferings":
		return map[string]any{"reservationOfferings": []any{s.reservationOfferingPayload()}, "nextToken": ""}

	case "GetReservationOffering":
		return map[string]any{"reservationOffering": s.reservationOfferingPayload()}

	case "CreateSnapshotCopyConfiguration":
		snapshotCopyConfig = s.ensureSnapshotCopyConfigurationLocked(redshiftServerlessPayloadString(payload, "snapshotCopyConfigurationName", snapshotCopyConfig.Name), namespace.Name)
		snapshotCopyConfig.Destination = redshiftServerlessPayloadString(payload, "destinationRegion", snapshotCopyConfig.Destination)
		snapshotCopyConfig.RetentionDays = redshiftServerlessPayloadInt(payload, "retentionPeriod", snapshotCopyConfig.RetentionDays)
		return map[string]any{"snapshotCopyConfiguration": redshiftServerlessSnapshotCopyConfigurationPayload(snapshotCopyConfig)}

	case "ListSnapshotCopyConfigurations":
		return map[string]any{"snapshotCopyConfigurations": s.listSnapshotCopyConfigurationsPayloadLocked(), "nextToken": ""}

	case "UpdateSnapshotCopyConfiguration":
		snapshotCopyConfig.Destination = redshiftServerlessPayloadString(payload, "destinationRegion", snapshotCopyConfig.Destination)
		snapshotCopyConfig.RetentionDays = redshiftServerlessPayloadInt(payload, "retentionPeriod", snapshotCopyConfig.RetentionDays)
		return map[string]any{"snapshotCopyConfiguration": redshiftServerlessSnapshotCopyConfigurationPayload(snapshotCopyConfig)}

	case "DeleteSnapshotCopyConfiguration":
		return map[string]any{}

	case "CreateCustomDomainAssociation":
		customDomain = s.ensureCustomDomainAssociationLocked(redshiftServerlessPayloadString(payload, "customDomainName", customDomain.Name), workgroup.Name)
		customDomain.CertificateARN = redshiftServerlessPayloadString(payload, "customDomainCertificateArn", customDomain.CertificateARN)
		customDomain.Status = "ACTIVE"
		return map[string]any{"association": redshiftServerlessCustomDomainAssociationPayload(customDomain)}

	case "GetCustomDomainAssociation":
		return map[string]any{"association": redshiftServerlessCustomDomainAssociationPayload(customDomain)}

	case "ListCustomDomainAssociations":
		return map[string]any{"associations": s.listCustomDomainAssociationsPayloadLocked(), "nextToken": ""}

	case "UpdateCustomDomainAssociation":
		customDomain.CertificateARN = redshiftServerlessPayloadString(payload, "customDomainCertificateArn", customDomain.CertificateARN)
		customDomain.Status = "ACTIVE"
		return map[string]any{"association": redshiftServerlessCustomDomainAssociationPayload(customDomain)}

	case "DeleteCustomDomainAssociation":
		customDomain.Status = "DELETING"
		return map[string]any{}

	case "PutResourcePolicy":
		resourceARN := redshiftServerlessPayloadString(payload, "resourceArn", workgroup.ARN)
		if resourceARN == "" {
			resourceARN = workgroup.ARN
		}
		s.resourcePolicies[resourceARN] = redshiftServerlessPayloadString(payload, "policy", `{"Version":"2012-10-17","Statement":[]}`)
		return map[string]any{}

	case "GetResourcePolicy":
		resourceARN := redshiftServerlessPayloadString(payload, "resourceArn", workgroup.ARN)
		if resourceARN == "" {
			resourceARN = workgroup.ARN
		}
		policy := strings.TrimSpace(s.resourcePolicies[resourceARN])
		if policy == "" {
			policy = `{"Version":"2012-10-17","Statement":[]}`
		}
		return map[string]any{"resourcePolicy": map[string]any{"resourceArn": resourceARN, "policy": policy}}

	case "DeleteResourcePolicy":
		resourceARN := redshiftServerlessPayloadString(payload, "resourceArn", workgroup.ARN)
		if resourceARN == "" {
			resourceARN = workgroup.ARN
		}
		delete(s.resourcePolicies, resourceARN)
		return map[string]any{}

	case "TagResource":
		resourceARN := redshiftServerlessPayloadString(payload, "resourceArn", workgroup.ARN)
		s.mergeTagsLocked(resourceARN, redshiftServerlessPayloadTags(payload, "tags"))
		return map[string]any{}

	case "UntagResource":
		resourceARN := redshiftServerlessPayloadString(payload, "resourceArn", workgroup.ARN)
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range redshiftServerlessPayloadStringSlice(payload, "tagKeys") {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := redshiftServerlessPayloadString(payload, "resourceArn", workgroup.ARN)
		return map[string]any{"tags": redshiftServerlessTagsToList(s.ensureTagsLocked(resourceARN))}

	case "GetCredentials":
		return map[string]any{
			"dbUser":          redshiftServerlessPayloadString(payload, "dbUser", namespace.AdminUser),
			"dbPassword":      "stackyard-password",
			"expiration":      time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
			"nextRefreshTime": time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339),
		}

	case "GetIdentityCenterAuthToken":
		return map[string]any{
			"accessToken": "stackyard-idc-token",
			"expiresAt":   time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
		}

	case "GetTrack":
		return map[string]any{"track": s.trackPayload(redshiftServerlessPayloadString(payload, "trackName", "current"))}

	case "ListTracks":
		return map[string]any{"tracks": []any{s.trackPayload("current"), s.trackPayload("preview")}, "nextToken": ""}

	case "ConvertRecoveryPointToSnapshot":
		name := redshiftServerlessPayloadString(payload, "snapshotName", fmt.Sprintf("stackyard-snapshot-%06d", s.nextID))
		snapshot = s.ensureSnapshotLocked(name, recoveryPoint.Namespace)
		snapshot.Status = "AVAILABLE"
		return map[string]any{"snapshot": redshiftServerlessSnapshotPayload(snapshot)}

	case "UpdateLakehouseConfiguration":
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *redshiftServerlessStore) seedLocked() {
	if len(s.namespaces) == 0 {
		ns := &redshiftServerlessNamespace{
			Name:         "stackyard-namespace",
			ARN:          redshiftServerlessARN("namespace", "stackyard-namespace"),
			AdminUser:    "admin",
			DBName:       "dev",
			DefaultIAM:   "arn:aws:iam::123456789012:role/stackyard-redshift-serverless",
			Status:       "AVAILABLE",
			CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
		}
		s.namespaces[ns.Name] = ns
		s.ensureTagsLocked(ns.ARN)["stackyard"] = "true"
	}
	if len(s.workgroups) == 0 {
		wg := &redshiftServerlessWorkgroup{
			Name:         "stackyard-workgroup",
			ARN:          redshiftServerlessARN("workgroup", "stackyard-workgroup"),
			Namespace:    "stackyard-namespace",
			BaseCapacity: 32,
			Status:       "AVAILABLE",
			Address:      "stackyard-workgroup.123456789012.us-east-1.redshift-serverless.amazonaws.com",
			Port:         5439,
			CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
		}
		s.workgroups[wg.Name] = wg
		s.ensureTagsLocked(wg.ARN)["stackyard"] = "true"
	}
	if len(s.snapshots) == 0 {
		ss := &redshiftServerlessSnapshot{
			Name:         "stackyard-snapshot",
			ARN:          redshiftServerlessARN("snapshot", "stackyard-snapshot"),
			Namespace:    "stackyard-namespace",
			Status:       "AVAILABLE",
			CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
		}
		s.snapshots[ss.Name] = ss
	}
	if len(s.recoveryPoints) == 0 {
		rp := &redshiftServerlessRecoveryPoint{
			ID:           "rp-000001",
			ARN:          redshiftServerlessARN("recoverypoint", "rp-000001"),
			Namespace:    "stackyard-namespace",
			Workgroup:    "stackyard-workgroup",
			Status:       "AVAILABLE",
			CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
		}
		s.recoveryPoints[rp.ID] = rp
	}
	if len(s.usageLimits) == 0 {
		s.usageLimits["ul-000001"] = &redshiftServerlessUsageLimit{
			ID:           "ul-000001",
			ARN:          redshiftServerlessARN("usagelimit", "ul-000001"),
			Namespace:    "stackyard-namespace",
			Workgroup:    "stackyard-workgroup",
			Amount:       100,
			UsageType:    "serverless-compute",
			Period:       "daily",
			BreachAction: "log",
			CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
		}
	}
	if len(s.reservations) == 0 {
		s.reservations["stackyard-reservation"] = &redshiftServerlessReservation{
			Name:           "stackyard-reservation",
			ARN:            redshiftServerlessARN("reservation", "stackyard-reservation"),
			OfferingID:     "offering-000001",
			NodeType:       "ra3.xlplus",
			NumberOfNodes:  2,
			TotalAmount:    1000,
			RecurringHours: 24,
			State:          "ACTIVE",
			CreatedAtRFC:   time.Now().UTC().Format(time.RFC3339),
		}
	}
}

func (s *redshiftServerlessStore) ensureNamespaceLocked(name string) *redshiftServerlessNamespace {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-namespace"
	}
	if ns, ok := s.namespaces[name]; ok {
		return ns
	}
	ns := &redshiftServerlessNamespace{
		Name:         name,
		ARN:          redshiftServerlessARN("namespace", name),
		AdminUser:    "admin",
		DBName:       "dev",
		DefaultIAM:   "arn:aws:iam::123456789012:role/stackyard-redshift-serverless",
		Status:       "AVAILABLE",
		CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
	}
	s.namespaces[name] = ns
	s.ensureTagsLocked(ns.ARN)["stackyard"] = "true"
	return ns
}

func (s *redshiftServerlessStore) ensureWorkgroupLocked(name, namespace string) *redshiftServerlessWorkgroup {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-workgroup"
	}
	if wg, ok := s.workgroups[name]; ok {
		return wg
	}
	wg := &redshiftServerlessWorkgroup{
		Name:         name,
		ARN:          redshiftServerlessARN("workgroup", name),
		Namespace:    namespace,
		BaseCapacity: 32,
		Status:       "AVAILABLE",
		Address:      fmt.Sprintf("%s.123456789012.us-east-1.redshift-serverless.amazonaws.com", name),
		Port:         5439,
		CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
	}
	s.workgroups[name] = wg
	s.ensureTagsLocked(wg.ARN)["stackyard"] = "true"
	return wg
}

func (s *redshiftServerlessStore) ensureSnapshotLocked(name, namespace string) *redshiftServerlessSnapshot {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-snapshot"
	}
	if ss, ok := s.snapshots[name]; ok {
		return ss
	}
	ss := &redshiftServerlessSnapshot{
		Name:         name,
		ARN:          redshiftServerlessARN("snapshot", name),
		Namespace:    namespace,
		Status:       "AVAILABLE",
		CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
	}
	s.snapshots[name] = ss
	return ss
}

func (s *redshiftServerlessStore) ensureRecoveryPointLocked(id, namespace, workgroup string) *redshiftServerlessRecoveryPoint {
	if strings.TrimSpace(id) == "" {
		id = "rp-000001"
	}
	if rp, ok := s.recoveryPoints[id]; ok {
		return rp
	}
	rp := &redshiftServerlessRecoveryPoint{
		ID:           id,
		ARN:          redshiftServerlessARN("recoverypoint", id),
		Namespace:    namespace,
		Workgroup:    workgroup,
		Status:       "AVAILABLE",
		CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
	}
	s.recoveryPoints[id] = rp
	return rp
}

func (s *redshiftServerlessStore) createTableRestoreLocked(namespace, workgroup, sourceTable, targetTable string) *redshiftServerlessTableRestoreStatus {
	id := s.nextTokenLocked("trs")
	t := &redshiftServerlessTableRestoreStatus{
		ID:           id,
		Namespace:    namespace,
		Workgroup:    workgroup,
		SourceTable:  sourceTable,
		TargetTable:  targetTable,
		Status:       "SUCCEEDED",
		CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
	}
	s.tableRestores[id] = t
	return t
}

func (s *redshiftServerlessStore) ensureTableRestoreLocked(id, namespace, workgroup string) *redshiftServerlessTableRestoreStatus {
	if strings.TrimSpace(id) == "" {
		id = "trs-000001"
	}
	if t, ok := s.tableRestores[id]; ok {
		return t
	}
	t := &redshiftServerlessTableRestoreStatus{
		ID:           id,
		Namespace:    namespace,
		Workgroup:    workgroup,
		SourceTable:  "source_table",
		TargetTable:  "target_table",
		Status:       "SUCCEEDED",
		CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
	}
	s.tableRestores[id] = t
	return t
}

func (s *redshiftServerlessStore) ensureEndpointAccessLocked(name, workgroup string) *redshiftServerlessEndpointAccess {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-endpoint"
	}
	if ep, ok := s.endpointAccesses[name]; ok {
		return ep
	}
	ep := &redshiftServerlessEndpointAccess{
		Name:         name,
		ARN:          redshiftServerlessARN("endpoint", name),
		Workgroup:    workgroup,
		Address:      fmt.Sprintf("%s.123456789012.us-east-1.redshift-serverless.amazonaws.com", name),
		Port:         5439,
		Status:       "AVAILABLE",
		CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
	}
	s.endpointAccesses[name] = ep
	return ep
}

func (s *redshiftServerlessStore) ensureScheduledActionLocked(name, namespace, workgroup string) *redshiftServerlessScheduledAction {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-scheduled-action"
	}
	if sa, ok := s.scheduledActions[name]; ok {
		return sa
	}
	sa := &redshiftServerlessScheduledAction{
		Name:         name,
		ARN:          redshiftServerlessARN("scheduledaction", name),
		Namespace:    namespace,
		Workgroup:    workgroup,
		Schedule:     "rate(1 day)",
		State:        "ACTIVE",
		CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
	}
	s.scheduledActions[name] = sa
	return sa
}

func (s *redshiftServerlessStore) ensureUsageLimitLocked(id, namespace, workgroup string) *redshiftServerlessUsageLimit {
	if strings.TrimSpace(id) == "" {
		id = "ul-000001"
	}
	if ul, ok := s.usageLimits[id]; ok {
		return ul
	}
	ul := &redshiftServerlessUsageLimit{
		ID:           id,
		ARN:          redshiftServerlessARN("usagelimit", id),
		Namespace:    namespace,
		Workgroup:    workgroup,
		Amount:       100,
		UsageType:    "serverless-compute",
		Period:       "daily",
		BreachAction: "log",
		CreatedAtRFC: time.Now().UTC().Format(time.RFC3339),
	}
	s.usageLimits[id] = ul
	return ul
}

func (s *redshiftServerlessStore) ensureReservationLocked(name string) *redshiftServerlessReservation {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-reservation"
	}
	if r, ok := s.reservations[name]; ok {
		return r
	}
	r := &redshiftServerlessReservation{
		Name:           name,
		ARN:            redshiftServerlessARN("reservation", name),
		OfferingID:     "offering-000001",
		NodeType:       "ra3.xlplus",
		NumberOfNodes:  2,
		TotalAmount:    1000,
		RecurringHours: 24,
		State:          "ACTIVE",
		CreatedAtRFC:   time.Now().UTC().Format(time.RFC3339),
	}
	s.reservations[name] = r
	return r
}

func (s *redshiftServerlessStore) ensureSnapshotCopyConfigurationLocked(name, namespace string) *redshiftServerlessSnapshotCopyConfiguration {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-snapshot-copy"
	}
	if cfg, ok := s.snapshotCopyConfigs[name]; ok {
		return cfg
	}
	cfg := &redshiftServerlessSnapshotCopyConfiguration{
		Name:          name,
		Namespace:     namespace,
		Destination:   "us-west-2",
		RetentionDays: 7,
		CreatedAtRFC:  time.Now().UTC().Format(time.RFC3339),
	}
	s.snapshotCopyConfigs[name] = cfg
	return cfg
}

func (s *redshiftServerlessStore) ensureCustomDomainAssociationLocked(name, workgroup string) *redshiftServerlessCustomDomainAssociation {
	if strings.TrimSpace(name) == "" {
		name = "stackyard.example.com"
	}
	if assoc, ok := s.customDomainAssociations[name]; ok {
		return assoc
	}
	assoc := &redshiftServerlessCustomDomainAssociation{
		Name:           name,
		ARN:            redshiftServerlessARN("custom-domain-association", name),
		Workgroup:      workgroup,
		CertificateARN: "arn:aws:acm:us-east-1:123456789012:certificate/stackyard-redshift-serverless",
		Status:         "ACTIVE",
		CreatedAtRFC:   time.Now().UTC().Format(time.RFC3339),
	}
	s.customDomainAssociations[name] = assoc
	return assoc
}

func (s *redshiftServerlessStore) ensureTagsLocked(resourceARN string) map[string]string {
	if strings.TrimSpace(resourceARN) == "" {
		resourceARN = redshiftServerlessARN("workgroup", "stackyard-workgroup")
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{"stackyard": "true"}
	s.tags[resourceARN] = tags
	return tags
}

func (s *redshiftServerlessStore) mergeTagsLocked(resourceARN string, tags map[string]string) {
	if len(tags) == 0 {
		return
	}
	out := s.ensureTagsLocked(resourceARN)
	for key, value := range tags {
		out[key] = value
	}
}

func (s *redshiftServerlessStore) listNamespacesPayloadLocked() []any {
	names := make([]string, 0, len(s.namespaces))
	for name := range s.namespaces {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, redshiftServerlessNamespacePayload(s.namespaces[name]))
	}
	return items
}

func (s *redshiftServerlessStore) listWorkgroupsPayloadLocked() []any {
	names := make([]string, 0, len(s.workgroups))
	for name := range s.workgroups {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, redshiftServerlessWorkgroupPayload(s.workgroups[name]))
	}
	return items
}

func (s *redshiftServerlessStore) listManagedWorkgroupsPayloadLocked() []any {
	names := make([]string, 0, len(s.workgroups))
	for name := range s.workgroups {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		wg := s.workgroups[name]
		items = append(items, map[string]any{
			"workgroupName": wg.Name,
			"workgroupId":   "mwg-" + wg.Name,
			"status":        wg.Status,
			"endpoint":      map[string]any{"address": wg.Address, "port": wg.Port},
		})
	}
	return items
}

func (s *redshiftServerlessStore) listSnapshotsPayloadLocked() []any {
	names := make([]string, 0, len(s.snapshots))
	for name := range s.snapshots {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, redshiftServerlessSnapshotPayload(s.snapshots[name]))
	}
	return items
}

func (s *redshiftServerlessStore) listRecoveryPointsPayloadLocked() []any {
	ids := make([]string, 0, len(s.recoveryPoints))
	for id := range s.recoveryPoints {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, redshiftServerlessRecoveryPointPayload(s.recoveryPoints[id]))
	}
	return items
}

func (s *redshiftServerlessStore) listTableRestoresPayloadLocked() []any {
	ids := make([]string, 0, len(s.tableRestores))
	for id := range s.tableRestores {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, redshiftServerlessTableRestorePayload(s.tableRestores[id]))
	}
	if len(items) == 0 {
		items = append(items, redshiftServerlessTableRestorePayload(s.ensureTableRestoreLocked("trs-000001", "stackyard-namespace", "stackyard-workgroup")))
	}
	return items
}

func (s *redshiftServerlessStore) listEndpointAccessesPayloadLocked() []any {
	names := make([]string, 0, len(s.endpointAccesses))
	for name := range s.endpointAccesses {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, redshiftServerlessEndpointAccessPayload(s.endpointAccesses[name]))
	}
	return items
}

func (s *redshiftServerlessStore) listScheduledActionsPayloadLocked() []any {
	names := make([]string, 0, len(s.scheduledActions))
	for name := range s.scheduledActions {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, redshiftServerlessScheduledActionPayload(s.scheduledActions[name]))
	}
	return items
}

func (s *redshiftServerlessStore) listUsageLimitsPayloadLocked() []any {
	ids := make([]string, 0, len(s.usageLimits))
	for id := range s.usageLimits {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, redshiftServerlessUsageLimitPayload(s.usageLimits[id]))
	}
	return items
}

func (s *redshiftServerlessStore) listReservationsPayloadLocked() []any {
	names := make([]string, 0, len(s.reservations))
	for name := range s.reservations {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, redshiftServerlessReservationPayload(s.reservations[name]))
	}
	return items
}

func (s *redshiftServerlessStore) listSnapshotCopyConfigurationsPayloadLocked() []any {
	names := make([]string, 0, len(s.snapshotCopyConfigs))
	for name := range s.snapshotCopyConfigs {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, redshiftServerlessSnapshotCopyConfigurationPayload(s.snapshotCopyConfigs[name]))
	}
	return items
}

func (s *redshiftServerlessStore) listCustomDomainAssociationsPayloadLocked() []any {
	names := make([]string, 0, len(s.customDomainAssociations))
	for name := range s.customDomainAssociations {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, redshiftServerlessCustomDomainAssociationPayload(s.customDomainAssociations[name]))
	}
	return items
}

func (s *redshiftServerlessStore) reservationOfferingPayload() map[string]any {
	return map[string]any{
		"reservationOfferingId": "offering-000001",
		"currencyCode":          "USD",
		"duration":              31536000,
		"hourlyRecurringPrice":  1.25,
		"monthlyRecurringPrice": 900,
		"upfrontPrice":          1000,
		"reservationType":       "AllUpfront",
		"nodeType":              "ra3.xlplus",
		"totalNodeCount":        2,
	}
}

func (s *redshiftServerlessStore) trackPayload(name string) map[string]any {
	if strings.TrimSpace(name) == "" {
		name = "current"
	}
	return map[string]any{
		"trackName":        name,
		"workgroupVersion": "1.0",
		"updateTargets":    []any{},
	}
}

func (s *redshiftServerlessStore) nextTokenLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%06d", prefix, id)
}

func redshiftServerlessNamespacePayload(ns *redshiftServerlessNamespace) map[string]any {
	return map[string]any{
		"namespaceName":     ns.Name,
		"namespaceArn":      ns.ARN,
		"adminUsername":     ns.AdminUser,
		"dbName":            ns.DBName,
		"defaultIamRoleArn": ns.DefaultIAM,
		"status":            ns.Status,
		"creationDate":      ns.CreatedAtRFC,
	}
}

func redshiftServerlessWorkgroupPayload(wg *redshiftServerlessWorkgroup) map[string]any {
	return map[string]any{
		"workgroupName": wg.Name,
		"workgroupArn":  wg.ARN,
		"namespaceName": wg.Namespace,
		"baseCapacity":  wg.BaseCapacity,
		"status":        wg.Status,
		"endpoint": map[string]any{
			"address": wg.Address,
			"port":    wg.Port,
		},
		"creationDate": wg.CreatedAtRFC,
	}
}

func redshiftServerlessSnapshotPayload(ss *redshiftServerlessSnapshot) map[string]any {
	return map[string]any{
		"snapshotName":       ss.Name,
		"snapshotArn":        ss.ARN,
		"namespaceName":      ss.Namespace,
		"status":             ss.Status,
		"snapshotCreateTime": ss.CreatedAtRFC,
	}
}

func redshiftServerlessRecoveryPointPayload(rp *redshiftServerlessRecoveryPoint) map[string]any {
	return map[string]any{
		"recoveryPointId":         rp.ID,
		"recoveryPointArn":        rp.ARN,
		"namespaceName":           rp.Namespace,
		"workgroupName":           rp.Workgroup,
		"status":                  rp.Status,
		"recoveryPointCreateTime": rp.CreatedAtRFC,
	}
}

func redshiftServerlessTableRestorePayload(ts *redshiftServerlessTableRestoreStatus) map[string]any {
	return map[string]any{
		"tableRestoreRequestId": ts.ID,
		"namespaceName":         ts.Namespace,
		"workgroupName":         ts.Workgroup,
		"sourceTableName":       ts.SourceTable,
		"targetTableName":       ts.TargetTable,
		"status":                ts.Status,
		"requestTime":           ts.CreatedAtRFC,
	}
}

func redshiftServerlessEndpointAccessPayload(ep *redshiftServerlessEndpointAccess) map[string]any {
	return map[string]any{
		"endpointName":  ep.Name,
		"endpointArn":   ep.ARN,
		"workgroupName": ep.Workgroup,
		"address":       ep.Address,
		"port":          ep.Port,
		"status":        ep.Status,
		"creationDate":  ep.CreatedAtRFC,
	}
}

func redshiftServerlessScheduledActionPayload(sa *redshiftServerlessScheduledAction) map[string]any {
	return map[string]any{
		"scheduledActionName": sa.Name,
		"scheduledActionArn":  sa.ARN,
		"namespaceName":       sa.Namespace,
		"workgroupName":       sa.Workgroup,
		"schedule":            sa.Schedule,
		"state":               sa.State,
		"creationDate":        sa.CreatedAtRFC,
	}
}

func redshiftServerlessUsageLimitPayload(ul *redshiftServerlessUsageLimit) map[string]any {
	return map[string]any{
		"usageLimitId":  ul.ID,
		"usageLimitArn": ul.ARN,
		"namespaceName": ul.Namespace,
		"workgroupName": ul.Workgroup,
		"amount":        ul.Amount,
		"usageType":     ul.UsageType,
		"period":        ul.Period,
		"breachAction":  ul.BreachAction,
		"creationDate":  ul.CreatedAtRFC,
	}
}

func redshiftServerlessReservationPayload(r *redshiftServerlessReservation) map[string]any {
	return map[string]any{
		"reservationName":       r.Name,
		"reservationArn":        r.ARN,
		"reservationOfferingId": r.OfferingID,
		"nodeType":              r.NodeType,
		"numberOfNodes":         r.NumberOfNodes,
		"totalAmount":           r.TotalAmount,
		"hourlyRecurringPrice":  1.25,
		"state":                 r.State,
		"reservationCreateTime": r.CreatedAtRFC,
	}
}

func redshiftServerlessSnapshotCopyConfigurationPayload(cfg *redshiftServerlessSnapshotCopyConfiguration) map[string]any {
	return map[string]any{
		"snapshotCopyConfigurationName": cfg.Name,
		"namespaceName":                 cfg.Namespace,
		"destinationRegion":             cfg.Destination,
		"retentionPeriod":               cfg.RetentionDays,
		"creationDate":                  cfg.CreatedAtRFC,
	}
}

func redshiftServerlessCustomDomainAssociationPayload(assoc *redshiftServerlessCustomDomainAssociation) map[string]any {
	return map[string]any{
		"customDomainName":           assoc.Name,
		"customDomainCertificateArn": assoc.CertificateARN,
		"workgroupName":              assoc.Workgroup,
		"status":                     assoc.Status,
		"associationArn":             assoc.ARN,
		"creationDate":               assoc.CreatedAtRFC,
	}
}

func redshiftServerlessTagsToList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]any, 0, len(keys))
	for _, key := range keys {
		items = append(items, map[string]any{"key": key, "value": tags[key]})
	}
	return items
}

func redshiftServerlessPayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func redshiftServerlessPayloadString(payload map[string]any, key, fallback string) string {
	value, ok := redshiftServerlessPayloadValue(payload, key)
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		v := strings.TrimSpace(typed)
		if v == "" {
			return fallback
		}
		return v
	case fmt.Stringer:
		v := strings.TrimSpace(typed.String())
		if v == "" {
			return fallback
		}
		return v
	default:
		v := strings.TrimSpace(fmt.Sprintf("%v", typed))
		if v == "" || v == "<nil>" {
			return fallback
		}
		return v
	}
}

func redshiftServerlessPayloadInt(payload map[string]any, key string, fallback int) int {
	value, ok := redshiftServerlessPayloadValue(payload, key)
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		var out int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &out); err == nil {
			return out
		}
	}
	return fallback
}

func redshiftServerlessPayloadTags(payload map[string]any, key string) map[string]string {
	value, ok := redshiftServerlessPayloadValue(payload, key)
	if !ok || value == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]any:
		for k, v := range typed {
			out[strings.TrimSpace(k)] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	case map[string]string:
		for k, v := range typed {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	case []any:
		for _, raw := range typed {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			k := redshiftServerlessPayloadString(item, "key", redshiftServerlessPayloadString(item, "Key", ""))
			if k == "" {
				continue
			}
			v := redshiftServerlessPayloadString(item, "value", redshiftServerlessPayloadString(item, "Value", ""))
			out[k] = v
		}
	}
	return out
}

func redshiftServerlessPayloadStringSlice(payload map[string]any, key string) []string {
	value, ok := redshiftServerlessPayloadValue(payload, key)
	if !ok || value == nil {
		return nil
	}
	rawList, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(rawList))
	for _, raw := range rawList {
		v := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if v != "" && v != "<nil>" {
			out = append(out, v)
		}
	}
	return out
}

func redshiftServerlessARN(resourceType, id string) string {
	cleanType := strings.Trim(strings.TrimSpace(resourceType), "/")
	cleanID := strings.Trim(strings.TrimSpace(id), "/")
	if cleanType == "" {
		cleanType = "resource"
	}
	if cleanID == "" {
		cleanID = "stackyard"
	}
	return fmt.Sprintf("arn:aws:redshift-serverless:us-east-1:123456789012:%s/%s", cleanType, cleanID)
}
