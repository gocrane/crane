import React from 'react';
import { Card, Avatar, Tag, Dropdown, Button } from 'tdesign-react';
import { UserAvatarIcon, CalendarIcon, LaptopIcon, ShopIcon, ServiceIcon, Icon } from 'tdesign-icons-react';
import { IProduct } from 'services/product';
import Style from './ProductCard.module.less';

const { Group: AvatarGroup } = Avatar;
const icons = [UserAvatarIcon, CalendarIcon, LaptopIcon, ShopIcon, ServiceIcon];

const CardIcon = React.memo(() => {
  const random = Math.floor(Math.random() * icons.length);
  const Icon = icons[random];
  return <Icon />;
});

const ProductCard = ({ product }: { product: IProduct }) => {
  const disabled = !product.isSetup;
  return (
    <Card
      className={Style.panel}
      actions={
        disabled ? (
          <Tag theme='default' disabled={true}>
            已停用
          </Tag>
        ) : (
          <Tag theme='success'>已启用</Tag>
        )
      }
      avatar={
        <Avatar size='56px'>
          <CardIcon />
        </Avatar>
      }
      footer={
        <div className={Style.footer}>
          <AvatarGroup cascading='left-up'>
            <Avatar>{String.fromCharCode(64 + product.type || 0)}</Avatar>
            <Avatar>+</Avatar>
          </AvatarGroup>
          <Dropdown
            trigger={'click'}
            options={[
              {
                content: '管理',
                value: 1,
              },
              {
                content: '删除',
                value: 2,
              },
            ]}
          >
            <Button theme='default' variant='text' disabled={disabled}>
              <Icon name='more' size='16' />
            </Button>
          </Dropdown>
        </div>
      }
    >
      <div className={Style.name}>{product?.name}</div>
      <div className={Style.description}>{product?.description}</div>
    </Card>
  );
};

export default React.memo(ProductCard);
