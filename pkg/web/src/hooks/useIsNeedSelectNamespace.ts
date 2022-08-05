import { useSelector } from './useSelector';
import { grafanaApi } from 'services/grafanaApi';

export const useIsNeedSelectNamespace = () => {
  const selectedDashboard = useSelector((state) => state.insight.selectedDashboard);

  const dashboardDetail = grafanaApi.useFetchDashboardDetailQuery(
    { dashboardUid: selectedDashboard?.uid },
    { skip: !selectedDashboard?.uid },
  );

  return (dashboardDetail?.data?.dashboard?.templating?.list ?? []).find(
    (item: { name: string }) => item.name === 'namespace',
  );
};
