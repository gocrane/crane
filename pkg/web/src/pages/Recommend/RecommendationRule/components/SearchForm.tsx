import { RECOMMENDATION_RULE_TYPE_OPTIONS } from '../consts';
import React, { memo, useRef } from 'react';
import { Button, Col, Form, Input, MessagePlugin, Row, Select } from 'tdesign-react';
import { FormInstanceFunctions, SubmitContext } from 'tdesign-react/es/form/type';
import _ from 'lodash';
import { useDispatch } from 'react-redux';

const { FormItem } = Form;

export type FormValueType = {
  name?: string;
  status?: string;
  number?: string;
  time?: string;
  type?: string;
};

export type SearchFormProps = {
  setFilterParams: any;
  filterParams: any;
};

const SearchForm: React.FC<SearchFormProps> = ({ setFilterParams, filterParams }) => {
  const formRef = useRef<FormInstanceFunctions>();

  const onValuesChange = (changeValue, allValues) => {
    console.log(allValues);
    if (!allValues.name) delete allValues.name;
    if (!allValues.recommenderType) delete allValues.recommenderType;
    setFilterParams(allValues);
  };

  const onReset = () => {
    setFilterParams({});
  };

  return (
    <div className='list-common-table-query'>
      <Form ref={formRef} onValuesChange={onValuesChange} onReset={onReset} labelWidth={80} >
        <Row>
          <Col flex='1'>
            <Row gutter={16}>
              <Col>
                <FormItem label='推荐名称' name='name'>
                  <Input placeholder='请输入推荐名称(支持前缀模糊)' autoWidth={true} />
                </FormItem>
              </Col>
              <Col>
                <FormItem label='推荐类型' name='recommenderType'>
                  <Select options={RECOMMENDATION_RULE_TYPE_OPTIONS} placeholder='请选择推荐类型' />
                </FormItem>
              </Col>
            </Row>
          </Col>
          <Col flex='160px'>
            <Button type='reset' variant='base' theme='default' style={{ margin: '0px 50px' }}>
              重置
            </Button>
          </Col>
        </Row>
      </Form>
    </div>
  );
};

export default memo(SearchForm);
