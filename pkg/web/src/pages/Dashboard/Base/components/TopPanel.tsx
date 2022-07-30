import React, { memo } from 'react';
import { Col, Row } from 'tdesign-react';
import Board, { ETrend, IBoardProps, TimeType } from 'components/BoardChart';

const PANE_LIST: Array<IBoardProps> = [
  {
    title: 'Nodes Monthly Estimated Costs',
    countPrefix: '¥ ',
    trend: ETrend.up,
    trendNum: '20.5%',
    lineColor: '#fff',
    // Icon: <PieChartIcon />,
    query: `sum (
    avg(
        avg_over_time(node_total_hourly_cost[1h])
        * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)
    )

by (node)) * 730 * (100/100.0)`,
    timeType: TimeType.Range,
  },
  {
    title: 'Total Requests Monthly Estimated Costs',
    countPrefix: '¥ ',
    trend: ETrend.up,
    trendNum: '20.5%',
    // Icon: <PieChartIcon />,
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
    timeType: TimeType.Range,
  },
  {
    title: 'Total Usage Monthly Estimated Costs',
    countPrefix: '¥ ',
    trend: ETrend.up,
    trendNum: '20.5%',
    // Icon: <PieChartIcon />,
    query: `sum (
  sum(label_replace(irate(container_cpu_usage_seconds_total{container!="POD", container!="",image!=""}[1h]), "node", "$1", "instance",  "(.*)")) by (container, pod, node, namespace)
  * on (node) group_left()
  avg(
    avg_over_time(node_cpu_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet",label_node_kubernetes_io_instance_type!~"eklet"}
    ) by (node)
  ) by (node)

+

  sum(label_replace(avg_over_time(container_memory_working_set_bytes{container!="POD",container!="",image!=""}[1h]), "node", "$1", "instance",  "(.*)")) by (container, pod, node, namespace) / 1024.0 / 1024.0 / 1024.0
  * on (node) group_left()
  avg(
    avg_over_time(node_ram_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}
    ) by (node)
  ) by (node)
) * 730 * (100./100.)`,
    timeType: TimeType.Range,
  },
  {
    title: 'Cpu Requests Monthly Estimated Costs',
    query: `sum(
  sum(kube_pod_container_resource_requests{resource="cpu", unit="core"}) by (container, pod, node, namespace)
  * on (node) group_left()
  avg(
      avg_over_time(node_cpu_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet",label_node_kubernetes_io_instance_type!~"eklet"}
      ) by (node)
  ) by (node)
) * 730 * (100./100.)`,
    countPrefix: '¥ ',
    trend: ETrend.down,
    trendNum: '20.5%',
    timeType: TimeType.Range,
  },
  {
    title: 'Cpu Usage Monthly Estimated Costs',
    query: `sum(
  sum(label_replace(irate(container_cpu_usage_seconds_total{container!="POD", container!="",image!=""}[1h]), "node", "$1", "instance",  "(.*)")) by (container, pod, node, namespace) * on (node) group_left()
  avg(
    avg_over_time(node_cpu_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}
    ) by (node)
  ) by (node)
) * 730 * (100./100.)`,
    countPrefix: '¥ ',
    trend: ETrend.down,
    trendNum: '20.5%',
    timeType: TimeType.Range,
  },
  {
    title: 'Ram Requests Monthly Estimated Costs',
    query: `sum(
  sum(kube_pod_container_resource_requests{resource="memory", unit="byte", namespace!=""} / 1024./ 1024. / 1024.) by (container, pod, node, namespace) * on (node) group_left()
  avg(
    avg_over_time(node_ram_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}
    ) by (node)
  ) by (node)
) * 730 * (100./100.)`,
    countPrefix: '¥ ',
    trend: ETrend.down,
    trendNum: '20.5%',
    timeType: TimeType.Range,
  },
  {
    title: 'Ram Usage Monthly Estimated Costs',
    query: `sum(
  sum(label_replace(avg_over_time(container_memory_working_set_bytes{container!="POD",container!="",image!=""}[1h]), "node", "$1", "instance",  "(.*)")) by (container, pod, node, namespace) / 1024.0 / 1024.0 / 1024.0 * on (node) group_left()
    avg(
    avg_over_time(node_ram_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}
    ) by (node)
  ) by (node)
) * 730 * (100./100.)`,
    countPrefix: '¥ ',
    trend: ETrend.down,
    trendNum: '20.5%',
    timeType: TimeType.Range,
  },
];

const TopPanel = () => (
  <Row gutter={[16, 16]}>
    {PANE_LIST.map((item, index) => (
      <Col key={item.title} xs={6} xl={3}>
        <Board
          title={item.title}
          trend={item.trend}
          trendNum={item.trendNum}
          count={item.count}
          countPrefix={item.countPrefix}
          lineColor={item.lineColor}
          desc={'自从上周以来'}
          Icon={item.Icon}
          dark={index === 0}
          query={item.query}
          timeType={item.timeType}
          start={item.start}
          end={item.end}
          step={item.step}
        />
      </Col>
    ))}
  </Row>
);

export default React.memo(TopPanel);
