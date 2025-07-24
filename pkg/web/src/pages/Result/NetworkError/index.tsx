import React, { memo } from 'react';
import { Button } from 'tdesign-react';

import NetworkErrorIcon from 'assets/svg/assets-result-network-error.svg?component';
import style from '../index.module.less';

const NetworkError = () => (
  <div className={style.Content}>
    <NetworkErrorIcon />
    <div className={style.title}>网络异常</div>
    <div className={style.description}>网络异常，请稍后再试</div>
    <div>
      <Button>重新加载</Button>
      <Button className={style.rightButton} theme='default'>
        返回首页
      </Button>
    </div>
  </div>
);

export default memo(NetworkError);
