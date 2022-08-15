import React from 'react';
import {Col, Row} from 'tdesign-react';
import Style from './MiddleChart.module.less';
import SeriesLineChart, {LineStyle} from '../../../../components/SeriesLineChart';
import PieChart from '../../../../components/PieChart';
import {useTranslation} from 'react-i18next';


const MiddleChart = () => {

  const {t} = useTranslation();

  return (
    <Row gutter={[16, 16]} className={Style.middleChartPanel}>
      <Col xs={12} xl={9}>
        <SeriesLineChart
          title={t('成本走势')}
          subTitle={t('( 元 )')}
          datePicker={true}
          step={'1h'}
          tips={t('过去一段时间内的成本走势图')}
          lines={[
            {
              name: t('总成本'),
              query: `sum(sum_over_time(node:node_total_hourly_cost:avg[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node))`,
            },
            {
              name: t('申请资源成本'),
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
) * (100.0/100.0)`,
            },
          ]}
        ></SeriesLineChart>
      </Col>
      <Col xs={12} xl={3}>
        <PieChart
          title={t('命名空间成本分布')}
          datePicker={true}
          step={'24h'}
          query={`sum(sum_over_time(namespace:container_cpu_usage_costs_hourly:sum_rate{}[{DURATION}m]) + sum_over_time(namespace:container_memory_usage_costs_hourly:sum_rate{}[{DURATION}m])) by (namespace) * (100.0/100.0)`}
        ></PieChart>
      </Col>
    </Row>
  );
}

export default React.memo(MiddleChart);