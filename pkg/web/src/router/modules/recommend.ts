import { IRouter } from '../index';
import { lazy } from 'react';
import { ScanIcon } from 'tdesign-icons-react';
import { useTranslation } from 'react-i18next';

export const useRecommendRouteConfig = () => {
  const { t } = useTranslation();
  return [
    {
      path: '/recommend',
      meta: {
        title: t('成本分析'),
        Icon: ScanIcon,
      },
      children: [
        {
          path: 'recommendationRule',
          Component: lazy(() => import('pages/Recommend/RecommendationRule')),
          meta: {
            title: t('推荐规则'),
          },
        },
        {
          path: 'resourcesRecommend',
          Component: lazy(() => import('pages/Recommend/ResourcesRecommend')),
          meta: {
            title: t('资源推荐'),
          },
        },
        {
          path: 'replicaRecommend',
          Component: lazy(() => import('pages/Recommend/ReplicaRecommend')),
          meta: {
            title: t('副本数推荐'),
          },
        },
      ],
    },
  ];
};
