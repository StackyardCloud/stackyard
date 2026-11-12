package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage74Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "EnableSerialConsoleAccess":
		enabled := s.ec2.EnableSerialConsoleAccess()
		respondEC2XML(w, ec2SerialConsoleAccessResponse{
			XMLName:                    xml.Name{Local: "EnableSerialConsoleAccessResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			SerialConsoleAccessEnabled: enabled,
		})
		return true
	case "DisableSerialConsoleAccess":
		enabled := s.ec2.DisableSerialConsoleAccess()
		respondEC2XML(w, ec2SerialConsoleAccessResponse{
			XMLName:                    xml.Name{Local: "DisableSerialConsoleAccessResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			SerialConsoleAccessEnabled: enabled,
		})
		return true
	case "GetSerialConsoleAccessStatus":
		out := s.ec2.GetSerialConsoleAccessStatus()
		respondEC2XML(w, ec2GetSerialConsoleAccessStatusResponse{
			XMLName:                    xml.Name{Local: "GetSerialConsoleAccessStatusResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			SerialConsoleAccessEnabled: out.SerialConsoleAccessEnabled,
		})
		return true
	case "EnableIpamOrganizationAdminAccount":
		success, err := s.ec2.EnableIpamOrganizationAdminAccount(strings.TrimSpace(r.Form.Get("DelegatedAdminAccountId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2IpamOrganizationAdminAccountResponse{
			XMLName:   xml.Name{Local: "EnableIpamOrganizationAdminAccountResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Success:   success,
		})
		return true
	case "DisableIpamOrganizationAdminAccount":
		success, err := s.ec2.DisableIpamOrganizationAdminAccount(strings.TrimSpace(r.Form.Get("DelegatedAdminAccountId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2IpamOrganizationAdminAccountResponse{
			XMLName:   xml.Name{Local: "DisableIpamOrganizationAdminAccountResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Success:   success,
		})
		return true
	default:
		return false
	}
}

type ec2SerialConsoleAccessResponse struct {
	XMLName                    xml.Name
	Xmlns                      string `xml:"xmlns,attr"`
	RequestID                  string `xml:"requestId"`
	SerialConsoleAccessEnabled bool   `xml:"serialConsoleAccessEnabled"`
}

type ec2GetSerialConsoleAccessStatusResponse struct {
	XMLName                    xml.Name
	Xmlns                      string `xml:"xmlns,attr"`
	RequestID                  string `xml:"requestId"`
	SerialConsoleAccessEnabled bool   `xml:"serialConsoleAccessEnabled"`
}

type ec2IpamOrganizationAdminAccountResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	Success   bool   `xml:"success"`
}
