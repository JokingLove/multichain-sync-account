package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/JokingLove/multichain-sync-account/common/json2"
	"github.com/JokingLove/multichain-sync-account/database"
	"github.com/JokingLove/multichain-sync-account/database/dynamic"
	"github.com/JokingLove/multichain-sync-account/protobuf/da-wallet-go"
	"github.com/JokingLove/multichain-sync-account/rpcclient/chain-account/account"
)

const (
	ChainName = "Ethereum"
	Network   = "mainnet"
)

var (
	EthGasLimit   uint64 = 60000
	TokenGasLimit uint64 = 120000
	Min2Gwei      uint64 = 1000000000
)

func (bws *BusinessMiddleWireServices) BusinessRegister(ctx context.Context, request *da_wallet_go.BusinessRegisterRequest) (*da_wallet_go.BusinessRegisterResponse, error) {
	if request.RequestId == "" || request.NotifyUrl == "" {
		return &da_wallet_go.BusinessRegisterResponse{
			Code: da_wallet_go.ReturnCode_ERROR,
			Msg:  "invalid params",
		}, nil
	}

	business := &database.Business{
		GUID:        uuid.New(),
		BusinessUid: request.RequestId,
		NotifyUrl:   request.NotifyUrl,
		Timestamp:   uint64(time.Now().Unix()),
	}

	err := bws.db.Business.StoreBusiness(business)
	if err != nil {
		log.Error("store business fail", "err", err)
		return &da_wallet_go.BusinessRegisterResponse{
			Code: da_wallet_go.ReturnCode_ERROR,
			Msg:  "store db fail",
		}, nil
	}

	dynamic.CreateTableFromTemplate(request.RequestId, bws.db)
	return &da_wallet_go.BusinessRegisterResponse{
		Code: da_wallet_go.ReturnCode_SUCCESS,
		Msg:  "config business success",
	}, nil
}

func (bws *BusinessMiddleWireServices) ExportAddressesByPublicKeys(ctx context.Context, request *da_wallet_go.ExportAddressesRequest) (*da_wallet_go.ExportAddressesResponse, error) {
	var (
		retAddresses []*da_wallet_go.Address
		dbAddresses  []*database.Addresses
		balances     []*database.Balances
	)

	for _, value := range request.PublicKeys {
		address := bws.accountClient.ExportAddressByPubKey("", value.PublicKey)
		item := &da_wallet_go.Address{
			Type:    value.Type,
			Address: address,
		}
		parseAddressType, err := database.ParseAddressType(value.Type)
		if err != nil {
			log.Error("parse address type fail", "err", err)
			return nil, err
		}

		_, _, balance := bws.accountClient.GetAccount(address)
		dbAddress := &database.Addresses{
			GUID:        uuid.New(),
			Address:     common.HexToAddress(address),
			AddressType: parseAddressType,
			PublicKey:   value.PublicKey,
			Timestamp:   uint64(time.Now().Unix()),
		}
		dbAddresses = append(dbAddresses, dbAddress)

		balanceItem := &database.Balances{
			GUID:         uuid.New(),
			Address:      common.HexToAddress(address),
			AddressType:  parseAddressType,
			TokenAddress: common.Address{},
			Balance:      big.NewInt(int64(balance)),
			LockBalance:  big.NewInt(0),
			Timestamp:    uint64(time.Now().Unix()),
		}

		balances = append(balances, balanceItem)

		retAddresses = append(retAddresses, item)
	}
	err := bws.db.Addresses.StoreAddresses(request.RequestId, dbAddresses)
	if err != nil {
		return &da_wallet_go.ExportAddressesResponse{
			Code: da_wallet_go.ReturnCode_ERROR,
			Msg:  "store address  to db fail",
		}, nil
	}
	err = bws.db.Balances.StoreBalances(request.RequestId, balances)
	if err != nil {
		return &da_wallet_go.ExportAddressesResponse{
			Code: da_wallet_go.ReturnCode_ERROR,
			Msg:  "store balance  to db fail",
		}, nil
	}

	return &da_wallet_go.ExportAddressesResponse{
		Code:      da_wallet_go.ReturnCode_SUCCESS,
		Msg:       "generate address success",
		Addresses: retAddresses,
	}, nil
}
func (bws *BusinessMiddleWireServices) CreateUnSignTransaction(ctx context.Context, request *da_wallet_go.UnSignTransactionRequest) (*da_wallet_go.UnSignTransactionResponse, error) {
	response := &da_wallet_go.UnSignTransactionResponse{
		Code:     da_wallet_go.ReturnCode_ERROR,
		UnSignTx: "0x00",
	}

	if err := validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request : %w", err)
	}

	// transactionType
	transactionType, err := database.ParseTransactionType(request.TxType)
	if err != nil {
		return nil, fmt.Errorf("invalid request TxType: %w", err)
	}

	// value
	amountBig, ok := new(big.Int).SetString(request.Value, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount value: %s", request.Value)
	}

	guid := uuid.New()

	nonce, err := bws.getAccountNonce(ctx, request.From)
	if err != nil {
		return nil, fmt.Errorf("get account nonce fail: %w", err)
	}

	feeInfo, err := bws.getFeeInfo(ctx, request.From)
	if err != nil {
		return nil, fmt.Errorf("get fee info failed: %w", err)
	}

	gasLimit, contractAddress := bws.getGasAndContractInfo(request.ContractAddress)

	switch transactionType {
	case database.TxTypeDeposit:
		err := bws.StoreDeposits(ctx, request, guid, amountBig, gasLimit, feeInfo, transactionType)
		if err != nil {
			return nil, fmt.Errorf("store deposits fail: %w", err)
		}
		break
	case database.TxTypeWithdraw:
	//if err := bws.StoreWithdraw(ctx, request, guid, amountBig, gasLimit, feeInfo, transactionType); err != nil {
	//
	//}
	case database.TxTypeCollection, database.TxTypeHot2Cold, database.TxTypeCold2Hot:
		break
	default:
		response.Msg = "Unsupported transaction type"
		response.UnSignTx = "0x00"
		return response, nil
	}

	dynamicFeeTxReq := Eip1559DynamicFeeTx{
		ChainId:              request.ChainId,
		Nonce:                uint64(nonce),
		FromAddress:          request.From,
		ToAddress:            request.To,
		GasLimit:             gasLimit,
		MaxFeePerGas:         feeInfo.MaxPriorityFee.String(),
		MaxPriorityFeePerGas: feeInfo.MultipliedTip.String(),
		Amount:               request.Value,
		ContractAddress:      contractAddress,
	}

	data := json2.ToJSON(dynamicFeeTxReq)
	log.Info("BusinessMiddleWireServices CreateUnSignTransaction dynamicFeeTxReq", json2.ToJSONString(dynamicFeeTxReq))
	base64Str := base64.StdEncoding.EncodeToString(data)
	unsignTx := &account.UnSignTransactionRequest{
		Chain:    ChainName,
		Network:  Network,
		Base64Tx: base64Str,
	}
	log.Info("BusinessMiddleWireServices CreateUnSignTransaction unsignTx", json2.ToJSONString(unsignTx))
	returnTx, err := bws.accountClient.AccountRpcClient.CreateUnSignTransaction(ctx, unsignTx)
	log.Info("BusinessMiddleWireServices CreateUnSignTransaction returnTx", json2.ToJSONString(returnTx))
	if err != nil {
		log.Error("create un sign transaction fail: %w", err)
		return nil, fmt.Errorf("create un sign transaction fail: %w", err)
	}

	response.Code = da_wallet_go.ReturnCode_SUCCESS
	response.Msg = "submit withdraw and build un sign transaction success"
	response.TransactionId = guid.String()
	response.UnSignTx = returnTx.UnSignTx
	return response, nil
}
func (bws *BusinessMiddleWireServices) BuildSignedTransaction(ctx context.Context, request *da_wallet_go.SignTransactionRequest) (*da_wallet_go.SignTransactionResponse, error) {
	response := &da_wallet_go.SignTransactionResponse{
		Code: da_wallet_go.ReturnCode_ERROR,
	}
	// 1. Get transaction from database based on type
	var (
		fromAddress          string
		toAddress            string
		amount               string
		tokenAddress         string
		gasLimit             uint64
		maxFeePerGas         string
		maxPriorityFeePerGas string
	)

	transactionType, err := database.ParseTransactionType(request.TxType)
	if err != nil {
		return nil, fmt.Errorf("invalid request TxType: %w", err)
	}

	switch transactionType {
	case database.TxTypeDeposit:
		tx, err := bws.db.Deposits.QueryDepositsById(request.RequestId, request.TransactionId)
		if err != nil {
			return nil, fmt.Errorf("query deposits fail: %w", err)
		}
		if tx == nil {
			response.Msg = "Deposit transaction not found"
			return response, nil
		}
		fromAddress = tx.FromAddress.String()
		toAddress = tx.ToAddress.String()
		amount = tx.Amount.String()
		tokenAddress = tx.TokenAddress.String()
		gasLimit = tx.GasLimit
		maxFeePerGas = tx.MaxFeePerGas
		maxPriorityFeePerGas = tx.MaxPriorityFeeGas
	case database.TxTypeWithdraw:
	case database.TxTypeCollection, database.TxTypeHot2Cold, database.TxTypeCold2Hot:
	default:
		response.Msg = "unsupported transaction type"
		response.SignTx = "0x00"
		return response, nil
	}

	// 2.Get current nonce
	nonce, err := bws.getAccountNonce(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("get account nonce fail: %w", err)
	}

	// 3. Build EIP-1559 transaction
	dynamicFeeTx := Eip1559DynamicFeeTx{
		ChainId:              request.ChainId,
		Nonce:                uint64(nonce),
		FromAddress:          fromAddress,
		ToAddress:            toAddress,
		GasLimit:             gasLimit,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		Amount:               amount,
		ContractAddress:      tokenAddress,
	}

	// 4. Build signed transaction
	data := json2.ToJSON(dynamicFeeTx)
	base64Str := base64.StdEncoding.EncodeToString(data)
	signedTxReq := &account.SignedTransactionRequest{
		Chain:     ChainName,
		Network:   Network,
		Signature: request.Signature,
		Base64Tx:  base64Str,
	}

	log.Info("BuildSignedTransaction request ", "dynamicFeeTx", json2.ToJSONString(signedTxReq))
	returnTx, err := bws.accountClient.AccountRpcClient.BuildSignedTransaction(ctx, signedTxReq)
	log.Info("BuildSignedTransaction returnTx", json2.ToJSONString(returnTx))
	if err != nil {
		return nil, fmt.Errorf("build signed transaction fail: %w", err)
	}

	// 5. Update transaction status in database
	var updateErr error
	switch transactionType {
	case database.TxTypeDeposit:
		updateErr = bws.db.Deposits.UpdateDepositById(request.RequestId, request.TransactionId, returnTx.SignedTx, database.TxStatusSigned)
	case database.TxTypeWithdraw:
	case database.TxTypeCollection, database.TxTypeHot2Cold, database.TxTypeCold2Hot:
	default:
		response.Msg = "unsupported transaction type"
		response.SignTx = "0x00"
		return response, nil
	}

	if updateErr != nil {
		return nil, fmt.Errorf("update transaction status failed: %w", updateErr)
	}

	response.Code = da_wallet_go.ReturnCode_SUCCESS
	response.Msg = "build signed transaction success"
	response.SignTx = returnTx.SignedTx
	return response, nil
}
func (bws *BusinessMiddleWireServices) getGasAndContractInfo(contractAddress string) (uint64, string) {
	if contractAddress == "0x00" {
		return EthGasLimit, "0x00"
	}
	return TokenGasLimit, contractAddress
}

func (bws *BusinessMiddleWireServices) getAccountNonce(ctx context.Context, address string) (int, error) {
	accountReq := &account.AccountRequest{
		Chain:           ChainName,
		Network:         Network,
		Address:         address,
		ContractAddress: "0x00",
	}

	accountInfo, err := bws.accountClient.AccountRpcClient.GetAccount(ctx, accountReq)
	if err != nil {
		return 0, fmt.Errorf("get account info fail: %w", err)
	}

	return strconv.Atoi(accountInfo.Sequence)
}

func (bws *BusinessMiddleWireServices) getFeeInfo(ctx context.Context, address string) (*FeeInfo, error) {
	feeReq := &account.FeeRequest{
		Chain:   ChainName,
		Network: Network,
		Address: address,
		RawTx:   "",
	}

	feeResponse, err := bws.accountClient.AccountRpcClient.GetFee(ctx, feeReq)
	if err != nil {
		return nil, fmt.Errorf("get fee info fail: %w", err)
	}

	return ParseFastFee(feeResponse.FastFee)
}

// ParseFastFee 解析 FastFee 字符串并计算相关费用
func ParseFastFee(fee string) (*FeeInfo, error) {
	// 1. 按 “|" 分隔字符串
	parts := strings.Split(fee, "|")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid fee format: %s", fee)
	}
	// 2. 解析 GasPrice （baseFee)
	gasPrice := new(big.Int)
	if _, ok := gasPrice.SetString(parts[0], 10); !ok {
		return nil, fmt.Errorf("invalid gas price: %s", parts[0])
	}

	// 3. 解析  GasTipCap
	gasTipCap := new(big.Int)
	if _, ok := gasTipCap.SetString(parts[1], 10); !ok {
		return nil, fmt.Errorf("invalid gas tip cap: %s", parts[1])
	}

	// 4. 解析倍数 （去掉 * 前缀）
	multiplierStr := strings.TrimPrefix(parts[2], "*")
	multiplier, err := strconv.ParseInt(multiplierStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid multiplier: %s", parts[2])
	}

	// 5.计算 MultipliedTip (小费 * 倍数）
	multipliedTip := new(big.Int).Mul(gasTipCap, big.NewInt(multiplier))

	// 设置最小小费阀值 （1 Gwei）
	//minTipCap := big.NewInt(int64(Min1Gwei))
	//if multipliedTip.Cmp(minTipCap) < 0 {
	//	multipliedTip = minTipCap
	//}

	// 6. 计算 MaxPriorityFee (baseFee + 小费 * 倍数 * 2）
	maxPriorityFee := new(big.Int).Mul(
		multipliedTip,
		big.NewInt(2),
	)

	// 加上 baseFee
	maxPriorityFee.Add(gasPrice, maxPriorityFee)

	return &FeeInfo{
		GasPrice:       gasPrice,
		GasTipCap:      gasTipCap,
		Multiplier:     multiplier,
		MultipliedTip:  multipliedTip,
		MaxPriorityFee: maxPriorityFee,
	}, nil

}

func (bws *BusinessMiddleWireServices) SetTokenAddress(ctx context.Context, request *da_wallet_go.SetTokenAddressRequest) (*da_wallet_go.SetTokenAddressResponse, error) {
	var (
		tokenList []database.Tokens
	)
	for _, value := range request.TokenList {
		collectAmountBigInt, _ := new(big.Int).SetString(value.CollectAmount, 10)
		coldAmountBigInt, _ := new(big.Int).SetString(value.ColdAmount, 10)
		token := database.Tokens{
			GUID:          uuid.New(),
			TokenAddress:  common.HexToAddress(value.Address),
			Decimals:      uint8(value.Decimals),
			TokenName:     value.TokenName,
			CollectAmount: collectAmountBigInt,
			ColdAmount:    coldAmountBigInt,
			TimeStamp:     uint64(time.Now().Unix()),
		}
		tokenList = append(tokenList, token)
	}

	err := bws.db.Tokens.StoreTokens(request.RequestId, tokenList)
	if err != nil {
		log.Error("set token address fail", "err", err)
		return nil, err
	}
	return &da_wallet_go.SetTokenAddressResponse{
		Code: da_wallet_go.ReturnCode_SUCCESS,
		Msg:  "set token address success",
	}, nil
}

func validateRequest(request *da_wallet_go.UnSignTransactionRequest) error {
	if request == nil {
		return errors.New("request cannot be nil")
	}
	if request.From == "" {
		return errors.New("from address cannot be empty")
	}
	if request.To == "" {
		return errors.New("to address cannot be empty")
	}
	if request.Value == "" {
		return errors.New("value cannot be empty")
	}
	return nil
}

func (bws *BusinessMiddleWireServices) StoreDeposits(
	ctx context.Context,
	depositsRequest *da_wallet_go.UnSignTransactionRequest,
	transactionId uuid.UUID,
	amountBig *big.Int,
	gasLimit uint64,
	feeInfo *FeeInfo,
	transactionType database.TransactionType,
) error {
	dbDeposit := &database.Deposits{
		GUID:              transactionId,
		Timestamp:         uint64(time.Now().Unix()),
		Status:            database.TxStatusCreateUnsigned,
		Confirms:          0,
		BlockHash:         common.Hash{},
		BlockNumber:       big.NewInt(1),
		TxHash:            common.Hash{},
		TxType:            transactionType,
		FromAddress:       common.HexToAddress(depositsRequest.From),
		ToAddress:         common.HexToAddress(depositsRequest.To),
		Amount:            amountBig,
		GasLimit:          gasLimit,
		MaxFeePerGas:      feeInfo.MaxPriorityFee.String(),
		MaxPriorityFeeGas: feeInfo.MultipliedTip.String(),
		TokenType:         determineTokenType(depositsRequest.ContractAddress),
		TokenAddress:      common.HexToAddress(depositsRequest.ContractAddress),
		TokenId:           depositsRequest.TokenId,
		TokenMeta:         depositsRequest.TokenMeta,
		TxSignHex:         "",
	}
	return bws.db.Deposits.StoreDeposits(depositsRequest.RequestId, []*database.Deposits{dbDeposit})
}

func (bws *BusinessMiddleWireServices) storeWithdraw(
	request *da_wallet_go.UnSignTransactionRequest,
	transactionId uuid.UUID,
	amountBig *big.Int,
	gasLimit uint64,
	feeInfo *FeeInfo,
	transactionType database.TransactionType,
) error {

	withdraw := &database.With
}

func determineTokenType(contractAddress string) database.TokenType {
	if contractAddress == "0x00" {
		return database.TokenTypeETH
	}
	// 这里可以添加更多的 token 类型判断逻辑
	return database.TokenTypeERC20
}
