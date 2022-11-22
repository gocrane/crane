import { buildRetryFetchBaseQuery } from './retryFetchBaseQuery';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { parse } from 'yaml';
import { RecommendationRuleSimpleInfo } from './recommendationRuleApi';

interface ownerReference {
  apiVersion: string;
  kind: string;
  name: string;
  uid: string;
  controller: boolean;
  blockOwnerDeletion: boolean;
}

interface metadata {
  name: string;
  uid?: string;
  resourceVersion?: string;
  generation?: number;
  creationTimestamp?: string;
  managedFields?: any;
  labels?: { [key: string]: string };
  ownerReferences?: ownerReference[];
}

interface TargetRef {
  kind: string;
  namespace: string;
  name: string;
  apiVersion: string;
}

enum CompletionStrategyType {
  Once = 'Once',
  Periodical = 'Periodical',
}

export enum RecommendationType {
  Replicas = 'Replicas',
  Resource = 'Resource',
  IdleNode = 'IdleNode',
}

enum AdoptionType {
  StatusAndAnnotation = 'StatusAndAnnotation',
}

enum ResourceTargetType {
  Utilization = 'Utilization',
}

interface Metrics {
  type: RecommendationType;
  resource: {
    name: string;
    target: {
      averageUtilization: number;
      type: ResourceTargetType;
    };
  };
}

interface ResourceRequest {
  containers?: {
    containerName: string;
    target: {
      cpu: string;
      memory: string;
    };
  }[];
}

interface RecommendedValue {
  // Resource
  resourceRequest?: ResourceRequest;
  // Replicas
  replicasRecommendation?: {
    replicas: number;
  };
  effectiveHPA?: {
    maxReplicas: number;
    minReplicas: number;
    metrics?: Metrics[];
  };
}

export interface RecommendationSimpleInfo {
  kind: string;
  apiVersion: string;
  metadata: metadata;
  spec: {
    targetRef?: TargetRef;
    type: RecommendationType;
    completionStrategy: {
      completionStrategyType: CompletionStrategyType;
    };
    adoptionType: AdoptionType;
  };
  status: {
    recommendedValue?: RecommendedValue | string;
    lastUpdateTime?: string;
  };
  workloadType: string;
  name: string;
  namespace: string;
}

interface AdoptRecommendationArgs {
  craneUrl: string;
  namespace?: string;
  name?: string;
}

interface FetchRecommendationArgs {
  craneUrl: string;
  recommendationType: RecommendationType;
}

interface FetchRecommendationResult {
  error?: string;
  data: {
    metadata?: metadata;
    items: RecommendationSimpleInfo[];
  };
}

const URI = '/api/v1/recommendation';

export const recommendationApi = createApi({
  reducerPath: 'recommendationApi',
  tagTypes: ['recommendation', 'idleNode'],
  baseQuery: buildRetryFetchBaseQuery(
    fetchBaseQuery({
      cache: 'no-cache',
      baseUrl: ``,
      timeout: 15000,
      prepareHeaders: (headers, api) => {
        headers.set('Content-Type', 'application/json');
        return headers;
      },
    }),
  ),
  endpoints: (builder) => ({
    adoptRecommendation: builder.mutation<any, AdoptRecommendationArgs>({
      invalidatesTags: ['recommendation'],
      query: (args) => ({
        method: 'post',
        url: `${args.craneUrl}${URI}/adopt/${args.namespace}/${args.name}`,
      }),
    }),
    fetchRecommendationList: builder.query<FetchRecommendationResult, FetchRecommendationArgs>({
      providesTags: ['recommendation'],
      query: (args) => ({
        url: `${args.craneUrl}${URI}`,
        method: 'get',
      }),
      transformResponse: (res: FetchRecommendationResult, meta, arg: FetchRecommendationArgs) => {
        res.data.items = res.data.items.filter((value) => arg.recommendationType === value.spec.type);
        res.data.items.map((value) => {
          const recommendedValue = value?.status?.recommendedValue;
          if (typeof recommendedValue === 'string' && recommendedValue.length > 0) {
            if (typeof value.status.recommendedValue === 'string') {
              value.status.recommendedValue = parse(value.status.recommendedValue);
            }
          }
          if (value?.metadata?.name) value.name = value?.metadata?.name;
          if (value?.metadata?.managedFields) delete value.metadata.managedFields;
          if (value?.spec.targetRef?.namespace) value.namespace = value?.spec.targetRef?.namespace;
          if (value?.spec.targetRef?.kind) value.workloadType = value?.spec.targetRef?.kind;
          return value;
        });
        return res;
      },
    }),
  }),
});

export const { useFetchRecommendationListQuery, useLazyFetchRecommendationListQuery, useAdoptRecommendationMutation } =
  recommendationApi;
