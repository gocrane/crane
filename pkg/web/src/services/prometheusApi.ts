import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { IBoardProps } from '../components/BoardChart';
import queryString from 'query-string';

interface QueryInstantPrometheusArgs extends IBoardProps {
  time?: number;
  // pql
  query: string;
  craneUrl: string;
}

interface QueryInstantPrometheusResult {
  error: string;
  latestValue?: number;
  emptyData?: boolean;
  data: {
    metric?: any;
    value?: (number | string)[];
  }[];
}

interface QueryRangePrometheusArgs extends IBoardProps {
  start?: number;
  end?: number;
  step?: string;
  query: string;
  craneUrl: string;
}

interface QueryRangePrometheusResult {
  error: string;
  latestValue?: number;
  emptyData?: boolean;
  metricData?: (number | string)[][];
  data: {
    metric?: any;
    values?: (number | string)[][];
  }[];
}

const URI = '/api/v1/prometheus';

export const prometheusApi = createApi({
  reducerPath: 'prometheus',
  tagTypes: ['prometheus'],
  baseQuery: fetchBaseQuery({
    cache: 'no-cache',
    baseUrl: ``,
    prepareHeaders: (headers, api) => {
      headers.set('Content-Type', 'application/json');
      return headers;
    },
  }),
  endpoints: (builder) => ({
    instantPrometheus: builder.query<QueryInstantPrometheusResult, QueryInstantPrometheusArgs>({
      providesTags: ['prometheus'],
      query: (args) => {
        // trans to second
        const time = args?.time ? Math.floor(args.time / 1000) : Math.floor(Date.now() / 1000);
        const url = queryString.stringifyUrl({
          url: `${args.craneUrl}${URI}/query`,
          query: {
            time,
            query: args.query,
          },
        });
        return {
          url,
          method: 'get',
        };
      },
      transformResponse: (res: QueryInstantPrometheusResult, meta, arg: QueryInstantPrometheusArgs) => {
        if (res.data.length > 0) {
          res.data.map((value) => {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            value.value[0] *= 1000;
            value.value[1] = Math.round(value.value[1] * 100) / 100;
            res.latestValue = value.value[1];
            return value;
          });
          res.emptyData = false;
        } else {
          res.emptyData = true;
          res.metricData = [];
        }
        return res;
      },
    }),
    rangePrometheus: builder.query<QueryRangePrometheusResult, QueryRangePrometheusArgs>({
      providesTags: ['prometheus'],
      query: (args) => {
        /**
         * Default
         * start: before 1h from now
         * end: now
         * step: 15m0s
         * will be got five data point from prometheus
         * 0m -- 15m -- 30m -- 45m -- 60m
         */
        const start = args?.start ? Math.floor(args.start / 1000) : Math.floor(Date.now() / 1000) - 3600;
        const end = args?.end ? Math.floor(args.end / 1000) : Math.floor(Date.now() / 1000);
        const step = args?.step ? args.step : '15m0s';
        console.log('args', start, end, step);
        const url = queryString.stringifyUrl({
          url: `${args.craneUrl}${URI}/query_range`,
          query: {
            start,
            end,
            step,
            query: args.query,
          },
        });
        return {
          url,
          method: 'get',
        };
      },
      transformResponse: (res: QueryRangePrometheusResult, meta, args: QueryRangePrometheusArgs) => {
        // Single Value - Line
        if (res.data.length > 0) {
          res.data.map(({ values }) => {
            values?.map((value1) => {
              value1[0] *= 1000;
              value1[1] = Math.round(value1[1] * 100) / 100;
            });
            res.latestValue = values[values.length - 1][1];
          });
          res.metricData = res.data[0].values;
          res.emptyData = false;
        } else {
          res.emptyData = true;
          res.metricData = [];
        }
        return res;
      },
    }),
  }),
});

export const {
  useInstantPrometheusQuery,
  useLazyInstantPrometheusQuery,
  useLazyRangePrometheusQuery,
  useRangePrometheusQuery,
} = prometheusApi;
