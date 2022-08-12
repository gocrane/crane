import { SerializedError } from '@reduxjs/toolkit';
import { FetchBaseQueryError } from '@reduxjs/toolkit/dist/query';

import { t } from '../i18n';

export const getErrorMsg = (error: FetchBaseQueryError | SerializedError | undefined): string => {
  let msg = t('发生未知错误，请稍候再试');

  const serializedError = error as SerializedError;
  const fetchBaseQueryError = error as FetchBaseQueryError;
  const anyErrror = error as any;

  if (serializedError.message) {
    msg = serializedError.message;
  } else if ((fetchBaseQueryError.data as any)?.message) {
    msg = (fetchBaseQueryError.data as any).message;
  } else if (anyErrror?.data?.ErrStatus) {
    msg = (error as any).data.ErrStatus.message;
  }

  return msg;
};
