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
			rest = append(rest, arg)
		}
	}
	return flags, rest
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
