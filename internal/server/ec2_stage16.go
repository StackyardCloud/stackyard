package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage16Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreatePlacementGroup":
		group, err := s.ec2.CreatePlacementGroup(
			strings.TrimSpace(r.Form.Get("GroupName")),
			strings.TrimSpace(r.Form.Get("Strategy")),
			parseEC2Int32(r.Form.Get("PartitionCount"), 0),
			strings.TrimSpace(r.Form.Get("SpreadLevel")),
			parseEC2TagSpecificationsForResource(r.Form, "placement-group"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreatePlacementGroupResponse{
			XMLName:        xml.Name{Local: "CreatePlacementGroupResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			PlacementGroup: ec2PlacementGroupItemFrom(group),
		})
		return true
	case "DescribePlacementGroups":
		groups := s.ec2.DescribePlacementGroups(
			parseEC2Members(r.Form, "GroupName."),
			parseEC2Members(r.Form, "GroupId."),
		)
		respondEC2XML(w, ec2DescribePlacementGroupsResponse{
			XMLName:           xml.Name{Local: "DescribePlacementGroupsResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			PlacementGroupSet: ec2PlacementGroupSet{Items: ec2PlacementGroupItems(groups)},
		})
		return true
	case "DeletePlacementGroup":
		if err := s.ec2.DeletePlacementGroup(strings.TrimSpace(r.Form.Get("GroupName"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeletePlacementGroupResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	default:
		return false
	}
}

func ec2PlacementGroupItems(in []ec2svc.PlacementGroup) []ec2PlacementGroupItem {
	out := make([]ec2PlacementGroupItem, 0, len(in))
	for _, group := range in {
		out = append(out, ec2PlacementGroupItemFrom(group))
	}
	return out
}

func ec2PlacementGroupItemFrom(in ec2svc.PlacementGroup) ec2PlacementGroupItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2PlacementGroupItem{
		GroupARN:       in.GroupARN,
		GroupID:        in.GroupID,
		GroupName:      in.GroupName,
		PartitionCount: in.PartitionCount,
		SpreadLevel:    in.SpreadLevel,
		State:          in.State,
		Strategy:       in.Strategy,
		TagSet:         ec2TagSet{Items: tags},
	}
}

type ec2CreatePlacementGroupResponse struct {
	XMLName        xml.Name
	Xmlns          string                `xml:"xmlns,attr"`
	RequestID      string                `xml:"requestId"`
	PlacementGroup ec2PlacementGroupItem `xml:"placementGroup"`
}

type ec2DescribePlacementGroupsResponse struct {
	XMLName           xml.Name             `xml:"DescribePlacementGroupsResponse"`
	Xmlns             string               `xml:"xmlns,attr"`
	RequestID         string               `xml:"requestId"`
	PlacementGroupSet ec2PlacementGroupSet `xml:"placementGroupSet"`
}

type ec2PlacementGroupSet struct {
	Items []ec2PlacementGroupItem `xml:"item"`
}

type ec2PlacementGroupItem struct {
	GroupARN       string    `xml:"groupArn,omitempty"`
	GroupID        string    `xml:"groupId,omitempty"`
	GroupName      string    `xml:"groupName,omitempty"`
	PartitionCount int32     `xml:"partitionCount,omitempty"`
	SpreadLevel    string    `xml:"spreadLevel,omitempty"`
	State          string    `xml:"state,omitempty"`
	Strategy       string    `xml:"strategy,omitempty"`
	TagSet         ec2TagSet `xml:"tagSet,omitempty"`
}
