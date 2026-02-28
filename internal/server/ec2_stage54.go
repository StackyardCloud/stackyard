package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage54Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateVpcEndpointConnectionNotification":
		notification, clientToken, err := s.ec2.CreateVpcEndpointConnectionNotification(
			strings.TrimSpace(r.Form.Get("ServiceId")),
			strings.TrimSpace(r.Form.Get("VpcEndpointId")),
			strings.TrimSpace(r.Form.Get("ConnectionNotificationArn")),
			parseEC2MembersOrItemList(r.Form, "ConnectionEvents"),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVpcEndpointConnectionNotificationResponse{
			XMLName:                xml.Name{Local: "CreateVpcEndpointConnectionNotificationResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			ClientToken:            clientToken,
			ConnectionNotification: ec2ConnectionNotificationItemFrom(notification),
		})
		return true
	default:
		return false
	}
}

func ec2ConnectionNotificationItemFrom(in ec2svc.VpcEndpointConnectionNotification) ec2ConnectionNotificationItem {
	return ec2ConnectionNotificationItem{
		ConnectionEvents:            append([]string(nil), in.ConnectionEvents...),
		ConnectionNotificationARN:   in.ConnectionNotificationARN,
		ConnectionNotificationID:    in.ConnectionNotificationID,
		ConnectionNotificationState: in.ConnectionNotificationState,
		ConnectionNotificationType:  in.ConnectionNotificationType,
		ServiceID:                   in.ServiceID,
		ServiceRegion:               in.ServiceRegion,
		VpcEndpointID:               in.VpcEndpointID,
	}
}

type ec2CreateVpcEndpointConnectionNotificationResponse struct {
	XMLName                xml.Name                      `xml:"CreateVpcEndpointConnectionNotificationResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	ClientToken            *string                       `xml:"clientToken,omitempty"`
	ConnectionNotification ec2ConnectionNotificationItem `xml:"connectionNotification"`
}

type ec2ConnectionNotificationItem struct {
	ConnectionEvents            []string `xml:"connectionEvents>item"`
	ConnectionNotificationARN   string   `xml:"connectionNotificationArn,omitempty"`
	ConnectionNotificationID    string   `xml:"connectionNotificationId,omitempty"`
	ConnectionNotificationState string   `xml:"connectionNotificationState,omitempty"`
	ConnectionNotificationType  string   `xml:"connectionNotificationType,omitempty"`
	ServiceID                   string   `xml:"serviceId,omitempty"`
	ServiceRegion               string   `xml:"serviceRegion,omitempty"`
	VpcEndpointID               string   `xml:"vpcEndpointId,omitempty"`
}
