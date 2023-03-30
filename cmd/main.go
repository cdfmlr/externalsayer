package main

import (
	"context"
	"flag"
	"log"
	"musayer/macsayer"
	"musayer/macsayer/musayerapi"
	"os"
	"os/signal"
	"syscall"
)

// args: --format=aac --no-clean
var (
	format *string = flag.String("format", "aiff",
		"the format (extension name) of the output audio file. Default is \"aiff\".")
	clean *bool = flag.Bool("clean", false,
		"CleanMode indicates whether to:\n - clean the temporary file when the Say method is returned.\n - clean the temporary directory when the MacSayer is closed. Default is false.")
	grpcaddr *string = flag.String("grpcaddr", "",
		"the address of the gRPC API server. Default is \"\" (disabled).")
)

func main() {
	flag.Parse()

	// MacSayer
	sayer, err := macsayer.NewMacSayer(
		macsayer.WithFormat(*format),
		macsayer.WithClean(*clean))
	if err != nil {
		panic(err)
	}
	defer sayer.Close()

	// gRPC API server
	ctx, cancel := context.WithCancel(context.Background())
	switch {
	case *grpcaddr != "":
		go func() {
			musayerapi.ServeGrpc(ctx, sayer, *grpcaddr)
		}()
	default:
		log.Fatalln("nothing to do! use --grpcaddr to enable gRPC API server.")
	}

	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Receive term signal. Gracefully shutdown...")
		cancel()
	}()

	// wait for shutdown
	<-ctx.Done()
}
