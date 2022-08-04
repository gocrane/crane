import React, { memo } from 'react';
import TopPanel from './components/TopPanel';

const DashBoard = () => (
  <div style={{ overflowX: 'hidden' }}>
    <TopPanel />
  </div>
);

export default memo(DashBoard);
