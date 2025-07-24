import { useCostRouteConfig } from './modules/cost';
import otherRoutes from './modules/others';
import React from 'react';
import { BrowserRouterProps } from 'react-router-dom';
import { useDashboardRouteConfig } from './modules/dashboard';
import { useRecommendRouteConfig } from './modules/recommend';
import { useSettingRouteConfig } from './modules/settings';

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
    path: '/',
    redirect: '/dashboard',
  },
];

export const useRouteConfig = () => {
  const cost = useCostRouteConfig();
  const dashboard = useDashboardRouteConfig();
  const recommend = useRecommendRouteConfig();
  const settings = useSettingRouteConfig();

  return [...routes, ...dashboard, ...cost, ...recommend, ...settings, ...otherRoutes];
};
