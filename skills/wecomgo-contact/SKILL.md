---
name: wecomgo-contact
description: 通过 wecom-go 查询企业微信可见范围内的 userid 列表，供其他 skill 做人员定位。适用于“查一下有哪些可用用户ID”“把某个人的 userid 找出来”“会议或消息发送前先取可见用户列表”等场景。
metadata:
  requires:
    bins: ["wecom-go"]
  cliHelp: "wecom-go contact list-ids --help"
---

# wecomgo-contact

通过 `wecom-go` 执行企业微信通讯录最小查询能力。

## 调用方式

```bash
wecom-go contact list-ids '{"limit": 100}'
```

或直接：

```bash
wecom-go contact list-ids
```

## 返回结果

返回企业微信 `user/list_id` 接口原始 JSON，重点字段通常有：

- `userid_list`
- `next_cursor`

## 使用规则

- 这是最小 MVP，只保证查询 **可见范围内的 userid 列表**，不做姓名反查。
- 当上层智能体需要选择目标用户时，优先要求用户直接提供 `userid`。
- 如果返回里有 `next_cursor`，说明还有下一页；需要时继续调用并传入 `cursor`。

## 推荐工作流

1. 当会议创建或其他写操作需要 `userid` 时，先调用一次 `contact list-ids`。
2. 如果用户已经提供明确 `userid`，可以跳过此步骤。
3. 如果列表很多，只展示前几项并提示可继续翻页。
