import React from 'react';
import { useTranslation } from 'react-i18next';
import { Icon } from 'tdesign-icons-react';
import { Button, Dropdown, Layout, Menu } from 'tdesign-react';

import { SupportLanguages } from '../../i18n';
import { ContentHeader } from '../common/ContentHeader';
import { SideMenu } from '../common/SideMenu';

const { Header, Content, Aside } = Layout;
const { HeadMenu } = Menu;

export function DefaultLayout({ content }: { content: React.ReactNode }) {
  const { t, i18n } = useTranslation();

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header>
        <HeadMenu
          logo={
            <img alt="logo" height="100" src="/logo.svg" style={{ marginTop: 20 }} width="240" />
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
        </Layout>
      </Layout>
    </Layout>
  );
}
