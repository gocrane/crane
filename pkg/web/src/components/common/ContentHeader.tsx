import { useSelector } from '../../hooks';
import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { useMatch } from 'react-router-dom';
import { Select } from 'tdesign-react';
import { clusterApi } from 'services/clusterApi';
import { insightAction } from 'modules/insightSlice';

export const ContentHeader = React.memo(() => {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const isOverview = useMatch('/overview');
  const isInsight = useMatch('/insight');

  const selectedClusterId = useSelector((state) => state.insight.selectedClusterId);

  const clusterList = clusterApi.useFetchClusterListQuery({});

  const options = React.useMemo(
    () =>
      (clusterList.data?.data?.items ?? []).map((item) => ({
        text: `${item.name} (${item.id})`,
        value: item.id,
      })),
    [clusterList.data?.data?.items],
  );

  React.useEffect(() => {
    if (clusterList.isSuccess && options.length > 0) {
      dispatch(insightAction.selectedClusterId(options[0]?.value));
    }
  }, [clusterList.isSuccess, dispatch, options]);

  return (
    <div
      style={{ background: 'white', padding: '0.5rem', display: 'flex', flexDirection: 'row', alignItems: 'center' }}
    >
      <h3 style={{ marginRight: '1rem', marginLeft: '0.5rem' }}>{isOverview ? t('成本概览') : t('成本洞察')}</h3>
      {isInsight ? (
        <div>
          <Select
            empty={t('暂无数据')}
            placeholder={t('请选择集群')}
            style={{ width: '200px' }}
            value={selectedClusterId}
            onChange={(value: any) => {
              dispatch(insightAction.selectedClusterId(value));
            }}
          >
            {options.map((option) => (
              <Select.Option key={option.value} label={option.text} value={option.value} />
            ))}
          </Select>
        </div>
      ) : null}
    </div>
  );
});
