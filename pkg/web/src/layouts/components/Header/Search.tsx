import React from 'react';
import { Input } from 'tdesign-react';
import { SearchIcon } from 'tdesign-icons-react';
import Style from './Search.module.less';

const Search = () => <Input className={Style.panel} prefixIcon={<SearchIcon />} placeholder='请输入搜索内容' />;

export default React.memo(Search);
