import React, { memo } from 'react';
import { Col, Row } from 'tdesign-react';
import { UsergroupIcon, FileIcon } from 'tdesign-icons-react';
import ReactEcharts from 'echarts-for-react';
import Board, { ETrend, IBoardProps } from 'components/Board';
import useDynamicChart from 'hooks/useDynamicChart';
import { MICRO_CHART_OPTIONS_BAR, MICRO_CHART_OPTIONS_LINE } from '../chart';
import Style from './TopPanel.module.less';

const BarChartIcon = memo(() => {
  const dynamicChartOption = useDynamicChart(MICRO_CHART_OPTIONS_BAR);
  return (
    <div className={Style.paneBarChart}>
      <ReactEcharts
        option={dynamicChartOption} // option：图表配置项
        notMerge={true}
        lazyUpdate={true}
        style={{ height: 36 }}
      />
    </div>
  );
});

const PieChartIcon = memo(() => (
  <div className={Style.paneLineChart}>
    <ReactEcharts
      option={MICRO_CHART_OPTIONS_LINE} // option：图表配置项
      notMerge={true}
      lazyUpdate={true}
      style={{ height: 56 }}
    />
  </div>
));

const PANE_LIST: Array<IBoardProps> = [
  {
    title: '总收入',
    count: '¥ 28,425.00',
    trend: ETrend.up,
    trendNum: '20.5%',
    Icon: <PieChartIcon />,
  },
  {
    title: '总退款',
    count: '¥ 768.00',
    trend: ETrend.down,
    trendNum: '20.5%',
    Icon: <BarChartIcon />,
  },
  {
    title: '活跃用户（个）',
    count: '1126',
    trend: ETrend.down,
    trendNum: '20.5%',
    Icon: (
      <div className={Style.iconWrap}>
        <UsergroupIcon className={Style.svgIcon} />
      </div>
    ),
  },
  {
    title: '产生订单（个）',
    count: '527',
    trend: ETrend.down,
    trendNum: '20.5%',
    Icon: (
      <div className={Style.iconWrap}>
        <FileIcon className={Style.svgIcon} />
      </div>
    ),
  },
];

const TopPanel = () => (
  <Row gutter={[16, 16]}>
    {PANE_LIST.map((item, index) => (
      <Col key={item.title} xs={6} xl={3}>
        <Board
          title={item.title}
          trend={item.trend}
          trendNum={item.trendNum}
          count={item.count}
          desc={'自从上周以来'}
          Icon={item.Icon}
          dark={index === 0}
        />
      </Col>
    ))}
  </Row>
);

export default React.memo(TopPanel);
