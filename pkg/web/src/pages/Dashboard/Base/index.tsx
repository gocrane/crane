import React, { memo } from 'react';
import TopPanel from './components/TopPanel';
import MiddleChart from "./components/MiddleChart";

const DashBoard = () => (
  <div style={{ overflowX: 'hidden' }}>
    <TopPanel />
    <MiddleChart />
  </div>
);

export default memo(DashBoard);
