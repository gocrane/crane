export {};

declare global {
  interface Window {
    __RUNTIME_CONFIG__: {
      CRANE_URL: string;
      CLUSTER_ID: string;
    };
  }
}
