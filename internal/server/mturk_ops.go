package server

type mtkOperation struct{ Name string }

// Amazon Mechanical Turk operations sourced from the AWS SDK for Go v2 operation
// files (the MTurk API reference is no longer on docs.aws.amazon.com), e.g.:
// https://github.com/aws/aws-sdk-go-v2/blob/main/service/mturk/api_op_CreateHIT.go
var mturkOperations = []mtkOperation{
	{Name: "AcceptQualificationRequest"},
	{Name: "ApproveAssignment"},
	{Name: "AssociateQualificationWithWorker"},
	{Name: "CreateAdditionalAssignmentsForHIT"},
	{Name: "CreateHIT"},
	{Name: "CreateHITType"},
	{Name: "CreateHITWithHITType"},
	{Name: "CreateQualificationType"},
	{Name: "CreateWorkerBlock"},
	{Name: "DeleteHIT"},
	{Name: "DeleteQualificationType"},
	{Name: "DeleteWorkerBlock"},
	{Name: "DisassociateQualificationFromWorker"},
	{Name: "GetAccountBalance"},
	{Name: "GetAssignment"},
	{Name: "GetFileUploadURL"},
	{Name: "GetHIT"},
	{Name: "GetQualificationScore"},
	{Name: "GetQualificationType"},
	{Name: "ListAssignmentsForHIT"},
	{Name: "ListBonusPayments"},
	{Name: "ListHITs"},
	{Name: "ListHITsForQualificationType"},
	{Name: "ListQualificationRequests"},
	{Name: "ListQualificationTypes"},
	{Name: "ListReviewPolicyResultsForHIT"},
	{Name: "ListReviewableHITs"},
	{Name: "ListWorkerBlocks"},
	{Name: "ListWorkersWithQualificationType"},
	{Name: "NotifyWorkers"},
	{Name: "RejectAssignment"},
	{Name: "RejectQualificationRequest"},
	{Name: "SendBonus"},
	{Name: "SendTestEventNotification"},
	{Name: "UpdateExpirationForHIT"},
	{Name: "UpdateHITReviewStatus"},
	{Name: "UpdateHITTypeOfHIT"},
	{Name: "UpdateNotificationSettings"},
	{Name: "UpdateQualificationType"},
}
