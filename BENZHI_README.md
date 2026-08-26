基于 Go 实现的核磁共振化学交换峰归属后端服务项目，一款后端服务，在多温度谱峰上做内标校正化学位移、关联峰随温度轨迹并依据合并/分裂判据裁决化学交换归属、冻结可溯源的归属快照。

# BENZHI 评测说明

## 1. 项目类型

核磁共振化学交换峰归属后端服务（非 OA/工单/预约，非数据看板消费类应用）。提供 JSON 形态的 `/api` 接口，供化学研究者以程序化方式导入多温度谱图、校正内标、关联峰轨迹、裁决化学交换并冻结归属快照。

## 2. 标准命令

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/task255-exchangepeak --smoke-test
go run ./cmd/task255-exchangepeak --addr :8080 --db task255-exchangepeak.db
```

- `--addr`：HTTP 监听地址，默认 `:8080`
- `--db`：SQLite 数据库文件路径，默认 `task255-exchangepeak.db`
- `--smoke-test`：不常驻；跑完端到端场景后关闭并重新打开数据库，退出码 0 表示通过

## 3. 评测镜像

`Dockerfile` 与 `benzhi.Dockerfile` 内容完全一致；使用 Go 1.26.3 Bookworm builder 和 Alpine 3.20 runtime 的多阶段构建，产物为 `/app/task255-exchangepeak`。脚本第二个参数为目标平台。镜像不声明固定端口，服务监听地址由 `--addr` 指定。

```bash
./build_benzhi_docker.sh task255-exchangepeak:amd64 linux/amd64
docker run --rm task255-exchangepeak:amd64 --smoke-test

./build_benzhi_docker.sh task255-exchangepeak:arm64 linux/arm64
docker run --rm task255-exchangepeak:arm64 --smoke-test

docker run --rm -P task255-exchangepeak:amd64 --addr :8080 --db ./app.db
```

## 4. 冒烟自测契约（--smoke-test）

创建临时数据库 → 写入样品、多温度谱图批次、峰与内标 → 执行内标校正、峰温度轨迹关联、化学交换候选评分、裁决与归属快照冻结 → 关闭并重新打开数据库，校验实体计数与关键字段在重启后仍一致后退出 0。容器里只传 flag，不传二进制路径位置参数。

## 5. 核心 API（`/api` 前缀）

- 自检：`GET /api/health`、`GET /api/selfcheck`
- 样品：`POST /api/samples`、`GET /api/samples`、`GET /api/samples/{id}`、`DELETE /api/samples/{id}`、`GET /api/samples/{id}/temperatures`、`POST /api/samples/{id}/temperatures`
- 谱图批次：`POST /api/batches`、`GET /api/batches`、`GET /api/batches/{id}`、`POST /api/batches/{id}/state`
- 峰：`POST /api/batches/{id}/peaks`、`GET /api/batches/{id}/peaks`、`GET /api/batches/{id}/peaks/temp`、`POST /api/batches/{id}/peaks/{peakId}/impurity`、`POST /api/batches/{id}/peaks/{peakId}/exclude`
- 内标：`POST /api/batches/{id}/standards`、`POST /api/standards/{id}/points`、`POST /api/standards/{id}/lock`
- 分析流水线：`POST /api/batches/{id}/calibrate`、`POST /api/batches/{id}/associate`、`POST /api/batches/{id}/score`、`POST /api/batches/{id}/analyze`
- 轨迹：`GET /api/batches/{id}/tracks`、`GET /api/tracks/{id}`、`GET /api/tracks/{id}/members`
- 交换候选：`GET /api/batches/{id}/candidates`、`GET /api/candidates/{id}`、`POST /api/candidates/{id}/confirm`、`POST /api/candidates/{id}/reject`
- 归属快照：`POST /api/batches/{id}/snapshots`、`GET /api/batches/{id}/snapshots`、`GET /api/snapshots/{id}`

## 6. 环境与组件

- Go 1.26.3（GOTOOLCHAIN=local，CGO_ENABLED=0）
- SQLite 3.46.1（modernc.org/sqlite v1.52.0，CGO 无关）
