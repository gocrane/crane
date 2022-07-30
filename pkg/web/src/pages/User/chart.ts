export const visitData = {
  tooltip: {
    trigger: 'axis',
  },
  legend: {
    data: ['杯子', '茶叶', '蜂蜜', '面粉'],
  },
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    containLabel: true,
  },
  toolbox: {
    feature: {
      saveAsImage: {},
    },
  },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
  },
  yAxis: {
    type: 'value',
  },
  series: [
    {
      name: '杯子',
      type: 'line',
      data: [21, 99, 56, 66, 55, 7, 83],
    },
    {
      name: '茶叶',
      type: 'line',
      data: [84, 30, 70, 14, 19, 75, 73],
    },
    {
      name: '蜂蜜',
      type: 'line',
      data: [57, 3, 25, 13, 49, 80, 11],
    },
    {
      name: '面粉',
      type: 'line',
      data: [8, 85, 2, 77, 10, 65, 90],
    },
  ],
};
