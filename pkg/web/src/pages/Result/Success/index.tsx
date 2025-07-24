import React, { memo } from 'react';
import { Button } from 'tdesign-react';
import { CheckCircleIcon } from 'tdesign-icons-react';

import style from './index.module.less';

const Success = () => (
  <div className={style.Content}>
    <CheckCircleIcon className={style.icon} />
    <div className={style.title}>项目已创建成功</div>
    <div className={style.description}>可以联系负责人分发应用</div>
    <div>
      <Button>返回首页</Button>
      <Button className={style.rightButton} theme='default'>
        查看进度
      </Button>
    </div>
  </div>
);

export default memo(Success);
