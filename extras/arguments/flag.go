package arguments

import "flag"

var FlagName = "command"

func NewFlagSet() *flag.FlagSet {
	return flag.NewFlagSet(FlagName, flag.ContinueOnError)
}

type Flag struct {
	arguments []string
}

func (f *Flag) ParseContent(arguments []string) error {
	// trim the command out
	f.arguments = arguments[1:]
	return nil
}

func (f *Flag) Usage() string {
	return "flags..."
}

func (f *Flag) With(fs *flag.FlagSet) error {
	return fs.Parse(f.arguments)
}
