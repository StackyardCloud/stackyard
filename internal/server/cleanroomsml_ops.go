package server

type cleanRoomsMLOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Clean Rooms ML actions sourced from:
// https://docs.aws.amazon.com/cleanrooms-ml/latest/APIReference/API_Operations.html
var cleanRoomsMLOperations = []cleanRoomsMLOperation{
	{Name: "CancelTrainedModel", Method: "PATCH", URI: "/memberships/{membershipIdentifier}/trained-models/{trainedModelArn}?versionIdentifier={versionIdentifier}"},
	{Name: "CancelTrainedModelInferenceJob", Method: "PATCH", URI: "/memberships/{membershipIdentifier}/trained-model-inference-jobs/{trainedModelInferenceJobArn}"},
	{Name: "CreateAudienceModel", Method: "POST", URI: "/audience-model"},
	{Name: "CreateConfiguredAudienceModel", Method: "POST", URI: "/configured-audience-model"},
	{Name: "CreateConfiguredModelAlgorithm", Method: "POST", URI: "/configured-model-algorithms"},
	{Name: "CreateConfiguredModelAlgorithmAssociation", Method: "POST", URI: "/memberships/{membershipIdentifier}/configured-model-algorithm-associations"},
	{Name: "CreateMLInputChannel", Method: "POST", URI: "/memberships/{membershipIdentifier}/ml-input-channels"},
	{Name: "CreateTrainedModel", Method: "POST", URI: "/memberships/{membershipIdentifier}/trained-models"},
	{Name: "CreateTrainingDataset", Method: "POST", URI: "/training-dataset"},
	{Name: "DeleteAudienceGenerationJob", Method: "DELETE", URI: "/audience-generation-job/{audienceGenerationJobArn}"},
	{Name: "DeleteAudienceModel", Method: "DELETE", URI: "/audience-model/{audienceModelArn}"},
	{Name: "DeleteConfiguredAudienceModel", Method: "DELETE", URI: "/configured-audience-model/{configuredAudienceModelArn}"},
	{Name: "DeleteConfiguredAudienceModelPolicy", Method: "DELETE", URI: "/configured-audience-model/{configuredAudienceModelArn}/policy"},
	{Name: "DeleteConfiguredModelAlgorithm", Method: "DELETE", URI: "/configured-model-algorithms/{configuredModelAlgorithmArn}"},
	{Name: "DeleteConfiguredModelAlgorithmAssociation", Method: "DELETE", URI: "/memberships/{membershipIdentifier}/configured-model-algorithm-associations/{configuredModelAlgorithmAssociationArn}"},
	{Name: "DeleteMLConfiguration", Method: "DELETE", URI: "/memberships/{membershipIdentifier}/ml-configurations"},
	{Name: "DeleteMLInputChannelData", Method: "DELETE", URI: "/memberships/{membershipIdentifier}/ml-input-channels/{mlInputChannelArn}"},
	{Name: "DeleteTrainedModelOutput", Method: "DELETE", URI: "/memberships/{membershipIdentifier}/trained-models/{trainedModelArn}?versionIdentifier={versionIdentifier}"},
	{Name: "DeleteTrainingDataset", Method: "DELETE", URI: "/training-dataset/{trainingDatasetArn}"},
	{Name: "GetAudienceGenerationJob", Method: "GET", URI: "/audience-generation-job/{audienceGenerationJobArn}"},
	{Name: "GetAudienceModel", Method: "GET", URI: "/audience-model/{audienceModelArn}"},
	{Name: "GetCollaborationConfiguredModelAlgorithmAssociation", Method: "GET", URI: "/collaborations/{collaborationIdentifier}/configured-model-algorithm-associations/{configuredModelAlgorithmAssociationArn}"},
	{Name: "GetCollaborationMLInputChannel", Method: "GET", URI: "/collaborations/{collaborationIdentifier}/ml-input-channels/{mlInputChannelArn}"},
	{Name: "GetCollaborationTrainedModel", Method: "GET", URI: "/collaborations/{collaborationIdentifier}/trained-models/{trainedModelArn}?versionIdentifier={versionIdentifier}"},
	{Name: "GetConfiguredAudienceModel", Method: "GET", URI: "/configured-audience-model/{configuredAudienceModelArn}"},
	{Name: "GetConfiguredAudienceModelPolicy", Method: "GET", URI: "/configured-audience-model/{configuredAudienceModelArn}/policy"},
	{Name: "GetConfiguredModelAlgorithm", Method: "GET", URI: "/configured-model-algorithms/{configuredModelAlgorithmArn}"},
	{Name: "GetConfiguredModelAlgorithmAssociation", Method: "GET", URI: "/memberships/{membershipIdentifier}/configured-model-algorithm-associations/{configuredModelAlgorithmAssociationArn}"},
	{Name: "GetMLConfiguration", Method: "GET", URI: "/memberships/{membershipIdentifier}/ml-configurations"},
	{Name: "GetMLInputChannel", Method: "GET", URI: "/memberships/{membershipIdentifier}/ml-input-channels/{mlInputChannelArn}"},
	{Name: "GetTrainedModel", Method: "GET", URI: "/memberships/{membershipIdentifier}/trained-models/{trainedModelArn}?versionIdentifier={versionIdentifier}"},
	{Name: "GetTrainedModelInferenceJob", Method: "GET", URI: "/memberships/{membershipIdentifier}/trained-model-inference-jobs/{trainedModelInferenceJobArn}"},
	{Name: "GetTrainingDataset", Method: "GET", URI: "/training-dataset/{trainingDatasetArn}"},
	{Name: "ListAudienceExportJobs", Method: "GET", URI: "/audience-export-job?audienceGenerationJobArn={audienceGenerationJobArn}&maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListAudienceGenerationJobs", Method: "GET", URI: "/audience-generation-job?collaborationId={collaborationId}&configuredAudienceModelArn={configuredAudienceModelArn}&maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListAudienceModels", Method: "GET", URI: "/audience-model?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListCollaborationConfiguredModelAlgorithmAssociations", Method: "GET", URI: "/collaborations/{collaborationIdentifier}/configured-model-algorithm-associations?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListCollaborationMLInputChannels", Method: "GET", URI: "/collaborations/{collaborationIdentifier}/ml-input-channels?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListCollaborationTrainedModelExportJobs", Method: "GET", URI: "/collaborations/{collaborationIdentifier}/trained-models/{trainedModelArn}/export-jobs?maxResults={maxResults}&nextToken={nextToken}&trainedModelVersionIdentifier={trainedModelVersionIdentifier}"},
	{Name: "ListCollaborationTrainedModelInferenceJobs", Method: "GET", URI: "/collaborations/{collaborationIdentifier}/trained-model-inference-jobs?maxResults={maxResults}&nextToken={nextToken}&trainedModelArn={trainedModelArn}&trainedModelVersionIdentifier={trainedModelVersionIdentifier}"},
	{Name: "ListCollaborationTrainedModels", Method: "GET", URI: "/collaborations/{collaborationIdentifier}/trained-models?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListConfiguredAudienceModels", Method: "GET", URI: "/configured-audience-model?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListConfiguredModelAlgorithmAssociations", Method: "GET", URI: "/memberships/{membershipIdentifier}/configured-model-algorithm-associations?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListConfiguredModelAlgorithms", Method: "GET", URI: "/configured-model-algorithms?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListMLInputChannels", Method: "GET", URI: "/memberships/{membershipIdentifier}/ml-input-channels?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListTrainedModelInferenceJobs", Method: "GET", URI: "/memberships/{membershipIdentifier}/trained-model-inference-jobs?maxResults={maxResults}&nextToken={nextToken}&trainedModelArn={trainedModelArn}&trainedModelVersionIdentifier={trainedModelVersionIdentifier}"},
	{Name: "ListTrainedModelVersions", Method: "GET", URI: "/memberships/{membershipIdentifier}/trained-models/{trainedModelArn}/versions?maxResults={maxResults}&nextToken={nextToken}&status={status}"},
	{Name: "ListTrainedModels", Method: "GET", URI: "/memberships/{membershipIdentifier}/trained-models?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTrainingDatasets", Method: "GET", URI: "/training-dataset?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "PutConfiguredAudienceModelPolicy", Method: "PUT", URI: "/configured-audience-model/{configuredAudienceModelArn}/policy"},
	{Name: "PutMLConfiguration", Method: "PUT", URI: "/memberships/{membershipIdentifier}/ml-configurations"},
	{Name: "StartAudienceExportJob", Method: "POST", URI: "/audience-export-job"},
	{Name: "StartAudienceGenerationJob", Method: "POST", URI: "/audience-generation-job"},
	{Name: "StartTrainedModelExportJob", Method: "POST", URI: "/memberships/{membershipIdentifier}/trained-models/{trainedModelArn}/export-jobs"},
	{Name: "StartTrainedModelInferenceJob", Method: "POST", URI: "/memberships/{membershipIdentifier}/trained-model-inference-jobs"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateConfiguredAudienceModel", Method: "PATCH", URI: "/configured-audience-model/{configuredAudienceModelArn}"},
}

var cleanRoomsMLOperationByName = func() map[string]cleanRoomsMLOperation {
	out := make(map[string]cleanRoomsMLOperation, len(cleanRoomsMLOperations))
	for _, op := range cleanRoomsMLOperations {
		out[op.Name] = op
	}
	return out
}()
