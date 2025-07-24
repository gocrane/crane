import React, { memo } from 'react';
import { Button } from 'tdesign-react';

import Light403Icon from 'assets/svg/assets-result-403.svg?component';
import Light404Icon from 'assets/svg/assets-result-404.svg?component';
import Light500Icon from 'assets/svg/assets-result-500.svg?component';
import style from './index.module.less';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

enum ECode {
  forbidden = 403,
  notFount = 404,
  error = 500,
}

interface IErrorPageProps {
  code: ECode;
  title?: string;
  desc?: string;
}

const ErrorPage: React.FC<IErrorPageProps> = (props) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const errorInfo = {
    [ECode.forbidden]: {
      title: t('403 Forbidden'),
      desc: t('抱歉，您无权限访问此页面'),
      icon: <Light403Icon />,
    },
    [ECode.notFount]: {
      title: t('404 Not Found'),
      desc: t('抱歉，您访问的页面不存在。'),
      icon: <Light404Icon />,
    },
    [ECode.error]: {
      title: t('500 Internal Server Error'),
      desc: t('抱歉，服务器出错啦！'),
      icon: <Light500Icon />,
    },
  };
  const info = errorInfo[props.code];

  return (
    <div className={style.errorBox}>
      {info?.icon}
      <div className={style.title}>{info?.title}</div>
      <div className={style.description}>{info?.desc}</div>
      <Button theme='primary' onClick={() => navigate('/')}>
        {t('返回首页')}
      </Button>
    </div>
  );
};

export default memo(ErrorPage);
