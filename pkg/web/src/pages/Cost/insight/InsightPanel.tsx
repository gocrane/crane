import { InsightSearchPanel } from './InsighSearchPanel';
import { PanelWrapper } from './PanelWrapper';
import { useSelector } from 'hooks';
import React, { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { grafanaApi, useFetchDashboardDetailQuery } from "services/grafanaApi";
import { Row } from 'tdesign-react';

export default memo(() => {
  const { t } = useTranslation();
  const selectedDashboard = useSelector((state) => state.insight.selectedDashboard);

  const dashboardDetail = useFetchDashboardDetailQuery(
    { dashboardUid: selectedDashboard?.uid },
    { skip: !selectedDashboard?.uid },
  );

  return (
    <>
      <InsightSearchPanel />
      <Row style={{ marginTop: 10 }}>
        {!selectedDashboard?.uid || dashboardDetail?.data?.dashboard?.panels?.length === 0 ? (
          <span>{t('暂无数据')}</span>
        ) : (
          (dashboardDetail?.data?.dashboard?.panels ?? []).map((panel: any) => (
            <PanelWrapper key={panel.id} panel={panel} />
          ))
        )}
      </Row>
    </>
  );
});
