import React from 'react';
import { Tooltip } from 'tdesign-react';

export interface HeadBubbleProps {
  /** 显示标题 */
  title?: string | React.ReactElement;

  /** 显示的文本 */
  text?: string | React.ReactElement;

  /** 气泡显示方式 */
  position?: 'top' | 'bottom' | 'left' | 'right';

  /** 对齐方式 */
  align?: 'start' | 'end';

  /** 用于title隐藏 */
  autoflow?: boolean;
}

export const HeadBubble = React.memo((props: HeadBubbleProps) => {
  const { title = '', text = '', position, autoflow } = props;
  return (
    <div>
      {autoflow ? <span className="text-overflow">{title}</span> : <span>{title}</span>}
      <Tooltip content={<p style={{ fontWeight: 'normal' }}>{text}</p>} placement={position || 'top'}>
        <span className="tc-15-bubble-icon">
          <i className="tc-icon icon-what" />
        </span>
      </Tooltip>
    </div>
  );
});
