package ec2

import (
	"sort"
	"strings"
)

func (s *Service) DescribeAwsNetworkPerformanceMetricSubscriptions(
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]AwsNetworkPerformanceSubscription, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	sourceFilterSet := toLowerStringSet(standardFilters["source"])
	destinationFilterSet := toLowerStringSet(standardFilters["destination"])
	metricFilterSet := toLowerStringSet(standardFilters["metric"])
	periodFilterSet := toLowerStringSet(standardFilters["period"])
	statisticFilterSet := toLowerStringSet(standardFilters["statistic"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]AwsNetworkPerformanceSubscription, 0, len(s.awsNetworkPerformanceMetricSubscriptions))
	for _, subscription := range s.awsNetworkPerformanceMetricSubscriptions {
		if len(sourceFilterSet) > 0 {
			if _, ok := sourceFilterSet[strings.ToLower(strings.TrimSpace(subscription.Source))]; !ok {
				continue
			}
		}
		if len(destinationFilterSet) > 0 {
			if _, ok := destinationFilterSet[strings.ToLower(strings.TrimSpace(subscription.Destination))]; !ok {
				continue
			}
		}
		if len(metricFilterSet) > 0 {
			if _, ok := metricFilterSet[strings.ToLower(strings.TrimSpace(subscription.Metric))]; !ok {
				continue
			}
		}
		if len(periodFilterSet) > 0 {
			if _, ok := periodFilterSet[strings.ToLower(strings.TrimSpace(subscription.Period))]; !ok {
				continue
			}
		}
		if len(statisticFilterSet) > 0 {
			if _, ok := statisticFilterSet[strings.ToLower(strings.TrimSpace(subscription.Statistic))]; !ok {
				continue
			}
		}

		items = append(items, AwsNetworkPerformanceSubscription{
			Source:      subscription.Source,
			Destination: subscription.Destination,
			Metric:      subscription.Metric,
			Period:      subscription.Period,
			Statistic:   subscription.Statistic,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Source != items[j].Source {
			return items[i].Source < items[j].Source
		}
		if items[i].Destination != items[j].Destination {
			return items[i].Destination < items[j].Destination
		}
		if items[i].Metric != items[j].Metric {
			return items[i].Metric < items[j].Metric
		}
		if items[i].Statistic != items[j].Statistic {
			return items[i].Statistic < items[j].Statistic
		}
		return items[i].Period < items[j].Period
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]AwsNetworkPerformanceSubscription(nil), items[start:end]...), outputToken, nil
}
