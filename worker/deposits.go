package worker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"

	"github.com/JokingLove/multichain-sync-account/common/retry"
	"github.com/JokingLove/multichain-sync-account/common/tasks"
	"github.com/JokingLove/multichain-sync-account/config"
	"github.com/JokingLove/multichain-sync-account/database"
	"github.com/JokingLove/multichain-sync-account/rpcclient"
	"github.com/JokingLove/multichain-sync-account/rpcclient/chain-account/account"
)

type Deposit struct {
	BaseSynchronizer

	confirms       uint8
	latestHeader   rpcclient.BlockHeader
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewDeposit(cfg *config.Config, db *database.DB, rpcClient *rpcclient.WalletChainAccountClient, shutdown context.CancelCauseFunc) (*Deposit, error) {
	dbLatestBlockHeader, err := db.Blocks.LatestBlocks()
	if err != nil {
		log.Error("get latest block from database fail")
		return nil, err
	}
	var fromHeader *rpcclient.BlockHeader

	if dbLatestBlockHeader != nil {
		log.Info("sync block", "number", dbLatestBlockHeader.Number, "hash", dbLatestBlockHeader.Hash)
		fromHeader = dbLatestBlockHeader
	} else if cfg.ChainNode.StartingHeight > 0 {
		chainLatestBlockHeader, err := rpcClient.GetBlockHeader(big.NewInt(int64(cfg.ChainNode.StartingHeight)))
		if err != nil {
			log.Error("get block from chain account fail", "err", err)
			return nil, err
		}
		fromHeader = chainLatestBlockHeader
	} else {
		chainLatestBlockHeader, err := rpcClient.GetBlockHeader(nil)
		if err != nil {
			log.Error("get latest block from chain account fail", "err", err)
			return nil, err
		}
		fromHeader = chainLatestBlockHeader
	}

	businessTxChannel := make(chan map[string]*TransactionChannel)

	baseSyncer := BaseSynchronizer{
		loopInterval:     cfg.ChainNode.SynchronizerInterval,
		headerBufferSize: cfg.ChainNode.BlocksStep,
		businessChannels: businessTxChannel,
		rpcClient:        rpcClient,
		blockBatch:       rpcclient.NewBatchBlock(rpcClient, fromHeader, big.NewInt(int64(cfg.ChainNode.BlocksStep))),
		database:         db,
	}

	resCtx, resCancel := context.WithCancel(context.Background())

	return &Deposit{
		BaseSynchronizer: baseSyncer,
		confirms:         uint8(cfg.ChainNode.Confirmations),
		resourceCtx:      resCtx,
		resourceCancel:   resCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutdown(fmt.Errorf("critical error in deposit: %w", err))
			},
		},
	}, nil
}

func (d *Deposit) Close() error {
	var result error
	if err := d.BaseSynchronizer.Close(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to close deposit database synchronizer: %w", err))
	}
	d.resourceCancel()
	if err := d.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to wait deposit batch handler completion: %w", err))
	}
	return result
}

func (d *Deposit) Start() error {
	log.Info("deposit starting")
	if err := d.BaseSynchronizer.Start(); err != nil {
		return fmt.Errorf("failed to start deposit database: %w", err)
	}
	d.tasks.Go(func() error {
		log.Info("handle deposit task start")
		for batch := range d.businessChannels {
			log.Info("deposit business channel", "batch length", len(batch))
			if err := d.handleBatch(batch); err != nil {
				log.Info("failed to handle deposit batch, stopping L2 synchronizer", "err", err)
				return fmt.Errorf("failed to handle batch, stopping L2 Synchronizer: %w", err)
			}
		}
		return nil
	})
	return nil
}

func (d *Deposit) handleBatch(batch map[string]*TransactionChannel) error {
	businessList, err := d.database.Business.QueryBusinessList()
	if err != nil {
		log.Error("query business list fail", "err", err)
		return err
	}

	if businessList == nil || len(businessList) <= 0 {
		err := fmt.Errorf("QueryBusinessList businessList is nil")
		return err
	}

	for _, business := range businessList {
		_, exists := batch[business.BusinessUid]
		if !exists {
			continue
		}

		var (
			transactionFlowList []*database.Transactions
			depositList         []*database.Deposits
			withdrawList        []*database.Withdraws
			internals           []*database.Internals
			balances            []*database.TokenBalance
		)

		log.Info("handle business flow",
			"businessId", business.BusinessUid,
			"chainLatestBlock", batch[business.BusinessUid].BlockHeight,
			"txn", len(batch[business.BusinessUid].Transactions))
		for _, tx := range batch[business.BusinessUid].Transactions {
			log.Info("Request transaction from chain account", "txHash", tx.Hash, "fromAddress", tx.FromAddress)
			txItem, err := d.rpcClient.GetTransactionByHash(tx.Hash)
			if err != nil {
				log.Info("get transaction by hash fail", "err", err)
				return err
			}

			if txItem == nil {
				err := fmt.Errorf("GetTransactionByHash txItem is nil ; txHash :  %s", tx.Hash)
				return err
			}

			amountBigInt, _ := new(big.Int).SetString(txItem.Values[0].Value, 10)
			log.Info("Transaction amount", "amount", amountBigInt, "fromAddress", tx.FromAddress, "toAddress", tx.ToAddress, "TokenAddress", tx.TokenAddress)
			balances = append(balances, &database.TokenBalance{
				FromAddress:  common.HexToAddress(tx.FromAddress),
				ToAddress:    common.HexToAddress(tx.ToAddress),
				TokenAddress: common.HexToAddress(tx.TokenAddress),
				Balance:      amountBigInt,
				TxType:       tx.TxType,
			})

			log.Info("get transaction success", "txHash", txItem.Hash)
			transactionFlow, err := d.BuildTransaction(tx, txItem)
			if err != nil {
				log.Info("handle transaction flow fail", "err", err)
				return err
			}

			transactionFlowList = append(transactionFlowList, transactionFlow)

			switch tx.TxType {
			case database.TxTypeDeposit:
				depositItem, _ := d.HandleDeposit(tx, txItem)
				depositList = append(depositList, depositItem)
				break
			case database.TxTypeWithdraw:
				withdrawItem, _ := d.HandleWithdraw(tx, txItem)
				withdrawList = append(withdrawList, withdrawItem)
				break
			case database.TxTypeCollection, database.TxTypeHot2Cold, database.TxTypeCold2Hot:
				internalItem, _ := d.HandleInternalTx(tx, txItem)
				internals = append(internals, internalItem)
				break
			default:
				break
			}
		}

		retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
		if _, err := retry.Do[interface{}](d.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
			if err := d.database.Transaction(func(tx *database.DB) error {
				if len(depositList) > 0 {
					log.Info("Store deposit transaction", "totalTx", len(depositList))
					if err := tx.Deposits.StoreDeposits(business.BusinessUid, depositList); err != nil {
						log.Error("store deposits fail", "err", err)
						return err
					}

					// update deposit confirms
					if err := tx.Deposits.UpdateDepositsConfirms(business.BusinessUid, batch[business.BusinessUid].BlockHeight, uint64(d.confirms)); err != nil {
						log.Error("handle confirms fail", "err", err)
						return err
					}

					// handle balance
					if len(balances) > 0 {
						log.Info("handle balance into db", "totalTx", len(balances))
						if err := tx.Balances.UpdateOrCreate(business.BusinessUid, balances); err != nil {
							log.Error("handle balances fail", "err", err)
							return err
						}
					}

					// handle withdraw
					if len(withdrawList) > 0 {
						if err := tx.Withdraws.UpdateWithdrawStatusByTxHash(business.BusinessUid, database.TxStatusWalletDone, withdrawList); err != nil {
							log.Error("handle withdraws fail", "err", err)
							return err
						}
					}

					//  handle collection hot 2 cold and cold 2 hot
					if len(internals) > 0 {
						if err := tx.Internals.UpdateInternalStatusByTxHash(business.BusinessUid, database.TxStatusWalletDone, internals); err != nil {
							log.Error("handle internals fail", "err", err)
							return err
						}
					}

					// handle transaction flow
					if len(transactionFlowList) > 0 {
						if err := tx.Trasactions.StoreTransactions(business.BusinessUid, transactionFlowList, uint64(len(transactionFlowList))); err != nil {
							log.Error("store transactions fail", "err", err)
							return err
						}
					}
				}
				return nil
			}); err != nil {
				log.Error("unable to persist batch", "err", err)
				return nil, err
			}
			return nil, nil
		}); err != nil {
			return err
		}

	}
	return nil
}

func (d *Deposit) BuildTransaction(tx *Transaction, txMsg *account.TxMessage) (*database.Transactions, error) {
	txFee, _ := new(big.Int).SetString(txMsg.Fee, 10)
	txAmount, _ := new(big.Int).SetString(txMsg.Values[0].Value, 10)
	transactionTx := &database.Transactions{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		Hash:         common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		Fee:          txFee,
		Status:       txMsg.Status,
		Amount:       txAmount,
		TxType:       tx.TxType,
		Timestamp:    uint64(time.Now().Unix()),
	}
	return transactionTx, nil
}

func (d *Deposit) HandleWithdraw(tx *Transaction, txMsg *account.TxMessage) (*database.Withdraws, error) {
	txAmount, _ := new(big.Int).SetString(txMsg.Values[0].Value, 10)
	withdrawTx := &database.Withdraws{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		TxHash:       common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		MaxFeePerGas: txMsg.Fee,
		Amount:       txAmount,
		Status:       database.TxStatusBoradcasted,
		Timestamp:    uint64(time.Now().Unix()),
	}
	return withdrawTx, nil
}

func (d *Deposit) HandleDeposit(tx *Transaction, txMsg *account.TxMessage) (*database.Deposits, error) {
	txAmount, _ := new(big.Int).SetString(txMsg.Values[0].Value, 10)
	depositTx := &database.Deposits{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		TxHash:       common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		MaxFeePerGas: txMsg.Fee,
		Amount:       txAmount,
		Status:       database.TxStatusBoradcasted,
		Timestamp:    uint64(time.Now().Unix()),
	}
	return depositTx, nil
}

func (d *Deposit) HandleInternalTx(tx *Transaction, txMsg *account.TxMessage) (*database.Internals, error) {
	txAmount, _ := new(big.Int).SetString(txMsg.Values[0].Value, 10)
	internalsTx := &database.Internals{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		TxHash:       common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		MaxFeePerGas: txMsg.Fee,
		Amount:       txAmount,
		Status:       database.TxStatusBoradcasted,
		Timestamp:    uint64(time.Now().Unix()),
	}
	return internalsTx, nil
}
