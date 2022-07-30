import { useSelector } from 'hooks';
import { editClusterActions } from 'modules/editClusterSlice';
import { overviewActions } from 'modules/overviewSlice';
import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { Button, Input } from 'tdesign-react';

export const OverviewActionPanel = React.memo(() => {
  const { t } = useTranslation();
  const dispatch = useDispatch();

  const clusters = useSelector((state) => state.editCluster.clusters);
  const searchText = useSelector((state) => state.overview.searchText);

  return (
    <div style={{ display: 'flex', flexDirection: 'row', justifyContent: 'space-between' }}>
      <Button
        type='button'
        onClick={() => {
          dispatch(editClusterActions.editingClusterId(clusters[0].id));
          dispatch(editClusterActions.modalVisible(true));
          dispatch(editClusterActions.mode('create'));
        }}
      >
        {t('添加集群')}
      </Button>
      <div style={{ width: '300px', display: 'inline-block' }}>
        <Input
          placeholder={t('支持搜索集群ID和集群名称')}
          value={searchText}
          onChange={(value) => {
            dispatch(overviewActions.searchText(value));
          }}
        />
        {/* <TagSearchBox
            attributes={attributes}
            minWidth={400}
            value={toTagValues(seachFilter, attributes)}
            onChange={tags => {
              dispatch(overviewActions.searchFilter(fromTagValues(tags, attributes)));
            }}
          /> */}
      </div>
    </div>
  );
});
