import { useCraneUrl, useSelector } from '../../../hooks';
import { insightAction } from '../../../modules/insightSlice';
import { HeaderMenu } from '../Menu';
import HeaderIcon from './HeaderIcon';
import Style from './index.module.less';
import { selectGlobal, toggleMenu } from 'modules/global';
import { useAppDispatch, useAppSelector } from 'modules/store';
import React, { memo, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router-dom';
import { ViewListIcon } from 'tdesign-icons-react';
import { Layout, Button, Row, Col, Select, MessagePlugin } from 'tdesign-react';
import { useFetchClusterListQuery } from '../../../services/clusterApi';

const { Header } = Layout;

export default memo((props: { showMenu?: boolean }) => {
  const { t } = useTranslation();
  const globalState = useAppSelector(selectGlobal);
  const dispatch = useAppDispatch();
  const selectedClusterId = useSelector((state) => state.insight.selectedClusterId);
  const clusterList = useFetchClusterListQuery({});
  const navigate = useNavigate();
  const location = useLocation();
  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    if (selectedClusterId === '' && clusterList?.data?.data?.items?.length > 0) {
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      dispatch(insightAction.selectedClusterId(clusterList.data.data.items[0].id));
    }

    if (clusterList?.data?.data?.items?.length === 0) {
      dispatch(insightAction.selectedClusterId(''));
      if (location.pathname !== '/manager/cluster') {
        console.log(`From ${location.pathname} to /manager/cluster`);
        MessagePlugin.error(
          {
            content: t('添加一个集群以启用Dashboard'),
            closeBtn: true,
          },
          10000,
        );
        navigate('/manager/cluster');
      }
    }
  });

  const options = React.useMemo(
    () =>
      (clusterList.data?.data?.items ?? []).map((item) => ({
        text: `${item.name} (${item.id})`,
        value: item.id,
      })),
    [clusterList.data?.data?.items],
  );

  if (!globalState.showHeader) {
    return null;
  }

  let HeaderLeft;
  if (props.showMenu) {
    HeaderLeft = (
      <div>
        <HeaderMenu />
      </div>
    );
  } else {
    HeaderLeft = (
      <Row gutter={16} align='middle'>
        <Col>
          <Button shape='square' size='large' variant='text' onClick={() => dispatch(toggleMenu(null))}>
            <ViewListIcon />
          </Button>
        </Col>
        <Col>
          <Select
            empty={t('暂无数据')}
            placeholder={t('请选择集群')}
            style={{ width: '200px' }}
            value={selectedClusterId}
            onChange={(value: any) => {
              dispatch(insightAction.selectedClusterId(value));
            }}
          >
            {options.map((option) => (
              <Select.Option key={option.value} label={option.text} value={option.value} />
            ))}
          </Select>
        </Col>
        {/* <Col> */}
        {/*  <Search /> */}
        {/* </Col> */}
      </Row>
    );
  }

  return (
    <Header className={Style.panel}>
      {HeaderLeft}
      <HeaderIcon />
    </Header>
  );
});
