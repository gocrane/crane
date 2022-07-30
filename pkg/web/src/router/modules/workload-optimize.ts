import { IRouter } from '../index';
import { lazy } from 'react';
import { AppIcon, SettingIcon } from 'tdesign-icons-react';

const workloadOptimize: IRouter[] = [
  {
    path: '/workload-optimize',
    meta: {
      title: '工作负载优化',
      Icon: AppIcon,
    },
    children: [
      {
        path: 'recommendationRule',
        Component: lazy(() => import('pages/WorkloadOptimize/RecommendationRule')),
        meta: {
          title: '推荐规则',
        },
      },
      {
        path: 'resourcesRecommend',
        Component: lazy(() => import('pages/WorkloadOptimize/ResourcesRecommend')),
        meta: {
          title: '资源推荐',
        },
      },
      {
        path: 'replicaRecommend',
        Component: lazy(() => import('pages/WorkloadOptimize/ReplicaRecommend')),
        meta: {
          title: '副本数推荐',
        },
      },
    ],
  },
];

export default workloadOptimize;
