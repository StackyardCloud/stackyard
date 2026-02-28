package server

type blockchainDataType struct {
	Name string
}

// Amazon Managed Blockchain Query data types sourced from:
// https://docs.aws.amazon.com/managed-blockchain/latest/AMBQ-APIReference/API_Types.html
var blockchainDataTypes = []blockchainDataType{
	{Name: "AddressIdentifierFilter"},
	{Name: "AssetContract"},
	{Name: "BatchGetTokenBalanceErrorItem"},
	{Name: "BatchGetTokenBalanceInputItem"},
	{Name: "BatchGetTokenBalanceOutputItem"},
	{Name: "BlockchainInstant"},
	{Name: "ConfirmationStatusFilter"},
	{Name: "ContractFilter"},
	{Name: "ContractIdentifier"},
	{Name: "ContractMetadata"},
	{Name: "ListFilteredTransactionEventsSort"},
	{Name: "ListTransactionsSort"},
	{Name: "OwnerFilter"},
	{Name: "OwnerIdentifier"},
	{Name: "TimeFilter"},
	{Name: "TokenBalance"},
	{Name: "TokenFilter"},
	{Name: "TokenIdentifier"},
	{Name: "Transaction"},
	{Name: "TransactionEvent"},
	{Name: "TransactionOutputItem"},
	{Name: "ValidationExceptionField"},
	{Name: "VoutFilter"},
}

var blockchainDataTypeByName = func() map[string]blockchainDataType {
	out := make(map[string]blockchainDataType, len(blockchainDataTypes))
	for _, dt := range blockchainDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
