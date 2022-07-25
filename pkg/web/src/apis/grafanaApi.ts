import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

type FetchDashboardDetailArgs = {
  dashboardUid: string;
  craneUrl?: string;
};

interface FetchDashboardListArgs {
  craneUrl: string;
}

export const grafanaApi = createApi({
  reducerPath: 'grafanaApi',
  baseQuery: fetchBaseQuery({
    baseUrl: '',
    prepareHeaders: (headers, api) => {
      return headers;
    },
    fetchFn: (input, init) => fetch(input, { ...init })
  }),
  endpoints: builder => ({
    fetchDashboardList: builder.query<any, FetchDashboardListArgs>({
      query: args => {
        return {
          url: `${args.craneUrl ?? ''}/grafana/api/search`,
          method: 'get'
        };
      }
    }),
    fetchDashboardDetail: builder.query<any, FetchDashboardDetailArgs>({
      query: args => {
        return {
          url: `${args.craneUrl ?? ''}/grafana/api/dashboards/uid/${args.dashboardUid}`,
          method: 'get'
        };
      }
    })
  })
});
