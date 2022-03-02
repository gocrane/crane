import { useCraneUrl } from './useCraneUrl';
import { useSelector } from './useSelector';

export const useExternalCraneUrl = () => {
  const isCurrentCluster = useSelector(state => state.insight.isCurrentCluster);
  const craneUrl = useCraneUrl();

  return isCurrentCluster ? null : craneUrl;
};
