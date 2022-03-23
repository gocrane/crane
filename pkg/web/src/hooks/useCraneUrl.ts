import { clusterApi } from '../apis/clusterApi';
import { useSelector } from './useSelector';

export const useCraneUrl = () => {
  const selectedClusterId = useSelector(state => state.insight.selectedClusterId);

  const clusterList = clusterApi.useFetchClusterListQuery(null);

  return (clusterList.data?.data?.items ?? []).find(cluster => cluster.id === selectedClusterId)?.craneUrl;
};
