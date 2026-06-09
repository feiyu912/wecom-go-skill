package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pangp/wecom-go-skill/internal/config"
	"github.com/pangp/wecom-go-skill/internal/wecom"
)

func Run(args []string) int {
	if len(args) == 0 {
		printHelp()
		return 0
	}

	switch args[0] {
	case "help", "--help", "-h":
		printHelp()
		return 0
	case "config":
		return runConfig(args[1:])
	case "token":
		return runToken(args[1:])
	case "contact":
		return runContact(args[1:])
	case "calendar":
		return runCalendar(args[1:])
	case "schedule":
		return runSchedule(args[1:])
	case "meeting":
		return runMeeting(args[1:])
	case "todo":
		return runTodo(args[1:])
	case "wedrive":
		return runWeDrive(args[1:])
	case "wedoc":
		return runWeDoc(args[1:])
	case "spec":
		return runSpec(args[1:])
	default:
		printError(fmt.Errorf("unknown command: %s", args[0]))
		printHelp()
		return 1
	}
}

func runConfig(args []string) int {
	if len(args) == 0 {
		printConfigHelp()
		return 0
	}

	switch args[0] {
	case "set":
		fs := flag.NewFlagSet("config set", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		corpID := fs.String("corp-id", "", "corp id")
		corpSecret := fs.String("corp-secret", "", "corp secret")
		baseURL := fs.String("base-url", "", "base url")
		timeout := fs.Int("timeout", 30, "timeout in seconds")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}
		if strings.TrimSpace(*corpID) == "" || strings.TrimSpace(*corpSecret) == "" {
			printError(errors.New("`config set` requires --corp-id and --corp-secret"))
			return 1
		}
		cfg := config.FileConfig{
			CorpID:     strings.TrimSpace(*corpID),
			CorpSecret: strings.TrimSpace(*corpSecret),
			BaseURL:    strings.TrimSpace(*baseURL),
			TimeoutSec: *timeout,
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://qyapi.weixin.qq.com"
		}
		if err := config.SaveFile(cfg); err != nil {
			printError(err)
			return 1
		}
		path, _ := config.ConfigPath()
		printJSON(map[string]any{
			"message":     "config saved",
			"path":        path,
			"corp_id":     cfg.CorpID,
			"corp_secret": config.MaskSecret(cfg.CorpSecret),
			"base_url":    cfg.BaseURL,
			"timeout_sec": cfg.TimeoutSec,
		})
		return 0
	case "show":
		cfg, err := config.LoadFile()
		if err != nil {
			printError(err)
			return 1
		}
		path, _ := config.ConfigPath()
		printJSON(map[string]any{
			"path":        path,
			"corp_id":     cfg.CorpID,
			"corp_secret": config.MaskSecret(cfg.CorpSecret),
			"base_url":    cfg.BaseURL,
			"timeout_sec": cfg.TimeoutSec,
		})
		return 0
	case "path":
		path, err := config.ConfigPath()
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(map[string]any{"path": path})
		return 0
	case "clear":
		path, _ := config.ConfigPath()
		if err := config.ClearFile(); err != nil {
			printError(err)
			return 1
		}
		printJSON(map[string]any{"message": "config cleared", "path": path})
		return 0
	default:
		printError(fmt.Errorf("unknown config subcommand: %s", args[0]))
		printConfigHelp()
		return 1
	}
}

func runToken(args []string) int {
	fs := flag.NewFlagSet("token", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	debug := fs.Bool("debug", false, "print request debug info")
	refresh := fs.Bool("refresh", false, "force refresh access token")
	if err := fs.Parse(args); err != nil {
		printError(err)
		return 1
	}

	client, err := newClient(*debug)
	if err != nil {
		printError(err)
		return 1
	}
	result, err := client.AccessToken(*refresh)
	if err != nil {
		printError(err)
		return 1
	}
	printJSON(result)
	return 0
}

func runContact(args []string) int {
	if len(args) == 0 {
		printContactHelp()
		return 0
	}
	switch args[0] {
	case "list-ids":
		fs := flag.NewFlagSet("contact list-ids", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		cursor := fs.String("cursor", "", "pagination cursor")
		limit := fs.Int("limit", 100, "page size")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		mergeNonEmpty(payload, "cursor", *cursor)
		if _, exists := payload["limit"]; !exists {
			payload["limit"] = *limit
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/user/list_id", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0
	default:
		printError(fmt.Errorf("unknown contact subcommand: %s", args[0]))
		printContactHelp()
		return 1
	}
}

func runCalendar(args []string) int {
	if len(args) == 0 {
		printCalendarHelp()
		return 0
	}

	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("calendar create", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		admins := fs.String("admins", "", "comma separated admin user ids")
		summary := fs.String("summary", "", "calendar summary")
		color := fs.String("color", "", "calendar color such as #FF3030")
		description := fs.String("description", "", "calendar description")
		shares := fs.String("shares", "", "comma separated shared user ids")
		sharePermission := fs.Int("share-permission", 1, "share permission for --shares users")
		setAsDefault := fs.Bool("set-as-default", false, "set this calendar as default")
		isPublic := fs.Bool("public", false, "make calendar visible to a public range")
		publicUserIDs := fs.String("public-userids", "", "comma separated public range user ids")
		publicPartyIDs := fs.String("public-partyids", "", "comma separated public range party ids")
		agentID := fs.Int("agentid", defaultAgentID(0), "optional agent id")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		calendar, err := ensureNestedMap(payload, "calendar")
		if err != nil {
			printError(err)
			return 1
		}
		mergeStringSlice(calendar, "admins", *admins)
		mergeNonEmpty(calendar, "summary", *summary)
		mergeNonEmpty(calendar, "color", *color)
		mergeNonEmpty(calendar, "description", *description)
		if shareList := buildShares(*shares, *sharePermission); len(shareList) > 0 {
			calendar["shares"] = shareList
		}
		if *setAsDefault {
			calendar["set_as_default"] = 1
		}
		if *isPublic {
			calendar["is_public"] = 1
		}
		publicRange, err := buildPublicRange(*publicUserIDs, *publicPartyIDs)
		if err != nil {
			printError(err)
			return 1
		}
		if len(publicRange) > 0 {
			calendar["public_range"] = publicRange
		}
		if _, exists := payload["agentid"]; !exists && *agentID > 0 {
			payload["agentid"] = *agentID
		}
		if len(calendar) == 0 {
			printError(errors.New("calendar create requires calendar fields"))
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/oa/calendar/add", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "update":
		fs := flag.NewFlagSet("calendar update", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		calID := fs.String("cal-id", "", "calendar id")
		admins := fs.String("admins", "", "comma separated admin user ids")
		summary := fs.String("summary", "", "calendar summary")
		color := fs.String("color", "", "calendar color such as #FF3030")
		description := fs.String("description", "", "calendar description")
		shares := fs.String("shares", "", "comma separated shared user ids")
		sharePermission := fs.Int("share-permission", 1, "share permission for --shares users")
		setAsDefault := fs.Bool("set-as-default", false, "set this calendar as default")
		isPublic := fs.Bool("public", false, "make calendar visible to a public range")
		publicUserIDs := fs.String("public-userids", "", "comma separated public range user ids")
		publicPartyIDs := fs.String("public-partyids", "", "comma separated public range party ids")
		skipPublicRange := fs.Bool("skip-public-range", false, "skip updating public range")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		calendar, err := ensureNestedMap(payload, "calendar")
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := calendar["cal_id"]; !exists {
			if strings.TrimSpace(*calID) == "" {
				printError(errors.New("calendar update requires --cal-id"))
				return 1
			}
			calendar["cal_id"] = strings.TrimSpace(*calID)
		}
		mergeStringSlice(calendar, "admins", *admins)
		mergeNonEmpty(calendar, "summary", *summary)
		mergeNonEmpty(calendar, "color", *color)
		mergeNonEmpty(calendar, "description", *description)
		if shareList := buildShares(*shares, *sharePermission); len(shareList) > 0 {
			calendar["shares"] = shareList
		}
		if *setAsDefault {
			calendar["set_as_default"] = 1
		}
		if *isPublic {
			calendar["is_public"] = 1
		}
		publicRange, err := buildPublicRange(*publicUserIDs, *publicPartyIDs)
		if err != nil {
			printError(err)
			return 1
		}
		if len(publicRange) > 0 {
			calendar["public_range"] = publicRange
		}
		if *skipPublicRange {
			payload["skip_public_range"] = 1
		}
		if len(calendar) == 1 && len(payload) == 1 {
			printError(errors.New("calendar update requires at least one field to update"))
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/oa/calendar/update", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "get":
		fs := flag.NewFlagSet("calendar get", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		calID := fs.String("cal-id", "", "calendar id")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		mergeNonEmpty(payload, "cal_id", *calID)
		if _, exists := payload["cal_id"]; !exists {
			printError(errors.New("calendar get requires --cal-id"))
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/oa/calendar/get", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "delete":
		fs := flag.NewFlagSet("calendar delete", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		calID := fs.String("cal-id", "", "calendar id")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		mergeNonEmpty(payload, "cal_id", *calID)
		if _, exists := payload["cal_id"]; !exists {
			printError(errors.New("calendar delete requires --cal-id"))
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/oa/calendar/del", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	default:
		printError(fmt.Errorf("unknown calendar subcommand: %s", args[0]))
		printCalendarHelp()
		return 1
	}
}

func runSchedule(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printScheduleHelp()
		return 0
	}

	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("schedule create", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		admins := fs.String("admins", "", "comma separated admin user ids")
		attendees := fs.String("attendees", "", "comma separated attendee user ids")
		start := fs.String("start", "", "schedule start time, ISO datetime or epoch")
		end := fs.String("end", "", "schedule end time, ISO datetime or epoch")
		summary := fs.String("summary", "", "schedule summary")
		description := fs.String("description", "", "schedule description")
		location := fs.String("location", "", "schedule location")
		calID := fs.String("cal-id", "", "calendar id")
		isWholeDay := fs.Bool("whole-day", false, "mark schedule as whole-day")
		remindBefore := fs.Int("remind-before", -1, "seconds before event to remind")
		agentID := fs.Int("agentid", defaultAgentID(0), "optional agent id")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		schedule, err := ensureNestedMap(payload, "schedule")
		if err != nil {
			printError(err)
			return 1
		}
		mergeStringSlice(schedule, "admins", *admins)
		mergeAttendees(schedule, *attendees)
		if strings.TrimSpace(*start) != "" {
			value, err := wecom.ParseTimeToEpoch(*start)
			if err != nil {
				printError(err)
				return 1
			}
			schedule["start_time"] = value
		}
		if strings.TrimSpace(*end) != "" {
			value, err := wecom.ParseTimeToEpoch(*end)
			if err != nil {
				printError(err)
				return 1
			}
			schedule["end_time"] = value
		}
		mergeNonEmpty(schedule, "summary", *summary)
		mergeNonEmpty(schedule, "description", *description)
		mergeNonEmpty(schedule, "location", *location)
		mergeNonEmpty(schedule, "cal_id", *calID)
		if *isWholeDay {
			schedule["is_whole_day"] = 1
		}
		if *remindBefore >= 0 {
			schedule["reminders"] = map[string]any{
				"is_remind":                1,
				"remind_before_event_secs": *remindBefore,
			}
		}
		if _, exists := payload["agentid"]; !exists && *agentID > 0 {
			payload["agentid"] = *agentID
		}
		if len(schedule) == 0 {
			printError(errors.New("schedule create requires schedule fields"))
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/oa/schedule/add", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "update":
		fs := flag.NewFlagSet("schedule update", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		scheduleID := fs.String("schedule-id", "", "schedule id")
		opMode := fs.Int("op-mode", -1, "repeat schedule operation mode")
		opStartTime := fs.String("op-start-time", "", "repeat schedule operation instance start time")
		admins := fs.String("admins", "", "comma separated admin user ids")
		attendees := fs.String("attendees", "", "comma separated attendee user ids")
		start := fs.String("start", "", "schedule start time, ISO datetime or epoch")
		end := fs.String("end", "", "schedule end time, ISO datetime or epoch")
		summary := fs.String("summary", "", "schedule summary")
		description := fs.String("description", "", "schedule description")
		location := fs.String("location", "", "schedule location")
		calID := fs.String("cal-id", "", "calendar id")
		isWholeDay := fs.Bool("whole-day", false, "mark schedule as whole-day")
		remindBefore := fs.Int("remind-before", -1, "seconds before event to remind")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		schedule, err := ensureNestedMap(payload, "schedule")
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := schedule["schedule_id"]; !exists {
			if strings.TrimSpace(*scheduleID) == "" {
				printError(errors.New("schedule update requires --schedule-id"))
				return 1
			}
			schedule["schedule_id"] = strings.TrimSpace(*scheduleID)
		}
		if *opMode >= 0 {
			payload["op_mode"] = *opMode
		}
		if strings.TrimSpace(*opStartTime) != "" {
			value, err := wecom.ParseTimeToEpoch(*opStartTime)
			if err != nil {
				printError(err)
				return 1
			}
			payload["op_start_time"] = value
		}
		mergeStringSlice(schedule, "admins", *admins)
		mergeAttendees(schedule, *attendees)
		if strings.TrimSpace(*start) != "" {
			value, err := wecom.ParseTimeToEpoch(*start)
			if err != nil {
				printError(err)
				return 1
			}
			schedule["start_time"] = value
		}
		if strings.TrimSpace(*end) != "" {
			value, err := wecom.ParseTimeToEpoch(*end)
			if err != nil {
				printError(err)
				return 1
			}
			schedule["end_time"] = value
		}
		mergeNonEmpty(schedule, "summary", *summary)
		mergeNonEmpty(schedule, "description", *description)
		mergeNonEmpty(schedule, "location", *location)
		mergeNonEmpty(schedule, "cal_id", *calID)
		if *isWholeDay {
			schedule["is_whole_day"] = 1
		}
		if *remindBefore >= 0 {
			schedule["reminders"] = map[string]any{
				"is_remind":                1,
				"remind_before_event_secs": *remindBefore,
			}
		}
		if len(schedule) == 1 && len(payload) == 1 {
			printError(errors.New("schedule update requires at least one field to update"))
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/oa/schedule/update", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "get":
		fs := flag.NewFlagSet("schedule get", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		scheduleIDs := fs.String("schedule-ids", "", "comma separated schedule ids")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := payload["schedule_id_list"]; !exists {
			idList := parseCSV(*scheduleIDs)
			if len(idList) == 0 {
				printError(errors.New("schedule get requires --schedule-ids"))
				return 1
			}
			payload["schedule_id_list"] = idList
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/oa/schedule/get", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "list":
		fs := flag.NewFlagSet("schedule list", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		calID := fs.String("cal-id", "", "calendar id")
		offset := fs.Int("offset", 0, "pagination offset")
		limit := fs.Int("limit", 100, "page size")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		mergeNonEmpty(payload, "cal_id", *calID)
		if _, exists := payload["cal_id"]; !exists {
			printError(errors.New("schedule list requires --cal-id"))
			return 1
		}
		if _, exists := payload["offset"]; !exists {
			payload["offset"] = *offset
		}
		if _, exists := payload["limit"]; !exists {
			payload["limit"] = *limit
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/oa/schedule/get_by_calendar", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "delete":
		fs := flag.NewFlagSet("schedule delete", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		scheduleID := fs.String("schedule-id", "", "schedule id")
		opMode := fs.Int("op-mode", -1, "repeat schedule operation mode")
		opStartTime := fs.String("op-start-time", "", "repeat schedule operation instance start time")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		mergeNonEmpty(payload, "schedule_id", *scheduleID)
		if _, exists := payload["schedule_id"]; !exists {
			printError(errors.New("schedule delete requires --schedule-id"))
			return 1
		}
		if _, exists := payload["op_mode"]; !exists && *opMode >= 0 {
			payload["op_mode"] = *opMode
		}
		if _, exists := payload["op_start_time"]; !exists && strings.TrimSpace(*opStartTime) != "" {
			value, err := wecom.ParseTimeToEpoch(*opStartTime)
			if err != nil {
				printError(err)
				return 1
			}
			payload["op_start_time"] = value
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/oa/schedule/del", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "add-attendees":
		return runScheduleAttendees(args[1:], "/cgi-bin/oa/schedule/add_attendees", "schedule add-attendees")
	case "remove-attendees":
		return runScheduleAttendees(args[1:], "/cgi-bin/oa/schedule/del_attendees", "schedule remove-attendees")

	default:
		printError(fmt.Errorf("unknown schedule subcommand: %s", args[0]))
		printScheduleHelp()
		return 1
	}
}

func runScheduleAttendees(args []string, endpoint string, commandName string) int {
	fs := flag.NewFlagSet(commandName, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	debug := fs.Bool("debug", false, "print request debug info")
	scheduleID := fs.String("schedule-id", "", "schedule id")
	attendees := fs.String("attendees", "", "comma separated attendee user ids")
	payloadFile := fs.String("payload-file", "", "payload json file")
	if err := fs.Parse(args); err != nil {
		printError(err)
		return 1
	}

	payload, err := loadJSONInput(fs.Args(), *payloadFile)
	if err != nil {
		printError(err)
		return 1
	}
	mergeNonEmpty(payload, "schedule_id", *scheduleID)
	if _, exists := payload["schedule_id"]; !exists {
		printError(errors.New(commandName + " requires --schedule-id"))
		return 1
	}
	if _, exists := payload["attendees"]; !exists {
		if attendeeList := buildUserObjects(*attendees); len(attendeeList) > 0 {
			payload["attendees"] = attendeeList
		}
	}
	if _, exists := payload["attendees"]; !exists {
		printError(errors.New(commandName + " requires --attendees"))
		return 1
	}

	client, err := newClient(*debug)
	if err != nil {
		printError(err)
		return 1
	}
	result, err := client.Post(endpoint, payload)
	if err != nil {
		printError(err)
		return 1
	}
	printJSON(result)
	return 0
}

func runMeeting(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printMeetingHelp()
		return 0
	}

	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("meeting create", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		adminUserID := fs.String("admin-userid", "", "meeting admin userid")
		title := ""
		fs.StringVar(&title, "title", "", "meeting title")
		fs.StringVar(&title, "subject", "", "alias for --title")
		start := fs.String("start", "", "meeting start time, ISO datetime or epoch")
		duration := fs.Int("duration", -1, "meeting duration in seconds")
		durationMinutes := fs.Int("duration-minutes", -1, "meeting duration in minutes")
		description := fs.String("description", "", "meeting description")
		location := fs.String("location", "", "meeting location")
		noLocation := fs.Bool("no-location", false, "create without a meeting location after user confirmation")
		agentID := fs.Int("agentid", defaultAgentID(0), "optional agent id")
		invitees := fs.String("invitees", "", "comma separated invitee user ids")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}

		if strings.TrimSpace(*start) == "" {
			printError(errors.New("meeting create requires --start; ask the user or use a user-confirmed datetime component value"))
			return 1
		}
		meetingStart, err := wecom.ParseTimeToEpoch(*start)
		if err != nil {
			printError(err)
			return 1
		}
		meetingDuration := normalizeMeetingDuration(*duration, *durationMinutes)

		inviteeList, err := parseInvitees(*invitees)
		if err != nil {
			printError(err)
			return 1
		}
		admin := strings.TrimSpace(*adminUserID)
		if admin == "" && len(inviteeList) == 1 {
			admin = inviteeList[0]
		}
		if admin == "" {
			printError(errors.New("missing admin userid; pass --admin-userid or provide exactly one invitee"))
			return 1
		}
		if len(inviteeList) == 0 {
			inviteeList = []string{admin}
		}

		mergeNonEmpty(payload, "admin_userid", admin)
		if _, exists := payload["title"]; !exists {
			payload["title"] = defaultString(title, "Scheduled Meeting")
		}
		if _, exists := payload["meeting_start"]; !exists {
			payload["meeting_start"] = meetingStart
		}
		if _, exists := payload["meeting_duration"]; !exists && meetingDuration < 0 {
			printError(errors.New("meeting create requires --duration; ask the user or use a user-confirmed duration component value"))
			return 1
		}
		if _, exists := payload["meeting_duration"]; !exists {
			payload["meeting_duration"] = meetingDuration
		}
		if _, exists := payload["description"]; !exists && strings.TrimSpace(*description) != "" {
			payload["description"] = *description
		}
		if rawLocation, exists := payload["location"]; exists {
			locationValue, ok := rawLocation.(string)
			if !ok {
				printError(errors.New("meeting location must be a string"))
				return 1
			}
			if strings.TrimSpace(locationValue) == "" && !*noLocation {
				printError(errors.New("meeting create requires --location or --no-location after user confirmation"))
				return 1
			}
		} else if strings.TrimSpace(*location) != "" {
			payload["location"] = strings.TrimSpace(*location)
		} else if !*noLocation {
			printError(errors.New("meeting create requires --location or --no-location after user confirmation"))
			return 1
		}
		if _, exists := payload["agentid"]; !exists && *agentID > 0 {
			payload["agentid"] = *agentID
		}
		if _, exists := payload["invitees"]; !exists {
			payload["invitees"] = map[string]any{"userid": inviteeList}
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/meeting/create", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "update":
		fs := flag.NewFlagSet("meeting update", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		meetingID := ""
		fs.StringVar(&meetingID, "meeting-id", "", "meeting id")
		fs.StringVar(&meetingID, "meetingid", "", "alias for --meeting-id")
		adminUserID := fs.String("admin-userid", "", "meeting admin userid")
		title := ""
		fs.StringVar(&title, "title", "", "meeting title")
		fs.StringVar(&title, "subject", "", "alias for --title")
		start := fs.String("start", "", "meeting start time, ISO datetime or epoch")
		duration := fs.Int("duration", -1, "meeting duration in seconds")
		description := fs.String("description", "", "meeting description")
		location := fs.String("location", "", "meeting location")
		agentID := fs.Int("agentid", -1, "optional agent id")
		invitees := fs.String("invitees", "", "comma separated invitee user ids")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := payload["meetingid"]; !exists {
			if strings.TrimSpace(meetingID) == "" {
				printError(errors.New("meeting update requires --meeting-id"))
				return 1
			}
			payload["meetingid"] = strings.TrimSpace(meetingID)
		}

		mergeNonEmpty(payload, "admin_userid", *adminUserID)
		mergeNonEmpty(payload, "title", title)
		if strings.TrimSpace(*start) != "" {
			meetingStart, err := wecom.ParseTimeToEpoch(*start)
			if err != nil {
				printError(err)
				return 1
			}
			payload["meeting_start"] = meetingStart
		}
		if *duration >= 0 {
			payload["meeting_duration"] = *duration
		}
		if strings.TrimSpace(*description) != "" {
			payload["description"] = strings.TrimSpace(*description)
		}
		if strings.TrimSpace(*location) != "" {
			payload["location"] = strings.TrimSpace(*location)
		}
		if *agentID > 0 {
			payload["agentid"] = *agentID
		}
		inviteeList, err := parseInvitees(*invitees)
		if err != nil {
			printError(err)
			return 1
		}
		if len(inviteeList) > 0 {
			payload["invitees"] = map[string]any{"userid": inviteeList}
		}
		if len(payload) == 1 {
			printError(errors.New("meeting update requires at least one field to update"))
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/meeting/update", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "list":
		fs := flag.NewFlagSet("meeting list", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		userID := ""
		fs.StringVar(&userID, "userid", "", "meeting owner userid")
		fs.StringVar(&userID, "admin-userid", "", "alias for --userid")
		start := fs.String("start", "", "range start time")
		end := fs.String("end", "", "range end time")
		_ = fs.Int("status", -1, "accepted for agent compatibility; ignored by WeCom meeting list")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := payload["userid"]; !exists {
			if strings.TrimSpace(userID) == "" {
				printError(errors.New("meeting list requires --userid"))
				return 1
			}
			payload["userid"] = strings.TrimSpace(userID)
		}
		if _, exists := payload["begin_time"]; !exists {
			begin, err := wecom.ParseTimeToEpoch(*start)
			if err != nil {
				printError(errors.New("meeting list requires --start"))
				return 1
			}
			payload["begin_time"] = begin
		}
		if _, exists := payload["end_time"]; !exists {
			finish, err := wecom.ParseTimeToEpoch(*end)
			if err != nil {
				printError(errors.New("meeting list requires --end"))
				return 1
			}
			payload["end_time"] = finish
		}
		if err := validateMeetingListWindow(payload); err != nil {
			printError(err)
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/meeting/get_user_meetingid", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "get":
		fs := flag.NewFlagSet("meeting get", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		meetingID := ""
		fs.StringVar(&meetingID, "meeting-id", "", "meeting id")
		fs.StringVar(&meetingID, "meetingid", "", "alias for --meeting-id")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := payload["meetingid"]; !exists {
			if strings.TrimSpace(meetingID) == "" {
				printError(errors.New("meeting get requires --meeting-id"))
				return 1
			}
			payload["meetingid"] = strings.TrimSpace(meetingID)
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/meeting/get_info", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "cancel":
		fs := flag.NewFlagSet("meeting cancel", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		meetingID := ""
		fs.StringVar(&meetingID, "meeting-id", "", "meeting id")
		fs.StringVar(&meetingID, "meetingid", "", "alias for --meeting-id")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := payload["meetingid"]; !exists {
			if strings.TrimSpace(meetingID) == "" {
				printError(errors.New("meeting cancel requires --meeting-id"))
				return 1
			}
			payload["meetingid"] = strings.TrimSpace(meetingID)
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/meeting/cancel", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	default:
		printError(fmt.Errorf("unknown meeting subcommand: %s", args[0]))
		printMeetingHelp()
		return 1
	}
}

func runTodo(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printTodoHelp()
		return 0
	}

	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("todo create", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		content := fs.String("content", "", "todo content")
		remindTime := fs.String("remind-time", "", "todo remind time, ISO datetime or epoch")
		status := fs.Int("status", -1, "todo status")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		mergeNonEmpty(payload, "content", *content)
		if strings.TrimSpace(*remindTime) != "" {
			value, err := wecom.ParseTimeToEpoch(*remindTime)
			if err != nil {
				printError(err)
				return 1
			}
			payload["remind_time"] = value
		}
		if *status >= 0 {
			payload["todo_status"] = *status
		}
		if len(payload) == 0 {
			printError(errors.New("todo create requires content, JSON payload, or other fields"))
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/todo/add", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "list":
		fs := flag.NewFlagSet("todo list", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		start := fs.String("start", "", "range start time")
		end := fs.String("end", "", "range end time")
		cursor := fs.String("cursor", "", "pagination cursor")
		limit := fs.Int("limit", 100, "page size")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if strings.TrimSpace(*start) != "" {
			value, err := wecom.ParseTimeToEpoch(*start)
			if err != nil {
				printError(err)
				return 1
			}
			payload["start_time"] = value
		}
		if strings.TrimSpace(*end) != "" {
			value, err := wecom.ParseTimeToEpoch(*end)
			if err != nil {
				printError(err)
				return 1
			}
			payload["end_time"] = value
		}
		mergeNonEmpty(payload, "cursor", *cursor)
		if _, exists := payload["limit"]; !exists {
			payload["limit"] = *limit
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/todo/get_todo_list", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "get":
		fs := flag.NewFlagSet("todo get", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		todoIDs := fs.String("todo-ids", "", "comma separated todo ids")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := payload["todo_id_list"]; !exists {
			idList := parseCSV(*todoIDs)
			if len(idList) == 0 {
				printError(errors.New("todo get requires --todo-ids"))
				return 1
			}
			payload["todo_id_list"] = idList
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/todo/get_todo_detail", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "update":
		fs := flag.NewFlagSet("todo update", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		todoID := fs.String("todo-id", "", "todo id")
		content := fs.String("content", "", "todo content")
		remindTime := fs.String("remind-time", "", "todo remind time, ISO datetime or epoch")
		status := fs.Int("status", -1, "todo status")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := payload["todo_id"]; !exists {
			if strings.TrimSpace(*todoID) == "" {
				printError(errors.New("todo update requires --todo-id"))
				return 1
			}
			payload["todo_id"] = strings.TrimSpace(*todoID)
		}
		mergeNonEmpty(payload, "content", *content)
		if strings.TrimSpace(*remindTime) != "" {
			value, err := wecom.ParseTimeToEpoch(*remindTime)
			if err != nil {
				printError(err)
				return 1
			}
			payload["remind_time"] = value
		}
		if *status >= 0 {
			payload["todo_status"] = *status
		}
		if len(payload) == 1 {
			printError(errors.New("todo update requires at least one field to update"))
			return 1
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/todo/update", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "delete":
		fs := flag.NewFlagSet("todo delete", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		todoID := fs.String("todo-id", "", "todo id")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := payload["todo_id"]; !exists {
			if strings.TrimSpace(*todoID) == "" {
				printError(errors.New("todo delete requires --todo-id"))
				return 1
			}
			payload["todo_id"] = strings.TrimSpace(*todoID)
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/todo/delete", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	case "change-status":
		fs := flag.NewFlagSet("todo change-status", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		debug := fs.Bool("debug", false, "print request debug info")
		todoID := fs.String("todo-id", "", "todo id")
		status := fs.Int("status", -1, "todo status")
		payloadFile := fs.String("payload-file", "", "payload json file")
		if err := fs.Parse(args[1:]); err != nil {
			printError(err)
			return 1
		}

		payload, err := loadJSONInput(fs.Args(), *payloadFile)
		if err != nil {
			printError(err)
			return 1
		}
		if _, exists := payload["todo_id"]; !exists {
			if strings.TrimSpace(*todoID) == "" {
				printError(errors.New("todo change-status requires --todo-id"))
				return 1
			}
			payload["todo_id"] = strings.TrimSpace(*todoID)
		}
		if _, exists := payload["todo_status"]; !exists {
			if *status < 0 {
				printError(errors.New("todo change-status requires --status"))
				return 1
			}
			payload["todo_status"] = *status
		}

		client, err := newClient(*debug)
		if err != nil {
			printError(err)
			return 1
		}
		result, err := client.Post("/cgi-bin/todo/change_todo_user_status", payload)
		if err != nil {
			printError(err)
			return 1
		}
		printJSON(result)
		return 0

	default:
		printError(fmt.Errorf("unknown todo subcommand: %s", args[0]))
		printTodoHelp()
		return 1
	}
}

func runWeDrive(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDriveHelp()
		return 0
	}

	switch args[0] {
	case "space":
		return runWeDriveSpace(args[1:])
	case "file":
		return runWeDriveFile(args[1:])
	default:
		printError(fmt.Errorf("unknown wedrive subcommand: %s", args[0]))
		printWeDriveHelp()
		return 1
	}
}

type payloadFinalizer func(map[string]any) error

func runWeDoc(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDocHelp()
		return 0
	}

	switch args[0] {
	case "doc":
		return runWeDocDoc(args[1:])
	case "content":
		return runWeDocContent(args[1:])
	case "form":
		return runWeDocForm(args[1:])
	case "smartsheet":
		return runWeDocSmartSheet(args[1:])
	default:
		printError(fmt.Errorf("unknown wedoc subcommand: %s", args[0]))
		printWeDocHelp()
		return 1
	}
}

func runWeDocDoc(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDocHelp()
		return 0
	}

	switch args[0] {
	case "create":
		return runWeDocPost("wedoc doc create", "/cgi-bin/wedoc/create_doc", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			spaceID := fs.String("spaceid", "", "WeDrive space id")
			fatherID := fs.String("fatherid", "", "parent folder id")
			docType := fs.Int("doc-type", -1, "doc type")
			docName := fs.String("doc-name", "", "doc name")
			adminUsers := fs.String("admin-users", "", "comma separated admin user ids")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "spaceid", *spaceID)
				mergeNonEmpty(payload, "fatherid", *fatherID)
				mergeIntIfSet(payload, "doc_type", *docType)
				mergeNonEmpty(payload, "doc_name", *docName)
				mergeStringSlice(payload, "admin_users", *adminUsers)
				if err := requirePayloadKey(payload, "spaceid", "wedoc doc create requires --spaceid"); err != nil {
					return err
				}
				if err := requirePayloadKey(payload, "doc_type", "wedoc doc create requires --doc-type"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "doc_name", "wedoc doc create requires --doc-name")
			}
		})
	case "rename":
		return runWeDocPost("wedoc doc rename", "/cgi-bin/wedoc/rename_doc", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			formID := fs.String("formid", "", "collect form id")
			newName := fs.String("new-name", "", "new name")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				mergeNonEmpty(payload, "formid", *formID)
				mergeNonEmpty(payload, "new_name", *newName)
				if err := requireExactlyOnePayloadKey(payload, "docid", "formid", "wedoc doc rename requires exactly one of --docid or --formid"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "new_name", "wedoc doc rename requires --new-name")
			}
		})
	case "delete":
		return runWeDocPost("wedoc doc delete", "/cgi-bin/wedoc/del_doc", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			formID := fs.String("formid", "", "collect form id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				mergeNonEmpty(payload, "formid", *formID)
				return requireExactlyOnePayloadKey(payload, "docid", "formid", "wedoc doc delete requires exactly one of --docid or --formid")
			}
		})
	case "info":
		return runWeDocPost("wedoc doc info", "/cgi-bin/wedoc/get_doc_base_info", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				return requirePayloadKey(payload, "docid", "wedoc doc info requires --docid")
			}
		})
	case "share":
		return runWeDocPost("wedoc doc share", "/cgi-bin/wedoc/doc_share", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			formID := fs.String("formid", "", "collect form id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				mergeNonEmpty(payload, "formid", *formID)
				return requireExactlyOnePayloadKey(payload, "docid", "formid", "wedoc doc share requires exactly one of --docid or --formid")
			}
		})
	case "auth":
		if len(args) < 2 || wantsHelp(args[1]) {
			printWeDocHelp()
			return 0
		}
		switch args[1] {
		case "get":
			return runWeDocPost("wedoc doc auth get", "/cgi-bin/wedoc/doc_get_auth", args[2:], func(fs *flag.FlagSet) payloadFinalizer {
				docID := fs.String("docid", "", "doc id")
				return func(payload map[string]any) error {
					mergeNonEmpty(payload, "docid", *docID)
					return requirePayloadKey(payload, "docid", "wedoc doc auth get requires --docid")
				}
			})
		case "member":
			return runWeDocPost("wedoc doc auth member", "/cgi-bin/wedoc/mod_doc_member", args[2:], func(fs *flag.FlagSet) payloadFinalizer {
				docID := fs.String("docid", "", "doc id")
				updateMembers := fs.String("update-members-json", "", "update_file_member_list JSON array")
				deleteMembers := fs.String("delete-members-json", "", "del_file_member_list JSON array")
				return func(payload map[string]any) error {
					mergeNonEmpty(payload, "docid", *docID)
					if err := mergeJSONArray(payload, "update_file_member_list", *updateMembers); err != nil {
						return err
					}
					if err := mergeJSONArray(payload, "del_file_member_list", *deleteMembers); err != nil {
						return err
					}
					if err := requirePayloadKey(payload, "docid", "wedoc doc auth member requires --docid"); err != nil {
						return err
					}
					if !hasPayloadKey(payload, "update_file_member_list") && !hasPayloadKey(payload, "del_file_member_list") {
						return errors.New("wedoc doc auth member requires --update-members-json or --delete-members-json")
					}
					return nil
				}
			})
		case "join-rule":
			return runWeDocPost("wedoc doc auth join-rule", "/cgi-bin/wedoc/mod_doc_join_rule", args[2:], func(fs *flag.FlagSet) payloadFinalizer {
				docID := fs.String("docid", "", "doc id")
				return func(payload map[string]any) error {
					mergeNonEmpty(payload, "docid", *docID)
					return requirePayloadKey(payload, "docid", "wedoc doc auth join-rule requires --docid and rule fields or JSON payload")
				}
			})
		case "security":
			return runWeDocPost("wedoc doc auth security", "/cgi-bin/wedoc/mod_doc_safty_setting", args[2:], func(fs *flag.FlagSet) payloadFinalizer {
				docID := fs.String("docid", "", "doc id")
				return func(payload map[string]any) error {
					mergeNonEmpty(payload, "docid", *docID)
					return requirePayloadKey(payload, "docid", "wedoc doc auth security requires --docid and security fields or JSON payload")
				}
			})
		default:
			printError(fmt.Errorf("unknown wedoc doc auth subcommand: %s", args[1]))
			printWeDocHelp()
			return 1
		}
	default:
		printError(fmt.Errorf("unknown wedoc doc subcommand: %s", args[0]))
		printWeDocHelp()
		return 1
	}
}

func runWeDocContent(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDocHelp()
		return 0
	}

	switch args[0] {
	case "doc":
		return runWeDocContentDoc(args[1:])
	case "sheet":
		return runWeDocContentSheet(args[1:])
	default:
		printError(fmt.Errorf("unknown wedoc content subcommand: %s", args[0]))
		printWeDocHelp()
		return 1
	}
}

func runWeDocContentDoc(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDocHelp()
		return 0
	}

	switch args[0] {
	case "data":
		return runWeDocPost("wedoc content doc data", "/cgi-bin/wedoc/document/get", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				return requirePayloadKey(payload, "docid", "wedoc content doc data requires --docid")
			}
		})
	case "modify":
		return runWeDocPost("wedoc content doc modify", "/cgi-bin/wedoc/document/batch_update", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			version := fs.Uint("version", 0, "document version (required for batch_update)")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				if *version > 0 {
					payload["version"] = *version
				}
				return requirePayloadKey(payload, "docid", "wedoc content doc modify requires --docid and official batch_update JSON payload")
			}
		})
	default:
		printError(fmt.Errorf("unknown wedoc content doc subcommand: %s", args[0]))
		printWeDocHelp()
		return 1
	}
}

func runWeDocContentSheet(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDocHelp()
		return 0
	}

	switch args[0] {
	case "rowcol":
		return runWeDocSheetContentPost("wedoc content sheet rowcol", "/cgi-bin/wedoc/get_sheet_row_col_info", args[1:], nil)
	case "data":
		return runWeDocSheetContentPost("wedoc content sheet data", "/cgi-bin/wedoc/get_sheet_data", args[1:], nil)
	case "modify":
		return runWeDocSheetContentPost("wedoc content sheet modify", "/cgi-bin/wedoc/mod_sheet", args[1:], nil)
	default:
		printError(fmt.Errorf("unknown wedoc content sheet subcommand: %s", args[0]))
		printWeDocHelp()
		return 1
	}
}

func runWeDocSheetContentPost(commandName string, endpoint string, args []string, configure func(*flag.FlagSet) payloadFinalizer) int {
	return runWeDocPost(commandName, endpoint, args, func(fs *flag.FlagSet) payloadFinalizer {
		docID := fs.String("docid", "", "doc id")
		sheetID := fs.String("sheet-id", "", "sheet id")
		var finalize payloadFinalizer
		if configure != nil {
			finalize = configure(fs)
		}
		return func(payload map[string]any) error {
			mergeNonEmpty(payload, "docid", *docID)
			mergeNonEmpty(payload, "sheet_id", *sheetID)
			if finalize != nil {
				if err := finalize(payload); err != nil {
					return err
				}
			}
			if err := requirePayloadKey(payload, "docid", commandName+" requires --docid"); err != nil {
				return err
			}
			return requirePayloadKey(payload, "sheet_id", commandName+" requires --sheet-id")
		}
	})
}

func runWeDocForm(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDocHelp()
		return 0
	}

	switch args[0] {
	case "create":
		return runWeDocPost("wedoc form create", "/cgi-bin/wedoc/create_form", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			spaceID := fs.String("spaceid", "", "WeDrive space id")
			fatherID := fs.String("fatherid", "", "parent folder id")
			formInfoJSON := fs.String("form-info-json", "", "form_info JSON object")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "spaceid", *spaceID)
				mergeNonEmpty(payload, "fatherid", *fatherID)
				if err := mergeJSONObject(payload, "form_info", *formInfoJSON); err != nil {
					return err
				}
				if err := requirePayloadKey(payload, "spaceid", "wedoc form create requires --spaceid"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "form_info", "wedoc form create requires --form-info-json or form_info in JSON payload")
			}
		})
	case "modify":
		return runWeDocPost("wedoc form modify", "/cgi-bin/wedoc/modify_form", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			formID := fs.String("formid", "", "form id")
			formInfoJSON := fs.String("form-info-json", "", "form_info JSON object")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "formid", *formID)
				if err := mergeJSONObject(payload, "form_info", *formInfoJSON); err != nil {
					return err
				}
				if err := requirePayloadKey(payload, "formid", "wedoc form modify requires --formid"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "form_info", "wedoc form modify requires --form-info-json or form_info in JSON payload")
			}
		})
	case "info":
		return runWeDocPost("wedoc form info", "/cgi-bin/wedoc/get_form_info", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			formID := fs.String("formid", "", "form id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "formid", *formID)
				return requirePayloadKey(payload, "formid", "wedoc form info requires --formid")
			}
		})
	case "statistic":
		return runWeDocPost("wedoc form statistic", "/cgi-bin/wedoc/get_form_statistic", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			repeatedID := fs.String("repeated-id", "", "repeated id")
			reqType := fs.Int("req-type", -1, "request type")
			startTime := fs.Int("start-time", -1, "start time epoch")
			endTime := fs.Int("end-time", -1, "end time epoch")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "repeated_id", *repeatedID)
				mergeIntIfSet(payload, "req_type", *reqType)
				mergeIntIfSet(payload, "start_time", *startTime)
				mergeIntIfSet(payload, "end_time", *endTime)
				if err := requirePayloadKey(payload, "repeated_id", "wedoc form statistic requires --repeated-id"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "req_type", "wedoc form statistic requires --req-type")
			}
		})
	case "answer":
		return runWeDocPost("wedoc form answer", "/cgi-bin/wedoc/get_form_answer", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			repeatedID := fs.String("repeated-id", "", "repeated id")
			answerIDs := fs.String("answer-ids", "", "comma separated answer ids")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "repeated_id", *repeatedID)
				if err := mergeIntSlice(payload, "answer_ids", *answerIDs); err != nil {
					return err
				}
				if err := requirePayloadKey(payload, "repeated_id", "wedoc form answer requires --repeated-id"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "answer_ids", "wedoc form answer requires --answer-ids")
			}
		})
	default:
		printError(fmt.Errorf("unknown wedoc form subcommand: %s", args[0]))
		printWeDocHelp()
		return 1
	}
}

func runWeDocSmartSheet(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDocHelp()
		return 0
	}

	switch args[0] {
	case "sheet":
		return runWeDocSmartSheetSheet(args[1:])
	case "record":
		return runWeDocSmartSheetRecord(args[1:])
	default:
		printError(fmt.Errorf("unknown wedoc smartsheet subcommand: %s", args[0]))
		printWeDocHelp()
		return 1
	}
}

func runWeDocSmartSheetSheet(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDocHelp()
		return 0
	}

	switch args[0] {
	case "list":
		return runWeDocPost("wedoc smartsheet sheet list", "/cgi-bin/wedoc/smartsheet/get_sheet", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				return requirePayloadKey(payload, "docid", "wedoc smartsheet sheet list requires --docid")
			}
		})
	case "add":
		return runWeDocPost("wedoc smartsheet sheet add", "/cgi-bin/wedoc/smartsheet/add_sheet", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			title := fs.String("title", "", "sheet title")
			index := fs.Int("index", -1, "sheet index")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				properties, err := ensureNestedMap(payload, "properties")
				if err != nil {
					return err
				}
				mergeNonEmpty(properties, "title", *title)
				mergeIntIfSet(properties, "index", *index)
				if err := requirePayloadKey(payload, "docid", "wedoc smartsheet sheet add requires --docid"); err != nil {
					return err
				}
				return requirePayloadKey(properties, "title", "wedoc smartsheet sheet add requires --title")
			}
		})
	case "delete":
		return runWeDocPost("wedoc smartsheet sheet delete", "/cgi-bin/wedoc/smartsheet/delete_sheet", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			sheetID := fs.String("sheet-id", "", "sheet id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				mergeNonEmpty(payload, "sheet_id", *sheetID)
				if err := requirePayloadKey(payload, "docid", "wedoc smartsheet sheet delete requires --docid"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "sheet_id", "wedoc smartsheet sheet delete requires --sheet-id")
			}
		})
	case "fields":
		return runWeDocPost("wedoc smartsheet sheet fields", "/cgi-bin/wedoc/smartsheet/get_fields", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			docID := fs.String("docid", "", "doc id")
			sheetID := fs.String("sheet-id", "", "sheet id")
			offset := fs.Int("offset", -1, "offset")
			limit := fs.Int("limit", -1, "limit")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "docid", *docID)
				mergeNonEmpty(payload, "sheet_id", *sheetID)
				mergeIntIfSet(payload, "offset", *offset)
				mergeIntIfSet(payload, "limit", *limit)
				if err := requirePayloadKey(payload, "docid", "wedoc smartsheet sheet fields requires --docid"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "sheet_id", "wedoc smartsheet sheet fields requires --sheet-id")
			}
		})
	default:
		printError(fmt.Errorf("unknown wedoc smartsheet sheet subcommand: %s", args[0]))
		printWeDocHelp()
		return 1
	}
}

func runWeDocSmartSheetRecord(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDocHelp()
		return 0
	}

	switch args[0] {
	case "list":
		return runWeDocSmartSheetRecordPost("wedoc smartsheet record list", "/cgi-bin/wedoc/smartsheet/get_records", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			recordIDs := fs.String("record-ids", "", "comma separated record ids")
			fieldTitles := fs.String("field-titles", "", "comma separated field titles")
			fieldIDs := fs.String("field-ids", "", "comma separated field ids")
			viewID := fs.String("view-id", "", "view id")
			offset := fs.Int("offset", -1, "offset")
			limit := fs.Int("limit", -1, "limit")
			sortJSON := fs.String("sort-json", "", "sort JSON array")
			filterJSON := fs.String("filter-json", "", "filter_spec JSON object")
			return func(payload map[string]any) error {
				mergeStringSlice(payload, "record_ids", *recordIDs)
				mergeStringSlice(payload, "field_titles", *fieldTitles)
				mergeStringSlice(payload, "field_ids", *fieldIDs)
				mergeNonEmpty(payload, "view_id", *viewID)
				mergeIntIfSet(payload, "offset", *offset)
				mergeIntIfSet(payload, "limit", *limit)
				if err := mergeJSONArray(payload, "sort", *sortJSON); err != nil {
					return err
				}
				return mergeJSONObject(payload, "filter_spec", *filterJSON)
			}
		})
	case "add":
		return runWeDocSmartSheetRecordPost("wedoc smartsheet record add", "/cgi-bin/wedoc/smartsheet/add_records", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			recordsJSON := fs.String("records-json", "", "records JSON array")
			return func(payload map[string]any) error {
				if err := mergeJSONArray(payload, "records", *recordsJSON); err != nil {
					return err
				}
				return requirePayloadKey(payload, "records", "wedoc smartsheet record add requires --records-json")
			}
		})
	case "update":
		return runWeDocSmartSheetRecordPost("wedoc smartsheet record update", "/cgi-bin/wedoc/smartsheet/update_records", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			recordsJSON := fs.String("records-json", "", "records JSON array")
			return func(payload map[string]any) error {
				if err := mergeJSONArray(payload, "records", *recordsJSON); err != nil {
					return err
				}
				return requirePayloadKey(payload, "records", "wedoc smartsheet record update requires --records-json")
			}
		})
	case "delete":
		return runWeDocSmartSheetRecordPost("wedoc smartsheet record delete", "/cgi-bin/wedoc/smartsheet/delete_records", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			recordIDs := fs.String("record-ids", "", "comma separated record ids")
			return func(payload map[string]any) error {
				mergeStringSlice(payload, "record_ids", *recordIDs)
				return requirePayloadKey(payload, "record_ids", "wedoc smartsheet record delete requires --record-ids")
			}
		})
	default:
		printError(fmt.Errorf("unknown wedoc smartsheet record subcommand: %s", args[0]))
		printWeDocHelp()
		return 1
	}
}

func runWeDocSmartSheetRecordPost(commandName string, endpoint string, args []string, configure func(*flag.FlagSet) payloadFinalizer) int {
	return runWeDocPost(commandName, endpoint, args, func(fs *flag.FlagSet) payloadFinalizer {
		docID := fs.String("docid", "", "doc id")
		sheetID := fs.String("sheet-id", "", "sheet id")
		keyType := fs.String("key-type", "", "cell value key type")
		finalize := configure(fs)
		return func(payload map[string]any) error {
			mergeNonEmpty(payload, "docid", *docID)
			mergeNonEmpty(payload, "sheet_id", *sheetID)
			mergeNonEmpty(payload, "key_type", *keyType)
			if finalize != nil {
				if err := finalize(payload); err != nil {
					return err
				}
			}
			if err := requirePayloadKey(payload, "docid", commandName+" requires --docid"); err != nil {
				return err
			}
			return requirePayloadKey(payload, "sheet_id", commandName+" requires --sheet-id")
		}
	})
}

func runWeDocPost(commandName string, endpoint string, args []string, configure func(*flag.FlagSet) payloadFinalizer) int {
	fs := flag.NewFlagSet(commandName, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	debug := fs.Bool("debug", false, "print request debug info")
	payloadFile := fs.String("payload-file", "", "payload json file")
	finalize := configure(fs)
	if err := fs.Parse(args); err != nil {
		printError(err)
		return 1
	}

	payload, err := loadJSONInput(fs.Args(), *payloadFile)
	if err != nil {
		printError(err)
		return 1
	}
	if finalize != nil {
		if err := finalize(payload); err != nil {
			printError(err)
			return 1
		}
	}

	client, err := newClient(*debug)
	if err != nil {
		printError(err)
		return 1
	}
	result, err := client.Post(endpoint, payload)
	if err != nil {
		printError(err)
		return 1
	}
	printJSON(result)
	return 0
}

func runWeDriveSpace(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDriveHelp()
		return 0
	}

	switch args[0] {
	case "create":
		return runWeDrivePost("wedrive space create", "/cgi-bin/wedrive/space_create", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			spaceName := fs.String("space-name", "", "space name")
			spaceID := fs.String("spaceid", "", "space id")
			authInfo := fs.String("auth-info", "", "auth info JSON")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "space_name", *spaceName)
				mergeNonEmpty(payload, "spaceid", *spaceID)
				if err := mergeJSONObject(payload, "auth_info", *authInfo); err != nil {
					return err
				}
				if len(payload) == 0 {
					return errors.New("wedrive space create requires --space-name or JSON payload")
				}
				return nil
			}
		})
	case "info":
		return runWeDrivePost("wedrive space info", "/cgi-bin/wedrive/space_info", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			spaceID := fs.String("spaceid", "", "space id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "spaceid", *spaceID)
				return requirePayloadKey(payload, "spaceid", "wedrive space info requires --spaceid")
			}
		})
	case "rename":
		return runWeDrivePost("wedrive space rename", "/cgi-bin/wedrive/space_rename", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			spaceID := fs.String("spaceid", "", "space id")
			spaceName := fs.String("space-name", "", "new space name")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "spaceid", *spaceID)
				mergeNonEmpty(payload, "space_name", *spaceName)
				if err := requirePayloadKey(payload, "spaceid", "wedrive space rename requires --spaceid"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "space_name", "wedrive space rename requires --space-name")
			}
		})
	case "dismiss":
		return runWeDrivePost("wedrive space dismiss", "/cgi-bin/wedrive/space_dismiss", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			spaceID := fs.String("spaceid", "", "space id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "spaceid", *spaceID)
				return requirePayloadKey(payload, "spaceid", "wedrive space dismiss requires --spaceid")
			}
		})
	case "share":
		return runWeDrivePost("wedrive space share", "/cgi-bin/wedrive/space_share", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			spaceID := fs.String("spaceid", "", "space id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "spaceid", *spaceID)
				return requirePayloadKey(payload, "spaceid", "wedrive space share requires --spaceid")
			}
		})
	default:
		printError(fmt.Errorf("unknown wedrive space subcommand: %s", args[0]))
		printWeDriveHelp()
		return 1
	}
}

func runWeDriveFile(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		printWeDriveHelp()
		return 0
	}

	switch args[0] {
	case "list":
		return runWeDrivePost("wedrive file list", "/cgi-bin/wedrive/file_list", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			spaceID := fs.String("spaceid", "", "space id")
			fatherID := fs.String("fatherid", "", "parent folder id")
			sortType := fs.Int("sort-type", -1, "sort type")
			start := fs.Int("start", -1, "pagination start")
			limit := fs.Int("limit", -1, "page size")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "spaceid", *spaceID)
				mergeNonEmpty(payload, "fatherid", *fatherID)
				mergeIntIfSet(payload, "sort_type", *sortType)
				mergeIntIfSet(payload, "start", *start)
				mergeIntIfSet(payload, "limit", *limit)
				if err := requirePayloadKey(payload, "spaceid", "wedrive file list requires --spaceid"); err != nil {
					return err
				}
				if !hasPayloadKey(payload, "fatherid") {
					payload["fatherid"] = payload["spaceid"]
				}
				return nil
			}
		})
	case "info":
		return runWeDrivePost("wedrive file info", "/cgi-bin/wedrive/file_info", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			fileID := fs.String("fileid", "", "file id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "fileid", *fileID)
				return requirePayloadKey(payload, "fileid", "wedrive file info requires --fileid")
			}
		})
	case "download":
		return runWeDriveDownload(args[1:])
	case "upload":
		return runWeDrivePost("wedrive file upload", "/cgi-bin/wedrive/file_upload", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			path := fs.String("path", "", "local file path to upload")
			fileName := fs.String("file-name", "", "uploaded file name")
			spaceID := fs.String("spaceid", "", "space id")
			fatherID := fs.String("fatherid", "", "parent folder id")
			selectedTicket := fs.String("selected-ticket", "", "selected ticket")
			replaceFileID := fs.String("replace-fileid", "", "file id to replace when tenant API supports it")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "spaceid", *spaceID)
				mergeNonEmpty(payload, "fatherid", *fatherID)
				mergeNonEmpty(payload, "selected_ticket", *selectedTicket)
				mergeNonEmpty(payload, "replace_fileid", *replaceFileID)
				if err := mergeUploadFile(payload, *path, *fileName); err != nil {
					return err
				}
				if err := requireWeDriveUploadTarget(payload); err != nil {
					return err
				}
				if err := requirePayloadKey(payload, "file_base64_content", "wedrive file upload requires --path or file_base64_content in JSON payload"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "file_name", "wedrive file upload requires --file-name or a basename from --path")
			}
		})
	case "create":
		return runWeDrivePost("wedrive file create", "/cgi-bin/wedrive/file_create", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			spaceID := fs.String("spaceid", "", "space id")
			fatherID := fs.String("fatherid", "", "parent folder id")
			fileName := fs.String("file-name", "", "file name")
			fileType := fs.Int("file-type", -1, "file type: 1 folder, 2 normal file, 3 doc, 4 sheet, 5 form, 6 slides")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "spaceid", *spaceID)
				mergeNonEmpty(payload, "fatherid", *fatherID)
				mergeNonEmpty(payload, "file_name", *fileName)
				mergeIntIfSet(payload, "file_type", *fileType)
				if err := requirePayloadKey(payload, "spaceid", "wedrive file create requires --spaceid"); err != nil {
					return err
				}
				if !hasPayloadKey(payload, "fatherid") {
					payload["fatherid"] = payload["spaceid"]
				}
				if err := requirePayloadKey(payload, "file_name", "wedrive file create requires --file-name"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "file_type", "wedrive file create requires --file-type")
			}
		})
	case "rename":
		return runWeDrivePost("wedrive file rename", "/cgi-bin/wedrive/file_rename", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			fileID := fs.String("fileid", "", "file id")
			fileName := fs.String("file-name", "", "new file name")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "fileid", *fileID)
				mergeNonEmpty(payload, "file_name", *fileName)
				if err := requirePayloadKey(payload, "fileid", "wedrive file rename requires --fileid"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "file_name", "wedrive file rename requires --file-name")
			}
		})
	case "move":
		return runWeDrivePost("wedrive file move", "/cgi-bin/wedrive/file_move", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			fileID := fs.String("fileid", "", "file id")
			fatherID := fs.String("fatherid", "", "target parent folder id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "fileid", *fileID)
				mergeNonEmpty(payload, "fatherid", *fatherID)
				if err := requirePayloadKey(payload, "fileid", "wedrive file move requires --fileid"); err != nil {
					return err
				}
				return requirePayloadKey(payload, "fatherid", "wedrive file move requires --fatherid")
			}
		})
	case "delete":
		return runWeDrivePost("wedrive file delete", "/cgi-bin/wedrive/file_delete", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			fileID := fs.String("fileid", "", "file id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "fileid", *fileID)
				return requirePayloadKey(payload, "fileid", "wedrive file delete requires --fileid")
			}
		})
	case "share":
		return runWeDrivePost("wedrive file share", "/cgi-bin/wedrive/file_share", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			fileID := fs.String("fileid", "", "file id")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "fileid", *fileID)
				return requirePayloadKey(payload, "fileid", "wedrive file share requires --fileid")
			}
		})
	case "setting":
		return runWeDrivePost("wedrive file setting", "/cgi-bin/wedrive/file_setting", args[1:], func(fs *flag.FlagSet) payloadFinalizer {
			fileID := fs.String("fileid", "", "file id")
			authInfo := fs.String("auth-info", "", "auth info JSON")
			return func(payload map[string]any) error {
				mergeNonEmpty(payload, "fileid", *fileID)
				if err := mergeJSONObject(payload, "auth_info", *authInfo); err != nil {
					return err
				}
				return requirePayloadKey(payload, "fileid", "wedrive file setting requires --fileid")
			}
		})
	default:
		printError(fmt.Errorf("unknown wedrive file subcommand: %s", args[0]))
		printWeDriveHelp()
		return 1
	}
}

func runWeDrivePost(commandName string, endpoint string, args []string, configure func(*flag.FlagSet) payloadFinalizer) int {
	fs := flag.NewFlagSet(commandName, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	debug := fs.Bool("debug", false, "print request debug info")
	userID := fs.String("userid", "", "operator userid")
	payloadFile := fs.String("payload-file", "", "payload json file")
	finalize := configure(fs)
	if err := fs.Parse(args); err != nil {
		printError(err)
		return 1
	}

	payload, err := loadJSONInput(fs.Args(), *payloadFile)
	if err != nil {
		printError(err)
		return 1
	}
	mergeNonEmpty(payload, "userid", *userID)
	if finalize != nil {
		if err := finalize(payload); err != nil {
			printError(err)
			return 1
		}
	}
	if err := requirePayloadKey(payload, "userid", commandName+" requires --userid"); err != nil {
		printError(err)
		return 1
	}
	removeDeprecatedWeDriveUserID(payload)

	client, err := newClient(*debug)
	if err != nil {
		printError(err)
		return 1
	}
	result, err := client.Post(endpoint, payload)
	if err != nil {
		printError(err)
		return 1
	}
	printJSON(result)
	return 0
}

func runWeDriveDownload(args []string) int {
	fs := flag.NewFlagSet("wedrive file download", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	debug := fs.Bool("debug", false, "print request debug info")
	userID := fs.String("userid", "", "operator userid")
	fileID := fs.String("fileid", "", "file id")
	selectedTicket := fs.String("selected-ticket", "", "selected ticket")
	output := fs.String("output", "", "local output path")
	payloadFile := fs.String("payload-file", "", "payload json file")
	if err := fs.Parse(args); err != nil {
		printError(err)
		return 1
	}

	payload, err := loadJSONInput(fs.Args(), *payloadFile)
	if err != nil {
		printError(err)
		return 1
	}
	mergeNonEmpty(payload, "userid", *userID)
	mergeNonEmpty(payload, "fileid", *fileID)
	mergeNonEmpty(payload, "selected_ticket", *selectedTicket)
	if err := requirePayloadKey(payload, "userid", "wedrive file download requires --userid"); err != nil {
		printError(err)
		return 1
	}
	if err := requireExactlyOnePayloadKey(payload, "fileid", "selected_ticket", "wedrive file download requires exactly one of --fileid or --selected-ticket"); err != nil {
		printError(err)
		return 1
	}
	removeDeprecatedWeDriveUserID(payload)
	if strings.TrimSpace(*output) == "" {
		printError(errors.New("wedrive file download requires --output"))
		return 1
	}

	client, err := newClient(*debug)
	if err != nil {
		printError(err)
		return 1
	}
	result, err := client.Post("/cgi-bin/wedrive/file_download", payload)
	if err != nil {
		printError(err)
		return 1
	}

	downloadURL, _ := result["download_url"].(string)
	cookieName, _ := result["cookie_name"].(string)
	cookieValue, _ := result["cookie_value"].(string)
	if strings.TrimSpace(downloadURL) == "" {
		printError(errors.New("wedrive file_download response missing download_url"))
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		printError(err)
		return 1
	}
	file, err := os.Create(*output)
	if err != nil {
		printError(err)
		return 1
	}
	if err := client.DownloadWithCookie(downloadURL, cookieName, cookieValue, file); err != nil {
		_ = file.Close()
		printError(err)
		return 1
	}
	if err := file.Close(); err != nil {
		printError(err)
		return 1
	}
	result["output"] = *output
	printJSON(result)
	return 0
}

func mergeIntIfSet(payload map[string]any, key string, value int) {
	if value >= 0 {
		payload[key] = value
	}
}

func mergeJSONObject(payload map[string]any, key string, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fmt.Errorf("%s must be JSON object: %w", key, err)
	}
	payload[key] = value
	return nil
}

func mergeJSONArray(payload map[string]any, key string, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var value []any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fmt.Errorf("%s must be JSON array: %w", key, err)
	}
	payload[key] = value
	return nil
}

func mergeIntSlice(payload map[string]any, key string, csv string) error {
	values := parseCSV(csv)
	if len(values) == 0 {
		return nil
	}
	parsed := make([]int, 0, len(values))
	for _, raw := range values {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("%s must be comma separated integers: %w", key, err)
		}
		parsed = append(parsed, value)
	}
	payload[key] = parsed
	return nil
}

func mergeUploadFile(payload map[string]any, path string, fileName string) error {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		mergeNonEmpty(payload, "file_name", fileName)
		return nil
	}
	data, err := os.ReadFile(trimmedPath)
	if err != nil {
		return err
	}
	if _, exists := payload["file_base64_content"]; !exists {
		payload["file_base64_content"] = base64.StdEncoding.EncodeToString(data)
	}
	if _, exists := payload["file_name"]; !exists {
		name := strings.TrimSpace(fileName)
		if name == "" {
			name = filepath.Base(trimmedPath)
		}
		payload["file_name"] = name
	}
	return nil
}

func requirePayloadKey(payload map[string]any, key string, message string) error {
	if !hasPayloadKey(payload, key) {
		return errors.New(message)
	}
	return nil
}

func hasPayloadKey(payload map[string]any, key string) bool {
	value, exists := payload[key]
	if !exists {
		return false
	}
	if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
		return false
	}
	if values, ok := value.([]any); ok && len(values) == 0 {
		return false
	}
	if values, ok := value.([]string); ok && len(values) == 0 {
		return false
	}
	if values, ok := value.([]int); ok && len(values) == 0 {
		return false
	}
	return true
}

func requireExactlyOnePayloadKey(payload map[string]any, first string, second string, message string) error {
	firstSet := hasPayloadKey(payload, first)
	secondSet := hasPayloadKey(payload, second)
	if firstSet == secondSet {
		return errors.New(message)
	}
	return nil
}

func requireWeDriveUploadTarget(payload map[string]any) error {
	hasSpaceTarget := hasPayloadKey(payload, "spaceid")
	hasTicketTarget := hasPayloadKey(payload, "selected_ticket")
	if hasSpaceTarget == hasTicketTarget {
		return errors.New("wedrive file upload requires exactly one target: --spaceid/--fatherid or --selected-ticket")
	}
	return nil
}

func removeDeprecatedWeDriveUserID(payload map[string]any) {
	delete(payload, "userid")
}

func newClient(debug bool) (*wecom.Client, error) {
	cfg, err := config.Resolve()
	if err != nil {
		return nil, err
	}
	return wecom.New(cfg, debug), nil
}

func loadJSONInput(positional []string, payloadFile string) (map[string]any, error) {
	if len(positional) > 1 {
		return nil, errors.New("only one positional JSON payload is supported")
	}

	if payloadFile != "" && len(positional) == 1 {
		return nil, errors.New("only one of positional JSON payload or --payload-file can be provided")
	}

	if payloadFile != "" {
		if payloadFile == "-" {
			data, err := readAllFrom(os.Stdin)
			if err != nil {
				return nil, err
			}
			return decodePayload(data)
		}
		data, err := os.ReadFile(payloadFile)
		if err != nil {
			return nil, err
		}
		return decodePayload(data)
	}

	if len(positional) == 1 {
		return decodePayload([]byte(positional[0]))
	}
	return map[string]any{}, nil
}

func readAllFrom(file *os.File) ([]byte, error) {
	return io.ReadAll(file)
}

func decodePayload(raw []byte) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func parseCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseInvitees(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "{") {
		return parseCSV(trimmed), nil
	}

	var value any
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return nil, fmt.Errorf("invalid --invitees JSON: %w", err)
	}
	return extractInviteeUserIDs(value)
}

func extractInviteeUserIDs(value any) ([]string, error) {
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			ids, err := extractInviteeUserIDs(item)
			if err != nil {
				return nil, err
			}
			result = append(result, ids...)
		}
		return result, nil
	case map[string]any:
		if raw, ok := typed["userid"]; ok {
			return extractInviteeUserIDs(raw)
		}
		if raw, ok := typed["userids"]; ok {
			return extractInviteeUserIDs(raw)
		}
		return nil, errors.New(`invalid --invitees JSON object: expected "userid" or "userids"`)
	case string:
		return parseCSV(typed), nil
	default:
		return nil, fmt.Errorf("invalid --invitees JSON value %T", value)
	}
}

func parseCSVInts(raw string) ([]int, error) {
	parts := parseCSV(raw)
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		var value int
		if _, err := fmt.Sscanf(part, "%d", &value); err != nil {
			return nil, fmt.Errorf("invalid integer value %q", part)
		}
		result = append(result, value)
	}
	return result, nil
}

func resolveMeetingStart(raw string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return wecom.DefaultMeetingStart(time.Now()), nil
	}
	return wecom.ParseTimeToEpoch(raw)
}

func validateMeetingListWindow(payload map[string]any) error {
	begin, ok := numericPayloadValue(payload["begin_time"])
	if !ok {
		return errors.New("meeting list begin_time must be an epoch number")
	}
	end, ok := numericPayloadValue(payload["end_time"])
	if !ok {
		return errors.New("meeting list end_time must be an epoch number")
	}
	if end <= begin {
		return errors.New("meeting list --end must be after --start")
	}
	const maxMeetingListWindow = 31 * 24 * 60 * 60
	if end-begin > maxMeetingListWindow {
		return errors.New("meeting list time range must be 31 days or less; query by month or a narrower window")
	}
	return nil
}

func numericPayloadValue(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func mergeNonEmpty(payload map[string]any, key string, value string) {
	if strings.TrimSpace(value) != "" {
		payload[key] = strings.TrimSpace(value)
	}
}

func mergeStringSlice(payload map[string]any, key string, csv string) {
	if values := parseCSV(csv); len(values) > 0 {
		payload[key] = values
	}
}

func ensureNestedMap(payload map[string]any, key string) (map[string]any, error) {
	if raw, exists := payload[key]; exists {
		nested, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s must be a JSON object", key)
		}
		return nested, nil
	}
	nested := map[string]any{}
	payload[key] = nested
	return nested, nil
}

func buildUserObjects(csv string) []map[string]any {
	users := parseCSV(csv)
	if len(users) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(users))
	for _, user := range users {
		result = append(result, map[string]any{"userid": user})
	}
	return result
}

func mergeAttendees(payload map[string]any, csv string) {
	if attendees := buildUserObjects(csv); len(attendees) > 0 {
		payload["attendees"] = attendees
	}
}

func buildShares(csv string, permission int) []map[string]any {
	users := parseCSV(csv)
	if len(users) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(users))
	for _, user := range users {
		result = append(result, map[string]any{
			"userid":     user,
			"permission": permission,
		})
	}
	return result
}

func buildPublicRange(userCSV string, partyCSV string) (map[string]any, error) {
	result := map[string]any{}
	if users := parseCSV(userCSV); len(users) > 0 {
		result["userids"] = users
	}
	if partyCSV = strings.TrimSpace(partyCSV); partyCSV != "" {
		partyIDs, err := parseCSVInts(partyCSV)
		if err != nil {
			return nil, err
		}
		if len(partyIDs) > 0 {
			result["partyids"] = partyIDs
		}
	}
	return result, nil
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func defaultAgentID(fallback int) int {
	raw := strings.TrimSpace(os.Getenv("WECOM_AGENT_ID"))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func normalizeMeetingDuration(seconds int, minutes int) int {
	if minutes > 0 {
		return minutes * 60
	}
	if seconds > 0 && seconds < 300 {
		return seconds * 60
	}
	return seconds
}

func wantsHelp(arg string) bool {
	switch arg {
	case "help", "--help", "-h":
		return true
	default:
		return false
	}
}

func runSpec(args []string) int {
	switch len(args) {
	case 0:
		printJSON(specCatalog())
		return 0
	case 1:
		command, ok := findCommandSpec(args[0])
		if !ok {
			printError(fmt.Errorf("unknown spec command: %s", args[0]))
			return 1
		}
		printJSON(command)
		return 0
	case 2:
		subcommand, ok := findSubcommandSpec(args[0], args[1])
		if !ok {
			printError(fmt.Errorf("unknown spec subcommand: %s %s", args[0], args[1]))
			return 1
		}
		printJSON(subcommand)
		return 0
	default:
		printError(errors.New("usage: wecom-go spec [command] [subcommand]"))
		return 1
	}
}

func printJSON(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		printError(err)
		return
	}
	fmt.Println(string(data))
}

func printError(err error) {
	printJSON(map[string]any{
		"error": err.Error(),
	})
}

func printHelp() {
	fmt.Print(`wecom-go - Minimal WeCom CLI for agent integration

Commands:
  config set|show|path|clear
  token [--refresh]
  contact list-ids
  calendar create|update|get|delete
  schedule create|update|get|list|delete|add-attendees|remove-attendees
  todo create|list|get|update|delete|change-status
  meeting create|update|get|list|cancel
  wedrive space create|info|rename|dismiss|share
  wedrive file list|info|download|upload|create|rename|move|delete|share|setting
  wedoc doc create|info|rename|delete|share|auth
  wedoc smartsheet sheet list|add|delete|fields
  wedoc smartsheet record list|add|update|delete
  spec [schedule|todo] [subcommand]

Examples:
  wecom-go config set --corp-id ww123 --corp-secret secret
  wecom-go token
  wecom-go contact list-ids
  wecom-go calendar create --summary "Team Calendar" --admins zhangsan
  wecom-go schedule create --summary "Project Review" --start 2026-05-12T15:00:00+08:00 --end 2026-05-12T16:00:00+08:00 --admins zhangsan
  wecom-go todo create --content "Submit weekly report"
  wecom-go meeting create --admin-userid zhangsan --invitees zhangsan
  wecom-go meeting update --meeting-id MEETING_ID --title "New topic"
  wecom-go meeting list --userid zhangsan --start 2026-05-07T09:00:00+08:00 --end 2026-05-07T18:00:00+08:00
  wecom-go wedrive file download --userid zhangsan --fileid FILEID --output ./file.docx
  wecom-go wedrive file upload --userid zhangsan --spaceid SPACEID --fatherid FOLDERID --path ./file.docx
  wecom-go wedoc doc create --spaceid SPACEID --doc-type 3 --doc-name "Project Notes" --admin-users zhangsan
  wecom-go wedoc smartsheet record list --docid DOCID --sheet-id SHEETID --limit 20
  wecom-go spec schedule
`)
}

func printConfigHelp() {
	fmt.Print(`config commands:
  wecom-go config set --corp-id <id> --corp-secret <secret> [--base-url <url>] [--timeout 30]
  wecom-go config show
  wecom-go config path
  wecom-go config clear`)
}

func printContactHelp() {
	fmt.Print(`contact commands:
  wecom-go contact list-ids [--cursor <cursor>] [--limit 100] [json_payload]

Example:
  wecom-go contact list-ids '{"limit":50}'`)
}

func printCalendarHelp() {
	fmt.Print(`calendar commands:
  wecom-go calendar create [--admins "a,b"] [--summary <text>] [--color <#RRGGBB>] [--description <text>] [--shares "u1,u2"] [--public] [--public-userids "u1,u2"] [--public-partyids "1,2"] [json_payload]
  wecom-go calendar update --cal-id <id> [--summary <text>] [--color <#RRGGBB>] [--description <text>] [--shares "u1,u2"] [--skip-public-range] [json_payload]
  wecom-go calendar get --cal-id <id> [json_payload]
  wecom-go calendar delete --cal-id <id> [json_payload]`)
}

func printScheduleHelp() {
	fmt.Print(renderCommandHelp("schedule"))
}

func printTodoHelp() {
	fmt.Print(renderCommandHelp("todo"))
}

func printMeetingHelp() {
	fmt.Print(`meeting commands:
  wecom-go meeting create [--admin-userid <userid>] [--invitees "a,b"] --start <time> (--duration <seconds> | --duration-minutes <minutes>) (--location <text> | --no-location) [json_payload]
  wecom-go meeting update --meeting-id <id> [--title <title>] [--start <time>] [--duration <seconds>] [--invitees "a,b"] [json_payload]
  wecom-go meeting list --userid <userid> --start <time> --end <time>
  wecom-go meeting get --meeting-id <id>
  wecom-go meeting cancel --meeting-id <id>`)
}

func printWeDriveHelp() {
	fmt.Print(`wedrive commands:
  wecom-go wedrive space create --userid <userid> --space-name <name> [json_payload]
  wecom-go wedrive space info --userid <userid> --spaceid <id> [json_payload]
  wecom-go wedrive space rename --userid <userid> --spaceid <id> --space-name <name> [json_payload]
  wecom-go wedrive space dismiss --userid <userid> --spaceid <id> [json_payload]
  wecom-go wedrive space share --userid <userid> --spaceid <id> [json_payload]
  wecom-go wedrive file list --userid <userid> --spaceid <id> [--fatherid <id>] [--start 0] [--limit 100] [json_payload]
  wecom-go wedrive file info --userid <userid> --fileid <id> [json_payload]
  wecom-go wedrive file download --userid <userid> (--fileid <id> | --selected-ticket <ticket>) --output <path> [json_payload]
  wecom-go wedrive file upload --userid <userid> (--spaceid <id> [--fatherid <id>] | --selected-ticket <ticket>) --path <local_file> [--file-name <name>] [--replace-fileid <id>] [json_payload]
  wecom-go wedrive file create --userid <userid> --spaceid <id> --file-name <name> --file-type <1 folder|2 file|3 doc|4 sheet|5 form|6 slides> [json_payload]
  wecom-go wedrive file rename --userid <userid> --fileid <id> --file-name <name> [json_payload]
  wecom-go wedrive file move --userid <userid> --fileid <id> --fatherid <target_folder_id> [json_payload]
  wecom-go wedrive file delete --userid <userid> --fileid <id> [json_payload]
  wecom-go wedrive file share --userid <userid> --fileid <id> [json_payload]
  wecom-go wedrive file setting --userid <userid> --fileid <id> [--auth-info <json>] [json_payload]`)
}

func printWeDocHelp() {
	fmt.Print(`wedoc commands:
  wecom-go wedoc doc create --spaceid <id> --doc-type <type> --doc-name <name> [--fatherid <id>] [--admin-users "u1,u2"] [json_payload]
  wecom-go wedoc doc info --docid <id> [json_payload]
  wecom-go wedoc doc rename (--docid <id> | --formid <id>) --new-name <name> [json_payload]
  wecom-go wedoc doc delete (--docid <id> | --formid <id>) [json_payload]
  wecom-go wedoc doc share (--docid <id> | --formid <id>) [json_payload]
  wecom-go wedoc doc auth get --docid <id> [json_payload]
  wecom-go wedoc doc auth member --docid <id> [--update-members-json <json_array>] [--delete-members-json <json_array>] [json_payload]
  wecom-go wedoc doc auth join-rule --docid <id> [json_payload]
  wecom-go wedoc doc auth security --docid <id> [json_payload]
  wecom-go wedoc content doc data --docid <id> [json_payload]
  wecom-go wedoc content doc modify --docid <id> [json_payload]
  wecom-go wedoc content sheet rowcol --docid <id> --sheet-id <id> [json_payload]
  wecom-go wedoc content sheet data --docid <id> --sheet-id <id> [json_payload]
  wecom-go wedoc content sheet modify --docid <id> --sheet-id <id> [json_payload]
  wecom-go wedoc form create --spaceid <id> --form-info-json <json_object> [--fatherid <id>] [json_payload]
  wecom-go wedoc form modify --formid <id> --form-info-json <json_object> [json_payload]
  wecom-go wedoc form info --formid <id> [json_payload]
  wecom-go wedoc form statistic --repeated-id <id> --req-type <int> [--start-time <epoch>] [--end-time <epoch>] [json_payload]
  wecom-go wedoc form answer --repeated-id <id> --answer-ids "1,2" [json_payload]
  wecom-go wedoc smartsheet sheet list --docid <id> [json_payload]
  wecom-go wedoc smartsheet sheet add --docid <id> --title <name> [--index <int>] [json_payload]
  wecom-go wedoc smartsheet sheet delete --docid <id> --sheet-id <id> [json_payload]
  wecom-go wedoc smartsheet sheet fields --docid <id> --sheet-id <id> [--offset 0] [--limit 100] [json_payload]
  wecom-go wedoc smartsheet record list --docid <id> --sheet-id <id> [--view-id <id>] [--record-ids "r1,r2"] [--field-titles "name,status"] [--offset 0] [--limit 100] [json_payload]
  wecom-go wedoc smartsheet record add --docid <id> --sheet-id <id> --records-json <json_array> [--key-type <type>] [json_payload]
  wecom-go wedoc smartsheet record update --docid <id> --sheet-id <id> --records-json <json_array> [--key-type <type>] [json_payload]
  wecom-go wedoc smartsheet record delete --docid <id> --sheet-id <id> --record-ids "r1,r2" [json_payload]`)
}

func init() {
	flag.CommandLine.SetOutput(os.Stderr)
}
