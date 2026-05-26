import { createSSRApp } from 'vue';
import App from './App.vue';
import './styles.css';

export function createApp() {
  return createSSRApp(App);
}
