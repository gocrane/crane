import { matchPath, matchRoutes, useLocation, useRoutes } from 'react-router-dom';
import { RoutesEnum } from 'router/routeEnum';

export const useSideMenuSelection = () => {
  const { pathname } = useLocation();

  const isOverview = matchPath(`${RoutesEnum.OVERVIEW}/*`, pathname);
  const isInsight = matchPath(`${RoutesEnum.INSIGHT}/*`, pathname);

  if (isOverview) return RoutesEnum.OVERVIEW;
  if (isInsight) return RoutesEnum.INSIGHT;
  return null;
};
