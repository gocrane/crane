import { EditClusterModal } from './EditClusterModal';
import { OverviewActionPanel } from './OverviewActionPanel';
import { OverviewTablePanel } from './OverviewTablePanel';
import classnames from 'classnames';
import React, { memo } from 'react';
import CommonStyle from 'styles/common.module.less';

export default memo(() => (
  <div className={classnames(CommonStyle.pageWithPadding, CommonStyle.pageWithColor)}>
    <>
      <OverviewActionPanel />
      <OverviewTablePanel />
      <EditClusterModal />
    </>
  </div>
));
