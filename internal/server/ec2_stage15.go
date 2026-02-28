package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage15Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "EnableAddressTransfer":
		transfer, err := s.ec2.EnableAddressTransfer(
			strings.TrimSpace(r.Form.Get("AllocationId")),
			strings.TrimSpace(r.Form.Get("TransferAccountId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AddressTransferResponse{
			XMLName:         xml.Name{Local: "EnableAddressTransferResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			AddressTransfer: ec2AddressTransferItemFrom(transfer),
		})
		return true
	case "DisableAddressTransfer":
		transfer, err := s.ec2.DisableAddressTransfer(strings.TrimSpace(r.Form.Get("AllocationId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AddressTransferResponse{
			XMLName:         xml.Name{Local: "DisableAddressTransferResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			AddressTransfer: ec2AddressTransferItemFrom(transfer),
		})
		return true
	case "AcceptAddressTransfer":
		transfer, err := s.ec2.AcceptAddressTransfer(strings.TrimSpace(r.Form.Get("Address")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AddressTransferResponse{
			XMLName:         xml.Name{Local: "AcceptAddressTransferResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			AddressTransfer: ec2AddressTransferItemFrom(transfer),
		})
		return true
	case "DescribeAddressTransfers":
		transfers := s.ec2.DescribeAddressTransfers(parseEC2Members(r.Form, "AllocationId."))
		respondEC2XML(w, ec2DescribeAddressTransfersResponse{
			XMLName:            xml.Name{Local: "DescribeAddressTransfersResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			AddressTransferSet: ec2AddressTransferSet{Items: ec2AddressTransferItems(transfers)},
		})
		return true
	case "DescribeMovingAddresses":
		statuses := s.ec2.DescribeMovingAddresses(parseEC2Members(r.Form, "PublicIp."))
		respondEC2XML(w, ec2DescribeMovingAddressesResponse{
			XMLName:                xml.Name{Local: "DescribeMovingAddressesResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			MovingAddressStatusSet: ec2MovingAddressStatusSet{Items: ec2MovingAddressStatusItems(statuses)},
		})
		return true
	default:
		return false
	}
}

func ec2AddressTransferItems(in []ec2svc.AddressTransfer) []ec2AddressTransferItem {
	out := make([]ec2AddressTransferItem, 0, len(in))
	for _, transfer := range in {
		out = append(out, ec2AddressTransferItemFrom(transfer))
	}
	return out
}

func ec2AddressTransferItemFrom(in ec2svc.AddressTransfer) ec2AddressTransferItem {
	item := ec2AddressTransferItem{
		AddressTransferStatus: in.AddressTransferStatus,
		AllocationID:          in.AllocationID,
		PublicIP:              in.PublicIP,
		TransferAccountID:     in.TransferAccountID,
	}
	if in.TransferOfferAcceptedTimestamp != nil {
		item.TransferOfferAcceptedTimestamp = in.TransferOfferAcceptedTimestamp.Format(timeRFC3339UTC)
	}
	if in.TransferOfferExpirationTime != nil {
		item.TransferOfferExpirationTime = in.TransferOfferExpirationTime.Format(timeRFC3339UTC)
	}
	return item
}

func ec2MovingAddressStatusItems(in []ec2svc.MovingAddressStatus) []ec2MovingAddressStatusItem {
	out := make([]ec2MovingAddressStatusItem, 0, len(in))
	for _, status := range in {
		out = append(out, ec2MovingAddressStatusItem{
			MoveStatus: status.MoveStatus,
			PublicIP:   status.PublicIP,
		})
	}
	return out
}

type ec2AddressTransferResponse struct {
	XMLName         xml.Name
	Xmlns           string                 `xml:"xmlns,attr"`
	RequestID       string                 `xml:"requestId"`
	AddressTransfer ec2AddressTransferItem `xml:"addressTransfer"`
}

type ec2DescribeAddressTransfersResponse struct {
	XMLName            xml.Name              `xml:"DescribeAddressTransfersResponse"`
	Xmlns              string                `xml:"xmlns,attr"`
	RequestID          string                `xml:"requestId"`
	AddressTransferSet ec2AddressTransferSet `xml:"addressTransferSet"`
	NextToken          string                `xml:"nextToken,omitempty"`
}

type ec2AddressTransferSet struct {
	Items []ec2AddressTransferItem `xml:"item"`
}

type ec2AddressTransferItem struct {
	AddressTransferStatus          string `xml:"addressTransferStatus,omitempty"`
	AllocationID                   string `xml:"allocationId,omitempty"`
	PublicIP                       string `xml:"publicIp,omitempty"`
	TransferAccountID              string `xml:"transferAccountId,omitempty"`
	TransferOfferAcceptedTimestamp string `xml:"transferOfferAcceptedTimestamp,omitempty"`
	TransferOfferExpirationTime    string `xml:"transferOfferExpirationTimestamp,omitempty"`
}

type ec2DescribeMovingAddressesResponse struct {
	XMLName                xml.Name                  `xml:"DescribeMovingAddressesResponse"`
	Xmlns                  string                    `xml:"xmlns,attr"`
	RequestID              string                    `xml:"requestId"`
	MovingAddressStatusSet ec2MovingAddressStatusSet `xml:"movingAddressStatusSet"`
	NextToken              string                    `xml:"nextToken,omitempty"`
}

type ec2MovingAddressStatusSet struct {
	Items []ec2MovingAddressStatusItem `xml:"item"`
}

type ec2MovingAddressStatusItem struct {
	MoveStatus string `xml:"moveStatus,omitempty"`
	PublicIP   string `xml:"publicIp,omitempty"`
}
