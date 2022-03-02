import './App.css';

import React from 'react';
import { Provider } from 'react-redux';
import { BrowserRouter } from 'react-router-dom';

import { Router } from './routes/router';
import { store } from './store/store';

const App = () => (
  <React.Suspense fallback={<span>loading</span>}>
    <Provider store={store}>
      <BrowserRouter>
        <Router />
      </BrowserRouter>
    </Provider>
  </React.Suspense>
);

export default App;
