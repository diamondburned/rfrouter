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

func (d *Debug) Name() string {
	return "d"
}

func (d *Debug) Description() string {
	return "debugging commands"
}

// ~debug goroutines
func (d *Debug) Goroutines(m *discordgo.MessageCreate) error {
	return d.Context.Send(m.ChannelID, fmt.Sprintf("goroutines: %d",
		runtime.NumGoroutine()))
}

// ~debug GOOS
func (d *Debug) RーGOOS(m *discordgo.MessageCreate) error {
	return d.Context.Send(m.ChannelID, runtime.GOOS)
}

// ~debug GC
func (d *Debug) RAーGC(m *discordgo.MessageCreate) error {
	runtime.GC()
	return nil
}

// ~debug die
func (d *Debug) AーDie(m *discordgo.MessageCreate) error {
	panic("Death requested from " + m.Author.Username)
}
