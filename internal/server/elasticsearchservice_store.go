package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// eslState is the es-only state kept in the OpenSearch store (packages, VPC
// endpoints, cross-cluster connections, reservations). Domains live in
// openSearchStore.domains so both APIs see them.
type eslState struct {
	packages     map[string]*eslPackage
	endpoints    map[string]map[string]any // VpcEndpoint, by id
	clientTokens map[string]string         // CreateVpcEndpoint ClientToken → id
	connections  map[string]*eslConnection
	reserved     map[string]map[string]any // ReservedElasticsearchInstance, by id
	seq          int
}

type eslPackage struct {
	details map[string]any
	history []any
	version int
}

type eslConnection struct {
	id                    string
	alias                 string
	source, destination   map[string]any
	outStatus, inStatus   string
	outMessage, inMessage string
}

const (
	eslMaxTags         = 50
	eslMaxPackages     = 500
	eslMaxEndpoints    = 200
	eslMaxConnections  = 200
	eslMaxReservations = 500
	eslMaxAccess       = 50
	eslDefaultVersion  = "1.5"
)

// eslVersions are the versions the es API accepts, oldest first.
var eslVersions = []string{
	"1.5", "2.3", "5.1", "5.3", "5.5", "5.6", "6.0", "6.2", "6.3", "6.4", "6.5", "6.7", "6.8",
	"7.1", "7.4", "7.7", "7.8", "7.9", "7.10",
	"OpenSearch_1.0", "OpenSearch_1.1", "OpenSearch_1.2", "OpenSearch_1.3",
	"OpenSearch_2.3", "OpenSearch_2.5", "OpenSearch_2.7", "OpenSearch_2.9", "OpenSearch_2.11",
}

func eslMajor(v string) string {
	prefix := ""
	if strings.HasPrefix(v, "OpenSearch_") {
		prefix, v = "os", strings.TrimPrefix(v, "OpenSearch_")
	}
	major, _, _ := strings.Cut(v, ".")
	return prefix + major
}

// eslUpgradeTargets lists the versions v can upgrade to: later versions of the
// same major, plus the next major once v is the last of its own.
func eslUpgradeTargets(v string) []any {
	i := slices.Index(eslVersions, v)
	if i < 0 {
		return []any{}
	}
	out := []any{}
	major := eslMajor(v)
	for _, t := range eslVersions[i+1:] {
		if eslMajor(t) == major {
			out = append(out, t)
		}
	}
	if len(out) == 0 && i+1 < len(eslVersions) {
		next := eslMajor(eslVersions[i+1])
		for _, t := range eslVersions[i+1:] {
			if eslMajor(t) == next {
				out = append(out, t)
			}
		}
	}
	return out
}

func eslVersionOf(engine string) string { return strings.TrimPrefix(engine, "Elasticsearch_") }

func eslEngineOf(version string) string {
	if strings.HasPrefix(version, "OpenSearch_") {
		return version
	}
	return "Elasticsearch_" + version
}

// eslClusterConfig translates a cluster config between the APIs' instance-type
// suffixes (".elasticsearch" ↔ ".search").
func eslClusterConfig(c map[string]any, from, to string) map[string]any {
	out := osCopyMap(c)
	for _, key := range []string{"InstanceType", "DedicatedMasterType", "WarmType"} {
		if s, ok := out[key].(string); ok {
			out[key] = strings.TrimSuffix(s, from) + to
			if !strings.HasSuffix(s, from) {
				out[key] = s
			}
		}
	}
	return out
}

func eslStr(in map[string]any, key string) string { s, _ := in[key].(string); return s }
func eslMap(in map[string]any, key string) map[string]any {
	m, _ := in[key].(map[string]any)
	return m
}
func eslList(in map[string]any, key string) []any { l, _ := in[key].([]any); return l }
func eslBool(in map[string]any, key string) bool  { b, _ := in[key].(bool); return b }
func eslNum(in map[string]any, key string) (float64, bool) {
	n, ok := in[key].(float64)
	return n, ok
}
func eslUnix(t time.Time) float64 { return float64(t.Unix()) }

// eslPage pages items by an offset NextToken; MaxResults defaults to 100.
func eslPage(items []any, in map[string]any) ([]any, string, *eslError) {
	limit := 100
	if n, ok := eslNum(in, "MaxResults"); ok && n > 0 {
		limit = int(min(n, 100))
	}
	start := 0
	if tok := eslStr(in, "NextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", eslErr(http.StatusBadRequest, "InvalidPaginationTokenException", "the NextToken is invalid")
		}
		start = int(n)
	}
	end := min(start+limit, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

// eslHandle runs one es operation under the store lock and returns the
// response conformed to the operation's output shape.
func (s *openSearchStore) eslHandle(op string, in map[string]any) (any, *eslError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := &s.esl
	if st.packages == nil {
		*st = eslState{packages: map[string]*eslPackage{}, endpoints: map[string]map[string]any{}, clientTokens: map[string]string{},
			connections: map[string]*eslConnection{}, reserved: map[string]map[string]any{}}
	}
	if s.domains == nil {
		s.domains = map[string]*osDomain{}
	}
	out, err := s.eslDispatch(op, in)
	if err != nil {
		return nil, err
	}
	m := awsmodels.ElasticsearchServiceModel()
	if shape := m.Operations[op].Output; shape != "" && out != nil {
		return m.Conform(shape, out), nil
	}
	return map[string]any{}, nil
}

func (s *openSearchStore) eslDomain(name string) (*osDomain, *eslError) {
	if d := s.domains[name]; d != nil {
		return d, nil
	}
	return nil, eslNotFound("Domain not found: " + name)
}

func (s *openSearchStore) eslDomainByARN(arn string) (*osDomain, *eslError) {
	prefix := fmt.Sprintf("arn:aws:es:%s:%s:domain/", osDomainRegion, osDomainAccountID)
	if !strings.HasPrefix(arn, prefix) {
		return nil, eslValidation("Invalid ARN: " + arn)
	}
	return s.eslDomain(strings.TrimPrefix(arn, prefix))
}

func (s *openSearchStore) eslDispatch(op string, in map[string]any) (map[string]any, *eslError) {
	switch op {
	case "CreateElasticsearchDomain":
		return s.eslCreateDomain(in)
	case "DescribeElasticsearchDomain":
		d, err := s.eslDomain(eslStr(in, "DomainName"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"DomainStatus": eslStatus(d, d.read())}, nil
	case "DescribeElasticsearchDomains":
		list := []any{}
		for _, n := range eslList(in, "DomainNames") {
			name, _ := n.(string)
			if d := s.domains[name]; d != nil {
				list = append(list, eslStatus(d, d.read()))
			}
		}
		return map[string]any{"DomainStatusList": list}, nil
	case "DeleteElasticsearchDomain":
		d, err := s.eslDomain(eslStr(in, "DomainName"))
		if err != nil {
			return nil, err
		}
		status := eslStatus(d, true)
		status["Deleted"] = true
		// ponytail: deletion is immediate; AWS keeps Deleted domains visible for a while.
		delete(s.domains, d.name)
		return map[string]any{"DomainStatus": status}, nil
	case "DescribeElasticsearchDomainConfig":
		d, err := s.eslDomain(eslStr(in, "DomainName"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"DomainConfig": eslConfig(d)}, nil
	case "UpdateElasticsearchDomainConfig":
		return s.eslUpdateDomain(in)
	case "ListDomainNames":
		want := eslStr(in, "EngineType")
		list := []any{}
		for _, d := range s.sortedDomainsLocked() {
			if want == "" || d.engineType() == want {
				list = append(list, map[string]any{"DomainName": d.name, "EngineType": d.engineType()})
			}
		}
		return map[string]any{"DomainNames": list}, nil
	case "AddTags", "ListTags", "RemoveTags":
		return s.eslTags(op, in)
	case "GetCompatibleElasticsearchVersions":
		return s.eslCompatibleVersions(in)
	case "ListElasticsearchVersions":
		all := []any{}
		for i := len(eslVersions) - 1; i >= 0; i-- {
			all = append(all, eslVersions[i])
		}
		page, next, err := eslPage(all, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{"ElasticsearchVersions": page, "NextToken": next}, nil
	case "UpgradeElasticsearchDomain", "GetUpgradeHistory", "GetUpgradeStatus":
		return s.eslUpgrade(op, in)
	case "ListElasticsearchInstanceTypes", "DescribeElasticsearchInstanceTypeLimits":
		return s.eslInstanceTypes(op, in)
	case "DescribeDomainAutoTunes", "DescribeDomainChangeProgress", "CancelDomainConfigChange",
		"StartElasticsearchServiceSoftwareUpdate", "CancelElasticsearchServiceSoftwareUpdate", "UpdateClone":
		return s.eslDomainActions(op, in)
	case "AuthorizeVpcEndpointAccess", "RevokeVpcEndpointAccess", "ListVpcEndpointAccess":
		return s.eslVpcAccess(op, in)
	case "DeleteElasticsearchServiceRole":
		return map[string]any{}, nil
	}
	switch {
	case strings.Contains(op, "Package"):
		return s.eslPackages(op, in)
	case strings.Contains(op, "VpcEndpoint"):
		return s.eslVpcEndpoints(op, in)
	case strings.Contains(op, "CrossClusterSearchConnection"):
		return s.eslConnections(op, in)
	case strings.Contains(op, "Reserved"):
		return s.eslReserved(op, in)
	}
	return nil, eslErr(http.StatusNotImplemented, "NotImplementedException", op+" is not implemented")
}

// --- domains -------------------------------------------------------------

func eslSoftware() map[string]any {
	return map[string]any{
		"CurrentVersion": "R20231113-P1", "NewVersion": "", "UpdateAvailable": false, "Cancellable": false,
		"UpdateStatus": "COMPLETED", "Description": "There is no software update available for this domain.",
		"AutomatedUpdateDate": float64(0), "OptionalDeployment": false,
	}
}

func eslStatus(d *osDomain, processing bool) map[string]any {
	state := "Active"
	if processing {
		state = map[bool]string{true: "Creating", false: "Modifying"}[d.version == 1]
	}
	out := map[string]any{
		"DomainId": osDomainAccountID + "/" + d.name, "DomainName": d.name, "ARN": osDomainARN(d.name),
		"Created": true, "Deleted": false, "Processing": processing, "UpgradeProcessing": false,
		"ElasticsearchVersion":       eslVersionOf(d.engine),
		"ElasticsearchClusterConfig": eslClusterConfig(d.cluster, ".search", ".elasticsearch"),
		"ServiceSoftwareOptions":     eslSoftware(), "DomainProcessingStatus": state,
	}
	for k, v := range d.options {
		out[k] = v
	}
	if vpc := d.vpcDerived(); vpc != nil {
		out["VPCOptions"] = vpc
	}
	d.endpointFields(out)
	return out
}

func eslConfig(d *osDomain) map[string]any {
	status := map[string]any{
		"CreationDate": eslUnix(d.created), "UpdateDate": eslUnix(d.updated),
		"UpdateVersion": float64(d.version), "State": "Active", "PendingDeletion": false,
	}
	out := map[string]any{
		"ElasticsearchVersion":       map[string]any{"Options": eslVersionOf(d.engine), "Status": status},
		"ElasticsearchClusterConfig": map[string]any{"Options": eslClusterConfig(d.cluster, ".search", ".elasticsearch"), "Status": status},
		"ChangeProgressDetails":      map[string]any{"ChangeId": d.changeID, "Message": "Change completed."},
	}
	for k, v := range d.options {
		if k == "VPCOptions" {
			v = d.vpcDerived()
		}
		if k == "AutoTuneOptions" {
			opts, _ := v.(map[string]any)
			at := map[string]any{"CreationDate": status["CreationDate"], "UpdateDate": status["UpdateDate"],
				"UpdateVersion": status["UpdateVersion"], "State": "DISABLED", "PendingDeletion": false}
			if eslStr(opts, "DesiredState") == "ENABLED" {
				at["State"] = "ENABLED"
			}
			out[k] = map[string]any{"Options": v, "Status": at}
			continue
		}
		out[k] = map[string]any{"Options": v, "Status": status}
	}
	return out
}

// eslCheckCluster validates a merged OpenSearch-form cluster config.
func eslCheckCluster(c map[string]any) *eslError {
	if n, ok := eslNum(c, "InstanceCount"); ok && n < 1 {
		return eslValidation("InstanceCount must be at least 1")
	}
	if eslBool(c, "DedicatedMasterEnabled") {
		if eslStr(c, "DedicatedMasterType") == "" {
			return eslValidation("DedicatedMasterType is required when DedicatedMasterEnabled is true")
		}
		if n, _ := eslNum(c, "DedicatedMasterCount"); n != 3 && n != 5 {
			return eslValidation("DedicatedMasterCount must be 3 or 5")
		}
	}
	if eslBool(c, "ZoneAwarenessEnabled") {
		azs := 2.0
		if n, ok := eslNum(eslMap(c, "ZoneAwarenessConfig"), "AvailabilityZoneCount"); ok {
			azs = n
		}
		if count, _ := eslNum(c, "InstanceCount"); azs <= 0 || int(count)%int(azs) != 0 {
			return eslValidation(fmt.Sprintf("InstanceCount must be a multiple of the AvailabilityZoneCount (%v) when zone awareness is enabled", azs))
		}
	}
	if eslBool(c, "WarmEnabled") {
		if !eslBool(c, "DedicatedMasterEnabled") {
			return eslValidation("UltraWarm requires dedicated master nodes")
		}
		if n, _ := eslNum(c, "WarmCount"); eslStr(c, "WarmType") == "" || n < 2 {
			return eslValidation("WarmType and a WarmCount of at least 2 are required when WarmEnabled is true")
		}
	}
	return nil
}

func eslCheckOptions(in map[string]any) *eslError {
	if ebs := eslMap(in, "EBSOptions"); eslBool(ebs, "EBSEnabled") {
		if n, _ := eslNum(ebs, "VolumeSize"); n < 10 {
			return eslValidation("EBSOptions.VolumeSize must be at least 10 GiB when EBS is enabled")
		}
	}
	if p := eslStr(in, "AccessPolicies"); p != "" && !json.Valid([]byte(p)) {
		return eslValidation("AccessPolicies must be a valid JSON policy document")
	}
	return nil
}

func eslCheckTags(tags []any, existing int) *eslError {
	if existing+len(tags) > eslMaxTags {
		return eslConflict("LimitExceededException", fmt.Sprintf("a domain can have at most %d tags", eslMaxTags))
	}
	for _, t := range tags {
		if strings.HasPrefix(eslStr(t.(map[string]any), "Key"), "aws:") {
			return eslValidation("tag keys can't start with aws:")
		}
	}
	return nil
}

func (s *openSearchStore) eslCreateDomain(in map[string]any) (map[string]any, *eslError) {
	name := eslStr(in, "DomainName")
	if s.domains[name] != nil {
		return nil, eslConflict("ResourceAlreadyExistsException", "Domain already exists: "+name)
	}
	if len(s.domains) >= osMaxDomains {
		return nil, eslConflict("LimitExceededException", fmt.Sprintf("an account can have at most %d domains", osMaxDomains))
	}
	version := eslStr(in, "ElasticsearchVersion")
	if version == "" {
		version = eslDefaultVersion
	}
	if !slices.Contains(eslVersions, version) {
		return nil, eslValidation("Invalid ElasticsearchVersion: " + version)
	}
	cluster := map[string]any{"InstanceType": "m4.large.search", "InstanceCount": float64(1), "DedicatedMasterEnabled": false, "ZoneAwarenessEnabled": false}
	for k, v := range eslClusterConfig(eslMap(in, "ElasticsearchClusterConfig"), ".elasticsearch", ".search") {
		cluster[k] = v
	}
	if err := eslCheckCluster(cluster); err != nil {
		return nil, err
	}
	if err := eslCheckOptions(in); err != nil {
		return nil, err
	}
	tags := eslList(in, "TagList")
	if err := eslCheckTags(tags, 0); err != nil {
		return nil, err
	}
	d := s.newOSDomainLocked(name, eslEngineOf(version), cluster, in)
	for _, t := range tags {
		tag := t.(map[string]any)
		d.tags[eslStr(tag, "Key")] = eslStr(tag, "Value")
	}
	return map[string]any{"DomainStatus": eslStatus(d, true)}, nil
}

func (s *openSearchStore) eslUpdateDomain(in map[string]any) (map[string]any, *eslError) {
	d, err := s.eslDomain(eslStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	next := *d
	next.cluster = osCopyMap(d.cluster)
	next.options = osCopyMap(d.options)
	for k, v := range eslClusterConfig(eslMap(in, "ElasticsearchClusterConfig"), ".elasticsearch", ".search") {
		next.cluster[k] = v
	}
	if err := eslCheckCluster(next.cluster); err != nil {
		return nil, err
	}
	if err := eslCheckOptions(in); err != nil {
		return nil, err
	}
	next.setOptions(in)
	if eslBool(in, "DryRun") {
		deployment := "DynamicUpdate"
		if len(eslMap(in, "ElasticsearchClusterConfig")) > 0 {
			deployment = "Blue/Green"
		}
		return map[string]any{"DomainConfig": eslConfig(&next),
			"DryRunResults": map[string]any{"DeploymentType": deployment, "Message": "This change requires a " + deployment + " deployment."}}, nil
	}
	next.touch()
	*d = next
	return map[string]any{"DomainConfig": eslConfig(d)}, nil
}

func (s *openSearchStore) eslTags(op string, in map[string]any) (map[string]any, *eslError) {
	d, err := s.eslDomainByARN(eslStr(in, "ARN"))
	if err != nil {
		return nil, err
	}
	switch op {
	case "AddTags":
		tags := eslList(in, "TagList")
		added := 0
		for _, t := range tags {
			if _, ok := d.tags[eslStr(t.(map[string]any), "Key")]; !ok {
				added++
			}
		}
		if err := eslCheckTags(tags, len(d.tags)+added-len(tags)); err != nil {
			return nil, err
		}
		for _, t := range tags {
			tag := t.(map[string]any)
			d.tags[eslStr(tag, "Key")] = eslStr(tag, "Value")
		}
		return nil, nil
	case "RemoveTags":
		for _, k := range eslList(in, "TagKeys") {
			key, _ := k.(string)
			delete(d.tags, key)
		}
		return nil, nil
	}
	keys := make([]string, 0, len(d.tags))
	for k := range d.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	list := []any{}
	for _, k := range keys {
		list = append(list, map[string]any{"Key": k, "Value": d.tags[k]})
	}
	return map[string]any{"TagList": list}, nil
}

func (s *openSearchStore) eslCompatibleVersions(in map[string]any) (map[string]any, *eslError) {
	list := []any{}
	if name := eslStr(in, "DomainName"); name != "" {
		d, err := s.eslDomain(name)
		if err != nil {
			return nil, err
		}
		v := eslVersionOf(d.engine)
		list = append(list, map[string]any{"SourceVersion": v, "TargetVersions": eslUpgradeTargets(v)})
	} else {
		for _, v := range eslVersions {
			if targets := eslUpgradeTargets(v); len(targets) > 0 {
				list = append(list, map[string]any{"SourceVersion": v, "TargetVersions": targets})
			}
		}
	}
	return map[string]any{"CompatibleElasticsearchVersions": list}, nil
}

func (s *openSearchStore) eslUpgrade(op string, in map[string]any) (map[string]any, *eslError) {
	d, err := s.eslDomain(eslStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	switch op {
	case "UpgradeElasticsearchDomain":
		from, target := eslVersionOf(d.engine), eslStr(in, "TargetVersion")
		if !slices.Contains(eslUpgradeTargets(from), any(target)) {
			return nil, eslValidation(fmt.Sprintf("Cannot upgrade from %s to %s", from, target))
		}
		checkOnly := eslBool(in, "PerformCheckOnly")
		stepNames := []string{"PRE_UPGRADE_CHECK"}
		if !checkOnly {
			stepNames = append(stepNames, "SNAPSHOT", "UPGRADE")
		}
		steps := []any{}
		for _, step := range stepNames {
			steps = append(steps, map[string]any{"UpgradeStep": step, "UpgradeStepStatus": "SUCCEEDED", "Issues": []any{}, "ProgressPercent": float64(100)})
		}
		d.upgrades = append(d.upgrades, map[string]any{
			"UpgradeName": fmt.Sprintf("Upgrade from %s to %s", from, target), "StartTimestamp": eslUnix(time.Now()),
			"UpgradeStatus": "SUCCEEDED", "StepsList": steps,
		})
		if len(d.upgrades) > osMaxUpgradeHistory {
			d.upgrades = d.upgrades[len(d.upgrades)-osMaxUpgradeHistory:]
		}
		if !checkOnly {
			d.engine = eslEngineOf(target)
			d.touch()
		}
		return map[string]any{"DomainName": d.name, "TargetVersion": target, "PerformCheckOnly": checkOnly,
			"ChangeProgressDetails": map[string]any{"ChangeId": d.changeID, "Message": "Upgrade completed."}}, nil
	case "GetUpgradeHistory":
		list := []any{}
		for i := len(d.upgrades) - 1; i >= 0; i-- {
			list = append(list, d.upgrades[i])
		}
		page, next, err := eslPage(list, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{"UpgradeHistories": page, "NextToken": next}, nil
	}
	if len(d.upgrades) == 0 {
		return nil, eslNotFound("No upgrade has been run on domain " + d.name)
	}
	last := d.upgrades[len(d.upgrades)-1]
	steps := last["StepsList"].([]any)
	step := steps[len(steps)-1].(map[string]any)
	return map[string]any{"UpgradeStep": step["UpgradeStep"], "StepStatus": "SUCCEEDED", "UpgradeName": last["UpgradeName"]}, nil
}

func (s *openSearchStore) eslInstanceTypes(op string, in map[string]any) (map[string]any, *eslError) {
	if v := eslStr(in, "ElasticsearchVersion"); !slices.Contains(eslVersions, v) {
		return nil, eslValidation("Invalid ElasticsearchVersion: " + v)
	}
	if name := eslStr(in, "DomainName"); name != "" {
		if _, err := s.eslDomain(name); err != nil {
			return nil, err
		}
	}
	if op == "ListElasticsearchInstanceTypes" {
		types := []any{}
		for _, t := range awsmodels.ElasticsearchServiceModel().Shapes["ESPartitionInstanceType"].Enum {
			types = append(types, t)
		}
		page, next, err := eslPage(types, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{"ElasticsearchInstanceTypes": page, "NextToken": next}, nil
	}
	limits := func(maxCount float64) map[string]any {
		return map[string]any{
			"StorageTypes": []any{map[string]any{"StorageTypeName": "ebs", "StorageSubTypeName": "gp2", "StorageTypeLimits": []any{
				map[string]any{"LimitName": "MinimumVolumeSize", "LimitValues": []any{"10"}},
				map[string]any{"LimitName": "MaximumVolumeSize", "LimitValues": []any{"1024"}},
			}}},
			"InstanceLimits":   map[string]any{"InstanceCountLimits": map[string]any{"MinimumInstanceCount": float64(1), "MaximumInstanceCount": maxCount}},
			"AdditionalLimits": []any{},
		}
	}
	return map[string]any{"LimitsByRole": map[string]any{"data": limits(80), "master": limits(5)}}, nil
}

func (s *openSearchStore) eslDomainActions(op string, in map[string]any) (map[string]any, *eslError) {
	d, err := s.eslDomain(eslStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	switch op {
	case "DescribeDomainAutoTunes":
		return map[string]any{"AutoTunes": []any{}, "NextToken": ""}, nil
	case "DescribeDomainChangeProgress":
		if id := eslStr(in, "ChangeId"); id != "" && id != d.changeID {
			return nil, eslNotFound("Change not found: " + id)
		}
		return map[string]any{"ChangeProgressStatus": map[string]any{
			"ChangeId": d.changeID, "StartTime": eslUnix(d.updated), "Status": "COMPLETED",
			"PendingProperties": []any{}, "CompletedProperties": []any{}, "TotalNumberOfStages": float64(1),
			"ChangeProgressStages": []any{map[string]any{"Name": "Completed", "Status": "COMPLETED", "Description": "Change completed", "LastUpdated": eslUnix(d.updated)}},
		}}, nil
	case "CancelDomainConfigChange":
		// ponytail: changes apply immediately, so there is never one to cancel.
		return map[string]any{"DryRun": eslBool(in, "DryRun"), "CancelledChangeIds": []any{}, "CancelledChangeProperties": []any{}}, nil
	case "UpdateClone":
		return map[string]any{"DomainName": d.name, "ChangeId": osNewChangeID()}, nil
	}
	return map[string]any{"ServiceSoftwareOptions": eslSoftware()}, nil
}

func (s *openSearchStore) eslVpcAccess(op string, in map[string]any) (map[string]any, *eslError) {
	d, err := s.eslDomain(eslStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	account := eslStr(in, "Account")
	switch op {
	case "AuthorizeVpcEndpointAccess":
		if !slices.Contains(d.access, account) {
			if len(d.access) >= eslMaxAccess {
				return nil, eslConflict("LimitExceededException", fmt.Sprintf("at most %d accounts can be authorized", eslMaxAccess))
			}
			d.access = append(d.access, account)
		}
		return map[string]any{"AuthorizedPrincipal": map[string]any{"PrincipalType": "AWS_ACCOUNT", "Principal": account}}, nil
	case "RevokeVpcEndpointAccess":
		i := slices.Index(d.access, account)
		if i < 0 {
			return nil, eslNotFound("Account " + account + " isn't authorized for " + d.name)
		}
		d.access = slices.Delete(d.access, i, i+1)
		return map[string]any{}, nil
	}
	list := []any{}
	for _, a := range d.access {
		list = append(list, map[string]any{"PrincipalType": "AWS_ACCOUNT", "Principal": a})
	}
	return map[string]any{"AuthorizedPrincipalList": list, "NextToken": ""}, nil
}

// --- packages ------------------------------------------------------------

func (s *openSearchStore) eslPackage(id string) (*eslPackage, *eslError) {
	if p := s.esl.packages[id]; p != nil {
		return p, nil
	}
	return nil, eslNotFound("Package not found: " + id)
}

func eslDomainPackage(d *osDomain, p *eslPackage, status string) map[string]any {
	return map[string]any{
		"PackageID": p.details["PackageID"], "PackageName": p.details["PackageName"], "PackageType": p.details["PackageType"],
		"LastUpdated": eslUnix(d.packages[eslStr(p.details, "PackageID")]), "DomainName": d.name,
		"DomainPackageStatus": status, "PackageVersion": p.details["AvailablePackageVersion"],
		"ReferencePath": "analyzers/" + eslStr(p.details, "PackageID"),
	}
}

func (s *openSearchStore) eslPackages(op string, in map[string]any) (map[string]any, *eslError) {
	st := &s.esl
	now := eslUnix(time.Now())
	switch op {
	case "CreatePackage":
		src := eslMap(in, "PackageSource")
		if eslStr(src, "S3BucketName") == "" || eslStr(src, "S3Key") == "" {
			return nil, eslValidation("PackageSource needs S3BucketName and S3Key")
		}
		for _, p := range st.packages {
			if p.details["PackageName"] == eslStr(in, "PackageName") {
				return nil, eslConflict("ResourceAlreadyExistsException", "Package already exists: "+eslStr(in, "PackageName"))
			}
		}
		if len(st.packages) >= eslMaxPackages {
			return nil, eslConflict("LimitExceededException", fmt.Sprintf("an account can have at most %d packages", eslMaxPackages))
		}
		st.seq++
		id := fmt.Sprintf("F%09d", 100000000+st.seq)
		p := &eslPackage{version: 1, details: map[string]any{
			"PackageID": id, "PackageName": eslStr(in, "PackageName"), "PackageType": eslStr(in, "PackageType"),
			"PackageDescription": eslStr(in, "PackageDescription"), "PackageStatus": "AVAILABLE",
			"CreatedAt": now, "LastUpdatedAt": now, "AvailablePackageVersion": "v1",
		}}
		p.history = []any{map[string]any{"PackageVersion": "v1", "CommitMessage": "", "CreatedAt": now}}
		st.packages[id] = p
		return map[string]any{"PackageDetails": osCopyMap(p.details)}, nil
	case "DescribePackages":
		ids := make([]string, 0, len(st.packages))
		for id := range st.packages {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		filters := map[string][]any{}
		for _, f := range eslList(in, "Filters") {
			fm := f.(map[string]any)
			filters[eslStr(fm, "Name")] = eslList(fm, "Value")
		}
		list := []any{}
		for _, id := range ids {
			p := st.packages[id]
			keep := true
			for name, values := range filters {
				if len(values) > 0 && !slices.Contains(values, p.details[name]) {
					keep = false
				}
			}
			if keep {
				list = append(list, osCopyMap(p.details))
			}
		}
		page, next, err := eslPage(list, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{"PackageDetailsList": page, "NextToken": next}, nil
	}
	if op == "ListPackagesForDomain" {
		d, err := s.eslDomain(eslStr(in, "DomainName"))
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(d.packages))
		for id := range d.packages {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		list := []any{}
		for _, id := range ids {
			if p := st.packages[id]; p != nil {
				list = append(list, eslDomainPackage(d, p, "ACTIVE"))
			}
		}
		page, next, err := eslPage(list, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{"DomainPackageDetailsList": page, "NextToken": next}, nil
	}
	p, err := s.eslPackage(eslStr(in, "PackageID"))
	if err != nil {
		return nil, err
	}
	id := eslStr(p.details, "PackageID")
	switch op {
	case "UpdatePackage":
		p.version++
		version := fmt.Sprintf("v%d", p.version)
		p.details["AvailablePackageVersion"], p.details["LastUpdatedAt"] = version, now
		if desc := eslStr(in, "PackageDescription"); desc != "" {
			p.details["PackageDescription"] = desc
		}
		p.history = append(p.history, map[string]any{"PackageVersion": version, "CommitMessage": eslStr(in, "CommitMessage"), "CreatedAt": now})
		if len(p.history) > 100 {
			p.history = p.history[1:]
		}
		return map[string]any{"PackageDetails": osCopyMap(p.details)}, nil
	case "DeletePackage":
		for _, d := range s.domains {
			if _, ok := d.packages[id]; ok {
				return nil, eslConflict("ConflictException", "Package "+id+" is associated with domain "+d.name)
			}
		}
		delete(st.packages, id)
		details := osCopyMap(p.details)
		details["PackageStatus"] = "DELETED"
		return map[string]any{"PackageDetails": details}, nil
	case "GetPackageVersionHistory":
		list := []any{}
		for i := len(p.history) - 1; i >= 0; i-- {
			list = append(list, p.history[i])
		}
		page, next, err := eslPage(list, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{"PackageID": id, "PackageVersionHistoryList": page, "NextToken": next}, nil
	case "ListDomainsForPackage":
		list := []any{}
		for _, d := range s.sortedDomainsLocked() {
			if _, ok := d.packages[id]; ok {
				list = append(list, eslDomainPackage(d, p, "ACTIVE"))
			}
		}
		page, next, err := eslPage(list, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{"DomainPackageDetailsList": page, "NextToken": next}, nil
	}
	d, derr := s.eslDomain(eslStr(in, "DomainName"))
	if derr != nil {
		return nil, derr
	}
	switch op {
	case "AssociatePackage":
		if _, ok := d.packages[id]; ok {
			return nil, eslConflict("ConflictException", "Package "+id+" is already associated with "+d.name)
		}
		// ponytail: association completes at once (AWS reports ASSOCIATING first).
		d.packages[id] = time.Now().UTC()
		return map[string]any{"DomainPackageDetails": eslDomainPackage(d, p, "ACTIVE")}, nil
	case "DissociatePackage":
		if _, ok := d.packages[id]; !ok {
			return nil, eslNotFound("Package " + id + " isn't associated with " + d.name)
		}
		details := eslDomainPackage(d, p, "DISSOCIATING")
		delete(d.packages, id)
		return map[string]any{"DomainPackageDetails": details}, nil
	}
	return nil, eslErr(http.StatusNotImplemented, "NotImplementedException", op+" is not implemented")
}

// --- VPC endpoints -----------------------------------------------------------

func (s *openSearchStore) eslVpcEndpoints(op string, in map[string]any) (map[string]any, *eslError) {
	st := &s.esl
	summary := func(e map[string]any) map[string]any {
		return map[string]any{"VpcEndpointId": e["VpcEndpointId"], "VpcEndpointOwner": e["VpcEndpointOwner"], "DomainArn": e["DomainArn"], "Status": e["Status"]}
	}
	ids := make([]string, 0, len(st.endpoints))
	for id := range st.endpoints {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	vpcOptions := func(o map[string]any) (map[string]any, *eslError) {
		if len(eslList(o, "SubnetIds")) == 0 {
			return nil, eslValidation("VpcOptions.SubnetIds is required")
		}
		out := osCopyMap(o)
		out["VPCId"], out["AvailabilityZones"] = "vpc-0"+osShortID(eslStr(o, "SubnetIds"))[:7], []any{osDomainRegion + "a"}
		return out, nil
	}
	switch op {
	case "CreateVpcEndpoint":
		if tok := eslStr(in, "ClientToken"); tok != "" {
			if id, ok := st.clientTokens[tok]; ok && st.endpoints[id] != nil {
				return map[string]any{"VpcEndpoint": st.endpoints[id]}, nil
			}
		}
		if _, err := s.eslDomainByARN(eslStr(in, "DomainArn")); err != nil {
			return nil, err
		}
		opts, err := vpcOptions(eslMap(in, "VpcOptions"))
		if err != nil {
			return nil, err
		}
		if len(st.endpoints) >= eslMaxEndpoints {
			return nil, eslConflict("LimitExceededException", fmt.Sprintf("an account can have at most %d VPC endpoints", eslMaxEndpoints))
		}
		st.seq++
		id := fmt.Sprintf("aos-%012x", st.seq)
		e := map[string]any{"VpcEndpointId": id, "VpcEndpointOwner": osDomainAccountID, "DomainArn": eslStr(in, "DomainArn"),
			"VpcOptions": opts, "Status": "ACTIVE", "Endpoint": id + "." + osDomainRegion + ".es.amazonaws.com"}
		st.endpoints[id] = e
		if tok := eslStr(in, "ClientToken"); tok != "" {
			st.clientTokens[tok] = id
		}
		return map[string]any{"VpcEndpoint": e}, nil
	case "DescribeVpcEndpoints":
		found, errs := []any{}, []any{}
		for _, v := range eslList(in, "VpcEndpointIds") {
			id, _ := v.(string)
			if e := st.endpoints[id]; e != nil {
				found = append(found, e)
			} else {
				errs = append(errs, map[string]any{"VpcEndpointId": id, "ErrorCode": "ENDPOINT_NOT_FOUND", "ErrorMessage": "VPC endpoint not found: " + id})
			}
		}
		return map[string]any{"VpcEndpoints": found, "VpcEndpointErrors": errs}, nil
	case "ListVpcEndpoints", "ListVpcEndpointsForDomain":
		arn := ""
		if op == "ListVpcEndpointsForDomain" {
			d, err := s.eslDomain(eslStr(in, "DomainName"))
			if err != nil {
				return nil, err
			}
			arn = osDomainARN(d.name)
		}
		list := []any{}
		for _, id := range ids {
			if e := st.endpoints[id]; arn == "" || e["DomainArn"] == arn {
				list = append(list, summary(e))
			}
		}
		return map[string]any{"VpcEndpointSummaryList": list, "NextToken": ""}, nil
	}
	e := st.endpoints[eslStr(in, "VpcEndpointId")]
	if e == nil {
		return nil, eslNotFound("VPC endpoint not found: " + eslStr(in, "VpcEndpointId"))
	}
	if op == "UpdateVpcEndpoint" {
		opts, err := vpcOptions(eslMap(in, "VpcOptions"))
		if err != nil {
			return nil, err
		}
		e["VpcOptions"] = opts
		return map[string]any{"VpcEndpoint": e}, nil
	}
	delete(st.endpoints, eslStr(e, "VpcEndpointId"))
	sum := summary(e)
	sum["Status"] = "DELETING"
	return map[string]any{"VpcEndpointSummary": sum}, nil
}

// --- cross-cluster search connections ---------------------------------------

func (c *eslConnection) outbound() map[string]any {
	return map[string]any{"SourceDomainInfo": c.source, "DestinationDomainInfo": c.destination, "CrossClusterSearchConnectionId": c.id,
		"ConnectionAlias": c.alias, "ConnectionStatus": map[string]any{"StatusCode": c.outStatus, "Message": c.outMessage}}
}

func (c *eslConnection) inbound() map[string]any {
	return map[string]any{"SourceDomainInfo": c.source, "DestinationDomainInfo": c.destination, "CrossClusterSearchConnectionId": c.id,
		"ConnectionStatus": map[string]any{"StatusCode": c.inStatus, "Message": c.inMessage}}
}

// eslLocalDomain checks a DomainInformation that names this account and region.
func (s *openSearchStore) eslLocalDomain(info map[string]any) *eslError {
	owner, region := eslStr(info, "OwnerId"), eslStr(info, "Region")
	if (owner == "" || owner == osDomainAccountID) && (region == "" || region == osDomainRegion) {
		_, err := s.eslDomain(eslStr(info, "DomainName"))
		return err
	}
	return nil
}

func (s *openSearchStore) eslConnections(op string, in map[string]any) (map[string]any, *eslError) {
	st := &s.esl
	switch op {
	case "CreateOutboundCrossClusterSearchConnection":
		src, dst := eslMap(in, "SourceDomainInfo"), eslMap(in, "DestinationDomainInfo")
		if err := s.eslLocalDomain(src); err != nil {
			return nil, err
		}
		if err := s.eslLocalDomain(dst); err != nil {
			return nil, err
		}
		if len(st.connections) >= eslMaxConnections {
			return nil, eslConflict("LimitExceededException", fmt.Sprintf("an account can have at most %d connections", eslMaxConnections))
		}
		st.seq++
		c := &eslConnection{id: fmt.Sprintf("cccs-%012x", st.seq), alias: eslStr(in, "ConnectionAlias"), source: src, destination: dst,
			outStatus: "PENDING_ACCEPTANCE", outMessage: "Connection is waiting for acceptance by the destination domain owner.",
			inStatus: "PENDING_ACCEPTANCE", inMessage: "Connection is waiting for your acceptance."}
		st.connections[c.id] = c
		out := c.outbound()
		return out, nil
	case "DescribeOutboundCrossClusterSearchConnections", "DescribeInboundCrossClusterSearchConnections":
		filterPaths := map[string][2]string{
			"cross-cluster-search-connection-id":  {"", ""},
			"source-domain-info.domain-name":      {"source", "DomainName"},
			"source-domain-info.owner-id":         {"source", "OwnerId"},
			"destination-domain-info.domain-name": {"destination", "DomainName"},
			"destination-domain-info.owner-id":    {"destination", "OwnerId"},
		}
		for _, f := range eslList(in, "Filters") {
			if _, ok := filterPaths[eslStr(f.(map[string]any), "Name")]; !ok {
				return nil, eslValidation("unknown filter name " + eslStr(f.(map[string]any), "Name"))
			}
		}
		ids := make([]string, 0, len(st.connections))
		for id := range st.connections {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		list := []any{}
		for _, id := range ids {
			c := st.connections[id]
			keep := true
			for _, f := range eslList(in, "Filters") {
				fm := f.(map[string]any)
				path := filterPaths[eslStr(fm, "Name")]
				value := c.id
				if path[0] == "source" {
					value = eslStr(c.source, path[1])
				} else if path[0] == "destination" {
					value = eslStr(c.destination, path[1])
				}
				if vals := eslList(fm, "Values"); len(vals) > 0 && !slices.Contains(vals, any(value)) {
					keep = false
				}
			}
			if keep {
				if strings.Contains(op, "Outbound") {
					list = append(list, c.outbound())
				} else {
					list = append(list, c.inbound())
				}
			}
		}
		page, next, err := eslPage(list, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{"CrossClusterSearchConnections": page, "NextToken": next}, nil
	}
	c := st.connections[eslStr(in, "CrossClusterSearchConnectionId")]
	if c == nil {
		return nil, eslNotFound("Connection not found: " + eslStr(in, "CrossClusterSearchConnectionId"))
	}
	switch op {
	case "AcceptInboundCrossClusterSearchConnection":
		if c.inStatus != "PENDING_ACCEPTANCE" {
			return nil, eslConflict("DisabledOperationException", "Connection "+c.id+" is "+c.inStatus+", not PENDING_ACCEPTANCE")
		}
		c.inStatus, c.inMessage, c.outStatus, c.outMessage = "APPROVED", "Connection is approved.", "ACTIVE", "Connection is active."
		return map[string]any{"CrossClusterSearchConnection": c.inbound()}, nil
	case "RejectInboundCrossClusterSearchConnection":
		if c.inStatus != "PENDING_ACCEPTANCE" {
			return nil, eslConflict("DisabledOperationException", "Connection "+c.id+" is "+c.inStatus+", not PENDING_ACCEPTANCE")
		}
		c.inStatus, c.inMessage, c.outStatus, c.outMessage = "REJECTED", "Connection is rejected.", "REJECTED", "Connection was rejected."
		return map[string]any{"CrossClusterSearchConnection": c.inbound()}, nil
	}
	delete(st.connections, c.id)
	c.inStatus, c.outStatus = "DELETED", "DELETED"
	c.inMessage, c.outMessage = "Connection is deleted.", "Connection is deleted."
	if strings.Contains(op, "Outbound") {
		return map[string]any{"CrossClusterSearchConnection": c.outbound()}, nil
	}
	return map[string]any{"CrossClusterSearchConnection": c.inbound()}, nil
}

// --- reserved instances ------------------------------------------------------

var eslOfferings = func() []map[string]any {
	out := []map[string]any{}
	for i, t := range []string{"m5.large.elasticsearch", "r5.large.elasticsearch", "c5.large.elasticsearch"} {
		out = append(out, map[string]any{
			"ReservedElasticsearchInstanceOfferingId": fmt.Sprintf("2f0c1e7a-0000-4000-8000-00000000000%d", i+1),
			"ElasticsearchInstanceType":               t, "Duration": float64(31536000), "FixedPrice": float64(0),
			"UsagePrice": float64(0), "CurrencyCode": "USD", "PaymentOption": "NO_UPFRONT",
			"RecurringCharges": []any{map[string]any{"RecurringChargeAmount": 0.098 + float64(i)*0.02, "RecurringChargeFrequency": "Hourly"}},
		})
	}
	return out
}()

func (s *openSearchStore) eslReserved(op string, in map[string]any) (map[string]any, *eslError) {
	st := &s.esl
	offering := func(id string) map[string]any {
		for _, o := range eslOfferings {
			if o["ReservedElasticsearchInstanceOfferingId"] == id {
				return o
			}
		}
		return nil
	}
	switch op {
	case "DescribeReservedElasticsearchInstanceOfferings":
		list := []any{}
		id := eslStr(in, "ReservedElasticsearchInstanceOfferingId")
		for _, o := range eslOfferings {
			if id == "" || o["ReservedElasticsearchInstanceOfferingId"] == id {
				list = append(list, osCopyMap(o))
			}
		}
		if id != "" && len(list) == 0 {
			return nil, eslNotFound("Offering not found: " + id)
		}
		page, next, err := eslPage(list, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{"ReservedElasticsearchInstanceOfferings": page, "NextToken": next}, nil
	case "PurchaseReservedElasticsearchInstanceOffering":
		o := offering(eslStr(in, "ReservedElasticsearchInstanceOfferingId"))
		if o == nil {
			return nil, eslNotFound("Offering not found: " + eslStr(in, "ReservedElasticsearchInstanceOfferingId"))
		}
		for _, r := range st.reserved {
			if r["ReservationName"] == eslStr(in, "ReservationName") {
				return nil, eslConflict("ResourceAlreadyExistsException", "Reservation already exists: "+eslStr(in, "ReservationName"))
			}
		}
		if len(st.reserved) >= eslMaxReservations {
			return nil, eslConflict("LimitExceededException", fmt.Sprintf("an account can have at most %d reservations", eslMaxReservations))
		}
		count := float64(1)
		if n, ok := eslNum(in, "InstanceCount"); ok {
			count = n
		}
		id := uuid.NewString()
		r := osCopyMap(o)
		r["ReservationName"], r["ReservedElasticsearchInstanceId"], r["StartTime"] = eslStr(in, "ReservationName"), id, eslUnix(time.Now())
		r["ElasticsearchInstanceCount"], r["State"] = count, "active"
		st.reserved[id] = r
		return map[string]any{"ReservedElasticsearchInstanceId": id, "ReservationName": r["ReservationName"]}, nil
	}
	id := eslStr(in, "ReservedElasticsearchInstanceId")
	if id != "" && st.reserved[id] == nil {
		return nil, eslNotFound("Reservation not found: " + id)
	}
	ids := make([]string, 0, len(st.reserved))
	for rid := range st.reserved {
		if id == "" || rid == id {
			ids = append(ids, rid)
		}
	}
	sort.Strings(ids)
	list := []any{}
	for _, rid := range ids {
		list = append(list, osCopyMap(st.reserved[rid]))
	}
	page, next, err := eslPage(list, in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ReservedElasticsearchInstances": page, "NextToken": next}, nil
}
