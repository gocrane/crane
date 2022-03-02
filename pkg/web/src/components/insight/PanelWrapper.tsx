import clsx from 'clsx';
import { Card, Col } from 'tea-component';

import styles from './PanelWrapper.module.css';

import React from 'react';

import { useCraneUrl } from '../../hooks/useCraneUrl';
import { useGrafanaQueryStr } from '../../hooks/useGrafanaQueyStr';
import { useIsNeedSelectNamespace } from '../../hooks/useIsNeedSelectNamespace';
import { useIsValidPanel } from '../../hooks/useIsValidPanel';
import { useSelector } from '../../hooks/useSelector';

export interface PanelWrapperProps {
  panel: any;
}

export const PanelWrapper = React.memo(({ panel }: PanelWrapperProps) => {
  const selectedDashboard = useSelector(state => state.insight.selectedDashboard);
  const baselineHeight = useSelector(state => state.config.chartBaselineHeight);
  const defaultHeight = useSelector(state => state.config.chartDefaultHeight);
  const selectedNamespace = useSelector(state => state.insight.selectedNamespace);

  // if it is using current cluster, use crane url from env
  const craneUrl = useCraneUrl();
  const isValidPanel = useIsValidPanel({ panel });
  const isNeedSelectNamespace = useIsNeedSelectNamespace();
  const queryStr = useGrafanaQueryStr({ panelId: panel.id });

  const span = panel?.gridPos?.w > 0 && panel?.gridPos?.w <= 24 ? panel?.gridPos?.w : 12;
  const minHeight = Math.max(panel?.gridPos?.h * baselineHeight, defaultHeight);

  return (isNeedSelectNamespace && !selectedNamespace) || !isValidPanel ? null : (
    <Col key={panel.id} span={span} style={{ minHeight: minHeight }}>
      <Card className={clsx([styles.fullHeightCard, 'fullHeightCard'])}>
        <Card.Body>
          <iframe
            frameBorder="0"
            height="100%"
            src={`${craneUrl}/grafana/d-solo/${selectedDashboard?.uid}/costs-by-dimension?${queryStr}`}
            width="100%"
          />
        </Card.Body>
      </Card>
    </Col>
  );
});
