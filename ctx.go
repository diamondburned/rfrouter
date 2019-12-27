package rfrouter

import (
	"reflect"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type Context struct {
	*discordgo.Session

	Commands interface{}

	// The prefix for commands
	Prefix string

	// Mapping first command name to function names
	MapName func(string) string

	// FormatError formats any errors returned by anything, including the method
	// commands or the reflect functions. This also includes invalid usage
	// errors or unknown command errors.
	FormatError func(error) string

	// Directly to struct
	cmdValue *reflect.Value
	cmdType  *reflect.Type

	// Pointer value
	ptrValue *reflect.Value
	ptrType  *reflect.Type

	commands []commandContext
}

// New makes a new context with a "~" as the prefix. cmds must be a pointer to a
// struct with a *Context field. Example:
//
//    type Commands struct {
//        Ctx *Context
//    }
//
//    cmds := &Commands{}
//    c, err := rfrouter.New(session, cmds)
//
// Commands' exported methods will all be used as commands. Messages are parsed
// with its first argument (the command) mapped accordingly to c.MapName, which
// capitalizes the first letter automatically to reflect the exported method
// name.
//
// The default prefix is "~", which means commands must start with "~" followed
// by the command name in the first argument, else it will be ignored.
func New(s *discordgo.Session, cmds interface{}) (*Context, error) {
	ctx := &Context{
		Session:  s,
		Commands: cmds,
		Prefix:   "~",
		MapName: func(s string) string {
			first := unicode.ToUpper(rune(s[0]))
			return string(first) + s[1:]
		},
		FormatError: func(err error) string {
			return err.Error()
		},
	}

	if err := ctx.reflectCommands(); err != nil {
		return nil, err
	}

	if err := ctx.initCommands(); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize with given cmds")
	}

	if err := ctx.parseCommands(); err != nil {
		return nil, errors.Wrap(err, "Failed to parse commands")
	}

	return ctx, nil
}

func (ctx *Context) Start() func() {
	return ctx.Session.AddHandler(func(_ *discordgo.Session, v interface{}) {
		if err := ctx.callCmd(v); err != nil {
			if str := ctx.FormatError(err); str != "" {
				mc, ok := v.(*discordgo.MessageCreate)
				if !ok {
					return
				}

				// TODO: hard-coded? idk
				ctx.Session.ChannelMessageSend(mc.ChannelID, str)
			}
		}
	})
}

func (ctx *Context) Call(event interface{}) error {
	return ctx.callCmd(event)
}

func (ctx *Context) Send(channelID string, content interface{}) (*discordgo.Message, error) {
	switch content := content.(type) {
	case string:
		return ctx.Session.ChannelMessageSend(channelID, content)
	case *discordgo.MessageEmbed:
		return ctx.Session.ChannelMessageSendEmbed(channelID, content)
	case *discordgo.MessageSend:
		return ctx.Session.ChannelMessageSendComplex(channelID, content)
	default:
		// BUG
		panic("Send received an unknown content type")
	}
}

func (ctx *Context) Member(guildID, memberID string) (*discordgo.Member, error) {
	m, err := ctx.Session.State.Member(guildID, memberID)
	if err != nil {
		m, err = ctx.Session.GuildMember(guildID, memberID)
		if err != nil {
			return nil, err
		}

		ctx.Session.State.MemberAdd(m)
	}

	return m, nil
}

func (ctx *Context) Role(guildID, roleID string) (*discordgo.Role, error) {
	r, err := ctx.Session.State.Role(guildID, roleID)
	if err != nil {
		roles, err := ctx.Session.GuildRoles(guildID)
		if err != nil {
			return nil, err
		}

		for _, role := range roles {
			ctx.Session.State.RoleAdd(guildID, role)

			if role.ID == roleID {
				r = role
			}
		}
	}

	if r == nil {
		return nil, errors.New("role not found")
	}

	return r, nil
}

func (ctx *Context) Channel(channelID string) (*discordgo.Channel, error) {
	c, err := ctx.Session.State.Channel(channelID)
	if err != nil {
		c, err = ctx.Session.Channel(channelID)
		if err != nil {
			return nil, err
		}

		ctx.Session.State.ChannelAdd(c)
	}

	return c, nil
}
