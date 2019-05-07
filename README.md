# History Watcher
Accelerate bash hisotry command.

# How to install
go get -u github.com/minoritea/history-watcher

# How to use
1. run history-watcher as a daemon.
    - You should run the command as a background service.
    - I recommend daemonize/supervisord.
2. add below lines to your `.bashrc`.
    - you need peco, fzf, or any other fuzzy finder tool.

```bash
function hp() {
  local action
  action=`curl -N -s localhost:14444 | peco`
  READLINE_LINE="${action}"
  READLINE_POINT="${#READLINE_LINE}"
}

bind -x '"\C-r": hp'
bind    '"\C-xr": hp'
```