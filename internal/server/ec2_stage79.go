package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage79Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AdvertiseByoipCidr":
		byoipCidr, err := s.ec2.AdvertiseByoipCidr(
			strings.TrimSpace(r.Form.Get("Cidr")),
			parseEC2OptionalString(r.Form.Get("Asn")),
			parseEC2OptionalString(r.Form.Get("NetworkBorderGroup")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AdvertiseByoipCidrResponse{
			XMLName:   xml.Name{Local: "AdvertiseByoipCidrResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ByoipCidr: ec2ByoipCidrItemFrom(byoipCidr),
		})
		return true
	default:
		return false
	}
}

func ec2ByoipCidrItemFrom(in ec2svc.ByoipCidr) ec2ByoipCidrItem {
	out := ec2ByoipCidrItem{
		Cidr:               in.Cidr,
		Description:        in.Description,
		NetworkBorderGroup: in.NetworkBorderGroup,
		State:              in.State,
		StatusMessage:      in.StatusMessage,
	}
	if len(in.AsnAssociations) > 0 {
		items := make([]ec2ByoipAsnAssociationItem, 0, len(in.AsnAssociations))
		for _, association := range in.AsnAssociations {
			items = append(items, ec2ByoipAsnAssociationItem{
				Asn:           association.Asn,
				Cidr:          association.Cidr,
				State:         association.State,
				StatusMessage: association.StatusMessage,
			})
		}
		out.AsnAssociationSet = &ec2ByoipAsnAssociationSet{Items: items}
	}
	return out
}

type ec2AdvertiseByoipCidrResponse struct {
	XMLName   xml.Name         `xml:"AdvertiseByoipCidrResponse"`
	Xmlns     string           `xml:"xmlns,attr"`
	RequestID string           `xml:"requestId"`
	ByoipCidr ec2ByoipCidrItem `xml:"byoipCidr"`
}

type ec2ByoipCidrItem struct {
	AsnAssociationSet  *ec2ByoipAsnAssociationSet `xml:"asnAssociationSet,omitempty"`
	Cidr               string                     `xml:"cidr,omitempty"`
	Description        string                     `xml:"description,omitempty"`
	NetworkBorderGroup string                     `xml:"networkBorderGroup,omitempty"`
	State              string                     `xml:"state,omitempty"`
	StatusMessage      string                     `xml:"statusMessage,omitempty"`
}

type ec2ByoipAsnAssociationSet struct {
	Items []ec2ByoipAsnAssociationItem `xml:"item"`
}

type ec2ByoipAsnAssociationItem struct {
	Asn           string `xml:"asn,omitempty"`
	Cidr          string `xml:"cidr,omitempty"`
	State         string `xml:"state,omitempty"`
	StatusMessage string `xml:"statusMessage,omitempty"`
}
