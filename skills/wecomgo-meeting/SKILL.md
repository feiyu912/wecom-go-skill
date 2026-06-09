---
name: wecomgo-meeting
description: Manage WeCom meetings through wecom-go, including create, update, list, get, and cancel flows. Use this for requests like creating a meeting, changing meeting details, checking a user's meetings, viewing meeting details, or canceling a meeting. Return JSON suitable for agent consumption.
metadata:
  requires:
    bins: ["wecom-go"]
  cliHelp: "wecom-go meeting --help"
---

# wecomgo-meeting

Use `wecom-go` to access WeCom meeting capabilities.

## Core Commands

### 1. Create a meeting

```bash
wecom-go meeting create --admin-userid zhangsan --invitees "zhangsan,lisi" --title "Project Review" --start 2026-05-07T15:00:00+08:00 --duration 3600
```

JSON is also supported:

```bash
wecom-go meeting create '{"admin_userid":"zhangsan","title":"Project Review","meeting_start":1786134000,"meeting_duration":3600,"invitees":{"userid":["zhangsan","lisi"]}}'
```

### 2. Update a meeting

```bash
wecom-go meeting update --meeting-id MEETING_ID --title "Updated Topic" --start 2026-05-07T16:00:00+08:00 --duration 5400
```

JSON is also supported:

```bash
wecom-go meeting update '{"meetingid":"MEETING_ID","title":"Updated Topic","meeting_duration":5400}'
```

### 3. List a user's meetings

```bash
wecom-go meeting list --userid zhangsan --start 2026-05-07T00:00:00+08:00 --end 2026-05-07T23:59:59+08:00
```

### 4. Get meeting details

```bash
wecom-go meeting get --meeting-id MEETING_ID
```

### 5. Cancel a meeting

```bash
wecom-go meeting cancel --meeting-id MEETING_ID
```

## Usage Rules

- When creating a meeting, if `--start` is missing, the CLI defaults to the next full hour.
- When creating a meeting, if `--invitees` is missing, the CLI defaults to using `admin-userid` as the only invitee.
- When creating a meeting, if `--admin-userid` is missing but exactly one invitee is provided, that invitee is used as the admin.
- When updating a meeting, only explicitly provided fields are sent, so existing fields are not overwritten by create-style defaults.
- `meeting list` requires an explicit `userid`; this MVP does not resolve names to user IDs.
- `meeting cancel` is destructive, so upper-layer agents should confirm before calling it.

## Recommended Workflow

### Create a meeting

1. Collect `title`, `userid`, time, and duration first.
2. If the user gives only one `userid`, it can be used as both `admin_userid` and the only invitee.
3. Call `meeting create` and return fields such as `meetingid`, `meeting_code`, and `meeting_link`.

### Query meetings

1. Call `meeting list` to get candidate `meetingid` values.
2. If details are needed, call `meeting get` for the selected meeting.

### Update a meeting

1. Make sure you already have a `meetingid`. If the user only gives a title or time, first narrow it down with `meeting list` and optionally `meeting get`.
2. Send only the fields that really need to change.
3. Call `meeting update` and return the result.

### Locate by title, then update

Use this flow when the user says things like "change today's Project Review meeting to 4pm" but does not know the `meetingid`.

1. Collect or infer the meeting owner `userid`.
2. Collect a time window large enough to find the meeting, such as today, tomorrow, or a specific date range.
3. Call `meeting list --userid <userid> --start <time> --end <time>` to get candidate `meetingid` values.
4. Call `meeting get` for the returned meetings and filter by title, time, organizer, or invitees at the upper layer.
5. If exactly one meeting matches, call `meeting update --meeting-id <id> ...` with only the changed fields.
6. If no meetings match, tell the user no target meeting was found in that range.
7. If multiple meetings match, ask the user a narrow disambiguation question before updating.

Recommended disambiguation fields:

- Meeting title
- Scheduled start time
- Organizer or admin userid
- Invitees

Example flow:

```text
User intent: Change Zhangsan's "Project Review" meeting today to 16:00.

1. meeting list --userid zhangsan --start 2026-05-11T00:00:00+08:00 --end 2026-05-11T23:59:59+08:00
2. meeting get --meeting-id <candidate_1>
3. meeting get --meeting-id <candidate_2>
4. Filter by title "Project Review"
5. meeting update --meeting-id <matched_id> --start 2026-05-11T16:00:00+08:00
```

### Cancel a meeting

1. If only a title is known and `meetingid` is missing, first use `meeting list` and upper-layer filtering to identify the target.
2. Confirm with the user before calling `meeting cancel`.
