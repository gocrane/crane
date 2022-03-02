import { grafanaApi } from '../apis/grafanaApi';
import { useSelector } from './useSelector';

export const useIsNeedSelectNamespace = () => {
  const selectedDashboard = useSelector(state => state.insight.selectedDashboard);

  const dashboardDetail = grafanaApi.useFetchDashboardDetailQuery(
    { dashboardUid: selectedDashboard?.uid },
    { skip: !selectedDashboard?.uid }
  );

  return (dashboardDetail?.data?.dashboard?.templating?.list ?? []).find(item => item.name === 'namespace');
};
