import moment from 'moment';

import { QueryWindow } from '../models';

export const rangeMap = {
  [QueryWindow.LAST_1_DAY]: [moment().startOf('day'), moment()],
  [QueryWindow.LAST_7_DAY]: [moment().subtract(7, 'd').startOf('day'), moment()],
  [QueryWindow.LAST_30_DAY]: [moment().subtract(30, 'd').startOf('day'), moment()]
};
