import { defineConfig, loadEnv } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  return {
    plugins: [vue()],
    server: {
      port: 5174,
      proxy: {
        '/api': env.VITE_API_TARGET || 'http://127.0.0.1:8080',
      },
    },
  };
});
