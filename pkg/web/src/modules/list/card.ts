import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import { RootState } from '../store';
import { getProductList, IProduct } from 'services/product';

const namespace = 'list/card';

interface IInitialState {
  pageLoading: boolean;
  loading: boolean;
  current: number;
  pageSize: number;
  total: number;
  productList: IProduct[];
}

const initialState: IInitialState = {
  pageLoading: true,
  loading: true,
  current: 1,
  pageSize: 12,
  total: 0,
  productList: [],
};

export const getList = createAsyncThunk(
  `${namespace}/getList`,
  async (params: { pageSize: number; current: number }) => {
    const { pageSize, current } = params;
    const result = await getProductList({
      pageSize,
      current,
    });
    return {
      list: result?.list,
      total: result?.total,
      pageSize: params.pageSize,
      current: params.current,
    };
  },
);

const listCardSlice = createSlice({
  name: namespace,
  initialState,
  reducers: {
    clearPageState: () => initialState,
    switchPageLoading: (state, action: PayloadAction<boolean>) => {
      state.pageLoading = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(getList.pending, (state) => {
        state.loading = true;
      })
      .addCase(getList.fulfilled, (state, action) => {
        state.loading = false;
        state.productList = action.payload?.list;
        state.total = action.payload?.total;
        state.pageSize = action.payload?.pageSize;
        state.current = action.payload?.current;
      })
      .addCase(getList.rejected, (state) => {
        state.loading = false;
      });
  },
});

export const { clearPageState, switchPageLoading } = listCardSlice.actions;

export const selectListCard = (state: RootState) => state.listCard;

export default listCardSlice.reducer;
