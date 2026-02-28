package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage132Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "RejectCapacityReservationBillingOwnership":
		ret, err := s.ec2.RejectCapacityReservationBillingOwnership(strings.TrimSpace(r.Form.Get("CapacityReservationId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "RejectCapacityReservationBillingOwnershipResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "ReleaseHosts":
		successful, unsuccessful, err := s.ec2.ReleaseHosts(parseEC2MembersOrItemList(r.Form, "HostId"))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage132ReleaseHostsResponse{
			XMLName:      xml.Name{Local: "ReleaseHostsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Successful:   ec2Stage132HostIDSet{Items: append([]string(nil), successful...)},
			Unsuccessful: ec2UnsuccessfulItemSet{Items: ec2UnsuccessfulItems(unsuccessful)},
		})
		return true
	case "ReleaseIpamPoolAllocation":
		success, err := s.ec2.ReleaseIpamPoolAllocation(
			strings.TrimSpace(r.Form.Get("IpamPoolId")),
			strings.TrimSpace(r.Form.Get("Cidr")),
			strings.TrimSpace(r.Form.Get("IpamPoolAllocationId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage132ReleaseIpamPoolAllocationResponse{
			XMLName:   xml.Name{Local: "ReleaseIpamPoolAllocationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Success:   success,
		})
		return true
	case "ReportInstanceStatus":
		startTime, ok := parseEC2Stage122OptionalRFC3339Time(r.Form, "StartTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		endTime, ok := parseEC2Stage122OptionalRFC3339Time(r.Form, "EndTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		err := s.ec2.ReportInstanceStatus(
			parseEC2MembersOrItemList(r.Form, "InstanceId"),
			strings.TrimSpace(r.Form.Get("Status")),
			parseEC2MembersOrItemList(r.Form, "ReasonCode"),
			startTime,
			endTime,
			parseEC2OptionalString(r.Form.Get("Description")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage132ReportInstanceStatusResponse{
			XMLName:   xml.Name{Local: "ReportInstanceStatusResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "RequestSpotFleet":
		targetCapacity, ok := parseEC2OptionalInt32(r.Form.Get("SpotFleetRequestConfig.TargetCapacity"))
		if !ok || targetCapacity == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		spotFleetRequestID, err := s.ec2.RequestSpotFleet(
			strings.TrimSpace(r.Form.Get("SpotFleetRequestConfig.IamFleetRole")),
			*targetCapacity,
			parseEC2OptionalString(r.Form.Get("SpotFleetRequestConfig.ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage132RequestSpotFleetResponse{
			XMLName:            xml.Name{Local: "RequestSpotFleetResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			SpotFleetRequestID: spotFleetRequestID,
		})
		return true
	case "RequestSpotInstances":
		instanceCount, ok := parseEC2OptionalInt32(r.Form.Get("InstanceCount"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		requests, err := s.ec2.RequestSpotInstances(
			instanceCount,
			parseEC2OptionalString(r.Form.Get("SpotPrice")),
			strings.TrimSpace(r.Form.Get("Type")),
			parseEC2OptionalString(r.Form.Get("LaunchGroup")),
			parseEC2OptionalString(r.Form.Get("AvailabilityZoneGroup")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage132RequestSpotInstancesResponse{
			XMLName:                xml.Name{Local: "RequestSpotInstancesResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			SpotInstanceRequestSet: ec2Stage122SpotInstanceRequestSet{Items: ec2Stage122SpotInstanceRequestItemsFrom(requests)},
		})
		return true
	case "ResetFpgaImageAttribute":
		ret, err := s.ec2.ResetFpgaImageAttribute(
			strings.TrimSpace(r.Form.Get("FpgaImageId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ResetFpgaImageAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "ResetSnapshotAttribute":
		err := s.ec2.ResetSnapshotAttribute(
			strings.TrimSpace(r.Form.Get("SnapshotId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage132ResetSnapshotAttributeResponse{
			XMLName:   xml.Name{Local: "ResetSnapshotAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "RestoreImageFromRecycleBin":
		ret, err := s.ec2.RestoreImageFromRecycleBin(strings.TrimSpace(r.Form.Get("ImageId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "RestoreImageFromRecycleBinResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "RestoreManagedPrefixListVersion":
		previousVersion, ok := parseEC2OptionalInt64(r.Form.Get("PreviousVersion"))
		if !ok || previousVersion == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		currentVersion, ok := parseEC2OptionalInt64(r.Form.Get("CurrentVersion"))
		if !ok || currentVersion == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		prefixList, err := s.ec2.RestoreManagedPrefixListVersion(
			strings.TrimSpace(r.Form.Get("PrefixListId")),
			*previousVersion,
			*currentVersion,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage132RestoreManagedPrefixListVersionResponse{
			XMLName:    xml.Name{Local: "RestoreManagedPrefixListVersionResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			PrefixList: ec2Stage109ManagedPrefixListItemFrom(prefixList),
		})
		return true
	default:
		return false
	}
}

type ec2Stage132ReleaseHostsResponse struct {
	XMLName      xml.Name               `xml:"ReleaseHostsResponse"`
	Xmlns        string                 `xml:"xmlns,attr"`
	RequestID    string                 `xml:"requestId"`
	Successful   ec2Stage132HostIDSet   `xml:"successful"`
	Unsuccessful ec2UnsuccessfulItemSet `xml:"unsuccessful"`
}

type ec2Stage132HostIDSet struct {
	Items []string `xml:"item"`
}

type ec2Stage132ReleaseIpamPoolAllocationResponse struct {
	XMLName   xml.Name `xml:"ReleaseIpamPoolAllocationResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Success   bool     `xml:"success,omitempty"`
}

type ec2Stage132ReportInstanceStatusResponse struct {
	XMLName   xml.Name `xml:"ReportInstanceStatusResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
}

type ec2Stage132RequestSpotFleetResponse struct {
	XMLName            xml.Name `xml:"RequestSpotFleetResponse"`
	Xmlns              string   `xml:"xmlns,attr"`
	RequestID          string   `xml:"requestId"`
	SpotFleetRequestID string   `xml:"spotFleetRequestId,omitempty"`
}

type ec2Stage132RequestSpotInstancesResponse struct {
	XMLName                xml.Name                          `xml:"RequestSpotInstancesResponse"`
	Xmlns                  string                            `xml:"xmlns,attr"`
	RequestID              string                            `xml:"requestId"`
	SpotInstanceRequestSet ec2Stage122SpotInstanceRequestSet `xml:"spotInstanceRequestSet"`
}

type ec2Stage132ResetSnapshotAttributeResponse struct {
	XMLName   xml.Name `xml:"ResetSnapshotAttributeResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
}

type ec2Stage132RestoreManagedPrefixListVersionResponse struct {
	XMLName    xml.Name                         `xml:"RestoreManagedPrefixListVersionResponse"`
	Xmlns      string                           `xml:"xmlns,attr"`
	RequestID  string                           `xml:"requestId"`
	PrefixList ec2Stage109ManagedPrefixListItem `xml:"prefixList"`
}
