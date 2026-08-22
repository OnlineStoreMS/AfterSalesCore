# OSMS 抖店售后工作台插件

Chrome / Edge 扩展：在已登录的抖店售后工作台采集「快捷筛选」卡片数字与对应售后单，上报到 AfterSalesCore 售后管理。

一个扩展实例绑定一个店铺。

## 安装

1. 打开 Chrome `chrome://extensions`
2. 打开右上角「开发者模式」
3. 「加载已解压的扩展程序」，选择本目录 `extensions/doudian-aftersale`

## 绑定

1. 打开售后管理 → **店铺管理** → 添加店铺（平台选抖店）
2. 复制绑定码
3. 点击扩展图标，填写售后管理地址和绑定码后点「绑定」
   - 生产：`https://osms.zfcycle.com/apps/aftersales`
   - 本地：`http://localhost:5176` 或 `http://localhost:8093`

## 同步

打开 [抖店售后工作台](https://fxg.jinritemai.com/ffa/merchant-aftersale-workbench/aftersale/list) 并保持登录。扩展会自动采集：

- 快捷筛选全部分组卡片及数字
- `count > 0` 的卡片下全部售后单（含翻页）

也可在弹窗点「立即同步」。同步结果在售后管理对应店铺的工作台查看。

## 重置

在售后管理店铺列表「重置绑定」后，旧密钥失效，需要在扩展里重新填写新绑定码。
