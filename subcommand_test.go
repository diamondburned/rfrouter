package rfrouter

import "testing"

func TestNewSubcommand(t *testing.T) {
	_, err := NewSubcommand(&testCommands{})
	if err != nil {
		t.Fatal("Failed to create new subcommand:", err)
	}
}

func TestSubcommand(t *testing.T) {
	var given = &testCommands{}
	var sub = &Subcommand{
		command: given,
	}

	t.Run("reflect commands", func(t *testing.T) {
		if err := sub.reflectCommands(); err != nil {
			t.Fatal("Failed to reflect commands:", err)
		}
	})

	t.Run("parse commands", func(t *testing.T) {
		if err := sub.parseCommands(); err != nil {
			t.Fatal("Failed to parse commands:", err)
		}

		if len(sub.commands) != 2 {
			t.Fatal("invalid ctx.commands len", len(sub.commands))
		}

		var (
			foundSend   bool
			foundCustom bool
		)

		for _, this := range sub.commands {
			switch this.name {
			case "send":
				foundSend = true
				if len(this.arguments) != 1 {
					t.Fatal("invalid arguments len", len(this.arguments))
				}

			case "custom":
				foundCustom = true
				if len(this.arguments) > 0 {
					t.Fatal("arguments should be 0 for custom")
				}
				if this.parseType == nil {
					t.Fatal("custom has nil manualParse")
				}

			default:
				t.Fatal("Unexpected command:", this.name)
			}

			if this.event != typeMessageCreate {
				t.Fatal("invalid event type:", this.event.String())
			}
		}

		if !foundSend {
			t.Fatal("missing send")
		}

		if !foundCustom {
			t.Fatal("missing custom")
		}
	})
}
