# TerDash

A terminal dashboard built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss). Displays live system stats, weather, news headlines, and your Notion tasks — all in one glanceable TUI.

```
  _|_|_|_|_|                    _|_|_|                          _|
      _|      _|_|    _|  _|_|  _|    _|    _|_|_|    _|_|_|    _|_|_|
      _|    _|_|_|_|  _|_|      _|    _|  _|    _|  _|_|        _|    _|
      _|    _|        _|        _|    _|  _|    _|      _|_|    _|    _|
      _|      _|_|_|  _|        _|_|_|      _|_|_|  _|_|_|      _|    _|
```

## Features

- **Header** — current date, time, season, and live local weather via [wttr.in](https://wttr.in)
- **System Stats** — real-time CPU, RAM, and disk usage with visual progress bars
- **News** — latest headlines from NPR, The Guardian, Telex, and HVG, refreshed hourly
- **Notion Tasks** — fetches your incomplete tasks from a Notion database, sorted by due date; shows only tasks due today or earlier (urgent) first — once all urgent tasks are cleared, future tasks become visible; mark tasks as done directly from the terminal

## Keybindings

| Key            | Action                                        |
| -------------- | --------------------------------------------- |
| `j` / `↓`      | Move cursor down                              |
| `k` / `↑`      | Move cursor up                                |
| `Enter`        | Mark selected task as done                    |
| `Tab`          | Cycle task priority (Top → High → No Priority)|
| `p`            | Start / pause / resume Pomodoro               |
| `b`            | Start 5-min break (after Pomodoro finishes)   |
| `s`            | Stop Pomodoro / Break                         |
| `?`            | Toggle help                                   |
| `q` / `Ctrl+C` | Quit                                          |

## Requirements

- Go 1.21+
- A [Notion API key](https://www.notion.so/my-integrations) and a database ID

## Installation

```bash
git clone https://github.com/yourusername/TerDash
cd TerDash/src
go build -o terdash .
./terdash
```

On first run, you will be prompted to enter your Notion API key and database ID. These are saved to `~/.config/ter_dash/config.json` with `0600` permissions.

## Notion Setup

1. Go to [notion.so/my-integrations](https://www.notion.so/my-integrations) and create a new integration.
2. Copy the **Internal Integration Token** — this is your API key.
3. Open the Notion database you want to track and share it with your integration.
4. Copy the database ID from the URL: `notion.so/<workspace>/<database_id>?v=...`
5. Your database needs a **Status** property (type: Status), a **Priority** property (type: Select) with values `Top`, `High`, and `No Priority`, and optionally a **Due Date** property (type: Date).

## Planned Features

- **AI news summarizer** — summarize headlines using an LLM
- **News tabs** — switch between general, economic/stock, and Hungarian news with Tab
- **Network traffic** — add live network I/O to system stats
- **Better layout** — improved weather, date/time, and ASCII art formatting with a softer color palette
- **Pomodoro timer** — built-in focus timer panel
- **Calendar panel** — display upcoming calendar events
- **Help menu** — in-dashboard keybinding reference
- **Focus mode** — press `h` to hide news and stats for a distraction-free view

## Tech Stack

| Library                                                               | Purpose                          |
| --------------------------------------------------------------------- | -------------------------------- |
| [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm architecture) |
| [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)   | Terminal styling and layout      |
| [shirou/gopsutil](https://github.com/shirou/gopsutil)                 | System metrics (CPU, RAM, disk)  |
