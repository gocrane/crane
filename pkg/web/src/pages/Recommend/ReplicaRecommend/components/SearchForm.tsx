import React, { memo } from 'react';
import { Button, Col, Form, Input, Row, Select } from 'tdesign-react';
import _ from 'lodash';
import { useTranslation } from 'react-i18next';

const { FormItem } = Form;

export type SearchFormProps = {
  recommendation: any;
  setFilterParams: any;
};

const SearchForm: React.FC<SearchFormProps> = ({ recommendation, setFilterParams }) => {
  const { t } = useTranslation();
  const onValuesChange = (changeValues: any, allValues: any) => {
    if (!allValues.name) delete allValues.name;
    if (!allValues.namespace) delete allValues.namespace;
    if (!allValues.workloadType) delete allValues.workloadType;
    setFilterParams(allValues);
  };

  const onReset = () => setFilterParams({});

  const nameSpaceOptions = _.uniqBy(
    recommendation.map((r: { namespace: any; label: any }) => ({ value: r.namespace, label: r.namespace })),
    'value',
  );

  const workloadTypeOptions = _.uniqBy(
    recommendation.map((r: { workloadType: any }) => ({ value: r.workloadType, label: r.workloadType })),
    'value',
  );

  return (
    <div className='list-common-table-query'>
      <Form onValuesChange={onValuesChange} onReset={onReset} labelWidth={80} layout={'inline'}>
        <Row>
          <Col>
            <Row>
              <Col>
                <FormItem label={t('推荐名称')} name='name'>
                  <Input placeholder={t('请输入推荐名称')} />
                </FormItem>
              </Col>
              <Col>
                <FormItem label={t('NameSpace')} name='namespace' style={{ margin: '0px 10px' }}>
                  <Select
                    options={nameSpaceOptions}
                    placeholder={t('请选择NameSpace')}
                    style={{ margin: '0px 20px' }}
                  />
                </FormItem>
              </Col>
              <Col>
                <FormItem label={t('工作负载类型')} name='workloadType'>
                  <Select
                    options={workloadTypeOptions}
                    placeholder={t('请选择工作负载类型')}
                    style={{ margin: '0px 20px' }}
                  />
                </FormItem>
              </Col>
            </Row>
          </Col>
          <Col>
            <Button type='reset' variant='base' theme='default'>
              {t('重置')}
            </Button>
          </Col>
        </Row>
      </Form>
    </div>
  );
};

export default memo(SearchForm);
