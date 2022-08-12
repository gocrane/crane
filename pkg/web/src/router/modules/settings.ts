import { IRouter } from '../index';
import { lazy } from 'react';
import { SettingIcon } from 'tdesign-icons-react';
import { useTranslation } from 'react-i18next';

export const useSettingRouteConfig = (): IRouter[] => {
  const { t } = useTranslation();
  return [
    {
      path: '/settings',
      meta: {
        title: t('设置'),
        Icon: SettingIcon,
      },
      children: [
        {
          path: 'cluster',
          Component: lazy(() => import('pages/Settings/cluster/OverviewPanel')),
          meta: {
            title: t('集群管理'),
          },
        },
      ],
    },
  ];
};
