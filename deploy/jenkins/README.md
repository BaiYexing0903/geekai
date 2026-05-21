# GateHub Jenkins 配置指南

## 前置准备

### 1. 安装必要插件

进入 **系统管理** -> **插件管理** -> **可选插件**，安装以下插件：

| 插件名称 | 用途 |
|---------|------|
| Docker Pipeline | Docker 流水线支持 |
| Credentials Binding | 凭据绑定 |
| GitLab | GitLab 集成 |

安装完成后重启 Jenkins。

---

### 2. 配置全局凭据

进入 **系统管理** -> **凭据** -> **系统** -> **全局凭据** -> **添加凭据**：

#### 3.1 Harbor 凭据
- **凭据类型**: 用户名和密码
- **用户名**: Harbor 账号
- **密码**: Harbor 密码
- **ID**: `harbor-credentials` (必须一致)
- **描述**: Harbor 镜像仓库凭据

#### 3.2 GitLab 凭据
- **凭据类型**: SSH 用户名和私钥
- **用户名**: `git`
- **私钥**: 选择 **直接输入**，粘贴你的 SSH 私钥内容
- **ID**: `gitlab-credentials` (必须一致)
- **描述**: GitLab 代码仓库凭据

---

## 创建 Jenkins Job

### 方式一：Multibranch Pipeline（推荐）

适合自动构建多个分支（main、release、develop 等）。

#### 步骤

1. **新建任务** -> 输入任务名称（如 `GateHub-Backend`）-> 选择 **Multibranch Pipeline** -> 确定

2. **Branch Sources** 配置：
   ```
   Source: GitLab
   Project Repository: http://your-gitlab.com/ai/GateHub.git
   Credentials: gitlab-credentials
   ```

3. **Behaviors** 配置（默认即可）：
   - 发现分支：排除带有 MR 的分支

4. **Build Configuration** 配置：
   ```
   Mode: by Jenkinsfile
   Script Path: deploy/jenkins/Jenkinsfile.backend
   ```

5. **Scan Multibranch Pipeline Triggers**：
   - 勾选 **定期扫描 SCM**
   - 计划：`H/5 * * * *` (每 5 分钟扫描一次)

6. 点击 **保存**，Jenkins 会自动扫描分支并触发构建

---

### 方式二：Pipeline（简单快捷）

适合单分支手动触发构建。

#### 步骤

1. **新建任务** -> 输入任务名称（如 `GateHub-Backend-Manual`）-> 选择 **Pipeline** -> 确定

2. **General** 配置：
   - 勾选 **GitHub project** 或 **GitLab project**（可选）
   - 勾选 **丢弃旧的构建**
     - 保持构建的天数：`30`
     - 保持构建的最大个数：`30`

3. **构建触发器** 配置：
   ```
   勾选：轮询 SCM
   计划：H/5 * * * *
   ```

4. **Pipeline** 配置：
   ```
   Definition: Pipeline script from SCM
   SCM: Git
   Repository URL: git@gitlab:ai/GateHub.git
   Credentials: gitlab-credentials
   Branch Specifier: */release
   Script Path: deploy/jenkins/Jenkinsfile.backend
   ```

5. 点击 **保存**

---

## Jenkins 参数说明

### 后端构建参数 (Jenkinsfile.backend)

| 参数名 | 类型 | 默认值 | 说明 |
|-------|------|--------|------|
| `VERSION_OVERRIDE` | 字符串 | (空) | 覆盖版本号，留空则使用构建编号 |
| `GIT_BRANCH` | 字符串 | `release` | Git 分支名 |
| `DOCKERFILE` | 字符串 | `Dockerfile` | Dockerfile 路径 |
| `PUSH` | 布尔值 | `true` | 是否推送到 Harbor |
| `DEPLOY_LOCAL` | 布尔值 | `false` | 构建后是否部署到本地环境 |

### 前端构建参数 (Jenkinsfile.frontend)

| 参数名 | 类型 | 默认值 | 说明 |
|-------|------|--------|------|
| `VERSION_OVERRIDE` | 字符串 | (空) | 覆盖版本号，留空则使用构建编号 |
| `GIT_BRANCH` | 字符串 | `release` | Git 分支名 |
| `DOCKERFILE` | 字符串 | `deploy/frontend/Dockerfile` | Dockerfile 路径 |
| `PUSH` | 布尔值 | `true` | 是否推送到 Harbor |
| `DEPLOY_LOCAL` | 布尔值 | `false` | 构建后是否部署到本地环境 |

---

## 使用示例

### 示例 1: 日常构建（自动递增版本）

1. 点击 **立即构建**
2. 参数保持默认（`VERSION_OVERRIDE` 留空）
3. 构建完成后，镜像版本为 `build-42`（假设是第 42 次构建）

### 示例 2: 指定版本发布

1. 点击 **立即构建**
2. 在 `VERSION_OVERRIDE` 中输入：`v1.0.0`
3. 构建完成后，镜像版本为 `v1.0.0`

### 示例 3: 仅构建不推送

1. 点击 **立即构建**
2. 将 `PUSH` 取消勾选
3. 构建完成后，镜像仅在本地，不会推送到 Harbor

### 示例 4: 构建并部署到本地环境

1. 点击 **立即构建**
2. 勾选 `DEPLOY_LOCAL`
3. 构建完成后自动部署到测试环境

---

## 构建日志查看

构建完成后：

1. 点击 Job 名称进入详情页
2. 点击左侧 **构建历史** 中的构建编号（如 `#42`）
3. 点击 **控制台输出** 查看完整日志

关键日志节点：
```
[INFO] 拉取代码：分支=release
[INFO] 使用 Jenkins 构建编号：build-42
[INFO] 构建后端镜像：64.32.12.251:28077/gatehub/gatehub-server:build-42
[INFO] 推送镜像到 Harbor...
[SUCCESS] 构建成功！镜像：64.32.12.251:28077/gatehub/gatehub-server:build-42
```

---

## 常见问题

### Q1: 构建失败 "Docker 未找到"
**解决**: 确保 Jenkins 用户有 Docker 执行权限：
```bash
sudo usermod -aG docker jenkins
sudo systemctl restart jenkins
```

### Q2: Git 克隆失败 "Permission denied"
**解决**: 检查 SSH key 配置：
1. 确认 `gitlab-credentials` 凭据配置正确
2. 确认 GitLab 仓库已添加该 SSH key 为 Deploy Key 或用户有访问权限

### Q3: Harbor 推送失败 "unauthorized"
**解决**: 检查 Harbor 凭据：
1. 进入 **凭据管理** -> 找到 `harbor-credentials`
2. 确认用户名密码正确
3. 确认该账号有对应 Harbor 项目的写入权限

### Q4: 前端构建失败 "bun: command not found"
**解决**: Dockerfile 中会自动安装 Bun，确保构建上下文正确：
- Script Path 应指向项目根目录
- Dockerfile 路径应相对于项目根目录

---

## 高级配置

### 定时构建

进入 Job 配置 -> **构建触发器**：
```
勾选：定时构建
计划：0 2 * * *   # 每天凌晨 2 点自动构建
```

### 构建通知（邮件/钉钉/企业微信）

进入 Job 配置 -> **构建后操作步骤**：
```
勾选：Editable Email Notification
收件人: team@example.com
触发条件：Failure
```

### 并发构建限制

进入 Job 配置 -> **General**：
```
勾选：丢弃旧的构建
保持构建的最大个数: 10
```

---

## 监控与告警

### 查看构建趋势

1. 进入 Job 主页
2. 查看 **构建历史** 和 **构建时间趋势图**

### 配置构建超时

在 Jenkinsfile 中已配置：
```groovy
options {
    timeout(time: 30, unit: 'MINUTES')  // 30 分钟超时
}
```

---

## 最佳实践

1. **分支策略**:
   - `release` 分支：生产发布，自动推送 Harbor
   - `develop` 分支：开发测试，可配置不推送

2. **版本标记**:
   - 日常构建：使用默认的 `build-{编号}`
   - 重要发布：使用 `VERSION_OVERRIDE` 指定语义化版本（如 `v1.0.0`）

3. **构建频率**:
   - 开发期间：每次提交自动构建
   - 稳定期间：每天定时构建或手动触发

4. **资源清理**:
   - 配置 **丢弃旧的构建** 避免磁盘爆满
   - 定期清理 Harbor 旧镜像
