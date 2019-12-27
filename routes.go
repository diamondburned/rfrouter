package rfrouter

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Commands struct {
	Context *Context
}

// A command must have its first argument the event.
// Arguments follow it afterwards. Variadic arguments should be supported.
// The return arguments must be (error).
func (c *Commands) Info(m *discordgo.MessageCreate, number int, variadic ...string) error {
	embed := &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{{
			Name:  "Content",
			Value: m.Content,
		}, {
			Name:  "`number`",
			Value: strconv.Itoa(number),
		}, {
			Name:  "variadic",
			Value: strings.Join(variadic, ", "),
		}},
	}

	_, err := c.Context.Send(m.ChannelID, embed)
	return err
}
