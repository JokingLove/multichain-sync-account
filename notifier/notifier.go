package notifier

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/JokingLove/multichain-sync-account/common/retry"
	"github.com/JokingLove/multichain-sync-account/common/tasks"
	"github.com/JokingLove/multichain-sync-account/database"
)

type Notifier struct {
	db             *database.DB
	businessIds    []string
	notifyClient   map[string]*NotifyClient
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker

	shutdown context.CancelCauseFunc
	stopped  atomic.Bool
}

func NewNotifier(db *database.DB, shutdown context.CancelCauseFunc) (*Notifier, error) {
	businessList, err := db.Business.QueryBusinessList()
	if err != nil {
		log.Error("query business list failed", "err", err)
		return nil, err
	}

	var businessIds []string
	notifyClient := make(map[string]*NotifyClient)
	for _, business := range businessList {
		log.Info("handle business id", "business", business.BusinessUid)
		businessIds = append(businessIds, business.BusinessUid)
		client, err := NewNotifyClient(business.NotifyUrl)
		if err != nil {
			log.Error("new notifier client failed for business : %s", business.BusinessUid, "err", err)
			return nil, err
		}
		notifyClient[business.BusinessUid] = client
	}

	resCtx, resCancel := context.WithCancel(context.Background())
	return &Notifier{
		db:             db,
		notifyClient:   notifyClient,
		businessIds:    businessIds,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutdown(fmt.Errorf("critical error in internals: %w", err))
			},
		},
		ticker: time.NewTicker(time.Second * 5),
	}, nil
}

func (nf *Notifier) Start(ctx context.Context) error {
	log.Info("start notifier......")
	nf.tasks.Go(func() error {
		for {
			select {
			case <-nf.ticker.C:
				return handleNotify(nf)
			case <-nf.resourceCtx.Done():
				log.Info("stop notifier in worker")
				return nil
			}
		}
	})
	return nil
}

func handleNotify(nf *Notifier) error {
	var txn []Transaction
	for _, businessId := range nf.businessIds {
		log.Info("txn and businessId ", "txn", txn, "businessId", businessId)

		// query notify deposits
		needNotifyDeposits, err := nf.db.Deposits.QueryNotifyDeposits(businessId)
		if err != nil {
			log.Error("query notify deposits failed", "err", err)
			return err
		}

		//  query notify withdraw
		needNotifyWithdraws, err := nf.db.Withdraws.QueryNotifyWithdraws(businessId)
		if err != nil {
			log.Error("query notify withdraws failed", "err", err)
			return err
		}

		// query notify internal
		needNotifyInternals, err := nf.db.Internals.QueryNotifyInternals(businessId)
		if err != nil {
			log.Error("query notify internals failed", "err", err)
			return err
		}

		// build notify transaction
		notifyRequest, err := nf.BuildNotifyTransaction(needNotifyDeposits, needNotifyWithdraws, needNotifyInternals)
		if err != nil {
			log.Error("build notify transaction failed", "err", err)
			return err
		}

		// Before Request
		err = nf.BeforeAfterNotify(businessId, true, false, needNotifyDeposits, needNotifyWithdraws, needNotifyInternals)
		if err != nil {
			log.Error("before notify update db status failed", "err", err)
			return err
		}

		// notify
		notify, err := nf.notifyClient[businessId].BusinessNotify(notifyRequest)
		if err != nil {
			log.Error("notify business platform failed", "err", err)
			return err
		}

		// AfterRequest
		err = nf.BeforeAfterNotify(businessId, false, notify, needNotifyDeposits, needNotifyWithdraws, needNotifyInternals)
		if err != nil {
			log.Error("after notify update db status failed", "err", err)
			return err
		}

	}
	return nil
}

func (nf *Notifier) BuildNotifyTransaction(deposits []*database.Deposits, withdraws []*database.Withdraws, internals []*database.Internals) (*NotifyRequest, error) {
	var notifyTransactions []*Transaction

	for _, deposit := range deposits {
		txItem := &Transaction{
			BlockHash:    deposit.BlockHash.String(),
			BlockNumber:  deposit.BlockNumber,
			Hash:         deposit.TxHash.String(),
			FromAddress:  deposit.FromAddress.String(),
			ToAddress:    deposit.ToAddress.String(),
			Value:        deposit.Amount.String(),
			Fee:          deposit.MaxFeePerGas,
			TxType:       deposit.TxType,
			Confirms:     deposit.Confirms,
			TokenAddress: deposit.TokenAddress.String(),
			TokenId:      deposit.TokenId,
			TokenMeta:    deposit.TokenMeta,
		}
		notifyTransactions = append(notifyTransactions, txItem)
	}

	for _, withdraw := range withdraws {
		txItem := &Transaction{
			BlockHash:    withdraw.BlockHash.String(),
			BlockNumber:  withdraw.BlockNumber,
			Hash:         withdraw.TxHash.String(),
			FromAddress:  withdraw.FromAddress.String(),
			ToAddress:    withdraw.ToAddress.String(),
			Value:        withdraw.Amount.String(),
			Fee:          withdraw.MaxFeePerGas,
			TxType:       withdraw.TxType,
			Confirms:     0,
			TokenAddress: withdraw.TokenAddress.String(),
			TokenId:      withdraw.TokenId,
			TokenMeta:    withdraw.TokenMeta,
		}
		notifyTransactions = append(notifyTransactions, txItem)
	}

	for _, internal := range internals {
		txItem := &Transaction{
			BlockHash:    internal.BlockHash.String(),
			BlockNumber:  internal.BlockNumber,
			Hash:         internal.TxHash.String(),
			FromAddress:  internal.FromAddress.String(),
			ToAddress:    internal.ToAddress.String(),
			Value:        internal.Amount.String(),
			Fee:          internal.MaxFeePerGas,
			TxType:       internal.TxType,
			Confirms:     0,
			TokenAddress: internal.TokenAddress.String(),
			TokenId:      internal.TokenId,
			TokenMeta:    internal.TokenMeta,
		}
		notifyTransactions = append(notifyTransactions, txItem)
	}

	notifyReq := &NotifyRequest{
		Txn: notifyTransactions,
	}
	return notifyReq, nil
}

func (nf *Notifier) Stop(ctx context.Context) error {
	var result error
	nf.resourceCancel()
	nf.ticker.Stop()
	if err := nf.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await notify %w", err))
		return result
	}
	log.Info("stop notifier stopped")
	return nil
}

func (nf *Notifier) Stopped() bool {
	return nf.stopped.Load()
}

func (nf *Notifier) BeforeAfterNotify(businessId string, isBefore bool, notifySuccess bool, deposits []*database.Deposits, withdraws []*database.Withdraws, internals []*database.Internals) error {
	var depositsNotifyStatus database.TxStatus
	var withdrawsNotifyStatus database.TxStatus
	var internalsNotifyStatus database.TxStatus
	if isBefore {
		depositsNotifyStatus = database.TxStatusNotified
		withdrawsNotifyStatus = database.TxStatusNotified
		internalsNotifyStatus = database.TxStatusNotified
	} else {
		if notifySuccess {
			depositsNotifyStatus = database.TxStatusSuccess
			withdrawsNotifyStatus = database.TxStatusSuccess
			internalsNotifyStatus = database.TxStatusSuccess
		} else {
			depositsNotifyStatus = database.TxStatusWalletDone
			withdrawsNotifyStatus = database.TxStatusWalletDone
			internalsNotifyStatus = database.TxStatusWalletDone
		}
	}
	// 过滤状态位  0 的交易
	var updateStatusDepositTxn []*database.Deposits

	for _, deposit := range deposits {
		if deposit.Status != database.TxStatusCreateUnsigned {
			updateStatusDepositTxn = append(updateStatusDepositTxn, deposit)
		}
	}

	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	if _, err := retry.Do[interface{}](nf.resourceCtx, 10, retryStrategy, func() (interface{}, error) {

		if err := nf.db.Transaction(func(tx *database.DB) error {
			if len(deposits) > 0 {
				if err := tx.Deposits.UpdateDepositsStatusByTxHash(businessId, depositsNotifyStatus, updateStatusDepositTxn); err != nil {
					return err
				}

				if len(withdraws) > 0 {
					if err := tx.Withdraws.UpdateWithdrawStatusByTxHash(businessId, withdrawsNotifyStatus, withdraws); err != nil {
						return err
					}
				}

				if len(internals) > 0 {
					if err := tx.Internals.UpdateInternalStatusByTxHash(businessId, internalsNotifyStatus, internals); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			log.Error("unable to persist batch  status failed", "err", err)
			return nil, err
		}

		return nil, nil
	}); err != nil {
		return err
	}

	return nil
}
