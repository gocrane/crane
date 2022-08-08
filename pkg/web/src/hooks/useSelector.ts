import { shallowEqual, useSelector as useRawSelector } from 'react-redux';

import { RootState } from 'modules/store';

export const useSelector = <T>(selector: (state: RootState) => T): T =>
  useRawSelector((state) => selector(state as RootState), shallowEqual);
