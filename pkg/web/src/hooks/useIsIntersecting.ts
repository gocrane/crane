import React from 'react';

export const useIsIntersecting = (ref: { current: Element }) => {
  const [isIntersecting, setIntersecting] = React.useState(false);

  const observer = React.useMemo(
    () => new IntersectionObserver(([entry]) => setIntersecting(entry.isIntersecting)),
    [],
  );

  // eslint-disable-next-line consistent-return
  React.useEffect(() => {
    if (ref && ref.current) {
      observer.observe(ref.current);
      // Remove the observer as soon as the component is unmounted
      return () => {
        observer.disconnect();
      };
    }
  }, [observer, ref]);

  return isIntersecting;
};
