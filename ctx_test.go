package rfrouter

import (
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type testCommands struct {
	Ctx       *Context
	retSend   chan string
	retCustom chan []string
}

func (t *testCommands) Send(_ *discordgo.MessageCreate, arg string) error {
	t.retSend <- arg
	return errors.New("oh no")
}

func (t *testCommands) Custom(_ *discordgo.MessageCreate, c *CustomParseable) error {
	t.retCustom <- c.args
	return nil
}

type CustomParseable struct {
	args []string
}

func (c *CustomParseable) ParseContent(args []string) error {
	c.args = args
	return nil
}

func TestNewContext(t *testing.T) {
	var session = &discordgo.Session{
		Token: "dumb token",
	}

	_, err := New(session, &testCommands{})
	if err != nil {
		t.Fatal("Failed to create new context:", err)
	}
}

func TestContext(t *testing.T) {
	var given = &testCommands{}
	var session = &discordgo.Session{
		Token: "dumb token",
	}

	s, err := NewSubcommand(given)
	if err != nil {
		t.Fatal("Failed to create subcommand:", err)
	}

	var ctx = &Context{
		Subcommand: s,
		Session:    session,
	}

	t.Run("init commands", func(t *testing.T) {
		if err := ctx.Subcommand.initCommands(ctx); err != nil {
			t.Fatal("Failed to init commands:", err)
		}

		if given.Ctx == nil {
			t.Fatal("given's Context field is nil")
		}

		if given.Ctx.Session.Token != "dumb token" {
			t.Fatal("given's Session token is wrong")
		}
	})

	t.Run("call command", func(t *testing.T) {
		// Set a custom prefix
		ctx.Prefix = "~"

		// Return channel for testing
		ret := make(chan string)
		given.retSend = ret

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

	t.Run("call command custom parser", func(t *testing.T) {
		ctx.Prefix = "!"

		ret := make(chan []string)
		given.retCustom = ret

		m := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				Content: "!custom arg1 :)",
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
		case args := <-ret:
			if !reflect.DeepEqual(args, []string{"custom", "arg1", ":)"}) {
				t.Fatal("returned argument is invalid:", args)
			}
			callErr = <-callCh

		case callErr = <-callCh:
			t.Fatal("expected return before error:", callErr)
		}

		if callErr != nil {
			t.Fatal("Unexpected call error:", callErr)
		}
	})
}
