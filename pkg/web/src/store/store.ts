import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/dist/query';

import { clusterApi } from '../apis/clusterApi';
import { namespaceApi } from '../apis/namespaceApi';
import { grafanaApi } from './../apis/grafanaApi';
import { rootReducer } from './rootReducer';

export const store = configureStore({
  reducer: rootReducer,
  devTools: process.env.NODE_ENV === 'development',
  middleware: getDefaultMiddleware => {
    const middlewares = getDefaultMiddleware({
      serializableCheck: false
    });
    if (process.env.NODE_ENV === 'development') {
      const { createLogger } = require('redux-logger');
      middlewares.push(createLogger({ diff: true, collapsed: true }));
    }
    middlewares.push(clusterApi.middleware, grafanaApi.middleware, namespaceApi.middleware);
    return middlewares;
  }
});

setupListeners(store.dispatch);

export type RootState = ReturnType<typeof rootReducer>;
export type GetState = () => RootState;
