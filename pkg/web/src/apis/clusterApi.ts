import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

import { ClusterSimpleInfo } from '../models';

interface AddClusterArgs {
  data: {
    clusters: Array<{ name: string; craneUrl: string }>;
  };
}

type FetchClusterListArgs = void | {};

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
  clusterId: string;
}

export const clusterApi = createApi({
  reducerPath: 'clusterApi',
  tagTypes: ['clusterList'],
  baseQuery: fetchBaseQuery({
    cache: 'no-cache',
    baseUrl: `/api/v1/cluster`,
    prepareHeaders: (headers, api) => {
      headers.set('Content-Type', 'application/json');
      return headers;
    }
  }),
  endpoints: builder => ({
    deleteCluster: builder.mutation<any, DeleteClusterArgs>({
      invalidatesTags: ['clusterList'],
      query: args => {
        return {
          method: 'delete',
          url: `/${args.clusterId}`
        };
      }
    }),
    updateCluster: builder.mutation<any, UpdateClusterArgs>({
      invalidatesTags: ['clusterList'],
      query: args => {
        return {
          body: args.data,
          method: 'put',
          url: `/${args.data.id}`
        };
      }
    }),
    addClusters: builder.mutation<any, AddClusterArgs>({
      invalidatesTags: ['clusterList'],
      query: args => {
        return {
          body: args.data,
          method: 'post',
          url: ''
        };
      }
    }),
    fetchClusterList: builder.query<FetchClusterListResult, FetchClusterListArgs>({
      providesTags: ['clusterList'],
      query: () => {
        return {
          url: '',
          method: 'get'
        };
      }
    })
  })
});
