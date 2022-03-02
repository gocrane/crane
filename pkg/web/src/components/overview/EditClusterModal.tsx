import { t } from 'i18next';
import { Modal, Button, Text, Form, Card, Input, Alert } from 'tea-component';

import React from 'react';
import { useDispatch } from 'react-redux';

import { clusterApi } from '../../apis/clusterApi';
import { useSelector } from '../../hooks';
import { editClusterActions } from '../../store/editClusterSlice';
import { getErrorMsg } from '../../utils/getErrorMsg';

type Validation = { error: boolean; msg: string };

export const EditClusterModal = React.memo(() => {
  const dispatch = useDispatch();

  const mode = useSelector(state => state.editCluster.mode);
  const visible = useSelector(state => state.editCluster.modalVisible);
  const clusters = useSelector(state => state.editCluster.clusters);

  const [validation, setValidation] = React.useState<
    Record<string, { clusterId: Validation; clusterName: Validation; craneUrl: Validation }>
  >({});

  const handleClose = () => {
    dispatch(editClusterActions.modalVisible(false));
    dispatch(editClusterActions.resetCluster());
  };

  const [addClustersMutation, addClusterMutationOptions] = clusterApi.useAddClustersMutation();
  const [updateClusterMutation, updateClusterMutationOptions] = clusterApi.useUpdateClusterMutation();

  React.useEffect(() => {
    return () => {
      dispatch(editClusterActions.resetCluster());
    };
  }, [dispatch]);

  React.useEffect(() => {
    if (addClusterMutationOptions.isSuccess && mode === 'create') {
      dispatch(editClusterActions.resetCluster());
      dispatch(editClusterActions.modalVisible(false));
    }
  }, [addClusterMutationOptions.isSuccess, dispatch, mode]);

  React.useEffect(() => {
    if (updateClusterMutationOptions.isSuccess && mode === 'update') {
      dispatch(editClusterActions.resetCluster());
      dispatch(editClusterActions.modalVisible(false));
    }
  }, [dispatch, mode, updateClusterMutationOptions.isSuccess]);

  const validateClusterId = (id: string) => {
    const res = { error: !clusters.find(cluster => cluster.id === id)?.clusterId, msg: t('集群ID不能为空') };
    setValidation(validation => ({
      ...validation,
      [id]: {
        ...validation[id],
        clusterId: res
      }
    }));
    return res;
  };

  const validateClusterName = (id: string) => {
    const res = { error: !clusters.find(cluster => cluster.id === id)?.clusterName, msg: t('集群名称不能为空') };
    setValidation(validation => ({
      ...validation,
      [id]: {
        ...validation[id],
        clusterName: res
      }
    }));
    return res;
  };

  const validateCraneUrl = (id: string) => {
    const res = { error: false, msg: '' };
    const cluster = clusters.find(cluster => cluster.id === id);

    if (!cluster?.craneUrl) {
      res.error = true;
      res.msg = t('Crane URL不能为空');
    } else if (!cluster.craneUrl.startsWith('http://') && !cluster.craneUrl.startsWith('https://')) {
      res.error = true;
      res.msg = t('Crane URL格式不正确，请输入正确的URL');
    }

    setValidation(validation => ({
      ...validation,
      [id]: {
        ...validation[id],
        craneUrl: res
      }
    }));

    return res;
  };

  const renderErrorContent = () => {
    if (mode === 'create') {
      return (
        addClusterMutationOptions.isError && (
          <Alert className="tea-mt-3n tea-mb-0" type="error">
            {getErrorMsg(addClusterMutationOptions.error)}
          </Alert>
        )
      );
    } else if (mode === 'update') {
      return (
        updateClusterMutationOptions.isError && (
          <Alert className="tea-mt-3n tea-mb-0" type="error">
            {getErrorMsg(updateClusterMutationOptions.error)}
          </Alert>
        )
      );
    } else return null;
  };

  const isLoading =
    mode === 'create'
      ? addClusterMutationOptions.isLoading
      : mode === 'update'
      ? updateClusterMutationOptions.isLoading
      : false;

  const handleSubmit = () => {
    let error = false;

    for (const cluster of clusters) {
      const clusterIdRes = validateClusterId(cluster.id);
      const clusterNameRes = validateClusterName(cluster.id);
      const craneUrlRes = validateCraneUrl(cluster.id);

      error = error || clusterIdRes.error || clusterNameRes.error || craneUrlRes.error;
    }

    if (!error) {
      if (mode === 'create') {
        addClustersMutation({
          data: {
            clusters: (clusters ?? []).map(cluster => {
              return {
                id: cluster.clusterId,
                name: cluster.clusterName,
                craneUrl: cluster.craneUrl
              };
            })
          }
        });
      } else if (mode === 'update') {
        updateClusterMutation({
          data: {
            id: clusters[0].clusterId,
            name: clusters[0].clusterName,
            craneUrl: clusters[0].craneUrl
          }
        });
      }
    }
  };

  return (
    <Modal
      caption={mode === 'create' ? t('添加集群') : t('更新集群')}
      size="l"
      visible={visible}
      onClose={() => {
        handleClose();
      }}
    >
      <Modal.Body>
        <Text parent="div" style={{ marginBottom: 10 }} theme="text">
          {t('请输入一个可访问的CRANE Endpoint，以获得新集群的相关成本数据')}
        </Text>
        {clusters.map(cluster => {
          return (
            <Card
              bordered={false}
              key={cluster.id}
              style={{ marginBottom: 5, padding: 15, backgroundColor: '#f3f4f7' }}
            >
              <Card.Body
                operation={
                  mode === 'create' ? (
                    <Button
                      type="link"
                      onClick={() => {
                        dispatch(editClusterActions.deleteCluster({ id: cluster.id }));
                      }}
                    >
                      {t('删除')}
                    </Button>
                  ) : null
                }
                style={{ padding: 10 }}
                title={t('集群配置')}
              >
                <Form>
                  <Form.Item
                    label={t('集群ID')}
                    message={validation[cluster.id]?.clusterId?.error ? validation[cluster.id]?.clusterId?.msg : null}
                    status={validation[cluster.id]?.clusterId?.error ? 'error' : null}
                  >
                    <Input
                      disabled={mode === 'update'}
                      size="l"
                      value={cluster.clusterId}
                      onBlur={() => {
                        validateClusterId(cluster.id);
                      }}
                      onChange={clusterId => {
                        dispatch(
                          editClusterActions.updateCluster({
                            id: cluster.id,
                            data: { clusterId }
                          })
                        );
                      }}
                    />
                  </Form.Item>
                  <Form.Item
                    label={t('集群名称')}
                    message={
                      validation[cluster.id]?.clusterName?.error ? validation[cluster.id]?.clusterName?.msg : null
                    }
                    status={validation[cluster.id]?.clusterName?.error ? 'error' : null}
                  >
                    <Input
                      size="l"
                      value={cluster.clusterName}
                      onBlur={() => {
                        validateClusterName(cluster.id);
                      }}
                      onChange={clusterName => {
                        dispatch(
                          editClusterActions.updateCluster({
                            id: cluster.id,
                            data: { clusterName }
                          })
                        );
                      }}
                    />
                  </Form.Item>
                  <Form.Item
                    label={t('CRANE URL')}
                    message={validation[cluster.id]?.craneUrl?.error ? validation[cluster.id]?.craneUrl?.msg : null}
                    status={validation[cluster.id]?.craneUrl?.error ? 'error' : null}
                  >
                    <Input
                      size="l"
                      value={cluster.craneUrl}
                      onBlur={() => {
                        validateCraneUrl(cluster.id);
                      }}
                      onChange={craneUrl => {
                        dispatch(
                          editClusterActions.updateCluster({
                            id: cluster.id,
                            data: { craneUrl }
                          })
                        );
                      }}
                    />
                  </Form.Item>
                </Form>
              </Card.Body>
            </Card>
          );
        })}
        {mode === 'create' ? (
          <Button
            style={{ width: '100%', marginTop: '1rem' }}
            onClick={() => {
              dispatch(editClusterActions.addCluster());
            }}
          >
            {t('添加')}
          </Button>
        ) : null}
        {renderErrorContent()}
      </Modal.Body>
      <Modal.Footer>
        <Button
          loading={isLoading}
          type="primary"
          onClick={() => {
            handleSubmit();
          }}
        >
          {t('确定')}
        </Button>
        <Button
          type="weak"
          onClick={() => {
            handleClose();
          }}
        >
          {t('取消')}
        </Button>
      </Modal.Footer>
    </Modal>
  );
});
