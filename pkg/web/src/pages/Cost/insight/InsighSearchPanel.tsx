import { QueryWindow, useQueryWindowOptions } from '../../../models';
import CommonStyle from '../../../styles/common.module.less';
import classnames from 'classnames';
import { Card } from 'components/common/Card';
import { useCraneUrl, useIsNeedSelectNamespace, useSelector } from 'hooks';
import { insightAction } from 'modules/insightSlice';
import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { useLocation, useNavigate } from 'react-router-dom';
import { DatePicker, DateValue, InputNumber, Radio, RadioValue, Select, SelectValue } from 'tdesign-react';
import { rangeMap } from 'utils/rangeMap';
import { useFetchDashboardListQuery } from '../../../services/grafanaApi';
import { useFetchNamespaceListQuery } from '../../../services/namespaceApi';

export const InsightSearchPanel = React.memo(() => {
  const dispatch = useDispatch();
  const location = useLocation();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const customRange = useSelector((state) => state.insight.customRange);
  const selectedDashboard = useSelector((state) => state.insight.selectedDashboard);
  const window = useSelector((state) => state.insight.window);
  const selectedNamespace = useSelector((state) => state.insight.selectedNamespace);
  const discount = useSelector((state) => state.insight.discount);
  const clusterId = useSelector((state) => state.insight.selectedClusterId);

  const isNeedSelectNamespace = useIsNeedSelectNamespace();
  const craneUrl: any = useCraneUrl();

  const dashboardList = useFetchDashboardListQuery({ craneUrl }, { skip: !craneUrl });
  const namespaceList = useFetchNamespaceListQuery({ clusterId }, { skip: !clusterId || !isNeedSelectNamespace });

  const queryWindowOptions = useQueryWindowOptions();

  const dashboardOptions = React.useMemo(
    () =>
      (dashboardList?.data ?? []).map((dashboard: any) => ({
        label: dashboard.title,
        value: dashboard.uid,
      })),
    [dashboardList?.data],
  );

  const namespaceOptions = React.useMemo(
    () =>
      (namespaceList?.data?.data?.items ?? []).map((namespace) => ({
        label: namespace,
        value: namespace,
      })),
    [namespaceList?.data?.data?.items],
  );

  React.useEffect(() => {
    if (dashboardList.isSuccess && dashboardOptions[0]?.value) {
      dispatch(
        insightAction.selectedDashboard({
          uid: dashboardOptions[0].value,
          title: (dashboardList?.data ?? []).find((data: { uid: any }) => data.uid === dashboardOptions[0].value)
            ?.title,
        }),
      );
    }
  }, [dashboardList?.data, dashboardList.isSuccess, dashboardOptions, dispatch, location.pathname, navigate]);

  React.useEffect(() => {
    if (namespaceList.isSuccess && isNeedSelectNamespace && namespaceOptions?.[0]?.value) {
      dispatch(insightAction.selectedNamespace(namespaceOptions[0].value));
    }
  }, [dispatch, isNeedSelectNamespace, namespaceList.isSuccess, namespaceOptions]);

  return (
    <div className={classnames(CommonStyle.pageWithPadding, CommonStyle.pageWithColor)}>
      <Card style={{ display: 'flex', flexDirection: 'row', flexWrap: 'wrap' }}>
        <div
          style={{
            display: 'flex',
            flexDirection: 'row',
            alignItems: 'center',
            marginRight: '1rem',
            marginTop: 5,
            marginBottom: 5,
          }}
        >
          <div style={{ marginRight: '1rem', width: '70px' }}>{t('Dashboard')}</div>
          <Select
            empty={t('暂无数据')}
            options={dashboardOptions}
            placeholder={t('请选择Dashboard')}
            style={{ paddingBottom: 0, width: '250px' }}
            value={selectedDashboard?.uid}
            onChange={(value: SelectValue) => {
              dispatch(
                insightAction.selectedDashboard({
                  uid: value,
                  title: (dashboardList?.data ?? []).find((data: { uid: string }) => data.uid === value)?.title,
                }),
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
            marginBottom: 5,
          }}
        >
          <div style={{ marginRight: '0.5rem', width: '70px' }}>{t('时间范围')}</div>
          <div style={{ marginRight: '0.5rem' }}>
            <Radio.Group
              value={window}
              onChange={(value: RadioValue) => {
                dispatch(insightAction.window(value as QueryWindow));
                const [start, end] = rangeMap[value as QueryWindow];
                dispatch(
                  insightAction.customRange({ start: start.toDate().toISOString(), end: end.toDate().toISOString() }),
                );
              }}
            >
              {queryWindowOptions.map((option) => (
                <Radio.Button key={option.value} value={option.value}>
                  {option.text}
                </Radio.Button>
              ))}
            </Radio.Group>
          </div>
          <DatePicker
            mode='date'
            style={{ marginRight: '0.5rem' }}
            value={customRange?.start}
            onChange={(start: DateValue) => {
              dispatch(insightAction.window(null as any));
              dispatch(
                insightAction.customRange({
                  ...customRange,
                  start: start as string,
                }),
              );
            }}
          />
          <DatePicker
            mode='date'
            style={{ marginRight: '0.5rem' }}
            value={customRange?.end ?? null}
            onChange={(end: any) => {
              dispatch(insightAction.window(null as any));
              dispatch(
                insightAction.customRange({
                  ...customRange,
                  end,
                }),
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
            marginBottom: 5,
          }}
        >
          <div style={{ marginRight: '1rem', width: '70px' }}>{t('Discount')}</div>
          <InputNumber
            min={0}
            theme='column'
            value={discount}
            onChange={(value) => {
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
              marginBottom: 5,
            }}
          >
            <div style={{ marginRight: '1rem', width: '80px' }}>{t('命名空间')}</div>
            <Select
              options={namespaceOptions}
              placeholder={t('命名空间')}
              value={selectedNamespace ?? undefined}
              onChange={(value: any) => {
                dispatch(insightAction.selectedNamespace(value));
              }}
            />
          </div>
        )}
      </Card>
    </div>
  );
});
