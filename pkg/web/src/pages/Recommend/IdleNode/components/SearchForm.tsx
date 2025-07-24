import React, { memo } from 'react';
import { Button, Col, Form, Input, Row } from 'tdesign-react';
import { useTranslation } from 'react-i18next';

const { FormItem } = Form;

export type SearchFormProps = {
  recommendation: any;
  setFilterParams: any;
};

const SearchForm: React.FC<SearchFormProps> = ({ setFilterParams }) => {
  const { t } = useTranslation();
  const onValuesChange = (changeValues: any, allValues: any) => {
    if (!allValues.name) delete allValues.name;
    if (!allValues.namespace) delete allValues.namespace;
    if (!allValues.workloadType) delete allValues.workloadType;
    setFilterParams(allValues);
  };

  const onReset = () => setFilterParams({});

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
