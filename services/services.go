package services

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/JokingLove/multichain-sync-account/database"
	"github.com/JokingLove/multichain-sync-account/protobuf/da-wallet-go"
	"github.com/JokingLove/multichain-sync-account/rpcclient"
)

const MaxRecvMessageSize = 1024 * 1024 * 300

type BusinessMiddleConfig struct {
	GrpcHostName string
	GrpcPort     int
}

type BusinessMiddleWireServices struct {
	*BusinessMiddleConfig
	da_wallet_go.UnimplementedBusinessMiddleWireServiceServer
	accountClient *rpcclient.WalletChainAccountClient
	db            *database.DB
	stopped       atomic.Bool
}

func NewBusinessMiddleWireServices(db *database.DB, config *BusinessMiddleConfig, accountClient *rpcclient.WalletChainAccountClient) (*BusinessMiddleWireServices, error) {
	return &BusinessMiddleWireServices{
		BusinessMiddleConfig: config,
		accountClient:        accountClient,
		db:                   db,
	}, nil
}

func (bws *BusinessMiddleWireServices) Stop(ctx context.Context) error {
	bws.stopped.Store(true)
	return nil
}

func (bws *BusinessMiddleWireServices) Stopped() bool {
	return bws.stopped.Load()
}

func (bws *BusinessMiddleWireServices) Start(ctx context.Context) error {
	go func(bws *BusinessMiddleWireServices) {
		addr := fmt.Sprintf("%s:%d", bws.GrpcHostName, bws.GrpcPort)
		log.Info("start rpc server ", "addr", addr)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Error("Could not start tcp listener", "err", err)
		}
		grpcServer := grpc.NewServer(
			grpc.MaxRecvMsgSize(MaxRecvMessageSize),
			grpc.ChainUnaryInterceptor(
				nil,
			),
		)

		reflection.Register(grpcServer)

		da_wallet_go.RegisterBusinessMiddleWireServiceServer(grpcServer, bws)

		log.Info("Grpc info", "port", bws.GrpcPort, "address", listener.Addr())
		if err := grpcServer.Serve(listener); err != nil {
			log.Error("Could not GRPC Server")
		}
	}(bws)

	return nil
}
