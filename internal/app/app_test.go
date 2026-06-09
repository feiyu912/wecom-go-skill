package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadJSONInputFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "payload.json")
	if err := os.WriteFile(path, []byte(`{"content":"drink water","todo_status":1}`), 0o600); err != nil {
		t.Fatalf("write payload file: %v", err)
	}

	payload, err := loadJSONInput(nil, path)
	if err != nil {
		t.Fatalf("loadJSONInput returned error: %v", err)
	}

	if got := payload["content"]; got != "drink water" {
		t.Fatalf("content = %v, want %q", got, "drink water")
	}
	if got := payload["todo_status"]; got != float64(1) {
		t.Fatalf("todo_status = %v, want %v", got, float64(1))
	}
}

func TestLoadJSONInputFromStdin(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	if _, err := writer.WriteString(`{"content":"drink water","todo_status":1}`); err != nil {
		t.Fatalf("write stdin payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	originalStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = originalStdin
		_ = reader.Close()
	})

	payload, err := loadJSONInput(nil, "-")
	if err != nil {
		t.Fatalf("loadJSONInput returned error: %v", err)
	}

	if got := payload["content"]; got != "drink water" {
		t.Fatalf("content = %v, want %q", got, "drink water")
	}
	if got := payload["todo_status"]; got != float64(1) {
		t.Fatalf("todo_status = %v, want %v", got, float64(1))
	}
}

func TestEnsureNestedMapCreatesAndReusesObject(t *testing.T) {
	payload := map[string]any{}

	nested, err := ensureNestedMap(payload, "schedule")
	if err != nil {
		t.Fatalf("ensureNestedMap returned error: %v", err)
	}
	nested["summary"] = "demo"

	reused, err := ensureNestedMap(payload, "schedule")
	if err != nil {
		t.Fatalf("ensureNestedMap returned error on reuse: %v", err)
	}
	if reused["summary"] != "demo" {
		t.Fatalf("summary = %v, want %q", reused["summary"], "demo")
	}
}

func TestBuildPublicRangeParsesUsersAndParties(t *testing.T) {
	got, err := buildPublicRange("zhangsan,lisi", "1,2")
	if err != nil {
		t.Fatalf("buildPublicRange returned error: %v", err)
	}

	if len(got["userids"].([]string)) != 2 {
		t.Fatalf("unexpected userids: %#v", got["userids"])
	}
	if len(got["partyids"].([]int)) != 2 {
		t.Fatalf("unexpected partyids: %#v", got["partyids"])
	}
}

func TestBuildPublicRangeRejectsInvalidPartyIDs(t *testing.T) {
	if _, err := buildPublicRange("", "x"); err == nil {
		t.Fatal("buildPublicRange returned nil error, want invalid integer error")
	}
}

func TestRunSpecCommandOutputsMachineReadableScheduleContract(t *testing.T) {
	output := captureStdout(t, func() {
		if code := Run([]string{"spec", "schedule"}); code != 0 {
			t.Fatalf("Run returned code %d, want 0", code)
		}
	})

	var spec CommandSpec
	if err := json.Unmarshal([]byte(output), &spec); err != nil {
		t.Fatalf("unmarshal spec output: %v\noutput=%s", err, output)
	}

	if spec.Name != "schedule" {
		t.Fatalf("spec.Name = %q, want %q", spec.Name, "schedule")
	}

	found := false
	for _, subcommand := range spec.Subcommands {
		if subcommand.Name == "delete" {
			found = true
			if len(subcommand.SafetyNotes) == 0 {
				t.Fatal("schedule delete safety notes missing")
			}
		}
	}
	if !found {
		t.Fatal("schedule delete subcommand missing from spec")
	}
}

func TestRunSpecSubcommandOutputsTodoChangeStatusContract(t *testing.T) {
	output := captureStdout(t, func() {
		if code := Run([]string{"spec", "todo", "change-status"}); code != 0 {
			t.Fatalf("Run returned code %d, want 0", code)
		}
	})

	var spec SubcommandSpec
	if err := json.Unmarshal([]byte(output), &spec); err != nil {
		t.Fatalf("unmarshal subcommand spec output: %v\noutput=%s", err, output)
	}

	if spec.Name != "change-status" {
		t.Fatalf("spec.Name = %q, want %q", spec.Name, "change-status")
	}

	requiredStatus := false
	for _, arg := range spec.Args {
		if arg.Name == "--status" && arg.Required {
			requiredStatus = true
		}
	}
	if !requiredStatus {
		t.Fatal("--status should be required in todo change-status spec")
	}
}

func TestRenderCommandHelpUsesSharedSpec(t *testing.T) {
	help := renderCommandHelp("todo")
	if !strings.Contains(help, "wecom-go todo change-status --todo-id <id> --status <int> [json_payload]") {
		t.Fatalf("todo help missing command usage:\n%s", help)
	}
	if !strings.Contains(help, "wecom-go spec todo") {
		t.Fatalf("todo help missing spec hint:\n%s", help)
	}
}

func TestDefaultAgentIDReadsEnvironment(t *testing.T) {
	t.Setenv("WECOM_AGENT_ID", "1000282")

	if got := defaultAgentID(0); got != 1000282 {
		t.Fatalf("defaultAgentID() = %d, want 1000282", got)
	}
}

func TestDefaultAgentIDFallsBackForInvalidEnvironment(t *testing.T) {
	t.Setenv("WECOM_AGENT_ID", "not-an-int")

	if got := defaultAgentID(7); got != 7 {
		t.Fatalf("defaultAgentID() = %d, want fallback 7", got)
	}
}

func TestParseInviteesAcceptsCSV(t *testing.T) {
	got, err := parseInvitees("029235,011949")
	if err != nil {
		t.Fatalf("parseInvitees returned error: %v", err)
	}
	if strings.Join(got, ",") != "029235,011949" {
		t.Fatalf("parseInvitees = %#v", got)
	}
}

func TestParseInviteesAcceptsObjectArray(t *testing.T) {
	got, err := parseInvitees(`[{"userid":"029235"},{"userid":"011949"}]`)
	if err != nil {
		t.Fatalf("parseInvitees returned error: %v", err)
	}
	if strings.Join(got, ",") != "029235,011949" {
		t.Fatalf("parseInvitees = %#v", got)
	}
}

func TestParseInviteesAcceptsUserIDArrayObject(t *testing.T) {
	got, err := parseInvitees(`{"userid":["029235","011949"]}`)
	if err != nil {
		t.Fatalf("parseInvitees returned error: %v", err)
	}
	if strings.Join(got, ",") != "029235,011949" {
		t.Fatalf("parseInvitees = %#v", got)
	}
}

func TestNormalizeMeetingDurationTreatsSmallSecondsAsMinutes(t *testing.T) {
	if got := normalizeMeetingDuration(60, -1); got != 3600 {
		t.Fatalf("normalizeMeetingDuration(60, -1) = %d, want 3600", got)
	}
}

func TestNormalizeMeetingDurationKeepsExplicitSeconds(t *testing.T) {
	if got := normalizeMeetingDuration(1800, -1); got != 1800 {
		t.Fatalf("normalizeMeetingDuration(1800, -1) = %d, want 1800", got)
	}
}

func TestNormalizeMeetingDurationPrefersMinutes(t *testing.T) {
	if got := normalizeMeetingDuration(1800, 60); got != 3600 {
		t.Fatalf("normalizeMeetingDuration(1800, 60) = %d, want 3600", got)
	}
}

func TestMeetingCreateRequiresLocationOrExplicitNoLocation(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{
			"meeting", "create",
			"--title", "demo",
			"--start", "2026-05-13T16:00:00+08:00",
			"--duration-minutes", "30",
			"--admin-userid", "029235",
			"--invitees", "029235",
		}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "meeting create requires --location or --no-location") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestMergeUploadFileUsesBasenameAndBase64Content(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.docx")
	if err := os.WriteFile(path, []byte("office bytes"), 0o600); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	payload := map[string]any{}
	if err := mergeUploadFile(payload, path, ""); err != nil {
		t.Fatalf("mergeUploadFile returned error: %v", err)
	}

	if got := payload["file_name"]; got != "demo.docx" {
		t.Fatalf("file_name = %v, want demo.docx", got)
	}
	want := base64.StdEncoding.EncodeToString([]byte("office bytes"))
	if got := payload["file_base64_content"]; got != want {
		t.Fatalf("file_base64_content = %v, want %v", got, want)
	}
}

func TestMergeUploadFileHonorsExplicitFileName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.xlsx")
	if err := os.WriteFile(path, []byte("sheet bytes"), 0o600); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	payload := map[string]any{}
	if err := mergeUploadFile(payload, path, "renamed.xlsx"); err != nil {
		t.Fatalf("mergeUploadFile returned error: %v", err)
	}

	if got := payload["file_name"]; got != "renamed.xlsx" {
		t.Fatalf("file_name = %v, want renamed.xlsx", got)
	}
}

func TestWeDriveFileUploadRequiresSpaceIDBeforeNetwork(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.docx")
	if err := os.WriteFile(path, []byte("office bytes"), 0o600); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	output := captureStdout(t, func() {
		if code := Run([]string{"wedrive", "file", "upload", "--userid", "029235", "--path", path}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedrive file upload requires exactly one target") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestWeDriveFileListRequiresSpaceIDBeforeNetwork(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{"wedrive", "file", "list", "--userid", "029235"}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedrive file list requires --spaceid") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestRemoveDeprecatedWeDriveUserID(t *testing.T) {
	payload := map[string]any{
		"userid":  "029235",
		"spaceid": "space",
	}

	removeDeprecatedWeDriveUserID(payload)

	if _, ok := payload["userid"]; ok {
		t.Fatalf("deprecated userid remains in payload: %#v", payload)
	}
	if got := payload["spaceid"]; got != "space" {
		t.Fatalf("spaceid = %v, want space", got)
	}
}

func TestWeDriveFileListDefaultsFatherIDToSpaceID(t *testing.T) {
	payload := map[string]any{
		"spaceid": "space",
	}
	if !hasPayloadKey(payload, "spaceid") {
		t.Fatal("spaceid should be present")
	}
	if !hasPayloadKey(payload, "fatherid") {
		payload["fatherid"] = payload["spaceid"]
	}
	if got := payload["fatherid"]; got != "space" {
		t.Fatalf("fatherid = %v, want space", got)
	}
}

func TestWeDriveFileCreateDefaultsRootFatherIDOnly(t *testing.T) {
	payload := map[string]any{
		"spaceid":   "space",
		"file_name": "demo",
	}
	if !hasPayloadKey(payload, "fatherid") {
		payload["fatherid"] = payload["spaceid"]
	}

	if got := payload["fatherid"]; got != "space" {
		t.Fatalf("fatherid = %v, want space", got)
	}
	if hasPayloadKey(payload, "file_type") {
		t.Fatalf("file_type should not be defaulted: %#v", payload)
	}
}

func TestWeDriveFileCreateRequiresFileTypeBeforeNetwork(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{"wedrive", "file", "create", "--userid", "029235", "--spaceid", "space", "--file-name", "demo"}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedrive file create requires --file-type") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestWeDriveFileUploadRequiresUserIDBeforeNetwork(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.docx")
	if err := os.WriteFile(path, []byte("office bytes"), 0o600); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	output := captureStdout(t, func() {
		if code := Run([]string{"wedrive", "file", "upload", "--spaceid", "space", "--path", path}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedrive file upload requires --userid") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestWeDriveFileDownloadRequiresExactlyOneTarget(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{
			"wedrive", "file", "download",
			"--userid", "029235",
			"--fileid", "file",
			"--selected-ticket", "ticket",
			"--output", filepath.Join(t.TempDir(), "demo.docx"),
		}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedrive file download requires exactly one") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestHelpIncludesWeDriveCommands(t *testing.T) {
	output := captureStdout(t, func() {
		if code := Run([]string{"wedrive"}); code != 0 {
			t.Fatalf("Run returned code %d, want 0", code)
		}
	})

	if !strings.Contains(output, "wecom-go wedrive file download") {
		t.Fatalf("wedrive help missing download command:\n%s", output)
	}
	if !strings.Contains(output, "wecom-go wedrive file upload") {
		t.Fatalf("wedrive help missing upload command:\n%s", output)
	}
}

func TestMergeJSONArrayRejectsObject(t *testing.T) {
	payload := map[string]any{}
	if err := mergeJSONArray(payload, "records", `{"values":{}}`); err == nil {
		t.Fatal("mergeJSONArray returned nil error, want JSON array error")
	}
}

func TestWeDocDocCreateRequiresSpaceIDBeforeNetwork(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{"wedoc", "doc", "create", "--doc-type", "3", "--doc-name", "demo"}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedoc doc create requires --spaceid") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestWeDocDocCreateRejectsNonNumericDocTypeBeforeNetwork(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{"wedoc", "doc", "create", "--spaceid", "space", "--doc-type", "doc", "--doc-name", "demo"}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "invalid value") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestWeDocDocRenameRequiresExactlyOneID(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{"wedoc", "doc", "rename", "--docid", "doc", "--formid", "form", "--new-name", "demo"}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedoc doc rename requires exactly one") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestWeDocSmartSheetRecordAddRequiresRecordsArrayBeforeNetwork(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{"wedoc", "smartsheet", "record", "add", "--docid", "doc", "--sheet-id", "sheet"}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedoc smartsheet record add requires --records-json") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestWeDocContentSheetDataRequiresSheetIDBeforeNetwork(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{"wedoc", "content", "sheet", "data", "--docid", "doc"}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedoc content sheet data requires --sheet-id") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestWeDocFormCreateRequiresFormInfoBeforeNetwork(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{"wedoc", "form", "create", "--spaceid", "space"}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "wedoc form create requires --form-info-json") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestWeDocFormAnswerRequiresIntegerAnswerIDs(t *testing.T) {
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")

	output := captureStdout(t, func() {
		if code := Run([]string{"wedoc", "form", "answer", "--repeated-id", "repeat", "--answer-ids", "one"}); code != 1 {
			t.Fatalf("Run returned code %d, want 1", code)
		}
	})

	if !strings.Contains(output, "answer_ids must be comma separated integers") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestHelpIncludesWeDocCommands(t *testing.T) {
	output := captureStdout(t, func() {
		if code := Run([]string{"wedoc"}); code != 0 {
			t.Fatalf("Run returned code %d, want 0", code)
		}
	})

	if !strings.Contains(output, "wecom-go wedoc doc create") {
		t.Fatalf("wedoc help missing doc create command:\n%s", output)
	}
	if !strings.Contains(output, "wecom-go wedoc smartsheet record add") {
		t.Fatalf("wedoc help missing smartsheet record add command:\n%s", output)
	}
	if !strings.Contains(output, "wecom-go wedoc content sheet data") {
		t.Fatalf("wedoc help missing content sheet data command:\n%s", output)
	}
	if !strings.Contains(output, "wecom-go wedoc form create") {
		t.Fatalf("wedoc help missing form create command:\n%s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})

	outputCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		outputCh <- buf.String()
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	return <-outputCh
}
