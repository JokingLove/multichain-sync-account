package worker

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"

	"github.com/JokingLove/multichain-sync-account/common/clock"
	"github.com/JokingLove/multichain-sync-account/database"
	"github.com/JokingLove/multichain-sync-account/rpcclient"
)

type Transaction struct {
	BusinessId     string
	BlockNumber    *big.Int
	FromAddress    string
	ToAddress      string
	Hash           string
	TokenAddress   string
	ContractWallet string
	TxType         database.TransactionType
}

type Config struct {
	LoopIntervalMsec int
	HeaderBufferSize int
	StartHeight      *big.Int
	Confirmations    uint64
}

type BaseSynchronizer struct {
	loopInterval     time.Duration
	headerBufferSize uint64

	businessChannels chan map[string]*TransactionChannel

	rpcClient  *rpcclient.WalletChainAccountClient
	blockBatch *rpcclient.BatchBlock
	database   *database.DB

	headers []rpcclient.BlockHeader
	worker  *clock.LoopFn
}

type TransactionChannel struct {
	BlockHeight  uint64
	ChannelId    string
	Transactions []*Transaction
}

func (syncer *BaseSynchronizer) Start() error {
	if syncer.worker != nil {
		return errors.New("worker is already started")
	}
	syncer.worker = clock.NewLoopFn(clock.SystemClock, syncer.tick, func() error {
		log.Info("shutting down batch producer")
		close(syncer.businessChannels)
		return nil
	}, syncer.loopInterval)
	return nil
}

func (syncer *BaseSynchronizer) Close() error {
	if syncer.worker != nil {
		return nil
	}
	return syncer.worker.Close()
}

func (syncer *BaseSynchronizer) tick(_ context.Context) {
	if len(syncer.headers) > 0 {
		log.Info("retrying previous batch")
	} else {
		newHeaders, err := syncer.blockBatch.NextHeaders(syncer.headerBufferSize)
		if err != nil {
			log.Error("error querying for headers", "err", err)
		} else if len(newHeaders) > 0 {
			log.Warn("no new headers, syncer at head ?")
		} else {
			syncer.headers = newHeaders
		}
	}
	err := syncer.processBatch(syncer.headers)
	if err != nil {
		syncer.headers = nil
	}
}

func (syncer *BaseSynchronizer) processBatch(headers []rpcclient.BlockHeader) error {
	if len(headers) == 0 {
		return nil
	}

	businessTxChannel := make(map[string]*TransactionChannel)
	blockHeaders := make([]database.Blocks, len(headers))

	for i := range headers {
		log.Info("Sync block data", "height", headers[i].Number)
		blockHeaders[i] = database.Blocks{
			Hash:       headers[i].Hash,
			ParentHash: headers[i].ParentHash,
			Number:     headers[i].Number,
			Timestamp:  headers[i].Timestamp,
		}
		txList, err := syncer.rpcClient.GetBlockInfo(headers[i].Number)
		if err != nil {
			log.Error(" get block info failed", "err", err)
			return err
		}

		businessList, err := syncer.database.Business.QueryBusinessList()
		if err != nil {
			log.Error(" query business list failed", "err", err)
			return err
		}

		for _, business := range businessList {
			var businessTransactions []*Transaction
			for _, tx := range txList {
				toAddress := common.HexToAddress(tx.To)
				fromAddress := common.HexToAddress(tx.From)
				existToAddress, toAddressType := syncer.database.Addresses.AddressExists(business.BusinessUid, &toAddress)
				existFromAddress, fromAddressType := syncer.database.Addresses.AddressExists(business.BusinessUid, &fromAddress)
				if !existToAddress && !existFromAddress {
					continue
				}

				log.Info("Found transaction ", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
				txItem := &Transaction{
					BusinessId:     business.BusinessUid,
					BlockNumber:    headers[i].Number,
					FromAddress:    tx.From,
					ToAddress:      tx.To,
					Hash:           tx.Hash,
					TokenAddress:   tx.TokenAddress,
					ContractWallet: tx.ContractWallet,
					TxType:         database.TxTypeUnknown,
				}

				/*
				 * If the 'from' address is an external address and the 'to' address is an internal user address, it is a deposit; call the callback interface to notifier the business side.
				 * If the 'from' address is a user address and the 'to' address is a hot wallet address, it is consolidation; call the callback interface to notifier the business side.
				 * If the 'from' address is a hot wallet address and the 'to' address is an external user address, it is a withdrawal; call the callback interface to notifier the business side.
				 * If the 'from' address is a hot wallet address and the 'to' address is a cold wallet address, it is a hot-to-cold transfer; call the callback interface to notifier the business side.
				 * If the 'from' address is a cold wallet address and the 'to' address is a hot wallet address, it is a cold-to-hot transfer; call the callback interface to notifier the business side.
				 */
				if !existFromAddress && (existToAddress && toAddressType == database.AddressTypeEOA) { // 充值
					log.Info("Found deposit transaction ", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = database.TxTypeDeposit
				}
				// 提现
				if (existFromAddress && fromAddressType == database.AddressTypeHot) && !existToAddress {
					log.Info("Found withdraw transaction ", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = database.TxTypeWithdraw
				}

				// 归集
				if (existFromAddress && fromAddressType == database.AddressTypeEOA) && (existToAddress && toAddressType == database.AddressTypeHot) && !existToAddress {
					log.Info("Found collection transaction ", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = database.TxTypeCollection
				}
				// 热转冷
				if (existFromAddress && fromAddressType == database.AddressTypeHot) && (existToAddress && toAddressType == database.AddressTypeCold) && !existToAddress {
					log.Info("Found hot2cold transaction ", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = database.TxTypeHot2Cold
				}
				// 冷转热
				if (existFromAddress && fromAddressType == database.AddressTypeCold) && (existToAddress && toAddressType == database.AddressTypeHot) && !existToAddress {
					log.Info("Found cold2hot transaction ", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = database.TxTypeCold2Hot
				}

				businessTransactions = append(businessTransactions, txItem)
			}

			if len(businessTransactions) > 0 {
				if businessTxChannel[business.BusinessUid] == nil {
					businessTxChannel[business.BusinessUid] = &TransactionChannel{
						BlockHeight:  headers[i].Number.Uint64(),
						Transactions: businessTransactions,
					}
				} else {
					businessTxChannel[business.BusinessUid].BlockHeight = headers[i].Number.Uint64()
					businessTxChannel[business.BusinessUid].Transactions = append(businessTxChannel[business.BusinessUid].Transactions, businessTransactions...)
				}
			}
		}
	}

	if len(blockHeaders) > 0 {
		log.Info("Store block headers success", "totalBlockHeader", len(blockHeaders))
		if err := syncer.database.Blocks.StoreBlocks(blockHeaders); err != nil {
			return err
		}
	}

	log.Info("business tx channel", "businessTxChannel", businessTxChannel, "map length", len(businessTxChannel))
	if len(businessTxChannel) > 0 {
		syncer.businessChannels <- businessTxChannel
	}

	return nil
}
