import { lazy } from 'react';
import { QueueIcon } from 'tdesign-icons-react';
import { IRouter } from '../index';

const result: IRouter[] = [
  {
    path: '/form',
    meta: {
      title: '表单类',
      Icon: QueueIcon,
    },
    children: [
      {
        path: 'base',
        Component: lazy(() => import('pages/Form/Base')),
        meta: {
          title: '基础表单页',
        },
      },
      {
        path: 'step',
        Component: lazy(() => import('pages/Form/Step')),
        meta: { title: '分步表单页' },
      },
    ],
  },
];

export default result;
