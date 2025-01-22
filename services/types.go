package services

import "math/big"

type Eip1559DynamicFeeTx struct {
	ChainId              string `json:"chain_id"`
	Nonce                uint64 `json:"nonce"`
	FromAddress          string `json:"from_address"`
	ToAddress            string `json:"to_address"`
	GasLimit             uint64 `json:"gas_limit"`
	MaxFeePerGas         string `json:"max_fee_per_gas"`
	MaxPriorityFeePerGas string `json:"max_priority_fee_per_gas"`

	// eth/erc20 amount
	Amount string `json:"amount"`
	// erc20 erc721 erc1155 contract_address
	ContractAddress string `json:"contract_address"`
}

// FeeInfo 结构体用于存储解析后的费用信息
type FeeInfo struct {
	GasPrice       *big.Int // 基础 gas 价格
	GasTipCap      *big.Int // 小费上限
	Multiplier     int64    // 小费 * 倍数
	MultipliedTip  *big.Int // 小费 * 倍数
	MaxPriorityFee *big.Int // 小费 * 倍数 * 2 （最大上限）
}
