package cli

import "slices"

type GlobalFlags struct {
	ConfigPath string
	JSON       bool
	Help       bool
	Version    bool
}

func parseGlobalFlags(args []string) (GlobalFlags, []string) {
	var flags GlobalFlags
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			flags.JSON = true
		case "--help", "-h":
			flags.Help = true
		case "--version", "-v":
			flags.Version = true
		case "--config":
			if i+1 < len(args) {
				flags.ConfigPath = args[i+1]
				i++
			}
		default:
			rest = append(rest, args[i:]...)
			if acceptsTrailingJSON(rest) {
				flags.JSON = true
				rest = rest[:len(rest)-1]
			}
			return flags, rest
		}
	}
	return flags, rest
}

func acceptsTrailingJSON(args []string) bool {
	if len(args) < 2 || args[len(args)-1] != "--json" {
		return false
	}
	switch args[0] {
	case "search":
		return len(args) > 2
	case "sql":
		return false
	default:
		return true
	}
}

func hasFlag(args []string, name string) bool {
	return slices.Contains(args, name)
}

func flagValue(args []string, name string) (string, bool) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			return args[i+1], true
		}
	}
	return "", false
}
