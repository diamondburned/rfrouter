# [rfrouter](https://godoc.org/git.sr.ht/~diamondburned/rfrouter)

Proof-of-concept

## Usage

```go
var token = "Bot 123456abcxyz"
var cmds  = Commands{}

c, err := rfrouter.StartBot(token, &cmds,
	func(ctx *rfrouter.Context) error {
		// Set the prefix
		ctx.Prefix = "~"

		// Set the descriptions
		ctx.Name = "bot example"
		ctx.Description = "https://git.sr.ht/~diamondburned/rfrouter"

		return err
	},
)

if err != nil {
	log.Fatalln("Failed to start the bot:", err)
}

defer c()
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

## Features

- Automatic command routing from Go methods
- Implicit name conversion between strings and method names
- Mapping Go arguments from strings
- Pluggable parsers and arguments
- Subcommands allow for plug-ins
- Help page generation

## Non-features

- Descriptions for commands: impossible (or otherwise impractical) without any
	form of code parsing.

## Some extra features nobody cares about

### Interfaces

#### Parseable

```go
// Parseable implements a Parse(string) method for data structures that can be
// used as arguments.
type Parseable interface {
	Parse(string) error
}
```

###### Example (refer to `extras/arguments/emoji.go`)

#### ManualParseable

```go
// ManualParseable implements a ParseContent(string) method. If the library sees
// this for an argument, it will send all of the arguments (including the
// command) into the method. If used, this should be the only argument followed
// after the Message Create event. Any more and the router will ignore.
type ManualParseable interface {
	// $0 will have its prefix trimmed.
	ParseContent([]string) error
}
```

###### Example (refer to `extras/arguments/flag.go`)

#### Usager

```go
// Usager is optionally used to override the generated usage for either an
// argument, or multiple (using ManualParseable).
type Usager interface {
	Usage() string
}
```

#### Example (refer to `extras/arguments/mention.go`)
