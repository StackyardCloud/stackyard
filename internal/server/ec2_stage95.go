package server

import (
	"encoding/xml"
	"net/http"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage95Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelReservedInstancesListing":
		listings, err := s.ec2.CancelReservedInstancesListing(r.Form.Get("ReservedInstancesListingId"))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage95CancelReservedInstancesListingResponse{
			XMLName:                   xml.Name{Local: "CancelReservedInstancesListingResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			ReservedInstancesListings: ec2Stage95ReservedInstancesListingSet{Items: ec2Stage95ReservedInstancesListingsFrom(listings)},
		})
		return true
	default:
		return false
	}
}

func ec2Stage95ReservedInstancesListingsFrom(in []ec2svc.ReservedInstancesListing) []ec2Stage95ReservedInstancesListingItem {
	out := make([]ec2Stage95ReservedInstancesListingItem, 0, len(in))
	for _, listing := range in {
		item := ec2Stage95ReservedInstancesListingItem{
			ReservedInstancesID:        listing.ReservedInstancesID,
			ReservedInstancesListingID: listing.ReservedInstancesListingID,
			Status:                     listing.Status,
			StatusMessage:              listing.StatusMessage,
		}
		if !listing.CreateDate.IsZero() {
			item.CreateDate = listing.CreateDate.UTC().Format(time.RFC3339)
		}
		if !listing.UpdateDate.IsZero() {
			item.UpdateDate = listing.UpdateDate.UTC().Format(time.RFC3339)
		}
		out = append(out, item)
	}
	return out
}

type ec2Stage95CancelReservedInstancesListingResponse struct {
	XMLName                   xml.Name                              `xml:"CancelReservedInstancesListingResponse"`
	Xmlns                     string                                `xml:"xmlns,attr"`
	RequestID                 string                                `xml:"requestId"`
	ReservedInstancesListings ec2Stage95ReservedInstancesListingSet `xml:"reservedInstancesListingsSet"`
}

type ec2Stage95ReservedInstancesListingSet struct {
	Items []ec2Stage95ReservedInstancesListingItem `xml:"item"`
}

type ec2Stage95ReservedInstancesListingItem struct {
	CreateDate                 string `xml:"createDate,omitempty"`
	ReservedInstancesID        string `xml:"reservedInstancesId,omitempty"`
	ReservedInstancesListingID string `xml:"reservedInstancesListingId,omitempty"`
	Status                     string `xml:"status,omitempty"`
	StatusMessage              string `xml:"statusMessage,omitempty"`
	UpdateDate                 string `xml:"updateDate,omitempty"`
}
