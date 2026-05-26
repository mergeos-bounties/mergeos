import { renderToString } from '@vue/server-renderer';
import { createApp } from './main.js';

export async function render() {
  return renderToString(createApp());
}
