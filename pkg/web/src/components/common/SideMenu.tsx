import React from 'react';
import { useTranslation } from 'react-i18next';
import { Routes, useNavigate } from 'react-router-dom';
import { Menu } from 'tdesign-react';

import { useSideMenuSelection } from '../../hooks/useSideMenuSelection';
import { RoutesEnum } from '../../routes/routeEnum';

export const SideMenu: React.FC = () => {
  const { t } = useTranslation();
  const selection = useSideMenuSelection();
  const navigate = useNavigate();

  return (
    <Menu style={{ minHeight: '100%' }} theme="dark" value={selection}>
      <Menu.MenuItem
        value={RoutesEnum.OVERVIEW}
        onClick={() => {
          navigate(RoutesEnum.OVERVIEW);
        }}
      >
        {t('成本概览')}
      </Menu.MenuItem>
      <Menu.MenuItem
        value={RoutesEnum.INSIGHT}
        onClick={() => {
          navigate(RoutesEnum.INSIGHT);
        }}
      >
        {t('成本洞察')}
      </Menu.MenuItem>
    </Menu>
  );
};
