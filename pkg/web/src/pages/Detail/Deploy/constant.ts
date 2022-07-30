import type { TdPrimaryTableProps, SorterFun, TableRowData } from 'tdesign-react/es/table';

export interface TableModel {
  name: string;
  adminName: string;
  updateTime: string;
  telephone: string | number;
  op?: undefined | string | number;
}

const nameSorter: SorterFun<TableRowData> = (a, b) => {
  const result = a.name.charCodeAt(0) - b.name.charCodeAt(0);
  if (result > 0) return 1;
  if (result < 0) return -1;
  return 0;
};

export const TABLE_COLUMNS: TdPrimaryTableProps['columns'] = [
  {
    width: '448',
    ellipsis: true,
    colKey: 'name',
    title: '项目名称',
    sortType: 'all',
    sorter: nameSorter,
  },
  {
    width: '224',
    ellipsis: true,
    colKey: 'adminName',
    title: '管理员',
  },
  {
    width: '224',
    ellipsis: true,
    colKey: 'telephone',
    title: '联系方式',
  },
  {
    width: '224',
    className: 'test',
    ellipsis: true,
    colKey: 'updateTime',
    title: '创建时间',
    sortType: 'all',
    sorter: true,
  },
  {
    align: 'left',
    width: '200',
    className: 'test2',
    ellipsis: true,
    colKey: 'op',
    fixed: 'right',
    title: '操作',
  },
];

export const POPUP_DATA = {
  formData: {
    name: '',
    warning: '',
    success: '',
    failB: '',
    warningB: '',
    loading: '',
    add: '',
    help: '',
  },
  options: [
    {
      label: '资源初始化后',
      value: 'beijing',
    },
    {
      label: '上海',
      value: 'shanghai',
    },
  ],
  options1: [
    {
      label: '资源初始化后',
      value: 'guangzhou',
    },
    {
      label: '深圳',
      value: 'shenzhen',
    },
    {
      label: '东莞',
      value: 'dongguang',
    },
  ],
  options2: [
    {
      label: '资源初始化后',
      value: 'nanjing',
    },
    {
      label: '苏州',
      value: '苏州',
    },
    {
      label: '无锡',
      value: 'wuxi',
    },
  ],
  tSelectOptions: [
    {
      label: 'Sanzhang',
      value: '1',
    },
    {
      label: 'ls',
      value: '2',
    },
    {
      label: 'James',
      value: '3',
    },
  ],
};

export const BASE_INFO_DATA = [
  {
    name: '集群名',
    value: 'helloworld',
  },
  {
    name: '集群ID',
    value: 'cls - 2ntelvxw',
    type: {
      key: 'color',
      value: 'blue',
    },
  },
  {
    name: '状态',
    value: '运行中',
    type: {
      key: 'color',
      value: 'green',
    },
  },
  {
    name: 'K8S版本',
    value: '1.7.8',
  },
  {
    name: '配置',
    value: '6.73 核 10.30 GB',
  },
  {
    name: '所在地域',
    value: '广州',
  },
  {
    name: '新增资源所属项目',
    value: '默认项目',
  },
  {
    name: '节点数量',
    value: '4 个',
  },
  {
    name: '节点网络',
    value: 'vpc - 5frmkm1x',
    type: {
      key: 'color',
      value: 'blue',
    },
  },
  {
    name: '容器网络',
    value: '172.16.0.0 / 16',
  },
  {
    name: '集群凭证',
    value: '显示凭证',
    type: {
      key: 'color',
      value: 'blue',
    },
  },
  {
    name: '创建/更新',
    value: '2018-05-31 22:11:44 2018-05-31 22:11:44',
    type: {
      key: 'contractAnnex',
      value: 'pdf',
    },
  },
  {
    name: '描述',
    value: 'istio_test',
  },
];
