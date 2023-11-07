# history-watcher
`history-watcher` is a tool designed to optimize and enhance the command history experience in bash by addressing common issues faced by power users. This tool aids in efficiently searching through a large command history by integrating with other fuzzy finders.

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

1. Launch `history-watcher` as a background service.
2. Install a fuzzy finder like `peco` or `fzf`.
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

## Implementation
history-watcher monitors the .bash_history file, captures command history, and stores it in an ordered set sorted by the last execution time. Therefore, it retains command history without duplication as long as it is the same command invocation. It also provides an HTTP endpoint, so when accessed from an HTTP client, it returns the stored command history. history-watcher includes all command history in the HTTP response, but it returns them in ascending order as a stream. This allows you to start narrowing down and selecting results without waiting for the retrieval of all results by combining it with a fuzzy finder that supports stream input.

## Options
You can change the behavior of the tool by defining the following environment variables:

- **HW_HOST**: The address on which the server binds
- **HW_PORT**: The port number on which the server binds
- **HW_TOKEN**: Required for authentication token for retrieving command history
- **HW_DB_FILE**: Persists command history to a file or database
- **HW_POLL**: Polls the .bash_history file for monitoring

By default, it binds to 127.0.0.1:14444 for the address and port. It does not listen on a global address, so it is safe enough for personal machines. If you are concerned about using a shared machine or being read by spyware, you can set HW_TOKEN to require token authentication at invocation. Please include the token in the Authorization header using the Bearer authentication method when sending a request to the server. By default, the command history is not saved, and the .bash_history file is reloaded each time the service starts. If you want to persist the history, please enter the destination file name in HW_DB_FILE. Monitoring of the .bash_history is usually done using INOTIFY, but it may not work well with some file systems used in WSL, etc. In such cases, you can enable polling by entering 1 or true in HW_POLL.

## License
`history-watcher` is licensed under the GNU General Public License v3.0.
Feel free to contribute to the project, report issues, or suggest improvements on GitHub.
