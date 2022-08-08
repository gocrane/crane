import React, { useState } from 'react';
import { Col, Row } from 'tdesign-react';
import useDynamicChart from 'hooks/useDynamicChart';
import { getLineChartOptions } from '../chart';
import Style from './CpuChart.module.less';
import SeriesLineChart, { ISeriesLineChart } from '../../../../components/SeriesLineChart';

const lineOptions = getLineChartOptions();

const item: ISeriesLineChart = {
  title: 'CPU 资源使用',
  subTitle: '',
  datePicker: true,
  timeRange: 3600,
  step: '1h',
  xAxis: { type: 'time' },
  lines: [
    {
      name: 'capacity',
      query: `SUM(kube_node_status_capacity{resource="cpu", unit="core"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node))`,
    },
    {
      name: 'request',
      query: `SUM(kube_pod_container_resource_requests{resource="cpu", unit="core"}  * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!="eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node))`,
    },
  ],
};

const CpuChart = () => {
  const [customOptions, setCustomOptions] = useState(lineOptions);

  const onTimeChange = (value: Array<string>) => {
    const options = getLineChartOptions(value);
    setCustomOptions(options);
  };

  const dynamicLineChartOption = useDynamicChart(customOptions, {
    placeholderColor: ['legend.textStyle.color', 'xAxis.axisLabel.color', 'yAxis.axisLabel.color'],
    borderColor: ['series.0.itemStyle.borderColor', 'series.1.itemStyle.borderColor'],
  });

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
