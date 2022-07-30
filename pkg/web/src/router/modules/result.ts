import { lazy } from 'react';
import { CheckCircleIcon } from 'tdesign-icons-react';
import { IRouter } from '../index';

const result: IRouter[] = [
  {
    path: '/result',
    meta: {
      title: '结果页',
      Icon: CheckCircleIcon,
    },
    children: [
      {
        path: 'success',
        Component: lazy(() => import('pages/Result/Success')),
        meta: {
          title: '成功页',
        },
      },
      {
        path: 'fail',
        Component: lazy(() => import('pages/Result/Fail')),
        meta: {
          title: '失败页',
        },
      },
      {
        path: 'network-error',
        Component: lazy(() => import('pages/Result/NetworkError')),
        meta: {
          title: '网络异常',
        },
      },
      {
        path: '403',
        Component: lazy(() => import('pages/Result/403')),
        meta: {
          title: '无权限',
        },
      },
      {
        path: '404',
        Component: lazy(() => import('pages/Result/404')),
        meta: {
          title: '访问页面不存在页',
        },
      },
      {
        path: '500',
        Component: lazy(() => import('pages/Result/500')),
        meta: {
          title: '服务器出错页',
        },
      },
      {
        path: 'browser-incompatible',
        Component: lazy(() => import('pages/Result/BrowserIncompatible')),
        meta: {
          title: '浏览器不兼容页',
        },
      },
      {
        path: 'maintenance',
        Component: lazy(() => import('pages/Result/Maintenance')),
        meta: {
          title: '系统维护页',
        },
      },
    ],
  },
];

export default result;
