import { useSelector } from './useSelector';
import { grafanaApi } from 'services/grafanaApi';

export const useIsNeedSelectNamespace = ({ selectedDashboard }: { selectedDashboard?: any } = {})=> {

  const dashboardDetail = grafanaApi.useFetchDashboardDetailQuery(
    { dashboardUid: selectedDashboard?.uid },
    { skip: !selectedDashboard?.uid },
  );

  return (dashboardDetail?.data?.dashboard?.templating?.list ?? []).find(
    (item: { name: string }) => item.name === 'namespace',
  );
};
