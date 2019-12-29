package rfrouter

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type testCommands struct {
	Ctx    *Context
	Return chan interface{}
}

func (t *testCommands) Send(_ *discordgo.MessageCreate, arg string) error {
	t.Return <- arg
	return errors.New("oh no")
}

func (t *testCommands) Custom(_ *discordgo.MessageCreate, c *CustomParseable) error {
	t.Return <- c.args
	return nil
}

func (t *testCommands) NoArgs(_ *discordgo.MessageCreate) error {
	return errors.New("passed")
}

func (t *testCommands) Noop(_ *discordgo.MessageCreate) error {
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
		if err := ctx.Subcommand.InitCommands(ctx); err != nil {
			t.Fatal("Failed to init commands:", err)
		}

		if given.Ctx == nil {
			t.Fatal("given's Context field is nil")
		}

		if given.Ctx.Session.Token != "dumb token" {
			t.Fatal("given's Session token is wrong")
		}
	})

	testReturn := func(expects interface{}, content string) (call error) {
		// Return channel for testing
		ret := make(chan interface{})
		given.Return = ret

		// Mock a messageCreate event
		m := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				Content: content,
			},
		}

		var (
			callCh = make(chan error)
		)

		go func() {
			callCh <- ctx.callCmd(m)
		}()

		select {
		case arg := <-ret:
			if !reflect.DeepEqual(arg, expects) {
				t.Fatal("returned argument is invalid:", arg)
			}
			call = <-callCh

		case call = <-callCh:
			t.Fatal("expected return before error:", call)
		}

		return
	}

	t.Run("call command", func(t *testing.T) {
		// Set a custom prefix
		ctx.Prefix = "~"

		if err := testReturn("test", "~send test"); err.Error() != "oh no" {
			t.Fatal("unexpected error:", err)
		}
	})

	t.Run("call command custom parser", func(t *testing.T) {
		ctx.Prefix = "!"
		expects := []string{"custom", "arg1", ":)"}

		if err := testReturn(expects, "!custom arg1 :)"); err != nil {
			t.Fatal("Unexpected call error:", err)
		}
	})

	testMessage := func(content string) error {
		// Mock a messageCreate event
		m := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				Content: content,
			},
		}

		return ctx.callCmd(m)
	}

	t.Run("call command without args", func(t *testing.T) {
		ctx.Prefix = ""

		if err := testMessage("noargs"); err.Error() != "passed" {
			t.Fatal("unexpected error:", err)
		}
	})

	// Test error cases

	t.Run("call unknown command", func(t *testing.T) {
		ctx.Prefix = "joe pls "

		err := testMessage("joe pls no")

		if err == nil || !strings.HasPrefix(err.Error(), "Unknown command:") {
			t.Fatal("unexpected error:", err)
		}
	})

	// Test subcommands

	t.Run("register subcommand", func(t *testing.T) {
		ctx.Prefix = "run "

		_, err := ctx.RegisterSubcommand(&testCommands{})
		if err != nil {
			t.Fatal("Failed to register subcommand:", err)
		}

		if err := testMessage("run testcommands noop"); err != nil {
			t.Fatal("unexpected error:", err)
		}
	})
}

func BenchmarkConstructor(b *testing.B) {
	var session = &discordgo.Session{
		Token: "dumb token",
	}

	for i := 0; i < b.N; i++ {
		_, _ = New(session, &testCommands{})
	}
}

func BenchmarkCall(b *testing.B) {
	var given = &testCommands{}
	var session = &discordgo.Session{
		Token: "dumb token",
	}

	s, _ := NewSubcommand(given)

	var ctx = &Context{
		Subcommand: s,
		Session:    session,
		Prefix:     "~",
	}

	m := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Content: "~noop",
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.callCmd(m)
	}
}

type hasID struct {
	ChannelID string
}

type embedsID struct {
	*hasID
	*embedsID
}

type hasChannelInName struct {
	ID string
}

func TestReflectChannelID(t *testing.T) {
	var s = &hasID{
		ChannelID: "channelID",
	}

	t.Run("hasID", func(t *testing.T) {
		if id := reflectChannelID(s); id != "channelID" {
			t.Fatal("unexpected channelID:", id)
		}
	})

	t.Run("embedsID", func(t *testing.T) {
		var e = &embedsID{
			hasID: s,
		}

		if id := reflectChannelID(e); id != "channelID" {
			t.Fatal("unexpected channelID:", id)
		}
	})

	t.Run("hasChannelInName", func(t *testing.T) {
		var s = &hasChannelInName{
			ID: "channelID",
		}

		if id := reflectChannelID(s); id != "channelID" {
			t.Fatal("unexpected channelID:", id)
		}
	})
}

func BenchmarkReflectChannelID_1Level(b *testing.B) {
	var s = &hasID{
		ChannelID: "channelID",
	}

	for i := 0; i < b.N; i++ {
		_ = reflectChannelID(s)
	}
}

func BenchmarkReflectChannelID_5Level(b *testing.B) {
	var s = &embedsID{nil, &embedsID{nil, &embedsID{nil, &embedsID{
		hasID: &hasID{
			ChannelID: "channelID",
		},
	}}}}

	for i := 0; i < b.N; i++ {
		_ = reflectChannelID(s)
	}
}
