import React, { memo } from 'react';
import { Button } from 'tdesign-react';

import BrowserIncompatibleIcon from 'assets/svg/assets-result-browser-incompatible.svg?component';
import style from './index.module.less';
import { useTranslation } from 'react-i18next';

const BrowserIncompatible = () => {
  const { t } = useTranslation();
  return (
    <div className={style.Content}>
      <BrowserIncompatibleIcon />
      <div className={style.title}>{t('浏览器版本低')}</div>
      <div className={style.description}>{t('抱歉，您正在使用的浏览器版本过低，无法打开当前网页。')}</div>

      <div className={style.resultSlotContainer}>
        <Button className={style.rightButton}>{t('返回首页')}</Button>
        <div className={style.recommendContainer}>
          <div>{t('TDesign Starter 推荐以下主流浏览器')}</div>
          <div className={style.recommendBrowser}>
            <div>
              <img src='https://tdesign.gtimg.com/starter/result-page/chorme.png' alt='Chrome' />
              <div>Chrome</div>
            </div>
            <div>
              <img src='https://tdesign.gtimg.com/starter/result-page/qq-browser.png' alt='QQ Browser' />
              <div>QQ Browser</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default memo(BrowserIncompatible);
