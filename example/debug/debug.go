package debug

import (
	"fmt"
	"runtime"

	"git.sr.ht/~diamondburned/rfrouter"
	"github.com/bwmarrin/discordgo"
)

type Debug struct {
	Context *rfrouter.Context
}

// ~debug goroutines
func (d *Debug) Goroutines(m *discordgo.MessageCreate) error {
	_, err := d.Context.Send(m.ChannelID, fmt.Sprintf("goroutines: %d",
		runtime.NumGoroutine()))
	return err
}

// ~debug goos
func (d *Debug) GOOS(m *discordgo.MessageCreate) error {
	_, err := d.Context.Send(m.ChannelID, runtime.GOOS)
	return err
}
