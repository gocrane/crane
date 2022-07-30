import React from 'react';
import { defaultColor } from 'configs/color';
import Style from './RadioColor.module.less';

interface IProps {
  defaultValue?: number | string;
  onChange: (color: string) => void;
}

const RadioColor = (props: IProps) => (
  <div className={Style.panel}>
    {defaultColor.map((color, index) => (
      <div
        key={index}
        onClick={() => props?.onChange(color)}
        className={Style.box}
        style={{ borderColor: props.defaultValue === color ? color : 'transparent' }}
      >
        <div className={Style.item} style={{ backgroundColor: color }} />
      </div>
    ))}
  </div>
);

export default React.memo(RadioColor);
