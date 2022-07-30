import { useSelector } from './useSelector';
import { useFetchClusterListQuery } from "services/clusterApi";

export const useCraneUrl = () => {
  const selectedClusterId = useSelector((state) => state.insight.selectedClusterId);

  const clusterList = useFetchClusterListQuery({});

  return (clusterList.data?.data?.items ?? []).find((cluster) => cluster.id === selectedClusterId)?.craneUrl;
};
