package cli

const usage = `graincrawl archives Granola notes locally.

Usage:
  graincrawl [--json] [--config <path>] [--version] <command> [args]

Commands:
  version                 Show build metadata.
  check-update            Check for a newer graincrawl release.
  init                    Create config and database directories.
  doctor                  Inspect Granola and graincrawl state.
  metadata                Print crawlkit control metadata.
  status                  Show archive status.
  sync                    Sync from a source.
  refresh                 Alias for sync.
  runs                    List sync runs.
  notes                   List archived notes.
  search <query>          Search archived notes.
  sql <query>             Run read-only SQL against the local archive.
  note get <id>           Show one archived note.
  transcripts get <id>    Show transcript chunks for a note.
  panels get <id>         Show panels for a note.
  people                  List retained people source objects.
  workspaces              List retained workspace source objects.
  sources                 Show source adapter support.
  unlock                  Explain explicit unlock surfaces.
  secrets                 Inspect graincrawl-managed secret state.
  export markdown         Export notes as Markdown.
  snapshot create         Create a portable crawlkit snapshot.
  import <path>           Import a portable crawlkit snapshot.
  tui                     Browse archived notes in the terminal.
  completion              Print shell completion for bash or zsh.

Examples:
  graincrawl doctor --json
  graincrawl sync --source private-api
  graincrawl sync --source desktop-cache
  graincrawl notes --json
  graincrawl sql "select count(*) as notes from notes"
`
