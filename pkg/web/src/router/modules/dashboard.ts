import { lazy } from 'react';
import { DashboardIcon } from 'tdesign-icons-react';
import { IRouter } from '../index';

const dashboard: IRouter[] = [
  {
    path: '/dashboard',
    meta: {
      title: '集群总览',
      Icon: DashboardIcon,
    },
    Component: lazy(() => import('pages/Dashboard/Base')),
  },
];

export default dashboard;
