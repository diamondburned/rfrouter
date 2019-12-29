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

	return c.Context.Send(m.ChannelID, fmt.Sprintf(
		"Hello, %s: %d", m.Author.Mention(), c.HelloCalled))
}

// ~echo - admin only
func (c *Commands) AーEcho(m *discordgo.MessageCreate) error {
	return c.Context.Send(m.ChannelID, m.Content)
}

func (c *Commands) AーEditMessage(m *discordgo.MessageUpdate) error {
	for _, user := range m.Mentions {
		if user.ID == c.Context.State.User.ID {
			return c.Context.Reply(m.Message, "you edited.")
		}
	}

	return nil
}

// ~help
func (c *Commands) Help(m *discordgo.MessageCreate) error {
	return c.Context.Send(m.ChannelID, c.Context.Help())
}
