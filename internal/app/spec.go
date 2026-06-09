package app

import (
	"fmt"
	"strings"
)

type CommandSpec struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Status      string           `json:"status"`
	Subcommands []SubcommandSpec `json:"subcommands"`
	ShellNotes  []string         `json:"shell_notes,omitempty"`
}

type SubcommandSpec struct {
	Name        string    `json:"name"`
	Summary     string    `json:"summary"`
	Usage       string    `json:"usage"`
	Examples    []string  `json:"examples,omitempty"`
	Resolution  []string  `json:"resolution,omitempty"`
	SafetyNotes []string  `json:"safety_notes,omitempty"`
	JSONInput   JSONInput `json:"json_input"`
	Args        []ArgSpec `json:"args"`
}

type JSONInput struct {
	Supported      bool   `json:"supported"`
	Mode           string `json:"mode,omitempty"`
	MutuallyExcl   string `json:"mutually_exclusive_with,omitempty"`
	StdinSupported bool   `json:"stdin_supported,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

type ArgSpec struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"default,omitempty"`
	MapsTo       string `json:"maps_to,omitempty"`
	Description  string `json:"description"`
}

type CommandCatalog struct {
	Version         int           `json:"version"`
	Description     string        `json:"description"`
	CoveredCommands []string      `json:"covered_commands"`
	Commands        []CommandSpec `json:"commands"`
}

func specCatalog() CommandCatalog {
	commands := []CommandSpec{
		scheduleCommandSpec(),
		todoCommandSpec(),
	}
	return CommandCatalog{
		Version:         1,
		Description:     "Machine-readable command contract for upper-layer agents. Prefer selecting a command and filling arguments instead of free-form shell generation.",
		CoveredCommands: []string{"schedule", "todo"},
		Commands:        commands,
	}
}

func findCommandSpec(name string) (CommandSpec, bool) {
	for _, command := range specCatalog().Commands {
		if command.Name == name {
			return command, true
		}
	}
	return CommandSpec{}, false
}

func findSubcommandSpec(commandName string, subcommandName string) (SubcommandSpec, bool) {
	command, ok := findCommandSpec(commandName)
	if !ok {
		return SubcommandSpec{}, false
	}
	for _, subcommand := range command.Subcommands {
		if subcommand.Name == subcommandName {
			return subcommand, true
		}
	}
	return SubcommandSpec{}, false
}

func scheduleCommandSpec() CommandSpec {
	return CommandSpec{
		Name:        "schedule",
		Description: "Manage WeCom schedules, including create, update, get, list, delete, and attendee changes.",
		Status:      "stable",
		ShellNotes: []string{
			"Prefer first-class flags for common fields and reserve JSON payloads for complex reminder or recurrence structures.",
			"On PowerShell, prefer a single quoted positional JSON payload or --payload-file over raw echo pipelines.",
		},
		Subcommands: []SubcommandSpec{
			{
				Name:    "create",
				Summary: "Create a schedule.",
				Usage:   "wecom-go schedule create [--admins \"a,b\"] [--attendees \"u1,u2\"] [--start <time>] [--end <time>] [--summary <text>] [--description <text>] [--location <text>] [--cal-id <id>] [--whole-day] [--remind-before <secs>] [--agentid <id>] [json_payload]",
				Examples: []string{
					"wecom-go schedule create --admins \"zhangsan\" --attendees \"lisi,wangwu\" --summary \"Project Review\" --start 2026-05-12T15:00:00+08:00 --end 2026-05-12T16:00:00+08:00 --location \"Room 1005\"",
					"wecom-go schedule create '{\"schedule\":{\"admins\":[\"zhangsan\"],\"start_time\":1778578800,\"end_time\":1778582400,\"summary\":\"Project Review\",\"reminders\":{\"is_remind\":1,\"remind_before_event_secs\":3600}}}'",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
					Notes:          "Complex reminder and recurrence fields should prefer JSON input. --payload-file - reads JSON from stdin.",
				},
				Args: []ArgSpec{
					{Name: "--admins", Type: "csv<string>", Required: false, MapsTo: "schedule.admins", Description: "Organizer user IDs."},
					{Name: "--attendees", Type: "csv<string>", Required: false, MapsTo: "schedule.attendees[].userid", Description: "Attendee user IDs."},
					{Name: "--start", Type: "time", Required: false, MapsTo: "schedule.start_time", Description: "Schedule start time. Accepts ISO datetime or epoch."},
					{Name: "--end", Type: "time", Required: false, MapsTo: "schedule.end_time", Description: "Schedule end time. Accepts ISO datetime or epoch."},
					{Name: "--summary", Type: "string", Required: false, MapsTo: "schedule.summary", Description: "Schedule summary."},
					{Name: "--description", Type: "string", Required: false, MapsTo: "schedule.description", Description: "Schedule description."},
					{Name: "--location", Type: "string", Required: false, MapsTo: "schedule.location", Description: "Schedule location."},
					{Name: "--cal-id", Type: "string", Required: false, MapsTo: "schedule.cal_id", Description: "Target calendar ID."},
					{Name: "--whole-day", Type: "bool", Required: false, MapsTo: "schedule.is_whole_day", Description: "Marks the schedule as whole-day when present."},
					{Name: "--remind-before", Type: "int", Required: false, MapsTo: "schedule.reminders.remind_before_event_secs", Description: "Reminder lead time in seconds."},
					{Name: "--agentid", Type: "int", Required: false, MapsTo: "agentid", Description: "Optional agent ID."},
					{Name: "--payload-file", Type: "path", Required: false, DefaultValue: "", Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "update",
				Summary: "Update an existing schedule.",
				Usage:   "wecom-go schedule update --schedule-id <id> [--op-mode <int>] [--op-start-time <time>] [--admins \"a,b\"] [--attendees \"u1,u2\"] [--start <time>] [--end <time>] [--summary <text>] [--description <text>] [--location <text>] [--cal-id <id>] [--whole-day] [--remind-before <secs>] [json_payload]",
				Resolution: []string{
					"Resolve the target schedule_id before updating.",
					"Send only the fields that should change.",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
					Notes:          "For repeated schedules, use op-mode and op-start-time when targeting a specific occurrence.",
				},
				Args: []ArgSpec{
					{Name: "--schedule-id", Type: "string", Required: true, MapsTo: "schedule.schedule_id", Description: "Target schedule ID unless already provided in JSON."},
					{Name: "--op-mode", Type: "int", Required: false, MapsTo: "op_mode", Description: "Repeat schedule operation mode."},
					{Name: "--op-start-time", Type: "time", Required: false, MapsTo: "op_start_time", Description: "Repeat schedule instance start time."},
					{Name: "--admins", Type: "csv<string>", Required: false, MapsTo: "schedule.admins", Description: "Organizer user IDs."},
					{Name: "--attendees", Type: "csv<string>", Required: false, MapsTo: "schedule.attendees[].userid", Description: "Attendee user IDs."},
					{Name: "--start", Type: "time", Required: false, MapsTo: "schedule.start_time", Description: "Updated start time."},
					{Name: "--end", Type: "time", Required: false, MapsTo: "schedule.end_time", Description: "Updated end time."},
					{Name: "--summary", Type: "string", Required: false, MapsTo: "schedule.summary", Description: "Updated summary."},
					{Name: "--description", Type: "string", Required: false, MapsTo: "schedule.description", Description: "Updated description."},
					{Name: "--location", Type: "string", Required: false, MapsTo: "schedule.location", Description: "Updated location."},
					{Name: "--cal-id", Type: "string", Required: false, MapsTo: "schedule.cal_id", Description: "Updated calendar ID."},
					{Name: "--whole-day", Type: "bool", Required: false, MapsTo: "schedule.is_whole_day", Description: "Marks the schedule as whole-day when present."},
					{Name: "--remind-before", Type: "int", Required: false, MapsTo: "schedule.reminders.remind_before_event_secs", Description: "Reminder lead time in seconds."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "get",
				Summary: "Fetch schedule details by ID.",
				Usage:   "wecom-go schedule get --schedule-ids \"id1,id2\" [json_payload]",
				JSONInput: JSONInput{
					Supported:    true,
					Mode:         "positional_json_or_payload_file",
					MutuallyExcl: "positional JSON payload and --payload-file",
				},
				Args: []ArgSpec{
					{Name: "--schedule-ids", Type: "csv<string>", Required: true, MapsTo: "schedule_id_list", Description: "One or more schedule IDs unless already provided in JSON."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "list",
				Summary: "List schedules from a calendar.",
				Usage:   "wecom-go schedule list --cal-id <id> [--offset 0] [--limit 100] [json_payload]",
				Resolution: []string{
					"Use this when the schedule ID is unknown but the calendar is known.",
				},
				JSONInput: JSONInput{
					Supported:    true,
					Mode:         "positional_json_or_payload_file",
					MutuallyExcl: "positional JSON payload and --payload-file",
				},
				Args: []ArgSpec{
					{Name: "--cal-id", Type: "string", Required: true, MapsTo: "cal_id", Description: "Calendar ID unless already provided in JSON."},
					{Name: "--offset", Type: "int", Required: false, DefaultValue: "0", MapsTo: "offset", Description: "Pagination offset."},
					{Name: "--limit", Type: "int", Required: false, DefaultValue: "100", MapsTo: "limit", Description: "Page size."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "delete",
				Summary: "Delete a schedule.",
				Usage:   "wecom-go schedule delete --schedule-id <id> [--op-mode <int>] [--op-start-time <time>] [json_payload]",
				Resolution: []string{
					"Resolve the exact schedule_id first.",
				},
				SafetyNotes: []string{
					"Destructive action. Confirm with the user before calling delete.",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
				},
				Args: []ArgSpec{
					{Name: "--schedule-id", Type: "string", Required: true, MapsTo: "schedule_id", Description: "Target schedule ID unless already provided in JSON."},
					{Name: "--op-mode", Type: "int", Required: false, MapsTo: "op_mode", Description: "Repeat schedule operation mode."},
					{Name: "--op-start-time", Type: "time", Required: false, MapsTo: "op_start_time", Description: "Repeat schedule instance start time."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "add-attendees",
				Summary: "Add attendees to a schedule.",
				Usage:   "wecom-go schedule add-attendees --schedule-id <id> --attendees \"u1,u2\" [json_payload]",
				Resolution: []string{
					"Resolve the target schedule_id before changing attendees.",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
				},
				Args: []ArgSpec{
					{Name: "--schedule-id", Type: "string", Required: true, MapsTo: "schedule_id", Description: "Target schedule ID unless already provided in JSON."},
					{Name: "--attendees", Type: "csv<string>", Required: true, MapsTo: "attendees[].userid", Description: "Attendee user IDs unless already provided in JSON."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "remove-attendees",
				Summary: "Remove attendees from a schedule.",
				Usage:   "wecom-go schedule remove-attendees --schedule-id <id> --attendees \"u1,u2\" [json_payload]",
				Resolution: []string{
					"Resolve the target schedule_id before changing attendees.",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
				},
				Args: []ArgSpec{
					{Name: "--schedule-id", Type: "string", Required: true, MapsTo: "schedule_id", Description: "Target schedule ID unless already provided in JSON."},
					{Name: "--attendees", Type: "csv<string>", Required: true, MapsTo: "attendees[].userid", Description: "Attendee user IDs unless already provided in JSON."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
		},
	}
}

func todoCommandSpec() CommandSpec {
	return CommandSpec{
		Name:        "todo",
		Description: "Manage WeCom todos, including create, list, get, update, delete, and status changes.",
		Status:      "stable",
		ShellNotes: []string{
			"Prefer first-class flags for common fields.",
			"On Windows PowerShell, prefer --content / --remind-time / --status or a single quoted JSON payload over raw echo pipelines.",
		},
		Subcommands: []SubcommandSpec{
			{
				Name:    "create",
				Summary: "Create a todo.",
				Usage:   "wecom-go todo create [--content <text>] [--remind-time <time>] [--status <int>] [json_payload]",
				Examples: []string{
					"wecom-go todo create --content \"Submit weekly report\" --remind-time 2026-05-12T18:00:00+08:00 --status 1",
					"wecom-go todo create '{\"content\":\"Submit weekly report\",\"remind_time\":1778570400,\"todo_status\":1}'",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
					Notes:          "If the fields are more complex than the first-class flags cover, pass a JSON payload or --payload-file. --payload-file - reads JSON from stdin.",
				},
				Args: []ArgSpec{
					{Name: "--content", Type: "string", Required: false, MapsTo: "content", Description: "Todo content."},
					{Name: "--remind-time", Type: "time", Required: false, MapsTo: "remind_time", Description: "Reminder time. Accepts ISO datetime or epoch."},
					{Name: "--status", Type: "int", Required: false, MapsTo: "todo_status", Description: "Initial todo status."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "list",
				Summary: "List todos in a time range.",
				Usage:   "wecom-go todo list [--start <time>] [--end <time>] [--cursor <cursor>] [--limit 100] [json_payload]",
				Resolution: []string{
					"Use this first when the todo ID is unknown but the date range is known.",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
				},
				Args: []ArgSpec{
					{Name: "--start", Type: "time", Required: false, MapsTo: "start_time", Description: "Range start time."},
					{Name: "--end", Type: "time", Required: false, MapsTo: "end_time", Description: "Range end time."},
					{Name: "--cursor", Type: "string", Required: false, MapsTo: "cursor", Description: "Pagination cursor."},
					{Name: "--limit", Type: "int", Required: false, DefaultValue: "100", MapsTo: "limit", Description: "Page size."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "get",
				Summary: "Fetch todo details by ID.",
				Usage:   "wecom-go todo get --todo-ids \"id1,id2\" [json_payload]",
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
				},
				Args: []ArgSpec{
					{Name: "--todo-ids", Type: "csv<string>", Required: true, MapsTo: "todo_id_list", Description: "One or more todo IDs unless already provided in JSON."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "update",
				Summary: "Update an existing todo.",
				Usage:   "wecom-go todo update --todo-id <id> [--content <text>] [--remind-time <time>] [--status <int>] [json_payload]",
				Resolution: []string{
					"Resolve the target todo_id before updating.",
					"Send only the fields that should change.",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
				},
				Args: []ArgSpec{
					{Name: "--todo-id", Type: "string", Required: true, MapsTo: "todo_id", Description: "Target todo ID unless already provided in JSON."},
					{Name: "--content", Type: "string", Required: false, MapsTo: "content", Description: "Updated todo content."},
					{Name: "--remind-time", Type: "time", Required: false, MapsTo: "remind_time", Description: "Updated reminder time."},
					{Name: "--status", Type: "int", Required: false, MapsTo: "todo_status", Description: "Updated todo status."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "delete",
				Summary: "Delete a todo.",
				Usage:   "wecom-go todo delete --todo-id <id> [json_payload]",
				Resolution: []string{
					"Resolve the exact todo_id before deleting.",
				},
				SafetyNotes: []string{
					"Destructive action. Confirm with the user before calling delete.",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
				},
				Args: []ArgSpec{
					{Name: "--todo-id", Type: "string", Required: true, MapsTo: "todo_id", Description: "Target todo ID unless already provided in JSON."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
			{
				Name:    "change-status",
				Summary: "Change todo completion status.",
				Usage:   "wecom-go todo change-status --todo-id <id> --status <int> [json_payload]",
				Resolution: []string{
					"Resolve the target todo_id before changing status.",
				},
				JSONInput: JSONInput{
					Supported:      true,
					Mode:           "positional_json_or_payload_file",
					MutuallyExcl:   "positional JSON payload and --payload-file",
					StdinSupported: true,
				},
				Args: []ArgSpec{
					{Name: "--todo-id", Type: "string", Required: true, MapsTo: "todo_id", Description: "Target todo ID unless already provided in JSON."},
					{Name: "--status", Type: "int", Required: true, MapsTo: "todo_status", Description: "Desired status unless already provided in JSON."},
					{Name: "--payload-file", Type: "path", Required: false, Description: "Read a JSON payload from file. Use - to read stdin."},
				},
			},
		},
	}
}

func renderCommandHelp(commandName string) string {
	command, ok := findCommandSpec(commandName)
	if !ok {
		return ""
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "%s commands:\n", command.Name)
	for _, subcommand := range command.Subcommands {
		fmt.Fprintf(&builder, "  %s\n", subcommand.Usage)
	}

	if len(command.ShellNotes) > 0 {
		builder.WriteString("\nShell notes:\n")
		for _, note := range command.ShellNotes {
			fmt.Fprintf(&builder, "  - %s\n", note)
		}
	}

	builder.WriteString("\nSpec:\n")
	fmt.Fprintf(&builder, "  wecom-go spec %s\n", command.Name)
	return strings.TrimRight(builder.String(), "\n")
}
