import { EmptyTip, Row } from 'tea-component';

import React from 'react';

import { grafanaApi } from '../../apis/grafanaApi';
import { useSelector } from '../../hooks';
import { InsightSearchPanel } from './InsighSearchPanel';
import { PanelWrapper } from './PanelWrapper';

export const InsightPanel = React.memo(() => {
  const selectedDashboard = useSelector(state => state.insight.selectedDashboard);

  const dashboardDetail = grafanaApi.useFetchDashboardDetailQuery(
    { dashboardUid: selectedDashboard?.uid },
    { skip: !selectedDashboard?.uid }
  );

  return (
    <>
      <InsightSearchPanel />
      <Row style={{ marginTop: 10 }}>
        {!selectedDashboard?.uid || dashboardDetail?.data?.dashboard?.panels?.length === 0 ? (
          <EmptyTip />
        ) : (
          (dashboardDetail?.data?.dashboard?.panels ?? []).map((panel: any) => {
            return <PanelWrapper key={panel.id} panel={panel} />;
          })
        )}
      </Row>
    </>
  );
});
