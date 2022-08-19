import React from 'react';
import {Col, Row} from 'tdesign-react';
import Style from './CarbonChart.module.less';
import SeriesLineChart, {ISeriesLineChart} from '../../../../components/SeriesLineChart';
import {useTranslation} from "react-i18next";


const CarbonChart = () => {
  const {t} = useTranslation();

  const energyConsumptionItem: ISeriesLineChart = {
    title: t('能耗'),
    subTitle: t('(瓦特/小时)'),
    datePicker: true,
    step: '1h',
    xAxis: {type: 'time'},
    lines: [
      {
        name: t('energy consumption'),
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

  const carbonItem: ISeriesLineChart = {
    title: t('碳排放'),
    subTitle: t('(克/小时)'),
    datePicker: true,
    step: '1h',
    xAxis: {type: 'time'},
    lines: [
      {
        name: t('carbon emissions'),
        query: `((sum(label_replace(irate(container_cpu_usage_seconds_total{container!="POD", container!="",image!=""}[1h]), "node", "$1", "instance",  "(.*)") * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node))
/
SUM(kube_node_status_capacity{resource="cpu", unit="core"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)))
*
(3.84 - 0.743) + 0.743)
*
(SUM(kube_node_status_capacity{resource="cpu", unit="core"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)))
*
0.581
`,
      },
    ],
  };

  return (
    <Row gutter={[16, 16]} className={Style.carbonChartPanel}>
      <Col span={12}>
        <SeriesLineChart
          title={carbonItem.title}
          subTitle={carbonItem.subTitle}
          datePicker={carbonItem.datePicker}
          lines={carbonItem.lines}
          timeRange={carbonItem.timeRange}
          step={carbonItem.step}
          xAxis={carbonItem.xAxis}
        ></SeriesLineChart>
      </Col>
      <Col span={12}>
        <SeriesLineChart
          title={energyConsumptionItem.title}
          subTitle={energyConsumptionItem.subTitle}
          datePicker={energyConsumptionItem.datePicker}
          lines={energyConsumptionItem.lines}
          timeRange={energyConsumptionItem.timeRange}
          step={energyConsumptionItem.step}
          xAxis={energyConsumptionItem.xAxis}
        ></SeriesLineChart>
      </Col>
    </Row>
  );
};

export default React.memo(CarbonChart);
