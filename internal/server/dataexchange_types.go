package server

type dataExchangeDataType struct {
	Name string
}

// AWS Data Exchange data types sourced from:
// https://docs.aws.amazon.com/data-exchange/latest/apireference/API_Types.html
var dataExchangeDataTypes = []dataExchangeDataType{
	{Name: "Action"},
	{Name: "ApiGatewayApiAsset"},
	{Name: "AssetDestinationEntry"},
	{Name: "AssetDetails"},
	{Name: "AssetEntry"},
	{Name: "AssetSourceEntry"},
	{Name: "AutoExportRevisionDestinationEntry"},
	{Name: "AutoExportRevisionToS3RequestDetails"},
	{Name: "CreateS3DataAccessFromS3BucketRequestDetails"},
	{Name: "CreateS3DataAccessFromS3BucketResponseDetails"},
	{Name: "DatabaseLFTagPolicy"},
	{Name: "DatabaseLFTagPolicyAndPermissions"},
	{Name: "DataGrantSummaryEntry"},
	{Name: "DataSetEntry"},
	{Name: "DataUpdateRequestDetails"},
	{Name: "DeprecationRequestDetails"},
	{Name: "Details"},
	{Name: "Event"},
	{Name: "EventActionEntry"},
	{Name: "ExportAssetsToS3RequestDetails"},
	{Name: "ExportAssetsToS3ResponseDetails"},
	{Name: "ExportAssetToSignedUrlRequestDetails"},
	{Name: "ExportAssetToSignedUrlResponseDetails"},
	{Name: "ExportRevisionsToS3RequestDetails"},
	{Name: "ExportRevisionsToS3ResponseDetails"},
	{Name: "ExportServerSideEncryption"},
	{Name: "ImportAssetFromApiGatewayApiRequestDetails"},
	{Name: "ImportAssetFromApiGatewayApiResponseDetails"},
	{Name: "ImportAssetFromSignedUrlJobErrorDetails"},
	{Name: "ImportAssetFromSignedUrlRequestDetails"},
	{Name: "ImportAssetFromSignedUrlResponseDetails"},
	{Name: "ImportAssetsFromLakeFormationTagPolicyRequestDetails"},
	{Name: "ImportAssetsFromLakeFormationTagPolicyResponseDetails"},
	{Name: "ImportAssetsFromRedshiftDataSharesRequestDetails"},
	{Name: "ImportAssetsFromRedshiftDataSharesResponseDetails"},
	{Name: "ImportAssetsFromS3RequestDetails"},
	{Name: "ImportAssetsFromS3ResponseDetails"},
	{Name: "JobEntry"},
	{Name: "JobError"},
	{Name: "KmsKeyToGrant"},
	{Name: "LakeFormationDataPermissionAsset"},
	{Name: "LakeFormationDataPermissionDetails"},
	{Name: "LakeFormationTagPolicyDetails"},
	{Name: "LFResourceDetails"},
	{Name: "LFTag"},
	{Name: "LFTagPolicyDetails"},
	{Name: "NotificationDetails"},
	{Name: "OriginDetails"},
	{Name: "ReceivedDataGrantSummariesEntry"},
	{Name: "RedshiftDataShareAsset"},
	{Name: "RedshiftDataShareAssetSourceEntry"},
	{Name: "RedshiftDataShareDetails"},
	{Name: "RequestDetails"},
	{Name: "ResponseDetails"},
	{Name: "RevisionDestinationEntry"},
	{Name: "RevisionEntry"},
	{Name: "RevisionPublished"},
	{Name: "S3DataAccessAsset"},
	{Name: "S3DataAccessAssetSourceEntry"},
	{Name: "S3DataAccessDetails"},
	{Name: "S3SnapshotAsset"},
	{Name: "SchemaChangeDetails"},
	{Name: "SchemaChangeRequestDetails"},
	{Name: "ScopeDetails"},
	{Name: "TableLFTagPolicy"},
	{Name: "TableLFTagPolicyAndPermissions"},
	{Name: "UpdateRevision"},
}

var dataExchangeDataTypeByName = func() map[string]dataExchangeDataType {
	out := make(map[string]dataExchangeDataType, len(dataExchangeDataTypes))
	for _, dt := range dataExchangeDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
