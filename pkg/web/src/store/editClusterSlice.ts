import { v4 } from 'uuid';

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface EditClusterState {
  mode: 'update' | 'create';
  modalVisible: boolean;

  clusters: Array<{ id: string; clusterId: string; clusterName: string; craneUrl: string }>;
}

type Cluster = EditClusterState['clusters'][0];

const initialEditClusterState: EditClusterState = {
  mode: 'create',
  modalVisible: false,
  clusters: [
    {
      id: v4(),
      clusterId: '',
      clusterName: '',
      craneUrl: ''
    }
  ]
};

const slice = createSlice({
  name: 'editCluster',
  initialState: initialEditClusterState,
  reducers: {
    setClusters: (state, action: PayloadAction<Cluster[]>) => {
      state.clusters = action.payload;
    },
    mode: (state, action: PayloadAction<'update' | 'create'>) => {
      state.mode = action.payload;
    },
    addCluster: state => {
      state.clusters = [
        ...(state.clusters ?? []),
        {
          id: v4(),
          clusterId: '',
          clusterName: '',
          craneUrl: ''
        }
      ];
    },
    updateCluster: (state, action: PayloadAction<{ id: string; data: Partial<Omit<Cluster, 'id'>> }>) => {
      state.clusters = state.clusters.map(cluster => {
        if (cluster.id === action.payload.id) {
          return {
            ...cluster,
            ...(action.payload.data ?? {})
          };
        } else return cluster;
      });
    },
    deleteCluster: (state, action: PayloadAction<{ id: string }>) => {
      state.clusters = state.clusters.filter(cluster => cluster.id !== action.payload.id);
    },
    resetCluster: state => {
      state.clusters = initialEditClusterState.clusters;
    },
    modalVisible: (state, action: PayloadAction<boolean>) => {
      state.modalVisible = action.payload;
    }
  }
});

export const editClusterActions = slice.actions;
export const editClusterReducer = slice.reducer;
