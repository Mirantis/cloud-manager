import { reactive } from 'vue'

/** Shared auth state for role-based UI (synced from App.vue checkAuth). */
export const authStore = reactive({
  authenticated: false,
  username: '',
  role: '' // admin | editor | viewer
})
