# wecom-go-skill

Minimal WeCom Go CLI for agent integration.

## Scope

Current commands:

- `config set|show|path|clear`
- `token`
- `contact list-ids`
- `calendar create|update|get|delete`
- `schedule create|update|get|list|delete|add-attendees|remove-attendees`
- `todo create|list|get|update|delete|change-status`
- `meeting create|update|get|list|cancel`
- `wedrive space create|info|rename|dismiss|share`
- `wedrive file list|info|download|upload|create|rename|move|delete|share|setting`
- `wedoc doc create|info|rename|delete|share|auth`
- `wedoc content doc data|modify`
- `wedoc content sheet rowcol|data|modify`
- `wedoc form create|modify|info|statistic|answer`
- `wedoc smartsheet sheet list|add|delete|fields`
- `wedoc smartsheet record list|add|update|delete`
- `spec [schedule|todo] [subcommand]`

## Build

```bash
go build -o wecom-go ./cmd/wecom-go
```

## Release

```bash
make release
```

Release archives are written to `dist/release/` for macOS, Linux, and Windows. Archives include only the binary, this README, and a manifest; credentials are not packaged.

## Config

```bash
wecom-go config set --corp-id wwxxxxxxxxxxxxxxxx --corp-secret xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Environment variables are also supported:

```bash
set WECOM_CORP_ID=wwxxxxxxxxxxxxxxxx
set WECOM_CORP_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
set WECOM_BASE_URL=https://qyapi.weixin.qq.com
set WECOM_TIMEOUT=30
```

Optional environment variable:

- `WECOM_GO_CONFIG_DIR`: override the local config directory

## Skills

This project currently includes:

- `skills/wecomgo-contact`
- `skills/wecomgo-schedule`
- `skills/wecomgo-todo`
- `skills/wecomgo-meeting`

They are designed so upper-layer agents can prefer higher-level capabilities instead of assembling raw HTTP parameters directly.

## WeDrive files

WeDrive commands operate on real WeCom Drive spaces and files:

```bash
wecom-go wedrive file list --userid zhangsan --spaceid SPACEID
wecom-go wedrive file download --userid zhangsan --fileid FILEID --output ./file.docx
wecom-go wedrive file upload --userid zhangsan --spaceid SPACEID --fatherid FOLDERID --path ./file.docx
wecom-go wedrive file upload --userid zhangsan --spaceid SPACEID --fatherid FOLDERID --replace-fileid FILEID --path ./file-edited.xlsx
```

For Office files such as `.docx`, `.pptx`, and `.xlsx`, download the binary file first, edit it locally with the appropriate document, presentation, or spreadsheet tooling, then upload the edited file back to WeCom Drive. Upload as a new file by default; use `--replace-fileid` only when the caller explicitly wants to overwrite or replace the original.

## WeDoc online docs

WeDoc commands operate on WeCom online documents and smart sheets created or managed through the WeCom Docs API:

```bash
wecom-go wedoc doc create --spaceid SPACEID --doc-type 3 --doc-name "Project Notes" --admin-users zhangsan
wecom-go wedoc doc info --docid DOCID
wecom-go wedoc doc share --docid DOCID
wecom-go wedoc content doc data --docid DOCID
wecom-go wedoc content sheet data --docid DOCID --sheet-id SHEETID
wecom-go wedoc form info --formid FORMID
wecom-go wedoc smartsheet sheet list --docid DOCID
wecom-go wedoc smartsheet record list --docid DOCID --sheet-id SHEETID --limit 20
wecom-go wedoc smartsheet record add --docid DOCID --sheet-id SHEETID --key-type CELL_VALUE_KEY_TYPE_FIELD_TITLE --records-json '[{"values":{"Name":[{"type":"text","text":"demo"}]}}]'
```

Use `wedoc` for online doc IDs, online sheet content, collection forms, and smart sheet records. Content modification commands intentionally accept official JSON payloads instead of inventing a simplified schema. Use `wedrive` for binary files such as `.docx`, `.pptx`, and `.xlsx`.

## Machine-readable spec

Upper-layer agents can inspect a structured command contract instead of free-form help text:

```bash
wecom-go spec
wecom-go spec schedule
wecom-go spec todo change-status
```

The current spec coverage focuses on the most drift-prone command groups first:

- `schedule`
- `todo`

## Notes

- `calendar` and `schedule` commands use the official WeCom OA endpoints under `/cgi-bin/oa/...`.
- `wedrive` commands use the official WeCom Drive endpoints under `/cgi-bin/wedrive/...`.
- `todo` commands currently surface the upstream HTTP status and endpoint details directly.
- If your tenant/app does not expose the WeCom todo API on `qyapi.weixin.qq.com`, the CLI now reports that explicitly instead of failing with a bare `EOF`.
