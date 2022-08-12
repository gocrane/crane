import React from 'react';

export const Card = ({
  children,
  title = null,
  operations = null,
  style = {},
  className,
}: {
  children: React.ReactNode;
  title?: string | null;
  operations?: React.ReactNode;
  style?: React.CSSProperties;
  className?: string;
}) => (
  <div className={className} style={{ padding: 20, background: 'white', ...style }}>
    <div>
      {title}
      {operations}
    </div>
    {children}
  </div>
);
