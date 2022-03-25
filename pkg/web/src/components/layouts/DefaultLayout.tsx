import React from 'react';
import { useTranslation } from 'react-i18next';
import { Icon } from 'tdesign-icons-react';
import { Button, Dropdown, Layout, Menu } from 'tdesign-react';

import { SupportLanguages } from '../../i18n';
import { ContentHeader } from '../common/ContentHeader';
import { SideMenu } from '../common/SideMenu';

const { Header, Content, Footer, Aside } = Layout;
const { HeadMenu } = Menu;

export function DefaultLayout({ content }: { content: React.ReactNode }) {
  const { t, i18n } = useTranslation();

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header>
        <HeadMenu
          logo={
            <span style={{ marginLeft: 18, fontSize: 18, color: 'var(--td-font-white-1)' }}>{'Crane Dashboard'}</span>
          }
          operations={
            <Dropdown
              minColumnWidth={87}
              options={[
                { value: SupportLanguages.zh, content: t('中文') },
                { value: SupportLanguages.en, content: t('英文') }
              ]}
              trigger="click"
              onClick={data => {
                i18n.changeLanguage(data.value as SupportLanguages);
              }}
            >
              <Button theme="default" variant="base">
                {t('切换语言')}
                <Icon name="chevron-down" size="16" />
              </Button>
            </Dropdown>
          }
          style={{ background: 'var(--td-gray-color-13)' }}
          theme="light"
        />
      </Header>
      <Layout>
        <Aside style={{ borderTop: '1px solid var(--component-border)' }}>
          <SideMenu />
        </Aside>
        <Layout>
          <Content>
            <ContentHeader />
            <div style={{ margin: '1rem' }}>{content}</div>
          </Content>
          <Footer>Copyright @ 2019-2020 Tencent. All Rights Reserved</Footer>
        </Layout>
      </Layout>
    </Layout>
  );
}
