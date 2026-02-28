package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage17Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateCustomerGateway":
		bgpASN, bgpASNExtended, ok := parseEC2CustomerGatewayASNValues(r.Form.Get("BgpAsn"), r.Form.Get("BgpAsnExtended"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		gateway, err := s.ec2.CreateCustomerGateway(
			strings.TrimSpace(r.Form.Get("Type")),
			firstNonEmptyEC2Param(r.Form.Get("IpAddress"), r.Form.Get("PublicIp")),
			bgpASN,
			bgpASNExtended,
			strings.TrimSpace(r.Form.Get("CertificateArn")),
			strings.TrimSpace(r.Form.Get("DeviceName")),
			parseEC2TagSpecificationsForResource(r.Form, "customer-gateway"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateCustomerGatewayResponse{
			XMLName:         xml.Name{Local: "CreateCustomerGatewayResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			CustomerGateway: ec2CustomerGatewayItemFrom(gateway),
		})
		return true
	case "DescribeCustomerGateways":
		gateways := s.ec2.DescribeCustomerGateways(
			parseEC2Members(r.Form, "CustomerGatewayId."),
			parseEC2FilterValues(r.Form, "customer-gateway-id"),
			parseEC2FilterValues(r.Form, "ip-address"),
			parseEC2FilterValues(r.Form, "state"),
			parseEC2FilterValues(r.Form, "type"),
			parseEC2FilterValues(r.Form, "bgp-asn"),
			parseEC2FilterValues(r.Form, "tag-key"),
			parseEC2TagValueFilters(r.Form),
		)
		respondEC2XML(w, ec2DescribeCustomerGatewaysResponse{
			XMLName:   xml.Name{Local: "DescribeCustomerGatewaysResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			CustomerGatewaySet: ec2CustomerGatewaySet{
				Items: ec2CustomerGatewayItems(gateways),
			},
		})
		return true
	case "DeleteCustomerGateway":
		if err := s.ec2.DeleteCustomerGateway(strings.TrimSpace(r.Form.Get("CustomerGatewayId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteCustomerGatewayResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	default:
		return false
	}
}

func parseEC2CustomerGatewayASNValues(bgpASNValue, bgpASNExtendedValue string) (*int64, *int64, bool) {
	bgpASN, ok := parseEC2OptionalInt64(bgpASNValue)
	if !ok {
		return nil, nil, false
	}
	bgpASNExtended, ok := parseEC2OptionalInt64(bgpASNExtendedValue)
	if !ok {
		return nil, nil, false
	}
	return bgpASN, bgpASNExtended, true
}

func parseEC2OptionalInt64(value string) (*int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, false
	}
	return &n, true
}

func firstNonEmptyEC2Param(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseEC2TagValueFilters(values url.Values) map[string][]string {
	indices := map[int]string{}
	for key := range values {
		if !strings.HasPrefix(key, "Filter.") || !strings.HasSuffix(key, ".Name") {
			continue
		}
		indexText := strings.TrimPrefix(key, "Filter.")
		indexText = strings.TrimSuffix(indexText, ".Name")
		index, err := strconv.Atoi(indexText)
		if err != nil || index <= 0 {
			continue
		}
		name := strings.TrimSpace(values.Get(key))
		if !strings.HasPrefix(strings.ToLower(name), "tag:") {
			continue
		}
		tagKey := strings.TrimSpace(name[4:])
		if tagKey == "" {
			continue
		}
		indices[index] = tagKey
	}

	ordered := make([]int, 0, len(indices))
	for index := range indices {
		ordered = append(ordered, index)
	}
	sort.Ints(ordered)

	out := map[string][]string{}
	for _, index := range ordered {
		tagKey := indices[index]
		tagValues := parseEC2Members(values, "Filter."+strconv.Itoa(index)+".Value.")
		if len(tagValues) == 0 {
			continue
		}
		out[tagKey] = append(out[tagKey], tagValues...)
	}
	return out
}

func ec2CustomerGatewayItems(in []ec2svc.CustomerGateway) []ec2CustomerGatewayItem {
	out := make([]ec2CustomerGatewayItem, 0, len(in))
	for _, gateway := range in {
		out = append(out, ec2CustomerGatewayItemFrom(gateway))
	}
	return out
}

func ec2CustomerGatewayItemFrom(in ec2svc.CustomerGateway) ec2CustomerGatewayItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2CustomerGatewayItem{
		BgpAsn:            in.BgpASN,
		BgpAsnExtended:    in.BgpASNExtended,
		CertificateARN:    in.CertificateARN,
		CustomerGatewayID: in.ID,
		DeviceName:        in.DeviceName,
		IPAddress:         in.IPAddress,
		State:             in.State,
		Type:              in.Type,
		TagSet:            ec2TagSet{Items: tags},
	}
}

type ec2CreateCustomerGatewayResponse struct {
	XMLName         xml.Name
	Xmlns           string                 `xml:"xmlns,attr"`
	RequestID       string                 `xml:"requestId"`
	CustomerGateway ec2CustomerGatewayItem `xml:"customerGateway"`
}

type ec2DescribeCustomerGatewaysResponse struct {
	XMLName            xml.Name              `xml:"DescribeCustomerGatewaysResponse"`
	Xmlns              string                `xml:"xmlns,attr"`
	RequestID          string                `xml:"requestId"`
	CustomerGatewaySet ec2CustomerGatewaySet `xml:"customerGatewaySet"`
}

type ec2CustomerGatewaySet struct {
	Items []ec2CustomerGatewayItem `xml:"item"`
}

type ec2CustomerGatewayItem struct {
	BgpAsn            string    `xml:"bgpAsn,omitempty"`
	BgpAsnExtended    string    `xml:"bgpAsnExtended,omitempty"`
	CertificateARN    string    `xml:"certificateArn,omitempty"`
	CustomerGatewayID string    `xml:"customerGatewayId,omitempty"`
	DeviceName        string    `xml:"deviceName,omitempty"`
	IPAddress         string    `xml:"ipAddress,omitempty"`
	State             string    `xml:"state,omitempty"`
	Type              string    `xml:"type,omitempty"`
	TagSet            ec2TagSet `xml:"tagSet,omitempty"`
}
