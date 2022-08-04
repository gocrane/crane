import React from 'react';
import ErrorPage from 'components/ErrorPage';

const UnAuthorized = () => <ErrorPage code={403} />;

export default React.memo(UnAuthorized);
