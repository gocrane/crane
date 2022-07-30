import React, { memo } from 'react';
import MonthPurchase from './components/MonthPurchase';
import PurchaseTrend from './components/PurchaseTrend';
import PurchaseSatisfaction from './components/Satisfaction';

export default memo(() => (
  <div>
    <MonthPurchase />
    <PurchaseTrend />
    <PurchaseSatisfaction />
  </div>
));
