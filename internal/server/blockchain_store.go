package server

import (
	"net/url"
	"strings"
	"sync"
	"time"
)

type blockchainStore struct {
	mu sync.Mutex
}

func newBlockchainStore() *blockchainStore {
	return &blockchainStore{}
}

func (s *blockchainStore) Handle(action string, payload map[string]any, _ map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	network := blockchainValueFromPayload(payload,
		"network",
		"tokenFilter.network",
		"contractFilter.network",
		"tokenIdentifier.network",
		"contractIdentifier.network",
	)
	if network == "" {
		network = "ETHEREUM_MAINNET"
	}

	contractAddress := blockchainValueFromPayload(payload,
		"tokenIdentifier.contractAddress",
		"contractIdentifier.contractAddress",
		"tokenFilter.contractAddress",
	)
	if contractAddress == "" {
		contractAddress = "0x0000000000000000000000000000000000000000"
	}

	tokenID := blockchainValueFromPayload(payload,
		"tokenIdentifier.tokenId",
		"tokenFilter.tokenId",
	)
	if tokenID == "" {
		tokenID = "1"
	}

	address := blockchainValueFromPayload(payload,
		"ownerIdentifier.address",
		"ownerFilter.address",
		"address",
		"addressIdentifierFilter.transactionEventToAddress[0]",
	)
	if address == "" {
		address = "0x1111111111111111111111111111111111111111"
	}

	transactionHash := blockchainValueFromPayload(payload, "transactionHash")
	if transactionHash == "" {
		transactionHash = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	}

	transactionID := blockchainValueFromPayload(payload, "transactionId")
	if transactionID == "" {
		transactionID = "stackyard-txid-000001"
	}

	switch action {
	case "BatchGetTokenBalance":
		return map[string]any{
			"tokenBalances": []any{
				blockchainTokenBalancePayload(network, contractAddress, tokenID, address),
			},
			"errors": []any{},
		}

	case "GetAssetContract":
		return blockchainAssetContractPayload(network, contractAddress, address)

	case "GetTokenBalance":
		return blockchainTokenBalancePayload(network, contractAddress, tokenID, address)

	case "GetTransaction":
		return map[string]any{
			"transaction": blockchainTransactionPayload(network, contractAddress, address, transactionHash, transactionID),
		}

	case "ListAssetContracts":
		return map[string]any{
			"contracts": []any{
				map[string]any{
					"contractIdentifier": map[string]any{
						"network":         network,
						"contractAddress": contractAddress,
					},
					"tokenStandard":   "ERC20",
					"deployerAddress": address,
				},
			},
			"nextToken": "",
		}

	case "ListFilteredTransactionEvents", "ListTransactionEvents":
		return map[string]any{
			"events": []any{
				blockchainTransactionEventPayload(network, contractAddress, address, transactionHash, transactionID),
			},
			"nextToken": "",
		}

	case "ListTokenBalances":
		return map[string]any{
			"tokenBalances": []any{
				blockchainTokenBalancePayload(network, contractAddress, tokenID, address),
			},
			"nextToken": "",
		}

	case "ListTransactions":
		return map[string]any{
			"transactions": []any{
				map[string]any{
					"transactionHash":      transactionHash,
					"transactionId":        transactionID,
					"network":              network,
					"transactionTimestamp": time.Now().UTC().Format(time.RFC3339),
					"confirmationStatus":   "FINAL",
				},
			},
			"nextToken": "",
		}
	}

	return map[string]any{}
}

func blockchainAssetContractPayload(network, contractAddress, deployerAddress string) map[string]any {
	return map[string]any{
		"contractIdentifier": map[string]any{
			"network":         network,
			"contractAddress": contractAddress,
		},
		"tokenStandard":   "ERC20",
		"deployerAddress": deployerAddress,
		"metadata": map[string]any{
			"name":     "StackyardToken",
			"symbol":   "STKY",
			"decimals": 18,
		},
	}
}

func blockchainTokenBalancePayload(network, contractAddress, tokenID, ownerAddress string) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{
		"ownerIdentifier": map[string]any{
			"address": ownerAddress,
		},
		"tokenIdentifier": map[string]any{
			"network":         network,
			"contractAddress": contractAddress,
			"tokenId":         tokenID,
		},
		"balance": "1000",
		"atBlockchainInstant": map[string]any{
			"time": now,
		},
		"lastUpdatedTime": map[string]any{
			"time": now,
		},
	}
}

func blockchainTransactionPayload(network, contractAddress, address, transactionHash, transactionID string) map[string]any {
	return map[string]any{
		"network":              network,
		"blockHash":            "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"transactionHash":      transactionHash,
		"blockNumber":          "123456",
		"transactionTimestamp": time.Now().UTC().Format(time.RFC3339),
		"transactionIndex":     0,
		"numberOfTransactions": 1,
		"to":                   address,
		"from":                 "0x2222222222222222222222222222222222222222",
		"contractAddress":      contractAddress,
		"gasUsed":              "21000",
		"cumulativeGasUsed":    "21000",
		"effectiveGasPrice":    "1000000000",
		"signatureV":           1,
		"signatureR":           "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"signatureS":           "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"transactionFee":       "21000000000000",
		"transactionId":        transactionID,
		"confirmationStatus":   "FINAL",
		"executionStatus":      "SUCCESS",
	}
}

func blockchainTransactionEventPayload(network, contractAddress, address, transactionHash, transactionID string) map[string]any {
	return map[string]any{
		"network":                  network,
		"transactionHash":          transactionHash,
		"eventType":                "TRANSFER",
		"from":                     "0x2222222222222222222222222222222222222222",
		"to":                       address,
		"value":                    "10",
		"contractAddress":          contractAddress,
		"tokenId":                  "1",
		"transactionId":            transactionID,
		"voutIndex":                0,
		"voutSpent":                false,
		"spentVoutTransactionId":   "",
		"spentVoutTransactionHash": "",
		"spentVoutIndex":           0,
		"blockchainInstant":        map[string]any{"time": time.Now().UTC().Format(time.RFC3339)},
		"confirmationStatus":       "FINAL",
	}
}

func blockchainValueFromPayload(payload map[string]any, paths ...string) string {
	for _, path := range paths {
		value := blockchainPathValue(payload, path)
		if value != "" {
			return value
		}
	}
	return ""
}

func blockchainPathValue(payload map[string]any, path string) string {
	current := any(payload)
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			return ""
		}

		key := part
		index := -1
		if open := strings.Index(part, "["); open >= 0 && strings.HasSuffix(part, "]") {
			key = strings.TrimSpace(part[:open])
			var parsed int
			for _, ch := range part[open+1 : len(part)-1] {
				if ch < '0' || ch > '9' {
					return ""
				}
				parsed = parsed*10 + int(ch-'0')
			}
			index = parsed
		}

		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[key]
			if !ok {
				return ""
			}
			current = next
		default:
			return ""
		}

		if index >= 0 {
			list, ok := current.([]any)
			if !ok || index < 0 || index >= len(list) {
				return ""
			}
			current = list[index]
		}
	}

	switch typed := current.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}
