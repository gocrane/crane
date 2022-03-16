import { v4 } from 'uuid';

import React, { memo, useMemo } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { Alert, Button, Dialog, Divider, PrimaryTableCol, Table } from 'tdesign-react';

import { clusterApi } from '../../apis/clusterApi';
import { useSelector } from '../../hooks';
import { ClusterSimpleInfo } from '../../models';
import { editClusterActions } from '../../store/editClusterSlice';
import { insightAction } from '../../store/insightSlice';
import { getErrorMsg } from '../../utils/getErrorMsg';
import { Card } from '../common/Card';

export const OverviewTablePanel = memo(() => {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const [pagination, setPagination] = React.useState<{ current: number; pageSize: number }>({
    current: 1,
    pageSize: 10
  });
  const [deleteDialog, setDeleteDialog] = React.useState<{
    visible: boolean;
    clusterName?: string;
    clusterId?: string;
  }>({ visible: false });

  const selectedClusterId = useSelector(state => state.insight.selectedClusterId);
  const searchText = useSelector(state => state.overview.searchText);

  const clusterList = clusterApi.useFetchClusterListQuery();
  const [deleteCluster, deleteClusterOptions] = clusterApi.useDeleteClusterMutation();

  const columns = useMemo(() => {
    const columns: PrimaryTableCol<ClusterSimpleInfo>[] = [
      {
        colKey: 'id',
        title: t('ID/名称'),
        width: '15%',
        render: ({ row: cluster }) => {
          return (
            <div>
              <div>{cluster.id}</div>
              <div>{cluster.name}</div>
            </div>
          );
        }
      },
      {
        colKey: 'craneUrl',
        title: t('Crane URL'),
        render: ({ row: cluster }) => {
          return cluster.craneUrl;
        }
      },
      {
        colKey: 'operation',
        title: t('操作'),
        render: ({ row: cluster }) => {
          return (
            <>
              <Button
                style={{ padding: 0 }}
                theme="primary"
                variant="text"
                onClick={() => {
                  dispatch(editClusterActions.modalVisible(true));
                  dispatch(editClusterActions.mode('update'));
                  dispatch(editClusterActions.editingClusterId(cluster.id));
                  dispatch(
                    editClusterActions.setClusters([
                      {
                        id: cluster.id,
                        clusterName: cluster.name,
                        craneUrl: cluster.craneUrl
                      }
                    ])
                  );
                }}
              >
                {t('更新')}
              </Button>
              <Divider layout="vertical" />
              <Button
                style={{ paddingLeft: '0.5rem', paddingRight: 0 }}
                theme="primary"
                variant="text"
                onClick={() => {
                  setDeleteDialog({ visible: true, clusterId: cluster.id, clusterName: cluster.name });
                }}
              >
                {t('删除')}
              </Button>
            </>
          );
        }
      }
    ];

    return columns;
  }, [dispatch, t]);

  React.useEffect(() => {
    if (deleteClusterOptions.isSuccess) {
      setDeleteDialog({ visible: false });
    }
  }, [deleteClusterOptions.isSuccess]);

  const filteredRecords = React.useMemo(() => {
    const list = clusterList.data?.data?.items ?? [];
    return searchText
      ? list.filter(item => {
          return item.id === searchText || item.name === searchText;
        })
      : list;
  }, [clusterList.data?.data?.items, searchText]);

  return (
    <>
      <Dialog
        cancelBtn={
          <Button
            theme="default"
            variant="base"
            onClick={() => {
              setDeleteDialog({ visible: false });
            }}
          >
            {t('取消')}
          </Button>
        }
        confirmBtn={
          <Button
            loading={deleteClusterOptions.isLoading}
            onClick={() => {
              // if it is deleting the selected cluster, deselect it
              if (deleteDialog.clusterId === selectedClusterId) {
                dispatch(insightAction.selectedClusterId(null));
              }
              deleteCluster({
                clusterId: deleteDialog.clusterId
              });
            }}
          >
            {t('确定')}
          </Button>
        }
        header={t('删除集群')}
        visible={deleteDialog.visible}
        onClose={() => setDeleteDialog({ visible: false })}
      >
        <div>
          {t('确认删除集群')}: {deleteDialog.clusterName ?? ''}
          {deleteClusterOptions.isError && (
            <Alert
              message={getErrorMsg(deleteClusterOptions.error)}
              style={{ marginBottom: 0, marginTop: '1rem' }}
              theme="error"
            />
          )}
        </div>
      </Dialog>
      <Card style={{ marginTop: '1rem' }}>
        <Table
          columns={columns}
          data={filteredRecords}
          empty={t('暂无数据')}
          pagination={{
            pageSizeOptions: [],
            totalContent: t('共 {total} 项数据', { total: filteredRecords.length }),
            pageSize: pagination.pageSize,
            current: pagination.current,
            onCurrentChange: current => {
              setPagination(pagination => ({ ...pagination, current }));
            },
            onPageSizeChange: pageSize => {
              setPagination(pagination => ({ ...pagination, pageSize }));
            }
          }}
          rowKey={'id'}
        />
      </Card>
    </>
  );
});
