import React, { memo } from 'react';
import Base from './components/Base';
import ProgressComp from './components/Progress';
import Product from './components/Product';
import Detail from './components/Detail';

export default memo(() => (
  <div>
    <Base />
    <ProgressComp />
    <Product />
    <Detail />
  </div>
));
