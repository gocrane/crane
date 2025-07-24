import dayjs from 'dayjs';

import { QueryWindow } from '../models';

export const rangeMap = {
  [QueryWindow.LAST_1_DAY]: [dayjs(+dayjs() - 3600 * 24 * 1000), dayjs()],
  [QueryWindow.LAST_7_DAY]: [dayjs().subtract(7, 'd').startOf('day'), dayjs()],
  [QueryWindow.LAST_30_DAY]: [dayjs().subtract(30, 'd').startOf('day'), dayjs()],
};
