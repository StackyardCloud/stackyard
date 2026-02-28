package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage123Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeVolumeAttribute":
		volumeAttribute, err := s.ec2.DescribeVolumeAttribute(
			strings.TrimSpace(r.Form.Get("Attribute")),
			strings.TrimSpace(r.Form.Get("VolumeId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		out := ec2Stage123DescribeVolumeAttributeResponse{
			XMLName:   xml.Name{Local: "DescribeVolumeAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VolumeID:  volumeAttribute.VolumeID,
		}
		if volumeAttribute.AutoEnableIO != nil {
			out.AutoEnableIO = &ec2Stage123AttributeBooleanValue{Value: volumeAttribute.AutoEnableIO}
		}
		if len(volumeAttribute.ProductCodes) > 0 {
			out.ProductCodes = &ec2Stage123ProductCodeSet{Items: ec2Stage123ProductCodeItemsFrom(volumeAttribute.ProductCodes)}
		}
		respondEC2XML(w, out)
		return true
	case "DescribeVolumeStatus":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		volumeStatuses, nextToken, err := s.ec2.DescribeVolumeStatus(
			parseEC2MembersOrItemList(r.Form, "VolumeId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage123DescribeVolumeStatusResponse{
			XMLName:         xml.Name{Local: "DescribeVolumeStatusResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			VolumeStatusSet: ec2Stage123VolumeStatusSet{Items: ec2Stage123VolumeStatusItemsFrom(volumeStatuses)},
			NextToken:       nextToken,
		})
		return true
	case "DescribeVolumesModifications":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		volumeModifications, nextToken, err := s.ec2.DescribeVolumesModifications(
			parseEC2MembersOrItemList(r.Form, "VolumeId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage123DescribeVolumesModificationsResponse{
			XMLName:               xml.Name{Local: "DescribeVolumesModificationsResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			VolumeModificationSet: ec2Stage123VolumeModificationSet{Items: ec2Stage123VolumeModificationItemsFrom(volumeModifications)},
			NextToken:             nextToken,
		})
		return true
	case "DisassociateCapacityReservationBillingOwner":
		ret, err := s.ec2.DisassociateCapacityReservationBillingOwner(
			strings.TrimSpace(r.Form.Get("CapacityReservationId")),
			strings.TrimSpace(r.Form.Get("UnusedReservationBillingOwnerId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DisassociateCapacityReservationBillingOwnerResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "DisassociateEnclaveCertificateIamRole":
		ret, err := s.ec2.DisassociateEnclaveCertificateIamRole(
			strings.TrimSpace(r.Form.Get("CertificateArn")),
			strings.TrimSpace(r.Form.Get("RoleArn")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DisassociateEnclaveCertificateIamRoleResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "DisassociateInstanceEventWindow":
		association, err := s.ec2.DisassociateInstanceEventWindow(
			strings.TrimSpace(r.Form.Get("InstanceEventWindowId")),
			parseEC2MembersOrItemList(r.Form, "AssociationTarget.DedicatedHostId"),
			parseEC2MembersOrItemList(r.Form, "AssociationTarget.InstanceId"),
			parseEC2Tags(r.Form, "AssociationTarget.InstanceTag."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage123DisassociateInstanceEventWindowResponse{
			XMLName:             xml.Name{Local: "DisassociateInstanceEventWindowResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			InstanceEventWindow: ec2Stage123InstanceEventWindowItemFromAssociation(association),
		})
		return true
	case "DisassociateIpamByoasn":
		association, err := s.ec2.DisassociateIpamByoasn(
			strings.TrimSpace(r.Form.Get("Asn")),
			strings.TrimSpace(r.Form.Get("Cidr")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage123DisassociateIpamByoasnResponse{
			XMLName:   xml.Name{Local: "DisassociateIpamByoasnResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			AsnAssociation: ec2ByoipAsnAssociationItem{
				Asn:           association.Asn,
				Cidr:          association.Cidr,
				State:         association.State,
				StatusMessage: association.StatusMessage,
			},
		})
		return true
	case "DisassociateIpamResourceDiscovery":
		association, err := s.ec2.DisassociateIpamResourceDiscovery(strings.TrimSpace(r.Form.Get("IpamResourceDiscoveryAssociationId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage123DisassociateIpamResourceDiscoveryResponse{
			XMLName:                          xml.Name{Local: "DisassociateIpamResourceDiscoveryResponse"},
			Xmlns:                            ec2Namespace,
			RequestID:                        "stackyard-request",
			IpamResourceDiscoveryAssociation: ec2Stage123IpamResourceDiscoveryAssociationItemFrom(association),
		})
		return true
	case "ExportImage":
		tags := parseEC2TagSpecificationsForResource(r.Form, "export-image-task")
		if len(tags) == 0 {
			tags = parseEC2TagSpecificationsForResource(r.Form, "image")
		}
		exportImageTask, err := s.ec2.ExportImage(
			strings.TrimSpace(r.Form.Get("ClientToken")),
			strings.TrimSpace(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("DiskImageFormat")),
			strings.TrimSpace(r.Form.Get("ImageId")),
			strings.TrimSpace(r.Form.Get("RoleName")),
			parseEC2OptionalString(r.Form.Get("S3ExportLocation.S3Bucket")),
			parseEC2OptionalString(r.Form.Get("S3ExportLocation.S3Prefix")),
			tags,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		out := ec2Stage123ExportImageResponse{
			XMLName:           xml.Name{Local: "ExportImageResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			Description:       exportImageTask.Description,
			DiskImageFormat:   exportImageTask.DiskImageFormat,
			ExportImageTaskID: exportImageTask.ExportImageTaskID,
			ImageID:           exportImageTask.ImageID,
			Progress:          exportImageTask.Progress,
			RoleName:          exportImageTask.RoleName,
			Status:            exportImageTask.Status,
			StatusMessage:     exportImageTask.StatusMessage,
		}
		if strings.TrimSpace(exportImageTask.S3ExportLocation.S3Bucket) != "" || strings.TrimSpace(exportImageTask.S3ExportLocation.S3Prefix) != "" {
			out.S3ExportLocation = &ec2Stage123ExportTaskS3LocationItem{
				S3Bucket: exportImageTask.S3ExportLocation.S3Bucket,
				S3Prefix: exportImageTask.S3ExportLocation.S3Prefix,
			}
		}
		if len(exportImageTask.Tags) > 0 {
			out.TagSet = &ec2TagSet{Items: ec2TagItemsFromMap(exportImageTask.Tags)}
		}
		respondEC2XML(w, out)
		return true
	case "GetAssociatedEnclaveCertificateIamRoles":
		associatedRoles, err := s.ec2.GetAssociatedEnclaveCertificateIamRoles(strings.TrimSpace(r.Form.Get("CertificateArn")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage123GetAssociatedEnclaveCertificateIamRolesResponse{
			XMLName:           xml.Name{Local: "GetAssociatedEnclaveCertificateIamRolesResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			AssociatedRoleSet: ec2Stage123AssociatedRoleSet{Items: ec2Stage123AssociatedRoleItemsFrom(associatedRoles)},
		})
		return true
	default:
		return false
	}
}

func ec2Stage123ProductCodeItemsFrom(in []ec2svc.VolumeProductCode) []ec2Stage123ProductCodeItem {
	out := make([]ec2Stage123ProductCodeItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage123ProductCodeItem{
			ProductCode: item.ProductCodeID,
			Type:        item.ProductCodeType,
		})
	}
	return out
}

func ec2Stage123VolumeStatusItemsFrom(in []ec2svc.VolumeStatus) []ec2Stage123VolumeStatusItem {
	out := make([]ec2Stage123VolumeStatusItem, 0, len(in))
	for _, item := range in {
		status := item.Status
		out = append(out, ec2Stage123VolumeStatusItem{
			AvailabilityZone:   item.AvailabilityZone,
			AvailabilityZoneID: item.AvailabilityZoneID,
			VolumeID:           item.VolumeID,
			VolumeStatus:       &ec2Stage123VolumeStatusInfoItem{Status: status},
		})
	}
	return out
}

func ec2Stage123VolumeModificationItemsFrom(in []ec2svc.VolumeModification) []ec2VolumeModificationItem {
	out := make([]ec2VolumeModificationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2VolumeModificationItemFrom(item))
	}
	return out
}

func ec2Stage123InstanceEventWindowItemFromAssociation(in ec2svc.InstanceEventWindowAssociation) ec2InstanceEventWindowItem {
	out := ec2InstanceEventWindowItem{
		InstanceEventWindowID: in.InstanceEventWindowID,
		State:                 in.State,
	}
	target := ec2InstanceEventWindowAssociationTargetItem{}
	if len(in.DedicatedHostIDs) > 0 {
		target.DedicatedHostIDSet = &ec2StringSet{Items: append([]string(nil), in.DedicatedHostIDs...)}
	}
	if len(in.InstanceIDs) > 0 {
		target.InstanceIDSet = &ec2StringSet{Items: append([]string(nil), in.InstanceIDs...)}
	}
	if len(in.InstanceTags) > 0 {
		tags := make([]ec2TagItem, 0, len(in.InstanceTags))
		for _, tag := range in.InstanceTags {
			tags = append(tags, ec2TagItem{Key: tag.Key, Value: tag.Value})
		}
		sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
		target.TagSet = &ec2TagSet{Items: tags}
	}
	if target.DedicatedHostIDSet != nil || target.InstanceIDSet != nil || target.TagSet != nil {
		out.AssociationTarget = &target
	}
	return out
}

func ec2Stage123IpamResourceDiscoveryAssociationItemFrom(in ec2svc.IpamResourceDiscoveryAssociation) ec2IpamResourceDiscoveryAssociationItem {
	out := ec2IpamResourceDiscoveryAssociationItem{
		IpamARN:                             in.IpamARN,
		IpamID:                              in.IpamID,
		IpamRegion:                          in.IpamRegion,
		IpamResourceDiscoveryAssociationARN: in.IpamResourceDiscoveryAssociationARN,
		IpamResourceDiscoveryAssociationID:  in.IpamResourceDiscoveryAssociationID,
		IpamResourceDiscoveryID:             in.IpamResourceDiscoveryID,
		OwnerID:                             in.OwnerID,
		ResourceDiscoveryStatus:             in.ResourceDiscoveryStatus,
		State:                               in.State,
	}
	isDefault := in.IsDefault
	out.IsDefault = &isDefault
	if len(in.Tags) > 0 {
		tags := make([]ec2TagItem, 0, len(in.Tags))
		for _, tag := range in.Tags {
			tags = append(tags, ec2TagItem{Key: tag.Key, Value: tag.Value})
		}
		sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
		out.TagSet = &ec2TagSet{Items: tags}
	}
	return out
}

func ec2Stage123AssociatedRoleItemsFrom(in []ec2svc.EnclaveCertificateRoleAssociation) []ec2Stage123AssociatedRoleItem {
	out := make([]ec2Stage123AssociatedRoleItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage123AssociatedRoleItem{
			AssociatedRoleArn:       item.AssociatedRoleArn,
			CertificateS3BucketName: item.CertificateS3BucketName,
			CertificateS3ObjectKey:  item.CertificateS3ObjectKey,
			EncryptionKmsKeyID:      item.EncryptionKmsKeyID,
		})
	}
	return out
}

type ec2Stage123DescribeVolumeAttributeResponse struct {
	XMLName      xml.Name                          `xml:"DescribeVolumeAttributeResponse"`
	Xmlns        string                            `xml:"xmlns,attr"`
	RequestID    string                            `xml:"requestId"`
	AutoEnableIO *ec2Stage123AttributeBooleanValue `xml:"autoEnableIO,omitempty"`
	ProductCodes *ec2Stage123ProductCodeSet        `xml:"productCodes,omitempty"`
	VolumeID     string                            `xml:"volumeId,omitempty"`
}

type ec2Stage123AttributeBooleanValue struct {
	Value *bool `xml:"value,omitempty"`
}

type ec2Stage123ProductCodeSet struct {
	Items []ec2Stage123ProductCodeItem `xml:"item"`
}

type ec2Stage123ProductCodeItem struct {
	ProductCode string `xml:"productCode,omitempty"`
	Type        string `xml:"type,omitempty"`
}

type ec2Stage123DescribeVolumeStatusResponse struct {
	XMLName         xml.Name                   `xml:"DescribeVolumeStatusResponse"`
	Xmlns           string                     `xml:"xmlns,attr"`
	RequestID       string                     `xml:"requestId"`
	VolumeStatusSet ec2Stage123VolumeStatusSet `xml:"volumeStatusSet"`
	NextToken       *string                    `xml:"nextToken,omitempty"`
}

type ec2Stage123VolumeStatusSet struct {
	Items []ec2Stage123VolumeStatusItem `xml:"item"`
}

type ec2Stage123VolumeStatusItem struct {
	AvailabilityZone   string                           `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID string                           `xml:"availabilityZoneId,omitempty"`
	VolumeID           string                           `xml:"volumeId,omitempty"`
	VolumeStatus       *ec2Stage123VolumeStatusInfoItem `xml:"volumeStatus,omitempty"`
}

type ec2Stage123VolumeStatusInfoItem struct {
	Status string `xml:"status,omitempty"`
}

type ec2Stage123DescribeVolumesModificationsResponse struct {
	XMLName               xml.Name                         `xml:"DescribeVolumesModificationsResponse"`
	Xmlns                 string                           `xml:"xmlns,attr"`
	RequestID             string                           `xml:"requestId"`
	VolumeModificationSet ec2Stage123VolumeModificationSet `xml:"volumeModificationSet"`
	NextToken             *string                          `xml:"nextToken,omitempty"`
}

type ec2Stage123VolumeModificationSet struct {
	Items []ec2VolumeModificationItem `xml:"item"`
}

type ec2Stage123DisassociateInstanceEventWindowResponse struct {
	XMLName             xml.Name                   `xml:"DisassociateInstanceEventWindowResponse"`
	Xmlns               string                     `xml:"xmlns,attr"`
	RequestID           string                     `xml:"requestId"`
	InstanceEventWindow ec2InstanceEventWindowItem `xml:"instanceEventWindow"`
}

type ec2Stage123DisassociateIpamByoasnResponse struct {
	XMLName        xml.Name                   `xml:"DisassociateIpamByoasnResponse"`
	Xmlns          string                     `xml:"xmlns,attr"`
	RequestID      string                     `xml:"requestId"`
	AsnAssociation ec2ByoipAsnAssociationItem `xml:"asnAssociation"`
}

type ec2Stage123DisassociateIpamResourceDiscoveryResponse struct {
	XMLName                          xml.Name                                `xml:"DisassociateIpamResourceDiscoveryResponse"`
	Xmlns                            string                                  `xml:"xmlns,attr"`
	RequestID                        string                                  `xml:"requestId"`
	IpamResourceDiscoveryAssociation ec2IpamResourceDiscoveryAssociationItem `xml:"ipamResourceDiscoveryAssociation"`
}

type ec2Stage123ExportImageResponse struct {
	XMLName           xml.Name                             `xml:"ExportImageResponse"`
	Xmlns             string                               `xml:"xmlns,attr"`
	RequestID         string                               `xml:"requestId"`
	Description       string                               `xml:"description,omitempty"`
	DiskImageFormat   string                               `xml:"diskImageFormat,omitempty"`
	ExportImageTaskID string                               `xml:"exportImageTaskId,omitempty"`
	ImageID           string                               `xml:"imageId,omitempty"`
	Progress          string                               `xml:"progress,omitempty"`
	RoleName          string                               `xml:"roleName,omitempty"`
	S3ExportLocation  *ec2Stage123ExportTaskS3LocationItem `xml:"s3ExportLocation,omitempty"`
	Status            string                               `xml:"status,omitempty"`
	StatusMessage     string                               `xml:"statusMessage,omitempty"`
	TagSet            *ec2TagSet                           `xml:"tagSet,omitempty"`
}

type ec2Stage123ExportTaskS3LocationItem struct {
	S3Bucket string `xml:"s3Bucket,omitempty"`
	S3Prefix string `xml:"s3Prefix,omitempty"`
}

type ec2Stage123GetAssociatedEnclaveCertificateIamRolesResponse struct {
	XMLName           xml.Name                     `xml:"GetAssociatedEnclaveCertificateIamRolesResponse"`
	Xmlns             string                       `xml:"xmlns,attr"`
	RequestID         string                       `xml:"requestId"`
	AssociatedRoleSet ec2Stage123AssociatedRoleSet `xml:"associatedRoleSet"`
}

type ec2Stage123AssociatedRoleSet struct {
	Items []ec2Stage123AssociatedRoleItem `xml:"item"`
}

type ec2Stage123AssociatedRoleItem struct {
	AssociatedRoleArn       string `xml:"associatedRoleArn,omitempty"`
	CertificateS3BucketName string `xml:"certificateS3BucketName,omitempty"`
	CertificateS3ObjectKey  string `xml:"certificateS3ObjectKey,omitempty"`
	EncryptionKmsKeyID      string `xml:"encryptionKmsKeyId,omitempty"`
}
