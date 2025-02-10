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

type Internal struct {
	rpcClient      *rpcclient.WalletChainAccountClient
	db             *database.DB
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

func NewInternal(cfg *config.Config,
	db *database.DB,
	rpcClient *rpcclient.WalletChainAccountClient,
	shutdown context.CancelCauseFunc) (*Internal, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Internal{
		rpcClient:      rpcClient,
		db:             db,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in internals : %w", err))
		}},
		ticker: time.NewTicker(cfg.ChainNode.WorkerInterval),
	}, nil
}

func (i *Internal) Close() error {
	var result error
	i.resourceCancel()
	i.ticker.Stop()
	log.Info("stop internal......")
	if err := i.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await internal : %w", err))
		return result
	}
	log.Info("stop internal success")
	return nil
}

func (i *Internal) Start() error {
	log.Info("starting internal...")
	i.tasks.Go(func() error {
		for {
			select {
			case <-i.ticker.C:
				log.Info("collection and hot to cold")
				businessList, err := i.db.Business.QueryBusinessList()
				if err != nil {
					log.Error("query business list fail: ", "err", err)
					continue
				}

				for _, business := range businessList {
					unSendInternalList, err := i.db.Internals.UnSendInternalList(business.BusinessUid)
					if err != nil {
						log.Error("query un send internal list fail: ", "err", err)
						continue
					}
					if len(unSendInternalList) == 0 {
						log.Info("internal query un send list is null", "business", business.BusinessUid)
						continue
					}

					var balanceList []*database.Balances

					for _, unSendInternalTx := range unSendInternalList {
						txHash, err := i.rpcClient.SendTx(unSendInternalTx.TxSignHex)
						if err != nil {
							log.Error("send internal tx fail: ", "err", err)
							continue
						} else {
							balanceItem := &database.Balances{
								TokenAddress: unSendInternalTx.TokenAddress,
								Address:      unSendInternalTx.FromAddress,
								LockBalance:  unSendInternalTx.Amount,
							}
							balanceList = append(balanceList, balanceItem)

							unSendInternalTx.TxHash = common.HexToHash(txHash)
							unSendInternalTx.Status = database.TxStatusBoradcasted
						}
					}

					retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
					if _, err := retry.Do[interface{}](i.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
						if err := i.db.Transaction(func(tx *database.DB) error {
							if len(balanceList) > 0 {
								log.Info("Update address balance", "totalTx", len(balanceList))
								if err := tx.Balances.UpdateBalanceListByTwoAddress(business.BusinessUid, balanceList); err != nil {
									log.Error("Update address balance fail", "err", err)
									return err
								}

								if len(unSendInternalList) > 0 {
									err = i.db.Internals.UpdateInternalListById(business.BusinessUid, unSendInternalList)
									if err != nil {
										log.Error("update internals status fail", "err", err)
										return err
									}
								}
							}
							return nil
						}); err != nil {
							return nil, err
						}

						return nil, nil
					}); err != nil {
						return err
					}
				}
			case <-i.resourceCtx.Done():
				log.Info("stop internals in worker")
				return nil
			}
		}
	})
	return nil
}
