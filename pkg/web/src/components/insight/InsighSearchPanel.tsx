import moment from 'moment';
import { Card, DatePicker, Form, InputNumber, Segment, Select, Text } from 'tea-component';

import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { useLocation, useNavigate } from 'react-router-dom';

import { grafanaApi } from '../../apis/grafanaApi';
import { namespaceApi } from '../../apis/namespaceApi';
import { useSelector } from '../../hooks';
import { useClusterId } from '../../hooks/useClusterId';
import { useExternalCraneUrl } from '../../hooks/useExternalCraneUrl';
import { useIsNeedSelectNamespace } from '../../hooks/useIsNeedSelectNamespace';
import { QueryWindow, QueryWindowOptions } from '../../models';
import { insightAction } from '../../store/insightSlice';
import { rangeMap } from '../../utils/rangeMap';

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

  const isNeedSelectNamespace = useIsNeedSelectNamespace();
  const clusterId = useClusterId();
  const craneUrl = useExternalCraneUrl();

  const dashboardList = grafanaApi.useFetchDashboardListQuery({ craneUrl }); // crane url will be null, if it's using current cluster
  const namespaceList = namespaceApi.useFetchNamespaceListQuery(
    { clusterId },
    { skip: !clusterId || !isNeedSelectNamespace }
  );

  const dashboardOptions = React.useMemo(() => {
    return (dashboardList?.data ?? []).map(dashboard => ({
      text: dashboard.title,
      value: dashboard.uid
    }));
  }, [dashboardList?.data]);

  const namespaceOptions = React.useMemo(() => {
    return (namespaceList?.data?.data?.items ?? []).map(namespace => ({
      text: namespace,
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
    <Card>
      <Card.Body>
        <Form className="insight-search-panel" layout="inline">
          <Form.Item
            label={<Text style={{ minWidth: 'auto', paddingRight: '0.5rem' }}>{t('Dashboard')}</Text>}
            style={{ paddingBottom: 0 }}
          >
            <Select
              appearance="button"
              matchButtonWidth
              options={dashboardOptions}
              size="l"
              style={{ paddingBottom: 0 }}
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
          </Form.Item>
          <Segment
            options={QueryWindowOptions}
            value={window}
            onChange={(value: QueryWindow) => {
              dispatch(insightAction.window(value));
              const [start, end] = rangeMap[value];
              dispatch(
                insightAction.customRange({ start: start.toDate().toISOString(), end: end.toDate().toISOString() })
              );
            }}
          />
          <DatePicker.RangePicker
            showTime
            style={{ marginRight: '1rem' }}
            value={[
              customRange?.start ? moment(customRange.start) : null,
              customRange?.end ? moment(customRange.end) : null
            ]}
            onChange={([start, end]) => {
              dispatch(insightAction.window(null));
              dispatch(
                insightAction.customRange({
                  start: start.toDate().toISOString(),
                  end: end.toDate().toISOString()
                })
              );
            }}
          />
          <Form.Item label={<Text style={{ minWidth: 'auto', marginRight: '0.5rem' }}>{t('Discount')}</Text>}>
            <InputNumber
              min={0}
              value={discount}
              onChange={value => {
                dispatch(insightAction.discount(value));
              }}
            />
          </Form.Item>
          {isNeedSelectNamespace && (
            <Form.Item label={<Text style={{ minWidth: 'auto', marginRight: '0.5rem' }}>{t('命名空间')}</Text>}>
              <Select
                appearance="button"
                matchButtonWidth
                options={namespaceOptions}
                placeholder={t('命名空间')}
                size="m"
                value={selectedNamespace ?? null}
                onChange={value => {
                  dispatch(insightAction.selectedNamespace(value));
                }}
              />
            </Form.Item>
          )}
        </Form>
      </Card.Body>
    </Card>
  );
});
