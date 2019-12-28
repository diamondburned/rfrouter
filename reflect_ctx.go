package rfrouter

import (
	"encoding/csv"
	"reflect"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (ctx *Context) callCmd(ev interface{}) error {
	evT := reflect.TypeOf(ev)

	if evT != typeMessageCreate {
		var callers []reflect.Value

		for _, cmd := range ctx.commands {
			if cmd.event == evT {
				callers = append(callers, cmd.value)
			}
		}

		for _, sub := range ctx.subcommands {
			for _, cmd := range sub.commands {
				if cmd.event == evT {
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

	// parse arguments
	args, err := parseArgs(mc.Content)
	if err != nil {
		return err
	}

	if len(args) < 1 {
		return nil // ???
	}

	// check if prefix
	if !strings.HasPrefix(args[0], ctx.Prefix) {
		// not a command, ignore
		return nil
	}

	// map the first arg
	args[0] = strings.TrimPrefix(args[0], ctx.Prefix)

	var cmd *commandContext
	var start int // arg starts from $start

	// Search for the command
	for _, c := range ctx.commands {
		if c.name == args[0] {
			cmd = &c
			start = 1
			break
		}
	}

	// Can't find command, look for subcommands of len(args) has a 2nd
	// entry.
	if cmd == nil && len(args) > 1 {
		for _, s := range ctx.subcommands {
			if s.name != args[0] {
				continue
			}

			for _, c := range s.commands {
				if c.name == args[1] {
					cmd = &c
					start = 2
					break
				}
			}
		}
	}

	if cmd == nil || start == 0 {
		return ErrUnknownCommand{
			Command: args[0],
			ctx:     ctx.commands,
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

	// Not enough arguments given
	if len(args[start:]) != len(cmd.arguments) {
		return ErrInvalidUsage{
			Command: args[0],
			ctx:     cmd,
		}
	}

	argv = make([]reflect.Value, len(cmd.arguments))

	for i := start; i < len(args); i++ {
		v, err := cmd.arguments[i-start](args[i])
		if err != nil {
			return ErrInvalidUsage{
				Command: args[0],
				Index:   i,
			}
		}

		argv[i-start] = v
	}

Call:
	// call the function and parse the error return value
	return callWith(cmd.value, ev, argv...)
}

func callWith(caller reflect.Value, ev interface{}, values ...reflect.Value) error {
	return errorReturns(caller.Call(append(
		[]reflect.Value{reflect.ValueOf(ev)},
		values...,
	)))
}

func parseArgs(args string) ([]string, error) {
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
