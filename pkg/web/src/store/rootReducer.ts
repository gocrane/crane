import { combineReducers } from 'redux';

import { grafanaApi } from '../apis/grafanaApi';
import { clusterApi } from './../apis/clusterApi';
import { namespaceApi } from './../apis/namespaceApi';
import { configReducer } from './configSlice';
import { editClusterReducer } from './editClusterSlice';
import { insightReducer } from './insightSlice';
import { overviewReducer } from './overviewSlice';

export const rootReducer = combineReducers({
  insight: insightReducer,
  overview: overviewReducer,
  editCluster: editClusterReducer,
  config: configReducer,

  [clusterApi.reducerPath]: clusterApi.reducer,
  [grafanaApi.reducerPath]: grafanaApi.reducer,
  [namespaceApi.reducerPath]: namespaceApi.reducer
});
