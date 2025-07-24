import React, { memo } from 'react';
import CarbonChart from './components/CarbonChart';

const CarbonDashBoard = () => (
  <div style={{ overflowX: 'hidden' }}>
    <CarbonChart />
  </div>
);

export default memo(CarbonDashBoard);
