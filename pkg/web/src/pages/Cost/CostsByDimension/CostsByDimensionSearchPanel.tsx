import { QueryWindow, useQueryWindowOptions } from '../../../models';
import CommonStyle from '../../../styles/common.module.less';
import classnames from 'classnames';
import { Card } from 'components/common/Card';
import { useSelector } from 'hooks';
import { insightAction } from 'modules/insightSlice';
import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { DatePicker, DateValue, InputNumber, Radio, RadioValue } from 'tdesign-react';
import { rangeMap } from 'utils/rangeMap';

export const CostsByDimensionSearchPanel = React.memo(() => {
  const dispatch = useDispatch();
  const { t } = useTranslation();

  const customRange = useSelector((state) => state.insight.customRange);
  const window = useSelector((state) => state.insight.window);
  const discount = useSelector((state) => state.insight.discount);

  const queryWindowOptions = useQueryWindowOptions();

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
      </Card>
    </div>
  );
});
