import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface OverviewState {
  searchText: string;
}

export const initialOverviewState: OverviewState = {
  searchText: '',
};

const slice = createSlice({
  name: 'overview',
  initialState: initialOverviewState,
  reducers: {
    searchText: (state, action: PayloadAction<any>) => {
      state.searchText = action.payload;
    },
  },
});

export const overviewActions = slice.actions;
export const overviewReducer = slice.reducer;
