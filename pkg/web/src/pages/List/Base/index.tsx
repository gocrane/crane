import React, { useState, memo, useEffect } from 'react';
import { Table, Tag, Row, Col, Button, Input } from 'tdesign-react';
import { ChevronUpCircleIcon, SearchIcon, ChevronDownCircleIcon } from 'tdesign-icons-react';
import classnames from 'classnames';
import { useAppDispatch, useAppSelector } from 'modules/store';
import { selectListBase, getList, clearPageState } from 'modules/list/base';
import CommonStyle from 'styles/common.module.less';
import style from './index.module.less';

export const PaymentTypeMap: {
  [key: number]: React.ReactElement;
} = {
  0: (
    <>
      付款
      <ChevronUpCircleIcon style={{ color: 'red', marginLeft: 4 }} />
    </>
  ),
  1: (
    <>
      收款
      <ChevronDownCircleIcon style={{ color: 'green', marginLeft: 4 }} />
    </>
  ),
};

export const StatusMap: {
  [key: number]: React.ReactElement;
} = {
  1: (
    <Tag theme='warning' variant='light'>
      待审核
    </Tag>
  ),
  2: (
    <Tag theme='warning' variant='light'>
      待履行
    </Tag>
  ),
  3: (
    <Tag theme='success' variant='light'>
      履行中
    </Tag>
  ),
  4: (
    <Tag theme='success' variant='light'>
      已完成
    </Tag>
  ),
  5: (
    <Tag theme='danger' variant='light'>
      审核失败
    </Tag>
  ),
};

export const ContractTypeMap: {
  [key: number]: string;
} = {
  0: '审核失败',
  1: '待审核',
  2: '待履行',
};

export default memo(() => {
  const dispatch = useAppDispatch();
  const pageState = useAppSelector(selectListBase);
  const [selectedRowKeys, setSelectedRowKeys] = useState<(string | number)[]>([1, 2]);

  const { loading, contractList, current, pageSize, total } = pageState;

  useEffect(() => {
    dispatch(
      getList({
        pageSize: pageState.pageSize,
        current: pageState.current,
      }),
    );
    return () => {
      console.log('clear');
      dispatch(clearPageState());
    };
  }, []);

  function onSelectChange(value: (string | number)[]) {
    setSelectedRowKeys(value);
  }
  return (
    <div className={classnames(CommonStyle.pageWithPadding, CommonStyle.pageWithColor)}>
      <Row justify='space-between' className={style.toolBar}>
        <Col>
          <Row gutter={8} align='middle'>
            <Col>
              <Button>新建合同</Button>
            </Col>
            <Col>
              <Button theme='default'>导出合同</Button>
            </Col>
            <Col>
              <div>已选 {selectedRowKeys?.length || 0} 项</div>
            </Col>
          </Row>
        </Col>
        <Col>
          <Input suffixIcon={<SearchIcon />} placeholder='请输入你需要搜索的型号' />
        </Col>
      </Row>

      <Table
        columns={[
          {
            colKey: 'row-select',
            fixed: 'left',
            type: 'multiple',
            // width: 50,
          },
          {
            align: 'left',
            width: 200,
            ellipsis: true,
            colKey: 'name',
            title: '合同名称',
          },
          {
            align: 'left',
            width: 200,
            ellipsis: true,
            colKey: 'status',
            title: '合同状态',
            cell({ row }) {
              return StatusMap[row.status || 5];
            },
          },
          {
            align: 'left',
            width: 200,
            ellipsis: true,
            colKey: 'no',
            title: '合同编号',
          },
          {
            align: 'left',
            width: 200,
            ellipsis: true,
            colKey: 'contractType',
            title: '合同类型',
            cell({ row }) {
              return ContractTypeMap[row.contractType];
            },
          },
          {
            align: 'left',
            width: 200,
            ellipsis: true,
            colKey: 'paymentType',
            title: '合同收付类型',
            cell({ row }) {
              return PaymentTypeMap[row.paymentType];
            },
          },
          {
            align: 'left',
            width: 200,
            ellipsis: true,
            colKey: 'amount',
            title: '合同金额（元）',
          },
          {
            align: 'left',
            fixed: 'right',
            width: 180,
            colKey: 'op',
            title: '操作',
            cell() {
              return (
                <>
                  <Button theme='primary' variant='text'>
                    管理
                  </Button>
                  <Button theme='primary' variant='text'>
                    删除
                  </Button>
                </>
              );
            },
          },
        ]}
        loading={loading}
        data={contractList}
        rowKey='index'
        selectedRowKeys={selectedRowKeys}
        verticalAlign='top'
        hover
        onSelectChange={onSelectChange}
        pagination={{
          pageSize,
          total,
          current,
          showJumper: true,
          onCurrentChange(current, pageInfo) {
            dispatch(
              getList({
                pageSize: pageInfo.pageSize,
                current: pageInfo.current,
              }),
            );
          },
          onPageSizeChange(size) {
            dispatch(
              getList({
                pageSize: size,
                current: 1,
              }),
            );
          },
        }}
      />
    </div>
  );
});
