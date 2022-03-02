export const getConfig = () => {
  return {
    clusterId: window.__RUNTIME_CONFIG__.CLUSTER_ID,
    craneUrl: window.__RUNTIME_CONFIG__.CRANE_URL
  };
};
