# Changelog

## [0.0.7] - 2025-01-23

- 修复: cors 检查失败

## [0.0.6] - 2025-01-20

- 添加: 支持使用 http HEAD 检查 Linker 是否可用

## [0.0.5] - 2025-01-19

- 修复: 当 device 删除时关闭对应的 endpoint 连接, 使用 app.Store 响应删除事件提升性能

## [0.0.4] - 2025-01-19

- 修复: 当 device 删除时关闭对应的 endpoint 连接
- 修复: endpoint 未绑定 device 时不允许连接

## [0.0.3] - 2025-01-13

- 修复 WHIP Endpoint 的 Body 可 Reread 和 proxy http.transferWriter 冲突, 通过禁用 Reread 解决
- 修复 WHIP Endpoint 因为没去除路由前缀导致的路由错误
- 变更 WHIP Endpoint 由 `/api/salt-link` 变更为 `/api/salt-whip/` ,以便后续支持 WHIP 重连

## [0.0.2] - 2025-01-12

- WHIP GET 请求返回设备是否在线, 有助于排查问题
- 将表字段设置为 system, 避免被错误删除

## [0.0.1] - 2025-01-12

- 初版完成
