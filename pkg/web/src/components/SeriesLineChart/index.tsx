import React, { useState } from 'react';
import { Card, DateRangePicker, MessagePlugin } from "tdesign-react";
import ReactEcharts from 'echarts-for-react';
import { useRangePrometheusQuery } from '../../services/prometheusApi';
import { useCraneUrl } from '../../hooks';
import dayjs from 'dayjs';

export interface ISeriesLine {
  name: string;
  query: string;
  data?: any;
  // if set true, that will be display two lines.
  // One is currentTime-timeRange -> current time, Other one is [(currentTime-timeRange)-preTimeRange] -> (currentTime-preTimeRange)
  preLine?: boolean;
  preTimeRange?: number;
  preLineName?: string;
}

export enum LineStyle {
  Line,
  Area,
}

export interface ISeriesLineChart {
  title?: string;
  subTitle?: string;
  // DatePicker option
  datePicker?: boolean;
  // Prometheus Query Time Range unit: second e.g. 1h => 3600
  timeRange?: number;
  // Prometheus Query Step e.g. 15m
  step?: string;
  // legend: string[];
  xAxis?: {
    // Default: time
    type: string;
    // if set type=category, must be set data
    data?: string[];
  };
  lines: ISeriesLine[];
  lineStyle?: LineStyle;
}

const fetchLinesData = (craneUrl, timeDateRangePicker, step, lines: ISeriesLine[]) => {
  const start = dayjs(timeDateRangePicker[0]).valueOf();
  const end = dayjs(timeDateRangePicker[1]).valueOf();
  return lines.map((line) => {
    const { name, query } = line;
    const { data, isError } = useRangePrometheusQuery({ craneUrl, start, end, step, query });
    if (isError) MessagePlugin.error(`[${name}] Check Your Network Or Query Params !!!`, 10 * 1000);
    console.log(name, data?.metricData, data?.emptyData);
    return {
      ...line,
      data: data?.emptyData ? [] : data?.metricData,
    };
  });
};

const buildLineChartOption = (lineStyle: LineStyle | undefined, linesData: ISeriesLine[]) => {
  if (!linesData) return {};
  const legend = Array.from(linesData, (line) => line.name);
  const series =
    lineStyle === LineStyle.Area
      ? Array.from(linesData, (series) => ({
          name: series.name,
          type: 'line',
          data: series.data,
          areaStyle: {},
          emphasis: {
            focus: 'series',
          },
        }))
      : Array.from(linesData, (series) => ({
          name: series.name,
          type: 'line',
          data: series.data,
        }));
  return {
    tooltip: {
      trigger: 'axis',
    },
    legend: {
      data: legend,
    },
    grid: {
      left: '1%',
      right: '1%',
      bottom: '3%',
      containLabel: true,
    },
    toolbox: {
      feature: {
        saveAsImage: {},
      },
    },
    xAxis: {
      axisLabel: {
        formatter: (axisValue: string | number | Date | dayjs.Dayjs | null | undefined) =>
          dayjs(axisValue).format('MM-DD HH:mm'),
      },
      type: 'time',
    },
    yAxis: {
      type: 'value',
    },
    series,
  };
};

const SeriesLineChart = ({
  title,
  subTitle,
  datePicker,
  timeRange,
  step,
  xAxis,
  lines,
  lineStyle,
}: ISeriesLineChart) => {
  const craneUrl: any = useCraneUrl();

  // Time
  let timeDateRangePicker;
  let setTimeDateRangePicker: (arg0: any) => void;
  if (timeRange != null) {
    [timeDateRangePicker, setTimeDateRangePicker] = useState([
      dayjs().subtract(timeRange, 's').format('YYYY-MM-DD HH:mm:ss'),
      dayjs().subtract(0, 's').format('YYYY-MM-DD HH:mm:ss'),
    ]);
  } else {
    [timeDateRangePicker, setTimeDateRangePicker] = useState([
      dayjs().subtract(30, 'days').format('YYYY-MM-DD HH:mm:ss'),
      dayjs().subtract(0, 's').format('YYYY-MM-DD HH:mm:ss'),
    ]);
  }
  const [presets] = useState({
    最近7天: [dayjs().subtract(6, 'day'), dayjs()],
    最近3天: [dayjs().subtract(2, 'day'), dayjs()],
    最近24小时: [dayjs().subtract(24, 'h'), dayjs()],
    最近1小时: [dayjs().subtract(1, 'h'), dayjs()],
  });

  const onTimeChange = (time: any) => {
    console.log(time);
    setTimeDateRangePicker(time);
  };

  // Fetch Data
  const linesData = fetchLinesData(craneUrl, timeDateRangePicker, step, lines);

  // Build ECharts Option
  const dynamicLineChartOption = buildLineChartOption(lineStyle, linesData);

  return (
    <Card
      title={title}
      subtitle={subTitle}
      actions={
        datePicker && (
          <DateRangePicker
            mode='date'
            placeholder={['开始时间', '结束时间']}
            value={timeDateRangePicker}
            format='YYYY-MM-DD HH:mm:ss'
            enableTimePicker
            presets={presets}
            onChange={onTimeChange}
          />
        )
      }
    >
      <ReactEcharts option={dynamicLineChartOption} notMerge={true} lazyUpdate={false} />
    </Card>
  );
};

export default React.memo(SeriesLineChart);
