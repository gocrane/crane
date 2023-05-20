import { buildRetryFetchBaseQuery } from './retryFetchBaseQuery';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

export interface FetchNamespaceListArgs {
  clusterId?: string;
}

export interface FetchNamespaceListResult {
  error?: any;
  data: {
    totalCount: number;
    items: Array<string>;
  };
}

export const namespaceApi = createApi({
  reducerPath: 'namespaceApi',
  tagTypes: ['namespaceList'],
  baseQuery: buildRetryFetchBaseQuery(
    fetchBaseQuery({
      baseUrl: '/api/v1/namespaces',
      timeout: 15000,
      prepareHeaders: (headers, _api) => {
        headers.set('Content-Type', 'application/json');
        headers.set('Accept', 'application/json');
        return headers;
      },
    }),
  ),
  endpoints: (builder) => ({
    fetchNamespaceList: builder.query<FetchNamespaceListResult, FetchNamespaceListArgs>({
      query: (args) => ({
        url: `/${args.clusterId}`,
      }),
    }),
  }),
});

export const { useLazyFetchNamespaceListQuery, useFetchNamespaceListQuery } = namespaceApi;
