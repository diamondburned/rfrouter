package rfrouter

import "strconv"

type ErrUnknownCommand struct {
	Command string
	// TODO: list available commands?
	// Here, as a reminder
	ctx []commandContext
}

func (err ErrUnknownCommand) Error() string {
	return "Unknown command: " + err.Command
}

type ErrInvalidUsage struct {
	Command string

	// TODO: field/arg name?
	Index int

	// TODO: usage generator?
	// Here, as a reminder
	ctx *commandContext
}

func (err ErrInvalidUsage) Error() string {
	if err.Index == 0 {
		return "Invalid usage"
	}

	return "Invalid usage at " + strconv.Itoa(err.Index)
}
