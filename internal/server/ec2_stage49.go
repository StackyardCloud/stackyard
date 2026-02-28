package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage49Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVpcEndpointServicePermissions":
		addedPrincipals, ret, err := s.ec2.ModifyVpcEndpointServicePermissions(
			strings.TrimSpace(r.Form.Get("ServiceId")),
			parseEC2MembersOrItemList(r.Form, "AddAllowedPrincipals"),
			parseEC2MembersOrItemList(r.Form, "RemoveAllowedPrincipals"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVpcEndpointServicePermissionsResponse{
			XMLName:           xml.Name{Local: "ModifyVpcEndpointServicePermissionsResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			Return:            ret,
			AddedPrincipalSet: ec2AddedPrincipalSet{Items: ec2AddedPrincipalItems(addedPrincipals)},
		})
		return true
	default:
		return false
	}
}

func ec2AddedPrincipalItems(in []ec2svc.VpcEndpointServiceAddedPrincipal) []ec2AddedPrincipalItem {
	out := make([]ec2AddedPrincipalItem, 0, len(in))
	for _, principal := range in {
		out = append(out, ec2AddedPrincipalItem{
			Principal:           principal.Principal,
			PrincipalType:       principal.PrincipalType,
			ServiceID:           principal.ServiceID,
			ServicePermissionID: principal.ServicePermissionID,
		})
	}
	return out
}

type ec2ModifyVpcEndpointServicePermissionsResponse struct {
	XMLName           xml.Name             `xml:"ModifyVpcEndpointServicePermissionsResponse"`
	Xmlns             string               `xml:"xmlns,attr"`
	RequestID         string               `xml:"requestId"`
	Return            bool                 `xml:"return"`
	AddedPrincipalSet ec2AddedPrincipalSet `xml:"addedPrincipalSet"`
}

type ec2AddedPrincipalSet struct {
	Items []ec2AddedPrincipalItem `xml:"item"`
}

type ec2AddedPrincipalItem struct {
	Principal           string `xml:"principal,omitempty"`
	PrincipalType       string `xml:"principalType,omitempty"`
	ServiceID           string `xml:"serviceId,omitempty"`
	ServicePermissionID string `xml:"servicePermissionId,omitempty"`
}
