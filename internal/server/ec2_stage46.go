package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage46Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVpcBlockPublicAccessOptions":
		options, err := s.ec2.ModifyVpcBlockPublicAccessOptions(strings.TrimSpace(r.Form.Get("InternetGatewayBlockMode")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVpcBlockPublicAccessOptionsResponse{
			XMLName:                     xml.Name{Local: "ModifyVpcBlockPublicAccessOptionsResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			VpcBlockPublicAccessOptions: ec2VpcBlockPublicAccessOptionsItemFrom(options),
		})
		return true
	default:
		return false
	}
}

func ec2VpcBlockPublicAccessOptionsItemFrom(options ec2svc.VpcBlockPublicAccessOptions) ec2VpcBlockPublicAccessOptionsItem {
	return ec2VpcBlockPublicAccessOptionsItem{
		AwsAccountID:             options.AwsAccountID,
		AwsRegion:                options.AwsRegion,
		ExclusionsAllowed:        options.ExclusionsAllowed,
		InternetGatewayBlockMode: options.InternetGatewayBlockMode,
		LastUpdateTimestamp:      options.LastUpdateTimestamp.Format(time.RFC3339),
		ManagedBy:                options.ManagedBy,
		Reason:                   options.Reason,
		State:                    options.State,
	}
}

type ec2ModifyVpcBlockPublicAccessOptionsResponse struct {
	XMLName                     xml.Name                           `xml:"ModifyVpcBlockPublicAccessOptionsResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	VpcBlockPublicAccessOptions ec2VpcBlockPublicAccessOptionsItem `xml:"vpcBlockPublicAccessOptions"`
}

type ec2VpcBlockPublicAccessOptionsItem struct {
	AwsAccountID             string `xml:"awsAccountId,omitempty"`
	AwsRegion                string `xml:"awsRegion,omitempty"`
	ExclusionsAllowed        string `xml:"exclusionsAllowed,omitempty"`
	InternetGatewayBlockMode string `xml:"internetGatewayBlockMode,omitempty"`
	LastUpdateTimestamp      string `xml:"lastUpdateTimestamp,omitempty"`
	ManagedBy                string `xml:"managedBy,omitempty"`
	Reason                   string `xml:"reason,omitempty"`
	State                    string `xml:"state,omitempty"`
}
