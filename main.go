package main

import (
	"context"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"time"

	"github.com/YusukeShimizu/c-neutrino/neutrino"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/neutrinorpc"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"
	"github.com/niftynei/glightning/glightning"
)

const MaxFeeMultiple uint64 = 10

var neutrinoC *neutrino.Neutrino

func main() {
	plugin := glightning.NewPlugin(onInit)
	bb := glightning.NewBitcoinBackend(plugin)

	bb.RegisterGetUtxOut(GetUtxOut)
	bb.RegisterGetChainInfo(GetChainInfo)
	bb.RegisterGetFeeRate(GetFeeRate)
	bb.RegisterSendRawTransaction(SendRawTx)
	bb.RegisterGetRawBlockByHeight(BlockByHeight)
	bb.RegisterEstimateFees(EstimateFees)

	plugin.RegisterNewOption("tls-cert-path", "tls cert path", "/tls.cert")
	plugin.RegisterNewOption("macaroon-path", "macaroon path", "/admin.macaroon")
	plugin.RegisterNewOption("grpc-dial", "gRPC port", "localhost:10009")

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func onInit(plugin *glightning.Plugin, options map[string]glightning.Option, config *glightning.Config) {
	// info is set via plugin 'options'
	tlsCertPath, _ := plugin.GetOption("tls-cert-path")
	macaroonPath, _ := plugin.GetOption("macaroon-path")
	grpcDial, _ := plugin.GetOption("grpc-dial")

	var err error
	// default startup
	neutrinoC, err = neutrino.NewNeutrino(tlsCertPath, macaroonPath, grpcDial)
	if err != nil {
		log.Printf(err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	i, err := neutrinoC.LightningClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		log.Printf("error returned: %s", err)
	}
	log.Printf("The node alias is %s:", i.GetAlias())
	log.Printf("successfully init'd! %s %s\n", config.LightningDir, config.RpcFile)
}

func GetUtxOut(txid string, vout uint32) (string, string, error) {
	log.Printf("called getutxo")
	var retout *lnrpc.Utxo
	tnds, err := neutrinoC.LightningClient.ListUnspent(context.Background(), &lnrpc.ListUnspentRequest{})
	if err != nil {
		log.Printf("error returned: %s", err)
		return "", "", err
	}
	for _, t := range tnds.GetUtxos() {
		if t.GetOutpoint().GetTxidStr() == txid && int64(t.GetOutpoint().GetOutputIndex()) == int64(vout) {
			retout = t
		}
	}
	// gettxout sends back an empty if there's nothing found,
	// which is ok, we just need to pass this info along
	if retout == nil {
		return "", "", nil
	}
	log.Printf("txout is %v", retout)
	amt := glightning.NewSat64(uint64(retout.GetAmountSat()))
	return amt.ConvertMsat().String(), retout.GetPkScript(), nil
}

func GetChainInfo() (*glightning.Btc_ChainInfo, error) {
	log.Printf("called getchaininfo")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	i, err := neutrinoC.LightningClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}
	return &glightning.Btc_ChainInfo{
		Chain:                "test",
		HeaderCount:          i.GetBlockHeight(),
		BlockCount:           i.GetBlockHeight(),
		InitialBlockDownload: false,
	}, nil
}

func GetFeeRate(blocks uint32, mode string) (uint64, error) {
	log.Printf("called getfeerate %d %s", blocks, mode)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	f, err := neutrinoC.LightningClient.EstimateFee(ctx, &lnrpc.EstimateFeeRequest{})
	if err != nil {
		return 0, err
	}
	// feerate's response must be denominated in satoshi per kilo-vbyte
	return uint64(f.GetFeeSat()), nil
}

func EstimateFees() (*glightning.Btc_EstimatedFees, error) {
	log.Printf("called estimatefees")
	f, err := neutrinoC.WalletClient.EstimateFee(context.Background(), &walletrpc.EstimateFeeRequest{
		ConfTarget: 20,
	})
	if err != nil {
		return nil, err
	}
	return &glightning.Btc_EstimatedFees{
		Opening:         uint64(f.GetSatPerKw()),
		MutualClose:     uint64(f.GetSatPerKw()),
		UnilateralClose: uint64(f.GetSatPerKw()),
		DelayedToUs:     uint64(f.GetSatPerKw()),
		HtlcResolution:  uint64(f.GetSatPerKw()),
		Penalty:         uint64(f.GetSatPerKw()),
		MinAcceptable:   uint64(f.GetSatPerKw()),
		MaxAcceptable:   uint64(f.GetSatPerKw()) * MaxFeeMultiple,
	}, nil
}

func SendRawTx(tx string) error {
	log.Printf("called SendRawTx")
	b, err := hex.DecodeString(tx)
	if err != nil {
		log.Fatal(err)
	}
	res, err := neutrinoC.WalletClient.PublishTransaction(context.Background(), &walletrpc.Transaction{
		TxHex: b,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("called sendrawtransaction %s(%s)", res.GetPublishError(), err)
	return err
}

// return a blockhash, block, error
func BlockByHeight(height uint32) (string, string, error) {
	log.Printf("called blockbyheight %d", height)

	h, err := neutrinoC.NeutrinoKitClient.GetBlockHash(context.Background(), &neutrinorpc.GetBlockHashRequest{
		Height: int32(height),
	})
	if err != nil {
		log.Println(err)
		return "", "", errors.New("Block height out of range")
	}
	b, err := neutrinoC.NeutrinoKitClient.GetBlock(context.Background(), &neutrinorpc.GetBlockRequest{
		Hash: h.GetHash(),
	})
	if err != nil {
		log.Println(err)
		return "", "", errors.New("Block height out of range")
	}
	return h.GetHash(), hex.EncodeToString(b.GetRawHex()), nil
}
