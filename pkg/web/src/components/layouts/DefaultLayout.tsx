import clsx from 'clsx';
import { Icon, Layout, List, NavMenu } from 'tea-component';

import React from 'react';
import { useTranslation } from 'react-i18next';

import { changeLanguage, SupportLanguages } from '../../i18n';
import { ContentHeader } from '../common/ContentHeader';
import { SideMenu } from '../common/SideMenu';

export function DefaultLayout({ content }: { content: React.ReactNode }) {
  const { t } = useTranslation();

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Header>
        <NavMenu
          left={null}
          right={
            <>
              <NavMenu.Item
                overlay={close => (
                  <List type="option">
                    <List.Item onClick={() => changeLanguage(SupportLanguages.zh)}>{t('中文')}</List.Item>
                    <List.Item onClick={() => changeLanguage(SupportLanguages.en)}>{t('英文')}</List.Item>
                  </List>
                )}
                type="dropdown"
              >
                {t('切换语言')}
              </NavMenu.Item>
            </>
          }
        />
      </Layout.Header>
      <Layout.Body>
        <Layout.Sider style={{ height: 'auto' }}>
          <SideMenu />
        </Layout.Sider>
        <Layout.Content>
          <ContentHeader />
          <div style={{ margin: '1rem' }}>{content}</div>
        </Layout.Content>
      </Layout.Body>
    </Layout>
  );
}
