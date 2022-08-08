# Crane Dashboard

### Api Docs

https://www.postman.com/gocrane/workspace/public/collection/14940923-8559deb0-9af9-4ace-bc64-14da3dd0c8f8?ctx=documentation


### 开发

开发前，请参考[指南](https://tdesign.tencent.com/starter/docs/react/develop)。

#### 配置Proxy

1. 修改 `vite.config.js`
```json
    proxy: {
      '/api': {
        // 用于开发环境下的转发请求
        // 更多请参考：https://vitejs.dev/config/#server-proxy
        // Set to your craned address
        target: 'http://10.100.100.112:31356',
        changeOrigin: true,
      },
      '/grafana': {
        // Set to your craned address
        target: 'http://10.100.100.112:31356',
        changeOrigin: true,
      },
    },
  },
```

2. 增加集群，填入`http://localhost:3003/`

### 命令
```bash
## 安装依赖
npm install

## 启动项目
npm run dev

## mock方式启动项目
npm run dev:mock
```

### 构建

```bash
## 构建正式环境
npm run build

## 构建测试环境
npm run build:test
```

### 其他

```bash
## 预览构建产物
npm run preview

## 代码格式检查
npm run lint

## 代码格式检查与自动修复
npm run lint:fix

```
