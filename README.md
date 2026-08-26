# task255-exchangepeak · 核磁共振化学交换峰归属服务

化学研究者从多温度 NMR 峰簇中区分化学交换、杂质与仪器漂移，形成可引用的峰归属证据。系统以样品条件与温度序列为输入，校正内标偏移、关联峰随温度变化的轨迹，并依据合并/分裂判据评分化学交换候选；研究者可排除杂质峰、锁定内标并发布归属快照。

## 业务闭环

创建样品并设定温度序列 → 导入多温度谱图批次与峰 → 设定内标并校正化学位移 → 关联峰随温度轨迹 → 评分化学交换候选（合并/分裂）→ 裁决确认/否决并锁定内标 → 发布归属快照。

## 核心实体与状态

| 实体 | 状态 |
| --- | --- |
| 谱图批次 | receiving → pending_link → needs_review → published → sealed |
| 峰记录 | raw → corrected → impurity → duplicate → excluded |
| 交换候选 | generated → continuous → split_conflict → confirmed → rejected |
| 归属快照 | draft → published → superseded |

## 关键不变量

- 同一温度序列内温度值唯一（拒绝温度重复）。
- 谱峰化学位移单位统一为 ppm（拒绝 Hz 与 ppm 混用未转换）。
- 内标缺失时禁止校正与轨迹关联。
- 已封存（sealed）批次不可再修改。
- 归属快照冻结当时的样品、温度条件、内标版本与峰轨迹集合。

## 标准命令

```bash
export GO_BIN=$(command -v go)          # go1.26.3, GOTOOLCHAIN=local
CGO_ENABLED=0 GOTOOLCHAIN=local $GO_BIN build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local $GO_BIN vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local $GO_BIN test  ./...
$GO_BIN run ./cmd/task255-exchangepeak --smoke-test       # 离线端到端自检（含 DB 关闭重开恢复验证）

# 长驻服务
$GO_BIN run ./cmd/task255-exchangepeak --addr=:8080 --db=task255-exchangepeak.db
# 浏览器打开 http://localhost:8080 或调用 /api/... 接口
```

## API 入口（前缀 /api，共 34 个接口）

- 自检：`GET /api/health`、`GET /api/selfcheck`
- 样品：`POST /api/samples`、`GET /api/samples`、`GET /api/samples/{id}`、`DELETE /api/samples/{id}`、`GET /api/samples/{id}/temperatures`、`POST /api/samples/{id}/temperatures`
- 谱图批次：`POST /api/batches`、`GET /api/batches`、`GET /api/batches/{id}`、`POST /api/batches/{id}/state`
- 峰：`POST /api/batches/{id}/peaks`、`GET /api/batches/{id}/peaks`、`GET /api/batches/{id}/peaks/temp`、`POST /api/batches/{id}/peaks/{peakId}/impurity`、`POST /api/batches/{id}/peaks/{peakId}/exclude`
- 内标：`POST /api/batches/{id}/standards`、`POST /api/standards/{id}/points`、`POST /api/standards/{id}/lock`
- 分析流水线：`POST /api/batches/{id}/calibrate`、`POST /api/batches/{id}/associate`、`POST /api/batches/{id}/score`、`POST /api/batches/{id}/analyze`
- 轨迹：`GET /api/batches/{id}/tracks`、`GET /api/tracks/{id}`、`GET /api/tracks/{id}/members`
- 交换候选：`GET /api/batches/{id}/candidates`、`GET /api/candidates/{id}`、`POST /api/candidates/{id}/confirm`、`POST /api/candidates/{id}/reject`
- 归属快照：`POST /api/batches/{id}/snapshots`、`GET /api/batches/{id}/snapshots`、`GET /api/snapshots/{id}`
