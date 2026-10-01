import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import reactPlugin from 'eslint-plugin-react';
import reactHooksPlugin from 'eslint-plugin-react-hooks';
import jsxA11yPlugin from 'eslint-plugin-jsx-a11y';

export default tseslint.config(
  { ignores: ['node_modules/', '.next/', 'out/', 'public/', '*.config.js', '*.config.mjs', 'coverage/', 'dist/', 'build/'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    plugins: {
      react: reactPlugin,
      'react-hooks': reactHooksPlugin,
      'jsx-a11y': jsxA11yPlugin,
    },
    languageOptions: {
      parserOptions: {
        ecmaFeatures: { jsx: true },
        project: './tsconfig.json',
        tsconfigRootDir: import.meta.dirname,
      },
      globals: {
        browser: true,
        es2024: true,
        node: true,
        React: 'writable',
        JSX: 'writable',
      },
    },
    settings: {
      react: { version: '19.2' },
    },
    rules: {
      ...reactPlugin.configs.recommended.rules,
      ...reactHooksPlugin.configs.recommended.rules,
      ...jsxA11yPlugin.configs.recommended.rules,
      'react/react-in-jsx-scope': 'off',
      'react/prop-types': 'off',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
      '@typescript-eslint/no-explicit-any': 'warn',
      'jsx-a11y/anchor-is-valid': 'off',
      'jsx-a11y/click-events-have-key-events': 'warn',
      'jsx-a11y/no-noninteractive-element-interactions': 'warn',
      'jsx-a11y/no-static-element-interactions': 'warn',
      'jsx-a11y/label-has-associated-control': 'warn',
      // React Compiler 系规则（eslint-plugin-react-hooks v7 新增）。
      // set-state-in-effect 会命中「从异步数据/外部存储同步派生状态」这一
      // 官方文档同样认可的写法：本项目 result 页把收藏态从 React Query 结果
      // 同步进 state（且必须如此 —— 乐观更新要叠加本地覆盖），ThemeToggle
      // 在挂载时读 localStorage 定主题，都是同一模式。改造为 render 期派生
      // 会与乐观更新/回滚机制冲突，收益低于风险，故降为告警而非错误。
      // 真正的死循环风险已由 result 页的 sameFavoriteStatus 守卫堵死。
      'react-hooks/set-state-in-effect': 'warn',
    },
  },
  {
    // 构建脚本：跑在 Node 里，process/console 是合法全局变量，
    // 不能按浏览器的 globals 规则判 no-undef
    files: ['scripts/**/*.mjs', 'scripts/**/*.js'],
    languageOptions: {
      ecmaVersion: 2024,
      sourceType: 'module',
      globals: {
        process: 'readonly',
        console: 'readonly',
        Buffer: 'readonly',
        URL: 'readonly',
      },
    },
    rules: {
      // 纯 Node 脚本，不涉及 React/TS 规则
      'no-undef': 'error',
    },
  },
  {
    files: ['**/*.test.{ts,tsx}', '**/*.spec.{ts,tsx}'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': 'off',
    },
  }
);