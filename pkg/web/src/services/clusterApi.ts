import { buildRetryFetchBaseQuery } from './retryFetchBaseQuery';
import { ClusterSimpleInfo } from '../models';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

interface AddClusterArgs {
  data: {
    clusters: Array<{ name: string; craneUrl: string; discount: string; preinstallRecommendation: string }>;
  };
}

type FetchClusterListArgs = { craneUrl?: string };

interface FetchClusterListResult {
  error?: string;
  data: {
    totalCount: number;
    items: Array<ClusterSimpleInfo>;
  };
}

interface UpdateClusterArgs {
  data: ClusterSimpleInfo;
}

interface DeleteClusterArgs {
  clusterId: string | undefined;
}

export const clusterApi = createApi({
  reducerPath: 'clusterApi',
  tagTypes: ['clusterList'],
  baseQuery: buildRetryFetchBaseQuery(
    fetchBaseQuery({
      cache: 'no-cache',
      baseUrl: `/api/v1/cluster`,
      timeout: 15000,
      prepareHeaders: (headers, _api) => {
        headers.set('Content-Type', 'application/json');
        return headers;
      },
    }),
  ),
  endpoints: (builder) => ({
    deleteCluster: builder.mutation<any, DeleteClusterArgs>({
      invalidatesTags: ['clusterList'],
      query: (args) => ({
        method: 'delete',
        url: `/${args.clusterId}`,
      }),
    }),
    updateCluster: builder.mutation<any, UpdateClusterArgs>({
      invalidatesTags: ['clusterList'],
      query: (args) => ({
        body: args.data,
        method: 'put',
        url: `/${args.data.id}`,
      }),
    }),
    addClusters: builder.mutation<any, AddClusterArgs>({
      invalidatesTags: ['clusterList'],
      query: (args) => ({
        body: args.data,
        method: 'post',
        url: '',
      }),
    }),
    fetchClusterList: builder.query<FetchClusterListResult, FetchClusterListArgs>({
      providesTags: ['clusterList'],
      query: () => ({
        url: '',
        method: 'get',
      }),
    }),
    fetchClusterListMu: builder.mutation<FetchClusterListResult, FetchClusterListArgs>({
      query: (args) => ({
        url: `${args.craneUrl ?? ''}/api/v1/cluster`,
        method: 'get',
      }),
    }),
  }),
});

export const {
  useUpdateClusterMutation,
  useLazyFetchClusterListQuery,
  useFetchClusterListQuery,
  useFetchClusterListMuMutation,
  useDeleteClusterMutation,
  useAddClustersMutation,
} = clusterApi;
