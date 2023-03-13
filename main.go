package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"HlsToLiveForFFMPEG/server/api"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)
	flag.Parse()
}

var (
	apiport = flag.String("p", "8000", "api server listen port")
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal, 1)
	stop := make(chan bool, 1)

	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for s := range signalCh {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				log.Println("Program Exit...", s)
				cancel()
				stop <- true
				//os.Exit(0)
			default:
				log.Println("signal.Notify default")
				stop <- true
				//os.Exit(0)
			}
		}
	}()

	apierr := api.NewServer(ctx, ":"+*apiport)
	if apierr != nil {
		log.Println("[error]", "API Server :", apierr)
		return
	}

	<-stop

}
