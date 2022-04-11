import { v4 } from 'uuid';

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface EditClusterState {
  mode: 'update' | 'create';
  modalVisible: boolean;

  clusters: Array<{ id: string; clusterName: string; craneUrl: string }>;

  editingClusterId: string | null;
}

type Cluster = EditClusterState['clusters'][0];

const defaultCluster = {
  id: v4(),
  clusterName: '',
  craneUrl: ''
};

const initialEditClusterState: EditClusterState = {
  mode: 'create',
  modalVisible: false,
  editingClusterId: null,
  clusters: [{ ...defaultCluster }]
};

const slice = createSlice({
  name: 'editCluster',
  initialState: initialEditClusterState,
  reducers: {
    editingClusterId: (state, action: PayloadAction<string>) => {
      state.editingClusterId = action.payload;
    },
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
      state.clusters = [{ ...defaultCluster, id: v4() }];
    },
    modalVisible: (state, action: PayloadAction<boolean>) => {
      state.modalVisible = action.payload;
    }
  }
});

export const editClusterActions = slice.actions;
export const editClusterReducer = slice.reducer;
