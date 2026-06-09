---
name: wecomgo-schedule
description: Manage WeCom calendars and schedules through wecom-go, including create, update, get, list, delete, and attendee-management flows. Use this for requests like creating a calendar, creating a schedule, checking a calendar's schedules, updating schedule details, canceling a schedule, or adding/removing attendees. Return JSON suitable for agent consumption.
metadata:
  requires:
    bins: ["wecom-go"]
  cliHelp: "wecom-go schedule --help"
---

# wecomgo-schedule

Use `wecom-go` to access WeCom calendar and schedule capabilities.

## Core Commands

### 1. Create a calendar

```bash
wecom-go calendar create --admins "zhangsan,lisi" --summary "Team Calendar" --color "#FF3030"
```

### 2. Create a schedule

```bash
wecom-go schedule create --admins "zhangsan" --attendees "lisi,wangwu" --summary "Project Review" --start 2026-05-12T15:00:00+08:00 --end 2026-05-12T16:00:00+08:00 --location "Room 1005"
```

JSON payloads are allowed only when the exact schedule schema is already known from current API/CLI docs or an existing known-good payload. Do not invent reminder or recurrence fields.

The first-class create flags expose a single pre-event reminder through `--remind-before`; they do not support “remind every N minutes during the event”.

### 3. List schedules in a calendar

```bash
wecom-go schedule list --cal-id CAL_ID --offset 0 --limit 100
```

### 4. Get schedule details

```bash
wecom-go schedule get --schedule-ids "SCHEDULE_ID_1,SCHEDULE_ID_2"
```

### 5. Update a schedule

```bash
wecom-go schedule update --schedule-id SCHEDULE_ID --summary "Updated Topic" --start 2026-05-12T16:00:00+08:00 --end 2026-05-12T17:00:00+08:00
```

### 6. Add attendees

```bash
wecom-go schedule add-attendees --schedule-id SCHEDULE_ID --attendees "lisi,wangwu"
```

### 7. Remove attendees

```bash
wecom-go schedule remove-attendees --schedule-id SCHEDULE_ID --attendees "wangwu"
```

### 8. Cancel a schedule

```bash
wecom-go schedule delete --schedule-id SCHEDULE_ID
```

## Usage Rules

- `calendar create` and `schedule create` accept either first-class flags or a positional JSON payload.
- Prefer first-class flags; use JSON only for proven schema fields, not guessed reminder or recurrence structures.
- `schedule get` accepts multiple IDs through `--schedule-ids`.
- `schedule list` requires a concrete `cal_id`; never pass a userid as a list filter.
- `--remind-before` and `remind_time_diffs` are pre-event reminder offsets, not repeated reminders during the event.
- If a requested reminder/repeat rule is unsupported or rejected, stop and ask before creating a degraded schedule.
- Treat `schedule update` as read-modify-write: get the existing schedule, preserve unchanged fields, and then update.
- `schedule delete` is destructive, so upper-layer agents should confirm before calling it.

## Recommended Workflow

### Create a schedule

1. Collect the organizer `userid`, target `cal_id`, start/end time, and summary first.
2. If there are invitees, pass them through `--attendees`.
3. Confirm unsupported repeat/reminder requirements before any API mutation; never create a partial fallback without user approval.
4. Call `schedule create` and return identifiers such as `schedule_id`.

### Query schedules

1. If the schedule id is known, call `schedule get`.
2. If only a calendar is known, call `schedule list` first.
3. If a specific attendee change is needed, resolve the target schedule first and then call `schedule add-attendees` or `schedule remove-attendees`.

### Update a schedule

1. Make sure the target `schedule_id` is known.
2. Call `schedule get --schedule-ids <id>` and keep the existing title, start, end, calendar, reminder, attendee, and whole-day values.
3. Build the update from the existing schedule plus the requested changes; do not send a payload with only the changed field.
4. For repeated schedules, use `--op-mode` and `--op-start-time` when updating a specific occurrence.

### Cancel a schedule

1. Resolve the exact `schedule_id`.
2. For repeated schedules, include the repeat-operation fields when needed.
3. Confirm with the user before calling `schedule delete`.
