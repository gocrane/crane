import React from 'react';
import { Input, Tree } from 'tdesign-react';
import { SearchIcon } from 'tdesign-icons-react';
import classnames from 'classnames';
import { SelectTable } from '../Select';
import { treeList } from './consts';
import CommonStyle from 'styles/common.module.less';
import Style from './index.module.less';

const TreeTable: React.FC = () => (
  <div className={classnames(CommonStyle.pageWithColor, Style.content)}>
    <div className={Style.treeContent}>
      <Input className={Style.search} suffixIcon={<SearchIcon />} placeholder='请输入关键词' />
      <Tree data={treeList} activable hover transition />
    </div>
    <div className={Style.tableContent}>
      <SelectTable />
    </div>
  </div>
);

export default TreeTable;
