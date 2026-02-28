package server

import (
	"encoding/xml"
	"net/http"
)

func (s *Server) handleEC2Stage78Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AcceptReservedInstancesExchangeQuote":
		exchangeID, err := s.ec2.AcceptReservedInstancesExchangeQuote(
			parseEC2MembersWithAliases(r.Form, "ReservedInstanceId", "ReservedInstanceIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AcceptReservedInstancesExchangeQuoteResponse{
			XMLName:    xml.Name{Local: "AcceptReservedInstancesExchangeQuoteResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			ExchangeID: exchangeID,
		})
		return true
	default:
		return false
	}
}

type ec2AcceptReservedInstancesExchangeQuoteResponse struct {
	XMLName    xml.Name
	Xmlns      string `xml:"xmlns,attr"`
	RequestID  string `xml:"requestId"`
	ExchangeID string `xml:"exchangeId,omitempty"`
}
