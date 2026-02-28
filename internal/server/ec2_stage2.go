package server

import (
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage2Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AuthorizeSecurityGroupEgress":
		if err := s.ec2.AuthorizeSecurityGroupEgress(
			strings.TrimSpace(r.Form.Get("GroupId")),
			strings.TrimSpace(r.Form.Get("GroupName")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2IPPermissions(r.Form, "IpPermissions."),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "AuthorizeSecurityGroupEgressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "RevokeSecurityGroupEgress":
		if err := s.ec2.RevokeSecurityGroupEgress(
			strings.TrimSpace(r.Form.Get("GroupId")),
			strings.TrimSpace(r.Form.Get("GroupName")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2IPPermissions(r.Form, "IpPermissions."),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "RevokeSecurityGroupEgressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateKeyPair":
		keyPair, err := s.ec2.CreateKeyPair(strings.TrimSpace(r.Form.Get("KeyName")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateKeyPairResponse{
			XMLName:        xml.Name{Local: "CreateKeyPairResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			ec2KeyPairItem: ec2KeyPairItemFrom(keyPair),
		})
		return true
	case "ImportKeyPair":
		keyPair, err := s.ec2.ImportKeyPair(
			strings.TrimSpace(r.Form.Get("KeyName")),
			decodeEC2PublicKeyMaterial(strings.TrimSpace(r.Form.Get("PublicKeyMaterial"))),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ImportKeyPairResponse{
			XMLName:        xml.Name{Local: "ImportKeyPairResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			KeyName:        keyPair.Name,
			KeyFingerprint: keyPair.Fingerprint,
			KeyPairID:      keyPair.ID,
		})
		return true
	case "DescribeKeyPairs":
		keyNames := parseEC2Members(r.Form, "KeyName.")
		if len(keyNames) == 0 {
			keyNames = parseEC2FilterValues(r.Form, "key-name")
		}
		keyPairIDs := parseEC2Members(r.Form, "KeyPairId.")
		if len(keyPairIDs) == 0 {
			keyPairIDs = parseEC2FilterValues(r.Form, "key-pair-id")
		}
		keyPairs := s.ec2.DescribeKeyPairs(keyNames, keyPairIDs)
		respondEC2XML(w, ec2DescribeKeyPairsResponse{
			XMLName:   xml.Name{Local: "DescribeKeyPairsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			KeySet:    ec2KeyPairSet{Items: ec2KeyPairItems(keyPairs)},
		})
		return true
	case "DeleteKeyPair":
		if err := s.ec2.DeleteKeyPair(strings.TrimSpace(r.Form.Get("KeyName")), strings.TrimSpace(r.Form.Get("KeyPairId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteKeyPairResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "AssociateIamInstanceProfile":
		association, err := s.ec2.AssociateIamInstanceProfile(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("IamInstanceProfile.Name")),
			strings.TrimSpace(r.Form.Get("IamInstanceProfile.Arn")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateIamInstanceProfileResponse{
			XMLName:     xml.Name{Local: "AssociateIamInstanceProfileResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Association: ec2IamInstanceProfileAssociationItemFrom(association),
		})
		return true
	case "DescribeIamInstanceProfileAssociations":
		associationIDs := parseEC2Members(r.Form, "AssociationId.")
		if len(associationIDs) == 0 {
			associationIDs = parseEC2FilterValues(r.Form, "association-id")
		}
		instanceIDs := parseEC2FilterValues(r.Form, "instance-id")
		associations := s.ec2.DescribeIamInstanceProfileAssociations(associationIDs, instanceIDs)
		respondEC2XML(w, ec2DescribeIamInstanceProfileAssociationsResponse{
			XMLName:        xml.Name{Local: "DescribeIamInstanceProfileAssociationsResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			AssociationSet: ec2IamInstanceProfileAssociationSet{Items: ec2IamInstanceProfileAssociationItems(associations)},
		})
		return true
	case "DisassociateIamInstanceProfile":
		association, err := s.ec2.DisassociateIamInstanceProfile(strings.TrimSpace(r.Form.Get("AssociationId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisassociateIamInstanceProfileResponse{
			XMLName:     xml.Name{Local: "DisassociateIamInstanceProfileResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Association: ec2IamInstanceProfileAssociationItemFrom(association),
		})
		return true
	case "ReplaceIamInstanceProfileAssociation":
		association, err := s.ec2.ReplaceIamInstanceProfileAssociation(
			strings.TrimSpace(r.Form.Get("AssociationId")),
			strings.TrimSpace(r.Form.Get("IamInstanceProfile.Name")),
			strings.TrimSpace(r.Form.Get("IamInstanceProfile.Arn")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ReplaceIamInstanceProfileAssociationResponse{
			XMLName:     xml.Name{Local: "ReplaceIamInstanceProfileAssociationResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Association: ec2IamInstanceProfileAssociationItemFrom(association),
		})
		return true
	default:
		return false
	}
}

func decodeEC2PublicKeyMaterial(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return string(decoded)
	}
	return value
}

func ec2KeyPairItems(in []ec2svc.KeyPair) []ec2KeyPairItem {
	out := make([]ec2KeyPairItem, 0, len(in))
	for _, keyPair := range in {
		out = append(out, ec2KeyPairItemFrom(keyPair))
	}
	return out
}

func ec2KeyPairItemFrom(keyPair ec2svc.KeyPair) ec2KeyPairItem {
	return ec2KeyPairItem{
		KeyName:        keyPair.Name,
		KeyPairID:      keyPair.ID,
		KeyFingerprint: keyPair.Fingerprint,
		KeyMaterial:    keyPair.Material,
		KeyType:        keyPair.Type,
	}
}

func ec2IamInstanceProfileAssociationItems(in []ec2svc.IamInstanceProfileAssociation) []ec2IamInstanceProfileAssociationItem {
	out := make([]ec2IamInstanceProfileAssociationItem, 0, len(in))
	for _, association := range in {
		out = append(out, ec2IamInstanceProfileAssociationItemFrom(association))
	}
	return out
}

func ec2IamInstanceProfileAssociationItemFrom(association ec2svc.IamInstanceProfileAssociation) ec2IamInstanceProfileAssociationItem {
	return ec2IamInstanceProfileAssociationItem{
		AssociationID: association.AssociationID,
		InstanceID:    association.InstanceID,
		State:         association.State,
		Timestamp:     association.Timestamp.Format(time.RFC3339),
		IamInstanceProfile: ec2IamInstanceProfile{
			Arn: association.ProfileARN,
			ID:  association.ProfileName,
		},
	}
}

type ec2CreateKeyPairResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	ec2KeyPairItem
}

type ec2ImportKeyPairResponse struct {
	XMLName        xml.Name
	Xmlns          string `xml:"xmlns,attr"`
	RequestID      string `xml:"requestId"`
	KeyName        string `xml:"keyName"`
	KeyPairID      string `xml:"keyPairId,omitempty"`
	KeyFingerprint string `xml:"keyFingerprint"`
}

type ec2DescribeKeyPairsResponse struct {
	XMLName   xml.Name
	Xmlns     string        `xml:"xmlns,attr"`
	RequestID string        `xml:"requestId"`
	KeySet    ec2KeyPairSet `xml:"keySet"`
}

type ec2KeyPairSet struct {
	Items []ec2KeyPairItem `xml:"item"`
}

type ec2KeyPairItem struct {
	KeyName        string `xml:"keyName"`
	KeyPairID      string `xml:"keyPairId,omitempty"`
	KeyFingerprint string `xml:"keyFingerprint,omitempty"`
	KeyMaterial    string `xml:"keyMaterial,omitempty"`
	KeyType        string `xml:"keyType,omitempty"`
}

type ec2AssociateIamInstanceProfileResponse struct {
	XMLName     xml.Name
	Xmlns       string                               `xml:"xmlns,attr"`
	RequestID   string                               `xml:"requestId"`
	Association ec2IamInstanceProfileAssociationItem `xml:"iamInstanceProfileAssociation"`
}

type ec2DescribeIamInstanceProfileAssociationsResponse struct {
	XMLName        xml.Name
	Xmlns          string                              `xml:"xmlns,attr"`
	RequestID      string                              `xml:"requestId"`
	AssociationSet ec2IamInstanceProfileAssociationSet `xml:"iamInstanceProfileAssociationSet"`
}

type ec2DisassociateIamInstanceProfileResponse struct {
	XMLName     xml.Name
	Xmlns       string                               `xml:"xmlns,attr"`
	RequestID   string                               `xml:"requestId"`
	Association ec2IamInstanceProfileAssociationItem `xml:"iamInstanceProfileAssociation"`
}

type ec2ReplaceIamInstanceProfileAssociationResponse struct {
	XMLName     xml.Name
	Xmlns       string                               `xml:"xmlns,attr"`
	RequestID   string                               `xml:"requestId"`
	Association ec2IamInstanceProfileAssociationItem `xml:"iamInstanceProfileAssociation"`
}

type ec2IamInstanceProfileAssociationSet struct {
	Items []ec2IamInstanceProfileAssociationItem `xml:"item"`
}

type ec2IamInstanceProfileAssociationItem struct {
	AssociationID      string                `xml:"associationId"`
	InstanceID         string                `xml:"instanceId"`
	IamInstanceProfile ec2IamInstanceProfile `xml:"iamInstanceProfile"`
	State              string                `xml:"state"`
	Timestamp          string                `xml:"timestamp"`
}

type ec2IamInstanceProfile struct {
	Arn string `xml:"arn,omitempty"`
	ID  string `xml:"id,omitempty"`
}
