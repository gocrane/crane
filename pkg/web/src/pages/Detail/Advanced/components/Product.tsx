import React from 'react';
import { Col, Row, Card } from 'tdesign-react';
import ProductCard from './ProductCard';
import Style from '../index.module.less';

const Product = () => (
  <Card title='产品目录' className={Style.cardBox} header>
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={12} xl={4}>
          <ProductCard isAdd={true} />
        </Col>
        <Col xs={12} xl={4}>
          <ProductCard
            title='MacBook Pro 2021'
            tags={['13.3 英寸', 'Apple M1', 'RAM 16GB']}
            desc='最高可选配 16GB 内存 · 最高可选配 2TB 存储设备 电池续航最长达 18 小时'
            percent='1420 / 1500（台）'
            Icon='cart'
            progress={(1420 / 1500) * 100}
            trackColor='#D4E3FC'
          />
        </Col>
        <Col xs={12} xl={4}>
          <ProductCard
            title='Surface Laptop Go'
            tags={['12.4 英寸', 'Core i7', 'RAM 16GB']}
            desc='常规使用 Surface，续航时间最长可达13小时 随时伴您工作'
            percent='120 / 2000（台）'
            Icon='cart'
            progress={(120 / 2000) * 100}
            color='#E24D59'
            trackColor='#FCD4D4'
          />
        </Col>
      </Row>
    </div>
  </Card>
);

export default React.memo(Product);
