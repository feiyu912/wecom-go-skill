---
name: wecomgo-todo
description: Manage WeCom todos through wecom-go, including create, list, get, update, delete, and status-change flows. Use this for requests like creating a todo, checking pending items, reading todo details, editing todo content, deleting a todo, or marking a todo done. Return JSON suitable for agent consumption.
metadata:
  requires:
    bins: ["wecom-go"]
  cliHelp: "wecom-go todo"
---

# wecomgo-todo

Use `wecom-go` to access WeCom todo capabilities.

## Core Commands

### 1. Create a todo

```bash
wecom-go todo create --content "Submit weekly report" --remind-time 2026-05-12T18:00:00+08:00 --status 1
```

JSON is also supported:

```bash
wecom-go todo create '{"content":"Submit weekly report","remind_time":1778570400,"todo_status":1}'
```

On PowerShell, prefer first-class flags or a single quoted positional JSON payload.
Avoid generating commands like `echo {...} | wecom-go todo create --payload-file -` unless the JSON is properly quoted/serialized first, because raw `{...}` is parsed by PowerShell before it reaches the CLI.

### 2. List todos

```bash
wecom-go todo list --start 2026-05-12T00:00:00+08:00 --end 2026-05-12T23:59:59+08:00 --limit 100
```

### 3. Get todo details

```bash
wecom-go todo get --todo-ids "TODO_ID_1,TODO_ID_2"
```

### 4. Update a todo

```bash
wecom-go todo update --todo-id TODO_ID --content "Submit weekly report and attach slides" --remind-time 2026-05-12T19:00:00+08:00
```

### 5. Delete a todo

```bash
wecom-go todo delete --todo-id TODO_ID
```

### 6. Change todo status

```bash
wecom-go todo change-status --todo-id TODO_ID --status 2
```

## Usage Rules

- The CLI exposes common fields through flags and keeps JSON payload support for advanced fields.
- On Windows PowerShell, prefer `--content` / `--remind-time` / `--status` or a single quoted JSON positional argument over raw `echo {json}` pipelines.
- `todo update` sends only explicitly provided fields, so existing values are not overwritten by defaults.
- `todo get` accepts multiple IDs through `--todo-ids`.
- `todo delete` is destructive, so upper-layer agents should confirm before calling it.
- If the needed todo fields are more complex than the first-class flags cover, pass a JSON payload or `--payload-file`. `--payload-file -` reads JSON from stdin.

## Recommended Workflow

### Create a todo

1. Collect the content and, if relevant, remind time and initial status.
2. Call `todo create`.
3. Return the created todo identifier and key summary fields.

### Query todos

1. If the todo id is known, call `todo get`.
2. If only a date range or "today's todos" style request is known, call `todo list` first.
3. If the list response is not detailed enough, call `todo get` for the selected ids.

### Update a todo

1. Make sure the target todo id is known. If not, locate candidates through `todo list` first.
2. Send only the fields that really need to change.
3. Call `todo update` and return the result.

### Complete a todo

1. If only the todo content or date range is known, find the candidate through `todo list` and optionally `todo get`.
2. Call `todo change-status` with the target id and the desired status.

### Delete a todo

1. Resolve the exact todo id before deleting.
2. Confirm with the user before calling `todo delete`.
