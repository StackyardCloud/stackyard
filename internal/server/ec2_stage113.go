package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage113Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DeleteNetworkInsightsAnalysis":
		networkInsightsAnalysisID, err := s.ec2.DeleteNetworkInsightsAnalysis(strings.TrimSpace(r.Form.Get("NetworkInsightsAnalysisId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeleteNetworkInsightsAnalysisResponse{
			XMLName:                   xml.Name{Local: "DeleteNetworkInsightsAnalysisResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			NetworkInsightsAnalysisID: networkInsightsAnalysisID,
		})
		return true
	case "DeleteNetworkInsightsPath":
		networkInsightsPathID, err := s.ec2.DeleteNetworkInsightsPath(strings.TrimSpace(r.Form.Get("NetworkInsightsPathId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeleteNetworkInsightsPathResponse{
			XMLName:               xml.Name{Local: "DeleteNetworkInsightsPathResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			NetworkInsightsPathID: networkInsightsPathID,
		})
		return true
	case "DeletePublicIpv4Pool":
		returnValue, err := s.ec2.DeletePublicIpv4Pool(
			strings.TrimSpace(r.Form.Get("PoolId")),
			parseEC2OptionalString(r.Form.Get("NetworkBorderGroup")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeletePublicIpv4PoolResponse{
			XMLName:     xml.Name{Local: "DeletePublicIpv4PoolResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			ReturnValue: returnValue,
		})
		return true
	case "DeleteQueuedReservedInstances":
		failed, successful, err := s.ec2.DeleteQueuedReservedInstances(parseEC2MembersOrItemList(r.Form, "ReservedInstancesId"))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeleteQueuedReservedInstancesResponse{
			XMLName:                             xml.Name{Local: "DeleteQueuedReservedInstancesResponse"},
			Xmlns:                               ec2Namespace,
			RequestID:                           "stackyard-request",
			FailedQueuedPurchaseDeletionSet:     ec2Stage113FailedQueuedPurchaseDeletionSet{Items: ec2Stage113FailedQueuedPurchaseDeletionItemsFrom(failed)},
			SuccessfulQueuedPurchaseDeletionSet: ec2Stage113SuccessfulQueuedPurchaseDeletionSet{Items: ec2Stage113SuccessfulQueuedPurchaseDeletionItemsFrom(successful)},
		})
		return true
	case "DeleteSpotDatafeedSubscription":
		if err := s.ec2.DeleteSpotDatafeedSubscription(); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeleteSpotDatafeedSubscriptionResponse{
			XMLName:   xml.Name{Local: "DeleteSpotDatafeedSubscriptionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "DeleteTrafficMirrorFilter":
		trafficMirrorFilterID, err := s.ec2.DeleteTrafficMirrorFilter(strings.TrimSpace(r.Form.Get("TrafficMirrorFilterId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeleteTrafficMirrorFilterResponse{
			XMLName:               xml.Name{Local: "DeleteTrafficMirrorFilterResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			TrafficMirrorFilterID: trafficMirrorFilterID,
		})
		return true
	case "DeleteTrafficMirrorFilterRule":
		trafficMirrorFilterRuleID, err := s.ec2.DeleteTrafficMirrorFilterRule(strings.TrimSpace(r.Form.Get("TrafficMirrorFilterRuleId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeleteTrafficMirrorFilterRuleResponse{
			XMLName:                   xml.Name{Local: "DeleteTrafficMirrorFilterRuleResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			TrafficMirrorFilterRuleID: trafficMirrorFilterRuleID,
		})
		return true
	case "DeleteTrafficMirrorSession":
		trafficMirrorSessionID, err := s.ec2.DeleteTrafficMirrorSession(strings.TrimSpace(r.Form.Get("TrafficMirrorSessionId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeleteTrafficMirrorSessionResponse{
			XMLName:                xml.Name{Local: "DeleteTrafficMirrorSessionResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			TrafficMirrorSessionID: trafficMirrorSessionID,
		})
		return true
	case "DeleteTrafficMirrorTarget":
		trafficMirrorTargetID, err := s.ec2.DeleteTrafficMirrorTarget(strings.TrimSpace(r.Form.Get("TrafficMirrorTargetId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeleteTrafficMirrorTargetResponse{
			XMLName:               xml.Name{Local: "DeleteTrafficMirrorTargetResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			TrafficMirrorTargetID: trafficMirrorTargetID,
		})
		return true
	case "DeprovisionByoipCidr":
		byoipCidr, err := s.ec2.DeprovisionByoipCidr(strings.TrimSpace(r.Form.Get("Cidr")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage113DeprovisionByoipCidrResponse{
			XMLName:   xml.Name{Local: "DeprovisionByoipCidrResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ByoipCidr: ec2ByoipCidrItemFrom(byoipCidr),
		})
		return true
	default:
		return false
	}
}

func ec2Stage113FailedQueuedPurchaseDeletionItemsFrom(in []ec2svc.FailedQueuedPurchaseDeletion) []ec2Stage113FailedQueuedPurchaseDeletionItem {
	out := make([]ec2Stage113FailedQueuedPurchaseDeletionItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage113FailedQueuedPurchaseDeletionItem{
			Error: ec2Stage113DeleteQueuedReservedInstancesErrorItem{
				Code:    item.Error.Code,
				Message: item.Error.Message,
			},
			ReservedInstancesID: item.ReservedInstancesID,
		})
	}
	return out
}

func ec2Stage113SuccessfulQueuedPurchaseDeletionItemsFrom(in []ec2svc.SuccessfulQueuedPurchaseDeletion) []ec2Stage113SuccessfulQueuedPurchaseDeletionItem {
	out := make([]ec2Stage113SuccessfulQueuedPurchaseDeletionItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage113SuccessfulQueuedPurchaseDeletionItem{ReservedInstancesID: item.ReservedInstancesID})
	}
	return out
}

type ec2Stage113DeleteNetworkInsightsAnalysisResponse struct {
	XMLName                   xml.Name `xml:"DeleteNetworkInsightsAnalysisResponse"`
	Xmlns                     string   `xml:"xmlns,attr"`
	RequestID                 string   `xml:"requestId"`
	NetworkInsightsAnalysisID string   `xml:"networkInsightsAnalysisId,omitempty"`
}

type ec2Stage113DeleteNetworkInsightsPathResponse struct {
	XMLName               xml.Name `xml:"DeleteNetworkInsightsPathResponse"`
	Xmlns                 string   `xml:"xmlns,attr"`
	RequestID             string   `xml:"requestId"`
	NetworkInsightsPathID string   `xml:"networkInsightsPathId,omitempty"`
}

type ec2Stage113DeletePublicIpv4PoolResponse struct {
	XMLName     xml.Name `xml:"DeletePublicIpv4PoolResponse"`
	Xmlns       string   `xml:"xmlns,attr"`
	RequestID   string   `xml:"requestId"`
	ReturnValue bool     `xml:"returnValue,omitempty"`
}

type ec2Stage113DeleteQueuedReservedInstancesResponse struct {
	XMLName                             xml.Name                                       `xml:"DeleteQueuedReservedInstancesResponse"`
	Xmlns                               string                                         `xml:"xmlns,attr"`
	RequestID                           string                                         `xml:"requestId"`
	FailedQueuedPurchaseDeletionSet     ec2Stage113FailedQueuedPurchaseDeletionSet     `xml:"failedQueuedPurchaseDeletionSet"`
	SuccessfulQueuedPurchaseDeletionSet ec2Stage113SuccessfulQueuedPurchaseDeletionSet `xml:"successfulQueuedPurchaseDeletionSet"`
}

type ec2Stage113FailedQueuedPurchaseDeletionSet struct {
	Items []ec2Stage113FailedQueuedPurchaseDeletionItem `xml:"item"`
}

type ec2Stage113FailedQueuedPurchaseDeletionItem struct {
	Error               ec2Stage113DeleteQueuedReservedInstancesErrorItem `xml:"error"`
	ReservedInstancesID string                                            `xml:"reservedInstancesId,omitempty"`
}

type ec2Stage113DeleteQueuedReservedInstancesErrorItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2Stage113SuccessfulQueuedPurchaseDeletionSet struct {
	Items []ec2Stage113SuccessfulQueuedPurchaseDeletionItem `xml:"item"`
}

type ec2Stage113SuccessfulQueuedPurchaseDeletionItem struct {
	ReservedInstancesID string `xml:"reservedInstancesId,omitempty"`
}

type ec2Stage113DeleteSpotDatafeedSubscriptionResponse struct {
	XMLName   xml.Name `xml:"DeleteSpotDatafeedSubscriptionResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
}

type ec2Stage113DeleteTrafficMirrorFilterResponse struct {
	XMLName               xml.Name `xml:"DeleteTrafficMirrorFilterResponse"`
	Xmlns                 string   `xml:"xmlns,attr"`
	RequestID             string   `xml:"requestId"`
	TrafficMirrorFilterID string   `xml:"trafficMirrorFilterId,omitempty"`
}

type ec2Stage113DeleteTrafficMirrorFilterRuleResponse struct {
	XMLName                   xml.Name `xml:"DeleteTrafficMirrorFilterRuleResponse"`
	Xmlns                     string   `xml:"xmlns,attr"`
	RequestID                 string   `xml:"requestId"`
	TrafficMirrorFilterRuleID string   `xml:"trafficMirrorFilterRuleId,omitempty"`
}

type ec2Stage113DeleteTrafficMirrorSessionResponse struct {
	XMLName                xml.Name `xml:"DeleteTrafficMirrorSessionResponse"`
	Xmlns                  string   `xml:"xmlns,attr"`
	RequestID              string   `xml:"requestId"`
	TrafficMirrorSessionID string   `xml:"trafficMirrorSessionId,omitempty"`
}

type ec2Stage113DeleteTrafficMirrorTargetResponse struct {
	XMLName               xml.Name `xml:"DeleteTrafficMirrorTargetResponse"`
	Xmlns                 string   `xml:"xmlns,attr"`
	RequestID             string   `xml:"requestId"`
	TrafficMirrorTargetID string   `xml:"trafficMirrorTargetId,omitempty"`
}

type ec2Stage113DeprovisionByoipCidrResponse struct {
	XMLName   xml.Name         `xml:"DeprovisionByoipCidrResponse"`
	Xmlns     string           `xml:"xmlns,attr"`
	RequestID string           `xml:"requestId"`
	ByoipCidr ec2ByoipCidrItem `xml:"byoipCidr"`
}
