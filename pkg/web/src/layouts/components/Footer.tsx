import { selectGlobal } from 'modules/global';
import { useAppSelector } from 'modules/store';
import React from 'react';
import { Layout, Row } from 'tdesign-react';

const { Footer: TFooter } = Layout;

const Footer = () => {
  const globalState = useAppSelector(selectGlobal);
  if (!globalState.showFooter) {
    return null;
  }

  return (
    <TFooter>
      <Row justify='center'>Thanks for all the crane contributors.</Row>
    </TFooter>
  );
};

export default React.memo(Footer);
