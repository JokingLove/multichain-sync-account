package database

import (
	"fmt"
	"strings"
)

type TxStatus string

const (
	TxStatusCreateUnsigned TxStatus = "create_unsigned"
	TxStatusSigned         TxStatus = "signed"
	TxStatusBoradcasted    TxStatus = "boradcasted"
	TxStatusWalletDone     TxStatus = "wallet_done"
	TxStatusNotified       TxStatus = "notified"
	TxStatusSuccess        TxStatus = "success"
)

type TokenType string

const (
	TokenTypeBTC     TokenType = "BTC"
	TokenTypeETH     TokenType = "ETH"
	TokenTypeERC20   TokenType = "ERC20"
	TokenTypeERC721  TokenType = "ERC721"
	TokenTypeERC1155 TokenType = "ERC1155"
)

type AddressType string

const (
	AddressTypeEOA  AddressType = "eoa"
	AddressTypeHot  AddressType = "hot"
	AddressTypeCold AddressType = "cold"
)

func (at AddressType) string() string {
	return string(at)
}

func ParseAddressType(s string) (AddressType, error) {
	switch strings.ToLower(s) {
	case string(AddressTypeEOA):
		return AddressTypeEOA, nil
	case string(AddressTypeHot):
		return AddressTypeHot, nil
	case string(AddressTypeCold):
		return AddressTypeCold, nil
	default:
		return AddressTypeEOA, fmt.Errorf("invalid address type: %s", s)

	}
}

type TransactionType string

const (
	TxTypeUnknown    TransactionType = "unknown"
	TxTypeDeposit    TransactionType = "deposit"
	TxTypeWithdraw   TransactionType = "withdraw"
	TxTypeCollection TransactionType = "collection"
	TxTypeHot2Cold   TransactionType = "hot2cold"
	TxTypeCold2Hot   TransactionType = "cold2hot"
)

func (tt TransactionType) string() string {
	return string(tt)
}

func ParseTransactionType(s string) (TransactionType, error) {
	switch strings.ToLower(s) {
	case string(TxTypeDeposit):
		return TxTypeDeposit, nil
	case string(TxTypeWithdraw):
		return TxTypeWithdraw, nil
	case string(TxTypeCollection):
		return TxTypeCollection, nil
	case string(TxTypeHot2Cold):
		return TxTypeHot2Cold, nil
	case string(TxTypeCold2Hot):
		return TxTypeCold2Hot, nil
	default:
		return TxTypeUnknown, fmt.Errorf("unkown transaction type: %s", s)

	}
}

const (
	TableAddressesPrefix    = "addresses_"
	TableTokensPrefix       = "tokens_"
	TableDepositsPrefix     = "deposits_"
	TableWithdrawsPrefix    = "withdraws_"
	TableBusinessPrefix     = "business_"
	TableTransactionsPrefix = "transactions_"
	TableBalancesPrefix     = "balances_"
	TableInternalsPrefix    = "internals_"
)
