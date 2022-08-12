import React, { useState } from 'react';
import { Card, Col, Row } from 'tdesign-react';
import ReactEcharts from 'echarts-for-react';
import useDynamicChart from 'hooks/useDynamicChart';
import { useLineChartOptions, usePieChartOptions } from '../chart';
import Style from './MiddleChart.module.less';
import SeriesLineChart, { LineStyle } from '../../../../components/SeriesLineChart';
import PieChart from '../../../../components/PieChart';
import { useTranslation } from 'react-i18next';

const MiddleChart = () => {
  const { t } = useTranslation();
  return (
    <Row gutter={[16, 16]} className={Style.middleChartPanel}>
      <Col xs={12} xl={9}>
        <SeriesLineChart
          title={t('成本走势')}
          subTitle={t('( 元 )')}
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
        <PieChart
          title={t('命名空间成本分布')}
          datePicker={true}
          step={'24h'}
          query={`sum(sum_over_time(namespace:container_cpu_usage_costs_hourly:sum_rate{}[7d]) + sum_over_time(namespace:container_memory_usage_costs_hourly:sum_rate{}[7d])) by (namespace) * (100.0/100.0)`}
        ></PieChart>
      </Col>
    </Row>
  );
};

export default React.memo(MiddleChart);
