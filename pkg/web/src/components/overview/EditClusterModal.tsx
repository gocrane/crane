import clsx from 'clsx';

import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { ControlPlatformIcon, LinkIcon } from 'tdesign-icons-react';
import { Alert, Button, Dialog, Form, Input, MessagePlugin, Tabs } from 'tdesign-react';

import { clusterApi } from '../../apis/clusterApi';
import { useSelector } from '../../hooks';
import { editClusterActions } from '../../store/editClusterSlice';
import { getErrorMsg } from '../../utils/getErrorMsg';

type Validation = { error: boolean; msg: string };

export const EditClusterModal = React.memo(() => {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const editingClusterId = useSelector(state => state.editCluster.editingClusterId);
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
    if (!visible) {
      dispatch(editClusterActions.resetCluster());
    }
  }, [dispatch, visible]);

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

  const validateClusterName = (id: string) => {
    const res = { error: !clusters.find(cluster => cluster.id === id)?.clusterName, msg: t('????????????????????????') };
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
      res.msg = t('Crane URL????????????');
    } else if (!cluster.craneUrl.startsWith('http://') && !cluster.craneUrl.startsWith('https://')) {
      res.error = true;
      res.msg = t('Crane URL????????????????????????????????????URL');
    }

    setValidation(validation => ({
      ...validation,
      [id]: {
        ...validation[id],
        craneUrl: res
      }
    }));

    if (!res.error) {
      const result: any = dispatch(clusterApi.endpoints.fetchClusterListMu.initiate({ craneUrl: cluster.craneUrl }));
      result
        .unwrap()
        .then(() => MessagePlugin.success(t('????????????: {craneUrl}', { craneUrl: cluster.craneUrl })))
        .catch(() =>
          MessagePlugin.error(
            {
              content: t('????????????: {craneUrl} , ???????????????????????????????????????', { craneUrl: cluster.craneUrl }),
              closeBtn: true
            },
            10000
          )
        );
    }

    return res;
  };

  const renderErrorContent = () => {
    if (mode === 'create') {
      return (
        addClusterMutationOptions.isError && (
          <Alert
            message={getErrorMsg(addClusterMutationOptions.error)}
            style={{ marginBottom: 0, marginTop: '1rem' }}
            theme="error"
          />
        )
      );
    } else if (mode === 'update') {
      return (
        updateClusterMutationOptions.isError && (
          <Alert
            message={getErrorMsg(updateClusterMutationOptions.error)}
            style={{ marginBottom: 0, marginTop: '1rem' }}
            theme="error"
          />
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
    let firstErrorClusterId = null;

    for (const cluster of clusters) {
      const clusterNameRes = validateClusterName(cluster.id);
      const craneUrlRes = validateCraneUrl(cluster.id);

      error = error || clusterNameRes.error || craneUrlRes.error;

      if (error && !firstErrorClusterId) {
        firstErrorClusterId = cluster.id;
      }
    }

    if (error) {
      dispatch(editClusterActions.editingClusterId(firstErrorClusterId));
    } else if (mode === 'create') {
      addClustersMutation({
        data: {
          clusters: (clusters ?? []).map(cluster => {
            return {
              name: cluster.clusterName,
              craneUrl: cluster.craneUrl
            };
          })
        }
      });
    } else if (mode === 'update') {
      updateClusterMutation({
        data: {
          id: clusters[0].id,
          name: clusters[0].clusterName,
          craneUrl: clusters[0].craneUrl
        }
      });
    }
  };

  return (
    <Dialog
      footer={
        <>
          <Button
            theme="default"
            onClick={() => {
              handleClose();
            }}
          >
            {t('??????')}
          </Button>
          <Button
            loading={isLoading}
            onClick={() => {
              handleSubmit();
            }}
          >
            {t('??????')}
          </Button>
        </>
      }
      header={mode === 'create' ? t('????????????') : t('????????????')}
      visible={visible}
      width="50%"
      onClose={() => {
        handleClose();
      }}
    >
      <div style={{ marginBottom: 10 }}>{t('???????????????????????????CRANE Endpoint??????????????????????????????????????????')}</div>
      <Form>
        <Tabs
          addable={mode === 'create'}
          style={{ border: '1px solid var(--td-component-stroke)' }}
          theme="card"
          value={editingClusterId}
          onAdd={
            mode === 'create'
              ? () => {
                  dispatch(editClusterActions.addCluster());
                }
              : null
          }
          onChange={(tabId: string) => {
            dispatch(editClusterActions.editingClusterId(tabId));
          }}
          onRemove={option => {
            dispatch(editClusterActions.deleteCluster({ id: option.value + '' }));
          }}
        >
          {clusters.map((cluster, index) => {
            return (
              <Tabs.TabPanel
                destroyOnHide={false}
                key={cluster.id}
                label={t('??????') + (index + 1)}
                removable={mode === 'create' ? clusters.length !== 1 : false}
                value={cluster.id}
              >
                <div style={{ padding: '24px' }}>
                  <Form.FormItem
                    className={clsx({ isError: validation[cluster.id]?.clusterName?.error })}
                    help={
                      (
                        <span style={{ color: 'var(--td-error-color)' }}>
                          {validation[cluster.id]?.clusterName?.error ? validation[cluster.id]?.clusterName?.msg : null}
                        </span>
                      ) as any
                    }
                    initialData={cluster.clusterName}
                    label={t('????????????')}
                    name={`clusters[${index}].clusterName`}
                    requiredMark
                  >
                    <div style={{ width: '100%' }}>
                      <Input
                        placeholder={t('????????????')}
                        prefixIcon={<ControlPlatformIcon />}
                        value={cluster.clusterName}
                        onBlur={() => {
                          validateClusterName(cluster.id);
                        }}
                        onChange={(clusterName: string) => {
                          dispatch(
                            editClusterActions.updateCluster({
                              id: cluster.id,
                              data: { clusterName }
                            })
                          );
                        }}
                      />
                    </div>
                  </Form.FormItem>
                  <Form.FormItem
                    className={clsx({ isError: validation[cluster.id]?.craneUrl?.error })}
                    help={
                      (
                        <span style={{ color: 'var(--td-error-color)' }}>
                          {validation[cluster.id]?.craneUrl?.error ? validation[cluster.id]?.craneUrl?.msg : null}
                        </span>
                      ) as any
                    }
                    initialData={cluster.craneUrl}
                    label={t('CRANE URL')}
                    name={`clusters[${index}].craneUrl`}
                  >
                    <div style={{ width: '100%' }}>
                      <Input
                        placeholder={'http(s)://(ip/domain):port e.g. http://192.168.1.1:9090 https://gocrane.io:9090'}
                        prefixIcon={<LinkIcon />}
                        value={cluster.craneUrl}
                        onBlur={() => {
                          validateCraneUrl(cluster.id);
                        }}
                        onChange={(craneUrl: string) => {
                          dispatch(
                            editClusterActions.updateCluster({
                              id: cluster.id,
                              data: { craneUrl }
                            })
                          );
                        }}
                      />
                    </div>
                  </Form.FormItem>
                </div>
              </Tabs.TabPanel>
            );
          })}
        </Tabs>
      </Form>
      {renderErrorContent()}
    </Dialog>
  );
});
