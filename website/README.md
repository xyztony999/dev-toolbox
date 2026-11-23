# DevToolbox Website

**Languages:** 中文 · [English](./README.en.md)

官网静态站点，部署在 Cloudflare Workers（Static Assets）。

正式域名：https://dev-toolbox.tonyxyz.com  
备用：https://dev-toolbox.zhangxinyi1230.workers.dev

站点支持：

- 浅色 / 深色 / 跟随系统（右上角主题按钮循环切换）
- 中文 / English（右上角语言按钮）

## 本地预览

```bash
cd website
npm install
npm run dev
```

## 部署

需已登录 Wrangler：

```bash
cd website
npm run deploy
```
