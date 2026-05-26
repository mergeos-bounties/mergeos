import { renderToString } from '@vue/server-renderer';
import { createApp } from './main.js';

export async function render() {
  const app = createApp();
  return renderToString(app);
}
