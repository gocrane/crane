import React from 'react';
import ErrorPage from 'components/ErrorPage';

const ServerError = () => <ErrorPage code={500} />;

export default React.memo(ServerError);
