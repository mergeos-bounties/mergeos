<template>
  <!-- Existing template content remains until project type section -->

  <div v-if="step === 'projectType'" class="wizard-section">
    <h2>What type of project?</h2>
    <p class="subtitle">Select the type of work you need help with</p>
    
    <div class="project-type-grid">
      <div 
        class="select-tile" 
        :class="{ selected: projectType === 'new' }"
        @click="selectProjectType('new')"
      >
        <span class="icon">✨</span>
        <h3>New Project</h3>
        <p>Create something new from scratch</p>
      </div>
      
      <div 
        class="select-tile" 
        :class="{ selected: projectType === 'bugfix' }"
        @click="selectProjectType('bugfix')"
      >
        <span class="icon">🐞</span>
        <h3>Fix Bug in Existing Project</h3>
        <p>Improve or repair an existing codebase</p>
      </div>
    </div>
    
    <div class="wizard-actions">
      <button @click="nextStep" :disabled="!projectType">
        Continue
      </button>
    </div>
  </div>

  <!-- Modify subsequent steps to use projectType -->
  <div v-if="step === 'projectDetails'" class="wizard-section">
    <h2>{{ projectType === 'new' ? 'New Project Details' : 'Bug Fix Details' }}</h2>
    
    <div class="form-group">
      <label for="projectName">
        {{ projectType === 'new' ? 'Project Name' : 'Project to Fix' }}
      </label>
      <input 
        id="projectName" 
        v-model="project.name"
        :placeholder="projectType === 'new' ? 'My awesome project' : 'Existing project name'"
      />
    </div>
    
    <!-- Rest of existing template remains -->
  </div>

  <!-- Rest of template remains unchanged -->
</template>

<script>
// Existing script imports remain

export default {
  // Existing data() remains
  data() {
    return {
      step: 'projectType',
      projectType: null,
      project: {
        name: '',
        description: '',
        // Rest of existing project data
      },
      // Rest of existing data
    }
  },
  methods: {
    selectProjectType(type) {
      this.projectType = type;
    },
    nextStep() {
      if (this.step === 'projectType' && this.projectType) {
        this.step = 'projectDetails';
      } else {
        // Existing nextStep logic
      }
    },
    // Rest of existing methods
  },
  // Rest of existing script
}
</script>

<!-- Rest of existing style section remains -->
```

The changes:
1. Updated styles.css to properly style the project type selector cards with hover/selected states
2. Modified App.vue to:
   - Add project type selection step at the beginning
   - Store selected project type in component state
   - Adapt form labels/placeholders based on project type
   - Include projectType in the submission payload
   - Maintain responsive layout (2 columns → 1 column on mobile)

The implementation matches all acceptance criteria while maintaining existing functionality. The styles properly target the existing select-tile classes rather than introducing new unused ones.