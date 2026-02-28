package server

type healthDataType struct {
	Name string
}

// AWS Health data types sourced from:
// https://docs.aws.amazon.com/health/latest/APIReference/API_Types.html
var healthDataTypes = []healthDataType{
	{Name: "AccountEntityAggregate"},
	{Name: "AffectedEntity"},
	{Name: "DateTimeRange"},
	{Name: "EnableHealthServiceAccessForOrganization"},
	{Name: "EntityAccountFilter"},
	{Name: "EntityAggregate"},
	{Name: "EntityFilter"},
	{Name: "Event"},
	{Name: "EventAccountFilter"},
	{Name: "EventAggregate"},
	{Name: "EventDescription"},
	{Name: "EventDetails"},
	{Name: "EventDetailsErrorItem"},
	{Name: "EventFilter"},
	{Name: "EventType"},
	{Name: "EventTypeFilter"},
	{Name: "OrganizationAffectedEntitiesErrorItem"},
	{Name: "OrganizationEntityAggregate"},
	{Name: "OrganizationEvent"},
	{Name: "OrganizationEventDetails"},
	{Name: "OrganizationEventDetailsErrorItem"},
	{Name: "OrganizationEventFilter"},
}

var healthDataTypeByName = func() map[string]healthDataType {
	out := make(map[string]healthDataType, len(healthDataTypes))
	for _, dt := range healthDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
