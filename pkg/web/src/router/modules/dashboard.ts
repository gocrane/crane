import { lazy } from 'react';
import { useTranslation } from 'react-i18next';
import { DashboardIcon } from 'tdesign-icons-react';
import { IRouter } from '../index';

export const useDashboardRouteConfig = (): IRouter[] => {
  const { t } = useTranslation();

  return [
    {
      path: '/dashboard',
      meta: {
        title: t('集群总览'),
        Icon: DashboardIcon,
      },
      Component: lazy(() => import('pages/Dashboard/Base')),
    },
  ];
};
