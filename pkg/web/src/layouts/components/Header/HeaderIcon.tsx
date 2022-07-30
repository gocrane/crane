import { SupportLanguages, changeLanguage } from '../../../i18n';
import Style from './HeaderIcon.module.less';
import { toggleSetting } from 'modules/global';
import { useAppDispatch } from 'modules/store';
import React, { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { Icon, LogoGithubIcon, HelpCircleIcon, SettingIcon } from 'tdesign-icons-react';
import { Button, Popup, Badge, Dropdown, Row, Col } from 'tdesign-react';

export default memo(() => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const navigate = useNavigate();

  const gotoWiki = () => {
    window.open('https://docs.gocrane.io');
  };

  const gotoGitHub = () => {
    window.open('https://github.com/gocrane/crane');
  };
  return (
    <Row align='middle' style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Col>
        <Dropdown
          className={Style.dropdown}
          trigger={'click'}
          options={[
            { value: SupportLanguages.zh, content: t('中文') },
            { value: SupportLanguages.en, content: t('英文') },
          ]}
          onClick={(data) => {
            changeLanguage(data.value as SupportLanguages);
          }}
        >
          <Button variant='text'>
            <span style={{ display: 'inline-flex', justifyContent: 'center', alignItems: 'center' }}>
              <span style={{ display: 'inline-block', margin: '0 5px' }}>{t('切换语言')}</span>
              <Icon name='chevron-down' size='20px' />
            </span>
          </Button>
        </Dropdown>
      </Col>
      <Col>
        <Button shape='square' size='large' variant='text' onClick={gotoGitHub}>
          <Popup content='代码仓库' placement='bottom' showArrow destroyOnClose>
            <LogoGithubIcon />
          </Popup>
        </Button>
      </Col>
      <Col>
        <Button shape='square' size='large' variant='text' onClick={gotoWiki}>
          <Popup content='帮助文档' placement='bottom' showArrow destroyOnClose>
            <HelpCircleIcon />
          </Popup>
        </Button>
      </Col>
      <Col>
        <Button shape='square' size='large' variant='text' onClick={() => dispatch(toggleSetting())}>
          <Popup content='页面设置' placement='bottom' showArrow destroyOnClose>
            <SettingIcon />
          </Popup>
        </Button>
      </Col>
    </Row>
  );
});
