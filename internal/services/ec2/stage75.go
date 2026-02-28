package ec2

import (
	"strings"
	"time"
)

type AwsNetworkPerformanceDataQuery struct {
	ID          string
	Source      string
	Destination string
	Metric      string
	Period      string
	Statistic   string
}

type AwsNetworkPerformanceMetricPoint struct {
	StartDate time.Time
	EndDate   time.Time
	Status    string
	Value     float32
}

type AwsNetworkPerformanceDataResponse struct {
	ID           string
	Source       string
	Destination  string
	Metric       string
	Period       string
	Statistic    string
	MetricPoints []AwsNetworkPerformanceMetricPoint
}

type AwsNetworkPerformanceSubscription struct {
	Source      string
	Destination string
	Metric      string
	Period      string
	Statistic   string
}

func (s *Service) EnableAwsNetworkPerformanceMetricSubscription(source, destination, metric, statistic string) (bool, error) {
	key, err := networkPerformanceSubscriptionKey(source, destination, metric, statistic)
	if err != nil {
		return false, err
	}

	source = strings.TrimSpace(source)
	destination = strings.TrimSpace(destination)
	metric = strings.TrimSpace(metric)
	statistic = strings.TrimSpace(statistic)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.awsNetworkPerformanceMetricSubscriptions[key] = AwsNetworkPerformanceSubscription{
		Source:      source,
		Destination: destination,
		Metric:      metric,
		Period:      "five-minutes",
		Statistic:   statistic,
	}
	return true, nil
}

func (s *Service) DisableAwsNetworkPerformanceMetricSubscription(source, destination, metric, statistic string) (bool, error) {
	key, err := networkPerformanceSubscriptionKey(source, destination, metric, statistic)
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.awsNetworkPerformanceMetricSubscriptions, key)
	return true, nil
}

func (s *Service) GetAwsNetworkPerformanceData(
	dataQueries []AwsNetworkPerformanceDataQuery,
	startTime time.Time,
	endTime time.Time,
	maxResults *int32,
	nextToken *string,
) ([]AwsNetworkPerformanceDataResponse, *string, error) {
	if len(dataQueries) == 0 || startTime.IsZero() || endTime.IsZero() || endTime.Before(startTime) {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}

	responses := make([]AwsNetworkPerformanceDataResponse, 0, len(dataQueries))

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, query := range dataQueries {
		normalizedSource := strings.TrimSpace(query.Source)
		normalizedDestination := strings.TrimSpace(query.Destination)
		normalizedMetric := strings.TrimSpace(query.Metric)
		normalizedStatistic := strings.TrimSpace(query.Statistic)
		normalizedPeriod := strings.TrimSpace(query.Period)
		if normalizedSource == "" || normalizedDestination == "" || normalizedMetric == "" || normalizedStatistic == "" {
			return nil, nil, ErrInvalidParameter
		}
		if normalizedPeriod == "" {
			normalizedPeriod = "five-minutes"
		}

		subscriptionKey, _ := networkPerformanceSubscriptionKey(normalizedSource, normalizedDestination, normalizedMetric, normalizedStatistic)
		_, subscribed := s.awsNetworkPerformanceMetricSubscriptions[subscriptionKey]

		status := "degraded"
		value := float32(0.0)
		if subscribed {
			status = "healthy"
			value = 1.0
		}

		responses = append(responses, AwsNetworkPerformanceDataResponse{
			ID:          strings.TrimSpace(query.ID),
			Source:      normalizedSource,
			Destination: normalizedDestination,
			Metric:      normalizedMetric,
			Period:      normalizedPeriod,
			Statistic:   normalizedStatistic,
			MetricPoints: []AwsNetworkPerformanceMetricPoint{
				{
					StartDate: startTime.UTC(),
					EndDate:   endTime.UTC(),
					Status:    status,
					Value:     value,
				},
			},
		})
	}

	start, end, outputToken, err := ec2PageWindow(len(responses), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]AwsNetworkPerformanceDataResponse(nil), responses[start:end]...), outputToken, nil
}

func (s *Service) EnableReachabilityAnalyzerOrganizationSharing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reachabilityAnalyzerOrganizationSharing = true
	return true
}

func (s *Service) EnableVolumeIO(volumeID string) error {
	volumeID = strings.TrimSpace(volumeID)
	if volumeID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	volume := s.volumes[volumeID]
	if volume == nil {
		return ErrNotFound
	}
	volume.AutoEnableIO = true
	return nil
}

func networkPerformanceSubscriptionKey(source, destination, metric, statistic string) (string, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	destination = strings.ToLower(strings.TrimSpace(destination))
	metric = strings.ToLower(strings.TrimSpace(metric))
	statistic = strings.ToLower(strings.TrimSpace(statistic))
	if source == "" || destination == "" || metric == "" || statistic == "" {
		return "", ErrInvalidParameter
	}
	return source + "|" + destination + "|" + metric + "|" + statistic, nil
}
