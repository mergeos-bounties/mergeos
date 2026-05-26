import { createApp as createClientApp } from 'vue';
import { createApp as createHydratedApp } from './main.js';
import App from './App.vue';

const root = document.getElementById('app');
const hasSSRMarkup = Boolean(root?.firstElementChild);
const app = hasSSRMarkup ? createHydratedApp() : createClientApp(App);

app.mount('#app');
