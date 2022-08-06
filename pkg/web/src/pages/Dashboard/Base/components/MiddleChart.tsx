import React, { useState } from 'react';
import { Card, Col, Row } from 'tdesign-react';
import ReactEcharts from 'echarts-for-react';
import useDynamicChart from 'hooks/useDynamicChart';
import { getLineChartOptions, getPieChartOptions } from '../chart';
import Style from './MiddleChart.module.less';
import SeriesLineChart, { LineStyle } from '../../../../components/SeriesLineChart';

const lineOptions = getLineChartOptions();
const pieOptions = getPieChartOptions();

const MiddleChart = () => {
  const [customOptions, setCustomOptions] = useState(lineOptions);

  const onTimeChange = (value: Array<string>) => {
    const options = getLineChartOptions(value);
    setCustomOptions(options);
  };

  const dynamicLineChartOption = useDynamicChart(customOptions, {
    placeholderColor: ['legend.textStyle.color', 'xAxis.axisLabel.color', 'yAxis.axisLabel.color'],
    borderColor: ['series.0.itemStyle.borderColor', 'series.1.itemStyle.borderColor'],
  });

  const dynamicPieChartOption = useDynamicChart(pieOptions, {
    placeholderColor: ['legend.textStyle.color'],
    containerColor: ['series.0.itemStyle.borderColor'],
    textColor: ['label.color', 'label.color'],
  });

  return (
    <Row gutter={[16, 16]} className={Style.middleChartPanel}>
      <Col xs={12} xl={9}>
        <SeriesLineChart
          title='成本走势'
          subTitle='( 元 )'
          datePicker={true}
          step={'1h'}
          lineStyle={LineStyle.Area}
          lines={[
            {
              name: 'Nodes-Monthly-Estimated-Costs',
              query: `sum (
    avg(
        avg_over_time(node_total_hourly_cost[1h])
        * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)
    )

by (node)) * 730 * (100.0/100.0)`,
            },
            {
              name: 'Total-Requests-Monthly-Estimated-Costs',
              query: `sum(
  sum(
    sum(kube_pod_container_resource_requests{resource="cpu", unit="core"}) by (container, pod, node, namespace)
    * on (node) group_left()
    avg(
      avg_over_time(node_cpu_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet",label_node_kubernetes_io_instance_type!~"eklet"}) by (node)
    ) by (node)

)

+

sum(
  sum(kube_pod_container_resource_requests{resource="memory", unit="byte", namespace!=""} / 1024./ 1024. / 1024.) by (container, pod, node, namespace)
  * on (node) group_left()
  avg(
    avg_over_time(node_ram_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)
  ) by (node)

)
) * 730 * (100.0/100.0)`,
            },
          ]}
        ></SeriesLineChart>
      </Col>
      <Col xs={12} xl={3}>
        <Card title='命名空间成本分布' subtitle='2021-12'>
          <ReactEcharts option={dynamicPieChartOption} notMerge={true} lazyUpdate={true} />
        </Card>
      </Col>
    </Row>
  );
};

export default React.memo(MiddleChart);
