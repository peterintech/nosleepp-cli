# nosleepp

Keep your computer awake while AI agents are actively working.

`nosleepp` is built for agent tools such as Codex, Claude Code, OpenCode, Antigravity, and Cursor. It does not treat an app as working just because it is open. It samples process activity and only reports agents that show measurable CPU or child-process activity.

## Install Globally

Install it globally so the `nosleepp` command is available from any terminal:

```bash
npm install -g nosleepp
```

Then run:

```bash
nosleepp watch
```

## Quick Start

Check whether any agent is actively working:

```bash
nosleepp list
```

Keep your computer awake while agents are working:

```bash
nosleepp watch
```

Show matching agents even when they are open but idle:

```bash
nosleepp list --all
```

Check once and exit with a script-friendly status code:

```bash
nosleepp watch --once
```

## Supported Platforms

| Platform | Status |
| --- | --- |
| Windows x64 | Supported |
| macOS Apple Silicon | Supported |
| macOS Intel | Supported |
| Linux | Not supported yet |

`nosleepp` prevents system idle sleep. It does not force your display to stay awake.

## How It Detects Work

`nosleepp` samples running processes for 2 seconds by default.

An agent is marked `working` when either:

- the agent process or one of its descendant processes gains enough CPU time, or
- a descendant process appears or disappears during the sample window.

If an agent app is open but idle, `nosleepp list` hides it by default. Use `nosleepp list --all` to show idle matches too.

## Useful Commands

```bash
nosleepp list
nosleepp list --json
nosleepp list --all
nosleepp watch
nosleepp watch --interval 5s
nosleepp watch --quiet 1m
nosleepp version
```

## Add Custom Agents

Use `--agent` to add or override process names:

```bash
nosleepp watch --agent "MyAgent=myagent,myagent-helper"
```

You can pass `--agent` more than once.

## Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Success, or `watch --once` found a working agent |
| `1` | `watch --once` found no working agents |
| `2` | Invalid CLI usage |
| `3` | Platform power-state or watcher runtime error |
