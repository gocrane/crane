import {QueryWindow, useQueryWindowOptions} from '../../../models';
import CommonStyle from '../../../styles/common.module.less';
import classnames from 'classnames';
import {Card} from 'components/common/Card';
import {useCraneUrl, useIsNeedSelectNamespace, useSelector} from 'hooks';
import {insightAction} from 'modules/insightSlice';
import React from 'react';
import {useTranslation} from 'react-i18next';
import {useDispatch} from 'react-redux';
import {DatePicker, DateValue, InputNumber, Radio, RadioValue, Select} from 'tdesign-react';
import {rangeMap} from 'utils/rangeMap';
import {useFetchNamespaceListQuery} from '../../../services/namespaceApi';
import {useFetchSeriesListQuery} from '../../../services/grafanaApi';
import {nanoid} from '@reduxjs/toolkit';
import dayjs from 'dayjs';
import _ from 'lodash';

const ALL_NAMESPACE_VALUE = nanoid();

export const OverviewSearchPanel = React.memo(() => {
  const dispatch = useDispatch();
  const {t} = useTranslation();

  const customRange = useSelector((state) => state.insight.customRange);
  const window = useSelector((state) => state.insight.window);
  const selectedNamespace = useSelector((state) => state.insight.selectedNamespace);
  const selectedWorkload = useSelector((state) => state.insight.selectedWorkload);
  const selectedWorkloadType = useSelector((state) => state.insight.selectedWorkloadType);
  const discount = useSelector((state) => state.insight.discount);
  const clusterId = useSelector((state) => state.insight.selectedClusterId);

  const isNeedSelectNamespace = true;
  const craneUrl: any = useCraneUrl();

  const namespaceList = useFetchNamespaceListQuery({clusterId}, {skip: !clusterId || !isNeedSelectNamespace});

  const queryWindowOptions = useQueryWindowOptions();

  const namespaceOptions = React.useMemo(
    () => [
      {
        value: ALL_NAMESPACE_VALUE,
        label: 'ALL',
     },
      ...(namespaceList?.data?.data?.items ?? []).map((namespace) => ({
        label: namespace,
        value: namespace,
     })),
    ],
    [namespaceList?.data?.data?.items],
  );

  const workloadTypeList = useFetchSeriesListQuery(
    {
      craneUrl,
      // start: dayjs(customRange.start).toDate().getTime(),
      // end: dayjs(customRange.end).toDate().getTime(),
      start: '1677721627',
      end: '1677743227',
      match: `crane_analysis_resource_recommendation{namespace=~"(${
        selectedNamespace === ALL_NAMESPACE_VALUE
          ? namespaceOptions
              .filter((option) => option.value === ALL_NAMESPACE_VALUE)
              .map((option) => option.value)
              .join('|')
          : selectedNamespace
     })"}`,
   },
    {},
  );

  const workloadTypeOptions: any[] = React.useMemo(
    () =>
      _.unionBy(
        (workloadTypeList.data?.data ?? []).map((data: any) => ({
          text: data.owner_kind,
          value: data.owner_kind,
       })),
        'value',
      ),
    [workloadTypeList.data],
  );

  const workloadList = useFetchSeriesListQuery({
    craneUrl,
    start: '1677721627',
    end: '1677743227',
    match: `crane_analysis_resource_recommendation{namespace=~"(${
      selectedNamespace === ALL_NAMESPACE_VALUE
        ? namespaceOptions
            .filter((option) => option.value === ALL_NAMESPACE_VALUE)
            .map((option) => option.value)
            .join('|')
        : selectedNamespace
   })"${selectedWorkloadType ? `, owner_kind="${selectedWorkloadType}"` : ''}}`,
 });

  const workloadOptions: any[] = React.useMemo(
    () =>
      _.unionBy(
        (workloadTypeList.data?.data ?? []).map((data: any) => ({
          text: data.owner_name,
          value: data.owner_name,
       })),
        'value',
      ),
    [workloadList],
  );

  React.useEffect(() => {
    if (namespaceList.isSuccess && isNeedSelectNamespace && namespaceOptions?.[0]?.value) {
      dispatch(insightAction.selectedNamespace(namespaceOptions[0].value));
   }
 }, [dispatch, isNeedSelectNamespace, namespaceList.isSuccess, namespaceOptions]);

  return (
    <div className={classnames(CommonStyle.pageWithPadding, CommonStyle.pageWithColor)}>
      <Card style={{display: 'flex', flexDirection: 'row', flexWrap: 'wrap'}}>
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
          <div style={{marginRight: '1rem', width: '80px'}}>{t('命名空间')}</div>
          <Select
            options={namespaceOptions}
            placeholder={t('命名空间')}
            value={selectedNamespace ?? undefined}
            onChange={(value: any) => {
              dispatch(insightAction.selectedNamespace(value));
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
          <div style={{marginRight: '1rem', width: '140px'}}>{t('Workload类型')}</div>
          <Select
            options={workloadTypeOptions}
            placeholder={t('Workload类型')}
            value={selectedWorkloadType ?? undefined}
            onChange={(value: any) => {
              dispatch(insightAction.selectedWorkloadType(value));
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
          <div style={{marginRight: '1rem', width: '80px'}}>{t('Workload')}</div>
          <Select
            options={workloadOptions}
            placeholder={t('Workload')}
            value={selectedWorkload ?? undefined}
            onChange={(value: any) => {
              dispatch(insightAction.selectedWorkload(value));
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
          <div style={{marginRight: '0.5rem', width: '70px'}}>{t('时间范围')}</div>
          <div style={{marginRight: '0.5rem'}}>
            <Radio.Group
              value={window}
              onChange={(value: RadioValue) => {
                dispatch(insightAction.window(value as QueryWindow));
                const [start, end] = rangeMap[value as QueryWindow];
                dispatch(
                  insightAction.customRange({start: start.toDate().toISOString(), end: end.toDate().toISOString()}),
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
            style={{marginRight: '0.5rem'}}
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
            style={{marginRight: '0.5rem'}}
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
      </Card>
    </div>
  );
});
