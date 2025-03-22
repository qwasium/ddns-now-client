# DDNS client for DDNS Now

DDNS client for [DDNS Now](https://ddns.kuku.lu/).

NOTE: This is my first project to use Go; more of a practice execise to getting used to the language, so don't trust this codebaes!

## NOTICE: this script will not work

The owner of ip4.me/ip6.me, Kevin Loch, passed away.

Due to ip4.me shutting down on Apr.1,2025, this script will not work in the near future.

## Install

### Config File

Create `config.yaml` in `~/.config/ddns-client`.

```bash
mkdir ~/.config/ddns-client
touch ~/.config/ddns-client/config.yaml
chmod 600 ~/.config/ddns-client/config.yaml
vim ~/.config/ddns-client/config.yaml
```

Write `config.yaml`.

```yaml
# ~/.config/ddns-client/config.yaml
waitTime: 1                # time to wait between HTTP request retry
maxAttempts: 5             # max # to retry HTTP request
checkIPChange: false       # Check if DNS points to correct IP address before updating DDNS
logPath: "ddns-client.log" # path to log file
domain: ""                 # domain name to solve
ddnsUser: ""               # User name of DDNS Now
token: ""                  # Token of DDNS Now
```
