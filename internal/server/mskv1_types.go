package server

type mskv1Resource struct {
	Name string
}

// Amazon MSK Replicator API resources sourced from:
// https://docs.aws.amazon.com/msk/1.0/apireference-replicator/resources.html
// and linked resource model pages.
var mskv1Resources = []mskv1Resource{
	{Name: "AmazonMskCluster"},
	{Name: "ConsumerGroupReplication"},
	{Name: "ConsumerGroupReplicationUpdate"},
	{Name: "CreateReplicatorRequest"},
	{Name: "CreateReplicatorResponse"},
	{Name: "DeleteReplicatorResponse"},
	{Name: "DescribeReplicatorResponse"},
	{Name: "KafkaCluster"},
	{Name: "KafkaClusterClientVpcConfig"},
	{Name: "KafkaClusterDescription"},
	{Name: "KafkaClusterSummary"},
	{Name: "ListReplicatorsResponse"},
	{Name: "ReplicationInfo"},
	{Name: "ReplicationInfoDescription"},
	{Name: "ReplicationInfoSummary"},
	{Name: "ReplicationStartingPosition"},
	{Name: "ReplicationStartingPositionType"},
	{Name: "ReplicationStateInfo"},
	{Name: "ReplicationTopicNameConfiguration"},
	{Name: "ReplicationTopicNameConfigurationType"},
	{Name: "ReplicatorState"},
	{Name: "ReplicatorSummary"},
	{Name: "TargetCompressionType"},
	{Name: "TopicReplication"},
	{Name: "TopicReplicationUpdate"},
	{Name: "UpdateReplicationInfoRequest"},
	{Name: "UpdateReplicationInfoResponse"},
}

var mskv1ResourceByName = func() map[string]mskv1Resource {
	out := make(map[string]mskv1Resource, len(mskv1Resources))
	for _, r := range mskv1Resources {
		out[r.Name] = r
	}
	return out
}()
