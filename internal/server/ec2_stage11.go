package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage11Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "EnableEbsEncryptionByDefault":
		respondEC2XML(w, ec2EbsEncryptionByDefaultResponse{
			XMLName:                xml.Name{Local: "EnableEbsEncryptionByDefaultResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			EbsEncryptionByDefault: s.ec2.EnableEbsEncryptionByDefault(),
		})
		return true
	case "DisableEbsEncryptionByDefault":
		respondEC2XML(w, ec2EbsEncryptionByDefaultResponse{
			XMLName:                xml.Name{Local: "DisableEbsEncryptionByDefaultResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			EbsEncryptionByDefault: s.ec2.DisableEbsEncryptionByDefault(),
		})
		return true
	case "GetEbsEncryptionByDefault":
		respondEC2XML(w, ec2EbsEncryptionByDefaultResponse{
			XMLName:                xml.Name{Local: "GetEbsEncryptionByDefaultResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			EbsEncryptionByDefault: s.ec2.GetEbsEncryptionByDefault(),
		})
		return true
	case "GetEbsDefaultKmsKeyId":
		respondEC2XML(w, ec2EbsDefaultKmsKeyIDResponse{
			XMLName:   xml.Name{Local: "GetEbsDefaultKmsKeyIdResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			KmsKeyID:  s.ec2.GetEbsDefaultKmsKeyID(),
		})
		return true
	case "ModifyEbsDefaultKmsKeyId":
		kmsKeyID, err := s.ec2.ModifyEbsDefaultKmsKeyID(strings.TrimSpace(r.Form.Get("KmsKeyId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2EbsDefaultKmsKeyIDResponse{
			XMLName:   xml.Name{Local: "ModifyEbsDefaultKmsKeyIdResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			KmsKeyID:  kmsKeyID,
		})
		return true
	case "ResetEbsDefaultKmsKeyId":
		respondEC2XML(w, ec2EbsDefaultKmsKeyIDResponse{
			XMLName:   xml.Name{Local: "ResetEbsDefaultKmsKeyIdResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			KmsKeyID:  s.ec2.ResetEbsDefaultKmsKeyID(),
		})
		return true
	default:
		return false
	}
}

type ec2EbsEncryptionByDefaultResponse struct {
	XMLName                xml.Name
	Xmlns                  string `xml:"xmlns,attr"`
	RequestID              string `xml:"requestId"`
	EbsEncryptionByDefault bool   `xml:"ebsEncryptionByDefault"`
}

type ec2EbsDefaultKmsKeyIDResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	KmsKeyID  string `xml:"kmsKeyId,omitempty"`
}
