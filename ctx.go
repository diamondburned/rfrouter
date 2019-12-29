package rfrouter

import (
	"encoding/csv"
	"log"
	"reflect"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type Context struct {
	*Subcommand
	*discordgo.Session

	// Descriptive (but optional) bot name
	Name string

	// Descriptive help body
	Description string

	// The prefix for commands
	Prefix string

	// TODO: add a complex string mapper API between commands and methods

	// FormatError formats any errors returned by anything, including the method
	// commands or the reflect functions. This also includes invalid usage
	// errors or unknown command errors.
	FormatError func(error) string

	// ErrorLogger logs any error that anything makes and the library can't
	// reply to the client. This includes any event callback errors that aren't
	// Message Create.
	ErrorLogger func(error)

	// Subcommands contains all the registered subcommands.
	Subcommands []*Subcommand
}

// StartBot quickly starts a bot with the given command. It will prepend "Bot"
// into the token automatically. Refer to example/ for usage.
func StartBot(token string, cmd interface{},
	opts func(*Context) error) (stop func() error, err error) {

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a dgo session")
	}

	c, err := New(s, cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create rfrouter")
	}

	if opts != nil {
		if err := opts(c); err != nil {
			return nil, err
		}
	}

	cancel := c.Start()

	if err := s.Open(); err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Discord")
	}

	return func() error {
		cancel()
		return s.Close()
	}, nil
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

	if err := ctx.InitCommands(ctx); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize with given cmds")
	}

	return ctx, nil
}

func (ctx *Context) RegisterSubcommand(cmd interface{}) (*Subcommand, error) {
	s, err := NewSubcommand(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to add subcommand")
	}

	// Register the subcommand's name.
	s.NeedsName()

	if err := s.InitCommands(ctx); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize subcommand")
	}

	// Do a collision check
	for _, sub := range ctx.Subcommands {
		if sub.name == s.name {
			return nil, errors.New(
				"New subcommand has duplicate name: " + s.name)
		}
	}

	ctx.Subcommands = append(ctx.Subcommands, s)
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
				_, Merr := ctx.Session.ChannelMessageSend(mc.ChannelID, str)
				if Merr != nil {
					// Log the main error first
					ctx.ErrorLogger(errors.Wrap(err, str))
					// Then the message error
					ctx.ErrorLogger(Merr)
					// TODO: there ought to be a better way lol
				}
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
func (ctx *Context) Send(channelID string, content interface{}) (err error) {
	switch content := content.(type) {
	case string:
		_, err = ctx.Session.ChannelMessageSend(channelID, content)
	case *discordgo.MessageEmbed:
		_, err = ctx.Session.ChannelMessageSendEmbed(channelID, content)
	case *discordgo.MessageSend:
		_, err = ctx.Session.ChannelMessageSendComplex(channelID, content)
	default:
		// BUG
		panic("Send received an unknown content type")
	}

	return
}

// Reply mentions the user when sending the message.
func (ctx *Context) Reply(m *discordgo.Message, reply string) error {
	return ctx.Send(m.ChannelID, m.Author.Mention()+", "+reply)
}

// Help generates one. This function is used more for reference than an actual
// help message. As such, it only uses exported fields or methods.
func (ctx *Context) Help() string {
	var help strings.Builder

	// Generate the headers and descriptions
	help.WriteString("__Help__")

	if ctx.Name != "" {
		help.WriteString(": " + ctx.Name)
	}

	if ctx.Description != "" {
		help.WriteString("\n      " + ctx.Description)
	}

	if ctx.Flag.Is(AdminOnly) {
		// That's it.
		return help.String()
	}

	// Separators
	help.WriteString("\n---\n")

	// Generate all commands
	help.WriteString("__Commands__\n")

	for _, cmd := range ctx.Commands {
		if cmd.Flag.Is(AdminOnly) {
			// Hidden
			continue
		}

		help.WriteString("      " + ctx.Prefix + cmd.Name())

		if cmd.Description != "" {
			help.WriteString(": " + cmd.Description)
		}

		help.WriteByte('\n')
	}

	help.WriteString("---\n")

	// Generate all subcommands
	help.WriteString("__Subcommands__\n")

	for _, sub := range ctx.Subcommands {
		if sub.Flag.Is(AdminOnly) {
			// Hidden
			continue
		}

		help.WriteString("      " + sub.Name())

		if sub.Description != "" {
			help.WriteString(": " + sub.Description)
		}

		help.WriteByte('\n')

		for _, cmd := range sub.Commands {
			if cmd.Flag.Is(AdminOnly) {
				continue
			}

			help.WriteString("            " +
				ctx.Prefix + sub.Name() + " " + cmd.Name())

			if cmd.Description != "" {
				help.WriteString(": " + cmd.Description)
			}

			help.WriteByte('\n')
		}
	}

	return help.String()
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

func (ctx *Context) callCmd(ev interface{}) error {
	evT := reflect.TypeOf(ev)

	if evT != typeMessageCreate {
		var callers []reflect.Value
		var isAdmin *bool // i want to die

		for _, cmd := range ctx.Commands {
			if cmd.event == evT {
				if cmd.Flag.Is(AdminOnly) &&
					!ctx.eventIsAdmin(ev, &isAdmin) {

					continue
				}

				callers = append(callers, cmd.value)
			}
		}

		for _, sub := range ctx.Subcommands {
			if sub.Flag.Is(AdminOnly) &&
				!ctx.eventIsAdmin(ev, &isAdmin) {

				continue
			}

			for _, cmd := range sub.Commands {
				if cmd.event == evT {
					if cmd.Flag.Is(AdminOnly) &&
						!ctx.eventIsAdmin(ev, &isAdmin) {

						continue
					}

					callers = append(callers, cmd.value)
				}
			}
		}

		for _, c := range callers {
			if err := callWith(c, ev); err != nil {
				ctx.ErrorLogger(err)
			}
		}

		return nil
	}

	// safe assertion always
	mc := ev.(*discordgo.MessageCreate)

	// check if prefix
	if !strings.HasPrefix(mc.Content, ctx.Prefix) {
		// not a command, ignore
		return nil
	}

	// trim the prefix before splitting, this way multi-words prefices work
	content := mc.Content[len(ctx.Prefix):]

	// parse arguments
	args, err := ParseArgs(content)
	if err != nil {
		return err
	}

	if len(args) < 1 {
		return nil // ???
	}

	var cmd *CommandContext
	var start int // arg starts from $start

	// Search for the command
	for _, c := range ctx.Commands {
		if c.name == args[0] {
			cmd = c
			start = 1
			break
		}
	}

	// Can't find command, look for subcommands of len(args) has a 2nd
	// entry.
	if cmd == nil && len(args) > 1 {
		for _, s := range ctx.Subcommands {
			if s.name != args[0] {
				continue
			}

			for _, c := range s.Commands {
				if c.name == args[1] {
					cmd = c
					start = 2
					break
				}
			}

			if cmd == nil {
				return &ErrUnknownCommand{
					Command: args[1],
					Parent:  args[0],
					Prefix:  ctx.Prefix,
					ctx:     s.Commands,
				}
			}
		}
	}

	if cmd == nil || start == 0 {
		return &ErrUnknownCommand{
			Command: args[0],
			Prefix:  ctx.Prefix,
			ctx:     ctx.Commands,
		}
	}

	// Start converting
	var argv []reflect.Value

	// Check manual parser
	if cmd.parseType != nil {
		// Create a zero value instance of this
		v := reflect.New(cmd.parseType)

		// Call the manual parse method
		ret := cmd.parseMethod.Func.Call([]reflect.Value{
			v, reflect.ValueOf(args),
		})

		// Check the method returns for error
		if err := errorReturns(ret); err != nil {
			// TODO: maybe wrap this?
			return err
		}

		// Add the pointer to the argument into argv
		argv = append(argv, v)
		goto Call
	}

	// Here's an edge case: when the handler takes no arguments, we allow that
	// anyway, as they might've used the raw content.
	if len(cmd.arguments) == 0 {
		goto Call
	}

	// Not enough arguments given
	if len(args[start:]) != len(cmd.arguments) {
		return &ErrInvalidUsage{
			Args:   args,
			Prefix: ctx.Prefix,
			Index:  len(cmd.arguments) - start,
			Err:    "Not enough arguments given",
			ctx:    cmd,
		}
	}

	argv = make([]reflect.Value, len(cmd.arguments))

	for i := start; i < len(args); i++ {
		v, err := cmd.arguments[i-start](args[i])
		if err != nil {
			return &ErrInvalidUsage{
				Args:   args,
				Prefix: ctx.Prefix,
				Index:  i,
				Err:    err.Error(),
				ctx:    cmd,
			}
		}

		argv[i-start] = v
	}

Call:
	// call the function and parse the error return value
	return callWith(cmd.value, ev, argv...)
}

func (ctx *Context) eventIsAdmin(ev interface{}, is **bool) bool {
	if *is != nil {
		return **is
	}

	var channelID = reflectChannelID(ev)
	if channelID == "" {
		return false
	}

	var userID = reflectUserID(ev)
	if userID == "" {
		return false
	}

	var res bool

	p, err := ctx.UserPermissions(channelID, userID)
	if err == nil && p&discordgo.PermissionAdministrator != 0 {
		res = true
	}

	*is = &res
	return res
}

func callWith(caller reflect.Value, ev interface{}, values ...reflect.Value) error {
	return errorReturns(caller.Call(append(
		[]reflect.Value{reflect.ValueOf(ev)},
		values...,
	)))
}

var ParseArgs = func(args string) ([]string, error) {
	// fuck me
	// TODO: make modular
	// TODO: actual tokenizer+parser
	r := csv.NewReader(strings.NewReader(args))
	r.Comma = ' '

	return r.Read()
}

func errorReturns(returns []reflect.Value) error {
	// assume first is always error, since we checked for this in parseCommands
	v := returns[0].Interface()

	if v == nil {
		return nil
	}

	return v.(error)
}

func reflectChannelID(_struct interface{}) string {
	return _reflectID(reflect.ValueOf(_struct), "Channel")
}

func reflectGuildID(_struct interface{}) string {
	return _reflectID(reflect.ValueOf(_struct), "Guild")
}

func reflectUserID(_struct interface{}) string {
	return _reflectID(reflect.ValueOf(_struct), "User")
}

func _reflectID(v reflect.Value, thing string) string {
	if !v.IsValid() {
		return ""
	}

	t := v.Type()

	if t.Kind() == reflect.Ptr {
		v = v.Elem()

		// Recheck after dereferring
		if !v.IsValid() {
			return ""
		}

		t = v.Type()
	}

	if t.Kind() != reflect.Struct {
		return ""
	}

	numFields := t.NumField()

	for i := 0; i < numFields; i++ {
		field := t.Field(i)
		fType := field.Type

		if fType.Kind() == reflect.Ptr {
			fType = fType.Elem()
		}

		switch fType.Kind() {
		case reflect.Struct:
			if chID := _reflectID(v.Field(i), thing); chID != "" {
				return chID
			}
		case reflect.String:
			if field.Name == thing+"ID" {
				// grab value real quick
				return v.Field(i).String()
			}

			// Special case where the struct name has Channel in it
			if field.Name == "ID" && strings.Contains(t.Name(), thing) {
				return v.Field(i).String()
			}
		}
	}

	return ""
}
