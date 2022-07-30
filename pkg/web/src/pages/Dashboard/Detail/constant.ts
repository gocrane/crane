import { IBoardProps, ETrend } from 'components/Board';

export const PANE_LIST: Array<IBoardProps> = [
  {
    title: '总申请数（次）',
    count: '1126',
    trendNum: '10%',
    trend: ETrend.up,
  },
  {
    title: '供应商数量（个）',
    count: '13',
    trendNum: '13%',
    trend: ETrend.down,
  },
  {
    title: '采购商品品类（类）',
    count: '4',
    trendNum: '10%',
    trend: ETrend.up,
  },
  {
    title: '申请人数量（人）',
    count: '90',
    trendNum: '44%',
    trend: ETrend.down,
  },
  {
    title: '申请完成率（%）',
    count: '80.5',
    trendNum: '70%',
    trend: ETrend.up,
  },
  {
    title: '到货及时率（%）',
    count: '78',
    trendNum: '16%',
    trend: ETrend.up,
  },
];

export const PRODUCT_LIST = [
  {
    description: 'SSL证书又叫服务器证书，腾讯云为您提供证书的一站式服务，包括免费、付费证书的申请、管理及部',
    index: 1,
    isSetup: true,
    name: 'SSL证书',
    type: 'D',
    icon: 'user',
  },
  {
    description: 'SSL证书又叫服务器证书，腾讯云为您提供证书的一站式服务，包括免费、付费证书的申请、管理及部',
    index: 1,
    isSetup: true,
    name: 'SSL证书',
    type: 'C',
    icon: 'calendar',
  },
];
