import Style from './Menu.module.less';
import MenuLogo from './MenuLogo';
import { selectGlobal } from 'modules/global';
import { useAppSelector } from 'modules/store';
import React, { memo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useRouteConfig, IRouter } from 'router';
import { Menu, MenuValue } from 'tdesign-react';
import { resolve } from 'utils/path';

const { SubMenu, MenuItem, HeadMenu } = Menu;

interface IMenuProps {
  showLogo?: boolean;
  showOperation?: boolean;
}

const renderMenuItems = (menu: IRouter[], parentPath = '') => {
  const navigate = useNavigate();
  return menu.map((item) => {
    const { children, meta, path } = item;

    if (!meta || meta?.hidden === true) {
      // 无meta信息 或 hidden == true，路由不显示为菜单
      return null;
    }

    const { Icon, title, single } = meta;
    const routerPath = resolve(parentPath, path);

    if (!children || children.length === 0) {
      return (
        <MenuItem
          key={routerPath}
          value={routerPath}
          icon={Icon ? <Icon /> : undefined}
          onClick={() => navigate(routerPath)}
        >
          {title}
        </MenuItem>
      );
    }

    if (single && children?.length > 0) {
      const firstChild = children[0];
      if (firstChild?.meta && !firstChild?.meta?.hidden) {
        const { Icon, title } = meta;
        const singlePath = resolve(resolve(parentPath, path), firstChild.path);
        return (
          <MenuItem
            key={singlePath}
            value={singlePath}
            icon={Icon ? <Icon /> : undefined}
            onClick={() => navigate(singlePath)}
          >
            {title}
          </MenuItem>
        );
      }
    }

    return (
      <SubMenu key={routerPath} value={routerPath} title={title} icon={Icon ? <Icon /> : undefined}>
        {renderMenuItems(children, routerPath)}
      </SubMenu>
    );
  });
};

/**
 * 顶部菜单
 */
export const HeaderMenu = memo(() => {
  const router = useRouteConfig();
  const globalState = useAppSelector(selectGlobal);
  const location = useLocation();
  const [active, setActive] = useState<MenuValue>(location.pathname); // todo

  return (
    <HeadMenu
      expandType='popup'
      style={{ marginBottom: 20 }}
      value={active}
      theme={globalState.theme}
      onChange={(v) => setActive(v)}
    >
      {renderMenuItems(router)}
    </HeadMenu>
  );
});

/**
 * 左侧菜单
 */
export default memo((props: IMenuProps) => {
  const router = useRouteConfig();
  const location = useLocation();
  const globalState = useAppSelector(selectGlobal);

  const { version } = globalState;
  const bottomText = globalState.collapsed ? version : `Crane Dashboard ${version}`;

  return (
    <Menu
      width='232px'
      style={{ flexShrink: 0, height: '100%' }}
      value={location.pathname}
      theme={globalState.theme}
      collapsed={globalState.collapsed}
      operations={props.showOperation ? <div className={Style.menuTip}>{bottomText}</div> : undefined}
      logo={props.showLogo ? <MenuLogo collapsed={globalState.collapsed} /> : undefined}
    >
      {renderMenuItems(router)}
    </Menu>
  );
});
