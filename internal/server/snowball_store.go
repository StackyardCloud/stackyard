package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	snowballDefaultAddressID      = "ADID000000000000000000000000000000000000"
	snowballDefaultClusterID      = "CID000000000000000000000000000000000000"
	snowballDefaultJobID          = "JID000000000000000000000000000000000000"
	snowballDefaultPricingID      = "LTP000000000000000000000"
	snowballDefaultAddressARN     = "arn:aws:snowball:us-east-1:123456789012:address/" + snowballDefaultAddressID
	snowballDefaultClusterARN     = "arn:aws:snowball:us-east-1:123456789012:cluster/" + snowballDefaultClusterID
	snowballDefaultJobARN         = "arn:aws:snowball:us-east-1:123456789012:job/" + snowballDefaultJobID
	snowballDefaultPricingARN     = "arn:aws:snowball:us-east-1:123456789012:long-term-pricing/" + snowballDefaultPricingID
	snowballDefaultManifestURL    = "https://stackyard.local/snowball/manifest/" + snowballDefaultJobID
	snowballDefaultSoftwareURL    = "https://stackyard.local/snowball/software/" + snowballDefaultJobID
	snowballDefaultUnlockCode     = "1111-2222-3333"
	snowballDefaultServiceName    = "snowball"
	snowballDefaultServiceVersion = "2026-01-01"
)

type snowballStore struct {
	mu sync.Mutex

	nextAddress int64
	nextCluster int64
	nextJob     int64
	nextPricing int64

	addresses map[string]map[string]any
	clusters  map[string]map[string]any
	jobs      map[string]map[string]any
	pricing   map[string]map[string]any
}

func newSnowballStore() *snowballStore {
	s := &snowballStore{
		nextAddress: 1,
		nextCluster: 1,
		nextJob:     1,
		nextPricing: 1,
		addresses:   map[string]map[string]any{},
		clusters:    map[string]map[string]any{},
		jobs:        map[string]map[string]any{},
		pricing:     map[string]map[string]any{},
	}

	s.addresses[snowballDefaultAddressID] = map[string]any{
		"AddressId":       snowballDefaultAddressID,
		"AddressArn":      snowballDefaultAddressARN,
		"Name":            "Stackyard Seed Address",
		"Company":         "Stackyard",
		"Street1":         "1 Stackyard Way",
		"City":            "Seattle",
		"StateOrProvince": "WA",
		"Country":         "US",
		"PostalCode":      "98101",
	}
	s.clusters[snowballDefaultClusterID] = map[string]any{
		"ClusterId":      snowballDefaultClusterID,
		"ClusterArn":     snowballDefaultClusterARN,
		"AddressId":      snowballDefaultAddressID,
		"Description":    "Stackyard seed cluster",
		"JobType":        "IMPORT",
		"SnowballType":   "STANDARD",
		"ShippingOption": "SECOND_DAY",
		"ClusterState":   "AwaitingQuorum",
		"CreationDate":   "2026-01-01T00:00:00Z",
	}
	s.jobs[snowballDefaultJobID] = map[string]any{
		"JobId":          snowballDefaultJobID,
		"JobArn":         snowballDefaultJobARN,
		"AddressId":      snowballDefaultAddressID,
		"ClusterId":      snowballDefaultClusterID,
		"Description":    "Stackyard seed job",
		"JobType":        "IMPORT",
		"SnowballType":   "STANDARD",
		"ShippingOption": "SECOND_DAY",
		"JobState":       "New",
		"CreationDate":   "2026-01-01T00:00:00Z",
		"ShipmentState":  "RECEIVED",
	}
	s.pricing[snowballDefaultPricingID] = map[string]any{
		"LongTermPricingId":          snowballDefaultPricingID,
		"LongTermPricingArn":         snowballDefaultPricingARN,
		"LongTermPricingType":        "ONE_YEAR",
		"SnowballType":               "STANDARD",
		"CurrentActiveJob":           "0",
		"IsLongTermPricingAutoRenew": false,
		"ReplacementJob":             snowballDefaultJobID,
	}

	return s
}

func (s *snowballStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CancelCluster":
		clusterID := snowballPayloadString(payload, "ClusterId", snowballDefaultClusterID)
		cluster := s.ensureClusterLocked(clusterID)
		cluster["ClusterState"] = "Cancelled"
		return map[string]any{}
	case "CancelJob":
		jobID := snowballPayloadString(payload, "JobId", snowballDefaultJobID)
		job := s.ensureJobLocked(jobID)
		job["JobState"] = "Cancelled"
		return map[string]any{}
	case "CreateAddress":
		addressID := s.nextAddressIDLocked()
		address := map[string]any{
			"AddressId":       addressID,
			"AddressArn":      snowballAddressARN(addressID),
			"Name":            "Stackyard Address",
			"Street1":         "1 Stackyard Way",
			"City":            "Seattle",
			"StateOrProvince": "WA",
			"Country":         "US",
			"PostalCode":      "98101",
		}
		if provided := snowballPayloadMap(payload, "Address"); provided != nil {
			for _, key := range []string{
				"Name",
				"Company",
				"PhoneNumber",
				"Street1",
				"Street2",
				"Street3",
				"City",
				"StateOrProvince",
				"PrefectureOrDistrict",
				"Landmark",
				"Country",
				"PostalCode",
			} {
				if value, ok := provided[key]; ok {
					address[key] = value
				}
			}
		}
		s.addresses[addressID] = address
		return map[string]any{"AddressId": addressID}
	case "CreateCluster":
		clusterID := s.nextClusterIDLocked()
		addressID := snowballPayloadString(payload, "AddressId", s.firstAddressIDLocked())
		cluster := map[string]any{
			"ClusterId":      clusterID,
			"ClusterArn":     snowballClusterARN(clusterID),
			"AddressId":      addressID,
			"Description":    snowballPayloadString(payload, "Description", "Stackyard cluster"),
			"JobType":        snowballPayloadString(payload, "JobType", "IMPORT"),
			"SnowballType":   snowballPayloadString(payload, "SnowballType", "STANDARD"),
			"ShippingOption": snowballPayloadString(payload, "ShippingOption", "SECOND_DAY"),
			"ClusterState":   "AwaitingQuorum",
			"CreationDate":   "2026-01-01T00:00:00Z",
		}
		s.clusters[clusterID] = cluster
		return map[string]any{"ClusterId": clusterID}
	case "CreateJob":
		jobID := s.nextJobIDLocked()
		job := map[string]any{
			"JobId":          jobID,
			"JobArn":         snowballJobARN(jobID),
			"AddressId":      snowballPayloadString(payload, "AddressId", s.firstAddressIDLocked()),
			"ClusterId":      snowballPayloadString(payload, "ClusterId", s.firstClusterIDLocked()),
			"Description":    snowballPayloadString(payload, "Description", "Stackyard job"),
			"JobType":        snowballPayloadString(payload, "JobType", "IMPORT"),
			"SnowballType":   snowballPayloadString(payload, "SnowballType", "STANDARD"),
			"ShippingOption": snowballPayloadString(payload, "ShippingOption", "SECOND_DAY"),
			"JobState":       "New",
			"ShipmentState":  snowballPayloadString(payload, "ShipmentState", "RECEIVED"),
			"CreationDate":   "2026-01-01T00:00:00Z",
		}
		s.jobs[jobID] = job
		return map[string]any{"JobId": jobID}
	case "CreateLongTermPricing":
		pricingID := s.nextPricingIDLocked()
		entry := map[string]any{
			"LongTermPricingId":          pricingID,
			"LongTermPricingArn":         snowballPricingARN(pricingID),
			"LongTermPricingType":        snowballPayloadString(payload, "LongTermPricingType", "ONE_YEAR"),
			"SnowballType":               snowballPayloadString(payload, "SnowballType", "STANDARD"),
			"CurrentActiveJob":           "0",
			"IsLongTermPricingAutoRenew": false,
		}
		s.pricing[pricingID] = entry
		return map[string]any{"LongTermPricingId": pricingID}
	case "CreateReturnShippingLabel":
		return map[string]any{"Status": "InProgress"}
	case "DescribeAddress":
		addressID := snowballPayloadString(payload, "AddressId", s.firstAddressIDLocked())
		address := s.ensureAddressLocked(addressID)
		return map[string]any{"Address": snowballCloneMap(address)}
	case "DescribeAddresses":
		return map[string]any{"Addresses": s.listAddressesLocked()}
	case "DescribeCluster":
		clusterID := snowballPayloadString(payload, "ClusterId", s.firstClusterIDLocked())
		cluster := s.ensureClusterLocked(clusterID)
		return map[string]any{"ClusterMetadata": snowballCloneMap(cluster)}
	case "DescribeJob":
		jobID := snowballPayloadString(payload, "JobId", s.firstJobIDLocked())
		job := s.ensureJobLocked(jobID)
		return map[string]any{"JobMetadata": snowballCloneMap(job)}
	case "DescribeReturnShippingLabel":
		return map[string]any{"Status": "SUCCEEDED"}
	case "GetJobManifest":
		jobID := snowballPayloadString(payload, "JobId", s.firstJobIDLocked())
		return map[string]any{"ManifestURI": "https://stackyard.local/snowball/manifest/" + jobID}
	case "GetJobUnlockCode":
		return map[string]any{"UnlockCode": snowballDefaultUnlockCode}
	case "GetSnowballUsage":
		return map[string]any{
			"SnowballLimit":  10,
			"SnowballsInUse": len(s.jobs),
		}
	case "GetSoftwareUpdates":
		jobID := snowballPayloadString(payload, "JobId", s.firstJobIDLocked())
		return map[string]any{"UpdatesURI": "https://stackyard.local/snowball/software/" + jobID}
	case "ListClusterJobs":
		clusterID := snowballPayloadString(payload, "ClusterId", s.firstClusterIDLocked())
		return map[string]any{"JobListEntries": s.listClusterJobsLocked(clusterID)}
	case "ListClusters":
		return map[string]any{"ClusterListEntries": s.listClustersLocked()}
	case "ListCompatibleImages":
		return map[string]any{
			"CompatibleImages": []any{
				map[string]any{
					"Name":         "Ubuntu 22.04 LTS",
					"Version":      "1.0.0",
					"AmiId":        "ami-00000000000000001",
					"TargetDevice": "SNOWBALL_EDGE",
				},
			},
		}
	case "ListJobs":
		return map[string]any{"JobListEntries": s.listJobsLocked()}
	case "ListLongTermPricing":
		return map[string]any{"LongTermPricingEntries": s.listPricingLocked()}
	case "ListPickupLocations":
		return map[string]any{
			"Addresses": []any{
				map[string]any{
					"Name":            "Seattle Pickup",
					"Street1":         "1200 1st Ave",
					"City":            "Seattle",
					"StateOrProvince": "WA",
					"Country":         "US",
					"PostalCode":      "98101",
				},
				map[string]any{
					"Name":            "San Jose Pickup",
					"Street1":         "100 W San Fernando St",
					"City":            "San Jose",
					"StateOrProvince": "CA",
					"Country":         "US",
					"PostalCode":      "95113",
				},
			},
		}
	case "ListServiceVersions":
		serviceName := snowballPayloadString(payload, "ServiceName", snowballDefaultServiceName)
		return map[string]any{
			"ServiceVersions": []any{
				map[string]any{
					"ServiceName": serviceName,
					"Version":     snowballDefaultServiceVersion,
				},
			},
		}
	case "UpdateCluster":
		clusterID := snowballPayloadString(payload, "ClusterId", s.firstClusterIDLocked())
		cluster := s.ensureClusterLocked(clusterID)
		if description := snowballPayloadString(payload, "Description", ""); description != "" {
			cluster["Description"] = description
		}
		if shipping := snowballPayloadString(payload, "ShippingOption", ""); shipping != "" {
			cluster["ShippingOption"] = shipping
		}
		return map[string]any{}
	case "UpdateJob":
		jobID := snowballPayloadString(payload, "JobId", s.firstJobIDLocked())
		job := s.ensureJobLocked(jobID)
		if description := snowballPayloadString(payload, "Description", ""); description != "" {
			job["Description"] = description
		}
		if shipping := snowballPayloadString(payload, "ShippingOption", ""); shipping != "" {
			job["ShippingOption"] = shipping
		}
		return map[string]any{}
	case "UpdateJobShipmentState":
		jobID := snowballPayloadString(payload, "JobId", s.firstJobIDLocked())
		job := s.ensureJobLocked(jobID)
		job["ShipmentState"] = snowballPayloadString(payload, "ShipmentState", "RECEIVED")
		return map[string]any{}
	case "UpdateLongTermPricing":
		pricingID := snowballPayloadString(payload, "LongTermPricingId", s.firstPricingIDLocked())
		entry := s.ensurePricingLocked(pricingID)
		if autoRenew, ok := payload["IsLongTermPricingAutoRenew"]; ok {
			entry["IsLongTermPricingAutoRenew"] = autoRenew
		}
		if replacementJob, ok := payload["ReplacementJob"]; ok {
			entry["ReplacementJob"] = replacementJob
		}
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func (s *snowballStore) nextAddressIDLocked() string {
	s.nextAddress++
	return fmt.Sprintf("ADID%036d", s.nextAddress)
}

func (s *snowballStore) nextClusterIDLocked() string {
	s.nextCluster++
	return fmt.Sprintf("CID%036d", s.nextCluster)
}

func (s *snowballStore) nextJobIDLocked() string {
	s.nextJob++
	return fmt.Sprintf("JID%036d", s.nextJob)
}

func (s *snowballStore) nextPricingIDLocked() string {
	s.nextPricing++
	return fmt.Sprintf("LTP%021d", s.nextPricing)
}

func (s *snowballStore) firstAddressIDLocked() string {
	for id := range s.addresses {
		return id
	}
	return snowballDefaultAddressID
}

func (s *snowballStore) firstClusterIDLocked() string {
	for id := range s.clusters {
		return id
	}
	return snowballDefaultClusterID
}

func (s *snowballStore) firstJobIDLocked() string {
	for id := range s.jobs {
		return id
	}
	return snowballDefaultJobID
}

func (s *snowballStore) firstPricingIDLocked() string {
	for id := range s.pricing {
		return id
	}
	return snowballDefaultPricingID
}

func (s *snowballStore) ensureAddressLocked(addressID string) map[string]any {
	addressID = strings.TrimSpace(addressID)
	if addressID == "" {
		addressID = snowballDefaultAddressID
	}
	if address, ok := s.addresses[addressID]; ok {
		return address
	}
	address := map[string]any{
		"AddressId":       addressID,
		"AddressArn":      snowballAddressARN(addressID),
		"Name":            "Stackyard Address",
		"Street1":         "1 Stackyard Way",
		"City":            "Seattle",
		"StateOrProvince": "WA",
		"Country":         "US",
		"PostalCode":      "98101",
	}
	s.addresses[addressID] = address
	return address
}

func (s *snowballStore) ensureClusterLocked(clusterID string) map[string]any {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		clusterID = snowballDefaultClusterID
	}
	if cluster, ok := s.clusters[clusterID]; ok {
		return cluster
	}
	cluster := map[string]any{
		"ClusterId":      clusterID,
		"ClusterArn":     snowballClusterARN(clusterID),
		"AddressId":      s.firstAddressIDLocked(),
		"Description":    "Stackyard cluster",
		"JobType":        "IMPORT",
		"SnowballType":   "STANDARD",
		"ShippingOption": "SECOND_DAY",
		"ClusterState":   "AwaitingQuorum",
		"CreationDate":   "2026-01-01T00:00:00Z",
	}
	s.clusters[clusterID] = cluster
	return cluster
}

func (s *snowballStore) ensureJobLocked(jobID string) map[string]any {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		jobID = snowballDefaultJobID
	}
	if job, ok := s.jobs[jobID]; ok {
		return job
	}
	job := map[string]any{
		"JobId":          jobID,
		"JobArn":         snowballJobARN(jobID),
		"AddressId":      s.firstAddressIDLocked(),
		"ClusterId":      s.firstClusterIDLocked(),
		"Description":    "Stackyard job",
		"JobType":        "IMPORT",
		"SnowballType":   "STANDARD",
		"ShippingOption": "SECOND_DAY",
		"JobState":       "New",
		"ShipmentState":  "RECEIVED",
		"CreationDate":   "2026-01-01T00:00:00Z",
	}
	s.jobs[jobID] = job
	return job
}

func (s *snowballStore) ensurePricingLocked(pricingID string) map[string]any {
	pricingID = strings.TrimSpace(pricingID)
	if pricingID == "" {
		pricingID = snowballDefaultPricingID
	}
	if entry, ok := s.pricing[pricingID]; ok {
		return entry
	}
	entry := map[string]any{
		"LongTermPricingId":          pricingID,
		"LongTermPricingArn":         snowballPricingARN(pricingID),
		"LongTermPricingType":        "ONE_YEAR",
		"SnowballType":               "STANDARD",
		"CurrentActiveJob":           "0",
		"IsLongTermPricingAutoRenew": false,
	}
	s.pricing[pricingID] = entry
	return entry
}

func (s *snowballStore) listAddressesLocked() []any {
	ids := make([]string, 0, len(s.addresses))
	for id := range s.addresses {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, snowballCloneMap(s.addresses[id]))
	}
	return items
}

func (s *snowballStore) listClustersLocked() []any {
	ids := make([]string, 0, len(s.clusters))
	for id := range s.clusters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		cluster := s.clusters[id]
		items = append(items, map[string]any{
			"ClusterId":    cluster["ClusterId"],
			"Description":  cluster["Description"],
			"ClusterState": cluster["ClusterState"],
			"CreationDate": cluster["CreationDate"],
		})
	}
	return items
}

func (s *snowballStore) listJobsLocked() []any {
	ids := make([]string, 0, len(s.jobs))
	for id := range s.jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		job := s.jobs[id]
		items = append(items, map[string]any{
			"JobId":        job["JobId"],
			"JobState":     job["JobState"],
			"Description":  job["Description"],
			"CreationDate": job["CreationDate"],
		})
	}
	return items
}

func (s *snowballStore) listClusterJobsLocked(clusterID string) []any {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		clusterID = s.firstClusterIDLocked()
	}

	ids := make([]string, 0, len(s.jobs))
	for id, job := range s.jobs {
		if strings.TrimSpace(toString(job["ClusterId"])) == clusterID {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return []any{}
	}

	items := make([]any, 0, len(ids))
	for _, id := range ids {
		job := s.jobs[id]
		items = append(items, map[string]any{
			"JobId":        job["JobId"],
			"JobState":     job["JobState"],
			"Description":  job["Description"],
			"CreationDate": job["CreationDate"],
		})
	}
	return items
}

func (s *snowballStore) listPricingLocked() []any {
	ids := make([]string, 0, len(s.pricing))
	for id := range s.pricing {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, snowballCloneMap(s.pricing[id]))
	}
	return items
}

func snowballAddressARN(addressID string) string {
	return "arn:aws:snowball:us-east-1:123456789012:address/" + strings.TrimSpace(addressID)
}

func snowballClusterARN(clusterID string) string {
	return "arn:aws:snowball:us-east-1:123456789012:cluster/" + strings.TrimSpace(clusterID)
}

func snowballJobARN(jobID string) string {
	return "arn:aws:snowball:us-east-1:123456789012:job/" + strings.TrimSpace(jobID)
}

func snowballPricingARN(pricingID string) string {
	return "arn:aws:snowball:us-east-1:123456789012:long-term-pricing/" + strings.TrimSpace(pricingID)
}

func snowballPayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	value, ok := payload[key]
	if !ok {
		return def
	}
	text := strings.TrimSpace(toString(value))
	if text == "" {
		return def
	}
	return text
}

func snowballPayloadMap(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok {
		return nil
	}
	out, _ := value.(map[string]any)
	return out
}

func snowballCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = snowballCloneMap(typed)
		case []any:
			out[k] = snowballCloneList(typed)
		default:
			out[k] = typed
		}
	}
	return out
}

func snowballCloneList(in []any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		switch typed := item.(type) {
		case map[string]any:
			out = append(out, snowballCloneMap(typed))
		case []any:
			out = append(out, snowballCloneList(typed))
		default:
			out = append(out, typed)
		}
	}
	return out
}
