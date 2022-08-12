import svgr from '@honkhonk/vite-plugin-svgr';
import react from '@vitejs/plugin-react';
import path from 'path';
import { viteMockServe } from 'vite-plugin-mock';

export default (params) => ({
  base: './',
  resolve: {
    alias: {
      assets: path.resolve(__dirname, './src/assets'),
      components: path.resolve(__dirname, './src/components'),
      configs: path.resolve(__dirname, './src/configs'),
      layouts: path.resolve(__dirname, './src/layouts'),
      modules: path.resolve(__dirname, './src/modules'),
      pages: path.resolve(__dirname, './src/pages'),
      styles: path.resolve(__dirname, './src/styles'),
      utils: path.resolve(__dirname, './src/utils'),
      services: path.resolve(__dirname, './src/services'),
      router: path.resolve(__dirname, './src/router'),
      hooks: path.resolve(__dirname, './src/hooks'),
      types: path.resolve(__dirname, './src/types'),
    },
  },

  css: {
    preprocessorOptions: {
      less: {
        modifyVars: {
          // 如需自定义组件其他 token, 在此处配置
        },
      },
    },
  },

  plugins: [
    // monacoEditorPlugin({
    //   languageWorkers: ['typescript'],
    // }),
    svgr(),
    react(),
    params.mode === 'mock' &&
      viteMockServe({
        mockPath: './mock',
        localEnabled: true,
      }),
  ],

  build: {
    cssCodeSplit: false,
  },

  server: {
    host: '0.0.0.0',
    port: 3003,
    proxy: {
      '/api': {
        // 用于开发环境下的转发请求
        // 更多请参考：https://vitejs.dev/config/#server-proxy
        // Set to your craned address
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/grafana': {
        // Set to your craned address
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
});
