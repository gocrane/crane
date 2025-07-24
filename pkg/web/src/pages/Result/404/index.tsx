import React from 'react';
import ErrorPage from 'components/ErrorPage';

const NotFound = () => <ErrorPage code={404} />;

export default React.memo(NotFound);
