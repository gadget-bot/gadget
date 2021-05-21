# Gadget

TODO: Make a real README

## Starting (on your local)

```sh
#!/bin/zsh

export GADGET_GLOBAL_ADMINS="U0.....,U1....."
export SLACK_OAUTH_TOKEN="xoxb-...."
export SLACK_SIGNING_SECRET="a...a"
export GADGET_DB_USER="gadgetuser"
export GADGET_DB_PASS="secretpassword"
# MySQL/MariaDB host and port
export GADGET_DB_HOST="127.0.0.1:3306"
# DB name
export GADGET_DB_NAME="gadget_dev"

go run .
```
