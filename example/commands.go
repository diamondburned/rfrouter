package main

import (
	"fmt"

	"git.sr.ht/~diamondburned/rfrouter"
	"git.sr.ht/~diamondburned/rfrouter/extras/arguments"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
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

// ~flagdemo -opt -str "test string" ayy lmao
func (c *Commands) FlagDemo(m *discordgo.MessageCreate, f *arguments.Flag) error {
	var fs = arguments.NewFlagSet()

	opt := fs.Bool("opt", false, "")
	str := fs.String("str", "", "")

	if err := f.With(fs); err != nil {
		return errors.Wrap(err, "Invalid flags")
	}

	args := fs.Args()

	return c.Context.Send(m.ChannelID, fmt.Sprintf(
		`opt: %v, str: "%s", args: %v`,
		*opt, *str, args),
	)
}

func (c *Commands) Channel(m *discordgo.MessageCreate, ch *arguments.ChannelMention) error {
	channel, err := c.Context.Channel(string(*ch))
	if err != nil {
		return errors.Wrap(err, "Failed to get channel")
	}

	return c.Context.Send(m.ChannelID, fmt.Sprintf(
		"Channel \"%s\" ID %s NSFW %v Topic \"%s\"",
		channel.Name, channel.ID, channel.NSFW, channel.Topic,
	))
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
