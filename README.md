# [rfrouter](https://godoc.org/git.sr.ht/~diamondburned/rfrouter)

Proof-of-concept

## Usage

```go
var s *discordgo.Session // initialize this
var c Commands

ctx, err := rfrouter.New(s, &c)
if err != nil {
	// crash and burn
}

ctx.Prefix = "!" // !command
ctx.MapName = func(s string) string {
	// first letter capitalized
	first := unicode.ToUpper(rune(s[0]))

	// "send" -> "Send" (exported method name)
	return string(first) + s[1:]
}

ctx.Handle()

s.Open()
defer s.Close()
```

```go
type Commands struct {
	// Field name can be anything, but this field must be provided
	Ctx *Context
}

func (c *Commands) Send(msg *discordgo.MessageCreate, arg string) error {
	_, err := c.Ctx.Send(msg.ChannelID, "You sent: " + arg)
	return err
}
```

## TODO (or to-be-implemented)

- [ ] Help page generator (modular as well)
- [ ] Usage guide generator (^)
