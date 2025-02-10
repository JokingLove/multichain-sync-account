package notifier

import (
	"math/big"

	"github.com/JokingLove/multichain-sync-account/database"
)

type NotifyRequest struct {
	Txn []*Transaction `json:"txn"`
}

type NotifyResponse struct {
	Success bool `json:"success"`
}

type Transaction struct {
	BlockHash    string                   `json:"block_hash"`
	BlockNumber  *big.Int                 `json:"block_number"`
	Hash         string                   `json:"hash"`
	FromAddress  string                   `json:"from_address"`
	ToAddress    string                   `json:"to_address"`
	Value        string                   `json:"value"`
	Fee          string                   `json:"fee"`
	TxType       database.TransactionType `json:"tx_type"`
	Confirms     uint8                    `json:"confirms"`
	TokenAddress string                   `json:"token_address"`
	TokenId      string                   `json:"token_id"`
	TokenMeta    string                   `json:"token_meta"`
}
