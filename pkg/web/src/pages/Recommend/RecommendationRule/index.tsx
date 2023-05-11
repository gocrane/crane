import {
  RecommendationRuleSimpleInfo,
  useFetchRecommendationRuleListQuery,
} from '../../../services/recommendationRuleApi';
import CommonStyle from '../../../styles/common.module.less';
import SearchForm from './components/SearchForm';
import './index.module.less';
import JsYaml from 'js-yaml';
import classnames from 'classnames';
import { useCraneUrl } from 'hooks';
import React, { memo, useState } from 'react';
import { Button, Col, Dialog, Divider, MessagePlugin, Row, Space, Table, Tag } from 'tdesign-react';
import { useTranslation } from 'react-i18next';

const Editor = React.lazy(() => import('components/common/Editor'));

export const SelectTable = () => {
  const { t } = useTranslation();
  const [yamlDialogVisible, setYamlDialogVisible] = useState<boolean>(false);
  const [currentSelection, setCurrentSelection] = useState<RecommendationRuleSimpleInfo | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<(string | number)[]>([0, 1]);
  const [visible, setVisible] = useState(false);
  const [filterParams, setFilterParams] = useState({
    recommenderType: undefined,
    name: undefined,
  });

  const [pageCurrent, setPageCurrent] = useState<object>({
  current: 1,
  pageSize: 5,
  previous: 2
})
  const craneUrl: any = useCraneUrl();
  const { data, isFetching, isError, isSuccess, error } = useFetchRecommendationRuleListQuery({ craneUrl });
  let recommendationRuleList: any[];
  if (isSuccess) {
    recommendationRuleList = data?.data?.items || [];
  } else {
    recommendationRuleList = [];
    if (isError) MessagePlugin.error(`${error.status} ${error.error}`);
  }
  const filterResult = recommendationRuleList
    .filter((recommendationRule) => {
      console.log(recommendationRule);
      if (filterParams?.name) {
        return new RegExp(`${filterParams.name}.*`).test(recommendationRule.name);
      }
      return true;
    })
    .filter((recommendationRule) => {
      if (filterParams?.recommenderType) return filterParams?.recommenderType === recommendationRule.recommenderType;
      return true;
    });



    let nowTotal = (pageCurrent.current -1 ) * pageCurrent.pageSize

    let newResult = filterResult.slice(nowTotal,filterResult.length >= (nowTotal + pageCurrent.pageSize)  ? nowTotal + pageCurrent.pageSize  : filterResult.length)

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
    return yaml;
  };

  return (
    <>
      {/* <Row>
        <Button disabled>{t('新建推荐规则')}</Button>
      </Row>
      <Divider></Divider> */}
      <Row justify='start' style={{ marginBottom: '20px' }}>
        <Col>
          <SearchForm filterParams={filterParams} setFilterParams={setFilterParams} />
        </Col>
      </Row>
      <Table
        loading={isFetching || isError}
        data={newResult}
        columns={[
          {
            title: t('名称'),
            colKey: 'metadata.name',
          },
          {
            title: t('推荐类型'),
            ellipsis: true,
            colKey: 'spec.recommenders',
            cell({ row }) {
              return (
                <Space direction='vertical'>
                  {row.spec.recommenders.map((recommender) => {
                    if (recommender.name === 'Replicas')
                      return (
                        <Tag theme='warning' variant='light'>
                          {recommender.name}
                        </Tag>
                      );
                    if (recommender.name === 'Resource')
                      return (
                        <Tag theme='primary' variant='light'>
                          {recommender.name}
                        </Tag>
                      );
                  })}
                </Space>
              );
            },
          },
          {
            title: t('推荐目标'),
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
            title: t('Namespace 选择器'),
            ellipsis: true,
            colKey: 'spec.namespaceSelector',
            cell({ row }) {
              const ns = row.spec.namespaceSelector;
              if (ns?.any) return 'Any';
              return ns.matchNames;
            },
          },
          {
            title: t('运行间隔'),
            ellipsis: true,
            colKey: 'spec.runInterval',
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
            align: 'left',
            fixed: 'right',
            width: 200,
            colKey: 'op',
            title: t('操作'),
            cell(record) {
              return (
                <>
                  {/* <Button
                    theme='primary'
                    variant='text'
                    onClick={() => {
                      rehandleClickOp(record);
                    }}
                    disabled={true}
                  >
                    {t('管理')}
                  </Button>
                  <Button
                    theme='primary'
                    variant='text'
                    disabled={true}
                    onClick={() => {
                      handleClickDelete(record);
                    }}
                  >
                    {t('删除')}
                  </Button> */}
                  <Button
                    theme='primary'
                    variant='text'
                    onClick={() => {
                      setCurrentSelection(record.row as RecommendationRuleSimpleInfo);
                      setYamlDialogVisible(true);
                    }}
                  >
                    {t('查看YAML')}
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
        pagination={{
          defaultCurrent: 1,
          defaultPageSize: 5,
          total: filterResult.length,
          showJumper: true,
          onChange(pageInfo) {
            console.log(pageInfo, 'onChange pageInfo');
            // setPageCurrent(pageInfo)

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
        top='15vh'
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
