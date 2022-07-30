const Mock = require('mockjs');
export default {
  url: '/api/get-list',
  method: 'get',
  response: () => {
    return {
      code: 0,
      msg: 'ok',
      data: {
        ...Mock.mock({
          'list|100': [
            {
              'index|+1': 1,
              'status|1': '@natural(0, 4)',
              no: 'BH00@natural(01, 100)',
              name: '@city()办公用品采购项目',
              'paymentType|1': '@natural(0, 1)',
              'contractType|1': '@natural(0, 2)',
              updateTime: '2020-05-30 @date("HH:mm:ss")',
              amount: '@natural(10, 500),000,000',
              adminName: '@cname()',
            },
          ],
        }),
      },
    };
  },
};
