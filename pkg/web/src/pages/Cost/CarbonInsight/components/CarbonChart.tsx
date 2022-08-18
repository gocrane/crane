import React from 'react';
import { Col, Row } from 'tdesign-react';
import Style from './CarbonChart.module.less';
import SeriesLineChart, { ISeriesLineChart } from '../../../../components/SeriesLineChart';
import { useTranslation } from 'react-i18next';

const CarbonChart = () => {
  const { t } = useTranslation();

  const item: ISeriesLineChart = {
    title: t('碳排放'),
    subTitle: t('(克/小时)'),
    datePicker: true,
    step: '1h',
    xAxis: { type: 'time' },
    lines: [
      {
        name: t('carbon emissions'),
        query: `((sum(label_replace(irate(container_cpu_usage_seconds_total{container!="POD", container!="",image!=""}[1h]), "node", "$1", "instance",  "(.*)") * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node))
/
SUM(kube_node_status_capacity{resource="cpu", unit="core"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)))
*
(3.84 - 0.743) + 0.743)
*
(SUM(kube_node_status_capacity{resource="cpu", unit="core"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)))`,
      },
    ],
  };

  return (
    <Row gutter={[16, 16]} className={Style.carbonChartPanel}>
      <Col span={12}>
        <SeriesLineChart
          title={item.title}
          subTitle={item.subTitle}
          datePicker={item.datePicker}
          lines={item.lines}
          timeRange={item.timeRange}
          step={item.step}
          xAxis={item.xAxis}
        ></SeriesLineChart>
      </Col>
    </Row>
  );
};

export default React.memo(CarbonChart);
