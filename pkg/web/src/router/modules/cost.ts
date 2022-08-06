import { IRouter } from '../index';
import { lazy } from 'react';
import { ChartIcon } from 'tdesign-icons-react';

const cost: IRouter[] = [
  {
    path: '/cost',
    meta: {
      title: '成本洞察',
      Icon: ChartIcon,
    },
    children: [
      {
        path: 'insight',
        Component: lazy(() => import('pages/Cost/insight/InsightPanel')),
        meta: {
          title: 'Grafana 图表',
        },
      },
    ],
  },
];

export default cost;
