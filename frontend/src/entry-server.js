import { renderToString } from '@vue/server-renderer';
import { createApp } from './main.js';

export async function render(url = '/') {
  const app = createApp(url);
  return renderToString(app);
}
