package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage85Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssociateIpamByoasn":
		association, err := s.ec2.AssociateIpamByoasn(
			strings.TrimSpace(r.Form.Get("Asn")),
			strings.TrimSpace(r.Form.Get("Cidr")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateIpamByoasnResponse{
			XMLName:   xml.Name{Local: "AssociateIpamByoasnResponse"},
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
	default:
		return false
	}
}

type ec2AssociateIpamByoasnResponse struct {
	XMLName        xml.Name                   `xml:"AssociateIpamByoasnResponse"`
	Xmlns          string                     `xml:"xmlns,attr"`
	RequestID      string                     `xml:"requestId"`
	AsnAssociation ec2ByoipAsnAssociationItem `xml:"asnAssociation"`
}
