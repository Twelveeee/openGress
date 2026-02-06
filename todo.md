# OpenIngress RTS TODO（MVP 可玩优先）

## 当前状态
- [x] 机制与接口文档已存在：`openGress/WIKI/*`
- [x] 后端 HTTP/WS 主流程已搭好（CONNECT/移动/视野/部署/充能/连线/攻击）
- [x] 前端已有地图与主要弹窗骨架（当前仍为 mock 数据）

## 近期优先（MVP 可玩闭环）
### 后端
- [ ] Hack 机制落地：`HACK_PORTAL` + `HACK_RESULT` + 自动 Hack 掉落与冷却 + AP 奖励
- [ ] 物资获取与容量校验：所有掉落/奖励/管理发放统一用 `gameplay.CanAddItem/CanAddKey`
- [ ] Portal 归零清理：中立后移除相关 Links/Fields，并向视野推送 `MAP_UPDATE`
- [ ] 等级系统：AP → Level、`MaxXM`/容量同步、等级限制检查
- [ ] 新手可操作性：注册时给起始物品或提供 admin/CLI 发放

### 前端
- [ ] 接入后端鉴权与玩家状态（注册/登录/refresh + 玩家信息）
- [ ] WS 连接与地图实时数据（PLAYER_STATE/NEARBY/MAP_UPDATE/MAP_TICK）
- [ ] Portal 交互链路改为真实请求（deploy/mod/attack/link）
- [ ] 背包读取与道具使用/回收接入 HTTP
- [ ] UI 三项修正：攻击弹窗贴底、打开弹窗时缩小 Global Log、AP 槽移到 XM 槽下方

## 规则完善（次优先）
- [ ] Link 距离按 8-slot 配置计算（对齐 `openGress/WIKI/portal.md`）
- [ ] Mod 与 Resonator 限制：每人最多 2 个 Mod、同类限制、部署等级配额
- [ ] Portal 衰减 + XM 自然恢复/充能规则
- [ ] Mod 效果进战斗/入侵（护盾减伤、炮塔等）
- [ ] 领地/日志/排行榜规则对齐 WIKI（记录上限、排序）

## 数据与运维
- [ ] 持久化落地：GORM 模型与快照导入/导出对齐
- [ ] 管理 CLI 补齐（导入地图、发公告、批量发物资）
- [ ] 运行监控与日志轮转流程完善

## 文档与测试
- [ ] 更新 `openGress/WIKI/api.md` 状态标注与字段说明
- [ ] 同步 `openGress/README.md` 与 `openGress/CLAUDE.md` 的当前实现状态
- [ ] Go 测试补齐：hack/掉落/neutralize 清理/容量校验
