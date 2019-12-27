package rfrouter

import (
	"testing"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type testCommands struct {
	Ctx       *Context
	cmdCalled chan string
}

func (t *testCommands) Send(_ *discordgo.MessageCreate, arg string) error {
	t.cmdCalled <- arg
	return errors.New("oh no")
}

func TestContext(t *testing.T) {
	var given = &testCommands{}
	var session = &discordgo.Session{
		Token: "dumb token",
	}

	var ctx = &Context{
		Session:  session,
		Commands: given,
		MapName: func(s string) string {
			return s
		},
	}

	t.Run("reflect commands", func(t *testing.T) {
		if err := ctx.reflectCommands(); err != nil {
			t.Fatal("Failed to reflect commands:", err)
		}
	})

	t.Run("init commands", func(t *testing.T) {
		if err := ctx.initCommands(); err != nil {
			t.Fatal("Failed to init commands:", err)
		}

		if given.Ctx == nil {
			t.Fatal("given's Context field is nil")
		}

		if given.Ctx.Session.Token != "dumb token" {
			t.Fatal("given's Session token is wrong")
		}
	})

	t.Run("parse commands", func(t *testing.T) {
		if err := ctx.parseCommands(); err != nil {
			t.Fatal("Failed to parse commands:", err)
		}

		if len(ctx.commands) != 1 {
			t.Fatal("invalid ctx.commands len", len(ctx.commands))
		}

		first := ctx.commands[0]

		if first.name != "Send" {
			t.Fatal("invalid command name:", first.name)
		}

		if first.event != typeMessageCreate {
			t.Fatal("invalid event type:", first.event.String())
		}

		if len(first.arguments) != 1 {
			t.Fatal("invalid arguments len", len(first.arguments))
		}
	})

	t.Run("call command", func(t *testing.T) {
		// Set MapName to a custom hard-coded return
		ctx.MapName = func(s string) string {
			// first letter capitalized
			first := unicode.ToUpper(rune(s[0]))
			return string(first) + s[1:]
		}

		// Set a custom prefix
		ctx.Prefix = "~"

		// Return channel for testing
		ret := make(chan string)
		given.cmdCalled = ret

		// Mock a messageCreate event
		m := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				Content: "~send test", // $0 doesn't matter, MapName
			},
		}

		var (
			callCh  = make(chan error)
			callErr error
		)

		go func() {
			callCh <- ctx.callCmd(m)
		}()

		select {
		case arg := <-ret:
			if arg != "test" {
				t.Fatal("returned argument is invalid:", arg)
			}
			callErr = <-callCh

		case callErr = <-callCh:
			t.Fatal("expected return before error:", callErr)
		}

		if callErr == nil {
			t.Fatal("no error returned, error expected")
		}

		if callErr.Error() != "oh no" {
			t.Fatal("unexpected error:", callErr)
		}
	})
}
