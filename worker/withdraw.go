package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/JokingLove/multichain-sync-account/common/retry"
	"github.com/JokingLove/multichain-sync-account/common/tasks"
	"github.com/JokingLove/multichain-sync-account/config"
	"github.com/JokingLove/multichain-sync-account/database"
	"github.com/JokingLove/multichain-sync-account/rpcclient"
)

type Withdraw struct {
	rpcClient      *rpcclient.WalletChainAccountClient
	db             *database.DB
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

func NewWithdraw(cfg *config.Config, db *database.DB, rpcClient *rpcclient.WalletChainAccountClient, shutdown context.CancelCauseFunc) (*Withdraw, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Withdraw{
		rpcClient:      rpcClient,
		db:             db,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in withdraw: %w", err))
		}},
		ticker: time.NewTicker(cfg.ChainNode.WorkerInterval),
	}, nil
}

func (w *Withdraw) Close() error {
	var result error
	w.resourceCancel()
	w.ticker.Stop()
	log.Info("stop withdraw ...... ")
	if err := w.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await withdraw: %w", err))
		return result
	}
	log.Info("stop withdraw successfully")
	return nil
}

func (w *Withdraw) Start() error {
	log.Info("start withdraw ...... ")
	w.tasks.Go(func() error {
		for {
			select {
			case <-w.ticker.C:
				businessList, err := w.db.Business.QueryBusinessList()
				if err != nil {
					log.Error("query business list failed", "err", err)
					continue
				}

				for _, business := range businessList {
					unSendTransactionList, err := w.db.Withdraws.UnSendWithdrawList(business.BusinessUid)
					if err != nil {
						log.Error("query un send withdraw list failed", "err", err)
						continue
					}

					if len(unSendTransactionList) == 0 {
						log.Error("Withdraw start", "businessId", business.BusinessUid, "unSendTransactionList", "is null")
						continue
					}

					var balanceList []*database.Balances

					for _, unSendTransaction := range unSendTransactionList {
						txHash, err := w.rpcClient.SendTx(unSendTransaction.TxSignHex)
						if err != nil {
							log.Error("send transaction failed", "err", err)
							continue
						} else {
							balanceItem := &database.Balances{
								Address:      unSendTransaction.FromAddress,
								TokenAddress: unSendTransaction.ToAddress,
								LockBalance:  unSendTransaction.Amount,
							}
							balanceList = append(balanceList, balanceItem)
							unSendTransaction.TxHash = common.HexToHash(txHash)
							unSendTransaction.Status = database.TxStatusBoradcasted
						}
					}

					retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
					if _, err := retry.Do[interface{}](w.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
						if err := w.db.Transaction(func(tx *database.DB) error {
							if len(balanceList) > 0 {
								log.Info("Update address balance", "totalTx", len(balanceList))
								if err := tx.Balances.UpdateBalanceListByTwoAddress(business.BusinessUid, balanceList); err != nil {
									log.Error("Update address balance fail", "err", err)
									return err
								}
							}

							if len(unSendTransactionList) > 0 {
								err = w.db.Withdraws.UpdateWithdrawListById(business.BusinessUid, unSendTransactionList)
								if err != nil {
									log.Error("Update address withdraw status failed", "err", err)
									return err
								}
							}

							return nil
						}); err != nil {
							log.Error("unable to persist batch ", "err", err)
							return nil, err
						}
						return nil, nil
					}); err != nil {
						return err
					}
				}
			case <-w.resourceCtx.Done():
				log.Info("stop withdraw in worker")
				return nil
			}
		}
	})

	return nil
}
