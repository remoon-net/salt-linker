# Changelog

## [0.0.3] - 2025-01-13

- 修复 WHIP Endpoint 的 Body 可 Reread 和 proxy http.transferWriter 冲突, 通过禁用 Reread 解决
- 修复 WHIP Endpoint 因为没去除路由前缀导致的路由错误
- 变更 WHIP Endpoint 由 `/api/salt-link` 变更为 `/api/salt-whip/` ,以便后续支持 WHIP 重连

## [0.0.2] - 2025-01-12

- WHIP GET 请求返回设备是否在线, 有助于排查问题
- 将表字段设置为 system, 避免被错误删除

## [0.0.1] - 2025-01-12

- 初版完成
