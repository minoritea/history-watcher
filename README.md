# history-watcher

`history-watcher` is a tool designed to optimize and enhance the command history experience in bash/zsh by addressing common issues faced by power users. This tool aids in efficiently searching through a large command history by integrating with other fuzzy finders.

## Problems Solved

- **Performance**: Improves the speed of history retrieval in bash when dealing with extensive history records.
- **Duplicates**: Eliminates the clutter of repeated commands in your history to streamline the search process.

## Installation

Install `history-watcher` by running the following command:

```bash
go install github.com/minoritea/history-watcher@latest
```

## How to use

To use `history-watcher`, perform the following steps:

### bash

1. Install a fuzzy finder like `peco` or `fzf`.
2. Launch `history-watcher` as a background service.
   [goemon](https://github.com/minoritea/goemon) is a useful tool for managing background services.
   ```bash
   goemon -- history-watcher
   ```
3. Define a function in your `.bashrc` to interact with `history-watcher` using the chosen fuzzy finder:
   ```bash
   function hp() {
     local action
     action=$(curl -N -s localhost:14444 | peco)
     READLINE_LINE="${action}"
     READLINE_POINT="${#READLINE_LINE}"
   }
   bind -x '"\C-r": hp'
   bind '"\C-xr": hp'
   ```
4. Retrieve your history using the regular bash history shortcut keys.

#### NOTE

If you want to specify a custom history format, split the format into two lines:
one for the execution command and one for the metadata.
Prefix the metadata line with `#`, such as for timestamps.

### zsh

1. Install a fuzzy finder like `peco` or `fzf`.
2. Launch `history-watcher` as a background service.
   [goemon](https://github.com/minoritea/goemon) is a useful tool for managing background services.
   ```zsh
   goemon -- env HW_HISTFILE_FORMAT=zsh_extended history-watcher
   ```
3. Define a function in your `.zshrc` to interact with `history-watcher` using the chosen fuzzy finder:
   ```zsh
   function __search_history() {
     BUFFER=`curl -N -s localhost:14444 | peco`
     CURSOR=$#BUFFER
     zle reset-prompt
   }
   zle -N __search_history
   bindkey '^R' __search_history
   ```
4. Retrieve your history using the regular zsh history shortcut keys.

## Implementation

history-watcher monitors the .bash_history file, captures command history, and stores it in an ordered set sorted by the last execution time. Therefore, it retains command history without duplication as long as it is the same command invocation. It also provides an HTTP endpoint, so when accessed from an HTTP client, it returns the stored command history. history-watcher includes all command history in the HTTP response, but it returns them in ascending order as a stream. This allows you to start narrowing down and selecting results without waiting for the retrieval of all results by combining it with a fuzzy finder that supports stream input.

## Options

You can change the behavior of the tool by defining the following environment variables:

- **HW_HOST**: The address on which the server binds(default: 127.0.0.1)
- **HW_PORT**: The port number on which the server binds(default: 14444)
- **HW_TOKEN**: Required for authentication token for retrieving command history(default: NONE)
- **HW_DB_FILE**: Persists command history to a file or database(default: ~/.cache/history-watcher.db)
  - to disable persistence, set HW_DB_FILE to `-`
- **HW_HISTFILE**: Shell's history file(default: ~/.bash_history, when zsh or zsh_extended is specified as HW_HISTFILE_FORMAT, ~/.zsh_history)
- **HW_HISTFILE_FORMAT**: Shell's history file format(default: bash)
  - **bash**: Default format.
  - **zsh**: Zsh's history file format.
  - **zsh_extended**: Extended Zsh's history file format.
- **HW_POLL**: Polls the .bash_history file for monitoring(default: false)

## License

`history-watcher` is licensed under the GNU General Public License v3.0.
Feel free to contribute to the project, report issues, or suggest improvements on GitHub.
