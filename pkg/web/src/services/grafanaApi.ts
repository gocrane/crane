import { buildRetryFetchBaseQuery } from './retryFetchBaseQuery';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

type FetchDashboardDetailArgs = {
  dashboardUid: string;
  craneUrl?: string;
};

interface FetchDashboardListArgs {
  craneUrl: string | undefined;
}

export const grafanaApi = createApi({
  reducerPath: 'grafanaApi',
  baseQuery: buildRetryFetchBaseQuery(
    fetchBaseQuery({
      baseUrl: '',
      timeout: 15000,
      prepareHeaders: (headers, api) => headers,
      fetchFn: (input, init) => fetch(input, { ...init }),
    }),
  ),
  endpoints: (builder) => ({
    fetchDashboardList: builder.query<any, FetchDashboardListArgs>({
      query: (args) => ({
        url: `${args.craneUrl ?? ''}/grafana/api/search`,
        method: 'get',
      }),
    }),
    fetchDashboardDetail: builder.query<any, FetchDashboardDetailArgs>({
      query: (args) => ({
        url: `${args.craneUrl ?? ''}/grafana/api/dashboards/uid/${args.dashboardUid}`,
        method: 'get',
      }),
    }),
  }),
});

export const {
  useLazyFetchDashboardListQuery,
  useLazyFetchDashboardDetailQuery,
  useFetchDashboardListQuery,
  useFetchDashboardDetailQuery,
} = grafanaApi;
