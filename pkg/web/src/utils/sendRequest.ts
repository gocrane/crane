export interface SendRequestArgs {
  requestInfo: RequestInfo;
  init: RequestInit;
}

export const sendRequest = (args: SendRequestArgs) => {
  const { requestInfo, init } = args;
};
