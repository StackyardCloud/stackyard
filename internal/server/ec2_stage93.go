package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage93Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelImageLaunchPermission":
		ret, err := s.ec2.CancelImageLaunchPermission(strings.TrimSpace(r.Form.Get("ImageId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "CancelImageLaunchPermissionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}
