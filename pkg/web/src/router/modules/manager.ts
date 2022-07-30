import { IRouter } from '../index';
import { lazy } from 'react';
import { SettingIcon } from 'tdesign-icons-react';

const manager: IRouter[] = [
  {
    path: '/manager',
    meta: {
      title: '管理中心',
      Icon: SettingIcon,
    },
    children: [
      {
        path: 'cluster',
        Component: lazy(() => import('pages/Manager/cluster/OverviewPanel')),
        meta: {
          title: '集群管理',
        },
      },
    ],
  },
];

export default manager;
