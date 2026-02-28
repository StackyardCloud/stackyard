package server

import "sync"

type devOpsGuruStore struct {
	mu sync.Mutex
}

func newDevOpsGuruStore() *devOpsGuruStore {
	return &devOpsGuruStore{}
}

func (s *devOpsGuruStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "DescribeAccountHealth":
		return map[string]any{
			"OpenReactiveInsights":  0,
			"OpenProactiveInsights": 0,
			"MetricsAnalyzed":       0,
			"ResourceHours":         0.0,
		}
	case "DescribeAccountOverview":
		return map[string]any{
			"ReactiveInsights":  []any{},
			"ProactiveInsights": []any{},
		}
	case "DescribeOrganizationHealth":
		return map[string]any{
			"OpenReactiveInsights":  0,
			"OpenProactiveInsights": 0,
			"MetricsAnalyzed":       0,
			"ResourceHours":         0.0,
		}
	case "DescribeOrganizationOverview":
		return map[string]any{"ReactiveInsights": []any{}, "ProactiveInsights": []any{}}
	case "DescribeOrganizationResourceCollectionHealth", "DescribeResourceCollectionHealth":
		return map[string]any{"OpenReactiveInsights": 0, "OpenProactiveInsights": 0, "MetricsAnalyzed": 0, "ResourceHours": 0.0}
	case "DescribeEventSourcesConfig":
		return map[string]any{}
	case "DescribeServiceIntegration":
		return map[string]any{"ServiceIntegration": map[string]any{}}
	case "GetResourceCollection":
		return map[string]any{"ResourceCollection": map[string]any{}}
	case "GetCostEstimation":
		return map[string]any{"Status": "COMPLETED", "Costs": []any{}}
	case "ListInsights", "SearchInsights":
		return map[string]any{"ProactiveInsights": []any{}, "ReactiveInsights": []any{}, "NextToken": ""}
	case "ListOrganizationInsights", "SearchOrganizationInsights":
		return map[string]any{"ProactiveInsights": []any{}, "ReactiveInsights": []any{}, "NextToken": ""}
	case "ListEvents":
		return map[string]any{"Events": []any{}, "NextToken": ""}
	case "ListRecommendations":
		return map[string]any{"Recommendations": []any{}, "NextToken": ""}
	case "ListAnomaliesForInsight":
		return map[string]any{"ProactiveAnomalies": []any{}, "ReactiveAnomalies": []any{}, "NextToken": ""}
	case "ListAnomalousLogGroups":
		return map[string]any{"AnomalousLogGroups": []any{}, "NextToken": ""}
	case "ListMonitoredResources":
		return map[string]any{"MonitoredResourceIdentifiers": []any{}, "NextToken": ""}
	case "ListNotificationChannels":
		return map[string]any{"Channels": []any{}, "NextToken": ""}
	case "AddNotificationChannel", "RemoveNotificationChannel", "DeleteInsight", "PutFeedback", "UpdateEventSourcesConfig", "UpdateResourceCollection", "UpdateServiceIntegration", "StartCostEstimation":
		return map[string]any{}
	case "DescribeAnomaly":
		return map[string]any{"ProactiveAnomaly": map[string]any{}, "ReactiveAnomaly": map[string]any{}}
	case "DescribeFeedback":
		return map[string]any{"InsightFeedback": map[string]any{}}
	case "DescribeInsight":
		return map[string]any{"ProactiveInsight": map[string]any{}, "ReactiveInsight": map[string]any{}}
	}

	return map[string]any{}
}
