package server

type locationOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Location Service actions sourced from:
// https://docs.aws.amazon.com/location/latest/APIReference/API_Operations.html
var locationOperations = []locationOperation{
	{Name: "AssociateTrackerConsumer", Method: "POST", URI: "/tracking/v0/trackers/{TrackerName}/consumers"},
	{Name: "BatchDeleteDevicePositionHistory", Method: "POST", URI: "/tracking/v0/trackers/{TrackerName}/delete-positions"},
	{Name: "BatchDeleteGeofence", Method: "POST", URI: "/geofencing/v0/collections/{CollectionName}/delete-geofences"},
	{Name: "BatchEvaluateGeofences", Method: "POST", URI: "/geofencing/v0/collections/{CollectionName}/positions"},
	{Name: "BatchGetDevicePosition", Method: "POST", URI: "/tracking/v0/trackers/{TrackerName}/get-positions"},
	{Name: "BatchPutGeofence", Method: "POST", URI: "/geofencing/v0/collections/{CollectionName}/put-geofences"},
	{Name: "BatchUpdateDevicePosition", Method: "POST", URI: "/tracking/v0/trackers/{TrackerName}/positions"},
	{Name: "CalculateRoute", Method: "POST", URI: "/routes/v0/calculators/{CalculatorName}/calculate/route"},
	{Name: "CalculateRouteMatrix", Method: "POST", URI: "/routes/v0/calculators/{CalculatorName}/calculate/route-matrix"},
	{Name: "CreateGeofenceCollection", Method: "POST", URI: "/geofencing/v0/collections"},
	{Name: "CreateKey", Method: "POST", URI: "/metadata/v0/keys"},
	{Name: "CreateMap", Method: "POST", URI: "/maps/v0/maps"},
	{Name: "CreatePlaceIndex", Method: "POST", URI: "/places/v0/indexes"},
	{Name: "CreateRouteCalculator", Method: "POST", URI: "/routes/v0/calculators"},
	{Name: "CreateTracker", Method: "POST", URI: "/tracking/v0/trackers"},
	{Name: "DeleteGeofenceCollection", Method: "DELETE", URI: "/geofencing/v0/collections/{CollectionName}"},
	{Name: "DeleteKey", Method: "DELETE", URI: "/metadata/v0/keys/{KeyName}"},
	{Name: "DeleteMap", Method: "DELETE", URI: "/maps/v0/maps/{MapName}"},
	{Name: "DeletePlaceIndex", Method: "DELETE", URI: "/places/v0/indexes/{IndexName}"},
	{Name: "DeleteRouteCalculator", Method: "DELETE", URI: "/routes/v0/calculators/{CalculatorName}"},
	{Name: "DeleteTracker", Method: "DELETE", URI: "/tracking/v0/trackers/{TrackerName}"},
	{Name: "DescribeGeofenceCollection", Method: "GET", URI: "/geofencing/v0/collections/{CollectionName}"},
	{Name: "DescribeKey", Method: "GET", URI: "/metadata/v0/keys/{KeyName}"},
	{Name: "DescribeMap", Method: "GET", URI: "/maps/v0/maps/{MapName}"},
	{Name: "DescribePlaceIndex", Method: "GET", URI: "/places/v0/indexes/{IndexName}"},
	{Name: "DescribeRouteCalculator", Method: "GET", URI: "/routes/v0/calculators/{CalculatorName}"},
	{Name: "DescribeTracker", Method: "GET", URI: "/tracking/v0/trackers/{TrackerName}"},
	{Name: "DisassociateTrackerConsumer", Method: "DELETE", URI: "/tracking/v0/trackers/{TrackerName}/consumers/{ConsumerArn}"},
	{Name: "ForecastGeofenceEvents", Method: "POST", URI: "/geofencing/v0/collections/{CollectionName}/forecast-geofence-events"},
	{Name: "GetDevicePosition", Method: "GET", URI: "/tracking/v0/trackers/{TrackerName}/devices/{DeviceId}/positions/latest"},
	{Name: "GetDevicePositionHistory", Method: "POST", URI: "/tracking/v0/trackers/{TrackerName}/devices/{DeviceId}/list-positions"},
	{Name: "GetGeofence", Method: "GET", URI: "/geofencing/v0/collections/{CollectionName}/geofences/{GeofenceId}"},
	{Name: "GetMapGlyphs", Method: "GET", URI: "/maps/v0/maps/{MapName}/glyphs/{FontStack}/{FontUnicodeRange}"},
	{Name: "GetMapSprites", Method: "GET", URI: "/maps/v0/maps/{MapName}/sprites/{FileName}"},
	{Name: "GetMapStyleDescriptor", Method: "GET", URI: "/maps/v0/maps/{MapName}/style-descriptor"},
	{Name: "GetMapTile", Method: "GET", URI: "/maps/v0/maps/{MapName}/tiles/{Z}/{X}/{Y}"},
	{Name: "GetPlace", Method: "GET", URI: "/places/v0/indexes/{IndexName}/places/{PlaceId}"},
	{Name: "ListDevicePositions", Method: "POST", URI: "/tracking/v0/trackers/{TrackerName}/list-positions"},
	{Name: "ListGeofenceCollections", Method: "POST", URI: "/geofencing/v0/list-collections"},
	{Name: "ListGeofences", Method: "POST", URI: "/geofencing/v0/collections/{CollectionName}/list-geofences"},
	{Name: "ListKeys", Method: "POST", URI: "/metadata/v0/list-keys"},
	{Name: "ListMaps", Method: "POST", URI: "/maps/v0/list-maps"},
	{Name: "ListPlaceIndexes", Method: "POST", URI: "/places/v0/list-indexes"},
	{Name: "ListRouteCalculators", Method: "POST", URI: "/routes/v0/list-calculators"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "ListTrackerConsumers", Method: "POST", URI: "/tracking/v0/trackers/{TrackerName}/list-consumers"},
	{Name: "ListTrackers", Method: "POST", URI: "/tracking/v0/list-trackers"},
	{Name: "PutGeofence", Method: "PUT", URI: "/geofencing/v0/collections/{CollectionName}/geofences/{GeofenceId}"},
	{Name: "SearchPlaceIndexForPosition", Method: "POST", URI: "/places/v0/indexes/{IndexName}/search/position"},
	{Name: "SearchPlaceIndexForSuggestions", Method: "POST", URI: "/places/v0/indexes/{IndexName}/search/suggestions"},
	{Name: "SearchPlaceIndexForText", Method: "POST", URI: "/places/v0/indexes/{IndexName}/search/text"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateGeofenceCollection", Method: "PATCH", URI: "/geofencing/v0/collections/{CollectionName}"},
	{Name: "UpdateKey", Method: "PATCH", URI: "/metadata/v0/keys/{KeyName}"},
	{Name: "UpdateMap", Method: "PATCH", URI: "/maps/v0/maps/{MapName}"},
	{Name: "UpdatePlaceIndex", Method: "PATCH", URI: "/places/v0/indexes/{IndexName}"},
	{Name: "UpdateRouteCalculator", Method: "PATCH", URI: "/routes/v0/calculators/{CalculatorName}"},
	{Name: "UpdateTracker", Method: "PATCH", URI: "/tracking/v0/trackers/{TrackerName}"},
	{Name: "VerifyDevicePosition", Method: "POST", URI: "/tracking/v0/trackers/{TrackerName}/positions/verify"},
}

var locationOperationByName = func() map[string]locationOperation {
	out := make(map[string]locationOperation, len(locationOperations))
	for _, op := range locationOperations {
		out[op.Name] = op
	}
	return out
}()
