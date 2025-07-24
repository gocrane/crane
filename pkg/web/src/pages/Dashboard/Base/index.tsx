import React, { memo } from 'react';
import TopPanel from './components/TopPanel';
import MiddleChart from './components/MiddleChart';
import CpuChart from './components/CpuChart';
import MemoryChart from './components/MemoryChart';

const DashBoard = () => (
  <div style={{ overflowX: 'hidden' }}>
    <TopPanel />
    <MiddleChart />
    <CpuChart />
    <MemoryChart />
  </div>
);

export default memo(DashBoard);
