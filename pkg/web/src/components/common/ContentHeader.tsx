import { t } from 'i18next';
import { Layout, Select } from 'tea-component';

import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { useMatch } from 'react-router-dom';

import { clusterApi } from '../../apis/clusterApi';
import { getConfig } from '../../config';
import { useSelector } from '../../hooks';
import { insightAction } from '../../store/insightSlice';
import { SegmentOption } from 'tea-component/lib/segment/SegmentOption';

export const ContentHeader = React.memo(() => {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const isOverview = useMatch('/overview');
  const isInsight = useMatch('/insight');

  const selectedClusterId = useSelector(state => state.insight.selectedClusterId);
  const isCurrentCluster = useSelector(state => state.insight.isCurrentCluster);

  const clusterList = clusterApi.useFetchClusterListQuery();

  const options = React.useMemo(() => {
    return [
      {
        text: `${t('当前集群')}`,
        value: getConfig().clusterId
      },
      ...((clusterList.data?.data?.items ?? []).map(item => ({
        text: `${item.name} (${item.id})`,
        value: item.id
      })) as SegmentOption[])
    ];
  }, [clusterList.data?.data?.items, t]);

  return (
    <Layout.Content.Header title={isOverview ? t('成本概览') : t('成本洞察')}>
      {isInsight ? (
        <>
          <Select
            appearance="button"
            matchButtonWidth
            options={options}
            size="m"
            value={isCurrentCluster ? getConfig().clusterId : selectedClusterId}
            onChange={value => {
              if (value === getConfig().clusterId) {
                dispatch(insightAction.isSelectedCurrentCluster(true));
                dispatch(insightAction.selectedClusterId(null));
              } else {
                dispatch(insightAction.isSelectedCurrentCluster(false));
                dispatch(insightAction.selectedClusterId(value));
              }
            }}
          />
        </>
      ) : null}
    </Layout.Content.Header>
  );
});
