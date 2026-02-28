package server

type notificationsContactsDataType struct {
	Name string
}

// AWS User Notifications Contacts data types sourced from:
// https://docs.aws.amazon.com/notificationscontacts/latest/APIReference/API_Types.html
var notificationsContactsDataTypes = []notificationsContactsDataType{
	{Name: "EmailContact"},
	{Name: "ValidationExceptionField"},
}

var notificationsContactsDataTypeByName = func() map[string]notificationsContactsDataType {
	out := make(map[string]notificationsContactsDataType, len(notificationsContactsDataTypes))
	for _, t := range notificationsContactsDataTypes {
		out[t.Name] = t
	}
	return out
}()
