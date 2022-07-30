import React, { memo } from 'react';
import TopChart from './components/TopChart';
import BottomTable from './components/BottomTable';

const Deploy = () => (
  <div>
    <TopChart />
    <BottomTable />
  </div>
);

export default memo(Deploy);
