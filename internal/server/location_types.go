package server

type locationDataType struct {
	Name string
}

// Amazon Location Service data types sourced from:
// https://docs.aws.amazon.com/location/latest/APIReference/API_Types.html
var locationDataTypes = []locationDataType{
	{Name: "ApiKeyFilter"},
	{Name: "ApiKeyRestrictions"},
	{Name: "BatchPutGeofenceRequestEntry"},
	{Name: "BatchPutGeofenceSuccess"},
	{Name: "CalculateRouteCarModeOptions"},
	{Name: "CalculateRouteMatrixSummary"},
	{Name: "CalculateRouteSummary"},
	{Name: "CalculateRouteTruckModeOptions"},
	{Name: "CellSignals"},
	{Name: "Circle"},
	{Name: "DataSourceConfiguration"},
	{Name: "DevicePosition"},
	{Name: "DevicePositionUpdate"},
	{Name: "DeviceState"},
	{Name: "ForecastGeofenceEventsDeviceState"},
	{Name: "ForecastedEvent"},
	{Name: "GeofenceGeometry"},
	{Name: "InferredState"},
	{Name: "Leg"},
	{Name: "LegGeometry"},
	{Name: "ListDevicePositionsResponseEntry"},
	{Name: "ListGeofenceCollectionsResponseEntry"},
	{Name: "ListGeofenceResponseEntry"},
	{Name: "ListKeysResponseEntry"},
	{Name: "ListMapsResponseEntry"},
	{Name: "ListPlaceIndexesResponseEntry"},
	{Name: "ListRouteCalculatorsResponseEntry"},
	{Name: "ListTrackersResponseEntry"},
	{Name: "LteCellDetails"},
	{Name: "LteLocalId"},
	{Name: "LteNetworkMeasurements"},
	{Name: "MapConfiguration"},
	{Name: "MapConfigurationUpdate"},
	{Name: "Place"},
	{Name: "PlaceGeometry"},
	{Name: "PositionalAccuracy"},
	{Name: "RouteMatrixEntry"},
	{Name: "SearchForPositionResult"},
	{Name: "SearchForSuggestionsResult"},
	{Name: "SearchForTextResult"},
	{Name: "SearchPlaceIndexForPositionSummary"},
	{Name: "SearchPlaceIndexForSuggestionsSummary"},
	{Name: "SearchPlaceIndexForTextSummary"},
	{Name: "Step"},
	{Name: "TimeZone"},
	{Name: "TrackingFilterGeometry"},
	{Name: "TruckDimensions"},
	{Name: "TruckWeight"},
	{Name: "ValidationExceptionField"},
	{Name: "WiFiAccessPoint"},
}

var locationDataTypeByName = func() map[string]locationDataType {
	out := make(map[string]locationDataType, len(locationDataTypes))
	for _, dt := range locationDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
