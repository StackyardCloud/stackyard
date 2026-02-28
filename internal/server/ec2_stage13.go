package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage13Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyIdentityIdFormat":
		useLongIDs, ok := parseEC2OptionalBool(r.Form.Get("UseLongIds"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if err := s.ec2.ModifyIdentityIDFormat(
			strings.TrimSpace(r.Form.Get("PrincipalArn")),
			strings.TrimSpace(r.Form.Get("Resource")),
			useLongIDs,
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyIdentityIdFormatResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	default:
		return false
	}
}
