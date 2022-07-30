import { lazy } from 'react';
import { UserCircleIcon } from 'tdesign-icons-react';
import { IRouter } from '../index';

const result: IRouter[] = [
  {
    path: '/user',
    meta: {
      title: '个人页',
      Icon: UserCircleIcon,
    },
    children: [
      {
        path: 'index',
        Component: lazy(() => import('pages/User')),
        meta: {
          title: '个人中心',
        },
      },
    ],
  },
];

export default result;
