package debug

import (
	"fmt"
	"runtime"

	"git.sr.ht/~diamondburned/rfrouter"
	"github.com/bwmarrin/discordgo"
)

// Admin only
type AーDebug struct {
	Context *rfrouter.Context
}

func (d *AーDebug) Name() string {
	return "d"
}

func (d *AーDebug) Description() string {
	return "debugging commands"
}

// ~debug goroutines
func (d *AーDebug) Goroutines(m *discordgo.MessageCreate) error {
	return d.Context.Send(m.ChannelID, fmt.Sprintf("goroutines: %d",
		runtime.NumGoroutine()))
}

// ~debug GOOS
func (d *AーDebug) RーGOOS(m *discordgo.MessageCreate) error {
	return d.Context.Send(m.ChannelID, runtime.GOOS)
}

// ~debug GC
func (d *AーDebug) RーGC(m *discordgo.MessageCreate) error {
	runtime.GC()
	return nil
}

// ~debug die
func (d *AーDebug) AーDie(m *discordgo.MessageCreate) error {
	panic("Death requested from " + m.Author.Username)
}
