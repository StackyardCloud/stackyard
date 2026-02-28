package server

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type cloudFrontStore struct {
	mu sync.Mutex

	distributionID string
	domainName     string
	nextID         int64
}

func newCloudFrontStore() *cloudFrontStore {
	return &cloudFrontStore{
		distributionID: "EDFDVBD632BHDS5",
		domainName:     "d111111abcdef8.cloudfront.net",
		nextID:         1,
	}
}

type cloudFrontResult struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

func (s *cloudFrontStore) Handle(action string, payload map[string]any, pathParams map[string]string) cloudFrontResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	distributionID := strings.TrimSpace(pathParams["distributionId"])
	if distributionID == "" {
		distributionID = strings.TrimSpace(cloudFrontPayloadString(payload, "Id", s.distributionID))
	}
	if distributionID == "" {
		distributionID = s.distributionID
	}

	switch action {
	case "ListDistributions":
		body := fmt.Sprintf(`<ListDistributionsResponse xmlns="http://cloudfront.amazonaws.com/doc/2020-05-31/"><DistributionList><Quantity>1</Quantity><Items><DistributionSummary><Id>%s</Id><DomainName>%s</DomainName><Status>Deployed</Status><Enabled>true</Enabled></DistributionSummary></Items></DistributionList></ListDistributionsResponse>`, s.distributionID, s.domainName)
		return cloudFrontResult{Status: 200, Body: []byte(body)}
	case "GetDistribution":
		body := fmt.Sprintf(`<GetDistributionResponse xmlns="http://cloudfront.amazonaws.com/doc/2020-05-31/"><Distribution><Id>%s</Id><DomainName>%s</DomainName><Status>Deployed</Status></Distribution><ETag>E2QWRUHAPOMQZL</ETag></GetDistributionResponse>`, distributionID, s.domainName)
		return cloudFrontResult{Status: 200, Body: []byte(body)}
	case "ListInvalidations":
		body := fmt.Sprintf(`<ListInvalidationsResponse xmlns="http://cloudfront.amazonaws.com/doc/2020-05-31/"><InvalidationList><Quantity>1</Quantity><Items><InvalidationSummary><Id>I1A2B3C4D5E6F7</Id><Status>Completed</Status><CreateTime>%s</CreateTime></InvalidationSummary></Items></InvalidationList></ListInvalidationsResponse>`, time.Now().UTC().Format(time.RFC3339))
		return cloudFrontResult{Status: 200, Body: []byte(body)}
	case "CreateInvalidation":
		invID := fmt.Sprintf("I%012d", s.nextID)
		s.nextID++
		location := fmt.Sprintf("/2020-05-31/distribution/%s/invalidation/%s", distributionID, invID)
		body := fmt.Sprintf(`<CreateInvalidationResponse xmlns="http://cloudfront.amazonaws.com/doc/2020-05-31/"><Location>%s</Location><Invalidation><Id>%s</Id><Status>InProgress</Status><CreateTime>%s</CreateTime></Invalidation></CreateInvalidationResponse>`, location, invID, time.Now().UTC().Format(time.RFC3339))
		return cloudFrontResult{Status: 201, Headers: map[string]string{"Location": location}, Body: []byte(body)}
	default:
		requestID := fmt.Sprintf("req-%012d", s.nextID)
		s.nextID++
		body := fmt.Sprintf(`<CloudFrontResponse xmlns="http://cloudfront.amazonaws.com/doc/2020-05-31/"><Operation>%s</Operation><RequestId>%s</RequestId><Status>OK</Status></CloudFrontResponse>`, action, requestID)
		return cloudFrontResult{Status: 200, Body: []byte(body)}
	}
}

func cloudFrontPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for payloadKey, value := range payload {
		if strings.EqualFold(strings.TrimSpace(payloadKey), strings.TrimSpace(key)) {
			out := strings.TrimSpace(fmt.Sprintf("%v", value))
			if out != "" && out != "%!v(<nil>)" {
				return out
			}
		}
	}
	return fallback
}
