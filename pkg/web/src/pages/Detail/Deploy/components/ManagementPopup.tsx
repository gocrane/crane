import React, { useState } from 'react';
import { Dialog } from 'tdesign-react';
import classnames from 'classnames';
import { BASE_INFO_DATA } from '../constant';
import Style from './ManagementPopup.module.less';

interface IProps {
  visible: boolean;
}

const ManagementPopup = ({ visible }: IProps): React.ReactElement => {
  const [isShow, setVisible] = useState<boolean>(visible);
  const handleConfirm = () => setVisible(!isShow);

  return (
    <Dialog
      header='基本信息'
      visible={isShow}
      onClose={handleConfirm}
      onConfirm={handleConfirm}
      onCancel={handleConfirm}
    >
      <div>
        <div className={Style.popupBox}>
          {BASE_INFO_DATA.map((item, index) => (
            <div key={index} className={Style.popupItem}>
              <h1>{item.name}</h1>
              <p
                className={classnames({
                  [Style.popupItem_green]: item.type && item.type.value === 'green',
                  [Style.popupItem_blue]: item.type && item.type.value === 'blue',
                })}
              >
                {item.value}
              </p>
            </div>
          ))}
        </div>
      </div>
    </Dialog>
  );
};

export default React.memo(ManagementPopup);
