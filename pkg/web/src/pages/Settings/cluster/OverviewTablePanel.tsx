import { Card } from 'components/common/Card';
import { useSelector } from 'hooks';
import { editClusterActions } from 'modules/editClusterSlice';
import { insightAction } from 'modules/insightSlice';
import React, { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { Alert, Button, Dialog, Table } from 'tdesign-react';
import { getErrorMsg } from 'utils/getErrorMsg';
import { useDeleteClusterMutation, useFetchClusterListQuery } from '../../../services/clusterApi';

export const OverviewTablePanel = memo(() => {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  // const [pagination, setPagination] = React.useState<{ current: number; pageSize: number }>({
  //   current: 1,
  //   pageSize: 10,
  // });
  const [deleteDialog, setDeleteDialog] = React.useState<{
    visible: boolean;
    clusterName?: string;
    clusterId?: string;
  }>({ visible: false });

  const selectedClusterId = useSelector((state) => state.insight.selectedClusterId);
  const searchText = useSelector((state) => state.overview.searchText);

  const clusterList = useFetchClusterListQuery({});
  const [deleteCluster, deleteClusterOptions] = useDeleteClusterMutation();

  React.useEffect(() => {
    if (deleteClusterOptions.isSuccess) {
      setDeleteDialog({ visible: false });
    }
  }, [deleteClusterOptions.isSuccess]);

  const filteredRecords = React.useMemo(() => {
    const list = clusterList.data?.data?.items ?? [];
    return searchText ? list.filter((item) => item.id === searchText || item.name === searchText) : list;
  }, [clusterList.data?.data?.items, searchText]);

  return (
    <div>
      <Dialog
        cancelBtn={
          <Button
            theme='default'
            variant='base'
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
                dispatch(insightAction.selectedClusterId(''));
              }
              deleteCluster({
                clusterId: deleteDialog.clusterId,
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
              theme='error'
            />
          )}
        </div>
      </Dialog>
      <Card style={{ marginTop: '1rem' }}>
        <Table
          columns={[
            {
              align: 'left',
              width: 200,
              ellipsis: true,
              colKey: 'id',
              title: t('集群ID'),
            },
            {
              align: 'left',
              width: 200,
              ellipsis: true,
              colKey: 'name',
              title: t('集群名称'),
            },
            {
              align: 'left',
              width: 200,
              ellipsis: true,
              colKey: 'craneUrl',
              title: t('CraneURL'),
            },
            {
              align: 'left',
              fixed: 'right',
              width: 180,
              colKey: 'op',
              title: t('操作'),
              cell({ row: cluster }) {
                return (
                  <>
                    <Button
                      theme='primary'
                      variant='text'
                      onClick={() => {
                        dispatch(editClusterActions.modalVisible(true));
                        dispatch(editClusterActions.mode('update'));
                        dispatch(editClusterActions.editingClusterId(cluster.id));
                        dispatch(
                          editClusterActions.setClusters([
                            {
                              id: cluster.id,
                              clusterName: cluster.name,
                              craneUrl: cluster.craneUrl,
                            },
                          ]),
                        );
                      }}
                    >
                      {t('修改')}
                    </Button>
                    <Button
                      theme='primary'
                      variant='text'
                      onClick={() => {
                        setDeleteDialog({ visible: true, clusterId: cluster.id, clusterName: cluster.name });
                      }}
                    >
                      {t('删除')}
                    </Button>
                  </>
                );
              },
            },
          ]}
          data={filteredRecords}
          empty={t('暂无数据')}
          // pagination={{
          //   pageSizeOptions: [],
          //   totalContent: t('共 {total} 项数据', { total: filteredRecords.length }),
          //   pageSize: pagination.pageSize,
          //   current: pagination.current,
          //   onCurrentChange: (current) => {
          //     setPagination((pagination) => ({ ...pagination, current }));
          //   },
          //   onPageSizeChange: (pageSize) => {
          //     setPagination((pagination) => ({ ...pagination, pageSize }));
          //   },
          // }}
          rowKey={'id'}
        />
      </Card>
    </div>
  );
});
