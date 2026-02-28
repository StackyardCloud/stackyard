package server

type applicationSignalsDataType struct {
	Name string
}

// Amazon Application Signals data types sourced from:
// https://docs.aws.amazon.com/applicationsignals/latest/APIReference/API_Types.html
var applicationSignalsDataTypes = []applicationSignalsDataType{
	{Name: "AttributeFilter"},
	{Name: "AuditFinding"},
	{Name: "AuditorResult"},
	{Name: "AuditTarget"},
	{Name: "AuditTargetEntity"},
	{Name: "BatchUpdateExclusionWindowsError"},
	{Name: "BurnRateConfiguration"},
	{Name: "CalendarInterval"},
	{Name: "CanaryEntity"},
	{Name: "ChangeEvent"},
	{Name: "DependencyConfig"},
	{Name: "DependencyGraph"},
	{Name: "Dimension"},
	{Name: "Edge"},
	{Name: "ExclusionWindow"},
	{Name: "Goal"},
	{Name: "GroupingAttributeDefinition"},
	{Name: "GroupingConfiguration"},
	{Name: "Interval"},
	{Name: "Metric"},
	{Name: "MetricDataQuery"},
	{Name: "MetricGraph"},
	{Name: "MetricReference"},
	{Name: "MetricStat"},
	{Name: "MonitoredRequestCountMetricDataQueries"},
	{Name: "Node"},
	{Name: "RecurrenceRule"},
	{Name: "RequestBasedServiceLevelIndicator"},
	{Name: "RequestBasedServiceLevelIndicatorConfig"},
	{Name: "RequestBasedServiceLevelIndicatorMetric"},
	{Name: "RequestBasedServiceLevelIndicatorMetricConfig"},
	{Name: "RollingInterval"},
	{Name: "Service"},
	{Name: "ServiceDependency"},
	{Name: "ServiceDependent"},
	{Name: "ServiceEntity"},
	{Name: "ServiceGroup"},
	{Name: "ServiceLevelIndicator"},
	{Name: "ServiceLevelIndicatorConfig"},
	{Name: "ServiceLevelIndicatorMetric"},
	{Name: "ServiceLevelIndicatorMetricConfig"},
	{Name: "ServiceLevelObjective"},
	{Name: "ServiceLevelObjectiveBudgetReport"},
	{Name: "ServiceLevelObjectiveBudgetReportError"},
	{Name: "ServiceLevelObjectiveEntity"},
	{Name: "ServiceLevelObjectiveSummary"},
	{Name: "ServiceOperation"},
	{Name: "ServiceOperationEntity"},
	{Name: "ServiceState"},
	{Name: "ServiceSummary"},
	{Name: "Tag"},
	{Name: "Window"},
}

var applicationSignalsDataTypeByName = func() map[string]applicationSignalsDataType {
	out := make(map[string]applicationSignalsDataType, len(applicationSignalsDataTypes))
	for _, dt := range applicationSignalsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
