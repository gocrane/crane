import cost from './modules/cost';
import dashboard from './modules/dashboard';
import detail from './modules/detail';
import form from './modules/form';
import list from './modules/list';
import login from './modules/login';
import manager from './modules/manager';
import otherRoutes from './modules/others';
import result from './modules/result';
import user from './modules/user';
import workloadOptimize from './modules/workload-optimize';
import React, { lazy } from 'react';
import { BrowserRouterProps } from 'react-router-dom';

export interface IRouter {
  path: string;
  redirect?: string;
  Component?: React.FC<BrowserRouterProps> | (() => any);
  /**
   * 当前路由是否全屏显示
   */
  isFullPage?: boolean;
  /**
   * meta未赋值 路由不显示到菜单中
   */
  meta?: {
    title?: string;
    Icon?: React.FC;
    /**
     * 侧边栏隐藏该路由
     */
    hidden?: boolean;
    /**
     * 单层路由
     */
    single?: boolean;
  };
  children?: IRouter[];
}

const routes: IRouter[] = [
  {
    path: '/login',
    Component: lazy(() => import('pages/Login')),
    isFullPage: true,
    meta: {
      hidden: true,
    },
  },
  {
    path: '/',
    redirect: '/dashboard/base',
  },
];

const allRoutes = [
  ...routes,
  ...dashboard,
  ...cost,
  ...workloadOptimize,
  ...manager,
  // ...list,
  // ...form,
  // ...detail,
  // ...result,
  // ...user,
  // ...login,
  ...otherRoutes,
];

export default allRoutes;
