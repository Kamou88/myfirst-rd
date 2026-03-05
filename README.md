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

Recipe APIs:

- `GET /api/recipes` 获取配方列表
- `POST /api/recipes` 新增配方

`POST /api/recipes` body 示例：

```json
{
  "name": "铁板",
  "machineName": "电炉",
  "cycleSeconds": 3.2,
  "powerKW": 180,
  "inputs": [{ "name": "铁矿", "amount": 1 }],
  "outputs": [{ "name": "铁板", "amount": 1 }]
}
```

Device APIs:

- `GET /api/devices` 获取设备列表
- `POST /api/devices` 新增设备
- `PUT /api/devices/{id}` 更新设备
- `DELETE /api/devices/{id}` 删除设备

`POST /api/devices` body 示例：

```json
{
  "name": "电炉 Mk1",
  "efficiencyPercent": 100
}
```

Material APIs:

- `GET /api/materials` 获取材料列表
- `POST /api/materials` 新增材料
- `PUT /api/materials/{id}` 更新材料
- `DELETE /api/materials/{id}` 删除材料

`POST /api/materials` body 示例：

```json
{
  "name": "铁矿"
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
