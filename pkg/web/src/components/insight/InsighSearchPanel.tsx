import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { useLocation, useNavigate } from 'react-router-dom';
import { DatePicker, InputNumber, Radio, Select } from 'tdesign-react';

import { grafanaApi } from '../../apis/grafanaApi';
import { namespaceApi } from '../../apis/namespaceApi';
import { useSelector, useCraneUrl, useIsNeedSelectNamespace } from '../../hooks';
import { QueryWindow, QueryWindowOptions } from '../../models';
import { insightAction } from '../../store/insightSlice';
import { rangeMap } from '../../utils/rangeMap';
import { Card } from '../common/Card';

export const InsightSearchPanel = React.memo(() => {
  const dispatch = useDispatch();
  const location = useLocation();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const customRange = useSelector(state => state.insight.customRange);
  const selectedDashboard = useSelector(state => state.insight.selectedDashboard);
  const window = useSelector(state => state.insight.window);
  const selectedNamespace = useSelector(state => state.insight.selectedNamespace);
  const discount = useSelector(state => state.insight.discount);
  const clusterId = useSelector(state => state.insight.selectedClusterId);

  const isNeedSelectNamespace = useIsNeedSelectNamespace();
  const craneUrl = useCraneUrl();

  const dashboardList = grafanaApi.useFetchDashboardListQuery({ craneUrl }, { skip: !craneUrl });
  const namespaceList = namespaceApi.useFetchNamespaceListQuery(
    { clusterId },
    { skip: !clusterId || !isNeedSelectNamespace }
  );

  const dashboardOptions = React.useMemo(() => {
    return (dashboardList?.data ?? []).map(dashboard => ({
      label: dashboard.title,
      value: dashboard.uid
    }));
  }, [dashboardList?.data]);

  const namespaceOptions = React.useMemo(() => {
    return (namespaceList?.data?.data?.items ?? []).map(namespace => ({
      label: namespace,
      value: namespace
    }));
  }, [namespaceList?.data?.data?.items]);

  React.useEffect(() => {
    if (dashboardList.isSuccess && dashboardOptions[0]?.value) {
      dispatch(
        insightAction.selectedDashboard({
          uid: dashboardOptions[0].value,
          title: (dashboardList?.data ?? []).find(data => data.uid === dashboardOptions[0].value)?.title
        })
      );
    }
  }, [dashboardList?.data, dashboardList.isSuccess, dashboardOptions, dispatch, location.pathname, navigate]);

  React.useEffect(() => {
    if (namespaceList.isSuccess && isNeedSelectNamespace && namespaceOptions?.[0]?.value) {
      dispatch(insightAction.selectedNamespace(namespaceOptions[0].value));
    }
  }, [dispatch, isNeedSelectNamespace, namespaceList.isSuccess, namespaceOptions]);

  return (
    <Card style={{ display: 'flex', flexDirection: 'row', flexWrap: 'wrap' }}>
      <div
        style={{
          display: 'flex',
          flexDirection: 'row',
          alignItems: 'center',
          marginRight: '1rem',
          marginTop: 5,
          marginBottom: 5
        }}
      >
        <div style={{ marginRight: '1rem', width: '70px' }}>{t('Dashboard')}</div>
        <Select
          empty={t('暂无数据')}
          options={dashboardOptions}
          placeholder={t('请选择Dashboard')}
          style={{ paddingBottom: 0, width: '250px' }}
          value={selectedDashboard?.uid}
          onChange={(value: string) => {
            dispatch(
              insightAction.selectedDashboard({
                uid: value,
                title: (dashboardList?.data ?? []).find(data => data.uid === value)?.title
              })
            );
          }}
        />
      </div>
      <div
        style={{
          display: 'flex',
          flexDirection: 'row',
          alignItems: 'center',
          marginRight: '1rem',
          marginTop: 5,
          marginBottom: 5
        }}
      >
        <div style={{ marginRight: '0.5rem', width: '70px' }}>{t('时间范围')}</div>
        <div style={{ marginRight: '0.5rem' }}>
          <Radio.Group
            value={window}
            onChange={(value: QueryWindow) => {
              dispatch(insightAction.window(value));
              const [start, end] = rangeMap[value];
              dispatch(
                insightAction.customRange({ start: start.toDate().toISOString(), end: end.toDate().toISOString() })
              );
            }}
          >
            {QueryWindowOptions.map(option => {
              return (
                <Radio.Button key={option.value} value={option.value}>
                  {option.text}
                </Radio.Button>
              );
            })}
          </Radio.Group>
        </div>
        <DatePicker
          mode="date"
          style={{ marginRight: '0.5rem' }}
          value={customRange?.start}
          onChange={(start: string) => {
            dispatch(insightAction.window(null));
            dispatch(
              insightAction.customRange({
                ...customRange,
                start
              })
            );
          }}
        />
        <DatePicker
          mode="date"
          style={{ marginRight: '0.5rem' }}
          value={customRange?.end ?? null}
          onChange={(end: string) => {
            dispatch(insightAction.window(null));
            dispatch(
              insightAction.customRange({
                ...customRange,
                end
              })
            );
          }}
        />
      </div>
      <div
        style={{
          display: 'flex',
          flexDirection: 'row',
          alignItems: 'center',
          marginRight: '1rem',
          marginTop: 5,
          marginBottom: 5
        }}
      >
        <div style={{ marginRight: '1rem', width: '70px' }}>{t('Discount')}</div>
        <InputNumber
          min={0}
          theme="column"
          value={discount}
          onChange={value => {
            dispatch(insightAction.discount(value));
          }}
        />
      </div>
      {isNeedSelectNamespace && (
        <div
          style={{
            display: 'flex',
            flexDirection: 'row',
            alignItems: 'center',
            marginRight: '1rem',
            marginTop: 5,
            marginBottom: 5
          }}
        >
          <div style={{ marginRight: '1rem', width: '80px' }}>{t('命名空间')}</div>
          <Select
            options={namespaceOptions}
            placeholder={t('命名空间')}
            value={selectedNamespace ?? null}
            onChange={(value: string) => {
              dispatch(insightAction.selectedNamespace(value));
            }}
          />
        </div>
      )}
    </Card>
  );
});
