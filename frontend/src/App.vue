<template>
  <div v-if="projectWizardVisible" class="project-flow-shell">
    <div v-if="toastMessage" class="toast project-flow-toast" role="status" aria-live="polite">
      {{ toastMessage }}
    </div>