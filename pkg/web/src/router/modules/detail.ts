import { lazy } from 'react';
import { LayersIcon } from 'tdesign-icons-react';
import { IRouter } from '../index';

const result: IRouter[] = [
  {
    path: '/detail',
    meta: {
      title: '详情页',
      Icon: LayersIcon,
    },
    children: [
      {
        path: 'base',
        Component: lazy(() => import('pages/Detail/Base')),
        meta: {
          title: '基础详情页',
        },
      },
      {
        path: 'advanced',
        Component: lazy(() => import('pages/Detail/Advanced')),
        meta: { title: '多卡片详情页' },
      },
      {
        path: 'deploy',
        Component: lazy(() => import('pages/Detail/Deploy')),
        meta: { title: '数据详情页' },
      },
      {
        path: 'secondary',
        Component: lazy(() => import('pages/Detail/Secondary')),
        meta: { title: '二级详情页' },
      },
    ],
  },
];

export default result;
