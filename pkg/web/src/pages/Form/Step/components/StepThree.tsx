import React, { memo } from 'react';
import { Button, Form, Select, Input, Textarea } from 'tdesign-react';

const { FormItem } = Form;
const { Option } = Select;

const addressOptions = [
  {
    label: '广东省深圳市南山区',
    value: '0',
  },
  {
    label: '北京市海淀区',
    value: '1',
  },
  {
    label: '四川省成都市高新区',
    value: '2',
  },
  {
    label: '广东省广州市天河区',
    value: '3',
  },
  {
    label: '陕西省西安市高新区',
    value: '4',
  },
];

export default memo((props: { current: number; callback: Function; steps: any[] }) => {
  const { current, callback, steps } = props;

  const next = () => {
    callback('next');
  };

  const prev = () => {
    callback('back');
  };

  return (
    <Form labelWidth={100}>
      <FormItem label='收货人' name='receiver' rules={[{ required: true, message: '请输入收货人', type: 'error' }]}>
        <Input placeholder='请输入收货人' />
      </FormItem>

      <FormItem
        label='收货人手机号'
        name='receiverPhone'
        rules={[{ required: true, message: '请输入收货人手机号', type: 'error' }]}
      >
        <Input placeholder='请输入收货人手机号号' />
      </FormItem>

      <FormItem
        label='收货地址'
        name='receiverAddress'
        rules={[{ required: true, message: '请选择收货地址', type: 'error' }]}
      >
        <Select value='3' placeholder='请选择收货地址'>
          {addressOptions.map((item: { label: string; value: string }) => {
            const { label, value } = item;
            return <Option key={value} label={label} value={value} />;
          })}
        </Select>
      </FormItem>

      <FormItem
        label='详细地址'
        name='taxpayerId'
        rules={[{ required: true, message: '请输入详细地址', type: 'error' }]}
      >
        <Textarea placeholder='请输入详细地址' value={'哈哈哈'} />
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
