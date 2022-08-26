package neutrino

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/neutrinorpc"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
)

type Neutrino struct {
	conn              *grpc.ClientConn
	NeutrinoKitClient neutrinorpc.NeutrinoKitClient
	LightningClient   lnrpc.LightningClient
	WalletClient      walletrpc.WalletKitClient
}

func NewNeutrino(tlsCertPath, macaroonPath, grpcDial string) (*Neutrino, error) {
	tlsCreds, err := credentials.NewClientTLSFromFile(tlsCertPath, "")
	if err != nil {
		return nil, err
	}
	macaroonBytes, err := ioutil.ReadFile(macaroonPath)
	if err != nil {
		return nil, err
	}
	mac := &macaroon.Macaroon{}
	mac.UnmarshalBinary(macaroonBytes)
	mc, err := macaroons.NewMacaroonCredential(mac)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
	defer cancel()
	conn, err := grpc.DialContext(ctx, grpcDial, []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCreds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(mc),
	}...)
	if err != nil {
		return nil, err
	}
	return &Neutrino{
		conn:              conn,
		NeutrinoKitClient: neutrinorpc.NewNeutrinoKitClient(conn),
		LightningClient:   lnrpc.NewLightningClient(conn),
		WalletClient:      walletrpc.NewWalletKitClient(conn),
	}, nil
}

func (n Neutrino) Close() error {
	return n.conn.Close()
}
