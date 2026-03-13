package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage7Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssignPrivateNatGatewayAddress":
		addresses, err := s.ec2.AssignPrivateNatGatewayAddress(
			strings.TrimSpace(r.Form.Get("NatGatewayId")),
			parseEC2Members(r.Form, "PrivateIpAddress."),
			parseEC2Int32(r.Form.Get("PrivateIpAddressCount"), 0),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2NatGatewayAddressOperationResponse{
			XMLName:      xml.Name{Local: "AssignPrivateNatGatewayAddressResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			NatGatewayID: strings.TrimSpace(r.Form.Get("NatGatewayId")),
			NatGatewayAddressSet: ec2NatGatewayAddressSet{
				Items: ec2NatGatewayAddressItems(addresses),
			},
		})
		return true
	case "AssociateNatGatewayAddress":
		addresses, err := s.ec2.AssociateNatGatewayAddress(
			strings.TrimSpace(r.Form.Get("NatGatewayId")),
			parseEC2Members(r.Form, "AllocationId."),
			parseEC2Members(r.Form, "PrivateIpAddress."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2NatGatewayAddressOperationResponse{
			XMLName:      xml.Name{Local: "AssociateNatGatewayAddressResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			NatGatewayID: strings.TrimSpace(r.Form.Get("NatGatewayId")),
			NatGatewayAddressSet: ec2NatGatewayAddressSet{
				Items: ec2NatGatewayAddressItems(addresses),
			},
		})
		return true
	case "DisassociateNatGatewayAddress":
		addresses, err := s.ec2.DisassociateNatGatewayAddress(
			strings.TrimSpace(r.Form.Get("NatGatewayId")),
			parseEC2Members(r.Form, "AssociationId."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2NatGatewayAddressOperationResponse{
			XMLName:      xml.Name{Local: "DisassociateNatGatewayAddressResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			NatGatewayID: strings.TrimSpace(r.Form.Get("NatGatewayId")),
			NatGatewayAddressSet: ec2NatGatewayAddressSet{
				Items: ec2NatGatewayAddressItems(addresses),
			},
		})
		return true
	case "UnassignPrivateNatGatewayAddress":
		addresses, err := s.ec2.UnassignPrivateNatGatewayAddress(
			strings.TrimSpace(r.Form.Get("NatGatewayId")),
			parseEC2Members(r.Form, "PrivateIpAddress."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2NatGatewayAddressOperationResponse{
			XMLName:      xml.Name{Local: "UnassignPrivateNatGatewayAddressResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			NatGatewayID: strings.TrimSpace(r.Form.Get("NatGatewayId")),
			NatGatewayAddressSet: ec2NatGatewayAddressSet{
				Items: ec2NatGatewayAddressItems(addresses),
			},
		})
		return true
	case "ModifyVpcPeeringConnectionOptions":
		accepter := peeringOptionsPatchFromForm(r, "AccepterPeeringConnectionOptions")
		requester := peeringOptionsPatchFromForm(r, "RequesterPeeringConnectionOptions")
		accepterOpts, requesterOpts, err := s.ec2.ModifyVpcPeeringConnectionOptions(
			strings.TrimSpace(r.Form.Get("VpcPeeringConnectionId")),
			accepter,
			requester,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVpcPeeringConnectionOptionsResponse{
			XMLName:   xml.Name{Local: "ModifyVpcPeeringConnectionOptionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			AccepterPeeringConnectionOptions: ec2PeeringConnectionOptionsItem{
				AllowDNSResolutionFromRemoteVPC:            accepterOpts.AllowDNSResolutionFromRemoteVPC,
				AllowEgressFromLocalClassicLinkToRemoteVPC: accepterOpts.AllowEgressFromLocalClassicLinkToRemoteVPC,
				AllowEgressFromLocalVPCToRemoteClassicLink: accepterOpts.AllowEgressFromLocalVPCToRemoteClassicLink,
			},
			RequesterPeeringConnectionOptions: ec2PeeringConnectionOptionsItem{
				AllowDNSResolutionFromRemoteVPC:            requesterOpts.AllowDNSResolutionFromRemoteVPC,
				AllowEgressFromLocalClassicLinkToRemoteVPC: requesterOpts.AllowEgressFromLocalClassicLinkToRemoteVPC,
				AllowEgressFromLocalVPCToRemoteClassicLink: requesterOpts.AllowEgressFromLocalVPCToRemoteClassicLink,
			},
		})
		return true
	case "ReplaceNetworkAclAssociation":
		replacement, err := s.ec2.ReplaceNetworkACLAssociation(
			strings.TrimSpace(r.Form.Get("AssociationId")),
			strings.TrimSpace(r.Form.Get("NetworkAclId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ReplaceNetworkAclAssociationResponse{
			XMLName:          xml.Name{Local: "ReplaceNetworkAclAssociationResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			NewAssociationID: replacement.ID,
		})
		return true
	case "CreateNetworkInterfacePermission":
		permission, err := s.ec2.CreateNetworkInterfacePermission(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			strings.TrimSpace(r.Form.Get("Permission")),
			strings.TrimSpace(r.Form.Get("AwsAccountId")),
			strings.TrimSpace(r.Form.Get("AwsService")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateNetworkInterfacePermissionResponse{
			XMLName:             xml.Name{Local: "CreateNetworkInterfacePermissionResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			InterfacePermission: ec2NetworkInterfacePermissionItemFrom(permission),
		})
		return true
	case "DeleteNetworkInterfacePermission":
		if err := s.ec2.DeleteNetworkInterfacePermission(strings.TrimSpace(r.Form.Get("NetworkInterfacePermissionId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteNetworkInterfacePermissionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DescribeNetworkInterfaceAttribute":
		attribute, err := s.ec2.DescribeNetworkInterfaceAttribute(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DescribeNetworkInterfaceAttributeResponse{
			XMLName:            xml.Name{Local: "DescribeNetworkInterfaceAttributeResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			NetworkInterfaceID: attribute.NetworkInterfaceID,
			Description:        ec2StringAttributeValue{Value: attribute.Description},
			GroupSet:           ec2GroupIdentifierSet{Items: ec2GroupIdentifierItems(attribute.GroupIDs)},
			SourceDestCheck:    ec2AttributeBooleanValue{Value: attribute.SourceDestCheck},
			Attachment:         ec2NetworkInterfaceAttributeAttachmentFrom(attribute.Attachment),
		})
		return true
	case "DescribeNetworkInterfacePermissions":
		permissions := s.ec2.DescribeNetworkInterfacePermissions(
			parseEC2Members(r.Form, "NetworkInterfacePermissionId."),
			parseEC2FilterValues(r.Form, "network-interface-permission.network-interface-id"),
			parseEC2FilterValues(r.Form, "network-interface-permission.aws-account-id"),
			parseEC2FilterValues(r.Form, "network-interface-permission.aws-service"),
			parseEC2FilterValues(r.Form, "network-interface-permission.permission"),
		)
		respondEC2XML(w, ec2DescribeNetworkInterfacePermissionsResponse{
			XMLName:                     xml.Name{Local: "DescribeNetworkInterfacePermissionsResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			NetworkInterfacePermissions: ec2NetworkInterfacePermissionSet{Items: ec2NetworkInterfacePermissionItems(permissions)},
		})
		return true
	case "ResetNetworkInterfaceAttribute":
		if err := s.ec2.ResetNetworkInterfaceAttribute(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			strings.TrimSpace(r.Form.Get("SourceDestCheck")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ResetNetworkInterfaceAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	default:
		return false
	}
}

func peeringOptionsPatchFromForm(r *http.Request, root string) *ec2svc.PeeringConnectionOptionsPatch {
	patch := &ec2svc.PeeringConnectionOptionsPatch{}
	set := false

	if value, ok := parseEC2OptionalBool(r.Form.Get(root + ".AllowDnsResolutionFromRemoteVpc")); ok {
		patch.AllowDNSResolutionFromRemoteVPC = &value
		set = true
	}
	if value, ok := parseEC2OptionalBool(r.Form.Get(root + ".AllowEgressFromLocalClassicLinkToRemoteVpc")); ok {
		patch.AllowEgressFromLocalClassicLinkToRemoteVPC = &value
		set = true
	}
	if value, ok := parseEC2OptionalBool(r.Form.Get(root + ".AllowEgressFromLocalVpcToRemoteClassicLink")); ok {
		patch.AllowEgressFromLocalVPCToRemoteClassicLink = &value
		set = true
	}

	if !set {
		return nil
	}
	return patch
}

func ec2NatGatewayAddressItems(in []ec2svc.NatGatewayAddress) []ec2NatGatewayAddressItem {
	out := make([]ec2NatGatewayAddressItem, 0, len(in))
	for _, address := range in {
		out = append(out, ec2NatGatewayAddressItem{
			AllocationID:       address.AllocationID,
			AssociationID:      address.AssociationID,
			NetworkInterfaceID: address.NetworkInterfaceID,
			PrivateIP:          address.PrivateIP,
			PublicIP:           address.PublicIP,
			Status:             address.Status,
			IsPrimary:          address.IsPrimary,
		})
	}
	return out
}

func ec2NetworkInterfacePermissionItems(in []ec2svc.NetworkInterfacePermission) []ec2NetworkInterfacePermissionItem {
	out := make([]ec2NetworkInterfacePermissionItem, 0, len(in))
	for _, permission := range in {
		out = append(out, ec2NetworkInterfacePermissionItemFrom(permission))
	}
	return out
}

func ec2NetworkInterfacePermissionItemFrom(in ec2svc.NetworkInterfacePermission) ec2NetworkInterfacePermissionItem {
	return ec2NetworkInterfacePermissionItem{
		AwsAccountID:                 in.AwsAccountID,
		AwsService:                   in.AwsService,
		NetworkInterfaceID:           in.NetworkInterfaceID,
		NetworkInterfacePermissionID: in.ID,
		Permission:                   in.Permission,
		PermissionState:              ec2NetworkInterfacePermissionStateItem{State: in.State, StatusMessage: in.StatusMessage},
	}
}

func ec2GroupIdentifierItems(groupIDs []string) []ec2GroupIdentifierItem {
	out := make([]ec2GroupIdentifierItem, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		groupID = strings.TrimSpace(groupID)
		if groupID == "" {
			continue
		}
		out = append(out, ec2GroupIdentifierItem{GroupID: groupID, GroupName: groupID})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GroupID < out[j].GroupID })
	return out
}

func ec2NetworkInterfaceAttributeAttachmentFrom(in *ec2svc.NetworkInterfaceAttachment) *ec2NetworkInterfaceAttributeAttachmentItem {
	if in == nil {
		return nil
	}
	return &ec2NetworkInterfaceAttributeAttachmentItem{
		AttachmentID: in.ID,
		AttachTime:   in.AttachTime.Format(time.RFC3339),
		DeviceIndex:  in.DeviceIndex,
		InstanceID:   in.InstanceID,
		Status:       in.Status,
	}
}

func ec2BoolPtr(v bool) *bool {
	return &v
}

type ec2NatGatewayAddressOperationResponse struct {
	XMLName              xml.Name
	Xmlns                string                  `xml:"xmlns,attr"`
	RequestID            string                  `xml:"requestId"`
	NatGatewayID         string                  `xml:"natGatewayId"`
	NatGatewayAddressSet ec2NatGatewayAddressSet `xml:"natGatewayAddressSet"`
}

type ec2ModifyVpcPeeringConnectionOptionsResponse struct {
	XMLName                           xml.Name
	Xmlns                             string                          `xml:"xmlns,attr"`
	RequestID                         string                          `xml:"requestId"`
	AccepterPeeringConnectionOptions  ec2PeeringConnectionOptionsItem `xml:"accepterPeeringConnectionOptions"`
	RequesterPeeringConnectionOptions ec2PeeringConnectionOptionsItem `xml:"requesterPeeringConnectionOptions"`
}

type ec2ReplaceNetworkAclAssociationResponse struct {
	XMLName          xml.Name
	Xmlns            string `xml:"xmlns,attr"`
	RequestID        string `xml:"requestId"`
	NewAssociationID string `xml:"newAssociationId"`
}

type ec2CreateNetworkInterfacePermissionResponse struct {
	XMLName             xml.Name
	Xmlns               string                            `xml:"xmlns,attr"`
	RequestID           string                            `xml:"requestId"`
	InterfacePermission ec2NetworkInterfacePermissionItem `xml:"interfacePermission"`
}

type ec2DescribeNetworkInterfaceAttributeResponse struct {
	XMLName                  xml.Name                                    `xml:"DescribeNetworkInterfaceAttributeResponse"`
	Xmlns                    string                                      `xml:"xmlns,attr"`
	RequestID                string                                      `xml:"requestId"`
	Attachment               *ec2NetworkInterfaceAttributeAttachmentItem `xml:"attachment,omitempty"`
	Description              ec2StringAttributeValue                     `xml:"description"`
	GroupSet                 ec2GroupIdentifierSet                       `xml:"groupSet"`
	NetworkInterfaceID       string                                      `xml:"networkInterfaceId"`
	SourceDestCheck          ec2AttributeBooleanValue                    `xml:"sourceDestCheck"`
}

type ec2DescribeNetworkInterfacePermissionsResponse struct {
	XMLName                     xml.Name                         `xml:"DescribeNetworkInterfacePermissionsResponse"`
	Xmlns                       string                           `xml:"xmlns,attr"`
	RequestID                   string                           `xml:"requestId"`
	NetworkInterfacePermissions ec2NetworkInterfacePermissionSet `xml:"networkInterfacePermissions"`
	NextToken                   string                           `xml:"nextToken,omitempty"`
}

type ec2StringAttributeValue struct {
	Value string `xml:"value"`
}

type ec2GroupIdentifierSet struct {
	Items []ec2GroupIdentifierItem `xml:"item"`
}

type ec2GroupIdentifierItem struct {
	GroupID   string `xml:"groupId,omitempty"`
	GroupName string `xml:"groupName,omitempty"`
}

type ec2NetworkInterfaceAttributeAttachmentItem struct {
	AttachmentID string `xml:"attachmentId,omitempty"`
	AttachTime   string `xml:"attachTime,omitempty"`
	DeviceIndex  int32  `xml:"deviceIndex,omitempty"`
	InstanceID   string `xml:"instanceId,omitempty"`
	Status       string `xml:"status,omitempty"`
}

type ec2NetworkInterfacePermissionSet struct {
	Items []ec2NetworkInterfacePermissionItem `xml:"item"`
}

type ec2NetworkInterfacePermissionItem struct {
	AwsAccountID                 string                                 `xml:"awsAccountId,omitempty"`
	AwsService                   string                                 `xml:"awsService,omitempty"`
	NetworkInterfaceID           string                                 `xml:"networkInterfaceId,omitempty"`
	NetworkInterfacePermissionID string                                 `xml:"networkInterfacePermissionId,omitempty"`
	Permission                   string                                 `xml:"permission,omitempty"`
	PermissionState              ec2NetworkInterfacePermissionStateItem `xml:"permissionState"`
}

type ec2NetworkInterfacePermissionStateItem struct {
	State         string `xml:"state,omitempty"`
	StatusMessage string `xml:"statusMessage,omitempty"`
}
