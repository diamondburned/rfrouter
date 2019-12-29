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

	c, err := rfrouter.StartBot(token, &commands,
		func(ctx *rfrouter.Context) error {
			// Set the prefix
			ctx.Prefix = "~"

			// Set the descriptions
			ctx.Name = "rfrouter example"
			ctx.Description = "https://git.sr.ht/~diamondburned/rfrouter"

			// Add the subcommand
			_, err := ctx.RegisterSubcommand(&debug.AーDebug{})
			return err
		},
	)

	if err != nil {
		log.Fatalln("Failed to start the bot:", err)
	}

	// Stop bot on exit
	defer c()

	log.Println("Started bot...")

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
