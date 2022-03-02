import { matchPath, matchRoutes, useLocation, useRoutes } from 'react-router-dom';

import { RoutesEnum } from '../routes/routeEnum';

export const useSideMenuSelection = () => {
  const { pathname } = useLocation();

  const isOverview = matchPath(RoutesEnum.OVERVIEW + '/*', pathname);
  const isInsight = matchPath(RoutesEnum.INSIGHT + '/*', pathname);

  if (isOverview) return RoutesEnum.OVERVIEW;
  else if (isInsight) return RoutesEnum.INSIGHT;
  return null;
};
