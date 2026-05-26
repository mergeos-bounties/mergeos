<template>
  <section class="api-docs-shell">
    <div class="api-docs-head">
      <div>
        <p>MergeOS Scan API</p>
        <h1>Scan API Reference</h1>
      </div>
      <div class="api-docs-actions">
        <a class="compact-button" href="/openapi.json" target="_blank" rel="noreferrer">
          <FileJson :size="16" />
          OpenAPI JSON
        </a>
        <a class="compact-button" href="/api/health" target="_blank" rel="noreferrer">
          <ExternalLink :size="16" />
          Health
        </a>
      </div>
    </div>

    <section v-if="loadError" class="notice error">
      <AlertTriangle :size="18" />
      <span>{{ loadError }}</span>
    </section>

    <div ref="swaggerRoot" class="swagger-frame" aria-label="MergeOS Scan API documentation" />
  </section>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue';
import { AlertTriangle, ExternalLink, FileJson } from '@lucide/vue';
import 'swagger-ui-dist/swagger-ui.css';

const swaggerRoot = ref(null);
const loadError = ref('');
let swaggerUI = null;
let disposed = false;

onMounted(async () => {
  try {
    const module = await import('swagger-ui-dist/swagger-ui-es-bundle.js');
    const SwaggerUIBundle = module.default || module.SwaggerUIBundle || module.default?.SwaggerUIBundle;
    if (!SwaggerUIBundle || disposed || !swaggerRoot.value) return;
    swaggerUI = SwaggerUIBundle({
      url: '/openapi.json',
      domNode: swaggerRoot.value,
      deepLinking: true,
      displayRequestDuration: true,
      docExpansion: 'list',
      filter: true,
      persistAuthorization: true,
      tryItOutEnabled: true,
    });
  } catch (error) {
    loadError.value = error?.message || 'Could not load Swagger UI.';
  }
});

onBeforeUnmount(() => {
  disposed = true;
  if (swaggerUI?.destroy) swaggerUI.destroy();
});
</script>
