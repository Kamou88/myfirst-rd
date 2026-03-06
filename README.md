# go-backend

## Run

```bash
go run .
```

Backend default address:

- http://localhost:8080

API endpoints:

- http://localhost:8080/api/health
- http://localhost:8080/api/recipes
- http://localhost:8080/api/devices
- http://localhost:8080/api/materials
- http://localhost:8080/api/device-types

Recipe APIs:

- `GET /api/recipes` 获取配方列表
- `POST /api/recipes` 新增配方（会按设备种类下的所有型号自动生成多条）
- `PUT /api/recipes/{id}` 更新配方模板（会按设备种类下所有型号重新生成整组配方）
- `PUT /api/recipes/{id}/booster` 更新单条配方增产剂（mk1/mk2/mk3）
- `DELETE /api/recipes/{id}` 删除单条配方

`POST /api/recipes` body 示例：

```json
{
  "name": "铁板",
  "machineName": "熔炉",
  "cycleSeconds": 3.2,
  "powerKW": 0,
  "canSpeedup": true,
  "canBoost": true,
  "inputs": [{ "name": "铁矿", "amount": 1 }],
  "outputs": [{ "name": "铁板", "amount": 1 }]
}
```

说明：系统会按设备型号始终生成一条“无效果”配方；当 `canSpeedup` 和/或 `canBoost` 为 `true` 时，再额外生成对应效果配方。

说明：`PUT /api/recipes/{id}` 与新增使用相同 body，返回值同样是“重建后整组配方数组”。

`PUT /api/recipes/{id}/booster` body 示例：

```json
{
  "boosterTier": "mk3"
}
```

当前效果系数：

- `mk1`：可加速（周期 -25%）、可增产（输出 +12.5%）、两者都会使功耗 +30%
- `mk2`：可加速（每分钟产量 +50%，即周期变为 2/3）、可增产（输出 +20%）、两者都会使功耗 +70%
- `mk3`：可加速（周期 -50%）、可增产（输出 +25%）、两者都会使功耗 +150%

Device APIs:

- `GET /api/devices` 获取设备列表
- `POST /api/devices` 新增设备
- `PUT /api/devices/{id}` 更新设备
- `DELETE /api/devices/{id}` 删除设备

`POST /api/devices` body 示例：

```json
{
  "deviceType": "熔炉",
  "name": "熔炉 Mk1",
  "efficiencyPercent": 100,
  "powerKW": 180
}
```

说明：`deviceType` 需要先在 `device-types` 中创建。

Material APIs:

- `GET /api/materials` 获取材料列表
- `POST /api/materials` 新增材料
- `PUT /api/materials/{id}` 更新材料
- `DELETE /api/materials/{id}` 删除材料

`POST /api/materials` body 示例：

```json
{
  "name": "铁矿",
  "isCraftable": false
}
```

Device Type APIs:

- `GET /api/device-types` 获取设备种类列表
- `POST /api/device-types` 新增设备种类
- `PUT /api/device-types/{id}` 更新设备种类
- `DELETE /api/device-types/{id}` 删除设备种类

`POST /api/device-types` body 示例：

```json
{
  "name": "熔炉"
}
```

## Environment variables

- `PORT` (default: `8080`)
- `FRONTEND_ORIGIN` (default: `http://localhost:5173`)
- `SQLITE_PATH` (default: `recipes.db`)

## Data persistence

- 配方数据已改为 SQLite 持久化存储
- 默认数据库文件：`go-backend/recipes.db`
- 重启后端后数据仍会保留
