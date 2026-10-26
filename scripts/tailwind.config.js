/**
 * Tailwind CSS 构建配置（仅在改动模板/样式后手动重建，运行期零 Node 依赖）。
 *
 * 产出：backend/internal/infrastructure/web/static/tailwind.css
 * 重建：npx --yes tailwindcss@3.4.3 -c scripts/tailwind.config.js \
 *         -i scripts/tailwind.input.css \
 *         -o backend/internal/infrastructure/web/static/tailwind.css --minify
 */
const path = require('path');

module.exports = {
  content: [
    path.join(__dirname, '../backend/internal/infrastructure/web/templates/*.html'),
    path.join(__dirname, '../backend/internal/infrastructure/web/*.go'),
  ],
  darkMode: 'class',
  theme: {
    extend: {},
  },
  plugins: [],
};
