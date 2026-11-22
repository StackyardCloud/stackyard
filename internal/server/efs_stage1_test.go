package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/services/ec2"
)

func TestEFSStage1FileSystemLifecycle(t *testing.T) {
	_, ts := newEFSTestServer(t)

	out := efsCall(t, ts, http.MethodPost, "file-systems", `{"CreationToken":"tok-1","Encrypted":true,"Tags":[{"Key":"Name","Value":"demo"}]}`, http.StatusCreated)
	fsID := out["FileSystemId"].(string)
	if out["LifeCycleState"] != "creating" || out["Name"] != "demo" || out["ThroughputMode"] != "bursting" || out["PerformanceMode"] != "generalPurpose" {
		t.Fatalf("create response: %v", out)
	}
	if kms, _ := out["KmsKeyId"].(string); !strings.HasPrefix(kms, "arn:aws:kms:us-east-1:123456789012:key/") {
		t.Fatalf("default KMS key: %v", out["KmsKeyId"])
	}
	if size := out["SizeInBytes"].(map[string]any); size["Value"] != float64(efsEmptySizeBytes) {
		t.Fatalf("SizeInBytes: %v", size)
	}

	// The same creation token is rejected with the existing ID.
	dup := efsCallError(t, ts, http.MethodPost, "file-systems", `{"CreationToken":"tok-1"}`, http.StatusConflict, "FileSystemAlreadyExists")
	if dup["FileSystemId"] != fsID {
		t.Fatalf("FileSystemAlreadyExists FileSystemId = %v, want %s", dup["FileSystemId"], fsID)
	}

	list := efsCall(t, ts, http.MethodGet, "file-systems?FileSystemId="+fsID, "", http.StatusOK)
	fss := list["FileSystems"].([]any)
	if len(fss) != 1 || fss[0].(map[string]any)["LifeCycleState"] != "available" {
		t.Fatalf("describe: %v", list)
	}
	byToken := efsCall(t, ts, http.MethodGet, "file-systems?CreationToken=tok-1", "", http.StatusOK)
	if len(byToken["FileSystems"].([]any)) != 1 {
		t.Fatalf("filter by token: %v", byToken)
	}
	efsCallError(t, ts, http.MethodGet, "file-systems?FileSystemId=fs-0123456789abcdef0", "", http.StatusNotFound, "FileSystemNotFound")
	efsCallError(t, ts, http.MethodGet, "file-systems?FileSystemId=nope", "", http.StatusBadRequest, "BadRequest")

	upd := efsCall(t, ts, http.MethodPut, "file-systems/"+fsID, `{"ThroughputMode":"provisioned","ProvisionedThroughputInMibps":128}`, http.StatusAccepted)
	if upd["LifeCycleState"] != "updating" || upd["ProvisionedThroughputInMibps"] != float64(128) {
		t.Fatalf("update: %v", upd)
	}
	efsCallError(t, ts, http.MethodPut, "file-systems/"+fsID, `{}`, http.StatusBadRequest, "BadRequest")
	efsCallError(t, ts, http.MethodPut, "file-systems/"+fsID, `{"ThroughputMode":"elastic","ProvisionedThroughputInMibps":5}`, http.StatusBadRequest, "BadRequest")

	efsCall(t, ts, http.MethodDelete, "file-systems/"+fsID, "", http.StatusNoContent)
	efsCallError(t, ts, http.MethodDelete, "file-systems/"+fsID, "", http.StatusNotFound, "FileSystemNotFound")
}

func TestEFSStage1CreateValidation(t *testing.T) {
	_, ts := newEFSTestServer(t)
	for name, body := range map[string]string{
		"no token":                 `{}`,
		"bad performance":          `{"CreationToken":"a","PerformanceMode":"fast"}`,
		"provisioned without rate": `{"CreationToken":"a","ThroughputMode":"provisioned"}`,
		"rate without provisioned": `{"CreationToken":"a","ProvisionedThroughputInMibps":10}`,
		"maxIO with elastic":       `{"CreationToken":"a","PerformanceMode":"maxIO","ThroughputMode":"elastic"}`,
		"maxIO one zone":           `{"CreationToken":"a","PerformanceMode":"maxIO","AvailabilityZoneName":"us-east-1a"}`,
		"kms without encryption":   `{"CreationToken":"a","KmsKeyId":"alias/x"}`,
		"reserved tag":             `{"CreationToken":"a","Tags":[{"Key":"aws:x","Value":"y"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			efsCallError(t, ts, http.MethodPost, "file-systems", body, http.StatusBadRequest, "BadRequest")
		})
	}
	efsCallError(t, ts, http.MethodPost, "file-systems", `{"CreationToken":"a","ThroughputMode":"provisioned","ProvisionedThroughputInMibps":99999}`, http.StatusBadRequest, "ThroughputLimitExceeded")
	efsCallError(t, ts, http.MethodPost, "file-systems", `{"CreationToken":"a","AvailabilityZoneName":"eu-west-1a"}`, http.StatusBadRequest, "UnsupportedAvailabilityZone")

	// One Zone file systems get backups by default and report their AZ ID.
	out := efsCall(t, ts, http.MethodPost, "file-systems", `{"CreationToken":"oz","AvailabilityZoneName":"us-east-1b"}`, http.StatusCreated)
	if out["AvailabilityZoneId"] != "use1-az2" {
		t.Fatalf("AvailabilityZoneId: %v", out["AvailabilityZoneId"])
	}
	backup := efsCall(t, ts, http.MethodGet, "file-systems/"+out["FileSystemId"].(string)+"/backup-policy", "", http.StatusOK)
	if backup["BackupPolicy"].(map[string]any)["Status"] != "ENABLED" {
		t.Fatalf("One Zone backup default: %v", backup)
	}
}

func TestEFSStage1MountTargets(t *testing.T) {
	srv, ts := newEFSTestServer(t)
	fsID := efsCreateFS(t, ts, `{"CreationToken":"mt"}`)

	mt := efsCall(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"`+fsID+`","SubnetId":"subnet-00000001"}`, http.StatusOK)
	mtID := mt["MountTargetId"].(string)
	if mt["IpAddress"] != "10.0.0.4" || mt["LifeCycleState"] != "creating" || mt["VpcId"] != "vpc-00000001" || mt["AvailabilityZoneId"] != "use1-az1" {
		t.Fatalf("mount target: %v", mt)
	}
	groups := efsCall(t, ts, http.MethodGet, "mount-targets/"+mtID+"/security-groups", "", http.StatusOK)
	if g := groups["SecurityGroups"].([]any); len(g) != 1 || g[0] != "sg-00000000" {
		t.Fatalf("default security group: %v", groups)
	}

	// One mount target per AZ; a second subnet in another AZ is fine.
	efsCallError(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"`+fsID+`","SubnetId":"subnet-00000001"}`, http.StatusConflict, "MountTargetConflict")
	sub, err := srv.ec2.CreateSubnet("vpc-00000001", "10.0.1.0/24", "us-east-1b", nil)
	if err != nil {
		t.Fatal(err)
	}
	efsCallError(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"`+fsID+`","SubnetId":"`+sub.ID+`","IpAddress":"10.0.0.9"}`, http.StatusBadRequest, "BadRequest")
	efsCallError(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"`+fsID+`","SubnetId":"`+sub.ID+`","SecurityGroups":["sg-0000000000000dead"]}`, http.StatusBadRequest, "SecurityGroupNotFound")
	efsCall(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"`+fsID+`","SubnetId":"`+sub.ID+`","IpAddress":"10.0.1.50"}`, http.StatusOK)

	// A subnet in a different VPC conflicts with the existing mount targets.
	vpc, err := srv.ec2.CreateVpc("10.9.0.0/16", nil)
	if err != nil {
		t.Fatal(err)
	}
	other, err := srv.ec2.CreateSubnet(vpc.ID, "10.9.0.0/24", "us-east-1c", nil)
	if err != nil {
		t.Fatal(err)
	}
	efsCallError(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"`+fsID+`","SubnetId":"`+other.ID+`"}`, http.StatusConflict, "MountTargetConflict")

	// The same address can't be used twice in a subnet.
	fs2 := efsCreateFS(t, ts, `{"CreationToken":"mt2"}`)
	efsCallError(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"`+fs2+`","SubnetId":"`+sub.ID+`","IpAddress":"10.0.1.50"}`, http.StatusConflict, "IpAddressInUse")
	efsCallError(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"`+fs2+`","SubnetId":"subnet-0000000000000beef"}`, http.StatusBadRequest, "SubnetNotFound")
	efsCallError(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"fs-0000000000000beef","SubnetId":"subnet-00000001"}`, http.StatusNotFound, "FileSystemNotFound")

	list := efsCall(t, ts, http.MethodGet, "mount-targets?FileSystemId="+fsID+"&MaxItems=1", "", http.StatusOK)
	if len(list["MountTargets"].([]any)) != 1 || list["NextMarker"] == nil {
		t.Fatalf("page 1: %v", list)
	}
	page2 := efsCall(t, ts, http.MethodGet, "mount-targets?FileSystemId="+fsID+"&Marker="+list["NextMarker"].(string), "", http.StatusOK)
	if len(page2["MountTargets"].([]any)) != 1 || page2["NextMarker"] != nil {
		t.Fatalf("page 2: %v", page2)
	}
	efsCallError(t, ts, http.MethodGet, "mount-targets", "", http.StatusBadRequest, "BadRequest")

	// Security groups can be replaced with groups from the same VPC.
	sg, err := srv.ec2.CreateSecurityGroup("efs", "efs clients", "vpc-00000001", []ec2.Tag{})
	if err != nil {
		t.Fatal(err)
	}
	efsCall(t, ts, http.MethodPut, "mount-targets/"+mtID+"/security-groups", `{"SecurityGroups":["`+sg.ID+`"]}`, http.StatusNoContent)
	groups = efsCall(t, ts, http.MethodGet, "mount-targets/"+mtID+"/security-groups", "", http.StatusOK)
	if g := groups["SecurityGroups"].([]any); len(g) != 1 || g[0] != sg.ID {
		t.Fatalf("modified groups: %v", groups)
	}
	efsCallError(t, ts, http.MethodPut, "mount-targets/"+mtID+"/security-groups", `{"SecurityGroups":["a","b","c","d","e","f"]}`, http.StatusBadRequest, "SecurityGroupLimitExceeded")

	// A file system with mount targets can't be deleted.
	efsCallError(t, ts, http.MethodDelete, "file-systems/"+fsID, "", http.StatusConflict, "FileSystemInUse")
	efsCall(t, ts, http.MethodDelete, "mount-targets/"+mtID, "", http.StatusNoContent)
	efsCallError(t, ts, http.MethodDelete, "mount-targets/"+mtID, "", http.StatusNotFound, "MountTargetNotFound")
	fsDesc := efsCall(t, ts, http.MethodGet, "file-systems?FileSystemId="+fsID, "", http.StatusOK)
	if n := fsDesc["FileSystems"].([]any)[0].(map[string]any)["NumberOfMountTargets"]; n != float64(1) {
		t.Fatalf("NumberOfMountTargets = %v, want 1", n)
	}
}

func TestEFSStage1AccessPointsAndTags(t *testing.T) {
	_, ts := newEFSTestServer(t)
	fsID := efsCreateFS(t, ts, `{"CreationToken":"ap","Tags":[{"Key":"team","Value":"a"}]}`)

	body := `{"ClientToken":"c1","FileSystemId":"` + fsID + `","PosixUser":{"Uid":1000,"Gid":1000},"RootDirectory":{"Path":"/data","CreationInfo":{"OwnerUid":1000,"OwnerGid":1000,"Permissions":"755"}},"Tags":[{"Key":"Name","Value":"app"}]}`
	ap := efsCall(t, ts, http.MethodPost, "access-points", body, http.StatusOK)
	apID := ap["AccessPointId"].(string)
	if ap["Name"] != "app" || ap["RootDirectory"].(map[string]any)["Path"] != "/data" || !strings.HasSuffix(ap["AccessPointArn"].(string), "access-point/"+apID) {
		t.Fatalf("access point: %v", ap)
	}
	dup := efsCallError(t, ts, http.MethodPost, "access-points", body, http.StatusConflict, "AccessPointAlreadyExists")
	if dup["AccessPointId"] != apID {
		t.Fatalf("AccessPointAlreadyExists AccessPointId = %v", dup["AccessPointId"])
	}
	for name, bad := range map[string]string{
		"no posix gid":   `{"ClientToken":"c2","FileSystemId":"` + fsID + `","PosixUser":{"Uid":1}}`,
		"relative path":  `{"ClientToken":"c2","FileSystemId":"` + fsID + `","RootDirectory":{"Path":"data"}}`,
		"bad perms":      `{"ClientToken":"c2","FileSystemId":"` + fsID + `","RootDirectory":{"Path":"/d","CreationInfo":{"OwnerUid":1,"OwnerGid":1,"Permissions":"999"}}}`,
		"no client tok":  `{"FileSystemId":"` + fsID + `"}`,
		"negative owner": `{"ClientToken":"c2","FileSystemId":"` + fsID + `","PosixUser":{"Uid":-1,"Gid":1}}`,
	} {
		t.Run(name, func(t *testing.T) {
			efsCallError(t, ts, http.MethodPost, "access-points", bad, http.StatusBadRequest, "BadRequest")
		})
	}
	efsCallError(t, ts, http.MethodGet, "access-points?AccessPointId="+apID+"&FileSystemId="+fsID, "", http.StatusBadRequest, "BadRequest")
	listed := efsCall(t, ts, http.MethodGet, "access-points?FileSystemId="+fsID, "", http.StatusOK)
	if len(listed["AccessPoints"].([]any)) != 1 {
		t.Fatalf("access points: %v", listed)
	}

	// Mount targets can be looked up through an access point.
	efsCall(t, ts, http.MethodPost, "mount-targets", `{"FileSystemId":"`+fsID+`","SubnetId":"subnet-00000001"}`, http.StatusOK)
	viaAP := efsCall(t, ts, http.MethodGet, "mount-targets?AccessPointId="+apID, "", http.StatusOK)
	if len(viaAP["MountTargets"].([]any)) != 1 {
		t.Fatalf("mount targets via access point: %v", viaAP)
	}

	// Resource tags on an access point by ARN, and legacy tags on the file system.
	arn := "arn%3Aaws%3Aelasticfilesystem%3Aus-east-1%3A123456789012%3Aaccess-point%2F" + apID
	efsCall(t, ts, http.MethodPost, "resource-tags/"+arn, `{"Tags":[{"Key":"env","Value":"dev"}]}`, http.StatusOK)
	tags := efsCall(t, ts, http.MethodGet, "resource-tags/"+apID, "", http.StatusOK)
	if len(tags["Tags"].([]any)) != 2 {
		t.Fatalf("access point tags: %v", tags)
	}
	efsCall(t, ts, http.MethodDelete, "resource-tags/"+apID+"?tagKeys=env&tagKeys=Name", "", http.StatusOK)
	if tags := efsCall(t, ts, http.MethodGet, "resource-tags/"+apID, "", http.StatusOK); len(tags["Tags"].([]any)) != 0 {
		t.Fatalf("after untag: %v", tags)
	}
	efsCall(t, ts, http.MethodPost, "create-tags/"+fsID, `{"Tags":[{"Key":"team","Value":"b"},{"Key":"cost","Value":"1"}]}`, http.StatusNoContent)
	legacy := efsCall(t, ts, http.MethodGet, "tags/"+fsID+"/?MaxItems=1", "", http.StatusOK)
	if first := legacy["Tags"].([]any)[0].(map[string]any); first["Value"] != "b" || legacy["NextMarker"] != "cost" {
		t.Fatalf("legacy tags page: %v", legacy)
	}
	efsCall(t, ts, http.MethodPost, "delete-tags/"+fsID, `{"TagKeys":["cost"]}`, http.StatusNoContent)
	if legacy := efsCall(t, ts, http.MethodGet, "tags/"+fsID+"/", "", http.StatusOK); len(legacy["Tags"].([]any)) != 1 {
		t.Fatalf("legacy tags after delete: %v", legacy)
	}
	efsCallError(t, ts, http.MethodGet, "resource-tags/fsap-0000000000000beef", "", http.StatusNotFound, "AccessPointNotFound")
	efsCallError(t, ts, http.MethodPost, "resource-tags/"+fsID, `{"Tags":[]}`, http.StatusBadRequest, "BadRequest")

	efsCall(t, ts, http.MethodDelete, "access-points/"+apID, "", http.StatusNoContent)
	efsCallError(t, ts, http.MethodDelete, "access-points/"+apID, "", http.StatusNotFound, "AccessPointNotFound")
}
