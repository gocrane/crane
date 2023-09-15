import React, { memo } from 'react';
import { Button, Col, Form, Input, Row, Select } from 'tdesign-react';
import { MinusCircleIcon } from "tdesign-icons-react";
import _ from 'lodash';
import { useTranslation } from 'react-i18next';

const { FormItem ,FormList} = Form;

export type SearchFormProps = {
  recommendation: any;
  setFilterParams: any;
  showFilter:Boolean
};
const SearchForm: React.FC<SearchFormProps> = ({ recommendation, setFilterParams ,showFilter}) => {
  const { t } = useTranslation();
  const onValuesChange = (changeValues: any, allValues: any) => {
    if (!allValues.name) delete allValues.name;
    if (!allValues.namespace) delete allValues.namespace;
    if (!allValues.workloadType) delete allValues.workloadType;
    let filter_options: any=[];
    if(allValues.filter_options_list.length){
      allValues.filter_options_list.map((item: any)=>{
        if(item.resource&&item.quotaType&& item.compareExpr &&item.ratio){
          filter_options.push(`${item.resource},${item.quotaType},${item.compareExpr },${item.ratio}`);
        }
      })
    }
    allValues.filter_options=filter_options.join('@');
    if (!allValues.filter_options) delete allValues.filter_options;
    delete allValues.filter_options_list;
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

  const resourceOptions = [{
    label:'cpu',value:'cpu'
  },{
    label:'mem',value:'mem'
  }];

  const quotaTypeOptions = [{
    label:'limit',value:'limit'
  },{
    label:'request',value:'request'
  }];
  
  const compareExprOptions = [{
    label:'eq(=)',value:'eq'
  },{
    label:'gt(>)',value:'gt'
  },{
    label:'ge(>=)',value:'ge'
  },{
    label:'lt(<)',value:'lt'
  },{
    label:'le(<=)',value:'le'
  }];

  return (
    <div className='list-common-table-query'>
      <Form onValuesChange={onValuesChange} onReset={onReset} labelWidth={80} layout={'inline'}>
        <FormItem label={t('推荐名称')} name='name'>
          <Input placeholder={t('请输入推荐名称')} />
        </FormItem>
        <FormItem label={t('Namespace')} name='namespace' style={{ margin: '0px 10px' }}>
          <Select
            options={nameSpaceOptions}
            placeholder={t('请选择Namespace')}
            style={{ margin: '0px 20px' }}
          />
        </FormItem>
        <FormItem label={t('工作负载类型')} name='workloadType'>
          <Select
            options={workloadTypeOptions}
            placeholder={t('请选择工作负载类型')}
            style={{ margin: '0px 20px' }}
          />
        </FormItem>
        <FormList name="filter_options_list" style={{display:showFilter?'block':'none'}}>
          {(fields, {add, remove }) => (
            <>
              {fields.map(({ key, name }) => (
                <FormItem key={key}>
                  <FormItem label={t('资源类型')} name={[name, 'resource']} >
                    <Select
                      options={resourceOptions}
                      placeholder={t('请选择资源类型')}
                    />
                  </FormItem>
                  <FormItem label={t('限制类型')} name={[name, 'quotaType']}>
                    <Select
                      options={quotaTypeOptions}
                      placeholder={t('请选择限制类型')}
                      style={{ margin: '0px 20px' }}
                    />
                  </FormItem>
                  <FormItem label={t('比较表达式')} name={[name, 'compareExpr']} >
                    <Select
                      options={compareExprOptions}
                      placeholder={t('请选择比较表达式')}
                      style={{ margin: '0px 20px' }}
                    />
                  </FormItem>
                  <FormItem label={t('比值')} name={[name, 'ratio']}>
                    <Input placeholder={t('请输入比值')} />
                  </FormItem>
                  <FormItem>
                    <MinusCircleIcon size="20px" style={{ cursor: 'pointer' }} onClick={() => remove(name)} />
                  </FormItem>
                </FormItem>
              ))}
              <FormItem style={{ marginLeft: 100 }}>
                <Button theme="default" variant="dashed" onClick={() => add({})}>
                  添加过滤条件
                </Button>
              </FormItem>
            </>
          )}
        </FormList>
        
        <FormItem>
          <Button type='reset' theme='default'>
            {t('重置')}
          </Button>
        </FormItem>
      </Form>
    </div>
  );
};

export default memo(SearchForm);
