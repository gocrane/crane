import React, { memo } from 'react';
import { Row, Col, Switch } from 'tdesign-react';
import { useAppDispatch, useAppSelector } from 'modules/store';
import {
  selectGlobal,
  toggleShowHeader,
  toggleShowBreadcrumbs,
  toggleShowFooter,
  switchTheme,
  switchColor,
  switchLayout,
  openSystemTheme,
  ELayout,
} from 'modules/global';
import { ETheme } from 'types/index.d';
import RadioColor from './RadioColor';
import RadioRect from './RadioRect';

import Light from 'assets/svg/assets-setting-light.svg?component';
import Dark from 'assets/svg/assets-setting-dark.svg?component';
import System from 'assets/svg/assets-setting-auto.svg?component';

import Style from './index.module.less';

enum ESettingTheme {
  system,
}

const themeList = [
  {
    value: ETheme.light,
    image: <Light />,
    name: '明亮',
  },
  {
    value: ETheme.dark,
    image: <Dark />,
    name: '黑暗',
  },
  {
    value: ESettingTheme.system,
    image: <System />,
    name: '跟随系统',
  },
];

const layoutList = [
  {
    value: ELayout.side,
    image: 'https://tdesign.gtimg.com/starter/setting/side.png',
  },
  {
    value: ELayout.top,
    image: 'https://tdesign.gtimg.com/starter/setting/top.png',
  },
  {
    value: ELayout.mix,
    image: 'https://tdesign.gtimg.com/starter/setting/mix.png',
  },
];

export default memo(() => {
  const dispatch = useAppDispatch();
  const globalState = useAppSelector(selectGlobal);

  const handleThemeSwitch = (value: any) => {
    if (value === ESettingTheme.system) {
      dispatch(openSystemTheme());
    } else {
      dispatch(switchTheme(value));
    }
  };

  return (
    <div>
      <div className={Style.settingTitle}>主题模式</div>
      <RadioRect
        defaultValue={globalState.systemTheme ? ESettingTheme.system : globalState.theme}
        onChange={handleThemeSwitch}
        options={themeList}
      />

      <div className={Style.settingTitle}>主题色</div>
      <RadioColor defaultValue={globalState.color} onChange={(value) => dispatch(switchColor(value))} />

      <div className={Style.settingTitle}>导航布局</div>
      <RadioRect
        defaultValue={globalState.layout}
        onChange={(value) => dispatch(switchLayout(value))}
        options={layoutList}
      />

      <div className={Style.settingTitle}>元素开关</div>
      <Row justify='space-between'>
        <Col>
          <div className={Style.settingSubTitle}>显示 Header</div>
        </Col>
        <Col>
          <Switch size='large' value={globalState.showHeader} onChange={() => dispatch(toggleShowHeader())} />
        </Col>
      </Row>

      <Row justify='space-between'>
        <Col>
          <div className={Style.settingSubTitle}>显示 Breadcrumbs</div>
        </Col>
        <Col>
          <Switch size='large' value={globalState.showBreadcrumbs} onChange={() => dispatch(toggleShowBreadcrumbs())} />
        </Col>
      </Row>

      <Row justify='space-between'>
        <Col>
          <div className={Style.settingSubTitle}>显示 Footer</div>
        </Col>
        <Col>
          <Switch size='large' value={globalState.showFooter} onChange={() => dispatch(toggleShowFooter())} />
        </Col>
      </Row>
    </div>
  );
});
