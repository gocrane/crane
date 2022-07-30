import React, { useEffect } from 'react';
import { Row, Col, Button, Input, Pagination, Loading } from 'tdesign-react';
import { SearchIcon } from 'tdesign-icons-react';
import { useAppDispatch, useAppSelector } from 'modules/store';
import { selectListCard, clearPageState, getList, switchPageLoading } from 'modules/list/card';
import ProductCard from './components/ProductCard';
import { PageInfo } from 'tdesign-react/es/pagination/type';
import Style from './index.module.less';

const CardList = () => {
  const dispatch = useAppDispatch();
  const pageState = useAppSelector(selectListCard);

  const pageInit = async () => {
    await dispatch(
      getList({
        pageSize: pageState.pageSize,
        current: 1,
      }),
    );
    await dispatch(switchPageLoading(false));
  };

  useEffect(() => {
    pageInit();
    return () => {
      clearPageState();
    };
  }, []);

  const onChange = async ({ current, pageSize }: PageInfo) => {
    await dispatch(
      getList({
        pageSize,
        current,
      }),
    );
  };

  const handlePageSizeChange = async (pageSize: number, { current }: PageInfo) => {
    await dispatch(
      getList({
        pageSize,
        current,
      }),
    );
  };

  return (
    <div>
      <div className={Style.toolBar}>
        <Button>新建产品</Button>
        <Input className={Style.search} suffixIcon={<SearchIcon />} placeholder='请输入你需要搜索的内容' />
      </div>
      {pageState.pageLoading ? (
        <div className={Style.loading}>
          <Loading text='加载数据中...' loading size='large' />
        </div>
      ) : (
        <>
          <div className={Style.cardList}>
            <Row gutter={[16, 12]}>
              {pageState?.productList?.map((product, index) => (
                <Col key={index} span={6} lg={4}>
                  <ProductCard product={product} />
                </Col>
              ))}
            </Row>
          </div>
          <Pagination
            className={Style.pagination}
            total={pageState.total}
            pageSizeOptions={[12, 24, 36]}
            pageSize={pageState.pageSize}
            onChange={onChange}
            onPageSizeChange={handlePageSizeChange}
          />
        </>
      )}
    </div>
  );
};

export default React.memo(CardList);
