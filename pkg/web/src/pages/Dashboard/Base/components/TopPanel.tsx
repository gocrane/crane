import React from 'react';
import {Col, Row} from 'tdesign-react';
import Board, {IBoardProps, TimeType} from 'components/BoardChart';
import { useTranslation } from 'react-i18next';


const TopPanel = () => {
  const {t} = useTranslation();
  const PANE_LIST: Array<IBoardProps> = [
    {
      title: t('当月总成本'),
      countPrefix: '¥ ',
      lineColor: '#fff',
      query: `sum (
      avg(
          sum_over_time(node:node_total_hourly_cost:avg[30d])
          * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)
      )
  
  by (node)) * (100/100.0)`,
      timeType: TimeType.Range,
      tips: t("过去一个月集群总成本。从安装Crane时间开始，按小时累加集群成本"),
    },
    {
      title: t('预估每月成本'),
      countPrefix: '¥ ',
      query: `sum (
      avg(
          avg_over_time(node_total_hourly_cost[1h])
          * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)
      )
  
  by (node)) * 730 * (100/100.0)`,
      timeType: TimeType.Range,
      tips: t("以最近一小时成本估算未来一个月的成本。每小时成本 * 24 * 30"),
    },
    {
      title: t('预估CPU总成本'),
      query: `sum(
    sum(kube_pod_container_resource_requests{resource="cpu", unit="core"}) by (container, pod, node, namespace)
    * on (node) group_left()
    avg(
        avg_over_time(node_cpu_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet",label_node_kubernetes_io_instance_type!~"eklet"}
        ) by (node)
    ) by (node)
  ) * 730 * (100./100.)`,
      countPrefix: '¥ ',
      timeType: TimeType.Range,
      tips: t("以最近一小时CPU成本估算未来一个月的CPU成本。每小时CPU成本 * 24 * 30"),
    },
    {
      title: t('预估Memory总成本'),
      query: `sum(
    sum(kube_pod_container_resource_requests{resource="memory", unit="byte", namespace!=""} / 1024./ 1024. / 1024.) by (container, pod, node, namespace) * on (node) group_left()
    avg(
      avg_over_time(node_ram_hourly_cost[1h]) * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}
      ) by (node)
    ) by (node)
  ) * 730 * (100./100.)`,
      countPrefix: '¥ ',
      timeType: TimeType.Range,
      tips: t("以最近一小时Memory成本估算未来一个月的Memory成本。每小时Memory成本 * 24 * 30"),
    },
  ];

  return <Row gutter={[16, 16]}>
    {PANE_LIST.map((item, index) => (
      <Col key={item.title} xs={6} xl={3}>
        <Board
          title={item.title}
          trend={item.trend}
          trendNum={item.trendNum}
          count={item.count}
          countPrefix={item.countPrefix}
          lineColor={item.lineColor}
          desc={t('自从上周以来')}
          Icon={item.Icon}
          dark={index === 0}
          query={item.query}
          timeType={item.timeType}
          start={item.start}
          end={item.end}
          step={item.step}
          tips={item.tips}
        />
      </Col>
    ))}
  </Row>
};

export default React.memo(TopPanel);