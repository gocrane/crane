import React from 'react';

import { getConfig } from './../config/index';
import { useSelector } from './useSelector';

export const useClusterId = () => {
  const isSelectCurrentCluster = useSelector(state => state.insight.isCurrentCluster);
  const selectedClusterId = useSelector(state => state.insight.selectedClusterId);

  const clusterId = React.useMemo(() => {
    if (isSelectCurrentCluster) {
      return getConfig().clusterId;
    } else {
      return selectedClusterId;
    }
  }, [isSelectCurrentCluster, selectedClusterId]);

  return clusterId;
};
