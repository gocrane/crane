import { OverviewSearchPanel } from './OverviewSearchPanel';
import { PanelWrapper } from './PanelWrapper';
import {useCraneUrl, useSelector} from 'hooks';
import React, { memo } from 'react';
import { useTranslation } from 'react-i18next';
import {useFetchDashboardDetailQuery, useFetchDashboardListQuery} from 'services/grafanaApi';
import { Row } from 'tdesign-react';

export default memo(() => {
  const { t } = useTranslation();

  const craneUrl: any = useCraneUrl();
  const dashboardList = useFetchDashboardListQuery({ craneUrl }, { skip: !craneUrl });
  const selectedDashboard = (dashboardList?.data ?? []).find((dashboard: any) => dashboard.uid === 'workload-overview');

  const dashboardDetail = useFetchDashboardDetailQuery(
    { dashboardUid: selectedDashboard?.uid },
    { skip: !selectedDashboard?.uid },
  );

  return (
    <>
      <OverviewSearchPanel />
      <Row style={{ marginTop: 10 }}>
        {!selectedDashboard?.uid || dashboardDetail?.data?.dashboard?.panels?.length === 0 ? (
          <span>{t('暂无数据')}</span>
        ) : (
          (dashboardDetail?.data?.dashboard?.panels ?? []).map((panel: any) => (
            <PanelWrapper key={panel.id} panel={panel} selectedDashboard={selectedDashboard} />
          ))
        )}
      </Row>
    </>
  );
});
