import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface OverviewState {
  searchFilter: { clusterNames: string[]; clusterIds: string[] } | null;
}

export const initialOverviewState: OverviewState = {
  searchFilter: null
};

const slice = createSlice({
  name: 'overview',
  initialState: initialOverviewState,
  reducers: {
    searchFilter: (state, action: PayloadAction<any>) => {
      state.searchFilter = action.payload;
    }
  }
});

export const overviewActions = slice.actions;
export const overviewReducer = slice.reducer;
