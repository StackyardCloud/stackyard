package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage50Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVpcEndpointConnectionNotification":
		ret, err := s.ec2.ModifyVpcEndpointConnectionNotification(
			strings.TrimSpace(r.Form.Get("ConnectionNotificationId")),
			parseEC2OptionalString(r.Form.Get("ConnectionNotificationArn")),
			parseEC2MembersOrItemList(r.Form, "ConnectionEvents"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyVpcEndpointConnectionNotificationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}
