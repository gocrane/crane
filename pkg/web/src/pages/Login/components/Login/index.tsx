import React, { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, MessagePlugin, Input, Checkbox, Button, FormInstanceFunctions, SubmitContext } from 'tdesign-react';
import { LockOnIcon, UserIcon, BrowseOffIcon, BrowseIcon, RefreshIcon } from 'tdesign-icons-react';
import classnames from 'classnames';
import QRCode from 'qrcode.react';
import { useAppDispatch } from 'modules/store';
import { login } from 'modules/user';
import useCountdown from '../../hooks/useCountDown';

import Style from './index.module.less';

const { FormItem } = Form;

export type ELoginType = 'password' | 'phone' | 'qrcode';

export default function Login() {
  const [loginType, changeLoginType] = useState<ELoginType>('password');
  const [showPsw, toggleShowPsw] = useState(false);
  const { countdown, setupCountdown } = useCountdown(60);
  const formRef = useRef<FormInstanceFunctions>();
  const navigate = useNavigate();
  const dispatch = useAppDispatch();

  const onSubmit = async (e: SubmitContext) => {
    if (e.validateResult === true) {
      try {
        const formValue = formRef.current?.getFieldsValue?.(true) || {};
        await dispatch(login(formValue));

        MessagePlugin.success('登录成功');

        navigate('/dashboard/base');
      } catch (e) {
        console.log(e);
        MessagePlugin.error('登录失败');
      }
    }
  };

  const switchType = (val: ELoginType) => {
    formRef.current?.reset?.();
    changeLoginType(val);
  };

  return (
    <div>
      <Form
        ref={formRef}
        className={classnames(Style.itemContainer, `login-${loginType}`)}
        labelWidth={0}
        onSubmit={onSubmit}
      >
        {loginType === 'password' && (
          <>
            <FormItem name='account' rules={[{ required: true, message: '账号必填', type: 'error' }]}>
              <Input size='large' placeholder='请输入账号：admin' prefixIcon={<UserIcon />}></Input>
            </FormItem>
            <FormItem name='password' rules={[{ required: true, message: '密码必填', type: 'error' }]}>
              <Input
                size='large'
                type={showPsw ? 'text' : 'password'}
                clearable
                placeholder='请输入登录密码：admin'
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
            <div className={classnames(Style.checkContainer, Style.rememberPwd)}>
              <Checkbox>记住账号</Checkbox>
              <span className={Style.checkContainerTip}>忘记账号？</span>
            </div>
          </>
        )}

        {/* 扫码登陆 */}
        {loginType === 'qrcode' && (
          <>
            <div className={Style.tipContainer}>
              <span className='tip'>请使用微信扫一扫登录</span>
              <span className='refresh'>
                刷新 <RefreshIcon />
              </span>
            </div>
            <QRCode value='' size={200} />
          </>
        )}
        {/* // 手机号登陆 */}
        {loginType === 'phone' && (
          <>
            <FormItem name='phone' rules={[{ required: true, message: '手机号必填', type: 'error' }]}>
              <Input maxlength={11} size='large' placeholder='请输入您的手机号' prefixIcon={<UserIcon />} />
            </FormItem>
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
          </>
        )}
        {loginType !== 'qrcode' && (
          <FormItem className={Style.btnContainer}>
            <Button block size='large' type='submit'>
              登录
            </Button>
          </FormItem>
        )}
        <div className={Style.switchContainer}>
          {loginType !== 'password' && (
            <span className='tip' onClick={() => switchType('password')}>
              使用账号密码登录
            </span>
          )}
          {loginType !== 'qrcode' && (
            <span className='tip' onClick={() => switchType('qrcode')}>
              使用微信扫码登录
            </span>
          )}
          {loginType !== 'phone' && (
            <span className='tip' onClick={() => switchType('phone')}>
              使用手机号登录
            </span>
          )}
        </div>
      </Form>
    </div>
  );
}
