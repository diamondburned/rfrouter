package rfrouter

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type Context struct {
	*Subcommand
	*discordgo.Session

	// The prefix for commands
	Prefix string

	// FormatError formats any errors returned by anything, including the method
	// commands or the reflect functions. This also includes invalid usage
	// errors or unknown command errors.
	FormatError func(error) string

	// ErrorLogger logs any error that anything makes and the library can't
	// reply to the client. This includes any event callback errors that aren't
	// Message Create.
	ErrorLogger func(error)

	subcommands []*Subcommand
	allCommands []commandContext
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
//
// c.Start() should be called afterwards to actually handle incoming events.
func New(s *discordgo.Session, cmd interface{}) (*Context, error) {
	c, err := NewSubcommand(cmd)
	if err != nil {
		return nil, err
	}

	ctx := &Context{
		Subcommand: c,
		Session:    s,
		Prefix:     "~",
		FormatError: func(err error) string {
			return err.Error()
		},
		ErrorLogger: func(err error) {
			log.Println("ERR:", err)
		},
	}

	if err := ctx.initCommands(ctx); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize with given cmds")
	}

	return ctx, nil
}

func (ctx *Context) RegisterSubcommand(cmd interface{}) (*Subcommand, error) {
	s, err := NewSubcommand(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to add subcommand")
	}

	s.needsName()

	if err := s.initCommands(ctx); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize subcommand")
	}

	ctx.subcommands = append(ctx.subcommands, s)
	return s, nil
}

// Start adds itself into the discordgo Session handlers. This needs to be run.
// The returned function is a delete function, which removes itself from the
// Session handlers.
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

// Call should only be used if you know what you're doing.
func (ctx *Context) Call(event interface{}) error {
	return ctx.callCmd(event)
}

// Send sends a string, an embed pointer or a MessageSend pointer. Any other
// type given will panic.
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

// Member returns the member, adding it to the State.
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

// Role returns the role, adding it to the State.
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

// Channel returns the channel, adding it to the State.
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
