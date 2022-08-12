import React from 'react';
import { DateRangePicker, DateRangeValue } from 'tdesign-react';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

const RECENT_7_DAYS: DateRangeValue = [
  dayjs().subtract(7, 'day').format('YYYY-MM-DD'),
  dayjs().subtract(1, 'day').format('YYYY-MM-DD'),
];

const LastWeekDatePicker = (onChange: (value: DateRangeValue) => void) => {
  const { t } = useTranslation();
  return (
    <DateRangePicker
      mode='date'
      placeholder={[t('开始时间'), t('结束时间')]}
      value={RECENT_7_DAYS}
      format='YYYY-MM-DD'
      onChange={(value) => onChange(value)}
    />
  );
};

export default LastWeekDatePicker;
