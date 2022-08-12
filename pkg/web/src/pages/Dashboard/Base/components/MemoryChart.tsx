import React from 'react';
import {Col, Row} from 'tdesign-react';
import Style from './MemoryChart.module.less';
import SeriesLineChart, {ISeriesLineChart} from '../../../../components/SeriesLineChart';
import { useTranslation } from 'react-i18next';


const MemoryChart = () => {
  const {t} = useTranslation();
  
  const item: ISeriesLineChart = {
    title: t('Memory 资源使用'),
    subTitle: '( GB )',
    datePicker: true,
    step: '1h',
    xAxis: {type: 'time'},
    lines: [
      {
        name: 'capacity',
        query: `SUM(kube_node_status_capacity{resource="memory", unit="byte"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)  / 1024 / 1024 / 1024 )`,
      },
      {
        name: 'request',
        query: `SUM(kube_pod_container_resource_requests{resource="memory", unit="byte", namespace!=""} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)/ 1024 / 1024 / 1024)`,
      },
      {
        name: 'limits',
        query: `SUM(kube_pod_container_resource_limits{resource="memory", unit="byte", namespace!=""} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node) / 1024 / 1024 / 1024)`,
      },
      {
        name: 'usage',
        query: `SUM(label_replace(container_memory_working_set_bytes{id="/"}, "node", "$1", "instance",  "(.*)")  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) / 1024 / 1024 / 1024`,
      },
    ],
  };

  return (
    <Row gutter={[16, 16]} className={Style.memoryChartPanel}>
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

export default React.memo(MemoryChart);