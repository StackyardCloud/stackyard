package server

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2QueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isEC2QueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ec2")
	if !ok {
		respondEC2ErrorXML(w, status, code, msg)
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondEC2ErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form data")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondEC2ErrorXML(w, http.StatusBadRequest, "MissingAction", "missing Action")
		return true
	}
	if _, known := ec2OperationByName[action]; !known {
		respondEC2ErrorXML(w, http.StatusBadRequest, "InvalidAction", "invalid Action")
		return true
	}
	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2016-11-15" {
		respondEC2ErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid Version")
		return true
	}

	switch action {
	case "RunInstances":
		res, err := s.ec2.RunInstances(
			strings.TrimSpace(r.Form.Get("ImageId")),
			strings.TrimSpace(r.Form.Get("InstanceType")),
			strings.TrimSpace(r.Form.Get("KeyName")),
			strings.TrimSpace(r.Form.Get("SubnetId")),
			firstNonEmpty(strings.TrimSpace(r.Form.Get("Placement.AvailabilityZone")), strings.TrimSpace(r.Form.Get("AvailabilityZone"))),
			parseEC2Members(r.Form, "SecurityGroupId."),
			parseEC2Int32(r.Form.Get("MinCount"), 1),
			parseEC2Int32(r.Form.Get("MaxCount"), 1),
			parseEC2TagSpecificationsForResource(r.Form, "instance"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2RunInstancesResponse{
			XMLName:       xml.Name{Local: "RunInstancesResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			ReservationID: res.Reservation.ID,
			OwnerID:       res.Reservation.OwnerID,
			GroupSet:      ec2GroupIDSet{Items: ec2GroupIDSetItems(res.Reservation.GroupIDs)},
			InstancesSet:  ec2InstanceSet{Items: ec2InstanceItems(res.Instances)},
		})
		return true
	case "DescribeInstances":
		reservations := s.ec2.DescribeInstances(parseEC2Members(r.Form, "InstanceId."))
		respondEC2XML(w, ec2DescribeInstancesResponse{
			XMLName:        xml.Name{Local: "DescribeInstancesResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			ReservationSet: ec2ReservationSet{Items: ec2ReservationItems(reservations)},
		})
		return true
	case "StartInstances":
		changes, err := s.ec2.StartInstances(parseEC2Members(r.Form, "InstanceId."))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2StateChangeResponse{
			XMLName:      xml.Name{Local: "StartInstancesResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			InstancesSet: ec2StateChangeSet{Items: ec2StateChangeItems(changes)},
		})
		return true
	case "StopInstances":
		changes, err := s.ec2.StopInstances(parseEC2Members(r.Form, "InstanceId."))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2StateChangeResponse{
			XMLName:      xml.Name{Local: "StopInstancesResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			InstancesSet: ec2StateChangeSet{Items: ec2StateChangeItems(changes)},
		})
		return true
	case "RebootInstances":
		if err := s.ec2.RebootInstances(parseEC2Members(r.Form, "InstanceId.")); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "RebootInstancesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "TerminateInstances":
		changes, err := s.ec2.TerminateInstances(parseEC2Members(r.Form, "InstanceId."))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2StateChangeResponse{
			XMLName:      xml.Name{Local: "TerminateInstancesResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			InstancesSet: ec2StateChangeSet{Items: ec2StateChangeItems(changes)},
		})
		return true
	case "DescribeInstanceStatus":
		statuses := s.ec2.DescribeInstanceStatus(
			parseEC2Members(r.Form, "InstanceId."),
			parseEC2Bool(r.Form.Get("IncludeAllInstances"), false),
		)
		respondEC2XML(w, ec2DescribeInstanceStatusResponse{
			XMLName:           xml.Name{Local: "DescribeInstanceStatusResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			InstanceStatusSet: ec2InstanceStatusSet{Items: ec2InstanceStatusItems(statuses)},
		})
		return true
	case "CreateTags":
		if err := s.ec2.CreateTags(parseEC2Members(r.Form, "ResourceId."), parseEC2Tags(r.Form, "Tag.")); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "CreateTagsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteTags":
		if err := s.ec2.DeleteTags(parseEC2Members(r.Form, "ResourceId."), parseEC2Tags(r.Form, "Tag.")); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteTagsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DescribeTags":
		tags := s.ec2.DescribeTags(parseEC2Members(r.Form, "Filter.1.Value."))
		if ids := parseEC2Members(r.Form, "ResourceId."); len(ids) > 0 {
			tags = s.ec2.DescribeTags(ids)
		}
		respondEC2XML(w, ec2DescribeTagsResponse{
			XMLName:   xml.Name{Local: "DescribeTagsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			TagSet:    ec2TagDescriptionSet{Items: ec2TagDescriptionItems(tags)},
		})
		return true
	case "DescribeRegions":
		respondEC2XML(w, ec2DescribeRegionsResponse{
			XMLName:    xml.Name{Local: "DescribeRegionsResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			RegionInfo: ec2RegionSet{Items: ec2RegionItems(s.ec2.DescribeRegions(parseEC2Members(r.Form, "RegionName.")))},
		})
		return true
	case "DescribeAvailabilityZones":
		respondEC2XML(w, ec2DescribeAvailabilityZonesResponse{
			XMLName:              xml.Name{Local: "DescribeAvailabilityZonesResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			AvailabilityZoneInfo: ec2AvailabilityZoneSet{Items: ec2AvailabilityZoneItems(s.ec2.DescribeAvailabilityZones(parseEC2Members(r.Form, "ZoneName.")))},
		})
		return true
	case "CreateSecurityGroup":
		group, err := s.ec2.CreateSecurityGroup(
			strings.TrimSpace(r.Form.Get("GroupName")),
			strings.TrimSpace(r.Form.Get("GroupDescription")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2TagSpecificationsForResource(r.Form, "security-group"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateSecurityGroupResponse{
			XMLName:   xml.Name{Local: "CreateSecurityGroupResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			GroupID:   group.ID,
		})
		return true
	case "DescribeSecurityGroups":
		groups := s.ec2.DescribeSecurityGroups(
			parseEC2Members(r.Form, "GroupId."),
			parseEC2Members(r.Form, "GroupName."),
			strings.TrimSpace(r.Form.Get("VpcId")),
		)
		respondEC2XML(w, ec2DescribeSecurityGroupsResponse{
			XMLName:           xml.Name{Local: "DescribeSecurityGroupsResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			SecurityGroupInfo: ec2SecurityGroupSet{Items: ec2SecurityGroupItems(groups)},
		})
		return true
	case "AuthorizeSecurityGroupIngress":
		if err := s.ec2.AuthorizeSecurityGroupIngress(
			strings.TrimSpace(r.Form.Get("GroupId")),
			strings.TrimSpace(r.Form.Get("GroupName")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2IPPermissions(r.Form, "IpPermissions."),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "AuthorizeSecurityGroupIngressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "RevokeSecurityGroupIngress":
		if err := s.ec2.RevokeSecurityGroupIngress(
			strings.TrimSpace(r.Form.Get("GroupId")),
			strings.TrimSpace(r.Form.Get("GroupName")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2IPPermissions(r.Form, "IpPermissions."),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "RevokeSecurityGroupIngressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteSecurityGroup":
		if err := s.ec2.DeleteSecurityGroup(strings.TrimSpace(r.Form.Get("GroupId")), strings.TrimSpace(r.Form.Get("GroupName")), strings.TrimSpace(r.Form.Get("VpcId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteSecurityGroupResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateVolume":
		volume, err := s.ec2.CreateVolume(
			parseEC2Int32(r.Form.Get("Size"), 8),
			strings.TrimSpace(r.Form.Get("AvailabilityZone")),
			strings.TrimSpace(r.Form.Get("VolumeType")),
			strings.TrimSpace(r.Form.Get("SnapshotId")),
			parseEC2TagSpecificationsForResource(r.Form, "volume"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVolumeResponse{
			XMLName:       xml.Name{Local: "CreateVolumeResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			ec2VolumeItem: ec2VolumeItemFrom(volume),
		})
		return true
	case "DescribeVolumes":
		volumes := s.ec2.DescribeVolumes(parseEC2Members(r.Form, "VolumeId."))
		respondEC2XML(w, ec2DescribeVolumesResponse{
			XMLName:   xml.Name{Local: "DescribeVolumesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VolumeSet: ec2VolumeSet{Items: ec2VolumeItems(volumes)},
		})
		return true
	case "AttachVolume":
		attach, err := s.ec2.AttachVolume(
			strings.TrimSpace(r.Form.Get("VolumeId")),
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("Device")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AttachVolumeResponse{
			XMLName:                 xml.Name{Local: "AttachVolumeResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			ec2VolumeAttachmentItem: ec2VolumeAttachmentItemFrom(attach),
		})
		return true
	case "DetachVolume":
		attach, err := s.ec2.DetachVolume(
			strings.TrimSpace(r.Form.Get("VolumeId")),
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("Device")),
			parseEC2Bool(r.Form.Get("Force"), false),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DetachVolumeResponse{
			XMLName:                 xml.Name{Local: "DetachVolumeResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			ec2VolumeAttachmentItem: ec2VolumeAttachmentItemFrom(attach),
		})
		return true
	case "DeleteVolume":
		if err := s.ec2.DeleteVolume(strings.TrimSpace(r.Form.Get("VolumeId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteVolumeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateSnapshot":
		snapshot, err := s.ec2.CreateSnapshot(
			strings.TrimSpace(r.Form.Get("VolumeId")),
			strings.TrimSpace(r.Form.Get("Description")),
			parseEC2TagSpecificationsForResource(r.Form, "snapshot"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateSnapshotResponse{
			XMLName:         xml.Name{Local: "CreateSnapshotResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			ec2SnapshotItem: ec2SnapshotItemFrom(snapshot),
		})
		return true
	case "DescribeSnapshots":
		snapshots := s.ec2.DescribeSnapshots(parseEC2Members(r.Form, "SnapshotId."))
		respondEC2XML(w, ec2DescribeSnapshotsResponse{
			XMLName:     xml.Name{Local: "DescribeSnapshotsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			SnapshotSet: ec2SnapshotSet{Items: ec2SnapshotItems(snapshots)},
		})
		return true
	case "DeleteSnapshot":
		if err := s.ec2.DeleteSnapshot(strings.TrimSpace(r.Form.Get("SnapshotId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteSnapshotResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	default:
		if s.handleEC2Stage1Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage2Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage3Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage4Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage5Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage6Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage7Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage8Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage9Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage10Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage11Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage12Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage13Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage14Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage15Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage16Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage17Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage18Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage19Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage20Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage21Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage22Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage23Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage24Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage25Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage26Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage27Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage28Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage29Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage30Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage31Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage32Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage33Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage34Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage35Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage36Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage38Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage39Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage40Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage41Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage42Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage43Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage44Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage45Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage46Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage47Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage48Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage49Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage50Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage51Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage52Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage53Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage54Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage55Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage56Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage57Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage58Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage59Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage60Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage61Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage62Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage63Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage64Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage65Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage66Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage67Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage68Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage69Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage70Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage71Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage72Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage73Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage74Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage75Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage76Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage77Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage78Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage79Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage80Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage81Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage82Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage83Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage84Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage85Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage86Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage87Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage88Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage89Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage90Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage91Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage92Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage93Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage94Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage95Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage96Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage97Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage98Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage99Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage100Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage101Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage102Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage103Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage104Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage105Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage106Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage107Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage108Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage109Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage110Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage111Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage112Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage113Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage114Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage115Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage116Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage117Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage118Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage119Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage120Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage121Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage122Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage123Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage124Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage125Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage126Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage127Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage128Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage129Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage130Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage131Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage132Action(w, r, action) {
			return true
		}
		if s.handleEC2Stage133Action(w, r, action) {
			return true
		}
		respondEC2ErrorXML(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
		return true
	}
}

func isEC2QueryCandidate(r *http.Request) bool {
	action := strings.TrimSpace(r.URL.Query().Get("Action"))
	if action != "" {
		if _, ok := ec2OperationByName[action]; !ok {
			return false
		}
		if version := strings.TrimSpace(r.URL.Query().Get("Version")); version != "" && version != "2016-11-15" {
			return false
		}
		if service := sigV4ServiceHint(r); service != "" && service != "ec2" {
			return false
		}
		return true
	}
	if r.Method != http.MethodPost {
		return false
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
		return false
	}
	body, err := readBodyBytes(r)
	if err != nil {
		return false
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return false
	}
	action = strings.TrimSpace(values.Get("Action"))
	if action == "" {
		return false
	}
	if _, ok := ec2OperationByName[action]; !ok {
		return false
	}
	if version := strings.TrimSpace(values.Get("Version")); version != "" && version != "2016-11-15" {
		return false
	}
	if service := sigV4ServiceHint(r); service != "" && service != "ec2" {
		return false
	}
	return true
}

func respondEC2XML(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	_ = xml.NewEncoder(w).Encode(body)
}

func respondEC2ErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ec2svc.ErrInvalidParameter):
		respondEC2ErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	case errors.Is(err, ec2svc.ErrAlreadyExists):
		respondEC2ErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	case errors.Is(err, ec2svc.ErrNotFound):
		respondEC2ErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	case errors.Is(err, ec2svc.ErrConflict):
		respondEC2ErrorXML(w, http.StatusBadRequest, "IncorrectState", err.Error())
	default:
		respondEC2ErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	}
}

func respondEC2ErrorXML(w http.ResponseWriter, status int, code, message string) {
	respondXML(w, status, ec2ErrorResponse{
		Xmlns: ec2Namespace,
		Errors: ec2ErrorList{
			Error: ec2ErrorBody{Code: code, Message: message},
		},
		RequestID: "stackyard-request",
	})
}

func parseEC2Members(values url.Values, prefix string) []string {
	type pair struct {
		idx int
		val string
	}
	items := make([]pair, 0)
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimPrefix(key, prefix))
		if err != nil || idx <= 0 || len(vals) == 0 {
			continue
		}
		value := strings.TrimSpace(vals[0])
		if value == "" {
			continue
		}
		items = append(items, pair{idx: idx, val: value})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].idx < items[j].idx })
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.val)
	}
	return out
}

func parseEC2Tags(values url.Values, prefix string) []ec2svc.Tag {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(key, prefix)
		part := suffix
		if dot := strings.IndexByte(suffix, '.'); dot >= 0 {
			part = suffix[:dot]
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}
	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)
	out := make([]ec2svc.Tag, 0, len(ordered))
	for _, idx := range ordered {
		key := strings.TrimSpace(values.Get(prefix + strconv.Itoa(idx) + ".Key"))
		if key == "" {
			continue
		}
		out = append(out, ec2svc.Tag{
			Key:   key,
			Value: strings.TrimSpace(values.Get(prefix + strconv.Itoa(idx) + ".Value")),
		})
	}
	return out
}

func parseEC2TagSpecificationsForResource(values url.Values, resourceType string) []ec2svc.Tag {
	type indexed struct {
		idx int
	}
	specs := make([]indexed, 0)
	for key := range values {
		if !strings.HasPrefix(key, "TagSpecification.") || !strings.HasSuffix(key, ".ResourceType") {
			continue
		}
		rest := strings.TrimPrefix(key, "TagSpecification.")
		rest = strings.TrimSuffix(rest, ".ResourceType")
		idx, err := strconv.Atoi(rest)
		if err != nil || idx <= 0 {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(values.Get(key)), resourceType) {
			continue
		}
		specs = append(specs, indexed{idx: idx})
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].idx < specs[j].idx })
	for _, spec := range specs {
		tags := parseEC2Tags(values, "TagSpecification."+strconv.Itoa(spec.idx)+".Tag.")
		if len(tags) > 0 {
			return tags
		}
	}
	return nil
}

func parseEC2IPPermissions(values url.Values, prefix string) []ec2svc.IPPermission {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		part := rest
		if dot := strings.IndexByte(rest, '.'); dot >= 0 {
			part = rest[:dot]
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}
	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)

	out := make([]ec2svc.IPPermission, 0)
	for _, idx := range ordered {
		base := prefix + strconv.Itoa(idx) + "."
		protocol := strings.TrimSpace(values.Get(base + "IpProtocol"))
		if protocol == "" {
			protocol = "tcp"
		}
		fromPort := parseEC2Int32(values.Get(base+"FromPort"), 0)
		toPort := parseEC2Int32(values.Get(base+"ToPort"), 0)
		ranges := parseEC2IPRangesWithDescriptions(values, base)
		if len(ranges) == 0 {
			out = append(out, ec2svc.IPPermission{
				Protocol: protocol,
				FromPort: fromPort,
				ToPort:   toPort,
				CidrIP:   "0.0.0.0/0",
			})
			continue
		}
		for _, ipRange := range ranges {
			out = append(out, ec2svc.IPPermission{
				Protocol:    protocol,
				FromPort:    fromPort,
				ToPort:      toPort,
				CidrIP:      ipRange.CidrIP,
				Description: ipRange.Description,
			})
		}
	}
	return out
}

type ec2ParsedIPRange struct {
	CidrIP      string
	Description string
}

func parseEC2IPRangesWithDescriptions(values url.Values, base string) []ec2ParsedIPRange {
	ranges := make([]ec2ParsedIPRange, 0)
	seen := map[string]struct{}{}

	appendRange := func(cidr, description string) {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			return
		}
		key := cidr + "|" + strings.TrimSpace(description)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		ranges = append(ranges, ec2ParsedIPRange{
			CidrIP:      cidr,
			Description: strings.TrimSpace(description),
		})
	}

	collectIndexed := func(prefix string) {
		indices := map[int]struct{}{}
		for key := range values {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			rest := strings.TrimPrefix(key, prefix)
			part := rest
			if dot := strings.IndexByte(rest, '.'); dot >= 0 {
				part = rest[:dot]
			}
			idx, err := strconv.Atoi(part)
			if err != nil || idx <= 0 {
				continue
			}
			indices[idx] = struct{}{}
		}
		ordered := make([]int, 0, len(indices))
		for idx := range indices {
			ordered = append(ordered, idx)
		}
		sort.Ints(ordered)
		for _, idx := range ordered {
			memberBase := prefix + strconv.Itoa(idx) + "."
			cidr := strings.TrimSpace(values.Get(memberBase + "CidrIp"))
			if cidr == "" {
				cidr = strings.TrimSpace(values.Get(memberBase + "CidrIpv4"))
			}
			appendRange(cidr, values.Get(memberBase+"Description"))
		}
	}

	collectIndexed(base + "IpRanges.")
	collectIndexed(base + "Ipv4Ranges.")

	legacyCIDRs := parseEC2Members(values, base+"IpRanges.")
	if len(legacyCIDRs) == 0 {
		legacyCIDRs = parseEC2Members(values, base+"Ipv4Ranges.")
	}
	for _, cidr := range legacyCIDRs {
		appendRange(cidr, "")
	}

	appendRange(values.Get(base+"CidrIp"), values.Get(base+"Description"))
	return ranges
}

func parseEC2Int32(value string, def int32) int32 {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return def
	}
	return int32(n)
}

func parseEC2Bool(value string, def bool) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return def
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func ec2GroupIDSetItems(ids []string) []ec2GroupIDItem {
	out := make([]ec2GroupIDItem, 0, len(ids))
	for _, id := range ids {
		out = append(out, ec2GroupIDItem{GroupID: id})
	}
	return out
}

func ec2InstanceItems(instances []ec2svc.Instance) []ec2InstanceItem {
	out := make([]ec2InstanceItem, 0, len(instances))
	for _, instance := range instances {
		out = append(out, ec2InstanceItemFrom(instance))
	}
	return out
}

func ec2InstanceItemFrom(instance ec2svc.Instance) ec2InstanceItem {
	groupItems := make([]ec2GroupSetItem, 0, len(instance.SecurityGroupIDs))
	for _, sgID := range instance.SecurityGroupIDs {
		groupItems = append(groupItems, ec2GroupSetItem{GroupID: sgID, GroupName: sgID})
	}
	tagItems := make([]ec2TagItem, 0, len(instance.Tags))
	for key, value := range instance.Tags {
		tagItems = append(tagItems, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tagItems, func(i, j int) bool { return tagItems[i].Key < tagItems[j].Key })
	return ec2InstanceItem{
		InstanceID:   instance.ID,
		ImageID:      instance.ImageID,
		InstanceType: instance.InstanceType,
		State: ec2InstanceStateItem{
			Code: instance.StateCode,
			Name: instance.StateName,
		},
		LaunchTime:       instance.LaunchTime.Format(time.RFC3339),
		PrivateIPAddress: instance.PrivateIP,
		IPAddress:        instance.PublicIP,
		SubnetID:         instance.SubnetID,
		VpcID:            instance.VpcID,
		KeyName:          instance.KeyName,
		Placement:        ec2PlacementItem{AvailabilityZone: instance.AvailabilityZone},
		GroupSet:         ec2GroupSet{Items: groupItems},
		TagSet:           ec2TagSet{Items: tagItems},
	}
}

func ec2ReservationItems(in []ec2svc.ReservationResult) []ec2ReservationItem {
	out := make([]ec2ReservationItem, 0, len(in))
	for _, reservation := range in {
		groupItems := make([]ec2GroupIDItem, 0, len(reservation.Reservation.GroupIDs))
		for _, sgID := range reservation.Reservation.GroupIDs {
			groupItems = append(groupItems, ec2GroupIDItem{GroupID: sgID})
		}
		out = append(out, ec2ReservationItem{
			ReservationID: reservation.Reservation.ID,
			OwnerID:       reservation.Reservation.OwnerID,
			GroupSet:      ec2GroupIDSet{Items: groupItems},
			InstancesSet:  ec2InstanceSet{Items: ec2InstanceItems(reservation.Instances)},
		})
	}
	return out
}

func ec2StateChangeItems(changes []ec2svc.InstanceStateChange) []ec2StateChangeItem {
	out := make([]ec2StateChangeItem, 0, len(changes))
	for _, change := range changes {
		out = append(out, ec2StateChangeItem{
			InstanceID: change.InstanceID,
			CurrentState: ec2InstanceStateItem{
				Code: change.CurrentCode,
				Name: change.CurrentName,
			},
			PreviousState: ec2InstanceStateItem{
				Code: change.PreviousCode,
				Name: change.PreviousName,
			},
		})
	}
	return out
}

func ec2InstanceStatusItems(in []ec2svc.InstanceStatus) []ec2InstanceStatusItem {
	out := make([]ec2InstanceStatusItem, 0, len(in))
	for _, status := range in {
		out = append(out, ec2InstanceStatusItem{
			InstanceID:       status.InstanceID,
			AvailabilityZone: status.AvailabilityZone,
			State:            ec2InstanceStateItem{Code: status.StateCode, Name: status.StateName},
			SystemStatus:     ec2StatusSummary{Status: status.SystemStatus},
			InstanceStatus:   ec2StatusSummary{Status: status.InstanceStatus},
		})
	}
	return out
}

func ec2TagDescriptionItems(in []ec2svc.ResourceTag) []ec2TagDescriptionItem {
	out := make([]ec2TagDescriptionItem, 0, len(in))
	for _, tag := range in {
		out = append(out, ec2TagDescriptionItem{
			ResourceID:   tag.ResourceID,
			ResourceType: tag.ResourceType,
			Key:          tag.Key,
			Value:        tag.Value,
		})
	}
	return out
}

func ec2RegionItems(in []ec2svc.Region) []ec2RegionItem {
	out := make([]ec2RegionItem, 0, len(in))
	for _, region := range in {
		out = append(out, ec2RegionItem{
			RegionName:     region.Name,
			RegionEndpoint: region.Endpoint,
			OptInStatus:    "opt-in-not-required",
		})
	}
	return out
}

func ec2AvailabilityZoneItems(in []ec2svc.AvailabilityZone) []ec2AvailabilityZoneItem {
	out := make([]ec2AvailabilityZoneItem, 0, len(in))
	for _, zone := range in {
		out = append(out, ec2AvailabilityZoneItem{
			ZoneName:   zone.Name,
			ZoneID:     zone.ZoneID,
			ZoneType:   "availability-zone",
			RegionName: zone.Region,
			State:      zone.State,
		})
	}
	return out
}

func ec2SecurityGroupItems(in []ec2svc.SecurityGroup) []ec2SecurityGroupItem {
	out := make([]ec2SecurityGroupItem, 0, len(in))
	for _, group := range in {
		tags := make([]ec2TagItem, 0, len(group.Tags))
		for key, value := range group.Tags {
			tags = append(tags, ec2TagItem{Key: key, Value: value})
		}
		sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
		perms := make([]ec2IPPermissionItem, 0, len(group.Ingress))
		for _, perm := range group.Ingress {
			perms = append(perms, ec2IPPermissionItem{
				IPProtocol: perm.Protocol,
				FromPort:   perm.FromPort,
				ToPort:     perm.ToPort,
				IPRanges:   ec2IPRangeSet{Items: []ec2IPRangeItem{{CidrIP: perm.CidrIP, Description: perm.Description}}},
			})
		}
		egress := make([]ec2IPPermissionItem, 0, len(group.Egress))
		for _, perm := range group.Egress {
			egress = append(egress, ec2IPPermissionItem{
				IPProtocol: perm.Protocol,
				FromPort:   perm.FromPort,
				ToPort:     perm.ToPort,
				IPRanges:   ec2IPRangeSet{Items: []ec2IPRangeItem{{CidrIP: perm.CidrIP, Description: perm.Description}}},
			})
		}
		out = append(out, ec2SecurityGroupItem{
			GroupID:             group.ID,
			GroupName:           group.Name,
			GroupDescription:    group.Description,
			VpcID:               group.VpcID,
			OwnerID:             ec2svc.DefaultAccountID,
			IPPermissions:       ec2IPPermissionSet{Items: perms},
			IPPermissionsEgress: ec2IPPermissionSet{Items: egress},
			TagSet:              ec2TagSet{Items: tags},
		})
	}
	return out
}

func ec2VolumeItems(in []ec2svc.Volume) []ec2VolumeItem {
	out := make([]ec2VolumeItem, 0, len(in))
	for _, volume := range in {
		out = append(out, ec2VolumeItemFrom(volume))
	}
	return out
}

func ec2VolumeItemFrom(volume ec2svc.Volume) ec2VolumeItem {
	attachments := make([]ec2VolumeAttachmentItem, 0, len(volume.Attachments))
	for _, attach := range volume.Attachments {
		attachments = append(attachments, ec2VolumeAttachmentItemFrom(attach))
	}
	tags := make([]ec2TagItem, 0, len(volume.Tags))
	for key, value := range volume.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	return ec2VolumeItem{
		VolumeID:         volume.ID,
		Size:             volume.SizeGiB,
		SnapshotID:       volume.SnapshotID,
		AvailabilityZone: volume.AvailabilityZone,
		Status:           volume.State,
		CreateTime:       volume.CreateTime.Format(time.RFC3339),
		VolumeType:       volume.VolumeType,
		AttachmentSet:    ec2VolumeAttachmentSet{Items: attachments},
		TagSet:           ec2TagSet{Items: tags},
	}
}

func ec2VolumeAttachmentItemFrom(attach ec2svc.VolumeAttachment) ec2VolumeAttachmentItem {
	return ec2VolumeAttachmentItem{
		VolumeID:   attach.VolumeID,
		InstanceID: attach.InstanceID,
		Device:     attach.Device,
		Status:     attach.State,
		AttachTime: attach.AttachTime.Format(time.RFC3339),
	}
}

func ec2SnapshotItems(in []ec2svc.Snapshot) []ec2SnapshotItem {
	out := make([]ec2SnapshotItem, 0, len(in))
	for _, snapshot := range in {
		out = append(out, ec2SnapshotItemFrom(snapshot))
	}
	return out
}

func ec2SnapshotItemFrom(snapshot ec2svc.Snapshot) ec2SnapshotItem {
	tags := make([]ec2TagItem, 0, len(snapshot.Tags))
	for key, value := range snapshot.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	return ec2SnapshotItem{
		SnapshotID:  snapshot.ID,
		VolumeID:    snapshot.VolumeID,
		Status:      snapshot.State,
		StartTime:   snapshot.StartTime.Format(time.RFC3339),
		Progress:    snapshot.Progress,
		Description: snapshot.Description,
		VolumeSize:  snapshot.VolumeSize,
		TagSet:      ec2TagSet{Items: tags},
	}
}

type ec2ErrorResponse struct {
	XMLName   xml.Name     `xml:"Response"`
	Xmlns     string       `xml:"xmlns,attr"`
	Errors    ec2ErrorList `xml:"Errors"`
	RequestID string       `xml:"RequestID"`
}

type ec2ErrorList struct {
	Error ec2ErrorBody `xml:"Error"`
}

type ec2ErrorBody struct {
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type ec2SimpleReturnResponse struct {
	XMLName   xml.Name `xml:"-"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return,omitempty"`
}

type ec2RunInstancesResponse struct {
	XMLName       xml.Name
	Xmlns         string         `xml:"xmlns,attr"`
	RequestID     string         `xml:"requestId"`
	ReservationID string         `xml:"reservationId"`
	OwnerID       string         `xml:"ownerId"`
	GroupSet      ec2GroupIDSet  `xml:"groupSet"`
	InstancesSet  ec2InstanceSet `xml:"instancesSet"`
}

type ec2DescribeInstancesResponse struct {
	XMLName        xml.Name
	Xmlns          string            `xml:"xmlns,attr"`
	RequestID      string            `xml:"requestId"`
	ReservationSet ec2ReservationSet `xml:"reservationSet"`
}

type ec2CreateSecurityGroupResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	GroupID   string `xml:"groupId"`
}

type ec2DescribeSecurityGroupsResponse struct {
	XMLName           xml.Name
	Xmlns             string              `xml:"xmlns,attr"`
	RequestID         string              `xml:"requestId"`
	SecurityGroupInfo ec2SecurityGroupSet `xml:"securityGroupInfo"`
}

type ec2CreateVolumeResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	ec2VolumeItem
}

type ec2DescribeVolumesResponse struct {
	XMLName   xml.Name
	Xmlns     string       `xml:"xmlns,attr"`
	RequestID string       `xml:"requestId"`
	VolumeSet ec2VolumeSet `xml:"volumeSet"`
}

type ec2AttachVolumeResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	ec2VolumeAttachmentItem
}

type ec2DetachVolumeResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	ec2VolumeAttachmentItem
}

type ec2CreateSnapshotResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	ec2SnapshotItem
}

type ec2DescribeSnapshotsResponse struct {
	XMLName     xml.Name
	Xmlns       string         `xml:"xmlns,attr"`
	RequestID   string         `xml:"requestId"`
	SnapshotSet ec2SnapshotSet `xml:"snapshotSet"`
}

type ec2StateChangeResponse struct {
	XMLName      xml.Name
	Xmlns        string            `xml:"xmlns,attr"`
	RequestID    string            `xml:"requestId"`
	InstancesSet ec2StateChangeSet `xml:"instancesSet"`
}

type ec2DescribeInstanceStatusResponse struct {
	XMLName           xml.Name
	Xmlns             string               `xml:"xmlns,attr"`
	RequestID         string               `xml:"requestId"`
	InstanceStatusSet ec2InstanceStatusSet `xml:"instanceStatusSet"`
}

type ec2DescribeTagsResponse struct {
	XMLName   xml.Name
	Xmlns     string               `xml:"xmlns,attr"`
	RequestID string               `xml:"requestId"`
	TagSet    ec2TagDescriptionSet `xml:"tagSet"`
}

type ec2DescribeRegionsResponse struct {
	XMLName    xml.Name
	Xmlns      string       `xml:"xmlns,attr"`
	RequestID  string       `xml:"requestId"`
	RegionInfo ec2RegionSet `xml:"regionInfo"`
}

type ec2DescribeAvailabilityZonesResponse struct {
	XMLName              xml.Name
	Xmlns                string                 `xml:"xmlns,attr"`
	RequestID            string                 `xml:"requestId"`
	AvailabilityZoneInfo ec2AvailabilityZoneSet `xml:"availabilityZoneInfo"`
}

type ec2ReservationSet struct {
	Items []ec2ReservationItem `xml:"item"`
}

type ec2ReservationItem struct {
	ReservationID string         `xml:"reservationId"`
	OwnerID       string         `xml:"ownerId"`
	GroupSet      ec2GroupIDSet  `xml:"groupSet"`
	InstancesSet  ec2InstanceSet `xml:"instancesSet"`
}

type ec2GroupIDSet struct {
	Items []ec2GroupIDItem `xml:"item"`
}

type ec2GroupIDItem struct {
	GroupID string `xml:"groupId"`
}

type ec2InstanceSet struct {
	Items []ec2InstanceItem `xml:"item"`
}

type ec2InstanceItem struct {
	InstanceID       string               `xml:"instanceId"`
	ImageID          string               `xml:"imageId"`
	InstanceType     string               `xml:"instanceType"`
	State            ec2InstanceStateItem `xml:"instanceState"`
	LaunchTime       string               `xml:"launchTime"`
	PrivateIPAddress string               `xml:"privateIpAddress,omitempty"`
	IPAddress        string               `xml:"ipAddress,omitempty"`
	SubnetID         string               `xml:"subnetId,omitempty"`
	VpcID            string               `xml:"vpcId,omitempty"`
	KeyName          string               `xml:"keyName,omitempty"`
	Placement        ec2PlacementItem     `xml:"placement"`
	GroupSet         ec2GroupSet          `xml:"groupSet"`
	TagSet           ec2TagSet            `xml:"tagSet"`
}

type ec2PlacementItem struct {
	AvailabilityZone string `xml:"availabilityZone,omitempty"`
}

type ec2GroupSet struct {
	Items []ec2GroupSetItem `xml:"item"`
}

type ec2GroupSetItem struct {
	GroupID   string `xml:"groupId,omitempty"`
	GroupName string `xml:"groupName,omitempty"`
}

type ec2TagSet struct {
	Items []ec2TagItem `xml:"item"`
}

type ec2TagItem struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type ec2InstanceStateItem struct {
	Code int32  `xml:"code"`
	Name string `xml:"name"`
}

type ec2StateChangeSet struct {
	Items []ec2StateChangeItem `xml:"item"`
}

type ec2StateChangeItem struct {
	InstanceID    string               `xml:"instanceId"`
	CurrentState  ec2InstanceStateItem `xml:"currentState"`
	PreviousState ec2InstanceStateItem `xml:"previousState"`
}

type ec2InstanceStatusSet struct {
	Items []ec2InstanceStatusItem `xml:"item"`
}

type ec2StatusSummary struct {
	Status string `xml:"status"`
}

type ec2InstanceStatusItem struct {
	InstanceID       string               `xml:"instanceId"`
	AvailabilityZone string               `xml:"availabilityZone,omitempty"`
	State            ec2InstanceStateItem `xml:"instanceState"`
	SystemStatus     ec2StatusSummary     `xml:"systemStatus"`
	InstanceStatus   ec2StatusSummary     `xml:"instanceStatus"`
}

type ec2TagDescriptionSet struct {
	Items []ec2TagDescriptionItem `xml:"item"`
}

type ec2TagDescriptionItem struct {
	ResourceID   string `xml:"resourceId"`
	ResourceType string `xml:"resourceType"`
	Key          string `xml:"key"`
	Value        string `xml:"value"`
}

type ec2RegionSet struct {
	Items []ec2RegionItem `xml:"item"`
}

type ec2RegionItem struct {
	RegionName     string `xml:"regionName"`
	RegionEndpoint string `xml:"regionEndpoint"`
	OptInStatus    string `xml:"optInStatus,omitempty"`
}

type ec2AvailabilityZoneSet struct {
	Items []ec2AvailabilityZoneItem `xml:"item"`
}

type ec2AvailabilityZoneItem struct {
	ZoneName   string `xml:"zoneName"`
	ZoneID     string `xml:"zoneId,omitempty"`
	ZoneType   string `xml:"zoneType,omitempty"`
	RegionName string `xml:"regionName"`
	State      string `xml:"zoneState,omitempty"`
}

type ec2SecurityGroupSet struct {
	Items []ec2SecurityGroupItem `xml:"item"`
}

type ec2SecurityGroupItem struct {
	GroupID             string             `xml:"groupId"`
	GroupName           string             `xml:"groupName"`
	GroupDescription    string             `xml:"groupDescription"`
	OwnerID             string             `xml:"ownerId"`
	VpcID               string             `xml:"vpcId,omitempty"`
	IPPermissions       ec2IPPermissionSet `xml:"ipPermissions"`
	IPPermissionsEgress ec2IPPermissionSet `xml:"ipPermissionsEgress"`
	TagSet              ec2TagSet          `xml:"tagSet"`
}

type ec2IPPermissionSet struct {
	Items []ec2IPPermissionItem `xml:"item"`
}

type ec2IPPermissionItem struct {
	IPProtocol string        `xml:"ipProtocol"`
	FromPort   int32         `xml:"fromPort"`
	ToPort     int32         `xml:"toPort"`
	IPRanges   ec2IPRangeSet `xml:"ipRanges"`
}

type ec2IPRangeSet struct {
	Items []ec2IPRangeItem `xml:"item"`
}

type ec2IPRangeItem struct {
	CidrIP      string `xml:"cidrIp"`
	Description string `xml:"description,omitempty"`
}

type ec2VolumeSet struct {
	Items []ec2VolumeItem `xml:"item"`
}

type ec2VolumeItem struct {
	VolumeID         string                 `xml:"volumeId"`
	Size             int32                  `xml:"size"`
	SnapshotID       string                 `xml:"snapshotId,omitempty"`
	AvailabilityZone string                 `xml:"availabilityZone"`
	Status           string                 `xml:"status"`
	CreateTime       string                 `xml:"createTime"`
	VolumeType       string                 `xml:"volumeType"`
	AttachmentSet    ec2VolumeAttachmentSet `xml:"attachmentSet"`
	TagSet           ec2TagSet              `xml:"tagSet"`
}

type ec2VolumeAttachmentSet struct {
	Items []ec2VolumeAttachmentItem `xml:"item"`
}

type ec2VolumeAttachmentItem struct {
	VolumeID   string `xml:"volumeId"`
	InstanceID string `xml:"instanceId"`
	Device     string `xml:"device"`
	Status     string `xml:"status"`
	AttachTime string `xml:"attachTime"`
}

type ec2SnapshotSet struct {
	Items []ec2SnapshotItem `xml:"item"`
}

type ec2SnapshotItem struct {
	SnapshotID  string    `xml:"snapshotId"`
	VolumeID    string    `xml:"volumeId"`
	Status      string    `xml:"status"`
	StartTime   string    `xml:"startTime"`
	Progress    string    `xml:"progress"`
	Description string    `xml:"description,omitempty"`
	VolumeSize  int32     `xml:"volumeSize"`
	TagSet      ec2TagSet `xml:"tagSet"`
}
