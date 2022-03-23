import dayjs from 'dayjs';
import { stringify } from 'query-string';

import React from 'react';

import { useIsNeedSelectNamespace } from './useIsNeedSelectNamespace';
import { useSelector } from './useSelector';

export const useGrafanaQueryStr = ({ panelId }: { panelId: string }) => {
  const customRange = useSelector(state => state.insight.customRange);
  const selectedNamespace = useSelector(state => state.insight.selectedNamespace);
  const discount = useSelector(state => state.insight.discount);

  const isNeedSelectNamespace = useIsNeedSelectNamespace();

  const [from, to] = React.useMemo(
    () => [dayjs(customRange.start).toDate().getTime(), dayjs(customRange.end).toDate().getTime()],
    [customRange.end, customRange.start]
  );

  let query: any = React.useMemo(
    () => ({
      orgId: '1',
      from,
      to,
      theme: 'light',
      panelId
    }),
    [from, panelId, to]
  );

  if (discount) {
    query = { ...query, ['var-discount']: discount };
  }

  if (isNeedSelectNamespace && selectedNamespace) {
    query = { ...query, ['var-namespace']: selectedNamespace };
  }

  return React.useMemo(() => stringify(query), [query]);
};
