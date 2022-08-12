import React from 'react';
import {Col, Row} from 'tdesign-react';
import Style from './CpuChart.module.less';
import SeriesLineChart, {ISeriesLineChart, LineStyle} from '../../../../components/SeriesLineChart';
import { useTranslation } from 'react-i18next';



const CpuChart = () => {
  const {t} = useTranslation();
  
  const item: ISeriesLineChart = {
    title: 'CPU 资源使用',
    subTitle: '(Core)',
    datePicker: true,
    step: '1h',
    xAxis: {type: 'time'},
    lines: [
      {
        name: 'capacity',
        query: `SUM(kube_node_status_capacity{resource="cpu", unit="core"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node))`,
      },
      {
        name: 'request',
        query: `SUM(kube_pod_container_resource_requests{resource="cpu", unit="core"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node))`,
      },
      {
        name: 'limit',
        query: `SUM(kube_pod_container_resource_limits{resource="cpu", unit="core"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node))`,
      },
      {
        name: 'usage',
        query: `sum(label_replace(irate(container_cpu_usage_seconds_total{container!="POD", container!="",image!=""}[1h]), "node", "$1", "instance",  "(.*)") * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node))`,
      },
    ],
  };
  return (
    <Row gutter={[16, 16]} className={Style.cpuChartPanel}>
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

export default React.memo(CpuChart);