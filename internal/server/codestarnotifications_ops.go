package server

type codestarnotificationsOperation struct {
	Name string
	Path string
}

// AWS CodeStar Notifications operations sourced from:
// https://docs.aws.amazon.com/codestar-notifications/latest/APIReference/API_Operations.html
// Every operation is a POST; UntagResource carries the rule ARN in the path.
var codestarnotificationsOperations = []codestarnotificationsOperation{
	{Name: "CreateNotificationRule", Path: "/createNotificationRule"},
	{Name: "DeleteNotificationRule", Path: "/deleteNotificationRule"},
	{Name: "DeleteTarget", Path: "/deleteTarget"},
	{Name: "DescribeNotificationRule", Path: "/describeNotificationRule"},
	{Name: "ListEventTypes", Path: "/listEventTypes"},
	{Name: "ListNotificationRules", Path: "/listNotificationRules"},
	{Name: "ListTagsForResource", Path: "/listTagsForResource"},
	{Name: "ListTargets", Path: "/listTargets"},
	{Name: "Subscribe", Path: "/subscribe"},
	{Name: "TagResource", Path: "/tagResource"},
	{Name: "Unsubscribe", Path: "/unsubscribe"},
	{Name: "UntagResource", Path: "/untagResource/"},
	{Name: "UpdateNotificationRule", Path: "/updateNotificationRule"},
}

var codestarnotificationsOperationByPath = func() map[string]codestarnotificationsOperation {
	out := make(map[string]codestarnotificationsOperation, len(codestarnotificationsOperations))
	for _, op := range codestarnotificationsOperations {
		out[op.Path] = op
	}
	return out
}()
