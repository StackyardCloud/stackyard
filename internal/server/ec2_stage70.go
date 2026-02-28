package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage70Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelDeclarativePoliciesReport":
		ret, err := s.ec2.CancelDeclarativePoliciesReport(strings.TrimSpace(r.Form.Get("ReportId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "CancelDeclarativePoliciesReportResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "DescribeDeclarativePoliciesReports":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		reports, nextToken, err := s.ec2.DescribeDeclarativePoliciesReports(
			parseEC2MembersWithAliases(r.Form, "ReportId", "ReportIds"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeDeclarativePoliciesReportsResponse{
			XMLName:   xml.Name{Local: "DescribeDeclarativePoliciesReportsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ReportSet: ec2DeclarativePoliciesReportSet{
				Items: ec2DeclarativePoliciesReportItemsFrom(reports),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "DisableAllowedImagesSettings":
		state := s.ec2.DisableAllowedImagesSettings()
		respondEC2XML(w, ec2DisableAllowedImagesSettingsResponse{
			XMLName:                    xml.Name{Local: "DisableAllowedImagesSettingsResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			AllowedImagesSettingsState: state,
		})
		return true
	case "EnableAllowedImagesSettings":
		state, err := s.ec2.EnableAllowedImagesSettings(strings.TrimSpace(r.Form.Get("AllowedImagesSettingsState")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2EnableAllowedImagesSettingsResponse{
			XMLName:                    xml.Name{Local: "EnableAllowedImagesSettingsResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			AllowedImagesSettingsState: state,
		})
		return true
	case "GetAllowedImagesSettings":
		settings := s.ec2.GetAllowedImagesSettings()
		respondEC2XML(w, ec2GetAllowedImagesSettingsResponse{
			XMLName:   xml.Name{Local: "GetAllowedImagesSettingsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ImageCriterionSet: ec2AllowedImageCriterionSet{
				Items: ec2AllowedImageCriterionItemsFrom(settings.ImageCriteria),
			},
			ManagedBy: settings.ManagedBy,
			State:     settings.State,
		})
		return true
	case "GetDeclarativePoliciesReportSummary":
		summary, err := s.ec2.GetDeclarativePoliciesReportSummary(strings.TrimSpace(r.Form.Get("ReportId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetDeclarativePoliciesReportSummaryResponse{
			XMLName:   xml.Name{Local: "GetDeclarativePoliciesReportSummaryResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			AttributeSummarySet: ec2DeclarativePoliciesAttributeSummarySet{
				Items: ec2DeclarativePoliciesAttributeSummaryItemsFrom(summary.AttributeSummaries),
			},
			EndTime:                summary.EndTime,
			NumberOfAccounts:       summary.NumberOfAccounts,
			NumberOfFailedAccounts: summary.NumberOfFailedAccounts,
			ReportID:               summary.ReportID,
			S3Bucket:               summary.S3Bucket,
			S3Prefix:               summary.S3Prefix,
			StartTime:              summary.StartTime,
			TargetID:               summary.TargetID,
		})
		return true
	case "ReplaceImageCriteriaInAllowedImagesSettings":
		ret, err := s.ec2.ReplaceImageCriteriaInAllowedImagesSettings(parseEC2AllowedImageCriteriaRequests(r.Form))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ReplaceImageCriteriaInAllowedImagesSettingsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "StartDeclarativePoliciesReport":
		reportID, err := s.ec2.StartDeclarativePoliciesReport(
			strings.TrimSpace(r.Form.Get("TargetId")),
			strings.TrimSpace(r.Form.Get("S3Bucket")),
			parseEC2OptionalString(r.Form.Get("S3Prefix")),
			parseEC2TagSpecificationsForResource(r.Form, "declarative-policies-report"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2StartDeclarativePoliciesReportResponse{
			XMLName:   xml.Name{Local: "StartDeclarativePoliciesReportResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ReportID:  reportID,
		})
		return true
	default:
		return false
	}
}

func parseEC2AllowedImageCriteriaRequests(values url.Values) []ec2svc.AllowedImageCriterion {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, "ImageCriterion.") {
			continue
		}
		rest := strings.TrimPrefix(key, "ImageCriterion.")
		part := rest
		if dot := strings.IndexByte(rest, '.'); dot >= 0 {
			part = rest[:dot]
		}
		index, err := strconv.Atoi(part)
		if err != nil || index <= 0 {
			continue
		}
		indices[index] = struct{}{}
	}

	ordered := make([]int, 0, len(indices))
	for index := range indices {
		ordered = append(ordered, index)
	}
	sort.Ints(ordered)

	out := make([]ec2svc.AllowedImageCriterion, 0, len(ordered))
	for _, index := range ordered {
		base := "ImageCriterion." + strconv.Itoa(index) + ".ImageProvider."
		imageProviders := parseEC2Members(values, base)
		if len(imageProviders) == 0 {
			imageProviders = parseEC2Members(values, "ImageCriterion."+strconv.Itoa(index)+".ImageProviders.")
		}
		if len(imageProviders) == 0 {
			continue
		}
		out = append(out, ec2svc.AllowedImageCriterion{
			ImageProviders: append([]string(nil), imageProviders...),
		})
	}
	return out
}

func ec2AllowedImageCriterionItemsFrom(in []ec2svc.AllowedImageCriterion) []ec2AllowedImageCriterionItem {
	out := make([]ec2AllowedImageCriterionItem, 0, len(in))
	for _, criterion := range in {
		out = append(out, ec2AllowedImageCriterionItem{
			ImageProviderSet: ec2StringSet{Items: append([]string(nil), criterion.ImageProviders...)},
		})
	}
	return out
}

func ec2DeclarativePoliciesReportItemsFrom(in []ec2svc.DeclarativePoliciesReport) []ec2DeclarativePoliciesReportItem {
	out := make([]ec2DeclarativePoliciesReportItem, 0, len(in))
	for _, report := range in {
		out = append(out, ec2DeclarativePoliciesReportItem{
			EndTime:   report.EndTime,
			ReportID:  report.ReportID,
			S3Bucket:  report.S3Bucket,
			S3Prefix:  report.S3Prefix,
			StartTime: report.StartTime,
			Status:    report.Status,
			TagSet:    ec2TagSet{Items: ec2TagItemsFromMap(report.Tags)},
			TargetID:  report.TargetID,
		})
	}
	return out
}

func ec2DeclarativePoliciesAttributeSummaryItemsFrom(in []ec2svc.DeclarativePoliciesAttributeSummary) []ec2DeclarativePoliciesAttributeSummaryItem {
	out := make([]ec2DeclarativePoliciesAttributeSummaryItem, 0, len(in))
	for _, summary := range in {
		out = append(out, ec2DeclarativePoliciesAttributeSummaryItem{
			AttributeName:             summary.AttributeName,
			MostFrequentValue:         summary.MostFrequentValue,
			NumberOfMatchedAccounts:   summary.NumberOfMatchedAccounts,
			NumberOfUnmatchedAccounts: summary.NumberOfUnmatchedAccounts,
			RegionalSummarySet: ec2DeclarativePoliciesRegionalSummarySet{
				Items: ec2DeclarativePoliciesRegionalSummaryItemsFrom(summary.RegionalSummaries),
			},
		})
	}
	return out
}

func ec2DeclarativePoliciesRegionalSummaryItemsFrom(in []ec2svc.DeclarativePoliciesRegionalSummary) []ec2DeclarativePoliciesRegionalSummaryItem {
	out := make([]ec2DeclarativePoliciesRegionalSummaryItem, 0, len(in))
	for _, summary := range in {
		out = append(out, ec2DeclarativePoliciesRegionalSummaryItem{
			NumberOfMatchedAccounts:   summary.NumberOfMatchedAccounts,
			NumberOfUnmatchedAccounts: summary.NumberOfUnmatchedAccounts,
			RegionName:                summary.RegionName,
		})
	}
	return out
}

type ec2DisableAllowedImagesSettingsResponse struct {
	XMLName                    xml.Name `xml:"DisableAllowedImagesSettingsResponse"`
	Xmlns                      string   `xml:"xmlns,attr"`
	RequestID                  string   `xml:"requestId"`
	AllowedImagesSettingsState string   `xml:"allowedImagesSettingsState,omitempty"`
}

type ec2EnableAllowedImagesSettingsResponse struct {
	XMLName                    xml.Name `xml:"EnableAllowedImagesSettingsResponse"`
	Xmlns                      string   `xml:"xmlns,attr"`
	RequestID                  string   `xml:"requestId"`
	AllowedImagesSettingsState string   `xml:"allowedImagesSettingsState,omitempty"`
}

type ec2GetAllowedImagesSettingsResponse struct {
	XMLName           xml.Name                    `xml:"GetAllowedImagesSettingsResponse"`
	Xmlns             string                      `xml:"xmlns,attr"`
	RequestID         string                      `xml:"requestId"`
	ImageCriterionSet ec2AllowedImageCriterionSet `xml:"imageCriterionSet"`
	ManagedBy         string                      `xml:"managedBy,omitempty"`
	State             string                      `xml:"state,omitempty"`
}

type ec2AllowedImageCriterionSet struct {
	Items []ec2AllowedImageCriterionItem `xml:"item"`
}

type ec2AllowedImageCriterionItem struct {
	ImageProviderSet ec2StringSet `xml:"imageProviderSet"`
}

type ec2StartDeclarativePoliciesReportResponse struct {
	XMLName   xml.Name `xml:"StartDeclarativePoliciesReportResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	ReportID  string   `xml:"reportId,omitempty"`
}

type ec2DescribeDeclarativePoliciesReportsResponse struct {
	XMLName   xml.Name                        `xml:"DescribeDeclarativePoliciesReportsResponse"`
	Xmlns     string                          `xml:"xmlns,attr"`
	RequestID string                          `xml:"requestId"`
	NextToken string                          `xml:"nextToken,omitempty"`
	ReportSet ec2DeclarativePoliciesReportSet `xml:"reportSet"`
}

type ec2DeclarativePoliciesReportSet struct {
	Items []ec2DeclarativePoliciesReportItem `xml:"item"`
}

type ec2DeclarativePoliciesReportItem struct {
	EndTime   *time.Time `xml:"endTime,omitempty"`
	ReportID  string     `xml:"reportId,omitempty"`
	S3Bucket  string     `xml:"s3Bucket,omitempty"`
	S3Prefix  string     `xml:"s3Prefix,omitempty"`
	StartTime time.Time  `xml:"startTime,omitempty"`
	Status    string     `xml:"status,omitempty"`
	TagSet    ec2TagSet  `xml:"tagSet"`
	TargetID  string     `xml:"targetId,omitempty"`
}

type ec2GetDeclarativePoliciesReportSummaryResponse struct {
	XMLName                xml.Name                                  `xml:"GetDeclarativePoliciesReportSummaryResponse"`
	Xmlns                  string                                    `xml:"xmlns,attr"`
	RequestID              string                                    `xml:"requestId"`
	AttributeSummarySet    ec2DeclarativePoliciesAttributeSummarySet `xml:"attributeSummarySet"`
	EndTime                *time.Time                                `xml:"endTime,omitempty"`
	NumberOfAccounts       int32                                     `xml:"numberOfAccounts,omitempty"`
	NumberOfFailedAccounts int32                                     `xml:"numberOfFailedAccounts,omitempty"`
	ReportID               string                                    `xml:"reportId,omitempty"`
	S3Bucket               string                                    `xml:"s3Bucket,omitempty"`
	S3Prefix               string                                    `xml:"s3Prefix,omitempty"`
	StartTime              time.Time                                 `xml:"startTime,omitempty"`
	TargetID               string                                    `xml:"targetId,omitempty"`
}

type ec2DeclarativePoliciesAttributeSummarySet struct {
	Items []ec2DeclarativePoliciesAttributeSummaryItem `xml:"item"`
}

type ec2DeclarativePoliciesAttributeSummaryItem struct {
	AttributeName             string                                   `xml:"attributeName,omitempty"`
	MostFrequentValue         string                                   `xml:"mostFrequentValue,omitempty"`
	NumberOfMatchedAccounts   int32                                    `xml:"numberOfMatchedAccounts,omitempty"`
	NumberOfUnmatchedAccounts int32                                    `xml:"numberOfUnmatchedAccounts,omitempty"`
	RegionalSummarySet        ec2DeclarativePoliciesRegionalSummarySet `xml:"regionalSummarySet"`
}

type ec2DeclarativePoliciesRegionalSummarySet struct {
	Items []ec2DeclarativePoliciesRegionalSummaryItem `xml:"item"`
}

type ec2DeclarativePoliciesRegionalSummaryItem struct {
	NumberOfMatchedAccounts   int32  `xml:"numberOfMatchedAccounts,omitempty"`
	NumberOfUnmatchedAccounts int32  `xml:"numberOfUnmatchedAccounts,omitempty"`
	RegionName                string `xml:"regionName,omitempty"`
}
