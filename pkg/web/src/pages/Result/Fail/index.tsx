import React, { memo } from 'react';
import { Button } from 'tdesign-react';
import { ErrorCircleIcon } from 'tdesign-icons-react';

import style from './index.module.less';

const Fail = () => (
  <div className={style.Content}>
    <ErrorCircleIcon className={style.icon} />
    <div className={style.title}>创建失败</div>
    <div className={style.description}>抱歉，您的项目创建失败，企业微信联系检查创建者权限，或返回修改。</div>
    <div>
      <Button>返回修改</Button>
      <Button className={style.rightButton} theme='default'>
        返回首页
      </Button>
    </div>
  </div>
);

export default memo(Fail);
