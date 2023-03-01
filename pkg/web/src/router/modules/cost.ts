import { IRouter } from '../index';
import { lazy } from 'react';
import { ChartIcon } from 'tdesign-icons-react';
import { useTranslation } from 'react-i18next';

export const useCostRouteConfig = (): IRouter[] => {
  const { t } = useTranslation();
  return [
    {
      path: '/cost',
      meta: {
        title: t('成本洞察'),
        Icon: ChartIcon,
      },
      children: [
        {
          path: 'insight',
          Component: lazy(() => import('pages/Cost/insight/InsightPanel')),
          meta: {
            title: t('成本分布'),
          },
        },
        {
          path: 'carbon',
          Component: lazy(() => import('pages/Cost/CarbonInsight/Index')),
          meta: {
            title: t('碳排放分析'),
          },
        },
        {
          path: 'workload-overview',
          Component: lazy(() => import('pages/Cost/WorkloadOverview/WorkloadOverviewPanel')),
          meta: {
            title: t('应用总览'),
          },
        },
        {
          path: 'workload-insight',
          Component: lazy(() => import('pages/Cost/WorkloadInsight/WorkloadInsightPanel')),
          meta: {
            title: t('应用洞察'),
          },
        },
      ],
    },
  ];
};
