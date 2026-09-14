# 账号内容清理

本模块通过已登录的网页操作自己的账号。它提供候选盘点、账号绑定的执行计划、单条执行和本地回执。**当前不是八项全自动清空工具**：网页没有完整历史或操作入口的项目会明确标记，不能靠删除网页元素或清除浏览器缓存冒充平台数据删除。

## 能力与覆盖范围

| 请求范围 | scope | 当前实现 | 不能保证的部分 |
| --- | --- | --- | --- |
| 永久删除笔记 | `notes` | 从创作服务平台“笔记管理”的“全部”页盘点；定位笔记 ID，打开“删除后将无法恢复”确认框，读取平台删除回执 | 管理页无权限、未加载完或无法识别 ID 时停止；未确认的页面空白不能视为零笔记 |
| 删除主动发表的评论 | `sent_comments` | 默认从当前网页“评论和@”读取被回复的原评论，核验原评论作者为当前账号、去重后加入计划；也可按已知笔记补充查找 | 未收到回复、通知已清除或关联笔记无法访问的评论可能遗漏；通知页面不等于账号全部已发评论历史 |
| 取消收藏 | `favorites` | 翻页读取个人主页“收藏”中的笔记，核实已收藏后逐条取消 | 不删除收藏夹本身；收藏列表不可读、达到上限、未包含的收藏夹内容不能算已清空 |
| 取消点赞 | `likes` | 翻页读取个人主页“点赞”，核实已点赞后逐条取消 | 无法完整读取点赞列表时不能保证覆盖全部；可补充已知笔记 ID |
| 删除收到的评论 | `received_comments` | 从通知网页读取自己笔记收到的评论候选，执行前再次核验笔记所有权 | 别人在其他笔记下回复自己，原评论仍归别人所有；网页通知删除入口未实现，不能删除别人笔记下的他人评论 |
| 清空并退出群聊 | `groups` | 盘点网页加载到的群会话数量，返回 `manual_required` | **未实现自动清空、删除或退群**；当前网页无已验证入口，请在手机 App 操作 |
| 取消关注 | `following` | 读取关注总数；对提供的用户 ID 核验“已关注/互相关注”后取消 | 网页没有完整关注名单；**不能自动发现全部关注账号**，需提供完整名单或在 App 操作 |
| 清空并删除私信会话 | `messages` | 盘点网页加载到的私信会话数量，返回 `manual_required` | **未实现自动删除或清空**；手机端删除也不等于撤回对方已经收到的消息 |

`count: -1` 表示未知。`complete: false` 表示盘点覆盖不完整或需要手机端处理；即使可见候选都成功，也不能据此声称账号已完全清空。范围内没有返回目标不等于没有历史数据。

## MCP 工具

### 1. `prepare_account_cleanup`

仅盘点和保存私有计划，不点击最终删除按钮。

```json
{"scopes":["notes","sent_comments","favorites","likes","received_comments","groups","following","messages"],"max_items":200}
```

`scopes` 可省略，默认盘点八类。`max_items` 默认 200，最大 1000；达到上限返回不完整状态。可用 `targets` 补充已知目标，但执行前仍会核实当前账号、目标和权限：

```json
{"scopes":["sent_comments"],"targets":[{"scope":"sent_comments","feed_id":"NOTE_ID","comment_id":"COMMENT_ID","xsec_token":"ACCESS_TOKEN"}]}
```

取消关注的目标填 `user_id`，而不是笔记 ID；访问令牌可通过正常检索结果取得。请只处理账号所有者明确授权的范围。

返回 `plan_id`、账号 ID、分类覆盖情况和目标状态。公开回传的计划不包含访问令牌和私信正文。私有计划包含执行所需数据，默认与登录文件同级的 `state/account-cleanup` 保存，也可通过 `XHS_CLEANUP_STATE_DIR` 配置。

评论清单默认直接来自当前通知页面，不扫描本地运营历史。`sent_comments` 使用 `commentInfo.targetComment` 的原评论 ID 和作者；不能把 `commentInfo.id`（收到的新回复）当作自己发表的评论 ID。页面存在通知也不代表拥有删除对方原评论的权限。读取通知会清除该分区的未读标记。

`list_notifications` 同时返回正常可见的 `target_comment`（被回复的原评论，含 ID、内容和作者）。在评论回复通知里，`comment_id` 表示收到的新回复；在 `liked/comment`（“赞了你的评论”）通知里，它表示被点赞的评论，可作为额外候选，但仍须在原帖核验作者，不能仅凭通知文字执行删除。“赞和收藏”通知分区表示收到的互动，不是自己点过赞、收藏过的笔记清单；后者仍来自个人主页相应标签。

### 2. `discover_cleanup_comments`

```json
{"feed_id":"NOTE_ID","xsec_token":"ACCESS_TOKEN","received":false,"limit":200}
```

`received:false` 查找该笔记下当前账号发表的评论；`received:true` 查找当前账号自己的笔记下收到的评论。返回候选 ID，可加入准备请求的 `targets`。每次只覆盖一篇已知笔记及实际加载到的评论，不返回完整账号历史证明。返回目标不含访问令牌，准备执行时使用该笔记已有令牌。

### 3. `execute_account_cleanup`

```json
{"plan_id":"PLAN_ID_FROM_PREPARE","confirm":true}
```

账号所有者明确授权后执行。一次最多处理一个候选，上一项结果明确后可以立即处理下一项，不设置固定的本地等待时间。旧版本保存的 `next_allowed_at` 不再作为执行限制，也不再返回 `waiting_interval`。这不取消平台自身的访问限制：安全验证、登录异常、平台限流及未确认结果仍会停止执行。同一账号的本地回执跨计划去重，`unknown` 或 `rejected` 会阻止继续自动执行。

共享同一私有状态目录的进程通过 `execution.lock` 防止并发执行。进程异常退出后不会自动抢占遗留锁；确认所有清理进程已停止、核验已有回执后，再由操作者处理遗留锁。不要删除执行回执来解除不确定结果。

删除笔记和评论通常不可恢复。取消收藏、取消赞、取消关注均是设置为未互动状态，不盲点切换按钮；已有清理回执或已处于目标状态时不会重复点击。

评论删除确认按钮会先核验布局稳定，再执行单次点击，避免弹窗动画导致点击位置过时。该检查有超时上限，不是两次清理之间的固定间隔；未稳定时不提交删除。

| 状态 | 含义 |
| --- | --- |
| `pending` | 尚未执行的候选；不代表已核验可删除 |
| `confirmed` | 本次目标的请求获得平台业务成功回执 |
| `already_clear` | 执行前确认已处于清理后的状态，本次没有变更 |
| `not_sent` | 未发送，定位、权限、登录或验证检查未通过 |
| `rejected` | 平台拒绝操作；HTTP 200 不等于成功 |
| `unknown` | 可能已发送但结果未确认，不自动重发 |
| `manual_required` | 当前网页不支持或找不到经过核实的入口 |

执行过程中出现安全验证、登录失效、访问频繁或账号切换时停止。验证交给用户手动完成，不绕过。平台受限不会因为“已登录”而自动恢复。重新验证后重新盘点；对于已发送但结果不确定的回执，必须先人工核验，不能删除回执以强行重发。

计划的 `finished` 只有在覆盖完整且所有目标均已确认清理时才返回；`incomplete`、`blocked` 和单条成功均不代表全部清空。

### 4. `get_account_cleanup_status`

```json
{"plan_id":"PLAN_ID_FROM_PREPARE"}
```

仅读取私有回执，不打开平台网页。应保存并使用返回的 `plan_id`，断线后先查状态，不能重新盲目提交删除。

## HTTP 接口

| 方法 | 路径 | 对应工具 |
| --- | --- | --- |
| POST | `/api/v1/cleanup/prepare` | `prepare_account_cleanup` |
| POST | `/api/v1/cleanup/execute` | `execute_account_cleanup` |
| GET | `/api/v1/cleanup/status?plan_id=PLAN_ID` | `get_account_cleanup_status` |
| POST | `/api/v1/cleanup/comments/discover` | `discover_cleanup_comments` |

参数与 MCP 相同，鉴权沿用现有配置。HTTP 响应外层的 `success` 只表示请求已处理；实际完成情况必须查看 `data.status`、`data.coverage` 和目标回执。

## 本地验证

```sh
go test ./...
go test -tags browser ./xiaohongshu -run TestCleanupBrowserFixtures
```

浏览器回归测试仅操作本地合成页面，覆盖目标 ID 匹配、平台失败、未知回执、受限页面和误判零条的情形，不创建或删除真实账号内容。
