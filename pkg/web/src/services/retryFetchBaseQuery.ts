import { BaseQueryFn, retry } from '@reduxjs/toolkit/query/react';

/**
 * BuildRetryFetchBaseQuery
 * https://redux-toolkit.js.org/rtk-query/usage/customizing-queries#automatic-retries
 *
 * RTK Query exports a utility called retry that you can wrap the baseQuery in your API definition with. It defaults to 5 attempts with a basic exponential backoff.
 * The default behavior would retry at these intervals:
 *
 * 600ms * random(0.4, 1.4)
 * 1200ms * random(0.4, 1.4)
 * 2400ms * random(0.4, 1.4)
 * 4800ms * random(0.4, 1.4)
 * 9600ms * random(0.4, 1.4)
 * @param fn fetchBaseQuery
 * @returns
 */
export const buildRetryFetchBaseQuery = (fn: any): BaseQueryFn => {
  const staggeredBaseQuery = retry(fn, {
    maxRetries: 10,
  });
  return staggeredBaseQuery;
};
