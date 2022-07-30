import React, { memo } from 'react';
import { Button, Form, Input } from 'tdesign-react';

const { FormItem } = Form;

export default memo((props: { current: number; callback: Function; steps: any[] }) => {
  const { current, callback, steps = [] } = props;

  const next = () => {
    callback('next');
  };

  const prev = () => {
    callback('back');
  };

  return (
    <Form labelWidth={100}>
      <FormItem label='发票抬头' name='invoice' rules={[{ required: true, message: '请输入发票抬头', type: 'error' }]}>
        <Input placeholder='请输入发票抬头' />
      </FormItem>

      <FormItem
        label='纳税人识别号'
        name='taxpayerId'
        rules={[{ required: true, message: '请输入纳税人识别号', type: 'error' }]}
      >
        <Input placeholder='请输入纳税人识别号' />
      </FormItem>

      <FormItem label='单位地址' name='address'>
        <Input placeholder='请输入单位地址' />
      </FormItem>

      <FormItem label='开户行' name='bank'>
        <Input placeholder='请输入开户行' />
      </FormItem>

      <FormItem label='银行账号' name='bankCount'>
        <Input placeholder='请输入银行账号' />
      </FormItem>

      <FormItem label='通知邮箱' name='email'>
        <Input placeholder='请输入通知邮箱' />
      </FormItem>

      <FormItem label='通知手机' name='phone'>
        <Input placeholder='请输入通知手机' />
      </FormItem>

      <FormItem>
        {current < steps.length - 1 && (
          <Button type='submit' onClick={() => next()}>
            下一步
          </Button>
        )}

        {current > 0 && (
          <Button style={{ margin: '0 8px' }} onClick={() => prev()}>
            上一步
          </Button>
        )}
      </FormItem>
    </Form>
  );
});
