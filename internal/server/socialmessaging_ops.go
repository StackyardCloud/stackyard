package server

type socialMessagingOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon End User Messaging Social actions sourced from:
// https://docs.aws.amazon.com/social-messaging/latest/APIReference/API_Operations.html
var socialMessagingOperations = []socialMessagingOperation{
	{Name: "AssociateWhatsAppBusinessAccount", Method: "POST", URI: "/v1/whatsapp/signup"},
	{Name: "CreateWhatsAppMessageTemplate", Method: "POST", URI: "/v1/whatsapp/template/put"},
	{Name: "CreateWhatsAppMessageTemplateFromLibrary", Method: "POST", URI: "/v1/whatsapp/template/create"},
	{Name: "CreateWhatsAppMessageTemplateMedia", Method: "POST", URI: "/v1/whatsapp/template/media"},
	{Name: "DeleteWhatsAppMessageMedia", Method: "DELETE", URI: "/v1/whatsapp/media"},
	{Name: "DeleteWhatsAppMessageTemplate", Method: "DELETE", URI: "/v1/whatsapp/template"},
	{Name: "DisassociateWhatsAppBusinessAccount", Method: "DELETE", URI: "/v1/whatsapp/waba/disassociate"},
	{Name: "GetLinkedWhatsAppBusinessAccount", Method: "GET", URI: "/v1/whatsapp/waba/details"},
	{Name: "GetLinkedWhatsAppBusinessAccountPhoneNumber", Method: "GET", URI: "/v1/whatsapp/waba/phone/details"},
	{Name: "GetWhatsAppMessageMedia", Method: "POST", URI: "/v1/whatsapp/media/get"},
	{Name: "GetWhatsAppMessageTemplate", Method: "GET", URI: "/v1/whatsapp/template"},
	{Name: "ListLinkedWhatsAppBusinessAccounts", Method: "GET", URI: "/v1/whatsapp/waba/list"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/v1/tags/list"},
	{Name: "ListWhatsAppMessageTemplates", Method: "GET", URI: "/v1/whatsapp/template/list"},
	{Name: "ListWhatsAppTemplateLibrary", Method: "POST", URI: "/v1/whatsapp/template/library"},
	{Name: "PostWhatsAppMessageMedia", Method: "POST", URI: "/v1/whatsapp/media"},
	{Name: "PutWhatsAppBusinessAccountEventDestinations", Method: "PUT", URI: "/v1/whatsapp/waba/eventdestinations"},
	{Name: "SendWhatsAppMessage", Method: "POST", URI: "/v1/whatsapp/send"},
	{Name: "TagResource", Method: "POST", URI: "/v1/tags/tag-resource"},
	{Name: "UntagResource", Method: "POST", URI: "/v1/tags/untag-resource"},
	{Name: "UpdateWhatsAppMessageTemplate", Method: "POST", URI: "/v1/whatsapp/template"},
}

var socialMessagingOperationByName = func() map[string]socialMessagingOperation {
	out := make(map[string]socialMessagingOperation, len(socialMessagingOperations))
	for _, op := range socialMessagingOperations {
		out[op.Name] = op
	}
	return out
}()
