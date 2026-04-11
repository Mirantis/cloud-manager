<template>
  <div class="api-tokens">
    <div class="card">
      <h2>API Tokens</h2>
      <p class="hint">
        Personal access tokens for API authentication. Use
        <code>Authorization: Bearer &lt;token&gt;</code> on requests. The secret is shown only once when you create a token.
      </p>

      <div v-if="error" class="error">{{ error }}</div>

      <h3>Create token</h3>
      <form class="create-form" @submit.prevent="createToken">
        <input
          v-model="newToken.description"
          type="text"
          placeholder="Description (optional)"
          autocomplete="off"
        />
        <label class="expiry-label">
          <span>Expires (optional)</span>
          <input v-model="newToken.expiresLocal" type="datetime-local" />
        </label>
        <button type="submit" class="btn" :disabled="creating">
          {{ creating ? 'Creating…' : 'Generate token' }}
        </button>
      </form>

      <div v-if="loadingList" class="loading">Loading tokens…</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Description</th>
            <th>Created</th>
            <th>Expires</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="tokens.length === 0">
            <td colspan="5" class="empty-row">No API tokens yet.</td>
          </tr>
          <tr v-for="t in tokens" :key="t.id">
            <td>{{ t.id }}</td>
            <td>{{ t.description || '—' }}</td>
            <td>{{ formatDate(t.created_at) }}</td>
            <td>{{ t.expires_at ? formatDate(t.expires_at) : '—' }}</td>
            <td class="actions">
              <button
                type="button"
                class="btn btn-sm btn-danger"
                :disabled="revokingId === t.id"
                @click="revokeToken(t)"
              >
                {{ revokingId === t.id ? '…' : 'Revoke' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="revealedToken" class="modal-overlay" @click.self="closeReveal">
      <div class="modal card reveal-modal" @click.stop>
        <h3>Token created</h3>
        <p class="warn">Copy this token now. You will not be able to see it again.</p>
        <div class="token-box">
          <code class="token-value">{{ revealedToken }}</code>
          <button type="button" class="btn btn-sm" @click="copyToken">Copy</button>
        </div>
        <div class="modal-actions">
          <button type="button" class="btn btn-secondary" @click="closeReveal">Done</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'APITokens',
  data() {
    return {
      tokens: [],
      loadingList: true,
      creating: false,
      error: '',
      newToken: { description: '', expiresLocal: '' },
      revealedToken: '',
      revokingId: null
    }
  },
  mounted() {
    this.loadTokens()
  },
  methods: {
    formatDate(s) {
      if (!s) return '—'
      const d = new Date(s)
      return isNaN(d.getTime()) ? String(s) : d.toLocaleString()
    },
    expiresAtRFC3339() {
      const v = this.newToken.expiresLocal
      if (!v || !String(v).trim()) return ''
      const d = new Date(v)
      if (isNaN(d.getTime())) return ''
      return d.toISOString()
    },
    async loadTokens() {
      this.loadingList = true
      this.error = ''
      try {
        const r = await fetch('/api/auth/tokens', { credentials: 'include' })
        const d = await r.json().catch(() => ({}))
        if (!r.ok) throw new Error(d.message || d.error || r.statusText)
        this.tokens = Array.isArray(d) ? d : []
      } catch (e) {
        this.error = e.message || 'Failed to load tokens'
        this.tokens = []
      } finally {
        this.loadingList = false
      }
    },
    async createToken() {
      this.creating = true
      this.error = ''
      try {
        const body = {}
        if (this.newToken.description && this.newToken.description.trim()) {
          body.description = this.newToken.description.trim()
        }
        const exp = this.expiresAtRFC3339()
        if (exp) body.expires_at = exp

        const r = await fetch('/api/auth/tokens', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body)
        })
        const d = await r.json().catch(() => ({}))
        if (!r.ok) throw new Error(d.error || d.message || r.statusText)
        if (d.token) {
          this.revealedToken = d.token
        }
        this.newToken = { description: '', expiresLocal: '' }
        await this.loadTokens()
      } catch (e) {
        this.error = e.message || 'Failed to create token'
      } finally {
        this.creating = false
      }
    },
    closeReveal() {
      this.revealedToken = ''
    },
    async copyToken() {
      try {
        await navigator.clipboard.writeText(this.revealedToken)
      } catch (_) {
        /* ignore */
      }
    },
    async revokeToken(t) {
      if (!confirm(`Revoke token #${t.id}? This cannot be undone.`)) return
      this.error = ''
      this.revokingId = t.id
      try {
        const r = await fetch(`/api/auth/tokens/${encodeURIComponent(String(t.id))}`, {
          method: 'DELETE',
          credentials: 'include'
        })
        const d = await r.json().catch(() => ({}))
        if (!r.ok) throw new Error(d.error || d.message || r.statusText)
        await this.loadTokens()
      } catch (e) {
        this.error = e.message || 'Failed to revoke token'
      } finally {
        this.revokingId = null
      }
    }
  }
}
</script>

<style scoped>
.api-tokens {
  max-width: 960px;
}
.hint {
  color: var(--color-text-secondary);
  margin-bottom: 1.5rem;
  font-size: 0.9rem;
  line-height: 1.5;
}
.hint code {
  font-size: 0.85em;
  padding: 0.1rem 0.35rem;
  border-radius: var(--radius-sm);
  background: var(--color-bg-tertiary);
}
.create-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: flex-end;
  margin-bottom: 2rem;
}
.create-form input[type='text'] {
  flex: 1 1 200px;
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
}
.expiry-label {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-size: 0.8rem;
  color: var(--color-text-secondary);
}
.expiry-label input {
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
}
.actions {
  white-space: nowrap;
}
.empty-row {
  text-align: center;
  color: var(--color-text-secondary);
  padding: 1.5rem !important;
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
.reveal-modal {
  min-width: min(100%, 480px);
  max-width: 560px;
}
.warn {
  color: var(--color-warning);
  margin: 0.75rem 0 1rem;
  font-size: 0.9rem;
}
.token-box {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  margin: 1rem 0;
  padding: 0.75rem;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}
.token-value {
  flex: 1 1 200px;
  word-break: break-all;
  font-size: 0.85rem;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}
</style>
