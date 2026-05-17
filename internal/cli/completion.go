package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/openclaw/graincrawl/internal/output"
)

var completionCommands = []string{
	"version", "init", "doctor", "metadata", "status", "sync", "refresh",
	"runs", "notes", "search", "sql", "note", "transcripts", "panels", "people", "workspaces",
	"sources", "unlock", "secrets", "export", "snapshot", "import", "tui",
	"completion", "help",
}

func (a App) runCompletion(w io.Writer, flags GlobalFlags, args []string) error {
	if flags.JSON {
		return output.WriteEnvelope(w, map[string]any{
			"shells":   []string{"bash", "zsh"},
			"commands": completionCommands,
		})
	}
	if len(args) != 1 {
		return fmt.Errorf("completion shell required: bash or zsh")
	}
	switch args[0] {
	case "bash":
		_, err := io.WriteString(w, bashCompletion())
		return err
	case "zsh":
		_, err := io.WriteString(w, zshCompletion())
		return err
	default:
		return fmt.Errorf("unsupported completion shell %q: use bash or zsh", args[0])
	}
}

func bashCompletion() string {
	commands := strings.Join(completionCommands, " ")
	return fmt.Sprintf(`# bash completion for graincrawl
_graincrawl_completions()
{
  local cur prev commands global_flags sources
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  commands="%s"
  global_flags="--json --config --help -h"
  sources="private-api desktop-cache public-api companion-cli encrypted-json opfs"

  case "${prev}" in
    --source)
      COMPREPLY=( $(compgen -W "${sources}" -- "${cur}") )
      return 0
      ;;
    completion)
      COMPREPLY=( $(compgen -W "bash zsh" -- "${cur}") )
      return 0
      ;;
  esac

  if [[ ${COMP_CWORD} -le 1 ]]; then
    COMPREPLY=( $(compgen -W "${commands} ${global_flags}" -- "${cur}") )
    return 0
  fi

  COMPREPLY=( $(compgen -W "${global_flags} --source --limit --out --no-transcripts --no-panels" -- "${cur}") )
}
complete -F _graincrawl_completions graincrawl
`, commands)
}

func zshCompletion() string {
	commands := strings.Join(completionCommands, " ")
	return fmt.Sprintf(`#compdef graincrawl
_graincrawl() {
  local -a commands
  commands=(${(z)${:-%q}})
  _arguments \
    '--json[write JSON output]' \
    '--config[config file]:config file:_files' \
    '--help[show help]' \
    '1:command:->command' \
    '*::arg:->args'
  case $state in
    command)
      _describe 'command' commands
      ;;
    args)
      case $words[2] in
        sync|refresh)
          _arguments '--source[source adapter]:(private-api desktop-cache public-api companion-cli encrypted-json opfs)' '--limit[limit]:limit:' '--no-transcripts[skip transcripts]' '--no-panels[skip panels]'
          ;;
        export|snapshot)
          _arguments '--out[output directory]:directory:_files -/'
          ;;
        completion)
          _values 'shell' bash zsh
          ;;
      esac
      ;;
  esac
}
_graincrawl "$@"
`, commands)
}
