import React from 'react';
import { Input } from 'tdesign-react';
import { SearchIcon } from 'tdesign-icons-react';
import Style from './Search.module.less';
import { useTranslation } from 'react-i18next';

const Search = () => {
  const { t } = useTranslation();
  return <Input className={Style.panel} prefixIcon={<SearchIcon />} placeholder={t('请输入搜索内容')} />;
};

export default React.memo(Search);
