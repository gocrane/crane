import { Alert, Button, Card, Modal, Table, TableColumn, Text } from 'tea-component';
import { v4 } from 'uuid';

import React, { memo, useEffect, useMemo, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';

import { clusterApi } from '../../apis/clusterApi';
import { useSelector } from '../../hooks';
import { ClusterSimpleInfo } from '../../models';
import { editClusterActions } from '../../store/editClusterSlice';
import { getErrorMsg } from '../../utils/getErrorMsg';

export const OverviewTablePanel = memo(() => {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const [deleteDialog, setDeleteDialog] = React.useState<{
    visible: boolean;
    clusterName?: string;
    clusterId?: string;
  }>({ visible: false });

  const searchFilter = useSelector(state => state.overview.searchFilter);

  const clusterList = clusterApi.useFetchClusterListQuery();
  const [deleteCluster, deleteClusterOptions] = clusterApi.useDeleteClusterMutation();

  const columns = useMemo(() => {
    const columns: TableColumn<ClusterSimpleInfo>[] = [
      {
        key: 'id',
        header: t('ID/名称'),
        render: cluster => {
          return (
            <div>
              <Text parent="div">{cluster.id}</Text>
              <Text parent="div" theme="label">
                {cluster.name}
              </Text>
            </div>
          );
        }
      },
      {
        key: 'craneUrl',
        header: t('Crane URL'),
        render: cluster => {
          return cluster.craneUrl;
        }
      },
      {
        key: 'operation',
        header: t('操作'),
        render: cluster => {
          return (
            <>
              <Button
                type="link"
                onClick={() => {
                  dispatch(editClusterActions.modalVisible(true));
                  dispatch(editClusterActions.mode('update'));
                  dispatch(
                    editClusterActions.setClusters([
                      {
                        id: v4(),
                        clusterId: cluster.id,
                        clusterName: cluster.name,
                        craneUrl: cluster.craneUrl
                      }
                    ])
                  );
                }}
              >
                {t('更新')}
              </Button>
              <Button
                type="link"
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
    return searchFilter && (searchFilter?.clusterIds?.length > 0 || searchFilter?.clusterNames?.length) > 0
      ? list.filter(item => {
          return (
            (searchFilter?.clusterIds ?? []).includes(item.id) || (searchFilter?.clusterNames ?? []).includes(item.name)
          );
        })
      : list;
  }, [clusterList.data?.data?.items, searchFilter]);

  return (
    <>
      <Modal caption={t('删除集群')} visible={deleteDialog.visible} onClose={() => setDeleteDialog({ visible: false })}>
        <Modal.Body>
          <Trans>
            确认删除集群: {{ clusterName: deleteDialog.clusterName ?? '' }}
            {deleteClusterOptions.isError && (
              <Alert className="tea-mt-3n tea-mb-0" type="error">
                {getErrorMsg(deleteClusterOptions.error)}
              </Alert>
            )}
          </Trans>
        </Modal.Body>
        <Modal.Footer>
          <Button
            loading={deleteClusterOptions.isLoading}
            type="primary"
            onClick={() => {
              deleteCluster({
                clusterId: deleteDialog.clusterId
              });
            }}
          >
            {t('确定')}
          </Button>
          <Button
            type="weak"
            onClick={() => {
              setDeleteDialog({ visible: false });
            }}
          >
            取消
          </Button>
        </Modal.Footer>
      </Modal>
      <Card>
        <Card.Body>
          <Table
            addons={[
              Table.addons.pageable(),
              Table.addons.autotip({
                isError: clusterList.isError,
                onRetry: () => {
                  clusterList.refetch();
                },
                isLoading: clusterList.isLoading
              })
            ]}
            columns={columns}
            recordKey={record => record.id}
            records={filteredRecords}
          />
        </Card.Body>
      </Card>
    </>
  );
});
