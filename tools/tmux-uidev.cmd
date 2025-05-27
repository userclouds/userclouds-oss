new
neww $SHELL

# easy exit
bind-key k kill-session # Ctrl-b k to end
set-option -g status-right-length 80
set-option -g status-right "Ctrl-b k to end --- #{=21:pane_title} %H:%M %d-%b-%y"

# show titles
set-option -g pane-border-status bottom

# set up main top services window
# NB: "direnv reload" here is a bit brute force, but it's the easiest way to ensure that we're running
# the version of eg. golang that we want to be running, and not the system version
# https://github.com/direnv/direnv/issues/106 explains some of the issue, but tmux session
# inheritance is a bit of a mess
send-keys "direnv reload" Enter
send-keys "UC_PLEX_UI_DEV_PORT=3011 UC_CONSOLE_UI_DEV_PORT=3010 make services-dev" Enter
select-pane -T services

# set up react servers
split-window -v $SHELL
send-keys "direnv reload" Enter
send-keys "make plexui-dev" Enter
select-pane -T plexui

# TODO: for some reason, a reason upgrade broke tmux -p. Was: split-window -h -p 66 $SHELL
split-window -h $SHELL
send-keys "direnv reload" Enter
send-keys "make consoleui-dev" Enter
select-pane -T consoleui

# TODO: for some reason, a recent upgrade broke tmux -p. Was: split-window -h -p 50 $SHELL
split-window -h $SHELL
send-keys "direnv reload" Enter
send-keys "make sharedui-dev" Enter
select-pane -T sharedui

# foreground it
attach
