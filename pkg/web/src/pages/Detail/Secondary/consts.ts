export type TStatus = 1 | 2; // 1: 未读 2:已读
export interface IItem {
  id: number;
  content: string;
  tag: string;
  createTime: string;
  status: TStatus;
  type: number;
}

export const dataItemList: IItem[] = [
  {
    id: 1,
    content: '腾讯大厦一楼改造施工项目 已通过审核！',
    tag: '合同动态',
    createTime: '2021-01-01 08:00',
    status: 1,
    type: 1,
  },
  {
    id: 2,
    content: '三季度生产原材料采购项目 开票成功！',
    tag: '票务动态',
    createTime: '2021-01-01 08:00',
    status: 2,
    type: 2,
  },
  {
    id: 3,
    content: '2021-01-01 10:00的【国家电网线下签约】会议即将开始，请提前10分钟前往 会议室1 进行签到！',
    tag: '会议通知',
    createTime: '2021-01-01 08:00',
    status: 1,
    type: 3,
  },
  {
    id: 4,
    content: '一季度生产原材料采购项目 开票成功！',
    tag: '票务动态',
    createTime: '2021-01-01 08:00',
    status: 1,
    type: 2,
  },
  {
    id: 5,
    content: '二季度生产原材料采购项目 开票成功！',
    tag: '票务动态',
    createTime: '2021-01-01 08:00',
    status: 1,
    type: 2,
  },
  {
    id: 6,
    content: '三季度生产原材料采购项目 开票成功！',
    tag: '票务动态',
    createTime: '2021-01-01 08:00',
    status: 1,
    type: 2,
  },
];
