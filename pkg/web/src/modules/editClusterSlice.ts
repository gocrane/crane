
import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { v4 } from 'uuid';

export interface EditClusterState {
  mode: 'update' | 'create';
  modalVisible: boolean;

  clusters: Array<{
    id: string;
    clusterName: string;
    craneUrl: string;
    discount: number;
    preinstallRecommendation: boolean;
  }>;

  editingClusterId: string | null;
}

type Cluster = EditClusterState['clusters'][0];

const defaultCluster = {
  id: v4(),
  clusterName: '',
  craneUrl: '',
  discount: 1,
  preinstallRecommendation: true,
};

const initialEditClusterState: EditClusterState = {
  mode: 'create',
  modalVisible: false,
  editingClusterId: null,
  clusters: [{ ...defaultCluster }],
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
    addCluster: (state) => {
      state.clusters = [
        ...(state.clusters ?? []),
        {
          id: v4(),
          clusterName: '',
          craneUrl: '',
          discount: 1,
          preinstallRecommendation: true,
        },
      ];
    },
    updateCluster: (state, action: PayloadAction<{ id: string; data: Partial<Omit<Cluster, 'id'>> }>) => {
      // Remove /
      // craneUrl = http://localhost:3000/ => http://localhost:3000
      const { craneUrl } = action.payload.data;
      if (craneUrl) {
        action.payload.data.craneUrl = craneUrl.endsWith('/')
          ? craneUrl.substring(0, craneUrl.lastIndexOf('/'))
          : craneUrl;
      }
      state.clusters = state.clusters.map((cluster) => {
        if (cluster.id === action.payload.id) {
          return {
            ...cluster,
            ...(action.payload.data ?? {}),
          };
        }
        return cluster;
      });
    },
    deleteCluster: (state, action: PayloadAction<{ id: string }>) => {
      state.clusters = state.clusters.filter((cluster) => cluster.id !== action.payload.id);
    },
    resetCluster: (state) => {
      state.clusters = [{ ...defaultCluster, id: v4() }];
    },
    modalVisible: (state, action: PayloadAction<boolean>) => {
      state.modalVisible = action.payload;
    },
  },
});

export const editClusterActions = slice.actions;
export const editClusterReducer = slice.reducer;
