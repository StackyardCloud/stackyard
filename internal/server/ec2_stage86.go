package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"
)

func (s *Server) handleEC2Stage86Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssociateIpamResourceDiscovery":
		association, err := s.ec2.AssociateIpamResourceDiscovery(
			strings.TrimSpace(r.Form.Get("IpamId")),
			strings.TrimSpace(r.Form.Get("IpamResourceDiscoveryId")),
			parseEC2TagSpecificationsForResource(r.Form, "ipam-resource-discovery-association"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		out := ec2AssociateIpamResourceDiscoveryResponse{
			XMLName:   xml.Name{Local: "AssociateIpamResourceDiscoveryResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamResourceDiscoveryAssociation: ec2IpamResourceDiscoveryAssociationItem{
				IpamARN:                             association.IpamARN,
				IpamID:                              association.IpamID,
				IpamRegion:                          association.IpamRegion,
				IpamResourceDiscoveryAssociationARN: association.IpamResourceDiscoveryAssociationARN,
				IpamResourceDiscoveryAssociationID:  association.IpamResourceDiscoveryAssociationID,
				IpamResourceDiscoveryID:             association.IpamResourceDiscoveryID,
				OwnerID:                             association.OwnerID,
				ResourceDiscoveryStatus:             association.ResourceDiscoveryStatus,
				State:                               association.State,
			},
		}
		isDefault := association.IsDefault
		out.IpamResourceDiscoveryAssociation.IsDefault = &isDefault
		if len(association.Tags) > 0 {
			tags := make([]ec2TagItem, 0, len(association.Tags))
			for _, tag := range association.Tags {
				tags = append(tags, ec2TagItem{Key: tag.Key, Value: tag.Value})
			}
			sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
			out.IpamResourceDiscoveryAssociation.TagSet = &ec2TagSet{Items: tags}
		}
		respondEC2XML(w, out)
		return true
	default:
		return false
	}
}

type ec2AssociateIpamResourceDiscoveryResponse struct {
	XMLName                          xml.Name                                `xml:"AssociateIpamResourceDiscoveryResponse"`
	Xmlns                            string                                  `xml:"xmlns,attr"`
	RequestID                        string                                  `xml:"requestId"`
	IpamResourceDiscoveryAssociation ec2IpamResourceDiscoveryAssociationItem `xml:"ipamResourceDiscoveryAssociation"`
}

type ec2IpamResourceDiscoveryAssociationItem struct {
	IpamARN                             string     `xml:"ipamArn,omitempty"`
	IpamID                              string     `xml:"ipamId,omitempty"`
	IpamRegion                          string     `xml:"ipamRegion,omitempty"`
	IpamResourceDiscoveryAssociationARN string     `xml:"ipamResourceDiscoveryAssociationArn,omitempty"`
	IpamResourceDiscoveryAssociationID  string     `xml:"ipamResourceDiscoveryAssociationId,omitempty"`
	IpamResourceDiscoveryID             string     `xml:"ipamResourceDiscoveryId,omitempty"`
	IsDefault                           *bool      `xml:"isDefault,omitempty"`
	OwnerID                             string     `xml:"ownerId,omitempty"`
	ResourceDiscoveryStatus             string     `xml:"resourceDiscoveryStatus,omitempty"`
	State                               string     `xml:"state,omitempty"`
	TagSet                              *ec2TagSet `xml:"tagSet,omitempty"`
}
