package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage32Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateTransitGatewayMulticastDomain":
		domain, err := s.ec2.CreateTransitGatewayMulticastDomain(
			strings.TrimSpace(r.Form.Get("TransitGatewayId")),
			parseEC2TagSpecificationsForResource(r.Form, "transit-gateway-multicast-domain"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayMulticastDomainResponse{
			XMLName:                       xml.Name{Local: "CreateTransitGatewayMulticastDomainResponse"},
			Xmlns:                         ec2Namespace,
			RequestID:                     "stackyard-request",
			TransitGatewayMulticastDomain: ec2TransitGatewayMulticastDomainItemFrom(domain),
		})
		return true
	case "DeleteTransitGatewayMulticastDomain":
		domain, err := s.ec2.DeleteTransitGatewayMulticastDomain(strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayMulticastDomainResponse{
			XMLName:                       xml.Name{Local: "DeleteTransitGatewayMulticastDomainResponse"},
			Xmlns:                         ec2Namespace,
			RequestID:                     "stackyard-request",
			TransitGatewayMulticastDomain: ec2TransitGatewayMulticastDomainItemFrom(domain),
		})
		return true
	case "CreateTransitGatewayPolicyTable":
		table, err := s.ec2.CreateTransitGatewayPolicyTable(
			strings.TrimSpace(r.Form.Get("TransitGatewayId")),
			parseEC2TagSpecificationsForResource(r.Form, "transit-gateway-policy-table"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayPolicyTableResponse{
			XMLName:                   xml.Name{Local: "CreateTransitGatewayPolicyTableResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			TransitGatewayPolicyTable: ec2TransitGatewayPolicyTableItemFrom(table),
		})
		return true
	case "DeleteTransitGatewayPolicyTable":
		table, err := s.ec2.DeleteTransitGatewayPolicyTable(strings.TrimSpace(r.Form.Get("TransitGatewayPolicyTableId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayPolicyTableResponse{
			XMLName:                   xml.Name{Local: "DeleteTransitGatewayPolicyTableResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			TransitGatewayPolicyTable: ec2TransitGatewayPolicyTableItemFrom(table),
		})
		return true
	case "CreateTransitGatewayRouteTable":
		table, err := s.ec2.CreateTransitGatewayRouteTable(
			strings.TrimSpace(r.Form.Get("TransitGatewayId")),
			parseEC2TagSpecificationsForResource(r.Form, "transit-gateway-route-table"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayRouteTableResponse{
			XMLName:                  xml.Name{Local: "CreateTransitGatewayRouteTableResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			TransitGatewayRouteTable: ec2TransitGatewayRouteTableItemFrom(table),
		})
		return true
	case "DeleteTransitGatewayRouteTable":
		table, err := s.ec2.DeleteTransitGatewayRouteTable(strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayRouteTableResponse{
			XMLName:                  xml.Name{Local: "DeleteTransitGatewayRouteTableResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			TransitGatewayRouteTable: ec2TransitGatewayRouteTableItemFrom(table),
		})
		return true
	default:
		return false
	}
}

func ec2TransitGatewayMulticastDomainItemFrom(in ec2svc.TransitGatewayMulticastDomain) ec2TransitGatewayMulticastDomainItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2TransitGatewayMulticastDomainItem{
		CreationTime:                     in.CreationTime,
		OwnerID:                          in.OwnerID,
		State:                            in.State,
		TagSet:                           ec2TagSet{Items: tags},
		TransitGatewayID:                 in.TransitID,
		TransitGatewayMulticastDomainArn: in.ARN,
		TransitGatewayMulticastDomainID:  in.ID,
	}
}

func ec2TransitGatewayPolicyTableItemFrom(in ec2svc.TransitGatewayPolicyTable) ec2TransitGatewayPolicyTableItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2TransitGatewayPolicyTableItem{
		CreationTime:                in.CreationTime,
		State:                       in.State,
		TagSet:                      ec2TagSet{Items: tags},
		TransitGatewayID:            in.TransitID,
		TransitGatewayPolicyTableID: in.ID,
	}
}

func ec2TransitGatewayRouteTableItemFrom(in ec2svc.TransitGatewayRouteTable) ec2TransitGatewayRouteTableItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2TransitGatewayRouteTableItem{
		CreationTime:                 in.CreationTime,
		DefaultAssociationRouteTable: in.DefaultAssociationRouteTable,
		DefaultPropagationRouteTable: in.DefaultPropagationRouteTable,
		State:                        in.State,
		TagSet:                       ec2TagSet{Items: tags},
		TransitGatewayID:             in.TransitID,
		TransitGatewayRouteTableID:   in.ID,
	}
}

type ec2CreateTransitGatewayMulticastDomainResponse struct {
	XMLName                       xml.Name                             `xml:"CreateTransitGatewayMulticastDomainResponse"`
	Xmlns                         string                               `xml:"xmlns,attr"`
	RequestID                     string                               `xml:"requestId"`
	TransitGatewayMulticastDomain ec2TransitGatewayMulticastDomainItem `xml:"transitGatewayMulticastDomain"`
}

type ec2DeleteTransitGatewayMulticastDomainResponse struct {
	XMLName                       xml.Name                             `xml:"DeleteTransitGatewayMulticastDomainResponse"`
	Xmlns                         string                               `xml:"xmlns,attr"`
	RequestID                     string                               `xml:"requestId"`
	TransitGatewayMulticastDomain ec2TransitGatewayMulticastDomainItem `xml:"transitGatewayMulticastDomain"`
}

type ec2TransitGatewayMulticastDomainItem struct {
	CreationTime                     time.Time `xml:"creationTime,omitempty"`
	OwnerID                          string    `xml:"ownerId,omitempty"`
	State                            string    `xml:"state,omitempty"`
	TagSet                           ec2TagSet `xml:"tagSet"`
	TransitGatewayID                 string    `xml:"transitGatewayId,omitempty"`
	TransitGatewayMulticastDomainArn string    `xml:"transitGatewayMulticastDomainArn,omitempty"`
	TransitGatewayMulticastDomainID  string    `xml:"transitGatewayMulticastDomainId,omitempty"`
}

type ec2CreateTransitGatewayPolicyTableResponse struct {
	XMLName                   xml.Name                         `xml:"CreateTransitGatewayPolicyTableResponse"`
	Xmlns                     string                           `xml:"xmlns,attr"`
	RequestID                 string                           `xml:"requestId"`
	TransitGatewayPolicyTable ec2TransitGatewayPolicyTableItem `xml:"transitGatewayPolicyTable"`
}

type ec2DeleteTransitGatewayPolicyTableResponse struct {
	XMLName                   xml.Name                         `xml:"DeleteTransitGatewayPolicyTableResponse"`
	Xmlns                     string                           `xml:"xmlns,attr"`
	RequestID                 string                           `xml:"requestId"`
	TransitGatewayPolicyTable ec2TransitGatewayPolicyTableItem `xml:"transitGatewayPolicyTable"`
}

type ec2TransitGatewayPolicyTableItem struct {
	CreationTime                time.Time `xml:"creationTime,omitempty"`
	State                       string    `xml:"state,omitempty"`
	TagSet                      ec2TagSet `xml:"tagSet"`
	TransitGatewayID            string    `xml:"transitGatewayId,omitempty"`
	TransitGatewayPolicyTableID string    `xml:"transitGatewayPolicyTableId,omitempty"`
}

type ec2CreateTransitGatewayRouteTableResponse struct {
	XMLName                  xml.Name                        `xml:"CreateTransitGatewayRouteTableResponse"`
	Xmlns                    string                          `xml:"xmlns,attr"`
	RequestID                string                          `xml:"requestId"`
	TransitGatewayRouteTable ec2TransitGatewayRouteTableItem `xml:"transitGatewayRouteTable"`
}

type ec2DeleteTransitGatewayRouteTableResponse struct {
	XMLName                  xml.Name                        `xml:"DeleteTransitGatewayRouteTableResponse"`
	Xmlns                    string                          `xml:"xmlns,attr"`
	RequestID                string                          `xml:"requestId"`
	TransitGatewayRouteTable ec2TransitGatewayRouteTableItem `xml:"transitGatewayRouteTable"`
}

type ec2TransitGatewayRouteTableItem struct {
	CreationTime                 time.Time `xml:"creationTime,omitempty"`
	DefaultAssociationRouteTable bool      `xml:"defaultAssociationRouteTable"`
	DefaultPropagationRouteTable bool      `xml:"defaultPropagationRouteTable"`
	State                        string    `xml:"state,omitempty"`
	TagSet                       ec2TagSet `xml:"tagSet"`
	TransitGatewayID             string    `xml:"transitGatewayId,omitempty"`
	TransitGatewayRouteTableID   string    `xml:"transitGatewayRouteTableId,omitempty"`
}
