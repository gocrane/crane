import { Menu } from 'tea-component';

import React from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { useSideMenuSelection } from '../../hooks/useSideMenuSelection';
import { RoutesEnum } from '../../routes/routeEnum';

export const SideMenu = () => {
  const { t } = useTranslation();
  const selection = useSideMenuSelection();
  const navigate = useNavigate();

  return (
    <Menu theme="dark" title={t('成本大师')}>
      <Menu.Item
        selected={selection === RoutesEnum.OVERVIEW}
        title={t('成本概览')}
        onClick={() => {
          navigate(RoutesEnum.OVERVIEW);
        }}
      />
      <Menu.Item
        selected={selection === RoutesEnum.INSIGHT}
        title={t('成本洞察')}
        onClick={() => {
          navigate(RoutesEnum.INSIGHT);
        }}
      />
    </Menu>
  );
};
