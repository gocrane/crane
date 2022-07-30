import React, { memo } from 'react';
import TopPanel from './components/TopPanel';
import MiddleChart from './components/MiddleChart';
import RankList from './components/RankList';
import Overview from './components/Overview';

const DashBoard = () => (
  <div style={{ overflowX: 'hidden' }}>
    <TopPanel />
    <MiddleChart />
    <RankList />
    <Overview />
  </div>
);

export default memo(DashBoard);
