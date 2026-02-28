package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage83Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssociateEnclaveCertificateIamRole":
		out, err := s.ec2.AssociateEnclaveCertificateIamRole(
			strings.TrimSpace(r.Form.Get("CertificateArn")),
			strings.TrimSpace(r.Form.Get("RoleArn")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateEnclaveCertificateIamRoleResponse{
			XMLName:                 xml.Name{Local: "AssociateEnclaveCertificateIamRoleResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			CertificateS3BucketName: out.CertificateS3BucketName,
			CertificateS3ObjectKey:  out.CertificateS3ObjectKey,
			EncryptionKmsKeyID:      out.EncryptionKmsKeyID,
		})
		return true
	default:
		return false
	}
}

type ec2AssociateEnclaveCertificateIamRoleResponse struct {
	XMLName                 xml.Name
	Xmlns                   string `xml:"xmlns,attr"`
	RequestID               string `xml:"requestId"`
	CertificateS3BucketName string `xml:"certificateS3BucketName,omitempty"`
	CertificateS3ObjectKey  string `xml:"certificateS3ObjectKey,omitempty"`
	EncryptionKmsKeyID      string `xml:"encryptionKmsKeyId,omitempty"`
}
