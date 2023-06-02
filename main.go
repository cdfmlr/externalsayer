package main

import (
	"context"
	"flag"
	musayerapi "github.com/murchinroom/sayerapigo"
	"log"
	"musayer/externalsayer/azuresayer"
	"os"
	"os/signal"
	"syscall"
)

var (
	configFile = flag.String("c", "", "specify config file (default: check config.yaml in . or /etc/extremesayer or $HOME/.extremesayer)")
	dryRun     = flag.Bool("dryrun", false, "print the config and exit")
)

func main() {
	flag.Parse()

	initConfig(*configFile)

	if *dryRun {
		os.Exit(0)
	}

	gracefulShutdown := make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, os.Interrupt, syscall.SIGTERM)

LOOP:
	for {
		ctx, cancel := context.WithCancel(context.Background())
		go run(ctx)

		select {
		case <-configChanged:
			log.Println("Config changed. Restarting...")
			cancel()
			<-ctx.Done()
			continue LOOP
		case <-gracefulShutdown:
			log.Println("Receive term signal. Gracefully shutdown...")
			cancel()
			<-ctx.Done()
			break LOOP
		}
	}
}

func run(ctx context.Context) {
	var sayer musayerapi.Sayer

	switch config.EnabledSayer {
	case "azure":
		asayer := azuresayer.NewAzureSayer(config.AzureSayer.SpeechKey, config.AzureSayer.SpeechRegion)
		asayer.Mutex.Lock()
		asayer.Roles = config.AzureSayer.Roles
		asayer.FormatMicrosoft = config.AzureSayer.FormatMicrosoft
		asayer.FormatMimeSubtype = config.AzureSayer.FormatMimeSubtype
		asayer.Mutex.Unlock()
		sayer = asayer
	default:
		log.Fatalf("unknown sayer: %s", config.EnabledSayer)
	}

	musayerapi.ServeGrpc(ctx, sayer, config.SrvAddr)
}
