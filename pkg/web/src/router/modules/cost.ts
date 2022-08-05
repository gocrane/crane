import { IRouter } from '../index';
import { lazy } from 'react';
import { WalletIcon } from 'tdesign-icons-react';

const cost: IRouter[] = [
  {
    path: '/cost',
    meta: {
      title: '成本大师',
      Icon: WalletIcon,
    },
    children: [
      {
        path: 'insight',
        Component: lazy(() => import('pages/Cost/insight/InsightPanel')),
        meta: {
          title: '成本洞察',
        },
      },
    ],
  },
];

export default cost;
