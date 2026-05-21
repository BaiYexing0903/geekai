# Jenkins Harbor 凭据配置指南

## 一、配置 Harbor 凭据（一次性配置）

### 步骤 1：进入凭据管理
1. 登录 Jenkins
2. 点击 **系统管理**（左侧菜单底部）
3. 点击 **凭据** -> **系统** -> **全局凭据 (未受限)**
4. 点击 **添加凭据** 按钮

### 步骤 2：填写 Harbor 账号信息

| 字段 | 值/说明 |
|------|---------|
| **种类** | 用户名和密码 |
| **范围** | 全局 (Jenkins, 节点，所有子项和所有用户) |
| **用户名** | Harbor 登录账号（如：`admin` 或你的 Harbor 账号） |
| **密码** | Harbor 登录密码 |
| **ID** | `harbor-credentials` ⚠️ **必须一致** |
| **描述** | Harbor 镜像仓库登录凭据 |

### 步骤 3：保存
点击 **确定** 按钮保存凭据。

---

## 二、Jenkinsfile 中的凭据使用

当前 Jenkinsfile.backend 已经配置好凭据使用：

```groovy
stage('Push Image') {
    when {
        expression { return params.PUSH }
    }
    steps {
        script {
            // 使用 Harbor 凭据登录
            withCredentials([usernamePassword(
                credentialsId: 'harbor-credentials',  // ← 这里引用上面配置的凭据 ID
                usernameVariable: 'HARBOR_USER',
                passwordVariable: 'HARBOR_PASS'
            )]) {
                sh """
                echo ${HARBOR_PASS} | docker login ${HARBOR_HOST} -u ${HARBOR_USER} --password-stdin
                """
            }

            echo "推送镜像到 Harbor..."
            sh """
            docker push ${HARBOR_HOST}/${HARBOR_PROJECT}/${IMAGE_NAME}:${VERSION}
            docker push ${HARBOR_HOST}/${HARBOR_PROJECT}/${IMAGE_NAME}:latest
            """
        }
    }
}
```

### 说明
- `credentialsId: 'harbor-credentials'`：引用步骤一中配置的凭据 ID
- `HARBOR_USER`：凭据中的用户名（自动注入）
- `HARBOR_PASS`：凭据中的密码（自动注入，会被 Jenkins 脱敏）

---

## 三、验证配置

### 3.1 测试凭据是否生效

1. 进入 Jenkins Job 页面
2. 点击 **立即构建**
3. 点击构建编号（如 `#1`）
4. 点击 **控制台输出**

查看日志中是否出现：
```
[Pipeline] withCredentials
...
+ echo **** | docker login 64.32.12.251:28077 -u **** --password-stdin
Login Succeeded
```

如果看到 `Login Succeeded` 或 `Succeeded`，说明凭据配置成功。

### 3.2 常见问题排查

| 问题 | 解决方案 |
|------|----------|
| `credentialsId: 'harbor-credentials' could not be found` | 检查凭据 ID 是否完全一致（区分大小写） |
| `unauthorized` 或 `Login failed` | 检查 Harbor 账号密码是否正确 |
| `denied` | 检查 Harbor 账号是否有对应项目的写入权限 |
| `connection refused` | 检查 Jenkins 服务器能否访问 Harbor 地址 |

---

## 四、前端 Jenkinsfile 配置

前端 Jenkinsfile 使用相同的凭据配置：

```groovy
// deploy/jenkins/Jenkinsfile.frontend
withCredentials([usernamePassword(
    credentialsId: 'harbor-credentials',  // 相同的凭据 ID
    usernameVariable: 'HARBOR_USER',
    passwordVariable: 'HARBOR_PASS'
)]) {
    sh """
    echo ${HARBOR_PASS} | docker login ${HARBOR_HOST} -u ${HARBOR_USER} --password-stdin
    """
}
```

---

## 五、凭据管理最佳实践

### ✅ 推荐做法
1. **使用凭据 ID**：不要在 Jenkinsfile 中硬编码账号密码
2. **凭据复用**：前后端构建共用一个 Harbor 凭据
3. **权限最小化**：为 Jenkins 创建专用的 Harbor 账号，仅授予必要的仓库写入权限
4. **定期轮换**：定期更换 Harbor 密码

### ❌ 不推荐做法
1. 直接在 Jenkinsfile 中写密码
2. 使用个人 Harbor 账号（人员离职会导致构建失败）
3. 将凭据截图发到群里

---

## 六、凭据截图指引

### Jenkins 凭据配置界面

```
系统管理 → 凭据 → 系统 → 全局凭据 → 添加凭据

┌─────────────────────────────────────────────────┐
│ 添加凭据                                         │
├─────────────────────────────────────────────────┤
│ 种类：▼ 用户名和密码                             │
│ 范围：● 全局 (Jenkins, 节点，所有子项...)        │
│                                                 │
│ 用户名：[admin                                  ] │
│ 密码：   [●●●●●●●●●●●●                        ] │
│                                                 │
│ ID：     [harbor-credentials                     ] ← 重要！
│ 描述：   [Harbor 镜像仓库登录凭据                ] │
│                                                 │
│                    [ 取消 ]  [ 确定 ]            │
└─────────────────────────────────────────────────┘
```

---

## 七、快速检查清单

- [ ] Jenkins 已安装 Credentials Binding 插件
- [ ] 凭据已添加到全局凭据
- [ ] 凭据 ID 为 `harbor-credentials`
- [ ] Harbor 账号有对应项目的写入权限
- [ ] Jenkins 服务器可以访问 Harbor 地址
