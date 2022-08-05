import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

type FetchRecommendationRuleListArgs = { craneUrl?: string };

export interface metadata {
  name: string;
  uid?: string;
  resourceVersion?: string;
  generation?: number;
  creationTimestamp?: string;
  managedFields?: any;
}

interface RecommendationRuleSimpleInfo {
  kind: string;
  apiVersion: string;
  metadata?: metadata;
}

interface FetchRecommendationRuleResult {
  error?: string;
  data: {
    metadata?: metadata;
    items: RecommendationRuleSimpleInfo[];
  };
}

interface UpdateRecommendationRuleArgs {
  craneUrl: string;
  data: RecommendationRuleSimpleInfo;
}

interface DeleteRecommendationRuleArgs {
  craneUrl: string;
  recommendationRuleName: string | undefined;
}

interface AddRecommendationRuleArgs {
  craneUrl: string;
  data: RecommendationRuleSimpleInfo;
}
const URI = '/api/v1/recommendationRule';

export const recommendationRuleApi = createApi({
  reducerPath: 'recommendationRuleApi',
  tagTypes: ['recommendationRuleList'],
  baseQuery: fetchBaseQuery({
    cache: 'no-cache',
    baseUrl: ``,
    prepareHeaders: (headers, api) => {
      headers.set('Content-Type', 'application/json');
      return headers;
    },
  }),
  endpoints: (builder) => ({
    deleteRecommendationRule: builder.mutation<any, DeleteRecommendationRuleArgs>({
      invalidatesTags: ['recommendationRuleList'],
      query: (args) => ({
        method: 'delete',
        url: `${args.craneUrl}${URI}/${args.recommendationRuleName}`,
      }),
    }),
    updateRecommendationRule: builder.mutation<any, UpdateRecommendationRuleArgs>({
      invalidatesTags: ['recommendationRuleList'],
      query: (args) => ({
        body: args.data,
        method: 'put',
        url: `${args.craneUrl}${URI}/${args.data?.metadata?.name}`,
      }),
    }),
    addRecommendationRule: builder.mutation<any, AddRecommendationRuleArgs>({
      invalidatesTags: ['recommendationRuleList'],
      query: (args) => ({
        body: args.data,
        method: 'post',
        url: `${args.craneUrl}${URI}`,
      }),
    }),
    fetchRecommendationRuleList: builder.query<FetchRecommendationRuleResult, FetchRecommendationRuleListArgs>({
      providesTags: ['recommendationRuleList'],
      query: (args) => ({
        url: `${args.craneUrl}${URI}`,
        method: 'get',
      }),
    }),
    fetchRecommendationRuleListMu: builder.mutation<FetchRecommendationRuleResult, FetchRecommendationRuleListArgs>({
      query: (args) => ({
        url: `${args.craneUrl}${URI}`,
        method: 'get',
      }),
    }),
  }),
});

export const {
  useUpdateRecommendationRuleMutation,
  useLazyFetchRecommendationRuleListQuery,
  useFetchRecommendationRuleListQuery,
  useFetchRecommendationRuleListMuMutation,
  useDeleteRecommendationRuleMutation,
  useAddRecommendationRuleMutation,
} = recommendationRuleApi;
