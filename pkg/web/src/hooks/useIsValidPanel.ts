export const useIsValidPanel = ({ panel }: { panel: any }) => {
  return !panel.hasOwnProperty('collapsed');
};
