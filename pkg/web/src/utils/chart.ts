// 获取 chart 的 mock 数据
import dayjs, { Dayjs } from 'dayjs';

const RECENT_7_DAYS: [Dayjs, Dayjs] = [dayjs().subtract(7, 'day'), dayjs().subtract(1, 'day')];
export const ONE_WEEK_LIST = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];

export const getRandomInt = (num = 100): number => {
  const resultNum = Number((Math.random() * num).toFixed(0));
  return resultNum <= 1 ? 1 : resultNum;
};

type ChartValue = number | string;

export function getTimeArray(dateTime: string[] = [], divideNum = 10, format = 'MM-DD'): string[] {
  const timeArray = [];
  if (dateTime.length === 0) {
    dateTime.push(...RECENT_7_DAYS.map((item) => item.format(format)));
  }
  for (let i = 0; i < divideNum; i++) {
    const dateAbsTime: number = (new Date(dateTime[1]).getTime() - new Date(dateTime[0]).getTime()) / divideNum;
    const timeNode: number = new Date(dateTime[0]).getTime() + dateAbsTime * i;
    timeArray.push(dayjs(timeNode).format(format));
  }

  return timeArray;
}

export const getChartDataSet = (dateTime: Array<string> = [], divideNum = 10): ChartValue[][] => {
  const timeArray = getTimeArray(dateTime, divideNum);
  const inArray = [];
  const outArray = [];
  for (let index = 0; index < divideNum; index++) {
    inArray.push(getRandomInt().toString());
    outArray.push(getRandomInt().toString());
  }

  return [timeArray, inArray, outArray];
};
