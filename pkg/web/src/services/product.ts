import request from 'utils/request';

export interface IProduct {
  banner: string;
  description: string;
  index: number;
  isSetup: boolean;
  name: string;
  type: number;
}

interface IResult {
  list: IProduct[];
}

interface IParams {
  pageSize: number;
  current: number;
}

export const getProductList = async (params: IParams) => {
  const result = await request.get<IResult>('api/get-card-list');

  // 模拟接口分页
  let list = result?.data?.list || [];
  const total = list.length;
  list = list.splice(params.pageSize * (params.current - 1), params.pageSize);
  return {
    list,
    total,
  };
};
