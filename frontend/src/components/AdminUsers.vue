<template>
  <div class="admin-users">
    <div class="card">
      <h2>Application users</h2>
      <p class="hint">Manage login accounts (not AWS IAM users). Only administrators see this page.</p>

      <div v-if="error" class="error">{{ error }}</div>

      <h3>Add user</h3>
      <form class="add-form" @submit.prevent="createUser">
        <input v-model="newUser.username" type="text" placeholder="Username" required autocomplete="off" />
        <input v-model="newUser.password" type="password" placeholder="Password" required autocomplete="new-password" />
        <select v-model="newUser.role" required>
          <option value="viewer">viewer</option>
          <option value="editor">editor</option>
          <option value="admin">admin</option>
        </select>
        <button type="submit" class="btn" :disabled="loading">{{ loading ? 'Creating…' : 'Create' }}</button>
      </form>

      <div v-if="loadingList" class="loading">Loading users…</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>Username</th>
            <th>Role</th>
            <th>Created</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.username">
            <td>{{ u.username }}</td>
            <td>{{ u.role }}</td>
            <td>{{ formatDate(u.created_at) }}</td>
            <td class="actions">
              <button type="button" class="btn btn-sm btn-secondary" @click="openReset(u)">Reset password</button>
              <button
                type="button"
                class="btn btn-sm btn-danger"
                :disabled="u.username === currentUsername"
                @click="removeUser(u)"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="resetUser" class="modal-overlay" @click.self="resetUser = null">
      <div class="modal card">
        <h3>Reset password: {{ resetUser.username }}</h3>
        <input v-model="resetPassword" type="password" placeholder="New password" autocomplete="new-password" />
        <div class="modal-actions">
          <button type="button" class="btn btn-secondary" @click="resetUser = null">Cancel</button>
          <button type="button" class="btn" :disabled="!resetPassword || resetLoading" @click="submitReset">
            {{ resetLoading ? 'Saving…' : 'Save' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'AdminUsers',
  data() {
    return {
      users: [],
      loading: false,
      loadingList: true,
      error: '',
      newUser: { username: '', password: '', role: 'viewer' },
      currentUsername: '',
      resetUser: null,
      resetPassword: '',
      resetLoading: false
    }
  },
  mounted() {
    this.loadMe()
    this.loadUsers()
  },
  methods: {
    formatDate(s) {
      if (!s) return '—'
      const d = new Date(s)
      return isNaN(d.getTime()) ? String(s) : d.toLocaleString()
    },
    async loadMe() {
      try {
        const r = await fetch('/api/auth/user', { credentials: 'include' })
        if (r.ok) {
          const d = await r.json()
          this.currentUsername = d.username || ''
        }
      } catch (_) {}
    },
    async loadUsers() {
      this.loadingList = true
      this.error = ''
      try {
        const r = await fetch('/api/admin/users', { credentials: 'include' })
        if (!r.ok) {
          const d = await r.json().catch(() => ({}))
          throw new Error(d.message || d.error || r.statusText)
        }
        this.users = await r.json()
      } catch (e) {
        this.error = e.message || 'Failed to load users'
      } finally {
        this.loadingList = false
      }
    },
    async createUser() {
      this.loading = true
      this.error = ''
      try {
        const r = await fetch('/api/admin/users', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(this.newUser)
        })
        const d = await r.json().catch(() => ({}))
        if (!r.ok) throw new Error(d.error || d.message || r.statusText)
        this.newUser = { username: '', password: '', role: 'viewer' }
        await this.loadUsers()
      } catch (e) {
        this.error = e.message || 'Create failed'
      } finally {
        this.loading = false
      }
    },
    async removeUser(u) {
      if (!confirm(`Delete user "${u.username}"?`)) return
      this.error = ''
      try {
        const r = await fetch(`/api/admin/users/${encodeURIComponent(u.username)}`, {
          method: 'DELETE',
          credentials: 'include'
        })
        const d = await r.json().catch(() => ({}))
        if (!r.ok) throw new Error(d.error || d.message || r.statusText)
        await this.loadUsers()
      } catch (e) {
        this.error = e.message || 'Delete failed'
      }
    },
    openReset(u) {
      this.resetUser = u
      this.resetPassword = ''
    },
    async submitReset() {
      if (!this.resetUser || !this.resetPassword) return
      this.resetLoading = true
      this.error = ''
      try {
        const r = await fetch(`/api/admin/users/${encodeURIComponent(this.resetUser.username)}/password`, {
          method: 'PUT',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ password: this.resetPassword })
        })
        const d = await r.json().catch(() => ({}))
        if (!r.ok) throw new Error(d.error || d.message || r.statusText)
        this.resetUser = null
        this.resetPassword = ''
      } catch (e) {
        this.error = e.message || 'Reset failed'
      } finally {
        this.resetLoading = false
      }
    }
  }
}
</script>

<style scoped>
.admin-users {
  max-width: 960px;
}
.hint {
  color: var(--color-text-secondary);
  margin-bottom: 1.5rem;
  font-size: 0.9rem;
}
.add-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  margin-bottom: 2rem;
}
.add-form input,
.add-form select {
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
.btn-sm {
  padding: 0.35rem 0.65rem;
  font-size: 0.8rem;
}
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  padding: 1rem;
}
.modal {
  min-width: 280px;
  max-width: 400px;
}
.modal input {
  width: 100%;
  margin: 1rem 0;
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
}
.modal-actions {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
}
</style>
