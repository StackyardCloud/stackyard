package server

type notificationsContactsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS User Notifications Contacts operations sourced from:
// https://docs.aws.amazon.com/notificationscontacts/latest/APIReference/API_Operations.html
var notificationsContactsOperations = []notificationsContactsOperation{
	{Name: "ActivateEmailContact", Method: "PUT", URI: "/emailcontacts/{arn}/activate/{code}"},
	{Name: "CreateEmailContact", Method: "POST", URI: "/2022-09-19/emailcontacts"},
	{Name: "DeleteEmailContact", Method: "DELETE", URI: "/emailcontacts/{arn}"},
	{Name: "GetEmailContact", Method: "GET", URI: "/emailcontacts/{arn}"},
	{Name: "ListEmailContacts", Method: "GET", URI: "/emailcontacts"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{arn}"},
	{Name: "SendActivationCode", Method: "POST", URI: "/2022-10-31/emailcontacts/{arn}/activate/send"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{arn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{arn}"},
}

var notificationsContactsOperationByName = func() map[string]notificationsContactsOperation {
	out := make(map[string]notificationsContactsOperation, len(notificationsContactsOperations))
	for _, op := range notificationsContactsOperations {
		out[op.Name] = op
	}
	return out
}()
