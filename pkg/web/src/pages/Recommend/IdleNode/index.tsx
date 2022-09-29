import './index.module.less';

import classnames from 'classnames';
import { useCraneUrl } from 'hooks';
import JsYaml from 'js-yaml';
import React, { memo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { Button, Col, Dialog, Divider, MessagePlugin, Row, Space, Table, Tag } from 'tdesign-react';

import {
  RecommendationSimpleInfo,
  RecommendationType,
  useFetchRecommendationListQuery,
} from '../../../services/recommendationApi';
import CommonStyle from '../../../styles/common.module.less';
import SearchForm from './components/SearchForm';

const Editor = React.lazy(() => import('components/common/Editor'));

const SelectTable = () => {
  const { t } = useTranslation();
  const [yamlDialogVisible, setYamlDialogVisible] = useState<boolean>(false);
  const [currentSelection, setCurrentSelection] = useState<RecommendationSimpleInfo | null>(null);

  const navigate = useNavigate();
  const [selectedRowKeys, setSelectedRowKeys] = useState<(string | number)[]>([0, 1]);
  const [visible, setVisible] = useState(false);
  const [filterParams, setFilterParams] = useState({
    namespace: undefined,
    workloadType: undefined,
    name: undefined,
  });
  const craneUrl: any = useCraneUrl();

  const { data, isFetching, isError, isSuccess, error } = useFetchRecommendationListQuery(
    {
      craneUrl,
      recommendationType: RecommendationType.IdleNode,
    },
    { skip: !craneUrl },
  );
  // const recommendation = data?.data?.items || [];

  let recommendation: any[];
  if (isSuccess) {
    recommendation = data?.data?.items || [];
  } else {
    recommendation = [];
    if (isError) MessagePlugin.error(`${(error as any).status} ${(error as any).error}`);
  }

  console.log(data);

  const filterResult = recommendation
    .filter((recommendation) => {
      if (filterParams?.name) {
        return new RegExp(`${filterParams.name}.*`).test(recommendation.name);
      }
      return true;
    })
    .filter((recommendation) => {
      if (filterParams?.workloadType) return filterParams?.workloadType === recommendation.workloadType;
      return true;
    })
    .filter((recommendation) => {
      if (filterParams?.namespace) return filterParams?.namespace === recommendation?.namespace;
      return true;
    });

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

  const toYaml = (resource: any) => {
    let yaml = null;
    try {
      yaml = JsYaml.dump(resource);
    } catch (error) {
      //
    }
    console.log(yaml);
    return yaml;
  };

  return (
    <>
      <Row>
        <Button onClick={() => navigate('/recommend/recommendationRule')}>{t('查看推荐规则')}</Button>
      </Row>
      <Divider></Divider>
      <Row justify='start' style={{ marginBottom: '20px' }}>
        <Col>
          <SearchForm recommendation={recommendation} setFilterParams={setFilterParams} />
        </Col>
      </Row>
      <Table
        loading={isFetching || isError}
        data={filterResult}
        verticalAlign='middle'
        columns={[
          {
            title: t('名称'),
            colKey: 'metadata.name',
            ellipsis: true,
          },
          {
            title: t('节点名'),
            colKey: 'spec.targetRef.name',
          },
          {
            title: t('目标类型'),
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
            title: t('创建时间'),
            ellipsis: true,
            colKey: 'metadata.creationTimestamp',
            cell({ row }) {
              const tmp = new Date(row.metadata.creationTimestamp);
              return `${tmp.toLocaleDateString()} ${tmp.toLocaleTimeString()}`;
            },
          },
          {
            title: t('更新时间'),
            ellipsis: true,
            colKey: 'status.lastUpdateTime',
            cell({ row }) {
              const tmp = new Date(row.status.lastUpdateTime);
              return `${tmp.toLocaleDateString()} ${tmp.toLocaleTimeString()}`;
            },
          },
          {
            align: 'left',
            fixed: 'right',
            width: 200,
            colKey: 'op',
            title: t('操作'),
            cell(record) {
              return (
                <Button
                  theme='primary'
                  variant='text'
                  onClick={() => {
                    setCurrentSelection(record.row as RecommendationSimpleInfo);
                    setYamlDialogVisible(true);
                  }}
                >
                  {t('查看YAML')}
                </Button>
              );
            },
          },
        ]}
        rowKey='index'
        selectedRowKeys={selectedRowKeys}
        hover
        onSelectChange={onSelectChange}
        pagination={{
          defaultCurrent: 1,
          defaultPageSize: 10,
          total: filterResult.length,
          showJumper: true,
          onChange(pageInfo) {
            console.log(pageInfo, 'onChange pageInfo');
          },
          onCurrentChange(current, pageInfo) {
            console.log(current, 'onCurrentChange current');
            console.log(pageInfo, 'onCurrentChange pageInfo');
          },
          onPageSizeChange(size, pageInfo) {
            console.log(size, 'onPageSizeChange size');
            console.log(pageInfo, 'onPageSizeChange pageInfo');
          },
        }}
      />
      <Dialog header={t('确认删除当前所选推荐规则？')} visible={visible} onClose={handleClose}>
        <p>{t('推荐规则将从API Server中删除,且无法恢复')}</p>
      </Dialog>
      <Dialog
        top='5vh'
        width={850}
        visible={yamlDialogVisible}
        onClose={() => {
          setYamlDialogVisible(false);
          setCurrentSelection(null);
        }}
        cancelBtn={null}
        onConfirm={() => {
          setYamlDialogVisible(false);
          setCurrentSelection(null);
        }}
      >
        <React.Suspense fallback={'loading'}>
          <Editor value={currentSelection ? toYaml(currentSelection) ?? '' : ''} />
        </React.Suspense>
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
