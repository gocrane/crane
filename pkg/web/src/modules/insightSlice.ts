import { Aggregation, QueryWindow } from '../models';
import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { rangeMap } from 'utils/rangeMap';

export interface InsightState {
  aggregation: Aggregation;

  window: QueryWindow;

  customRange: { start: string; end: string };

  selectedDashboard?: any;

  selectedClusterId?: string;

  selectedNamespace?: string | null;

  discount: number;

  selectedWorkloadType?: string;

  selectedWorkload?: string;

}

export const initialInsightState: InsightState = {
  discount: 100,

  aggregation: Aggregation.CLUSTER,

  window: QueryWindow.LAST_1_DAY,

  customRange: {
    start: rangeMap[QueryWindow.LAST_1_DAY][0].format('YYYY-MM-DD HH:mm:ss'),
    end: rangeMap[QueryWindow.LAST_1_DAY][1].format('YYYY-MM-DD HH:mm:ss'),
  },

  selectedDashboard: null,

  selectedClusterId: '',

  selectedNamespace: null,
};

const slice = createSlice({
  name: 'insight',
  initialState: initialInsightState,
  reducers: {
    discount: (state, action: PayloadAction<number>) => {
      state.discount = action.payload;
    },
    selectedNamespace: (state, action: PayloadAction<string>) => {
      state.selectedNamespace = action.payload;
    },
    selectedClusterId: (state, action: PayloadAction<string>) => {
      state.selectedClusterId = action.payload;
    },
    selectedDashboard: (state, action: PayloadAction<any>) => {
      state.selectedDashboard = action.payload;
    },
    aggregation: (state, action: PayloadAction<Aggregation>) => {
      state.aggregation = action.payload;
    },
    window: (state, action: PayloadAction<QueryWindow>) => {
      state.window = action.payload;
    },
    customRange: (state, action: PayloadAction<InsightState['customRange']>) => {
      state.customRange = action.payload;
    },
    selectedWorkloadType: (state, action: PayloadAction<any>) => {
      state.selectedWorkloadType = action.payload;
    },
    selectedWorkload: (state, action: PayloadAction<any>) => {
      state.selectedWorkload = action.payload;
    },
  },
});

export const insightAction = slice.actions;
export const insightReducer = slice.reducer;
