package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage76Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeAwsNetworkPerformanceMetricSubscriptions":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		subscriptions, nextToken, err := s.ec2.DescribeAwsNetworkPerformanceMetricSubscriptions(
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		response := ec2DescribeAwsNetworkPerformanceMetricSubscriptionsResponse{
			XMLName:   xml.Name{Local: "DescribeAwsNetworkPerformanceMetricSubscriptionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SubscriptionSet: ec2AwsNetworkPerformanceSubscriptionSet{
				Items: ec2AwsNetworkPerformanceSubscriptionItems(subscriptions),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	default:
		return false
	}
}

func ec2AwsNetworkPerformanceSubscriptionItems(in []ec2svc.AwsNetworkPerformanceSubscription) []ec2AwsNetworkPerformanceSubscriptionItem {
	out := make([]ec2AwsNetworkPerformanceSubscriptionItem, 0, len(in))
	for _, subscription := range in {
		out = append(out, ec2AwsNetworkPerformanceSubscriptionItem{
			Destination: subscription.Destination,
			Metric:      subscription.Metric,
			Period:      subscription.Period,
			Source:      subscription.Source,
			Statistic:   subscription.Statistic,
		})
	}
	return out
}

type ec2DescribeAwsNetworkPerformanceMetricSubscriptionsResponse struct {
	XMLName         xml.Name                                `xml:"DescribeAwsNetworkPerformanceMetricSubscriptionsResponse"`
	Xmlns           string                                  `xml:"xmlns,attr"`
	RequestID       string                                  `xml:"requestId"`
	SubscriptionSet ec2AwsNetworkPerformanceSubscriptionSet `xml:"subscriptionSet"`
	NextToken       string                                  `xml:"nextToken,omitempty"`
}

type ec2AwsNetworkPerformanceSubscriptionSet struct {
	Items []ec2AwsNetworkPerformanceSubscriptionItem `xml:"item"`
}

type ec2AwsNetworkPerformanceSubscriptionItem struct {
	Destination string `xml:"destination,omitempty"`
	Metric      string `xml:"metric,omitempty"`
	Period      string `xml:"period,omitempty"`
	Source      string `xml:"source,omitempty"`
	Statistic   string `xml:"statistic,omitempty"`
}
