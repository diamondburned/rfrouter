package main

import (
	"log"
	"os"
	"os/signal"

	"git.sr.ht/~diamondburned/rfrouter"
	"git.sr.ht/~diamondburned/rfrouter/example/debug"
)

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("$BOT_TOKEN not given")
	}

	var commands = Commands{
		HelloCalled: 69,
	}

	cancel, err := rfrouter.StartBot(token, &commands, &debug.Debug{})
	if err != nil {
		log.Fatalln(err)
	}

	defer cancel()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
