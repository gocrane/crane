import React, { useEffect, useState } from 'react';
import { Table, Card } from 'tdesign-react';
import { TableSort, TdPrimaryTableProps } from 'tdesign-react/es/table';
import request from 'utils/request';
import { TABLE_COLUMNS } from '../constant';
import ManagementPopup from './ManagementPopup';
import Style from './BottomTable.module.less';

const BottomTable = () => {
  const [sort, setSort] = useState<TableSort>({ sortBy: 'name', descending: true });
  const [visible, setVisible] = useState(false);
  const [{ tableData }, setTableData] = useState({ tableData: [] });
  const pagination = {
    pageSize: 10,
    total: tableData.length,
    pageSizeOptions: [],
  };

  useEffect(() => {
    request.get('/api/get-project-list').then((res: any) => {
      if (res.code === 0) {
        const { list = [] } = res.data;
        setTableData({ tableData: list });
      }
    });
  }, []);

  const removeRow = (rowIndex: number) => {
    console.log(' rowIndex = ', rowIndex);
    console.log(' tableData = ', tableData);

    tableData.splice(rowIndex, 1);
    setTableData({ tableData });
  };

  const getTableColumns = (columns: TdPrimaryTableProps['columns']): TdPrimaryTableProps['columns'] => {
    if (columns) {
      columns[4].cell = (context) => {
        const { rowIndex } = context;
        return (
          <>
            <a className={Style.operationLink} onClick={() => setVisible(!visible)}>
              管理
            </a>
            <a className={Style.operationLink} onClick={() => removeRow(rowIndex)}>
              删除
            </a>
          </>
        );
      };
    }
    return columns;
  };

  return (
    <>
      <Card title='项目列表' header>
        <Table
          columns={getTableColumns(TABLE_COLUMNS)}
          rowKey='index'
          pagination={pagination}
          data={tableData}
          sort={sort}
          onSortChange={(newSort: TableSort) => setSort(newSort)}
        />
      </Card>
      {visible && <ManagementPopup visible={visible} />}
    </>
  );
};

export default React.memo(BottomTable);
