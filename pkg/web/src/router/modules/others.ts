import { lazy } from 'react';
import { IRouter } from '../index';

const otherRoutes: IRouter[] = [
  {
    path: '/403',
    Component: lazy(() => import('pages/Result/403')),
  },
  {
    path: '/500',
    Component: lazy(() => import('pages/Result/500')),
  },
  {
    path: '*',
    Component: lazy(() => import('pages/Result/404')),
  },
];

export default otherRoutes;
