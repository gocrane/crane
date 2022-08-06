import { useFetchRecommendationRuleListQuery } from '../../../services/recommendationRuleApi';
import CommonStyle from '../../../styles/common.module.less';
import SearchForm from './components/SearchForm';
import './index.module.less';
import classnames from 'classnames';
import { useCraneUrl } from 'hooks';
import React, { memo, useState } from 'react';
import { Button, Col, Dialog, Divider, Row, Space, Table, Tag } from 'tdesign-react';

export const SelectTable = () => {
  const [selectedRowKeys, setSelectedRowKeys] = useState<(string | number)[]>([0, 1]);
  const [visible, setVisible] = useState(false);
  const craneUrl: any = useCraneUrl();
  const { data, isFetching } = useFetchRecommendationRuleListQuery({ craneUrl });
  const recommendationRuleList = data?.data?.items || [];

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
        <Button>新建推荐规则</Button>
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
        data={recommendationRuleList}
        columns={[
          {
            title: '推荐规则名称',
            colKey: 'metadata.name',
          },
          {
            title: '推荐类型',
            ellipsis: true,
            colKey: 'spec.recommenders[0].name',
            cell({ row }) {
              const recommender = row.spec.recommenders[0].name;
              if (recommender === 'Replicas')
                return (
                  <Tag theme='warning' variant='light'>
                    Replicas
                  </Tag>
                );
              if (recommender === 'Resource')
                return (
                  <Tag theme='primary' variant='light'>
                    Resource
                  </Tag>
                );
              return recommender;
            },
          },
          {
            title: '资源分析对象',
            width: 300,
            ellipsis: true,
            colKey: 'spec.resourceSelectors',
            cell({ row }) {
              const { resourceSelectors } = row.spec;
              return (
                <Space direction='vertical'>
                  {resourceSelectors.map((o: { kind: string; name: string }, i: number) => (
                    <Tag key={i} theme='success' variant='light'>
                      {o.kind}
                      {o.name ? ' / ' : ''}
                      {o.name ?? ''}
                    </Tag>
                  ))}
                </Space>
              );
            },
          },
          {
            title: 'NameSpace',
            ellipsis: true,
            colKey: 'spec.namespaceSelector',
            cell({ row }) {
              const ns = row.spec.namespaceSelector;
              if (ns?.any) return 'Any';
              return ns.matchNames;
            },
          },
          {
            title: '定时推荐',
            ellipsis: true,
            colKey: 'spec.runInterval',
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
