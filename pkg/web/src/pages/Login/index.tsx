import React, { memo, useState } from 'react';
import classNames from 'classnames';
import Login from './components/Login';
import Register from './components/Register';
import LoginHeader from './components/Header';
import { useAppSelector } from 'modules/store';
import { selectGlobal } from 'modules/global';

import Style from './index.module.less';

export default memo(() => {
  const [type, setType] = useState('login');
  const globalState = useAppSelector(selectGlobal);
  const { theme } = globalState;
  const handleSwitchLoginType = () => {
    setType(type === 'register' ? 'login' : 'register');
  };

  return (
    <div
      className={classNames(Style.loginWrapper, { [Style.light]: theme === 'light', [Style.dark]: theme !== 'light' })}
    >
      <LoginHeader />
      <div className={Style.loginContainer}>
        <div className={Style.titleContainer}>
          <h1 className={Style.title}>登录到</h1>
          <h1 className={Style.title}>TDesign Starter</h1>
          <div className={Style.subTitle}>
            <p className={classNames(Style.tip, Style.registerTip)}>
              {type === 'register' ? '已有账号?' : '没有账号吗?'}
            </p>
            <p className={classNames(Style.tip, Style.loginTip)} onClick={handleSwitchLoginType}>
              {type === 'register' ? '登录' : '注册新账号'}
            </p>
          </div>
        </div>
        {type === 'login' ? <Login /> : <Register />}
      </div>
      <footer className={Style.copyright}>Copyright @ 2021-2022 Tencent. All Rights Reserved</footer>
    </div>
  );
});
