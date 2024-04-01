# Tmux Session Manager (tsm)

`tsm` provides the ability to quickly switch between tmux sessions.
Each session is tied to a directory allowing projects to be isolated, but in active development.
To simplify switching sessions, `fzf` is used.

## Installation

`tsm` assumes that the following binaries exist within your path:

- [`tmux`](https://github.com/tmux/tmux)
- [`fzf`](https://github.com/junegunn/fzf)

To install from source, run:

```sh
$ go install github.com/mattmeyers/tsm@latest
```

## Usage

```
tsm - The Tmux Session Manager

tsm manages your tmux sessions by creating a new session per project directory.
Sessions may contain multiple windows which are isolated and maintained when
switching between projects. Omitting any commands will trigger the session
switcher.

USAGE:
    tsm [OPTIONS] [COMMAND]

COMMANDS:
    0                     Switch to the zero session.

OPTIONS:
    -h, --help            Show this help message.
```

Upon first run of `tsm`, a fresh configuration file is placed in `{config dir}/tsm`.
On linux, this corresponds to `~/.config/tsm`.
This configuration file contains the directories to search in and which directories to ignore.
To get started with `tsm`, place some directory paths in the `base_dirs` array.
All child directories within these configured directories will be listed the next time `tsm` is run.
Note that `tsm` does not recursively list directories; only direct children are listed.
Optionally, ignore patterns can be provided in the `ignore_dirs` array to omit them from the list.
For example, `.git` directories can be ignored.

Invoking the `tsm` command with no subcommand triggers the session switcher.
This requires `fzf` to be installed, otherwise `tsm` will exit.
Selecting a directory will trigger a session creation if a session does not already exist for the target directory.
If a session does exist, then tmux will simply switch sessions.
The `0` subcommand switches to the zero session which is not tied to any specific directory

## Inspiration

This is based on the ideas from ThePrimeagen's [tmux-sessionizer] script.
It generally works the same, but mixes in some personal preferences and requirements.

[tmux-sessionizer]: https://github.com/ThePrimeagen/.dotfiles/blob/602019e902634188ab06ea31251c01c1a43d1621/bin/.local/scripts/tmux-sessionizer
