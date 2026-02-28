package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage57Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DeleteVpcEndpointConnectionNotifications":
		unsuccessful, err := s.ec2.DeleteVpcEndpointConnectionNotifications(
			parseEC2MembersWithAliases(r.Form, "ConnectionNotificationId", "ConnectionNotificationIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteVpcEndpointConnectionNotificationsResponse{
			XMLName:      xml.Name{Local: "DeleteVpcEndpointConnectionNotificationsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Unsuccessful: ec2UnsuccessfulItemSet{Items: ec2UnsuccessfulItems(unsuccessful)},
		})
		return true
	default:
		return false
	}
}

func ec2UnsuccessfulItems(in []ec2svc.UnsuccessfulItem) []ec2UnsuccessfulItem {
	out := make([]ec2UnsuccessfulItem, 0, len(in))
	for _, item := range in {
		err := &ec2UnsuccessfulItemError{
			Code:    item.Code,
			Message: item.Message,
		}
		if err.Code == "" && err.Message == "" {
			err = nil
		}
		out = append(out, ec2UnsuccessfulItem{
			ResourceID: item.ResourceID,
			Error:      err,
		})
	}
	return out
}

type ec2DeleteVpcEndpointConnectionNotificationsResponse struct {
	XMLName      xml.Name               `xml:"DeleteVpcEndpointConnectionNotificationsResponse"`
	Xmlns        string                 `xml:"xmlns,attr"`
	RequestID    string                 `xml:"requestId"`
	Unsuccessful ec2UnsuccessfulItemSet `xml:"unsuccessful"`
}

type ec2UnsuccessfulItemSet struct {
	Items []ec2UnsuccessfulItem `xml:"item"`
}

type ec2UnsuccessfulItem struct {
	ResourceID string                    `xml:"resourceId,omitempty"`
	Error      *ec2UnsuccessfulItemError `xml:"error,omitempty"`
}

type ec2UnsuccessfulItemError struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}
