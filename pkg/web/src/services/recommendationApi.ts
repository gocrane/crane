import { fetchBaseQuery, createApi } from '@reduxjs/toolkit/query/react';
import { parse, stringify } from 'yaml';

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
  tagTypes: ['recommendation'],
  baseQuery: fetchBaseQuery({
    cache: 'no-cache',
    baseUrl: ``,
    prepareHeaders: (headers, api) => {
      headers.set('Content-Type', 'application/json');
      return headers;
    },
  }),
  endpoints: (builder) => ({
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
          return value;
        });
        return res;
      },
    }),
  }),
});

export const { useFetchRecommendationListQuery, useLazyFetchRecommendationListQuery } = recommendationApi;
