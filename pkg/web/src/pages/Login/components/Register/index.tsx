import React, { useState, useRef } from 'react';
import classnames from 'classnames';
import { Form, MessagePlugin, Input, Checkbox, Button, FormInstanceFunctions, SubmitContext } from 'tdesign-react';
import { LockOnIcon, UserIcon, MailIcon, BrowseOffIcon, BrowseIcon } from 'tdesign-icons-react';
import useCountdown from '../../hooks/useCountDown';

import Style from './index.module.less';

const { FormItem } = Form;

export type ERegisterType = 'phone' | 'email';

export default function Register() {
  const [registerType, changeRegisterType] = useState('phone');
  const [showPsw, toggleShowPsw] = useState(false);
  const { countdown, setupCountdown } = useCountdown(60);
  const formRef = useRef<FormInstanceFunctions>();

  const onSubmit = (e: SubmitContext) => {
    if (e.validateResult === true) {
      const { checked } = formRef.current?.getFieldsValue?.(['checked']) as { checked: boolean };
      if (!checked) {
        MessagePlugin.error('请同意 TDesign 服务协议和 TDesign 隐私声明');
        return;
      }
      MessagePlugin.success('注册成功');
    }
  };

  const switchType = (val: ERegisterType) => {
    formRef.current?.reset?.();
    changeRegisterType(val);
  };

  return (
    <div>
      <Form
        ref={formRef}
        className={classnames(Style.itemContainer, `register-${registerType}`)}
        labelWidth={0}
        onSubmit={onSubmit}
      >
        {registerType === 'phone' && (
          <FormItem name='phone' rules={[{ required: true, message: '手机号必填', type: 'error' }]}>
            <Input maxlength={11} size='large' placeholder='请输入您的手机号' prefixIcon={<UserIcon />} />
          </FormItem>
        )}

        {registerType === 'email' && (
          <FormItem
            name='email'
            rules={[
              { required: true, message: '邮箱必填', type: 'error' },
              { email: true, message: '请输入正确的邮箱', type: 'warning' },
            ]}
          >
            <Input type='text' size='large' placeholder='请输入您的邮箱' prefixIcon={<MailIcon />} />
          </FormItem>
        )}

        <FormItem name='password' rules={[{ required: true, message: '密码必填', type: 'error' }]}>
          <Input
            size='large'
            type={showPsw ? 'text' : 'password'}
            clearable
            placeholder='请输入登录密码'
            prefixIcon={<LockOnIcon />}
            suffixIcon={
              showPsw ? (
                <BrowseIcon onClick={() => toggleShowPsw((current) => !current)} />
              ) : (
                <BrowseOffIcon onClick={() => toggleShowPsw((current) => !current)} />
              )
            }
          />
        </FormItem>
        {registerType === 'phone' && (
          <FormItem name='verifyCode' rules={[{ required: true, message: '验证码必填', type: 'error' }]}>
            <Input size='large' placeholder='请输入验证码' />
            <Button
              variant='outline'
              className={Style.verificationBtn}
              disabled={countdown > 0}
              onClick={setupCountdown}
            >
              {countdown === 0 ? '发送验证码' : `${countdown}秒后可重发`}
            </Button>
          </FormItem>
        )}
        <FormItem className={Style.checkContainer} name='checked' initialData={false}>
          <Checkbox>我已阅读并同意 </Checkbox> <span className='tip'>TDesign服务协议</span> 和
          <span className='tip'>TDesign 隐私声明</span>
        </FormItem>
        <FormItem>
          <Button block size='large' type='submit'>
            注册
          </Button>
        </FormItem>
        <div className={Style.switchContainer}>
          <span className={Style.switchTip} onClick={() => switchType(registerType === 'phone' ? 'email' : 'phone')}>
            {registerType === 'phone' ? '使用邮箱注册' : '使用手机号注册'}
          </span>
        </div>
      </Form>
    </div>
  );
}
