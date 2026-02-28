package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"
)

func (s *Server) handleEC2Stage84Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssociateInstanceEventWindow":
		association, err := s.ec2.AssociateInstanceEventWindow(
			strings.TrimSpace(r.Form.Get("InstanceEventWindowId")),
			parseEC2MembersOrItemList(r.Form, "AssociationTarget.DedicatedHostId"),
			parseEC2MembersOrItemList(r.Form, "AssociationTarget.InstanceId"),
			parseEC2Tags(r.Form, "AssociationTarget.InstanceTag."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		response := ec2AssociateInstanceEventWindowResponse{
			XMLName:   xml.Name{Local: "AssociateInstanceEventWindowResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			InstanceEventWindow: ec2InstanceEventWindowItem{
				InstanceEventWindowID: association.InstanceEventWindowID,
				State:                 association.State,
			},
		}
		target := ec2InstanceEventWindowAssociationTargetItem{}
		if len(association.DedicatedHostIDs) > 0 {
			target.DedicatedHostIDSet = &ec2StringSet{Items: append([]string(nil), association.DedicatedHostIDs...)}
		}
		if len(association.InstanceIDs) > 0 {
			target.InstanceIDSet = &ec2StringSet{Items: append([]string(nil), association.InstanceIDs...)}
		}
		if len(association.InstanceTags) > 0 {
			tags := make([]ec2TagItem, 0, len(association.InstanceTags))
			for _, tag := range association.InstanceTags {
				tags = append(tags, ec2TagItem{Key: tag.Key, Value: tag.Value})
			}
			sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
			target.TagSet = &ec2TagSet{Items: tags}
		}
		if target.DedicatedHostIDSet != nil || target.InstanceIDSet != nil || target.TagSet != nil {
			response.InstanceEventWindow.AssociationTarget = &target
		}

		respondEC2XML(w, response)
		return true
	default:
		return false
	}
}

type ec2AssociateInstanceEventWindowResponse struct {
	XMLName             xml.Name                   `xml:"AssociateInstanceEventWindowResponse"`
	Xmlns               string                     `xml:"xmlns,attr"`
	RequestID           string                     `xml:"requestId"`
	InstanceEventWindow ec2InstanceEventWindowItem `xml:"instanceEventWindow"`
}

type ec2InstanceEventWindowItem struct {
	AssociationTarget     *ec2InstanceEventWindowAssociationTargetItem `xml:"associationTarget,omitempty"`
	InstanceEventWindowID string                                       `xml:"instanceEventWindowId,omitempty"`
	State                 string                                       `xml:"state,omitempty"`
}

type ec2InstanceEventWindowAssociationTargetItem struct {
	DedicatedHostIDSet *ec2StringSet `xml:"dedicatedHostIdSet,omitempty"`
	InstanceIDSet      *ec2StringSet `xml:"instanceIdSet,omitempty"`
	TagSet             *ec2TagSet    `xml:"tagSet,omitempty"`
}
