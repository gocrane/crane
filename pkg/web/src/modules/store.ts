import { clusterApi } from '../services/clusterApi';
import { grafanaApi } from '../services/grafanaApi';
import { namespaceApi } from '../services/namespaceApi';
import { recommendationRuleApi } from '../services/recommendationRuleApi';
import { configReducer } from './configSlice';
import { editClusterReducer } from './editClusterSlice';
import global from './global';
import { insightReducer } from './insightSlice';
import { overviewReducer } from './overviewSlice';
import { configureStore, combineReducers } from '@reduxjs/toolkit';
import { TypedUseSelectorHook, useSelector, useDispatch } from 'react-redux';
import { recommendationApi } from '../services/recommendationApi';
import { prometheusApi } from '../services/prometheusApi';

const reducer = combineReducers({
  global,
  insight: insightReducer,
  overview: overviewReducer,
  editCluster: editClusterReducer,
  config: configReducer,

  [clusterApi.reducerPath]: clusterApi.reducer,
  [grafanaApi.reducerPath]: grafanaApi.reducer,
  [namespaceApi.reducerPath]: namespaceApi.reducer,
  [recommendationRuleApi.reducerPath]: recommendationRuleApi.reducer,
  [recommendationApi.reducerPath]: recommendationApi.reducer,
  [prometheusApi.reducerPath]: prometheusApi.reducer,
});

export const store = configureStore({
  reducer,
  devTools: process.env.NODE_ENV === 'development',
  middleware: (getDefaultMiddleware) => {
    const middlewares = getDefaultMiddleware({
      serializableCheck: false,
    });
    middlewares.push(
      clusterApi.middleware,
      grafanaApi.middleware,
      namespaceApi.middleware,
      recommendationRuleApi.middleware,
      recommendationApi.middleware,
      prometheusApi.middleware,
    );
    return middlewares;
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

export const useAppDispatch = () => useDispatch<AppDispatch>();
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector;

export default store;
