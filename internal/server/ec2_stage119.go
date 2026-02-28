package server

import (
	"encoding/xml"
	"net/http"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage119Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeLaunchTemplates":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		launchTemplates, nextToken, err := s.ec2.DescribeLaunchTemplates(
			parseEC2MembersOrItemList(r.Form, "LaunchTemplateId"),
			parseEC2MembersOrItemList(r.Form, "LaunchTemplateName"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeLaunchTemplatesResponse{
			XMLName:         xml.Name{Local: "DescribeLaunchTemplatesResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			LaunchTemplates: ec2Stage119LaunchTemplateSet{Items: ec2Stage119LaunchTemplateItemsFrom(launchTemplates)},
			NextToken:       nextToken,
		})
		return true
	case "DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		associations, nextToken, err := s.ec2.DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociations(
			parseEC2MembersOrItemList(r.Form, "LocalGatewayRouteTableVirtualInterfaceGroupAssociationId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociationsResponse{
			XMLName:   xml.Name{Local: "DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			LocalGatewayRouteTableVirtualInterfaceGroupAssociationSet: ec2Stage119LocalGatewayRouteTableVirtualInterfaceGroupAssociationSet{
				Items: ec2Stage119LocalGatewayRouteTableVirtualInterfaceGroupAssociationItemsFrom(associations),
			},
			NextToken: nextToken,
		})
		return true
	case "DescribeLocalGatewayRouteTableVpcAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		associations, nextToken, err := s.ec2.DescribeLocalGatewayRouteTableVpcAssociations(
			parseEC2MembersOrItemList(r.Form, "LocalGatewayRouteTableVpcAssociationId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeLocalGatewayRouteTableVpcAssociationsResponse{
			XMLName:   xml.Name{Local: "DescribeLocalGatewayRouteTableVpcAssociationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			LocalGatewayRouteTableVpcAssociationSet: ec2Stage119LocalGatewayRouteTableVpcAssociationSet{
				Items: ec2Stage119LocalGatewayRouteTableVpcAssociationItemsFrom(associations),
			},
			NextToken: nextToken,
		})
		return true
	case "DescribeLocalGatewayRouteTables":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		routeTables, nextToken, err := s.ec2.DescribeLocalGatewayRouteTables(
			parseEC2MembersOrItemList(r.Form, "LocalGatewayRouteTableId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeLocalGatewayRouteTablesResponse{
			XMLName:   xml.Name{Local: "DescribeLocalGatewayRouteTablesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			LocalGatewayRouteTableSet: ec2Stage119LocalGatewayRouteTableSet{
				Items: ec2Stage119LocalGatewayRouteTableItemsFrom(routeTables),
			},
			NextToken: nextToken,
		})
		return true
	case "DescribeLocalGatewayVirtualInterfaceGroups":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		groups, nextToken, err := s.ec2.DescribeLocalGatewayVirtualInterfaceGroups(
			parseEC2MembersOrItemList(r.Form, "LocalGatewayVirtualInterfaceGroupId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeLocalGatewayVirtualInterfaceGroupsResponse{
			XMLName:   xml.Name{Local: "DescribeLocalGatewayVirtualInterfaceGroupsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			LocalGatewayVirtualInterfaceGroupSet: ec2Stage119LocalGatewayVirtualInterfaceGroupSet{
				Items: ec2Stage119LocalGatewayVirtualInterfaceGroupItemsFrom(groups),
			},
			NextToken: nextToken,
		})
		return true
	case "DescribeLocalGatewayVirtualInterfaces":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		virtualInterfaces, nextToken, err := s.ec2.DescribeLocalGatewayVirtualInterfaces(
			parseEC2MembersOrItemList(r.Form, "LocalGatewayVirtualInterfaceId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeLocalGatewayVirtualInterfacesResponse{
			XMLName:   xml.Name{Local: "DescribeLocalGatewayVirtualInterfacesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			LocalGatewayVirtualInterfaceSet: ec2Stage119LocalGatewayVirtualInterfaceSet{
				Items: ec2Stage119LocalGatewayVirtualInterfaceItemsFrom(virtualInterfaces),
			},
			NextToken: nextToken,
		})
		return true
	case "DescribeLocalGateways":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		localGateways, nextToken, err := s.ec2.DescribeLocalGateways(
			parseEC2MembersOrItemList(r.Form, "LocalGatewayId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeLocalGatewaysResponse{
			XMLName:   xml.Name{Local: "DescribeLocalGatewaysResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			LocalGatewaySet: ec2Stage119LocalGatewaySet{
				Items: ec2Stage119LocalGatewayItemsFrom(localGateways),
			},
			NextToken: nextToken,
		})
		return true
	case "DescribeLockedSnapshots":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		snapshots, nextToken, err := s.ec2.DescribeLockedSnapshots(
			parseEC2MembersOrItemList(r.Form, "SnapshotId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeLockedSnapshotsResponse{
			XMLName:   xml.Name{Local: "DescribeLockedSnapshotsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SnapshotSet: ec2Stage119LockedSnapshotInfoSet{
				Items: ec2Stage119LockedSnapshotInfoItemsFrom(snapshots),
			},
			NextToken: nextToken,
		})
		return true
	case "DescribeMacHosts":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		hosts, nextToken, err := s.ec2.DescribeMacHosts(
			parseEC2MembersOrItemList(r.Form, "HostId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeMacHostsResponse{
			XMLName:   xml.Name{Local: "DescribeMacHostsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			MacHostSet: ec2Stage119MacHostSet{
				Items: ec2Stage119MacHostItemsFrom(hosts),
			},
			NextToken: nextToken,
		})
		return true
	case "DescribeMacModificationTasks":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		tasks, nextToken, err := s.ec2.DescribeMacModificationTasks(
			parseEC2MembersOrItemList(r.Form, "MacModificationTaskId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage119DescribeMacModificationTasksResponse{
			XMLName:   xml.Name{Local: "DescribeMacModificationTasksResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			MacModificationTaskSet: ec2Stage119MacModificationTaskSet{
				Items: ec2Stage119MacModificationTaskItemsFrom(tasks),
			},
			NextToken: nextToken,
		})
		return true
	default:
		return false
	}
}

func ec2Stage119LaunchTemplateItemsFrom(in []ec2svc.LaunchTemplate) []ec2Stage108LaunchTemplateItem {
	out := make([]ec2Stage108LaunchTemplateItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108LaunchTemplateItemFrom(item))
	}
	return out
}

func ec2Stage119LocalGatewayRouteTableVirtualInterfaceGroupAssociationItemsFrom(in []ec2svc.LocalGatewayRouteTableVirtualInterfaceGroupAssociation) []ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItem {
	out := make([]ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItemFrom(item))
	}
	return out
}

func ec2Stage119LocalGatewayRouteTableVpcAssociationItemsFrom(in []ec2svc.LocalGatewayRouteTableVpcAssociation) []ec2Stage108LocalGatewayRouteTableVpcAssociationItem {
	out := make([]ec2Stage108LocalGatewayRouteTableVpcAssociationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108LocalGatewayRouteTableVpcAssociationItemFrom(item))
	}
	return out
}

func ec2Stage119LocalGatewayRouteTableItemsFrom(in []ec2svc.LocalGatewayRouteTable) []ec2Stage108LocalGatewayRouteTableItem {
	out := make([]ec2Stage108LocalGatewayRouteTableItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108LocalGatewayRouteTableItemFrom(item))
	}
	return out
}

func ec2Stage119LocalGatewayVirtualInterfaceGroupItemsFrom(in []ec2svc.LocalGatewayVirtualInterfaceGroup) []ec2Stage109LocalGatewayVirtualInterfaceGroupItem {
	out := make([]ec2Stage109LocalGatewayVirtualInterfaceGroupItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage109LocalGatewayVirtualInterfaceGroupItemFrom(item))
	}
	return out
}

func ec2Stage119LocalGatewayVirtualInterfaceItemsFrom(in []ec2svc.LocalGatewayVirtualInterface) []ec2Stage108LocalGatewayVirtualInterfaceItem {
	out := make([]ec2Stage108LocalGatewayVirtualInterfaceItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108LocalGatewayVirtualInterfaceItemFrom(item))
	}
	return out
}

func ec2Stage119LocalGatewayItemsFrom(in []ec2svc.LocalGateway) []ec2Stage119LocalGatewayItem {
	out := make([]ec2Stage119LocalGatewayItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage119LocalGatewayItem{
			LocalGatewayID: item.LocalGatewayID,
			OutpostARN:     item.OutpostARN,
			OwnerID:        item.OwnerID,
			State:          item.State,
			TagSet:         ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage119LockedSnapshotInfoItemsFrom(in []ec2svc.LockedSnapshotInfo) []ec2Stage119LockedSnapshotInfoItem {
	out := make([]ec2Stage119LockedSnapshotInfoItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage119LockedSnapshotInfoItem{
			CoolOffPeriod:          item.CoolOffPeriod,
			CoolOffPeriodExpiresOn: ec2Stage119RFC3339Ptr(item.CoolOffPeriodExpiresOn),
			LockCreatedOn:          ec2Stage119RFC3339Ptr(item.LockCreatedOn),
			LockDuration:           item.LockDuration,
			LockDurationStartTime:  ec2Stage119RFC3339Ptr(item.LockDurationStartTime),
			LockExpiresOn:          ec2Stage119RFC3339Ptr(item.LockExpiresOn),
			LockState:              item.LockState,
			OwnerID:                item.OwnerID,
			SnapshotID:             item.SnapshotID,
		})
	}
	return out
}

func ec2Stage119MacHostItemsFrom(in []ec2svc.MacHost) []ec2Stage119MacHostItem {
	out := make([]ec2Stage119MacHostItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage119MacHostItem{
			HostID: item.HostID,
			MacOSLatestSupportedVersionSet: ec2StringSet{
				Items: append([]string(nil), item.MacOSLatestSupportedVersion...),
			},
		})
	}
	return out
}

func ec2Stage119MacModificationTaskItemsFrom(in []ec2svc.MacModificationTask) []ec2Stage107MacModificationTaskItem {
	out := make([]ec2Stage107MacModificationTaskItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage107MacModificationTaskItemFrom(item))
	}
	return out
}

func ec2Stage119RFC3339Ptr(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

type ec2Stage119DescribeLaunchTemplatesResponse struct {
	XMLName         xml.Name                     `xml:"DescribeLaunchTemplatesResponse"`
	Xmlns           string                       `xml:"xmlns,attr"`
	RequestID       string                       `xml:"requestId"`
	LaunchTemplates ec2Stage119LaunchTemplateSet `xml:"launchTemplates"`
	NextToken       *string                      `xml:"nextToken,omitempty"`
}

type ec2Stage119LaunchTemplateSet struct {
	Items []ec2Stage108LaunchTemplateItem `xml:"item"`
}

type ec2Stage119DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociationsResponse struct {
	XMLName                                                   xml.Name                                                             `xml:"DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociationsResponse"`
	Xmlns                                                     string                                                               `xml:"xmlns,attr"`
	RequestID                                                 string                                                               `xml:"requestId"`
	LocalGatewayRouteTableVirtualInterfaceGroupAssociationSet ec2Stage119LocalGatewayRouteTableVirtualInterfaceGroupAssociationSet `xml:"localGatewayRouteTableVirtualInterfaceGroupAssociationSet"`
	NextToken                                                 *string                                                              `xml:"nextToken,omitempty"`
}

type ec2Stage119LocalGatewayRouteTableVirtualInterfaceGroupAssociationSet struct {
	Items []ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItem `xml:"item"`
}

type ec2Stage119DescribeLocalGatewayRouteTableVpcAssociationsResponse struct {
	XMLName                                 xml.Name                                           `xml:"DescribeLocalGatewayRouteTableVpcAssociationsResponse"`
	Xmlns                                   string                                             `xml:"xmlns,attr"`
	RequestID                               string                                             `xml:"requestId"`
	LocalGatewayRouteTableVpcAssociationSet ec2Stage119LocalGatewayRouteTableVpcAssociationSet `xml:"localGatewayRouteTableVpcAssociationSet"`
	NextToken                               *string                                            `xml:"nextToken,omitempty"`
}

type ec2Stage119LocalGatewayRouteTableVpcAssociationSet struct {
	Items []ec2Stage108LocalGatewayRouteTableVpcAssociationItem `xml:"item"`
}

type ec2Stage119DescribeLocalGatewayRouteTablesResponse struct {
	XMLName                   xml.Name                             `xml:"DescribeLocalGatewayRouteTablesResponse"`
	Xmlns                     string                               `xml:"xmlns,attr"`
	RequestID                 string                               `xml:"requestId"`
	LocalGatewayRouteTableSet ec2Stage119LocalGatewayRouteTableSet `xml:"localGatewayRouteTableSet"`
	NextToken                 *string                              `xml:"nextToken,omitempty"`
}

type ec2Stage119LocalGatewayRouteTableSet struct {
	Items []ec2Stage108LocalGatewayRouteTableItem `xml:"item"`
}

type ec2Stage119DescribeLocalGatewayVirtualInterfaceGroupsResponse struct {
	XMLName                              xml.Name                                        `xml:"DescribeLocalGatewayVirtualInterfaceGroupsResponse"`
	Xmlns                                string                                          `xml:"xmlns,attr"`
	RequestID                            string                                          `xml:"requestId"`
	LocalGatewayVirtualInterfaceGroupSet ec2Stage119LocalGatewayVirtualInterfaceGroupSet `xml:"localGatewayVirtualInterfaceGroupSet"`
	NextToken                            *string                                         `xml:"nextToken,omitempty"`
}

type ec2Stage119LocalGatewayVirtualInterfaceGroupSet struct {
	Items []ec2Stage109LocalGatewayVirtualInterfaceGroupItem `xml:"item"`
}

type ec2Stage119DescribeLocalGatewayVirtualInterfacesResponse struct {
	XMLName                         xml.Name                                   `xml:"DescribeLocalGatewayVirtualInterfacesResponse"`
	Xmlns                           string                                     `xml:"xmlns,attr"`
	RequestID                       string                                     `xml:"requestId"`
	LocalGatewayVirtualInterfaceSet ec2Stage119LocalGatewayVirtualInterfaceSet `xml:"localGatewayVirtualInterfaceSet"`
	NextToken                       *string                                    `xml:"nextToken,omitempty"`
}

type ec2Stage119LocalGatewayVirtualInterfaceSet struct {
	Items []ec2Stage108LocalGatewayVirtualInterfaceItem `xml:"item"`
}

type ec2Stage119DescribeLocalGatewaysResponse struct {
	XMLName         xml.Name                   `xml:"DescribeLocalGatewaysResponse"`
	Xmlns           string                     `xml:"xmlns,attr"`
	RequestID       string                     `xml:"requestId"`
	LocalGatewaySet ec2Stage119LocalGatewaySet `xml:"localGatewaySet"`
	NextToken       *string                    `xml:"nextToken,omitempty"`
}

type ec2Stage119LocalGatewaySet struct {
	Items []ec2Stage119LocalGatewayItem `xml:"item"`
}

type ec2Stage119LocalGatewayItem struct {
	LocalGatewayID string    `xml:"localGatewayId,omitempty"`
	OutpostARN     string    `xml:"outpostArn,omitempty"`
	OwnerID        string    `xml:"ownerId,omitempty"`
	State          string    `xml:"state,omitempty"`
	TagSet         ec2TagSet `xml:"tagSet"`
}

type ec2Stage119DescribeLockedSnapshotsResponse struct {
	XMLName     xml.Name                         `xml:"DescribeLockedSnapshotsResponse"`
	Xmlns       string                           `xml:"xmlns,attr"`
	RequestID   string                           `xml:"requestId"`
	SnapshotSet ec2Stage119LockedSnapshotInfoSet `xml:"snapshotSet"`
	NextToken   *string                          `xml:"nextToken,omitempty"`
}

type ec2Stage119LockedSnapshotInfoSet struct {
	Items []ec2Stage119LockedSnapshotInfoItem `xml:"item"`
}

type ec2Stage119LockedSnapshotInfoItem struct {
	CoolOffPeriod          *int32 `xml:"coolOffPeriod,omitempty"`
	CoolOffPeriodExpiresOn string `xml:"coolOffPeriodExpiresOn,omitempty"`
	LockCreatedOn          string `xml:"lockCreatedOn,omitempty"`
	LockDuration           *int32 `xml:"lockDuration,omitempty"`
	LockDurationStartTime  string `xml:"lockDurationStartTime,omitempty"`
	LockExpiresOn          string `xml:"lockExpiresOn,omitempty"`
	LockState              string `xml:"lockState,omitempty"`
	OwnerID                string `xml:"ownerId,omitempty"`
	SnapshotID             string `xml:"snapshotId,omitempty"`
}

type ec2Stage119DescribeMacHostsResponse struct {
	XMLName    xml.Name              `xml:"DescribeMacHostsResponse"`
	Xmlns      string                `xml:"xmlns,attr"`
	RequestID  string                `xml:"requestId"`
	MacHostSet ec2Stage119MacHostSet `xml:"macHostSet"`
	NextToken  *string               `xml:"nextToken,omitempty"`
}

type ec2Stage119MacHostSet struct {
	Items []ec2Stage119MacHostItem `xml:"item"`
}

type ec2Stage119MacHostItem struct {
	HostID                         string       `xml:"hostId,omitempty"`
	MacOSLatestSupportedVersionSet ec2StringSet `xml:"macOSLatestSupportedVersionSet"`
}

type ec2Stage119DescribeMacModificationTasksResponse struct {
	XMLName                xml.Name                          `xml:"DescribeMacModificationTasksResponse"`
	Xmlns                  string                            `xml:"xmlns,attr"`
	RequestID              string                            `xml:"requestId"`
	MacModificationTaskSet ec2Stage119MacModificationTaskSet `xml:"macModificationTaskSet"`
	NextToken              *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage119MacModificationTaskSet struct {
	Items []ec2Stage107MacModificationTaskItem `xml:"item"`
}
