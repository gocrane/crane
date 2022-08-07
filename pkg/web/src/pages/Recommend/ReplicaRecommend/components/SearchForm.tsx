import React, { memo, useRef } from 'react';
import { Button, Col, Form, Input, MessagePlugin, Row, Select } from 'tdesign-react';
import { FormInstanceFunctions, SubmitContext } from 'tdesign-react/es/form/type';
import _ from 'lodash';

const { FormItem } = Form;

export type SearchFormProps = {
  recommendation: any;
  setFilterParams: any;
};

const SearchForm: React.FC<SearchFormProps> = ({ recommendation, setFilterParams }) => {
  const formRef = useRef<FormInstanceFunctions>();
  const onValuesChange = (changeValues: any, allValues: { name: any; namespace: any; workloadType: any }) => {
    if (!allValues.name) delete allValues.name;
    if (!allValues.namespace) delete allValues.namespace;
    if (!allValues.workloadType) delete allValues.workloadType;
    setFilterParams(allValues);
  };

  const onReset = () => setFilterParams({});

  const nameSpaceOptions = _.uniqBy(
    recommendation.map((r: { namespace: any; label: any }) => ({ value: r.namespace, label: r.label })),
    'value',
  );
  const workloadTypeOptions = _.uniqBy(
    recommendation.map((r: { workloadType: any }) => ({ value: r.workloadType, label: r.workloadType })),
    'value',
  );

  return (
    <div className='list-common-table-query'>
      <Form ref={formRef} onValuesChange={onValuesChange} onReset={onReset} labelWidth={80} layout={'inline'} >
        <Row>
          <Col>
            <Row>
              <Col>
                <FormItem label='推荐名称' name='name'>
                  <Input placeholder='请输入推荐名称' />
                </FormItem>
              </Col>
              <Col>
                <FormItem label='NameSpace' name='namespace' style={{ margin: '0px 10px' }}>
                  <Select
                    options={nameSpaceOptions}
                    placeholder='请选择NameSpace'
                    autoWidth={true}
                    style={{ margin: '0px 20px' }}
                  />
                </FormItem>
              </Col>
              <Col>
                <FormItem label='工作负载类型' name='workloadType'>
                  <Select
                    options={workloadTypeOptions}
                    placeholder='请选择工作负载类型'
                    autoWidth={true}
                    style={{ margin: '0px 20px' }}
                  />
                </FormItem>
              </Col>
            </Row>
          </Col>
          <Col>
            <Button type='reset' variant='base' theme='default'>
              重置
            </Button>
          </Col>
        </Row>
      </Form>
    </div>
  );
};

export default memo(SearchForm);
