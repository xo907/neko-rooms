import Component from 'vue-class-component'

// allow router hooks as class component methods
Component.registerHooks([
  'beforeRouteEnter',
  'beforeRouteLeave',
  'beforeRouteUpdate',
])
