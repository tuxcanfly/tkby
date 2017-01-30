package main

import (
	"flag"
	"log"
	"time"

	"github.com/decred/dcrutil"
	pb "github.com/decred/dcrwallet/rpc/walletrpc"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	pass       = flag.String("pass", "", "The wallet passphrase")
	wait       = flag.Int64("wait", 10*60, "Time in seconds to run purchaser")
	tls        = flag.Bool("tls", true, "Connection uses TLS if true, else plain TCP")
	caFile     = flag.String("ca_file", "rpc.cert", "The file containning the CA root cert file")
	serverAddr = flag.String("server_addr", "127.0.0.1:19110", "The server address in the format of host:port")

	accountName   = flag.String("account", "", "Account to purchase from")
	finalBalance  = flag.Float64("balance", 0, "Balance to maintain")
	maxPrice      = flag.Float64("maxprice", 0, "Max ticket price")
	maxFee        = flag.Float64("maxfee", 0, "Max ticket fee")
	ticketAddress = flag.String("address", "", "Ticket address")
)

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	if *tls {
		var sn string
		var creds credentials.TransportCredentials
		if *caFile != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(*caFile, sn)
			if err != nil {
				log.Fatalf("Failed to create TLS credentials %v", err)
			}
		} else {
			creds = credentials.NewClientTLSFromCert(nil, sn)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewTicketBuyerServiceClient(conn)

	log.Printf("Starting ticket buyer")
	_, err = client.StartTicketPurchase(context.Background(), &pb.StartTicketPurchaseRequest{
		Passphrase:        []byte(*pass),
		AccountName:       *accountName,
		BalanceToMaintain: int64(*finalBalance * 1e8),
		MaxFee:            int64(*maxFee * 1e8),
		MaxPriceAbsolute:  int64(*maxPrice * 1e8),
		TicketAddress:     *ticketAddress,
	})
	if err != nil {
		log.Fatalf("rpc err: %v", err)
	}

	log.Printf("fetching config")
	config, err := client.Config(context.Background(), &pb.TicketBuyerConfigRequest{})
	if err != nil {
		log.Fatalf("rpc err: %v", err)
	}
	log.Printf("=== Ticket buyer config ==")
	log.Printf("account: %v", config.AccountName)
	log.Printf("balance to maintain: %v", dcrutil.Amount(config.BalanceToMaintain).ToCoin())
	log.Printf("max fee: %v", dcrutil.Amount(config.MaxFee).ToCoin())
	log.Printf("max per block: %v", config.MaxPerBlock)
	log.Printf("max price absolute: %v", dcrutil.Amount(config.MaxPriceAbsolute).ToCoin())
	log.Printf("max price relative: %v", config.MaxPriceRelative)
	log.Printf("pool address: %v", config.PoolAddress)
	log.Printf("pool fees: %v", config.PoolFees)
	log.Printf("ticket address: %v", config.TicketAddress)

	log.Printf("Purchasing for %vs", *wait)
	time.Sleep(time.Duration(*wait) * time.Second)

	log.Printf("Stopping ticket buyer")
	_, err = client.StopTicketPurchase(context.Background(), &pb.StopTicketPurchaseRequest{})
	if err != nil {
		log.Fatalf("rpc err: %v", err)
	}

}
