module.exports = {
  env: {
    browser: true,
    es2021: true
  },
  extends: [
    'plugin:react/recommended',
    'google',
    'prettier',
    'plugin:react/jsx-runtime',
    'plugin:react-hooks/recommended'
  ],
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaFeatures: {
      jsx: true
    },
    ecmaVersion: 'latest',
    sourceType: 'module'
  },
  plugins: ['react', '@typescript-eslint'],
  rules: {
    'react/display-name': [0],
    'no-unused-vars': [1],
    'valid-jsdoc': [0],
    'require-jsdoc': [0],
    'guard-for-in': [1],
    'prefer-spread': [1],
    'react/jsx-sort-props': [2, { callbacksLast: true }]
  }
};
