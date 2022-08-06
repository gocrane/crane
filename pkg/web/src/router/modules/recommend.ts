import { IRouter } from '../index';
import { lazy } from 'react';
import { ScanIcon } from 'tdesign-icons-react';

const recommend: IRouter[] = [
  {
    path: '/recommend',
    meta: {
      title: '成本分析',
      Icon: ScanIcon,
    },
    children: [
      {
        path: 'recommendationRule',
        Component: lazy(() => import('pages/Recommend/RecommendationRule')),
        meta: {
          title: '推荐规则',
        },
      },
      {
        path: 'resourcesRecommend',
        Component: lazy(() => import('pages/Recommend/ResourcesRecommend')),
        meta: {
          title: '资源推荐',
        },
      },
      {
        path: 'replicaRecommend',
        Component: lazy(() => import('pages/Recommend/ReplicaRecommend')),
        meta: {
          title: '副本数推荐',
        },
      },
    ],
  },
];

export default recommend;
