package rpcclient

import (
	"context"
	"math/big"
	"strconv"

	common2 "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	
	"github.com/JokingLove/multichain-sync-account/rpcclient/chain-account/account"
	"github.com/JokingLove/multichain-sync-account/rpcclient/chain-account/common"
)

type WalletChainAccountClient struct {
	Ctx              context.Context
	ChainName        string
	AccountRpcClient account.WalletAccountServiceClient
}

func NewWalletChainAccountClient(ctx context.Context, rcp account.WalletAccountServiceClient, chainName string) (*WalletChainAccountClient, error) {
	log.Info("New account chain rpc client", "chainName", chainName)
	return &WalletChainAccountClient{Ctx: ctx, ChainName: chainName, AccountRpcClient: rcp}, nil
}

func (wac *WalletChainAccountClient) ExportAddressByPubKey(typeOrVersion, publicKey string) string {
	req := &account.ConvertAddressRequest{
		Chain:     wac.ChainName,
		Type:      typeOrVersion,
		PublicKey: publicKey,
	}

	address, err := wac.AccountRpcClient.ConvertAddress(wac.Ctx, req)
	if err != nil {
		log.Error("convert address failed", "err", err)
		return ""
	}
	if address.Code == common.ReturnCode_ERROR {
		log.Error("convert address failed", "err", err)
		return ""
	}
	return address.Address
}

func (wac *WalletChainAccountClient) GetBlockHeader(number *big.Int) (*BlockHeader, error) {
	var height int64
	if number == nil {
		height = 0
	} else {
		height = number.Int64()
	}

	req := &account.BlockHeaderNumberRequest{
		Chain:   wac.ChainName,
		Network: "mainnet",
		Height:  height,
	}

	blockHeader, err := wac.AccountRpcClient.GetBlockHeaderByNumber(wac.Ctx, req)
	if err != nil {
		log.Error("get latest block GetBlockHeaderByNumber failed", "err", err)
		return nil, err
	}
	if blockHeader.Code == common.ReturnCode_ERROR {
		log.Error("get latest block fail", "err", err)
		return nil, err
	}
	blockNumber, _ := new(big.Int).SetString(blockHeader.BlockHeader.Number, 10)
	return &BlockHeader{
		Hash:       common2.HexToHash(blockHeader.BlockHeader.Hash),
		ParentHash: common2.HexToHash(blockHeader.BlockHeader.ParentHash),
		Number:     blockNumber,
		Timestamp:  blockHeader.BlockHeader.Time,
	}, nil
}

func (wac *WalletChainAccountClient) GetBlockInfo(blockNumber *big.Int) ([]*account.BlockInfoTransactionList, error) {
	req := &account.BlockNumberRequest{
		Chain:  wac.ChainName,
		Height: blockNumber.Int64(),
		ViewTx: true,
	}
	blockInfo, err := wac.AccountRpcClient.GetBlockByNumber(wac.Ctx, req)
	if err != nil {
		log.Error("get latest block GetBlockByNumber failed", "err", err)
		return nil, err
	}
	if blockInfo.Code == common.ReturnCode_ERROR {
		log.Error("get latest block fail", "err", err)
		return nil, err
	}

	return blockInfo.Transactions, nil
}

func (wac *WalletChainAccountClient) GetTransactionByHash(hash string) (*account.TxMessage, error) {
	req := &account.TxHashRequest{
		Chain:   wac.ChainName,
		Hash:    hash,
		Network: "mainnet",
	}
	tx, err := wac.AccountRpcClient.GetTxByHash(wac.Ctx, req)
	if err != nil {
		log.Error("get latest block GetTxByHash failed", "err", err)
		return nil, err
	}
	if tx.Code == common.ReturnCode_ERROR {
		log.Error("get latest block fail", "err", err)
		return nil, err
	}
	return tx.Tx, nil
}

func (wac *WalletChainAccountClient) GetAccountAccountNumber(address string) (int, error) {
	req := &account.AccountRequest{
		Chain:   wac.ChainName,
		Address: address,
		Network: "mainnet",
	}
	account, err := wac.AccountRpcClient.GetAccount(wac.Ctx, req)
	if err != nil {
		log.Error("get latest block GetAccountAccount failed", "err", err)
		return 0, err
	}
	if account.Code == common.ReturnCode_ERROR {
		log.Error("get latest block fail", "err", err)
		return 0, err
	}
	return strconv.Atoi(account.AccountNumber)
}

func (wac *WalletChainAccountClient) GetAccount(address string) (int, int, int) {
	req := &account.AccountRequest{
		Chain:           wac.ChainName,
		Address:         address,
		Network:         "mainnet",
		ContractAddress: "0x00",
	}
	account, err := wac.AccountRpcClient.GetAccount(wac.Ctx, req)
	if err != nil {
		log.Info("get  account info GetAccountAccount failed", "err", err)
		return 0, 0, 0
	}

	if account.Code == common.ReturnCode_ERROR {
		log.Info("get account info fail", "err", err)
		return 0, 0, 0
	}

	accountNumber, err := strconv.Atoi(account.AccountNumber)
	if err != nil {
		log.Error("convert account account number to int failed", "err", err)
		return 0, 0, 0
	}

	sequence, err := strconv.Atoi(account.Sequence)
	if err != nil {
		log.Error("convert account account sequence to int failed", "err", err)
		return 0, 0, 0
	}

	balance, err := strconv.Atoi(account.Balance)
	if err != nil {
		log.Error("convert account account balance to int failed", "err", err)
		return 0, 0, 0
	}
	return accountNumber, sequence, balance
}

func (wac *WalletChainAccountClient) SendTx(rawTx string) (string, error) {
	log.Info("send tx", "rawTx", rawTx, "ChainName", wac.ChainName)
	req := &account.SendTxRequest{
		Chain:   wac.ChainName,
		RawTx:   rawTx,
		Network: "mainnet",
	}

	txInfo, err := wac.AccountRpcClient.SendTx(wac.Ctx, req)
	if err != nil {
		log.Error("send tx failed", "err", err)
		return "", err
	}
	if txInfo == nil {
		log.Error("send tx failed , txInfo is null", "err", err)
		return "", err
	}
	if txInfo.Code == common.ReturnCode_ERROR {
		log.Error("send tx failed", "err", err)
		return "", err
	}
	return txInfo.TxHash, nil
}
