import { IRouter } from '../index';
import { lazy } from 'react';
import { SettingIcon } from 'tdesign-icons-react';

const settings: IRouter[] = [
  {
    path: '/settings',
    meta: {
      title: '设置',
      Icon: SettingIcon,
    },
    children: [
      {
        path: 'cluster',
        Component: lazy(() => import('pages/Settings/cluster/OverviewPanel')),
        meta: {
          title: '集群管理',
        },
      },
    ],
  },
];

export default settings;
