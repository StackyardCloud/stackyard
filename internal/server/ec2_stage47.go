package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage47Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVpcBlockPublicAccessExclusion":
		exclusion, err := s.ec2.ModifyVpcBlockPublicAccessExclusion(
			strings.TrimSpace(r.Form.Get("ExclusionId")),
			strings.TrimSpace(r.Form.Get("InternetGatewayExclusionMode")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVpcBlockPublicAccessExclusionResponse{
			XMLName:                       xml.Name{Local: "ModifyVpcBlockPublicAccessExclusionResponse"},
			Xmlns:                         ec2Namespace,
			RequestID:                     "stackyard-request",
			VpcBlockPublicAccessExclusion: ec2VpcBlockPublicAccessExclusionItemFrom(exclusion),
		})
		return true
	default:
		return false
	}
}

func ec2VpcBlockPublicAccessExclusionItemFrom(exclusion ec2svc.VpcBlockPublicAccessExclusion) ec2VpcBlockPublicAccessExclusionItem {
	tags := make([]ec2TagItem, 0, len(exclusion.Tags))
	for key, value := range exclusion.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	out := ec2VpcBlockPublicAccessExclusionItem{
		ExclusionID:                  exclusion.ExclusionID,
		InternetGatewayExclusionMode: exclusion.InternetGatewayExclusionMode,
		LastUpdateTimestamp:          exclusion.LastUpdateTimestamp.Format(time.RFC3339),
		ResourceARN:                  exclusion.ResourceARN,
		State:                        exclusion.State,
		Reason:                       exclusion.Reason,
		TagSet:                       ec2TagSet{Items: tags},
	}
	if !exclusion.CreationTimestamp.IsZero() {
		out.CreationTimestamp = exclusion.CreationTimestamp.Format(time.RFC3339)
	}
	if exclusion.DeletionTimestamp != nil {
		out.DeletionTimestamp = exclusion.DeletionTimestamp.Format(time.RFC3339)
	}
	return out
}

type ec2ModifyVpcBlockPublicAccessExclusionResponse struct {
	XMLName                       xml.Name                             `xml:"ModifyVpcBlockPublicAccessExclusionResponse"`
	Xmlns                         string                               `xml:"xmlns,attr"`
	RequestID                     string                               `xml:"requestId"`
	VpcBlockPublicAccessExclusion ec2VpcBlockPublicAccessExclusionItem `xml:"vpcBlockPublicAccessExclusion"`
}

type ec2VpcBlockPublicAccessExclusionItem struct {
	CreationTimestamp            string    `xml:"creationTimestamp,omitempty"`
	DeletionTimestamp            string    `xml:"deletionTimestamp,omitempty"`
	ExclusionID                  string    `xml:"exclusionId,omitempty"`
	InternetGatewayExclusionMode string    `xml:"internetGatewayExclusionMode,omitempty"`
	LastUpdateTimestamp          string    `xml:"lastUpdateTimestamp,omitempty"`
	Reason                       string    `xml:"reason,omitempty"`
	ResourceARN                  string    `xml:"resourceArn,omitempty"`
	State                        string    `xml:"state,omitempty"`
	TagSet                       ec2TagSet `xml:"tagSet"`
}
