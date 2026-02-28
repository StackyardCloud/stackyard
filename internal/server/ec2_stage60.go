package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage60Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeVpcEndpointConnectionNotifications":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		notifications, nextToken, err := s.ec2.DescribeVpcEndpointConnectionNotifications(
			parseEC2OptionalString(r.Form.Get("ConnectionNotificationId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVpcEndpointConnectionNotificationsResponse{
			XMLName:   xml.Name{Local: "DescribeVpcEndpointConnectionNotificationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ConnectionNotificationSet: ec2ConnectionNotificationSet{
				Items: ec2ConnectionNotificationItems(notifications),
			},
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

func ec2ConnectionNotificationItems(in []ec2svc.VpcEndpointConnectionNotification) []ec2ConnectionNotificationItem {
	out := make([]ec2ConnectionNotificationItem, 0, len(in))
	for _, notification := range in {
		out = append(out, ec2ConnectionNotificationItemFrom(notification))
	}
	return out
}

type ec2DescribeVpcEndpointConnectionNotificationsResponse struct {
	XMLName                   xml.Name                     `xml:"DescribeVpcEndpointConnectionNotificationsResponse"`
	Xmlns                     string                       `xml:"xmlns,attr"`
	RequestID                 string                       `xml:"requestId"`
	NextToken                 string                       `xml:"nextToken,omitempty"`
	ConnectionNotificationSet ec2ConnectionNotificationSet `xml:"connectionNotificationSet"`
}

type ec2ConnectionNotificationSet struct {
	Items []ec2ConnectionNotificationItem `xml:"item"`
}
