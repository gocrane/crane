import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface ConfigState {
  chartBaselineHeight: number;
  chartDefaultHeight: number;
}

export const initialConfigState: ConfigState = {
  chartBaselineHeight: 50,
  chartDefaultHeight: 300
};

const slice = createSlice({
  name: 'config',
  initialState: initialConfigState,
  reducers: {
    chartBaselineHeight: (state, action: PayloadAction<number>) => {
      state.chartBaselineHeight = action.payload;
    },
    chartDefaultHeight: (state, action: PayloadAction<number>) => {
      state.chartDefaultHeight = action.payload;
    }
  }
});

export const configActions = slice.actions;
export const configReducer = slice.reducer;
