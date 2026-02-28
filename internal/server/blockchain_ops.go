package server

type blockchainOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Managed Blockchain Query actions sourced from:
// https://docs.aws.amazon.com/managed-blockchain/latest/AMBQ-APIReference/API_Operations.html
var blockchainOperations = []blockchainOperation{
	{Name: "BatchGetTokenBalance", Method: "POST", URI: "/batch-get-token-balance"},
	{Name: "GetAssetContract", Method: "POST", URI: "/get-asset-contract"},
	{Name: "GetTokenBalance", Method: "POST", URI: "/get-token-balance"},
	{Name: "GetTransaction", Method: "POST", URI: "/get-transaction"},
	{Name: "ListAssetContracts", Method: "POST", URI: "/list-asset-contracts"},
	{Name: "ListFilteredTransactionEvents", Method: "POST", URI: "/list-filtered-transaction-events"},
	{Name: "ListTokenBalances", Method: "POST", URI: "/list-token-balances"},
	{Name: "ListTransactionEvents", Method: "POST", URI: "/list-transaction-events"},
	{Name: "ListTransactions", Method: "POST", URI: "/list-transactions"},
}

var blockchainOperationByName = func() map[string]blockchainOperation {
	out := make(map[string]blockchainOperation, len(blockchainOperations))
	for _, op := range blockchainOperations {
		out[op.Name] = op
	}
	return out
}()
