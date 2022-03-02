import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';

import { InsightPanel } from '../components/insight/InsightPanel';
import { DefaultLayout } from '../components/layouts/DefaultLayout';
import { OverviewPanel } from '../components/overview/OverviewPanel';
import { RoutesEnum } from './routeEnum';

const Layout = DefaultLayout;

export const Router = () => {
  return (
    <Routes>
      <Route element={<Navigate to={RoutesEnum.OVERVIEW} />} path={RoutesEnum.DEFAULT} />
      <Route element={<Layout content={<OverviewPanel />} />} path={RoutesEnum.OVERVIEW} />
      <Route element={<Layout content={<InsightPanel />} />} path={RoutesEnum.INSIGHT} />
    </Routes>
  );
};
