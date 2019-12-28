package rfrouter

import (
	"reflect"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type Subcommand struct {
	// struct name
	name string

	// Directly to struct
	cmdValue reflect.Value
	cmdType  reflect.Type

	// Pointer value
	ptrValue reflect.Value
	ptrType  reflect.Type

	command  interface{}
	commands []commandContext
}

func NewSubcommand(cmd interface{}) (*Subcommand, error) {
	var sub = Subcommand{
		command: cmd,
	}

	if err := sub.reflectCommands(); err != nil {
		return nil, errors.Wrap(err, "Failed to reflect commands")
	}

	if err := sub.parseCommands(); err != nil {
		return nil, errors.Wrap(err, "Failed to parse commands")
	}

	return &sub, nil
}

// Name returns the command name in lower case. This only returns non-zero for
// subcommands.
func (sub *Subcommand) Name() string {
	return sub.name
}

func (sub *Subcommand) needsName() {
	sub.name = strings.ToLower(sub.cmdType.Name())
}

func (sub *Subcommand) reflectCommands() error {
	t := reflect.TypeOf(sub.command)
	v := reflect.ValueOf(sub.command)

	if t.Kind() != reflect.Ptr {
		return errors.New("sub is not a pointer")
	}

	// Set the pointer fields
	sub.ptrValue = v
	sub.ptrType = t

	ts := t.Elem()
	vs := v.Elem()

	if ts.Kind() != reflect.Struct {
		return errors.New("sub is not pointer to struct")
	}

	// Set the struct fields
	sub.cmdValue = vs
	sub.cmdType = ts

	return nil
}

// called later
func (sub *Subcommand) initCommands(ctx *Context) error {
	// Start filling up a *Context field
	for i := 0; i < sub.cmdValue.NumField(); i++ {
		field := sub.cmdValue.Field(i)

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
	name      string        // all lower-case
	value     reflect.Value // Func
	event     reflect.Type  // discordgo.*
	method    reflect.Method
	arguments []argumentValueFn

	parseMethod reflect.Method
	parseType   reflect.Type
}

var (
	typeMessageCreate = reflect.TypeOf((*discordgo.MessageCreate)(nil))
	// typeof.Implements(typeI*)
	typeIError = reflect.TypeOf((*error)(nil)).Elem()
	typeIManP  = reflect.TypeOf((*ManualParseable)(nil)).Elem()
)

func (sub *Subcommand) parseCommands() error {
	var numMethods = sub.ptrValue.NumMethod()
	var commands = make([]commandContext, 0, numMethods)

	for i := 0; i < numMethods; i++ {
		method := sub.ptrValue.Method(i)

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
		if err := methodT.Out(0); err == nil || !err.Implements(typeIError) {
			continue
		}

		var command = commandContext{
			method: sub.ptrType.Method(i),
			value:  method,
			event:  methodT.In(0), // parse event
		}

		// Grab the method name
		command.name = strings.ToLower(command.method.Name)

		// TODO: allow more flexibility
		if command.event != typeMessageCreate {
			continue
		}

		if numArgs == 1 {
			// done
			continue
		}

		// TODO: manual parser
		if t := methodT.In(1); t.Implements(typeIManP) {
			if t.Kind() != reflect.Ptr {
				return errors.New("ManualParser is not pointer " + t.String())
			}

			mt, ok := t.MethodByName("ParseContent")
			if !ok {
				panic("BUG: type IManP does not implement ParseContent")
			}

			command.parseMethod = mt
			command.parseType = t.Elem()
			goto Continue
		}

		command.arguments = make([]argumentValueFn, 0, numArgs)

		// Fill up arguments
		for i := 1; i < numArgs; i++ {
			t := methodT.In(i)

			avfs, err := getArgumentValueFn(t)
			if err != nil {
				return errors.Wrap(err, "Error parsing argument "+t.String())
			}

			command.arguments = append(command.arguments, avfs)
		}

	Continue:
		// Append
		commands = append(commands, command)
	}

	sub.commands = commands
	return nil
}
