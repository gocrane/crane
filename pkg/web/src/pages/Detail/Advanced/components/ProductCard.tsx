import React from 'react';
import { Progress, Tag } from 'tdesign-react';
import { AddIcon, ChevronRightIcon, Icon } from 'tdesign-icons-react';
import Styles from './ProductCard.module.less';

interface IProps {
  isAdd?: boolean;
  title?: string;
  tags?: string[];
  desc?: string;
  progress?: number;
  percent?: string;
  Icon?: string;
  color?: string;
  trackColor?: string;
}

const ProductCard = (props: IProps) => (
  <div className={Styles.productCard}>
    {props.isAdd ? (
      <div className={Styles.productionAdd}>
        <div className={Styles.productionAddBtn}>
          <AddIcon className={Styles.productionAddIcon} />
          <span>新增产品</span>
        </div>
      </div>
    ) : (
      <div className={Styles.productionCard}>
        <div className={Styles.productionTitle}>
          <Icon name={props.Icon} className={Styles.productionIcon} />
          <div className={Styles.title}>{props.title}</div>
          <div className={Styles.tags}>
            {props?.tags?.map((tag, index) => (
              <Tag key={index} className={Styles.tag} theme='success' variant='dark' size='small'>
                {tag}
              </Tag>
            ))}
          </div>
        </div>
        <div className={Styles.item}>
          <span className={Styles.info}>{props.desc}</span>
          <ChevronRightIcon className={Styles.icon} />
        </div>
        <div className={Styles.footer}>
          <span className={Styles.percent}>{props.percent}</span>
          <div className={Styles.progress}>
            <Progress percentage={props.progress} label={false} color={props.color} trackColor={props.trackColor} />
          </div>
        </div>
      </div>
    )}
  </div>
);

export default React.memo(ProductCard);
