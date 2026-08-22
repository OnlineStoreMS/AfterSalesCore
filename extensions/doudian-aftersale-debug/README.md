# OSMS 抖店售后【诊断】插件

临时排障版本，目录独立于正式插件 `extensions/doudian-aftersale`。

## 做什么

- 只支持手动「打开抖店工作台」和「立即同步」
- **没有心跳、没有 5 分钟自动同步**
- 每次执行会记录步骤、错误、页面探测（URL / iframe / 菜单 / 页签 / 卡片）
- 绑定后把日志 POST 到 `/api/v1/plugin/debug-log`
- 云端写到容器临时目录（默认 `/tmp/aftersales-plugin-debug`），**重启 API 即清空**，最多保留 80 份

## 怎么用

1. 重启 AfterSalesCore API（让诊断接口生效）
2. Chrome `chrome://extensions` → 加载已解压的扩展，选本目录
3. **先关掉正式版插件**，避免两个插件抢同一个抖店标签
4. 用店铺绑定码绑定（与正式版相同）
5. 点「打开抖店工作台」，再点「立即同步」
6. 在售后管理侧栏 **插件诊断日志** 查看过程

不要把本插件当日常采集用。排障结束后改回正式版 `doudian-aftersale`。
