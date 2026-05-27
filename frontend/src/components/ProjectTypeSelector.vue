<template>
  <div class="project-type-container">
    <div 
      class="project-type-card new-project"
      :class="{ selected: selectedType === 'new' }"
      @click="selectType('new')"
    >
      <font-awesome-icon icon="code" class="project-type-icon" />
      <h3 class="project-type-title">New Project</h3>
      <p class="project-type-description">Start a brand new project from scratch</p>
      <p class="project-type-example">Example: "Build a React dashboard for analytics"</p>
    </div>
    
    <div 
      class="project-type-card bug-fix"
      :class="{ selected: selectedType === 'bug' }"
      @click="selectType('bug')"
    >
      <font-awesome-icon icon="bug" class="project-type-icon" />
      <h3 class="project-type-title">Fix Bug in Existing Project</h3>
      <p class="project-type-description">Improve or fix issues in an existing codebase</p>
      <p class="project-type-example">Example: "Fix login page redirect issue"</p>
    </div>
  </div>
</template>

<script>
export default {
  props: {
    modelValue: {
      type: String,
      default: ''
    }
  },
  emits: ['update:modelValue'],
  computed: {
    selectedType() {
      return this.modelValue
    }
  },
  methods: {
    selectType(type) {
      this.$emit('update:modelValue', type)
    }
  }
}
</script>