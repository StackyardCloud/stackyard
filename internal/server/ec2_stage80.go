package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage80Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AllocateHosts":
		quantity, ok := parseEC2OptionalInt32(r.Form.Get("Quantity"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		hostIDs, err := s.ec2.AllocateHosts(
			parseEC2MembersWithAliases(r.Form, "AssetId", "AssetIds"),
			quantity,
			parseEC2OptionalString(r.Form.Get("AvailabilityZone")),
			parseEC2OptionalString(r.Form.Get("AvailabilityZoneId")),
			parseEC2OptionalString(r.Form.Get("InstanceType")),
			parseEC2OptionalString(r.Form.Get("InstanceFamily")),
			parseEC2OptionalString(r.Form.Get("AutoPlacement")),
			parseEC2OptionalString(r.Form.Get("HostRecovery")),
			parseEC2OptionalString(r.Form.Get("HostMaintenance")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2AllocateHostsResponse{
			XMLName:   xml.Name{Local: "AllocateHostsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			HostIDSet: ec2HostIDSet{Items: append([]string(nil), hostIDs...)},
		})
		return true
	default:
		return false
	}
}

type ec2AllocateHostsResponse struct {
	XMLName   xml.Name     `xml:"AllocateHostsResponse"`
	Xmlns     string       `xml:"xmlns,attr"`
	RequestID string       `xml:"requestId"`
	HostIDSet ec2HostIDSet `xml:"hostIdSet"`
}

type ec2HostIDSet struct {
	Items []string `xml:"item"`
}
