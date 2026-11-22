package server

type lexruntimev2Operation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Lex Runtime V2 operations sourced from:
// https://docs.aws.amazon.com/lexv2/latest/APIReference/API_Operations_Amazon_Lex_Runtime_V2.html
var lexruntimev2Operations = []lexruntimev2Operation{
	{Name: "DeleteSession", Method: "DELETE", URI: "/bots/{botId}/botAliases/{botAliasId}/botLocales/{localeId}/sessions/{sessionId}"},
	{Name: "GetSession", Method: "GET", URI: "/bots/{botId}/botAliases/{botAliasId}/botLocales/{localeId}/sessions/{sessionId}"},
	{Name: "PutSession", Method: "POST", URI: "/bots/{botId}/botAliases/{botAliasId}/botLocales/{localeId}/sessions/{sessionId}"},
	{Name: "RecognizeText", Method: "POST", URI: "/bots/{botId}/botAliases/{botAliasId}/botLocales/{localeId}/sessions/{sessionId}/text"},
	{Name: "RecognizeUtterance", Method: "POST", URI: "/bots/{botId}/botAliases/{botAliasId}/botLocales/{localeId}/sessions/{sessionId}/utterance"},
	{Name: "StartConversation", Method: "POST", URI: "/bots/{botId}/botAliases/{botAliasId}/botLocales/{localeId}/sessions/{sessionId}/conversation"},
}

var lexruntimev2OperationByName = func() map[string]lexruntimev2Operation {
	out := make(map[string]lexruntimev2Operation, len(lexruntimev2Operations))
	for _, op := range lexruntimev2Operations {
		out[op.Name] = op
	}
	return out
}()
