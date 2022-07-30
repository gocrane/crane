import React, { memo } from 'react';
import { Button } from 'tdesign-react';

import MaintenanceIcon from 'assets/svg/assets-result-maintenance.svg?component';
import style from './index.module.less';

const BrowserIncompatible = () => (
  <div className={style.Content}>
    <MaintenanceIcon />
    <div className={style.title}>系统维护中</div>
    <div className={style.description}>系统维护中，请稍后再试。</div>

    <div className={style.resultSlotContainer}>
      <Button className={style.rightButton}>返回首页</Button>
    </div>
  </div>
);

export default memo(BrowserIncompatible);
