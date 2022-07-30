import React, { memo } from 'react';
import { Steps, Card } from 'tdesign-react';
import classnames from 'classnames';
import { dataInfo, dataStep } from './consts';
import Style from './index.module.less';

const { StepItem } = Steps;

export default memo(() => (
  <div>
    <Card title='基本信息' header>
      <div className={classnames(Style.infoBox)}>
        {dataInfo.map((item) => (
          <div key={item.id} className={classnames(Style.infoBoxItem)}>
            <h1>{item.name}</h1>
            <span
              className={classnames({
                [Style.inProgress]: item.type === 'status',
                [Style.pdf]: item.type === 'link',
              })}
            >
              {item.type === 'status' && <i />}
              {item.value}
            </span>
          </div>
        ))}
      </div>
    </Card>
    <Card title='变更记录' header className={Style.logBox}>
      <div>
        <Steps layout='vertical' theme='dot' current={1}>
          {dataStep.map((item) => (
            <StepItem key={item.id} title={item.name} content={item.detail} />
          ))}
        </Steps>
      </div>
    </Card>
  </div>
));
