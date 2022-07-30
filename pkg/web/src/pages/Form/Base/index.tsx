import React, { memo, useRef } from 'react';
import {
  Form,
  Row,
  Col,
  Input,
  Radio,
  Button,
  DatePicker,
  Select,
  Textarea,
  Avatar,
  Upload,
  MessagePlugin,
} from 'tdesign-react';
import classnames from 'classnames';
import { SubmitContext, FormInstanceFunctions } from 'tdesign-react/es/form/type';
import CommonStyle from 'styles/common.module.less';
import Style from './index.module.less';

const { FormItem } = Form;
const { Option } = Select;
const { Group } = Avatar;

const INITIAL_DATA = {
  name: '',
  type: '',
  payment: '',
  partyA: '',
  partyB: '',
  signDate: '',
  effectiveDate: '',
  endDate: '',
  remark: '',
  notary: '',
  file: [],
};

export default memo(() => {
  const formRef = useRef<FormInstanceFunctions>();

  const onSubmit = (e: SubmitContext) => {
    if (e.validateResult === true) {
      console.log('form 值', formRef.current?.getFieldsValue?.(true));
      MessagePlugin.info('提交成功');
    }
  };

  const handleFail = ({ file }: { file: any }) => {
    console.error(`文件 ${file.name} 上传失败`);
  };

  return (
    <div className={classnames(CommonStyle.pageWithColor)}>
      <div className={Style.formContainer}>
        <Form ref={formRef} onSubmit={onSubmit} labelWidth={100} labelAlign='top'>
          <div className={Style.titleBox}>
            <div className={Style.titleText}>合同信息</div>
          </div>
          <Row gutter={[32, 24]}>
            <Col span={6}>
              <FormItem
                label='合同名称'
                name='name'
                initialData={INITIAL_DATA.name}
                rules={[{ required: true, message: '合同名称必填', type: 'error' }]}
              >
                <Input placeholder='请输入内容' />
              </FormItem>
            </Col>

            <Col span={6}>
              <FormItem
                label='合同类型'
                name='type'
                initialData={INITIAL_DATA.type}
                rules={[{ required: true, message: '合同类型必填', type: 'error' }]}
              >
                <Select placeholder='请选择类型'>
                  <Option key='A' label='类型A' value='A' />
                  <Option key='B' label='类型B' value='B' />
                  <Option key='C' label='类型C' value='C' />
                </Select>
              </FormItem>
            </Col>

            <Col span={12}>
              <FormItem
                label='合同收付类型'
                name='payment'
                initialData={INITIAL_DATA.payment}
                rules={[{ required: true }]}
              >
                <Radio.Group>
                  <Radio value='0'>收款</Radio>
                  <Radio value='1'>付款</Radio>
                </Radio.Group>
                <Input placeholder='请输入金额' style={{ width: 160 }} />
              </FormItem>
            </Col>

            <Col span={6}>
              <FormItem label='甲方' name='partyA' initialData={INITIAL_DATA.partyA} rules={[{ required: true }]}>
                <Select placeholder='请选择类型'>
                  <Option key='A' label='公司A' value='A' />
                  <Option key='B' label='公司B' value='B' />
                  <Option key='C' label='公司C' value='C' />
                </Select>
              </FormItem>
            </Col>

            <Col span={6}>
              <FormItem label='乙方' name='partyB' initialData={INITIAL_DATA.partyB} rules={[{ required: true }]}>
                <Select value='A' placeholder='请选择类型'>
                  <Option key='A' label='公司A' value='A' />
                  <Option key='B' label='公司B' value='B' />
                  <Option key='C' label='公司C' value='C' />
                </Select>
              </FormItem>
            </Col>

            <Col span={6} className={Style.dateCol} rules={[{ required: true }]}>
              <FormItem label='合同签订日期' name='signDate' initialData={INITIAL_DATA.signDate}>
                <DatePicker mode='date' />
              </FormItem>
            </Col>

            <Col span={6} className={Style.dateCol} rules={[{ required: true }]}>
              <FormItem label='合同生效日期' name='effectiveDate' initialData={INITIAL_DATA.effectiveDate}>
                <DatePicker mode='date' />
              </FormItem>
            </Col>

            <Col span={6} className={Style.dateCol} rules={[{ required: true }]}>
              <FormItem label='合同结束日期' name='endDate' initialData={INITIAL_DATA.endDate}>
                <DatePicker mode='date' />
              </FormItem>
            </Col>

            <Col span={6}>
              <FormItem label='合同文件' name='file' initialData={INITIAL_DATA.file}>
                <Upload
                  onFail={handleFail}
                  tips='请上传pdf文件，大小在60M以内'
                  action='//service-bv448zsw-1257786608.gz.apigw.tencentcs.com/api/upload-demo'
                />
              </FormItem>
            </Col>
          </Row>

          <div className={Style.titleBox}>
            <div className={Style.titleText}>其他信息</div>
          </div>

          <FormItem label='备注' name='remark' initialData={INITIAL_DATA.remark}>
            <Textarea placeholder='请输入备注' />
          </FormItem>

          <FormItem label='公证人' name='notary' initialData={INITIAL_DATA.notary}>
            <Group>
              <Avatar>D</Avatar>
              <Avatar>S</Avatar>
              <Avatar>+</Avatar>
            </Group>
          </FormItem>

          <FormItem>
            <Button type='submit' theme='primary'>
              提交
            </Button>
            <Button type='reset' style={{ marginLeft: 12 }}>
              重置
            </Button>
          </FormItem>
        </Form>
      </div>
    </div>
  );
});
