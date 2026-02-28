package server

type mskv1Operation struct {
	Name   string
	Method string
	URI    string
}

// Amazon MSK Replicator API operations sourced from:
// https://docs.aws.amazon.com/msk/1.0/apireference-replicator/operations.html
var mskv1Operations = []mskv1Operation{
	{Name: "CreateReplicator", Method: "POST", URI: "/replication/v1/replicators"},
	{Name: "DeleteReplicator", Method: "DELETE", URI: "/replication/v1/replicators/{replicatorArn}"},
	{Name: "DescribeReplicator", Method: "GET", URI: "/replication/v1/replicators/{replicatorArn}"},
	{Name: "ListReplicators", Method: "GET", URI: "/replication/v1/replicators"},
	{Name: "UpdateReplicationInfo", Method: "PUT", URI: "/replication/v1/replicators/{replicatorArn}/replication-info"},
}

var mskv1OperationByName = func() map[string]mskv1Operation {
	out := make(map[string]mskv1Operation, len(mskv1Operations))
	for _, op := range mskv1Operations {
		out[op.Name] = op
	}
	return out
}()
