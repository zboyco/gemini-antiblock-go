# 部署指南

本文档详细说明如何将 Gemini Antiblock Go 项目打包为 Docker 镜像并发布到 GitHub Container Registry (ghcr.io)。

## 快速开始

### 1. 使用预构建镜像

```bash
# 拉取最新镜像
docker pull ghcr.io/davidasx/gemini-antiblock-go:latest

# 运行容器
docker run -d \
  --name gemini-antiblock \
  -p 8080:8080 \
  -e DEBUG_MODE=true \
  ghcr.io/davidasx/gemini-antiblock-go:latest
```

### 2. 本地构建

```bash
# 克隆仓库
git clone https://github.com/Davidasx/gemini-antiblock-go.git
cd gemini-antiblock-go

# 构建镜像
docker build -t gemini-antiblock-go .

# 运行容器
docker run -d \
  --name gemini-antiblock \
  -p 8080:8080 \
  gemini-antiblock-go
```

## 自动化部署

### GitHub Actions 工作流

项目已配置 GitHub Actions 自动化 CI/CD 流程：

**文件位置**: `.github/workflows/docker-publish.yml`

**触发条件**:

- 推送到 `master` 或 `main` 分支
- 创建版本标签 (如 `v1.0.0`)
- Pull Request 到主分支

**构建特性**:

- 多架构支持 (`linux/amd64`, `linux/arm64`)
- 自动标签生成
- 缓存优化
- 构建证明生成

### 手动触发部署

如果需要手动触发部署：

1. 在 GitHub 仓库页面进入 "Actions" 标签
2. 选择 "Build and Push Docker Image" 工作流
3. 点击 "Run workflow"
4. 选择分支并运行

## 版本管理

### 创建版本发布

```bash
# 创建并推送标签
git tag v1.0.0
git push origin v1.0.0

# 或创建带注释的标签
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

### 镜像标签策略

- `latest`: 最新的主分支构建
- `v1.0.0`: 特定版本标签
- `main`/`master`: 对应分支的最新构建

## 生产部署

### 使用 Docker Compose

```yaml
version: "3.8"
services:
  gemini-antiblock:
    image: ghcr.io/davidasx/gemini-antiblock-go:v1.0.0
    ports:
      - "8080:8080"
    environment:
      - UPSTREAM_URL_BASE=https://generativelanguage.googleapis.com
      - MAX_CONSECUTIVE_RETRIES=100
      - DEBUG_MODE=false
      - RETRY_DELAY_MS=750
      - SWALLOW_THOUGHTS_AFTER_RETRY=true
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 256M
          cpus: "0.5"
```

### Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gemini-antiblock
spec:
  replicas: 2
  selector:
    matchLabels:
      app: gemini-antiblock
  template:
    metadata:
      labels:
        app: gemini-antiblock
    spec:
      containers:
        - name: gemini-antiblock
          image: ghcr.io/davidasx/gemini-antiblock-go:v1.0.0
          ports:
            - containerPort: 8080
          env:
            - name: DEBUG_MODE
              value: "false"
            - name: MAX_CONSECUTIVE_RETRIES
              value: "100"
          resources:
            limits:
              memory: "256Mi"
              cpu: "500m"
            requests:
              memory: "128Mi"
              cpu: "250m"
          livenessProbe:
            httpGet:
              path: /
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: gemini-antiblock-service
spec:
  selector:
    app: gemini-antiblock
  ports:
    - port: 80
      targetPort: 8080
  type: LoadBalancer
```

## 配置选项

### 环境变量

| 变量名                         | 默认值                                      | 描述                |
| ------------------------------ | ------------------------------------------- | ------------------- |
| `PORT`                         | `8080`                                      | 服务监听端口        |
| `UPSTREAM_URL_BASE`            | `https://generativelanguage.googleapis.com` | Gemini API 基础 URL |
| `MAX_CONSECUTIVE_RETRIES`      | `100`                                       | 最大重试次数        |
| `DEBUG_MODE`                   | `true`                                      | 调试模式            |
| `RETRY_DELAY_MS`               | `750`                                       | 重试延迟（毫秒）    |
| `SWALLOW_THOUGHTS_AFTER_RETRY` | `true`                                      | 重试后过滤思考内容  |

### 资源建议

**最小配置**:

- 内存: 64MB
- CPU: 0.1 核心

**推荐配置**:

- 内存: 256MB
- CPU: 0.5 核心

**高负载配置**:

- 内存: 512MB
- CPU: 1.0 核心

## 监控和日志

### 日志查看

```bash
# 查看容器日志
docker logs gemini-antiblock

# 实时跟踪日志
docker logs -f gemini-antiblock

# 查看最近100行日志
docker logs --tail 100 gemini-antiblock
```

### 健康检查

容器基于Alpine Linux构建，支持内置健康检查：

```bash
# 查看健康状态
docker ps

# 查看健康检查详情
docker inspect --format='{{json .State.Health}}' gemini-antiblock

# 手动健康检查
docker exec gemini-antiblock curl -f http://localhost:8080/
```

**Kubernetes健康检查配置**:
```yaml
livenessProbe:
  httpGet:
    path: /
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 监控指标

可以通过以下方式监控应用状态：

```bash
# 检查容器状态
docker ps

# 查看资源使用情况
docker stats gemini-antiblock

# 检查端口连通性
curl -f http://localhost:8080/ || echo "Service is down"
```

## 故障排除

### 常见问题

1. **容器启动失败**

   ```bash
   docker logs gemini-antiblock
   ```

2. **端口冲突**

   ```bash
   # 使用不同端口
   docker run -p 8081:8080 ghcr.io/davidasx/gemini-antiblock-go:latest
   ```

3. **权限问题**

   ```bash
   # 检查 Docker 权限
   sudo usermod -aG docker $USER
   ```

4. **镜像拉取失败**
   ```bash
   # 登录 GitHub Container Registry
   echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
   ```

### 调试模式

启用详细日志进行调试：

```bash
docker run -d \
  --name gemini-antiblock \
  -p 8080:8080 \
  -e DEBUG_MODE=true \
  ghcr.io/davidasx/gemini-antiblock-go:latest
```

## 安全考虑

1. **使用特定版本标签**: 避免使用 `latest` 标签
2. **资源限制**: 设置内存和 CPU 限制
3. **网络隔离**: 使用 Docker 网络或 Kubernetes 网络策略
4. **定期更新**: 及时更新到最新版本
5. **环境变量安全**: 避免在日志中暴露敏感信息

## 支持

如有问题，请：

1. 查看 [GitHub Issues](https://github.com/Davidasx/gemini-antiblock-go/issues)
2. 提交新的 Issue
3. 查看项目文档和 README
