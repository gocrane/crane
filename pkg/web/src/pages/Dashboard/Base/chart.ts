import type { EChartOption } from 'echarts';
import { getChartDataSet, ONE_WEEK_LIST } from 'utils/chart';

export const getLineChartOptions = (dateTime: Array<string> = []): EChartOption => {
  const [timeArray, inArray, outArray] = getChartDataSet(dateTime);
  return {
    tooltip: {
      trigger: 'item',
    },
    grid: {
      left: '0',
      right: '20px',
      top: '5px',
      bottom: '36px',
      containLabel: true,
    },
    legend: {
      left: 'center',
      bottom: '0',
      orient: 'horizontal', // legend 横向布局。
      data: ['本月', '上月'],
      textStyle: {
        fontSize: 12,
      },
    },
    xAxis: {
      type: 'category',
      data: timeArray,
      boundaryGap: false,
      axisLine: {
        lineStyle: {
          color: '#E3E6EB',
          width: 1,
        },
      },
    },
    yAxis: {
      type: 'value',
    },
    series: [
      {
        name: '本月',
        data: outArray,
        type: 'line',
        smooth: false,
        showSymbol: true,
        symbol: 'circle',
        symbolSize: 8,
        itemStyle: {
          borderWidth: 1,
        },
        areaStyle: {
          color: '#0053D92F',
        },
      },
      {
        name: '上月',
        data: inArray,
        type: 'line',
        smooth: false,
        showSymbol: true,
        symbol: 'circle',
        symbolSize: 8,
        itemStyle: {
          borderWidth: 1,
        },
      },
    ],
  };
};

export const getPieChartOptions = (radius = 42): EChartOption => ({
  tooltip: {
    trigger: 'item',
  },
  grid: {
    top: '0',
    right: '0',
  },
  legend: {
    itemWidth: 12,
    itemHeight: 4,
    textStyle: {
      fontSize: 12,
    },
    left: 'center',
    bottom: '0',
    orient: 'horizontal', // legend 横向布局。
  },
  series: [
    {
      name: '销售渠道',
      type: 'pie',
      radius: ['48%', '60%'],
      avoidLabelOverlap: false,
      silent: true,
      itemStyle: {
        borderWidth: 1,
      },
      label: {
        show: true,
        position: 'center',
        formatter: ['{value|{d}%}', '{name|{b}渠道占比}'].join('\n'),
        rich: {
          value: {
            fontSize: 28,
            fontWeight: 'normal',
            lineHeight: 46,
          },
          name: {
            color: '#909399',
            fontSize: 12,
            lineHeight: 14,
          },
        },
      },
      labelLine: {
        show: false,
      },
      data: [
        { value: 1048, name: '线上' },
        { value: radius * 7, name: '门店' },
      ],
    },
  ],
});

export const getBarChartOptions = (dateTime: Array<string> = []): EChartOption => {
  const [timeArray, inArray, outArray] = getChartDataSet(dateTime);
  return {
    tooltip: {
      trigger: 'item',
    },
    xAxis: {
      type: 'category',
      data: timeArray,
      axisLine: {
        lineStyle: {
          width: 1,
        },
      },
    },
    yAxis: {
      type: 'value',
    },
    grid: {
      top: '5%',
      left: '25px',
      right: 0,
      bottom: '60px',
    },
    legend: {
      icon: 'rect',
      itemWidth: 12,
      itemHeight: 4,
      itemGap: 48,
      textStyle: {
        fontSize: 12,
        color: 'rgba(0, 0, 0, 0.6)',
      },
      left: 'center',
      bottom: '0',
      orient: 'horizontal',
      data: ['本月', '上月'],
    },
    series: [
      {
        name: '本月',
        data: outArray,
        type: 'bar',
      },
      {
        name: '上月',
        data: inArray,
        type: 'bar',
      },
    ],
  };
};

// PieChartIcon Data
export const MICRO_CHART_OPTIONS_LINE: EChartOption = {
  xAxis: {
    type: 'category',
    show: false,
    data: ONE_WEEK_LIST,
  },
  yAxis: {
    show: false,
    type: 'value',
  },
  grid: {
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    tooltip: {
      show: false,
    },
  },
  color: ['#fff'],
  series: [
    {
      data: [150, 230, 224, 218, 135, 147, 260],
      type: 'line',
      showSymbol: false,
    },
  ],
};

// BarChartIcon Data
export const MICRO_CHART_OPTIONS_BAR: EChartOption = {
  xAxis: {
    type: 'category',
    show: false,
    data: ONE_WEEK_LIST,
  },
  yAxis: {
    show: false,
    type: 'value',
  },
  grid: {
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    tooltip: {
      show: false,
    },
  },
  series: [
    {
      data: [
        100,
        130,
        184,
        218,
        {
          value: 135,
          itemStyle: {
            opacity: 0.2,
          },
        },
        {
          value: 118,
          itemStyle: {
            opacity: 0.2,
          },
        },
        {
          value: 60,
          itemStyle: {
            opacity: 0.2,
          },
        },
      ],
      type: 'bar',
      barWidth: 9,
    },
  ],
};
