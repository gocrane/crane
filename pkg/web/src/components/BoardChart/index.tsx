import React from 'react';
import { ChevronRightIcon, CloseCircleFilledIcon, CloseCircleIcon, UsergroupIcon } from 'tdesign-icons-react';
import { Card } from 'tdesign-react';
import classnames from 'classnames';
import Style from './index.module.less';
import { useInstantPrometheusQuery, useRangePrometheusQuery } from '../../services/prometheusApi';
import { useCraneUrl } from '../../hooks';
import ReactEcharts from 'echarts-for-react';

export enum ETrend {
  up,
  down,
  error,
}

export enum DataSource {
  Prometheus = 'Prometheus',
}

export enum ChartType {
  Pie = 'Pie',
  Line = 'Line',
}

export enum TimeType {
  Instant = 'Instant',
  Range = 'Range',
}

export interface IBoardProps extends React.HTMLAttributes<HTMLElement> {
  // Board Title
  title?: string;
  // The latest value from the metrics
  count?: string;
  countPrefix?: string;
  Icon?: React.ReactElement;
  desc?: string;
  trend?: ETrend;
  // Calc after fetch metrics
  trendNum?: string;
  dark?: boolean;
  border?: boolean;
  // Line Color
  lineColor?: string;
  // Data source - Prometheus
  dataSource?: DataSource;
  // Chart Type - Pie, Line
  chartType?: ChartType;
  // Time Type - Instant, Range
  timeType?: TimeType;
  // Prometheus Query Language
  query?: string;
  // Query time range, unit: second. If start time not set, will be use current time to calc it.
  timeRange?: number;
  // Prometheus Query Step
  step?: string;
  // Prometheus Query Start Time, unit: unix timestamp; Trans to sec: Math.floor(Date.now() / 1000)
  start?: number;
  // Prometheus Query End Time, unit: unix timestamp; Trans to sec: Math.floor(Date.now() / 1000)
  end?: number;
}

const fetchData = (craneUrl: string, { query, timeType, start, end, step }: IBoardProps) => {
  if (typeof query !== 'string')
    return {
      error: 'must be set query',
    };
  let result;
  let preTime;
  let preResult;
  switch (timeType) {
    case TimeType.Instant:
      result = useInstantPrometheusQuery({ craneUrl, query });
      return {
        result,
        error: '',
      };
    case TimeType.Range:
      result = useRangePrometheusQuery({ craneUrl, start, end, step, query });
      // 1 week
      preTime = Math.floor(Date.now() / 1000) - 604800;
      // Hour
      // preTime = (Math.floor(Date.now() / 1000) - 3600) * 1000;
      preResult = useInstantPrometheusQuery({ craneUrl, query, time: preTime });
      return {
        result,
        preResult,
        error: '',
      };
    default:
      return {
        error: 'must be set timeType',
      };
  }
};

const buildIcon = ({ data }, { title, timeType, lineColor = '#0352d9' }): { Icon: any; error: string } => {
  console.log(title, 'data', data);
  const dynamicChartOption = {
    dataset: {
      dimensions: ['timestamp', title],
      source: data?.data[0]?.values || [],
    },
    xAxis: {
      type: 'time',
      show: false,
    },
    yAxis: {
      show: false,
    },
    grid: {
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      tooltip: {
        show: false,
      },
    },
    color: [lineColor],
    series: [
      {
        name: title,
        type: 'line',
        smooth: false,
        encode: {
          x: 'timestamp',
          y: title, // refer sensor 1 value
        },
        showSymbol: false,
      },
    ],
  };
  if (data?.emptyData)
    return {
      Icon: (
        <div className={Style.iconWrap}>
          <UsergroupIcon className={Style.svgIcon} />
        </div>
      ),
      error: '',
    };
  switch (timeType) {
    case TimeType.Range:
      return {
        Icon: (
          <div className={Style.paneLineChart}>
            <ReactEcharts
              option={dynamicChartOption} // option：图表配置项
              notMerge={true}
              lazyUpdate={true}
              style={{ height: 56 }}
            />
          </div>
        ),
        error: '',
      };
    case TimeType.Instant:
      return {
        Icon: {},
        error: '',
      };
    default:
      return {
        Icon: '',
        error: 'must be set timeType',
      };
  }
};

export const TrendIcon = ({ trend, trendNum }: { trend?: ETrend; trendNum?: string | number }) => (
  <div
    className={classnames({
      [Style.trendColorUp]: trend === ETrend.up,
      [Style.trendColorDown]: trend === ETrend.down,
      [Style.trendColorError]: trend === ETrend.error,
    })}
  >
    <div
      className={classnames(Style.trendIcon, {
        [Style.trendIconUp]: trend === ETrend.up,
        [Style.trendIconDown]: trend === ETrend.down,
        [Style.trendColorError]: trend === ETrend.error,
      })}
    >
      {((trend: ETrend) => {
        switch (trend) {
          case ETrend.up:
            return (
              <svg width='16' height='16' viewBox='0 0 16 16' fill='none' xmlns='http://www.w3.org/2000/svg'>
                <path d='M4.5 8L8 4.5L11.5 8' stroke='currentColor' strokeWidth='1.5' />
                <path d='M8 5V12' stroke='currentColor' strokeWidth='1.5' />
              </svg>
            );
          case ETrend.down:
            return (
              <svg width='16' height='16' viewBox='0 0 16 16' fill='none' xmlns='http://www.w3.org/2000/svg'>
                <path d='M11.5 8L8 11.5L4.5 8' stroke='currentColor' strokeWidth='1.5' />
                <path d='M8 11L8 4' stroke='currentColor' strokeWidth='1.5' />
              </svg>
            );
          case ETrend.error:
            return <CloseCircleIcon />;
          default:
            return <CloseCircleIcon />;
        }
      })(trend)}
    </div>
    {trendNum}
  </div>
);
/**
 * Calculates in percent, the change between 2 numbers.
 * e.g from 1000 to 500 = 50%
 *
 * @param oldNumber The initial value
 * @param newNumber The value that changed
 */
function getPercentageChange(oldNumber, newNumber) {
  const decreaseValue = oldNumber - newNumber;

  return (decreaseValue / oldNumber) * 100;
}

const BoardChart = ({
  title,
  count,
  prefix,
  countPrefix,
  desc,
  trend,
  trendNum,
  Icon,
  dark,
  border,
  query,
  timeType,
  lineColor,
  start,
  end,
  step,
}: IBoardProps) => {
  const craneUrl: any = useCraneUrl();
  let fetchDataResult;
  try {
    fetchDataResult = fetchData(craneUrl, { query, timeType, start, end, step });
    console.log(title, 'fetchDataResult', fetchDataResult);
  } catch (e) {
    fetchDataResult = {
      error: e,
      result: {},
    };
    console.log(e);
  }

  const { result, error } = fetchDataResult;

  let IconResult;
  if (!Icon && result?.ifFetching !== true) {
    // Build Icon
    IconResult = buildIcon(result, { title, timeType, lineColor });
    console.log(IconResult);
  }

  if (error) {
    count = error;
  } else if ((typeof result?.isFetching === 'boolean' && result?.isFetching === true) || result?.data?.emptyData) {
    count = 'No Data';
  } else {
    count = `${countPrefix || ''}${result?.data?.latestValue || ''}`;
  }

  if (
    fetchDataResult?.result?.data &&
    fetchDataResult?.preResult?.data &&
    !fetchDataResult?.preResult?.data?.emptyData
  ) {
    const calc = getPercentageChange(
      fetchDataResult.preResult?.data?.latestValue,
      fetchDataResult.result?.data?.latestValue,
    );
    trendNum = `${(Math.floor(calc * 100) / 100) * -1}%`;
    trend = calc < 0 ? ETrend.up : ETrend.down;
  } else {
    console.log('emptyData', fetchDataResult.preResult?.data);
    trendNum = '历史数据不足';
    trend = ETrend.error;
  }

  return (
    <Card
      header={<div className={Style.boardTitle}>{title}</div>}
      className={classnames({
        [Style.boardPanelDark]: dark,
      })}
      bordered={border}
      footer={
        <div className={Style.boardItemBottom}>
          <div className={Style.boardItemDesc}>
            {desc}
            <TrendIcon trend={trend} trendNum={trendNum} />
          </div>
          <ChevronRightIcon className={Style.boardItemIcon} />
        </div>
      }
    >
      <div className={Style.boardItem}>
        <div className={Style.boardItemLeft}>{count}</div>
        <div className={Style.boardItemRight}>{Icon || IconResult.Icon}</div>
      </div>
    </Card>
  );
};

export default React.memo(BoardChart);
