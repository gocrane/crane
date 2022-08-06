import CommonStyle from '../../../styles/common.module.less';
import SearchForm from './components/SearchForm';
import './index.module.less';
import classnames from 'classnames';
import { useCraneUrl } from 'hooks';
import React, { useState, memo } from 'react';
import { Table, Dialog, Button, Row, Col, Divider, Tag, Space} from "tdesign-react";
import { RecommendationType, useFetchRecommendationListQuery } from '../../../services/recommendationApi';
import { useNavigate } from 'react-router-dom';

export const SelectTable = () => {
  const navigate = useNavigate();
  const [selectedRowKeys, setSelectedRowKeys] = useState<(string | number)[]>([0, 1]);
  const [visible, setVisible] = useState(false);
  const craneUrl: any = useCraneUrl();

  const { data, isFetching } = useFetchRecommendationListQuery({
    craneUrl,
    recommendationType: RecommendationType.Resource,
  });
  const recommendation = data?.data?.items || [];

  function onSelectChange(value: (string | number)[]) {
    setSelectedRowKeys(value);
  }

  function rehandleClickOp(record: any) {
    console.log(record);
  }

  function handleClickDelete(record: any) {
    console.log(record);
    setVisible(true);
  }

  function handleClose() {
    setVisible(false);
  }

  return (
    <>
      <Row>
        <Button onClick={() => navigate('/recommend/recommendationRule')}>管理推荐规则</Button>
      </Row>
      <Divider></Divider>
      <Row justify='start' style={{ marginBottom: '20px' }}>
        <Col>
          <SearchForm
            onSubmit={async (value) => {
              console.log(value);
            }}
            onCancel={() => {}}
          />
        </Col>
      </Row>
      <Table
        loading={isFetching}
        data={recommendation}
        tableLayout='auto'
        verticalAlign='middle'
        columns={[
          {
            title: '推荐规则名称',
            colKey: 'metadata.name',
            ellipsis: true,
          },
          {
            title: '工作负载名称',
            colKey: 'spec.targetRef.name',
          },
          {
            title: 'NameSpace',
            ellipsis: true,
            sortType: 'all',
            colKey: 'spec.targetRef.namespace',
          },
          {
            title: '工作负载类型',
            ellipsis: true,
            colKey: 'spec.targetRef',
            cell({ row }) {
              const { targetRef } = row.spec;
              return (
                <Space direction='vertical'>
                  <Tag theme='success' variant='light'>
                    {targetRef.kind}
                  </Tag>
                </Space>
              );
            },
          },
          {
            title: '推荐资源(容器/CPU/Memory)',
            colKey: 'status.resourceRequest.containers',
            cell({ row }) {
              console.log('row', row);
              if (typeof row.status.recommendedValue !== 'string'){
                const containers = row?.status?.recommendedValue?.resourceRequest?.containers || [];
                if (containers.length > 0) {
                  return (
                    <Space direction='vertical'>
                      {containers.map((o: any, i: number) => (
                        <Tag key={i} theme='primary' variant='light'>
                          {o.containerName} / {o.target.cpu} / {Math.floor(parseFloat(o.target.memory) / 1048576)}Mi
                        </Tag>
                      ))}
                    </Space>
                  );
                }
              }
              return 'abc';
            },
          },
          {
            title: '周期性',
            width: 200,
            colKey: 'spec.completionStrategy.completionStrategyType',
          },
          {
            title: '创建时间',
            ellipsis: true,
            colKey: 'metadata.creationTimestamp',
            cell({ row }) {
              const tmp = new Date(row.metadata.creationTimestamp);
              return `${tmp.toLocaleDateString()} ${tmp.toLocaleTimeString()}`;
            },
          },
          {
            align: 'left',
            fixed: 'right',
            width: 200,
            colKey: 'op',
            title: '操作',
            cell(record) {
              return (
                <>
                  <Button
                    theme='primary'
                    variant='text'
                    onClick={() => {
                      rehandleClickOp(record);
                    }}
                    disabled={true}
                  >
                    管理
                  </Button>
                  <Button
                    theme='primary'
                    variant='text'
                    disabled={true}
                    onClick={() => {
                      handleClickDelete(record);
                    }}
                  >
                    删除
                  </Button>
                </>
              );
            },
          },
        ]}
        rowKey='index'
        selectedRowKeys={selectedRowKeys}
        hover
        onSelectChange={onSelectChange}
      />
      <Dialog header='确认删除当前所选推荐规则？' visible={visible} onClose={handleClose}>
        <p>推荐规则将从API Server中删除,且无法恢复</p>
      </Dialog>
    </>
  );
};

const selectPage: React.FC = () => (
  <div className={classnames(CommonStyle.pageWithPadding, CommonStyle.pageWithColor)}>
    <SelectTable />
  </div>
);

export default memo(selectPage);
