import { Card } from 'components/common/Card';
import { useCraneUrl, useGrafanaQueryStr, useIsNeedSelectNamespace, useIsValidPanel, useSelector } from 'hooks';
import React from 'react';
import { Col } from 'tdesign-react';

export interface PanelWrapperProps {
  panel: any;
}

export const PanelWrapper = React.memo(({ panel }: PanelWrapperProps) => {
  const selectedDashboard = useSelector((state) => state.insight.selectedDashboard);
  const baselineHeight = useSelector((state) => state.config.chartBaselineHeight);
  const defaultHeight = useSelector((state) => state.config.chartDefaultHeight);
  const selectedNamespace = useSelector((state) => state.insight.selectedNamespace);

  // if it is using current cluster, use crane url from env
  const craneUrl = useCraneUrl();
  const isValidPanel = useIsValidPanel({ panel });
  const isNeedSelectNamespace = useIsNeedSelectNamespace();
  const queryStr = useGrafanaQueryStr({ panelId: panel.id });

  const span = panel?.gridPos?.w > 0 && panel?.gridPos?.w <= 24 ? Math.floor(panel.gridPos.w / 2) : 6;
  const minHeight = panel?.gridPos?.h ? Math.max(panel.gridPos.h * baselineHeight, defaultHeight) : defaultHeight;

  return (isNeedSelectNamespace && !selectedNamespace) || !isValidPanel ? null : (
    <Col key={panel.id} span={span}>
      <Card style={{ marginBottom: '0.5rem', marginTop: '0.5rem', height: minHeight }}>
        <iframe
          frameBorder='0'
          height='100%'
          src={`${craneUrl}/grafana/d-solo/${selectedDashboard?.uid}/costs-by-dimension?${queryStr}`}
          width='100%'
        />
      </Card>
    </Col>
  );
});
