import CommonStyle from '../../../styles/common.module.less';
import SearchForm from './components/SearchForm';
import './index.module.less';
import classnames from 'classnames';
import { useCraneUrl } from 'hooks';
import React, { memo, useState } from 'react';
import { Button, Col, Dialog, Divider, Row, Space, Table, Tag } from 'tdesign-react';
import { RecommendationType, useFetchRecommendationListQuery } from '../../../services/recommendationApi';
import { useNavigate } from 'react-router-dom';

export const SelectTable = () => {
  const navigate = useNavigate();
  const [selectedRowKeys, setSelectedRowKeys] = useState<(string | number)[]>([0, 1]);
  const [visible, setVisible] = useState(false);
  const craneUrl: any = useCraneUrl();

  const { data, isFetching } = useFetchRecommendationListQuery({
    craneUrl,
    recommendationType: RecommendationType.Replicas,
  });
  const recommendation = data?.data?.items || [];
  console.log('recommendation', recommendation);

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
            title: '副本数推荐',
            ellipsis: true,
            colKey: 'status.recommendedValue.replicasRecommendation.replicas',
          },
          {
            title: '周期性',
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
                    disabled={true}
                    onClick={() => {
                      rehandleClickOp(record);
                    }}
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
