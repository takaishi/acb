package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/takaishi/my_go_cli_example"
)

var Version = "dev"
var Revision = "HEAD"

func init() {
	Version = Version
	Revision = Revision
}

func main() {
	ctx := context.TODO()
	ctx, stop := signal.NotifyContext(ctx, []os.Signal{os.Interrupt}...)
	defer stop()
	if err := my_go_cli_example.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
