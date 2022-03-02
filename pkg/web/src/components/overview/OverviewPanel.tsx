import React from 'react';

import { clusterApi } from '../../apis/clusterApi';
import { EditClusterModal } from './EditClusterModal';
import { OverviewActionPanel } from './OverviewActionPanel';
import { OverviewTablePanel } from './OverviewTablePanel';

export const OverviewPanel = React.memo(() => {
  return (
    <>
      <OverviewActionPanel />
      <OverviewTablePanel />
      <EditClusterModal />
    </>
  );
});
