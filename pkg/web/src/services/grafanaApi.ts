import { buildRetryFetchBaseQuery } from './retryFetchBaseQuery';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import queryString from "query-string";

type FetchDashboardDetailArgs = {
  dashboardUid: string;
  craneUrl?: string;
};

interface FetchDashboardListArgs {
  craneUrl: string | undefined;
}

interface FetchSeriesListArgs {
  craneUrl: string | undefined;
  match: string;
  namespace: string;
  start: string;
  end: string;
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
    fetchSeriesList: builder.query<any, FetchSeriesListArgs>({
      query: (args) => {
        // trans to second
        const url = queryString.stringifyUrl({
          url: `${args.craneUrl ?? ''}/grafana/api/datasources/1/resources/api/v1/series`,
          query: {
            match: args.match,
            namespace: args.namespace,
            start: args.start,
            end: args.end,
          },
        });
        return {
          url,
          method: 'get',
        };
      },
    }),
  }),
});

export const {
  useLazyFetchDashboardListQuery,
  useLazyFetchDashboardDetailQuery,
  useFetchDashboardListQuery,
  useFetchDashboardDetailQuery,
  useFetchSeriesListQuery,
} = grafanaApi;
