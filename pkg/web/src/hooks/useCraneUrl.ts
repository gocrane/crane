import { clusterApi } from '../apis/clusterApi';
import { getConfig } from '../config';
import { useSelector } from './useSelector';

export const useCraneUrl = () => {
  const isCurrentCluster = useSelector(state => state.insight.isCurrentCluster);
  const selectedClusterId = useSelector(state => state.insight.selectedClusterId);

  const clusterList = clusterApi.useFetchClusterListQuery(null, { skip: isCurrentCluster });

  return isCurrentCluster
    ? process.env.NODE_ENV === 'development'
      ? getConfig().craneUrl
      : location.origin // 生產環境不需要CraneURL，可通過Nginx代理至Grafana
    : (clusterList.data?.data?.items ?? []).find(cluster => cluster.id === selectedClusterId)?.craneUrl;
};
