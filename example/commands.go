package main

import (
	"fmt"

	"git.sr.ht/~diamondburned/rfrouter"
	"github.com/bwmarrin/discordgo"
)

type Commands struct {
	Context     *rfrouter.Context
	HelloCalled int
}

// ~hello
func (c *Commands) Hello(m *discordgo.MessageCreate) error {
	c.HelloCalled++

	_, err := c.Context.Send(m.ChannelID, fmt.Sprintf(
		"Hello, world: %d", c.HelloCalled))
	return err
}
