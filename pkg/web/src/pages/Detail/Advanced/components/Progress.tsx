import React from 'react';
import { Steps, Card } from 'tdesign-react';
import { dataStep, stepCurrent } from '../consts';
import Style from '../index.module.less';

const { StepItem } = Steps;

const Progress = () => (
  <Card title='发票进度' className={Style.cardBox} header>
    <Steps current={stepCurrent}>
      {dataStep.map((item) => (
        <StepItem key={item.id} title={item.name} content={item.detail} />
      ))}
    </Steps>
  </Card>
);

export default React.memo(Progress);
