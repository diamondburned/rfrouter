package rfrouter

import (
	"encoding/csv"
	"reflect"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

func (ctx *Context) reflectCommands() error {
	t := reflect.TypeOf(ctx.Commands)
	v := reflect.ValueOf(ctx.Commands)

	if t.Kind() != reflect.Ptr {
		return errors.New("cmds is not a pointer")
	}

	// Set the pointer fields
	ctx.ptrValue = &v
	ctx.ptrType = &t

	ts := t.Elem()
	vs := v.Elem()

	if ts.Kind() != reflect.Struct {
		return errors.New("cmds is not pointer to struct")
	}

	// Set the struct fields
	ctx.cmdValue = &vs
	ctx.cmdType = &ts
	return nil
}

func (ctx *Context) initCommands() error {
	// Start filling up a *Context field
	for i := 0; i < ctx.cmdValue.NumField(); i++ {
		field := ctx.cmdValue.Field(i)

		if !field.CanSet() || !field.CanInterface() {
			continue
		}

		if _, ok := field.Interface().(*Context); !ok {
			continue
		}

		field.Set(reflect.ValueOf(ctx))
		return nil
	}

	return errors.New("No fields with *Command found")
}

type commandContext struct {
	name      string
	value     reflect.Value
	event     reflect.Type
	method    reflect.Method
	arguments []argumentValueFn
}

var (
	typeMessageCreate = reflect.TypeOf((*discordgo.MessageCreate)(nil))
	// typeof.Implements(v)
	typeError = reflect.TypeOf((*error)(nil)).Elem()
)

func (ctx *Context) parseCommands() error {
	var numMethods = ctx.ptrValue.NumMethod()
	var commands = make([]commandContext, 0, numMethods)

	for i := 0; i < numMethods; i++ {
		method := ctx.ptrValue.Method(i)

		if !method.CanInterface() {
			continue
		}

		methodT := method.Type()
		numArgs := methodT.NumIn()

		// Doesn't meet requirement for an event
		if numArgs == 0 {
			continue
		}

		// Check return type
		if err := methodT.Out(0); err == nil || !err.Implements(typeError) {
			continue
		}

		var command = commandContext{
			method: (*ctx.ptrType).Method(i),
			value:  method,
			event:  methodT.In(0), // parse event
		}

		// Grab the method name
		command.name = command.method.Name

		// TODO: allow more flexibility
		if command.event != typeMessageCreate {
			continue
		}

		command.arguments = make([]argumentValueFn, 0, numArgs)

		// Fill up arguments
		for i := 1; i < numArgs; i++ {
			t := methodT.In(i)

			avfs, err := getArgumentValueFn(t)
			if err != nil {
				return errors.Wrap(err, "Error parsing argument "+t.Name())
			}

			command.arguments = append(command.arguments, avfs)
		}

		// Append
		commands = append(commands, command)
	}

	ctx.commands = commands
	return nil
}

func (ctx *Context) callCmd(ev interface{}) error {
	evT := reflect.TypeOf(ev)

	if evT != typeMessageCreate {
		for _, cmd := range ctx.commands {
			if cmd.event == evT {
				return errorReturns(cmd.value.Call([]reflect.Value{
					reflect.ValueOf(ev),
				}))
			}
		}

		// There is no command, we're just ignoring this event
		return nil
	}

	// safe assertion always
	mc := ev.(*discordgo.MessageCreate)

	// parse arguments
	args, err := parseArgs(mc.Content)
	if err != nil {
		return err
	}

	// check if prefix
	if !strings.HasPrefix(args[0], ctx.Prefix) {
		// not a command, ignore
		return nil
	}

	// map the first arg
	args[0] = ctx.MapName(strings.TrimPrefix(args[0], ctx.Prefix))

	var cmd *commandContext

	for _, c := range ctx.commands {
		if c.name == args[0] {
			cmd = &c
			break
		}
	}

	if cmd == nil {
		return ErrUnknownCommand{
			Command: args[0],
			ctx:     ctx.commands,
		}
	}

	if len(args) == 1 || len(args[1:]) != len(cmd.arguments) {
		return ErrInvalidUsage{
			Command: args[0],
			ctx:     cmd,
		}
	}

	// Start converting
	argv := make([]reflect.Value, len(cmd.arguments)+1)
	argv[0] = reflect.ValueOf(ev)

	for i := 1; i < len(args); i++ {
		v, err := cmd.arguments[i-1](args[i])
		if err != nil {
			return ErrInvalidUsage{
				Command: args[0],
				Index:   i,
			}
		}

		argv[i] = v
	}

	// call the function and parse the error return value
	return errorReturns(cmd.value.Call(argv))
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
	return returns[0].Interface().(error)
}
