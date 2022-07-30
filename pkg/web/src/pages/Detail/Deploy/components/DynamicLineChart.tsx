import React, { useEffect, useState } from 'react';
import { EChartOption } from 'echarts';
import ReactEcharts from 'echarts-for-react';
import useDynamicChart from 'hooks/useDynamicChart';
import { getLineOptions } from '../chart';

const DynamicLineChart = () => {
  const [lineOptions, setLineOptions] = useState<EChartOption>(getLineOptions());
  const dynamicLineChartOptions = useDynamicChart(lineOptions, {
    placeholderColor: ['legend.textStyle.color', 'xAxis.axisLabel.color', 'yAxis.axisLabel.color'],
  });

  useEffect(() => {
    const timer = setInterval(() => setLineOptions(getLineOptions()), 3000);
    return () => {
      clearInterval(timer);
    };
  }, []);

  return (
    <ReactEcharts
      option={dynamicLineChartOptions} // option：图表配置项
      notMerge={true}
      lazyUpdate={true}
      style={{ height: 265 }}
    />
  );
};

export default React.memo(DynamicLineChart);
