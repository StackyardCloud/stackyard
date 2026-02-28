package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage75Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "EnableAwsNetworkPerformanceMetricSubscription":
		out, err := s.ec2.EnableAwsNetworkPerformanceMetricSubscription(
			strings.TrimSpace(r.Form.Get("Source")),
			strings.TrimSpace(r.Form.Get("Destination")),
			strings.TrimSpace(r.Form.Get("Metric")),
			strings.TrimSpace(r.Form.Get("Statistic")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AwsNetworkPerformanceMetricSubscriptionResponse{
			XMLName:   xml.Name{Local: "EnableAwsNetworkPerformanceMetricSubscriptionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Output:    out,
		})
		return true
	case "DisableAwsNetworkPerformanceMetricSubscription":
		out, err := s.ec2.DisableAwsNetworkPerformanceMetricSubscription(
			strings.TrimSpace(r.Form.Get("Source")),
			strings.TrimSpace(r.Form.Get("Destination")),
			strings.TrimSpace(r.Form.Get("Metric")),
			strings.TrimSpace(r.Form.Get("Statistic")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AwsNetworkPerformanceMetricSubscriptionResponse{
			XMLName:   xml.Name{Local: "DisableAwsNetworkPerformanceMetricSubscriptionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Output:    out,
		})
		return true
	case "GetAwsNetworkPerformanceData":
		startTime, err := parseEC2RFC3339Time(r.Form.Get("StartTime"))
		if err != nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		endTime, err := parseEC2RFC3339Time(r.Form.Get("EndTime"))
		if err != nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		responses, nextToken, err := s.ec2.GetAwsNetworkPerformanceData(
			parseEC2AwsNetworkPerformanceDataQueries(r.Form),
			startTime,
			endTime,
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		response := ec2GetAwsNetworkPerformanceDataResponse{
			XMLName:   xml.Name{Local: "GetAwsNetworkPerformanceDataResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			DataResponseSet: ec2AwsNetworkPerformanceDataResponseSet{
				Items: ec2AwsNetworkPerformanceDataResponseItems(responses),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "EnableReachabilityAnalyzerOrganizationSharing":
		ret := s.ec2.EnableReachabilityAnalyzerOrganizationSharing()
		respondEC2XML(w, ec2EnableReachabilityAnalyzerOrganizationSharingResponse{
			XMLName:     xml.Name{Local: "EnableReachabilityAnalyzerOrganizationSharingResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			ReturnValue: ret,
		})
		return true
	case "EnableVolumeIO":
		if err := s.ec2.EnableVolumeIO(strings.TrimSpace(r.Form.Get("VolumeId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2EnableVolumeIOResponse{
			XMLName:   xml.Name{Local: "EnableVolumeIOResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	default:
		return false
	}
}

func parseEC2AwsNetworkPerformanceDataQueries(values map[string][]string) []ec2svc.AwsNetworkPerformanceDataQuery {
	byIndex := map[int]*ec2svc.AwsNetworkPerformanceDataQuery{}
	for key := range values {
		if !strings.HasPrefix(key, "DataQuery.") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(key, "DataQuery."), ".", 2)
		if len(parts) != 2 {
			continue
		}
		index, err := strconv.Atoi(parts[0])
		if err != nil || index <= 0 {
			continue
		}
		fieldName := parts[1]
		value := strings.TrimSpace(firstFormValue(values, key))

		query := byIndex[index]
		if query == nil {
			query = &ec2svc.AwsNetworkPerformanceDataQuery{}
			byIndex[index] = query
		}

		switch fieldName {
		case "Id":
			query.ID = value
		case "Source":
			query.Source = value
		case "Destination":
			query.Destination = value
		case "Metric":
			query.Metric = value
		case "Period":
			query.Period = value
		case "Statistic":
			query.Statistic = value
		}
	}

	indexes := make([]int, 0, len(byIndex))
	for index := range byIndex {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	out := make([]ec2svc.AwsNetworkPerformanceDataQuery, 0, len(indexes))
	for _, index := range indexes {
		if byIndex[index] == nil {
			continue
		}
		out = append(out, *byIndex[index])
	}
	return out
}

func ec2AwsNetworkPerformanceDataResponseItems(in []ec2svc.AwsNetworkPerformanceDataResponse) []ec2AwsNetworkPerformanceDataResponseItem {
	out := make([]ec2AwsNetworkPerformanceDataResponseItem, 0, len(in))
	for _, response := range in {
		out = append(out, ec2AwsNetworkPerformanceDataResponseItem{
			Destination: response.Destination,
			ID:          response.ID,
			Metric:      response.Metric,
			MetricPointSet: ec2AwsNetworkPerformanceMetricPointSet{
				Items: ec2AwsNetworkPerformanceMetricPointItems(response.MetricPoints),
			},
			Period:    response.Period,
			Source:    response.Source,
			Statistic: response.Statistic,
		})
	}
	return out
}

func ec2AwsNetworkPerformanceMetricPointItems(in []ec2svc.AwsNetworkPerformanceMetricPoint) []ec2AwsNetworkPerformanceMetricPointItem {
	out := make([]ec2AwsNetworkPerformanceMetricPointItem, 0, len(in))
	for _, point := range in {
		out = append(out, ec2AwsNetworkPerformanceMetricPointItem{
			EndDate:   point.EndDate.UTC().Format(time.RFC3339),
			StartDate: point.StartDate.UTC().Format(time.RFC3339),
			Status:    point.Status,
			Value:     point.Value,
		})
	}
	return out
}

type ec2AwsNetworkPerformanceMetricSubscriptionResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	Output    bool   `xml:"output"`
}

type ec2GetAwsNetworkPerformanceDataResponse struct {
	XMLName         xml.Name                                `xml:"GetAwsNetworkPerformanceDataResponse"`
	Xmlns           string                                  `xml:"xmlns,attr"`
	RequestID       string                                  `xml:"requestId"`
	DataResponseSet ec2AwsNetworkPerformanceDataResponseSet `xml:"dataResponseSet"`
	NextToken       string                                  `xml:"nextToken,omitempty"`
}

type ec2AwsNetworkPerformanceDataResponseSet struct {
	Items []ec2AwsNetworkPerformanceDataResponseItem `xml:"item"`
}

type ec2AwsNetworkPerformanceDataResponseItem struct {
	Destination    string                                 `xml:"destination,omitempty"`
	ID             string                                 `xml:"id,omitempty"`
	Metric         string                                 `xml:"metric,omitempty"`
	MetricPointSet ec2AwsNetworkPerformanceMetricPointSet `xml:"metricPointSet"`
	Period         string                                 `xml:"period,omitempty"`
	Source         string                                 `xml:"source,omitempty"`
	Statistic      string                                 `xml:"statistic,omitempty"`
}

type ec2AwsNetworkPerformanceMetricPointSet struct {
	Items []ec2AwsNetworkPerformanceMetricPointItem `xml:"item"`
}

type ec2AwsNetworkPerformanceMetricPointItem struct {
	EndDate   string  `xml:"endDate,omitempty"`
	StartDate string  `xml:"startDate,omitempty"`
	Status    string  `xml:"status,omitempty"`
	Value     float32 `xml:"value,omitempty"`
}

type ec2EnableReachabilityAnalyzerOrganizationSharingResponse struct {
	XMLName     xml.Name `xml:"EnableReachabilityAnalyzerOrganizationSharingResponse"`
	Xmlns       string   `xml:"xmlns,attr"`
	RequestID   string   `xml:"requestId"`
	ReturnValue bool     `xml:"returnValue"`
}

type ec2EnableVolumeIOResponse struct {
	XMLName   xml.Name `xml:"EnableVolumeIOResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
}
