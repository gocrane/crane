import { AttributeValue, Button, Justify, Table, TagSearchBox } from 'tea-component';

import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';

import { useSelector } from '../../hooks/useSelector';
import { editClusterActions } from '../../store/editClusterSlice';
import { overviewActions } from '../../store/overviewSlice';
import { toTagValues, fromTagValues } from './../../utils/tagSearchBoxValues';

export const OverviewActionPanel = React.memo(() => {
  const seachFilter = useSelector(state => state.overview.searchFilter);
  const { t } = useTranslation();
  const dispatch = useDispatch();

  const attributes: AttributeValue[] = [
    {
      type: 'input',
      key: 'clusterNames',
      name: t('集群名称')
    },
    {
      type: 'input',
      key: 'clusterIds',
      name: t('集群ID')
    }
  ];

  return (
    <Table.ActionPanel>
      <Justify
        left={
          <Button
            type="primary"
            onClick={() => {
              dispatch(editClusterActions.modalVisible(true));
              dispatch(editClusterActions.mode('create'));
            }}
          >
            {t('添加集群')}
          </Button>
        }
        right={
          <div style={{ width: 400, display: 'inline-block' }}>
            <TagSearchBox
              attributes={attributes}
              minWidth={400}
              value={toTagValues(seachFilter, attributes)}
              onChange={tags => {
                dispatch(overviewActions.searchFilter(fromTagValues(tags, attributes)));
              }}
            />
          </div>
        }
      />
    </Table.ActionPanel>
  );
});
