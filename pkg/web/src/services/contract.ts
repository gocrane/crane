import request from 'utils/request';

export interface IContract {
  adminName: string;
  amount: string;
  contractType: number;
  index: number;
  name: string;
  no: string;
  paymentType: 1 | 2;
  status: number;
  updateTime: string;
}

interface IResult {
  list: IContract[];
}

interface IParams {
  pageSize: number;
  current: number;
}

export const getContractList = async (params: IParams) => {
  const result = await request.get<IResult>('/api/get-list');

  // 模拟接口分页
  let list = result?.data?.list || [];
  const total = list.length;
  list = list.splice(params.pageSize * (params.current - 1), params.pageSize);
  return {
    list,
    total,
  };
};
