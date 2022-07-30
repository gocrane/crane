import React from 'react';
import { ChevronRightIcon } from 'tdesign-icons-react';
import { Card } from 'tdesign-react';
import classnames from 'classnames';
import Style from './index.module.less';

export enum ETrend {
  up,
  down,
}

export interface IBoardProps extends React.HTMLAttributes<HTMLElement> {
  title?: string;
  count?: string;
  Icon?: React.ReactElement;
  desc?: string;
  trend?: ETrend;
  trendNum?: string;
  dark?: boolean;
  border?: boolean;
}

export const TrendIcon = ({ trend, trendNum }: { trend?: ETrend; trendNum?: string | number }) => (
  <div
    className={classnames({
      [Style.trendColorUp]: trend === ETrend.up,
      [Style.trendColorDown]: trend === ETrend.down,
    })}
  >
    <div
      className={classnames(Style.trendIcon, {
        [Style.trendIconUp]: trend === ETrend.up,
        [Style.trendIconDown]: trend === ETrend.down,
      })}
    >
      {trend === ETrend.up ? (
        <svg width='16' height='16' viewBox='0 0 16 16' fill='none' xmlns='http://www.w3.org/2000/svg'>
          <path d='M4.5 8L8 4.5L11.5 8' stroke='currentColor' strokeWidth='1.5' />
          <path d='M8 5V12' stroke='currentColor' strokeWidth='1.5' />
        </svg>
      ) : (
        <svg width='16' height='16' viewBox='0 0 16 16' fill='none' xmlns='http://www.w3.org/2000/svg'>
          <path d='M11.5 8L8 11.5L4.5 8' stroke='currentColor' strokeWidth='1.5' />
          <path d='M8 11L8 4' stroke='currentColor' strokeWidth='1.5' />
        </svg>
      )}
    </div>
    {trendNum}
  </div>
);

const Board = ({ title, count, desc, trend, trendNum, Icon, dark, border }: IBoardProps) => (
  <Card
    title={<div className={Style.boardTitle}>{title}</div>}
    header
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
      <div className={Style.boardItemRight}>{Icon}</div>
    </div>
  </Card>
);

export default React.memo(Board);
