module.exports = {
  root: true,
  extends: [
    'airbnb-base',
    'prettier',
    'plugin:react/recommended',
    'plugin:import/typescript',
    'plugin:@typescript-eslint/recommended',
    // 'plugin:i18next/recommended',
  ],
  plugins: ['eslint-plugin-prettier', 'unused-imports'],
  parser: '@typescript-eslint/parser',
  env: {
    jest: true,
  },
  settings: {
    react: {
      version: 'detect',
    },
  },
  rules: {
    'no-shadow': 'off',
    'no-param-reassign': ['error', { props: false }],
    'no-console': 'off',
    'no-plusplus': [
      'error',
      {
        allowForLoopAfterthoughts: true,
      },
    ],
    'no-unused-vars': 0,
    '@typescript-eslint/no-unused-vars': 0,
    'unused-imports/no-unused-imports': 'error',
    'unused-imports/no-unused-vars': [
      'warn',
      { vars: 'all', varsIgnorePattern: '^_', args: 'after-used', argsIgnorePattern: '^_' },
    ],
    'prefer-destructuring': 0,
    'array-callback-return': 0,
    'no-useless-escape': 0,
    'react/display-name': 'off',
    // jsx 单引号
    'jsx-quotes': [2, 'prefer-single'],
    'import/no-cycle': 'off', // TODO: remove
    'import/extensions': 'off',
    'import/no-unresolved': 'off',
    'import/order': 'off',
    'import/prefer-default-export': 'off',
    'import/no-extraneous-dependencies': ['error', { devDependencies: true }],
    // 关闭variable必须全部大写规则
    '@typescript-eslint/naming-convention': [
      'error',
      {
        selector: 'variable',
        modifiers: ['const'],
        format: null,
      },
    ],
    '@typescript-eslint/no-empty-function': 'off',
    '@typescript-eslint/no-explicit-any': 'off',
    '@typescript-eslint/ban-types': 'off',
    // 统一eslint prettier配置
    'prettier/prettier': [
      'warn',
      {},
      {
        usePrettierrc: true,
      },
    ],
  },
};
