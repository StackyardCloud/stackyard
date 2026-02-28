package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage41Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateTransitGatewayRouteTableAnnouncement":
		announcement, err := s.ec2.CreateTransitGatewayRouteTableAnnouncement(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("PeeringAttachmentId")),
			parseEC2TagSpecificationsForResource(r.Form, "transit-gateway-route-table-announcement"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayRouteTableAnnouncementResponse{
			XMLName:                              xml.Name{Local: "CreateTransitGatewayRouteTableAnnouncementResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			TransitGatewayRouteTableAnnouncement: ec2TransitGatewayRouteTableAnnouncementItemFrom(announcement),
		})
		return true
	case "DeleteTransitGatewayRouteTableAnnouncement":
		announcement, err := s.ec2.DeleteTransitGatewayRouteTableAnnouncement(strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableAnnouncementId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayRouteTableAnnouncementResponse{
			XMLName:                              xml.Name{Local: "DeleteTransitGatewayRouteTableAnnouncementResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			TransitGatewayRouteTableAnnouncement: ec2TransitGatewayRouteTableAnnouncementItemFrom(announcement),
		})
		return true
	case "DescribeTransitGatewayRouteTableAnnouncements":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		announcements, nextToken, err := s.ec2.DescribeTransitGatewayRouteTableAnnouncements(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayRouteTableAnnouncementIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewayRouteTableAnnouncementsResponse{
			XMLName:                               xml.Name{Local: "DescribeTransitGatewayRouteTableAnnouncementsResponse"},
			Xmlns:                                 ec2Namespace,
			RequestID:                             "stackyard-request",
			TransitGatewayRouteTableAnnouncements: ec2TransitGatewayRouteTableAnnouncementSet{Items: ec2TransitGatewayRouteTableAnnouncementItems(announcements)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	default:
		return false
	}
}

func ec2TransitGatewayRouteTableAnnouncementItemFrom(in ec2svc.TransitGatewayRouteTableAnnouncement) ec2TransitGatewayRouteTableAnnouncementItem {
	return ec2TransitGatewayRouteTableAnnouncementItem{
		AnnouncementDirection:                  in.AnnouncementDirection,
		CoreNetworkID:                          in.CoreNetworkID,
		CreationTime:                           in.CreationTime,
		PeerCoreNetworkID:                      in.PeerCoreNetworkID,
		PeerTransitGatewayID:                   in.PeerTransitGatewayID,
		PeeringAttachmentID:                    in.PeeringAttachmentID,
		State:                                  in.State,
		TagSet:                                 ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TransitGatewayID:                       in.TransitGatewayID,
		TransitGatewayRouteTableAnnouncementID: in.TransitGatewayRouteTableAnnouncementID,
		TransitGatewayRouteTableID:             in.TransitGatewayRouteTableID,
	}
}

func ec2TransitGatewayRouteTableAnnouncementItems(in []ec2svc.TransitGatewayRouteTableAnnouncement) []ec2TransitGatewayRouteTableAnnouncementItem {
	out := make([]ec2TransitGatewayRouteTableAnnouncementItem, 0, len(in))
	for _, announcement := range in {
		out = append(out, ec2TransitGatewayRouteTableAnnouncementItemFrom(announcement))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TransitGatewayRouteTableAnnouncementID < out[j].TransitGatewayRouteTableAnnouncementID
	})
	return out
}

type ec2CreateTransitGatewayRouteTableAnnouncementResponse struct {
	XMLName                              xml.Name                                    `xml:"CreateTransitGatewayRouteTableAnnouncementResponse"`
	Xmlns                                string                                      `xml:"xmlns,attr"`
	RequestID                            string                                      `xml:"requestId"`
	TransitGatewayRouteTableAnnouncement ec2TransitGatewayRouteTableAnnouncementItem `xml:"transitGatewayRouteTableAnnouncement"`
}

type ec2DeleteTransitGatewayRouteTableAnnouncementResponse struct {
	XMLName                              xml.Name                                    `xml:"DeleteTransitGatewayRouteTableAnnouncementResponse"`
	Xmlns                                string                                      `xml:"xmlns,attr"`
	RequestID                            string                                      `xml:"requestId"`
	TransitGatewayRouteTableAnnouncement ec2TransitGatewayRouteTableAnnouncementItem `xml:"transitGatewayRouteTableAnnouncement"`
}

type ec2DescribeTransitGatewayRouteTableAnnouncementsResponse struct {
	XMLName                               xml.Name                                   `xml:"DescribeTransitGatewayRouteTableAnnouncementsResponse"`
	Xmlns                                 string                                     `xml:"xmlns,attr"`
	RequestID                             string                                     `xml:"requestId"`
	NextToken                             string                                     `xml:"nextToken,omitempty"`
	TransitGatewayRouteTableAnnouncements ec2TransitGatewayRouteTableAnnouncementSet `xml:"transitGatewayRouteTableAnnouncements"`
}

type ec2TransitGatewayRouteTableAnnouncementSet struct {
	Items []ec2TransitGatewayRouteTableAnnouncementItem `xml:"item"`
}

type ec2TransitGatewayRouteTableAnnouncementItem struct {
	AnnouncementDirection                  string    `xml:"announcementDirection,omitempty"`
	CoreNetworkID                          string    `xml:"coreNetworkId,omitempty"`
	CreationTime                           time.Time `xml:"creationTime,omitempty"`
	PeerCoreNetworkID                      string    `xml:"peerCoreNetworkId,omitempty"`
	PeerTransitGatewayID                   string    `xml:"peerTransitGatewayId,omitempty"`
	PeeringAttachmentID                    string    `xml:"peeringAttachmentId,omitempty"`
	State                                  string    `xml:"state,omitempty"`
	TagSet                                 ec2TagSet `xml:"tagSet"`
	TransitGatewayID                       string    `xml:"transitGatewayId,omitempty"`
	TransitGatewayRouteTableAnnouncementID string    `xml:"transitGatewayRouteTableAnnouncementId,omitempty"`
	TransitGatewayRouteTableID             string    `xml:"transitGatewayRouteTableId,omitempty"`
}
