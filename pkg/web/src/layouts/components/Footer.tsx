import { selectGlobal } from 'modules/global';
import { useAppSelector } from 'modules/store';
import React from 'react';
import { useTranslation } from 'react-i18next';
import { Layout, Row } from 'tdesign-react';

const { Footer: TFooter } = Layout;

const Footer = () => {
  const { t } = useTranslation();
  const globalState = useAppSelector(selectGlobal);
  if (!globalState.showFooter) {
    return null;
  }

  return (
    <TFooter>
      <Row justify='center'>{t('Thanks for all the crane contributors.')}</Row>
    </TFooter>
  );
};

export default React.memo(Footer);
