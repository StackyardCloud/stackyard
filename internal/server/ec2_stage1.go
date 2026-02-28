package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage1Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateVpc":
		vpc, err := s.ec2.CreateVpc(
			strings.TrimSpace(r.Form.Get("CidrBlock")),
			parseEC2TagSpecificationsForResource(r.Form, "vpc"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVpcResponse{
			XMLName:   xml.Name{Local: "CreateVpcResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Vpc:       ec2VPCItemFrom(vpc),
		})
		return true
	case "DescribeVpcs":
		vpcs := s.ec2.DescribeVpcs(parseEC2Members(r.Form, "VpcId."))
		if len(vpcs) == 0 {
			if ids := parseEC2FilterValues(r.Form, "vpc-id"); len(ids) > 0 {
				vpcs = s.ec2.DescribeVpcs(ids)
			}
		}
		respondEC2XML(w, ec2DescribeVpcsResponse{
			XMLName:   xml.Name{Local: "DescribeVpcsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcSet:    ec2VPCSet{Items: ec2VPCItems(vpcs)},
		})
		return true
	case "DeleteVpc":
		if err := s.ec2.DeleteVpc(strings.TrimSpace(r.Form.Get("VpcId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteVpcResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateSubnet":
		subnet, err := s.ec2.CreateSubnet(
			strings.TrimSpace(r.Form.Get("VpcId")),
			strings.TrimSpace(r.Form.Get("CidrBlock")),
			firstNonEmpty(strings.TrimSpace(r.Form.Get("AvailabilityZone")), strings.TrimSpace(r.Form.Get("AvailabilityZoneId"))),
			parseEC2TagSpecificationsForResource(r.Form, "subnet"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateSubnetResponse{
			XMLName:   xml.Name{Local: "CreateSubnetResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Subnet:    ec2SubnetItemFrom(subnet),
		})
		return true
	case "DescribeSubnets":
		subnetIDs := parseEC2Members(r.Form, "SubnetId.")
		vpcIDs := parseEC2FilterValues(r.Form, "vpc-id")
		if len(vpcIDs) == 0 {
			vpcIDs = parseEC2Members(r.Form, "VpcId.")
		}
		subnets := s.ec2.DescribeSubnets(subnetIDs, vpcIDs)
		respondEC2XML(w, ec2DescribeSubnetsResponse{
			XMLName:   xml.Name{Local: "DescribeSubnetsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SubnetSet: ec2SubnetSet{Items: ec2SubnetItems(subnets)},
		})
		return true
	case "DeleteSubnet":
		if err := s.ec2.DeleteSubnet(strings.TrimSpace(r.Form.Get("SubnetId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteSubnetResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateInternetGateway":
		gateway, err := s.ec2.CreateInternetGateway(parseEC2TagSpecificationsForResource(r.Form, "internet-gateway"))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateInternetGatewayResponse{
			XMLName:         xml.Name{Local: "CreateInternetGatewayResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			InternetGateway: ec2InternetGatewayItemFrom(gateway),
		})
		return true
	case "DescribeInternetGateways":
		gatewayIDs := parseEC2Members(r.Form, "InternetGatewayId.")
		vpcIDs := parseEC2FilterValues(r.Form, "attachment.vpc-id")
		gateways := s.ec2.DescribeInternetGateways(gatewayIDs, vpcIDs)
		respondEC2XML(w, ec2DescribeInternetGatewaysResponse{
			XMLName:            xml.Name{Local: "DescribeInternetGatewaysResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			InternetGatewaySet: ec2InternetGatewaySet{Items: ec2InternetGatewayItems(gateways)},
		})
		return true
	case "AttachInternetGateway":
		if err := s.ec2.AttachInternetGateway(strings.TrimSpace(r.Form.Get("InternetGatewayId")), strings.TrimSpace(r.Form.Get("VpcId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "AttachInternetGatewayResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DetachInternetGateway":
		if err := s.ec2.DetachInternetGateway(strings.TrimSpace(r.Form.Get("InternetGatewayId")), strings.TrimSpace(r.Form.Get("VpcId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DetachInternetGatewayResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteInternetGateway":
		if err := s.ec2.DeleteInternetGateway(strings.TrimSpace(r.Form.Get("InternetGatewayId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteInternetGatewayResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateRouteTable":
		table, err := s.ec2.CreateRouteTable(strings.TrimSpace(r.Form.Get("VpcId")), parseEC2TagSpecificationsForResource(r.Form, "route-table"))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateRouteTableResponse{
			XMLName:    xml.Name{Local: "CreateRouteTableResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			RouteTable: ec2RouteTableItemFrom(table),
		})
		return true
	case "DescribeRouteTables":
		tableIDs := parseEC2Members(r.Form, "RouteTableId.")
		vpcIDs := parseEC2FilterValues(r.Form, "vpc-id")
		subnetIDs := parseEC2FilterValues(r.Form, "association.subnet-id")
		tables := s.ec2.DescribeRouteTables(tableIDs, vpcIDs, subnetIDs)
		respondEC2XML(w, ec2DescribeRouteTablesResponse{
			XMLName:       xml.Name{Local: "DescribeRouteTablesResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			RouteTableSet: ec2RouteTableSet{Items: ec2RouteTableItems(tables)},
		})
		return true
	case "AssociateRouteTable":
		association, err := s.ec2.AssociateRouteTable(strings.TrimSpace(r.Form.Get("RouteTableId")), strings.TrimSpace(r.Form.Get("SubnetId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateRouteTableResponse{
			XMLName:       xml.Name{Local: "AssociateRouteTableResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			AssociationID: association.ID,
		})
		return true
	case "DisassociateRouteTable":
		if err := s.ec2.DisassociateRouteTable(strings.TrimSpace(r.Form.Get("AssociationId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DisassociateRouteTableResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateRoute":
		if err := s.ec2.CreateRoute(
			strings.TrimSpace(r.Form.Get("RouteTableId")),
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
			strings.TrimSpace(r.Form.Get("GatewayId")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "CreateRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteRoute":
		if err := s.ec2.DeleteRoute(strings.TrimSpace(r.Form.Get("RouteTableId")), strings.TrimSpace(r.Form.Get("DestinationCidrBlock"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteRouteTable":
		if err := s.ec2.DeleteRouteTable(strings.TrimSpace(r.Form.Get("RouteTableId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteRouteTableResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateNetworkAcl":
		acl, err := s.ec2.CreateNetworkACL(strings.TrimSpace(r.Form.Get("VpcId")), parseEC2TagSpecificationsForResource(r.Form, "network-acl"))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateNetworkACLResponse{
			XMLName:    xml.Name{Local: "CreateNetworkAclResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			NetworkACL: ec2NetworkACLItemFrom(acl),
		})
		return true
	case "DescribeNetworkAcls":
		aclIDs := parseEC2Members(r.Form, "NetworkAclId.")
		vpcIDs := parseEC2FilterValues(r.Form, "vpc-id")
		acls := s.ec2.DescribeNetworkACLs(aclIDs, vpcIDs)
		respondEC2XML(w, ec2DescribeNetworkACLsResponse{
			XMLName:       xml.Name{Local: "DescribeNetworkAclsResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			NetworkACLSet: ec2NetworkACLSet{Items: ec2NetworkACLItems(acls)},
		})
		return true
	case "DeleteNetworkAcl":
		if err := s.ec2.DeleteNetworkACL(strings.TrimSpace(r.Form.Get("NetworkAclId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteNetworkAclResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateNetworkInterface":
		groupIDs := parseEC2Members(r.Form, "SecurityGroupId.")
		if len(groupIDs) == 0 {
			groupIDs = parseEC2Members(r.Form, "GroupId.")
		}
		iface, err := s.ec2.CreateNetworkInterface(
			strings.TrimSpace(r.Form.Get("SubnetId")),
			strings.TrimSpace(r.Form.Get("Description")),
			firstNonEmpty(strings.TrimSpace(r.Form.Get("PrivateIpAddress")), strings.TrimSpace(r.Form.Get("PrivateIpAddresses.1.PrivateIpAddress"))),
			groupIDs,
			parseEC2TagSpecificationsForResource(r.Form, "network-interface"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateNetworkInterfaceResponse{
			XMLName:          xml.Name{Local: "CreateNetworkInterfaceResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			NetworkInterface: ec2NetworkInterfaceItemFrom(iface),
		})
		return true
	case "DescribeNetworkInterfaces":
		ids := parseEC2Members(r.Form, "NetworkInterfaceId.")
		subnetID := firstNonEmpty(strings.TrimSpace(r.Form.Get("SubnetId")), firstFilterValue(r.Form, "subnet-id"))
		vpcID := firstFilterValue(r.Form, "vpc-id")
		ifaces := s.ec2.DescribeNetworkInterfaces(ids, subnetID, vpcID)
		respondEC2XML(w, ec2DescribeNetworkInterfacesResponse{
			XMLName:             xml.Name{Local: "DescribeNetworkInterfacesResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			NetworkInterfaceSet: ec2NetworkInterfaceSet{Items: ec2NetworkInterfaceItems(ifaces)},
		})
		return true
	case "AttachNetworkInterface":
		attachment, err := s.ec2.AttachNetworkInterface(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			strings.TrimSpace(r.Form.Get("InstanceId")),
			parseEC2Int32(r.Form.Get("DeviceIndex"), 0),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AttachNetworkInterfaceResponse{
			XMLName:      xml.Name{Local: "AttachNetworkInterfaceResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			AttachmentID: attachment.ID,
		})
		return true
	case "DetachNetworkInterface":
		if err := s.ec2.DetachNetworkInterface(strings.TrimSpace(r.Form.Get("AttachmentId")), parseEC2Bool(r.Form.Get("Force"), false)); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DetachNetworkInterfaceResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteNetworkInterface":
		if err := s.ec2.DeleteNetworkInterface(strings.TrimSpace(r.Form.Get("NetworkInterfaceId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteNetworkInterfaceResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	default:
		return false
	}
}

func parseEC2FilterValues(values url.Values, filterName string) []string {
	items := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, "Filter.") || !strings.HasSuffix(key, ".Name") {
			continue
		}
		indexText := strings.TrimPrefix(key, "Filter.")
		indexText = strings.TrimSuffix(indexText, ".Name")
		index, err := strconv.Atoi(indexText)
		if err != nil || index <= 0 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(values.Get(key)), filterName) {
			items[index] = struct{}{}
		}
	}
	ordered := make([]int, 0, len(items))
	for index := range items {
		ordered = append(ordered, index)
	}
	sort.Ints(ordered)
	out := make([]string, 0)
	for _, index := range ordered {
		out = append(out, parseEC2Members(values, "Filter."+strconv.Itoa(index)+".Value.")...)
	}
	return out
}

func firstFilterValue(values url.Values, filterName string) string {
	filterValues := parseEC2FilterValues(values, filterName)
	if len(filterValues) == 0 {
		return ""
	}
	return strings.TrimSpace(filterValues[0])
}

func ec2VPCItems(in []ec2svc.VPC) []ec2VPCItem {
	out := make([]ec2VPCItem, 0, len(in))
	for _, vpc := range in {
		out = append(out, ec2VPCItemFrom(vpc))
	}
	return out
}

func ec2VPCItemFrom(vpc ec2svc.VPC) ec2VPCItem {
	tags := make([]ec2TagItem, 0, len(vpc.Tags))
	for key, value := range vpc.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	return ec2VPCItem{
		VpcID:           vpc.ID,
		State:           vpc.State,
		CidrBlock:       vpc.CidrBlock,
		InstanceTenancy: vpc.InstanceTenancy,
		IsDefault:       vpc.IsDefault,
		DhcpOptionsID:   vpc.DhcpOptionsID,
		TagSet:          ec2TagSet{Items: tags},
	}
}

func ec2SubnetItems(in []ec2svc.Subnet) []ec2SubnetItem {
	out := make([]ec2SubnetItem, 0, len(in))
	for _, subnet := range in {
		out = append(out, ec2SubnetItemFrom(subnet))
	}
	return out
}

func ec2SubnetItemFrom(subnet ec2svc.Subnet) ec2SubnetItem {
	tags := make([]ec2TagItem, 0, len(subnet.Tags))
	for key, value := range subnet.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	return ec2SubnetItem{
		SubnetID:                subnet.ID,
		VpcID:                   subnet.VpcID,
		State:                   subnet.State,
		CidrBlock:               subnet.CidrBlock,
		AvailabilityZone:        subnet.AvailabilityZone,
		AvailableIPAddressCount: subnet.AvailableIPAddressCount,
		MapPublicIPOnLaunch:     subnet.MapPublicIPOnLaunch,
		TagSet:                  ec2TagSet{Items: tags},
	}
}

func ec2InternetGatewayItems(in []ec2svc.InternetGateway) []ec2InternetGatewayItem {
	out := make([]ec2InternetGatewayItem, 0, len(in))
	for _, gateway := range in {
		out = append(out, ec2InternetGatewayItemFrom(gateway))
	}
	return out
}

func ec2InternetGatewayItemFrom(gateway ec2svc.InternetGateway) ec2InternetGatewayItem {
	attachments := make([]ec2InternetGatewayAttachmentItem, 0, len(gateway.Attachments))
	for _, attachment := range gateway.Attachments {
		attachments = append(attachments, ec2InternetGatewayAttachmentItem{VpcID: attachment.VpcID, State: attachment.State})
	}
	tags := make([]ec2TagItem, 0, len(gateway.Tags))
	for key, value := range gateway.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	return ec2InternetGatewayItem{
		InternetGatewayID: gateway.ID,
		AttachmentSet:     ec2InternetGatewayAttachmentSet{Items: attachments},
		TagSet:            ec2TagSet{Items: tags},
	}
}

func ec2RouteTableItems(in []ec2svc.RouteTable) []ec2RouteTableItem {
	out := make([]ec2RouteTableItem, 0, len(in))
	for _, table := range in {
		out = append(out, ec2RouteTableItemFrom(table))
	}
	return out
}

func ec2RouteTableItemFrom(table ec2svc.RouteTable) ec2RouteTableItem {
	routes := make([]ec2RouteItem, 0, len(table.Routes))
	for _, route := range table.Routes {
		routes = append(routes, ec2RouteItem{
			DestinationCidrBlock: route.DestinationCIDR,
			GatewayID:            route.GatewayID,
			State:                route.State,
			Origin:               route.Origin,
		})
	}
	associations := make([]ec2RouteTableAssociationItem, 0, len(table.Associations))
	for _, association := range table.Associations {
		associations = append(associations, ec2RouteTableAssociationItem{
			RouteTableAssociationID: association.ID,
			RouteTableID:            table.ID,
			SubnetID:                association.SubnetID,
			Main:                    association.Main,
		})
	}
	tags := make([]ec2TagItem, 0, len(table.Tags))
	for key, value := range table.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	return ec2RouteTableItem{
		RouteTableID:   table.ID,
		VpcID:          table.VpcID,
		RouteSet:       ec2RouteSet{Items: routes},
		AssociationSet: ec2RouteTableAssociationSet{Items: associations},
		TagSet:         ec2TagSet{Items: tags},
	}
}

func ec2NetworkACLItems(in []ec2svc.NetworkACL) []ec2NetworkACLItem {
	out := make([]ec2NetworkACLItem, 0, len(in))
	for _, acl := range in {
		out = append(out, ec2NetworkACLItemFrom(acl))
	}
	return out
}

func ec2NetworkACLItemFrom(acl ec2svc.NetworkACL) ec2NetworkACLItem {
	entries := make([]ec2NetworkACLEntryItem, 0, len(acl.Entries))
	for _, entry := range acl.Entries {
		entries = append(entries, ec2NetworkACLEntryItem{
			RuleNumber: entry.RuleNumber,
			Protocol:   entry.Protocol,
			RuleAction: entry.RuleAction,
			Egress:     entry.Egress,
			CidrBlock:  entry.CidrBlock,
		})
	}
	associations := make([]ec2NetworkACLAssociationItem, 0, len(acl.Associations))
	for _, association := range acl.Associations {
		associations = append(associations, ec2NetworkACLAssociationItem{
			NetworkACLAssociationID: association.ID,
			NetworkACLID:            acl.ID,
			SubnetID:                association.SubnetID,
		})
	}
	tags := make([]ec2TagItem, 0, len(acl.Tags))
	for key, value := range acl.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	return ec2NetworkACLItem{
		NetworkACLID:   acl.ID,
		VpcID:          acl.VpcID,
		IsDefault:      acl.IsDefault,
		EntrySet:       ec2NetworkACLEntrySet{Items: entries},
		AssociationSet: ec2NetworkACLAssociationSet{Items: associations},
		TagSet:         ec2TagSet{Items: tags},
	}
}

func ec2NetworkInterfaceItems(in []ec2svc.NetworkInterface) []ec2NetworkInterfaceItem {
	out := make([]ec2NetworkInterfaceItem, 0, len(in))
	for _, iface := range in {
		out = append(out, ec2NetworkInterfaceItemFrom(iface))
	}
	return out
}

func ec2NetworkInterfaceItemFrom(iface ec2svc.NetworkInterface) ec2NetworkInterfaceItem {
	groups := make([]ec2GroupSetItem, 0, len(iface.GroupIDs))
	for _, groupID := range iface.GroupIDs {
		groups = append(groups, ec2GroupSetItem{GroupID: groupID, GroupName: groupID})
	}
	tags := make([]ec2TagItem, 0, len(iface.Tags))
	for key, value := range iface.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	var attachment *ec2NetworkInterfaceAttachmentItem
	if iface.Attachment != nil {
		attachment = &ec2NetworkInterfaceAttachmentItem{
			AttachmentID: iface.Attachment.ID,
			InstanceID:   iface.Attachment.InstanceID,
			DeviceIndex:  iface.Attachment.DeviceIndex,
			Status:       iface.Attachment.Status,
			AttachTime:   iface.Attachment.AttachTime.Format(time.RFC3339),
		}
	}
	return ec2NetworkInterfaceItem{
		NetworkInterfaceID: iface.ID,
		SubnetID:           iface.SubnetID,
		VpcID:              iface.VpcID,
		Description:        iface.Description,
		PrivateIPAddress:   iface.PrivateIP,
		Status:             iface.Status,
		SourceDestCheck:    iface.SourceDestCheck,
		GroupSet:           ec2GroupSet{Items: groups},
		Attachment:         attachment,
		TagSet:             ec2TagSet{Items: tags},
	}
}

type ec2CreateVpcResponse struct {
	XMLName   xml.Name
	Xmlns     string     `xml:"xmlns,attr"`
	RequestID string     `xml:"requestId"`
	Vpc       ec2VPCItem `xml:"vpc"`
}

type ec2DescribeVpcsResponse struct {
	XMLName   xml.Name
	Xmlns     string    `xml:"xmlns,attr"`
	RequestID string    `xml:"requestId"`
	VpcSet    ec2VPCSet `xml:"vpcSet"`
}

type ec2CreateSubnetResponse struct {
	XMLName   xml.Name
	Xmlns     string        `xml:"xmlns,attr"`
	RequestID string        `xml:"requestId"`
	Subnet    ec2SubnetItem `xml:"subnet"`
}

type ec2DescribeSubnetsResponse struct {
	XMLName   xml.Name
	Xmlns     string       `xml:"xmlns,attr"`
	RequestID string       `xml:"requestId"`
	SubnetSet ec2SubnetSet `xml:"subnetSet"`
}

type ec2CreateInternetGatewayResponse struct {
	XMLName         xml.Name
	Xmlns           string                 `xml:"xmlns,attr"`
	RequestID       string                 `xml:"requestId"`
	InternetGateway ec2InternetGatewayItem `xml:"internetGateway"`
}

type ec2DescribeInternetGatewaysResponse struct {
	XMLName            xml.Name
	Xmlns              string                `xml:"xmlns,attr"`
	RequestID          string                `xml:"requestId"`
	InternetGatewaySet ec2InternetGatewaySet `xml:"internetGatewaySet"`
}

type ec2CreateRouteTableResponse struct {
	XMLName    xml.Name
	Xmlns      string            `xml:"xmlns,attr"`
	RequestID  string            `xml:"requestId"`
	RouteTable ec2RouteTableItem `xml:"routeTable"`
}

type ec2DescribeRouteTablesResponse struct {
	XMLName       xml.Name
	Xmlns         string           `xml:"xmlns,attr"`
	RequestID     string           `xml:"requestId"`
	RouteTableSet ec2RouteTableSet `xml:"routeTableSet"`
}

type ec2AssociateRouteTableResponse struct {
	XMLName       xml.Name
	Xmlns         string `xml:"xmlns,attr"`
	RequestID     string `xml:"requestId"`
	AssociationID string `xml:"associationId"`
}

type ec2CreateNetworkACLResponse struct {
	XMLName    xml.Name
	Xmlns      string            `xml:"xmlns,attr"`
	RequestID  string            `xml:"requestId"`
	NetworkACL ec2NetworkACLItem `xml:"networkAcl"`
}

type ec2DescribeNetworkACLsResponse struct {
	XMLName       xml.Name
	Xmlns         string           `xml:"xmlns,attr"`
	RequestID     string           `xml:"requestId"`
	NetworkACLSet ec2NetworkACLSet `xml:"networkAclSet"`
}

type ec2CreateNetworkInterfaceResponse struct {
	XMLName          xml.Name
	Xmlns            string                  `xml:"xmlns,attr"`
	RequestID        string                  `xml:"requestId"`
	NetworkInterface ec2NetworkInterfaceItem `xml:"networkInterface"`
}

type ec2DescribeNetworkInterfacesResponse struct {
	XMLName             xml.Name
	Xmlns               string                 `xml:"xmlns,attr"`
	RequestID           string                 `xml:"requestId"`
	NetworkInterfaceSet ec2NetworkInterfaceSet `xml:"networkInterfaceSet"`
}

type ec2AttachNetworkInterfaceResponse struct {
	XMLName      xml.Name
	Xmlns        string `xml:"xmlns,attr"`
	RequestID    string `xml:"requestId"`
	AttachmentID string `xml:"attachmentId"`
}

type ec2VPCSet struct {
	Items []ec2VPCItem `xml:"item"`
}

type ec2VPCItem struct {
	VpcID           string    `xml:"vpcId"`
	State           string    `xml:"state"`
	CidrBlock       string    `xml:"cidrBlock"`
	InstanceTenancy string    `xml:"instanceTenancy,omitempty"`
	IsDefault       bool      `xml:"isDefault"`
	DhcpOptionsID   string    `xml:"dhcpOptionsId,omitempty"`
	TagSet          ec2TagSet `xml:"tagSet"`
}

type ec2SubnetSet struct {
	Items []ec2SubnetItem `xml:"item"`
}

type ec2SubnetItem struct {
	SubnetID                string    `xml:"subnetId"`
	VpcID                   string    `xml:"vpcId"`
	State                   string    `xml:"state"`
	CidrBlock               string    `xml:"cidrBlock"`
	AvailabilityZone        string    `xml:"availabilityZone"`
	AvailableIPAddressCount int32     `xml:"availableIpAddressCount"`
	MapPublicIPOnLaunch     bool      `xml:"mapPublicIpOnLaunch"`
	TagSet                  ec2TagSet `xml:"tagSet"`
}

type ec2InternetGatewaySet struct {
	Items []ec2InternetGatewayItem `xml:"item"`
}

type ec2InternetGatewayItem struct {
	InternetGatewayID string                          `xml:"internetGatewayId"`
	AttachmentSet     ec2InternetGatewayAttachmentSet `xml:"attachmentSet"`
	TagSet            ec2TagSet                       `xml:"tagSet"`
}

type ec2InternetGatewayAttachmentSet struct {
	Items []ec2InternetGatewayAttachmentItem `xml:"item"`
}

type ec2InternetGatewayAttachmentItem struct {
	VpcID string `xml:"vpcId"`
	State string `xml:"state"`
}

type ec2RouteTableSet struct {
	Items []ec2RouteTableItem `xml:"item"`
}

type ec2RouteTableItem struct {
	RouteTableID   string                      `xml:"routeTableId"`
	VpcID          string                      `xml:"vpcId"`
	RouteSet       ec2RouteSet                 `xml:"routeSet"`
	AssociationSet ec2RouteTableAssociationSet `xml:"associationSet"`
	TagSet         ec2TagSet                   `xml:"tagSet"`
}

type ec2RouteSet struct {
	Items []ec2RouteItem `xml:"item"`
}

type ec2RouteItem struct {
	DestinationCidrBlock string `xml:"destinationCidrBlock"`
	GatewayID            string `xml:"gatewayId,omitempty"`
	State                string `xml:"state,omitempty"`
	Origin               string `xml:"origin,omitempty"`
}

type ec2RouteTableAssociationSet struct {
	Items []ec2RouteTableAssociationItem `xml:"item"`
}

type ec2RouteTableAssociationItem struct {
	RouteTableAssociationID string `xml:"routeTableAssociationId"`
	RouteTableID            string `xml:"routeTableId"`
	SubnetID                string `xml:"subnetId,omitempty"`
	Main                    bool   `xml:"main,omitempty"`
}

type ec2NetworkACLSet struct {
	Items []ec2NetworkACLItem `xml:"item"`
}

type ec2NetworkACLItem struct {
	NetworkACLID   string                      `xml:"networkAclId"`
	VpcID          string                      `xml:"vpcId"`
	IsDefault      bool                        `xml:"isDefault"`
	EntrySet       ec2NetworkACLEntrySet       `xml:"entrySet"`
	AssociationSet ec2NetworkACLAssociationSet `xml:"associationSet"`
	TagSet         ec2TagSet                   `xml:"tagSet"`
}

type ec2NetworkACLEntrySet struct {
	Items []ec2NetworkACLEntryItem `xml:"item"`
}

type ec2NetworkACLEntryItem struct {
	RuleNumber int32  `xml:"ruleNumber"`
	Protocol   string `xml:"protocol"`
	RuleAction string `xml:"ruleAction"`
	Egress     bool   `xml:"egress"`
	CidrBlock  string `xml:"cidrBlock"`
}

type ec2NetworkACLAssociationSet struct {
	Items []ec2NetworkACLAssociationItem `xml:"item"`
}

type ec2NetworkACLAssociationItem struct {
	NetworkACLAssociationID string `xml:"networkAclAssociationId"`
	NetworkACLID            string `xml:"networkAclId"`
	SubnetID                string `xml:"subnetId"`
}

type ec2NetworkInterfaceSet struct {
	Items []ec2NetworkInterfaceItem `xml:"item"`
}

type ec2NetworkInterfaceItem struct {
	NetworkInterfaceID string                             `xml:"networkInterfaceId"`
	SubnetID           string                             `xml:"subnetId"`
	VpcID              string                             `xml:"vpcId"`
	Description        string                             `xml:"description,omitempty"`
	PrivateIPAddress   string                             `xml:"privateIpAddress,omitempty"`
	Status             string                             `xml:"status,omitempty"`
	SourceDestCheck    bool                               `xml:"sourceDestCheck"`
	GroupSet           ec2GroupSet                        `xml:"groupSet"`
	Attachment         *ec2NetworkInterfaceAttachmentItem `xml:"attachment,omitempty"`
	TagSet             ec2TagSet                          `xml:"tagSet"`
}

type ec2NetworkInterfaceAttachmentItem struct {
	AttachmentID string `xml:"attachmentId"`
	InstanceID   string `xml:"instanceId"`
	DeviceIndex  int32  `xml:"deviceIndex"`
	Status       string `xml:"status"`
	AttachTime   string `xml:"attachTime"`
}
