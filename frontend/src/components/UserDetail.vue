<template>
  <div>
    <div class="card">
      <nav>
        <router-link to="/aws/users">← All Users</router-link> / 
        <span v-if="account">{{ account.name }}</span><span v-else>{{ accountId }}</span> / 
        {{ username }}
      </nav>
      <h2>
        User Details: {{ username }}
        <span v-if="account" class="account-subtitle">in {{ account.name }}</span>
      </h2>
      <div class="header-actions">
        <button @click="refreshUser" class="btn btn-secondary">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
          </svg>
          Refresh
        </button>
        <button v-if="canModify" @click="deleteUser" class="btn btn-danger" :disabled="deletingUser">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
          </svg>
          {{ deletingUser ? 'Deleting...' : 'Delete User' }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading">
      <span>Loading user details...</span>
    </div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="user" class="content-grid">
      <!-- User Information Card -->
      <div class="info-card">
        <div class="card-header">
          <div class="card-header-content">
            <div class="card-icon">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2C13.1 2 14 2.9 14 4C14 5.1 13.1 6 12 6C10.9 6 10 5.1 10 4C10 2.9 10.9 2 12 2Z"/>
              </svg>
            </div>
            <h3>User Information</h3>
          </div>
          <div class="user-status">
            <div class="password-status-container">
              <span :class="['status-badge', user?.password_set ? 'status-success' : 'status-warning']">
                <svg class="status-icon" viewBox="0 0 24 24" fill="currentColor">
                  <path v-if="user?.password_set" d="M21,7L9,19L3.5,13.5L4.91,12.09L9,16.17L19.59,5.59L21,7Z"/>
                  <path v-else d="M12,17A2,2 0 0,0 14,15C14,13.89 13.1,13 12,13A2,2 0 0,0 10,15A2,2 0 0,0 12,17M18,8A2,2 0 0,1 20,10V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V10C4,8.89 4.9,8 6,8H7V6A5,5 0 0,1 12,1A5,5 0 0,1 17,6V8H18M12,3A3,3 0 0,0 9,6V8H15V6A3,3 0 0,0 12,3Z"/>
                </svg>
                {{ user?.password_set ? 'Password Set' : 'No Password' }}
              </span>
              <div class="password-actions" v-if="canModify">
                <button
                  @click="rotateUserPassword"
                  class="btn btn-primary btn-xs"
                  :disabled="rotatingPassword"
                  :title="user?.password_set ? 'Rotate console password' : 'Create console password'"
                >
                  <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
                  </svg>
                  {{ rotatingPassword ? (user?.password_set ? 'Rotating...' : 'Creating...') : (user?.password_set ? 'Rotate' : 'Create') }}
                </button>
                <button
                  v-if="user?.password_set"
                  @click="removeUserPassword"
                  class="btn btn-warning btn-xs"
                  :disabled="removingPassword"
                  title="Remove console password"
                >
                  <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                  </svg>
                  {{ removingPassword ? 'Removing...' : 'Remove' }}
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="card-body">
          <div class="info-grid">
            <div class="info-item">
              <div class="info-label">Username</div>
              <div class="info-value user-name">{{ user?.username || 'N/A' }}</div>
            </div>
            <div class="info-item">
              <div class="info-label">Account</div>
              <div class="info-value">
                <div class="account-info">
                  <div class="account-name">{{ account?.name || 'Loading...' }}</div>
                  <code class="account-id">{{ account?.id || accountId }}</code>
                </div>
              </div>
            </div>
            <div class="info-item">
              <div class="info-label">User ID</div>
              <div class="info-value">
                <code class="user-id">{{ user?.user_id || 'N/A' }}</code>
              </div>
            </div>
            <div class="info-item full-width">
              <div class="info-label">ARN</div>
              <div class="info-value">
                <code class="arn">{{ user?.arn || 'N/A' }}</code>
              </div>
            </div>
            <div class="info-item">
              <div class="info-label">Created</div>
              <div class="info-value">
                <div class="date-display" v-if="user?.create_date">
                  <span class="date-value">{{ formatDate(user.create_date) }}</span>
                  <span class="date-relative">{{ formatRelativeDate(user.create_date) }}</span>
                </div>
              </div>
            </div>
            <div class="info-item">
              <div class="info-label">Access Keys</div>
              <div class="info-value">
                <div class="key-count">
                  <span class="count-number">{{ user.access_keys?.length || 0 }}</span>
                  <span class="count-label">{{ (user.access_keys?.length || 0) === 1 ? 'key' : 'keys' }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Access Keys Card -->
      <div class="keys-card">
        <div class="card-header">
          <div class="card-header-content">
            <div class="card-icon">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M7 14C5.9 14 5 13.1 5 12S5.9 10 7 10 9 10.9 9 12 8.1 14 7 14M12.6 10C11.8 7.7 9.6 6 7 6C3.7 6 1 8.7 1 12S3.7 18 7 18C9.6 18 11.8 16.3 12.6 14H16L17.5 15.5L19 14L17.5 12.5L19 11L17.5 9.5L16 11H12.6Z"/>
              </svg>
            </div>
            <h3>Access Keys</h3>
          </div>
          <div class="card-actions">
            <button @click="downloadAccessKeysJSON" class="btn btn-success btn-sm" :disabled="!user || (user.access_keys?.length || 0) === 0">
              <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                <path d="M14,2H6A2,2 0 0,0 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2M18,20H6V4H13V9H18V20Z"/>
              </svg>
              Export JSON
            </button>
            <button v-if="canModify" @click="createAccessKey" class="btn btn-primary" :disabled="creatingKey">
              <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                <path d="M19,13H13V19H11V13H5V11H11V5H13V11H19V13Z"/>
              </svg>
              {{ creatingKey ? 'Creating...' : 'Create New Key' }}
            </button>
          </div>
        </div>
        <div class="card-body">
          <div v-if="(user.access_keys?.length || 0) === 0" class="empty-keys">
            <div class="empty-icon">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M7 14C5.9 14 5 13.1 5 12S5.9 10 7 10 9 10.9 9 12 8.1 14 7 14M12.6 10C11.8 7.7 9.6 6 7 6C3.7 6 1 8.7 1 12S3.7 18 7 18C9.6 18 11.8 16.3 12.6 14H16L17.5 15.5L19 14L17.5 12.5L19 11L17.5 9.5L16 11H12.6Z"/>
              </svg>
            </div>
            <h4>No Access Keys</h4>
            <p>This user doesn't have any access keys yet. Create one to enable programmatic access.</p>
          </div>
          <div v-else>
            <div class="keys-controls">
              <div class="search-section">
                <div class="search-box">
                  <svg class="search-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M9.5,3A6.5,6.5 0 0,1 16,9.5C16,11.11 15.41,12.59 14.44,13.73L14.71,14H15.5L20.5,19L19,20.5L14,15.5V14.71L13.73,14.44C12.59,15.41 11.11,16 9.5,16A6.5,6.5 0 0,1 3,9.5A6.5,6.5 0 0,1 9.5,3M9.5,5C7,5 5,7 5,9.5C5,12 7,14 9.5,14C12,14 14,12 14,9.5C14,7 12,5 9.5,5Z"/>
                  </svg>
                  <input
                    v-model="keySearchQuery"
                    type="text"
                    placeholder="Search access keys..."
                    class="search-input"
                  />
                </div>
              </div>
            </div>
            <div class="keys-list">
              <div v-for="key in filteredAccessKeys" :key="key.access_key_id" class="key-item">
              <div class="key-header">
                <div class="key-info">
                  <div class="key-id">
                    <svg class="key-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M7 14C5.9 14 5 13.1 5 12S5.9 10 7 10 9 10.9 9 12 8.1 14 7 14M12.6 10C11.8 7.7 9.6 6 7 6C3.7 6 1 8.7 1 12S3.7 18 7 18C9.6 18 11.8 16.3 12.6 14H16L17.5 15.5L19 14L17.5 12.5L19 11L17.5 9.5L16 11H12.6Z"/>
                    </svg>
                    <code>{{ key.access_key_id }}</code>
                  </div>
                  <div class="key-meta">
                    <span :class="['key-status', key.status === 'Active' ? 'status-active' : 'status-inactive']">
                      <svg class="status-dot" viewBox="0 0 24 24" fill="currentColor">
                        <circle cx="12" cy="12" r="4"/>
                      </svg>
                      {{ key.status === 'Active' ? 'Active ' : key.status }}
                    </span>
                    <div class="key-times">
                      <div class="key-date">
                        <span class="date-label">Created:</span>
                        <div class="date-display">
                          <span class="date-value">{{ formatDate(key.create_date) }}</span>
                          <span class="date-relative">{{ formatRelativeDate(key.create_date) }}</span>
                        </div>
                      </div>
                      <div v-if="key.last_used_date" class="key-last-used">
                        <span class="date-label">Last used:</span>
                        <div class="date-display">
                          <span class="date-value">{{ formatDate(key.last_used_date) }}</span>
                          <span class="date-relative">{{ formatRelativeDate(key.last_used_date) }}</span>
                        </div>
                        <div v-if="key.last_used_service" class="last-used-service">
                          {{ key.last_used_service }}{{ key.last_used_region ? ' in ' + key.last_used_region : '' }}
                        </div>
                      </div>
                      <div v-else-if="key.status === 'Active'" class="key-never-used">
                        <span class="date-label">Last used:</span>
                        <span class="never-used-text">Never used</span>
                      </div>
                    </div>
                  </div>
                </div>
                <div class="key-actions" v-if="canModify">
                  <button @click="rotateKey(key.access_key_id)" class="btn btn-warning btn-sm">
                    <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
                    </svg>
                    Rotate
                  </button>
                  <button @click="deleteKey(key.access_key_id)" class="btn btn-danger btn-sm">
                    <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                    </svg>
                    Delete
                  </button>
                </div>
              </div>
            </div>
          </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal for displaying new key -->
    <div v-if="newKey" class="modal-overlay" @click="closeModal">
      <div class="modern-modal" @click.stop>
        <div class="modal-header">
          <div class="modal-icon success">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M21,7L9,19L3.5,13.5L4.91,12.09L9,16.17L19.59,5.59L21,7Z"/>
            </svg>
          </div>
          <h3>New Access Key Created</h3>
          <button @click="closeModal" class="modal-close">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"/>
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <div class="key-display">
            <div class="key-field">
              <label>Access Key ID</label>
              <div class="key-value">
                <code>{{ newKey.access_key_id }}</code>
                <button @click="copyToClipboard(newKey.access_key_id)" class="copy-btn" title="Copy to clipboard">
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M19,21H8V7H19M19,5H8A2,2 0 0,0 6,7V21A2,2 0 0,0 8,23H19A2,2 0 0,0 21,21V7A2,2 0 0,0 19,5M16,1H4A2,2 0 0,0 2,3V17H4V3H16V1Z"/>
                  </svg>
                </button>
              </div>
            </div>
            <div class="key-field">
              <label>Secret Access Key</label>
              <div class="key-value">
                <code>{{ newKey.secret_access_key }}</code>
                <button @click="copyToClipboard(newKey.secret_access_key)" class="copy-btn" title="Copy to clipboard">
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M19,21H8V7H19M19,5H8A2,2 0 0,0 6,7V21A2,2 0 0,0 8,23H19A2,2 0 0,0 21,21V7A2,2 0 0,0 19,5M16,1H4A2,2 0 0,0 2,3V17H4V3H16V1Z"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>
          <div class="warning-message">
            <svg class="warning-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
            </svg>
            <div class="warning-text">
              <strong>Important:</strong> Save these credentials now. The secret access key cannot be retrieved again once this dialog is closed.
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeModal" class="btn btn-primary">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M21,7L9,19L3.5,13.5L4.91,12.09L9,16.17L19.59,5.59L21,7Z"/>
            </svg>
            I've Saved the Credentials
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'UserDetail',
  props: ['accountId', 'username'],
  data() {
    return {
      user: null,
      account: null,
      loading: true,
      error: null,
      creatingKey: false,
      newKey: null,
      deletingUser: false,
      removingPassword: false,
      rotatingPassword: false,
      keySearchQuery: ''
    }
  },
  computed: {
    filteredAccessKeys() {
      if (!this.user || !this.user.access_keys) return []

      let keys = [...this.user.access_keys]

      // Apply search filter
      if (this.keySearchQuery) {
        const query = this.keySearchQuery.toLowerCase()
        keys = keys.filter(key =>
          key.access_key_id.toLowerCase().includes(query) ||
          key.status.toLowerCase().includes(query) ||
          (key.last_used_service && key.last_used_service.toLowerCase().includes(query)) ||
          (key.last_used_region && key.last_used_region.toLowerCase().includes(query))
        )
      }

      return keys
    }
  },
  async mounted() {
    await this.loadUser()
  },
  methods: {
    async loadUser() {
      try {
        this.loading = true
        this.error = null
        
        // Load user data and accounts in parallel
        const [userResponse, accountsResponse] = await Promise.all([
          axios.get(`/api/accounts/${this.accountId}/users/${this.username}`),
          axios.get('/api/accounts')
        ])
        
        this.user = userResponse.data
        
        // Find the account information
        const accounts = accountsResponse.data
        this.account = accounts.find(acc => acc.id === this.accountId) || {
          id: this.accountId,
          name: 'Unknown Account',
          accessible: false
        }
        
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load user'
      } finally {
        this.loading = false
      }
    },
    async refreshUser() {
      if (this.canModify) {
        try {
          await axios.post(`/api/cache/accounts/${this.accountId}/users/${this.username}/invalidate`)
        } catch (error) {
          console.warn('Failed to invalidate user cache:', error)
        }
      }
      
      await this.loadUser()
    },
    async createAccessKey() {
      this.creatingKey = true
      try {
        const response = await axios.post(`/api/accounts/${this.accountId}/users/${this.username}/keys`)
        this.newKey = response.data
        
        // Explicitly invalidate cache to ensure fresh data on navigation back to AllUsers
        try {
          await axios.post(`/api/cache/accounts/${this.accountId}/invalidate`)
        } catch (cacheErr) {
          console.warn('Failed to invalidate account cache:', cacheErr)
        }
        
        await this.loadUser() // Refresh user data
      } catch (err) {
        alert(err.response?.data?.error || 'Failed to create access key')
      } finally {
        this.creatingKey = false
      }
    },
    async deleteKey(keyId) {
      if (!confirm('Are you sure you want to delete this access key?')) {
        return
      }
      try {
        await axios.delete(`/api/accounts/${this.accountId}/users/${this.username}/keys/${keyId}`)
        
        // Explicitly invalidate cache to ensure fresh data on navigation back to AllUsers
        try {
          await axios.post(`/api/cache/accounts/${this.accountId}/invalidate`)
        } catch (cacheErr) {
          console.warn('Failed to invalidate account cache:', cacheErr)
        }
        
        await this.loadUser() // Refresh user data
        alert('Access key deleted successfully')
      } catch (err) {
        alert(err.response?.data?.error || 'Failed to delete access key')
      }
    },
    async rotateKey(keyId) {
      if (!confirm('Are you sure you want to rotate this access key? The old key will be deleted.')) {
        return
      }
      try {
        const response = await axios.put(`/api/accounts/${this.accountId}/users/${this.username}/keys/${keyId}/rotate`)
        this.newKey = response.data
        
        // Explicitly invalidate cache to ensure fresh data on navigation back to AllUsers
        try {
          await axios.post(`/api/cache/accounts/${this.accountId}/invalidate`)
        } catch (cacheErr) {
          console.warn('Failed to invalidate account cache:', cacheErr)
        }
        
        await this.loadUser() // Refresh user data
      } catch (err) {
        alert(err.response?.data?.error || 'Failed to rotate access key')
      }
    },
    closeModal() {
      this.newKey = null
    },
    formatDate(dateString) {
      return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
      })
    },
    formatRelativeDate(dateString) {
      const now = new Date()
      const date = new Date(dateString)
      const diffTime = Math.abs(now - date)
      const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24))
      
      if (diffDays === 0) return 'today'
      if (diffDays === 1) return 'yesterday'
      if (diffDays < 30) return `${diffDays} days ago`
      if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`
      return `${Math.floor(diffDays / 365)} years ago`
    },
    copyToClipboard(text) {
      if (navigator.clipboard) {
        navigator.clipboard.writeText(text).then(() => {
          // Could add a toast notification here
          // Successfully copied to clipboard
        })
      } else {
        // Fallback for older browsers
        const textArea = document.createElement('textarea')
        textArea.value = text
        document.body.appendChild(textArea)
        textArea.select()
        document.execCommand('copy')
        document.body.removeChild(textArea)
      }
    },

    downloadAccessKeysJSON() {
      try {
        if (!this.user || !this.user.access_keys || this.user.access_keys.length === 0) {
          alert('No access keys to export')
          return
        }

        const exportData = {
          exported_at: new Date().toISOString(),
          user: {
            username: this.user.username,
            user_id: this.user.user_id,
            arn: this.user.arn,
            create_date: this.user.create_date,
            password_set: this.user.password_set
          },
          account: {
            id: this.accountId,
            name: this.user.arn ? this.user.arn.split(':')[4] : this.accountId
          },
          access_keys: this.user.access_keys.map(key => ({
            access_key_id: key.access_key_id,
            status: key.status,
            create_date: key.create_date,
            last_used_date: key.last_used_date,
            last_used_service: key.last_used_service,
            last_used_region: key.last_used_region
          })),
          total_keys: this.user.access_keys.length
        }
        
        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr)
        
        const exportFileDefaultName = `aws-access-keys-${this.user.username}-${new Date().toISOString().split('T')[0]}.json`
        
        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (error) {
        console.error('Failed to download JSON:', error)
        alert('Failed to download JSON file')
      }
    },

    async deleteUser() {
      const confirmMessage = `Are you sure you want to DELETE the user "${this.username}"?\n\nThis will:\n1. Delete all access keys for this user\n2. Delete the user's login profile (if exists)\n3. Permanently delete the user\n\nThis action cannot be undone!`
      
      if (!confirm(confirmMessage)) return
      
      this.deletingUser = true
      
      try {
        await axios.delete(`/api/accounts/${this.accountId}/users/${this.username}`)
        
        alert(`User "${this.username}" has been successfully deleted.`)
        // Navigate back to AllUsers page with deleted user info so it can update local cache
        this.$router.push({
          path: '/',
          query: { deletedUser: this.username, deletedFromAccount: this.accountId }
        })
      } catch (error) {
        console.error('Failed to delete user:', error)
        alert(error.response?.data?.error || 'Failed to delete user. Please try again.')
      } finally {
        this.deletingUser = false
      }
    },

    async removeUserPassword() {
      const confirmMessage = `Are you sure you want to remove the console password for user "${this.username}"?\n\nThis will:\n1. Delete the user's console login password\n2. Prevent the user from logging into the AWS Console\n3. Not affect programmatic access (access keys)\n\nThis action cannot be undone!`

      if (!confirm(confirmMessage)) return

      this.removingPassword = true

      try {
        await axios.delete(`/api/accounts/${this.accountId}/users/${this.username}/password`)

        // Explicitly invalidate cache to ensure fresh data on navigation back to AllUsers
        try {
          await axios.post(`/api/cache/accounts/${this.accountId}/invalidate`)
        } catch (cacheErr) {
          console.warn('Failed to invalidate account cache:', cacheErr)
        }

        alert(`Console password for user "${this.username}" has been successfully removed.`)
        await this.loadUser() // Refresh user data to show updated password status
      } catch (error) {
        console.error('Failed to remove user password:', error)
        alert(error.response?.data?.error || 'Failed to remove user password. Please try again.')
      } finally {
        this.removingPassword = false
      }
    },

    async rotateUserPassword() {
      const confirmMessage = `Are you sure you want to rotate the console password for user "${this.username}"?\n\nThis will:\n1. Generate a new random console password\n2. ${this.user?.password_set ? 'Replace the existing password' : 'Create a new password'}\n3. Display the new password once (save it immediately)\n\nContinue?`

      if (!confirm(confirmMessage)) return

      this.rotatingPassword = true

      try {
        const response = await axios.post(`/api/accounts/${this.accountId}/users/${this.username}/password/rotate`)

        // Explicitly invalidate cache to ensure fresh data on navigation back to AllUsers
        try {
          await axios.post(`/api/cache/accounts/${this.accountId}/invalidate`)
        } catch (cacheErr) {
          console.warn('Failed to invalidate account cache:', cacheErr)
        }

        // Show the new password in an alert
        const newPassword = response.data.new_password
        alert(`New password for user "${this.username}":\n\n${newPassword}\n\nSave this password now! This is the only time it will be displayed.`)

        await this.loadUser() // Refresh user data to show updated password status
      } catch (error) {
        console.error('Failed to rotate user password:', error)
        alert(error.response?.data?.error || 'Failed to rotate user password. Please try again.')
      } finally {
        this.rotatingPassword = false
      }
    }
  }
}
</script>

<style scoped>
.user-detail-container {
  animation: fadeIn 0.5s ease-in-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}

.header-actions {
  display: flex;
  gap: var(--spacing-sm);
  margin-top: var(--spacing);
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--spacing-xl);
  padding: var(--spacing-lg);
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow);
}

.account-subtitle {
  font-size: 1rem;
  font-weight: 400;
  color: var(--color-text-secondary);
  margin-left: 0.5rem;
}

.header-content {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

/* Breadcrumb */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.breadcrumb-link {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  color: var(--color-btn-primary);
  text-decoration: none;
  transition: color var(--transition-fast);
}

.breadcrumb-link:hover {
  color: var(--color-btn-primary-hover);
}

.breadcrumb-icon {
  width: 1rem;
  height: 1rem;
}

.breadcrumb-separator {
  width: 1rem;
  height: 1rem;
  color: var(--color-text-tertiary);
}

.breadcrumb-current {
  font-weight: 500;
  color: var(--color-text-primary);
}

/* Header Text */
.header-text {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
}

.header-icon {
  width: 3rem;
  height: 3rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-accent);
  border-radius: var(--radius-lg);
  color: var(--color-text-inverse);
}

.header-icon svg {
  width: 1.5rem;
  height: 1.5rem;
}

.header-text h2 {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
}

.header-text p {
  font-size: 0.9rem;
  color: var(--color-text-secondary);
  margin: var(--spacing-xs) 0 0 0;
}

.header-actions {
  display: flex;
  gap: var(--spacing-sm);
}

/* Content Grid */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--spacing-xl);
}

/* Card Styles */
.info-card, .keys-card {
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  border: 1px solid var(--color-border-light);
  overflow: hidden;
  transition: all var(--transition);
}

.info-card:hover, .keys-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px var(--color-shadow);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-lg);
  border-bottom: 1px solid var(--color-border-light);
  background: var(--color-bg-secondary);
}

.card-header-content {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
}

.card-icon {
  width: 2.5rem;
  height: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--color-btn-primary), var(--color-bg-accent));
  border-radius: var(--radius-md);
  color: var(--color-text-inverse);
}

.card-icon svg {
  width: 1.25rem;
  height: 1.25rem;
}

.card-header h3 {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.card-body {
  padding: var(--spacing-lg);
}

.card-actions {
  display: flex;
  gap: var(--spacing-sm);
}

/* User Status Badge */
.password-status-container {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  flex-wrap: wrap;
}

.password-actions {
  display: flex;
  gap: var(--spacing-xs);
}

.user-status .status-badge {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--radius-xl);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.status-badge.status-success {
  background: rgba(40, 167, 69, 0.1);
  color: var(--color-success);
  border: 1px solid rgba(40, 167, 69, 0.2);
}

.status-badge.status-warning {
  background: rgba(255, 193, 7, 0.1);
  color: var(--color-warning);
  border: 1px solid rgba(255, 193, 7, 0.2);
}

.status-icon {
  width: 0.875rem;
  height: 0.875rem;
}

/* Info Grid */
.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--spacing-lg);
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.info-item.full-width {
  grid-column: 1 / -1;
}

.info-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-value {
  color: var(--color-text-primary);
}

.user-name {
  font-size: 1.125rem;
  font-weight: 600;
}

.user-id {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.75rem;
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-light);
  display: inline-block;
}

.account-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.account-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.account-id {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.75rem;
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-light);
  display: inline-block;
  width: fit-content;
}

.arn {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.75rem;
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-light);
  display: block;
  word-break: break-all;
}

.date-display {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.date-value {
  font-size: 0.875rem;
  font-weight: 500;
  line-height: 1;
}

.date-relative {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.key-count {
  display: flex;
  align-items: baseline;
  gap: var(--spacing-xs);
}

.count-number {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--color-btn-primary);
  line-height: 1;
}

.count-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

/* Keys Card */
.empty-keys {
  text-align: center;
  padding: var(--spacing-xxl);
}

.empty-icon {
  width: 4rem;
  height: 4rem;
  margin: 0 auto var(--spacing-md);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-secondary);
  border-radius: 50%;
  color: var(--color-text-secondary);
}

.empty-icon svg {
  width: 2rem;
  height: 2rem;
}

.empty-keys h4 {
  font-size: 1.125rem;
  color: var(--color-text-primary);
  margin-bottom: var(--spacing-sm);
}

.empty-keys p {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}

.keys-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.key-item {
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  transition: all var(--transition);
  margin-bottom: var(--spacing-xs);
}

.key-item:hover {
  border-color: var(--color-btn-primary);
  transform: translateX(4px);
}

.key-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-lg);
  gap: var(--spacing-xl);
  min-height: 4rem;
}

.key-info {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
  flex: 1;
  min-width: 0;
}

.key-id {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-weight: 500;
}

.key-icon {
  width: 1rem;
  height: 1rem;
  color: var(--color-btn-primary);
}

.key-id code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875rem;
  color: var(--color-text-primary);
}

.key-meta {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
}

.key-status {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  font-size: 0.75rem;
  font-weight: 500;
}

.key-status.status-active {
  color: var(--color-success);
}

.key-status.status-inactive {
  color: var(--color-danger);
}

.status-dot {
  width: 0.5rem;
  height: 0.5rem;
}

.key-date {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.key-times {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.key-date,
.key-last-used {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.date-label {
  font-size: 0.7rem;
  color: var(--color-text-tertiary);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.date-display {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.date-value {
  font-size: 0.75rem;
  color: var(--color-text-primary);
  font-weight: 500;
}

.date-relative {
  font-size: 0.7rem;
  color: var(--color-text-secondary);
}

.key-never-used {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.never-used-text {
  font-size: 0.75rem;
  color: var(--color-warning);
  font-style: italic;
  font-weight: 500;
}

.last-used-service {
  color: var(--color-text-tertiary);
  font-size: 0.7rem;
  margin-top: 0.25rem;
}

.keys-controls {
  margin-bottom: 1rem;
  padding: 1rem;
  background: var(--color-bg-secondary);
  border-radius: 6px;
  border: 1px solid var(--color-border);
}

.search-section {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  max-width: 400px;
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  width: 1rem;
  height: 1rem;
  color: var(--color-text-tertiary);
  pointer-events: none;
  z-index: 1;
}

.search-input {
  width: 100%;
  padding: 0.75rem 1rem 0.75rem 2.5rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 0.9rem;
  transition: all var(--transition-fast);
}

.search-input:focus {
  outline: none;
  border-color: var(--color-btn-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-input::placeholder {
  color: var(--color-text-tertiary);
}


.key-actions {
  display: flex;
  gap: var(--spacing-lg);
  flex-shrink: 0;
}

/* Button Variants */
.btn-sm {
  font-size: 0.75rem;
  padding: var(--spacing-xs) var(--spacing-sm);
}

.btn-xs {
  font-size: 0.625rem;
  padding: 0.25rem 0.5rem;
  gap: 0.25rem;
}

.btn-warning {
  background: var(--color-btn-warning);
  color: var(--color-text-primary);
}

.btn-warning:hover {
  background: var(--color-btn-warning-hover);
}

.btn-success {
  background: #10b981;
  color: white;
  border: 1px solid #059669;
}

.btn-success:hover:not(:disabled) {
  background: #059669;
  border-color: #047857;
}

.btn-icon {
  width: 1rem;
  height: 1rem;
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modern-modal {
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  max-width: 600px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.3);
  border: 1px solid var(--color-border-light);
  animation: modalSlideIn 0.3s ease-out;
}

@keyframes modalSlideIn {
  from {
    opacity: 0;
    transform: translateY(-20px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.modal-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-lg);
  border-bottom: 1px solid var(--color-border-light);
  background: var(--color-bg-secondary);
  position: relative;
}

.modal-icon {
  width: 3rem;
  height: 3rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  flex-shrink: 0;
}

.modal-icon.success {
  background: rgba(40, 167, 69, 0.1);
  color: var(--color-success);
}

.modal-icon svg {
  width: 1.5rem;
  height: 1.5rem;
}

.modal-header h3 {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  flex: 1;
}

.modal-close {
  position: absolute;
  top: var(--spacing-md);
  right: var(--spacing-md);
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  padding: var(--spacing-xs);
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.modal-close:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-tertiary);
}

.modal-close svg {
  width: 1.25rem;
  height: 1.25rem;
}

.modal-body {
  padding: var(--spacing-lg);
}

.key-display {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.key-field {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.key-field label {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.key-value {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
  position: relative;
}

.key-value code {
  flex: 1;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875rem;
  color: var(--color-text-primary);
  word-break: break-all;
}

.copy-btn {
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  padding: var(--spacing-xs);
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.copy-btn:hover {
  color: var(--color-btn-primary);
  background: var(--color-bg-primary);
}

.copy-btn svg {
  width: 1rem;
  height: 1rem;
}

.warning-message {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-md);
  background: rgba(255, 193, 7, 0.1);
  color: var(--color-warning);
  padding: var(--spacing-md);
  border-radius: var(--radius-md);
  border: 1px solid rgba(255, 193, 7, 0.2);
  margin-top: var(--spacing-lg);
}

.warning-icon {
  width: 1.25rem;
  height: 1.25rem;
  flex-shrink: 0;
  margin-top: 2px;
}

.warning-text {
  font-size: 0.875rem;
  line-height: 1.5;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-sm);
  padding: var(--spacing-lg);
  border-top: 1px solid var(--color-border-light);
  background: var(--color-bg-secondary);
}

/* Loading and Error States */
.loading {
  text-align: center;
  padding: var(--spacing-xxl);
  color: var(--color-text-secondary);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--spacing-md);
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  margin-top: var(--spacing-lg);
}

.loading::before {
  content: '';
  width: 2rem;
  height: 2rem;
  border: 2px solid var(--color-border);
  border-top: 2px solid var(--color-btn-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error {
  background: rgba(220, 53, 69, 0.1);
  color: var(--color-danger);
  padding: var(--spacing-md);
  border-radius: var(--radius-md);
  margin: var(--spacing-lg) 0;
  border: 1px solid rgba(220, 53, 69, 0.2);
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.error::before {
  content: '⚠';
  font-size: 1.2rem;
}

/* Responsive Design */
@media (max-width: 1024px) {
  .content-grid {
    grid-template-columns: 1fr;
    gap: var(--spacing-lg);
  }
  
  .page-header {
    flex-direction: column;
    gap: var(--spacing-md);
    align-items: stretch;
  }
  
  .header-actions {
    justify-content: center;
  }
}

@media (max-width: 768px) {
  .header-text {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--spacing-sm);
  }
  
  .info-grid {
    grid-template-columns: 1fr;
    gap: var(--spacing-md);
  }
  
  .card-header {
    flex-direction: column;
    gap: var(--spacing-md);
    align-items: stretch;
  }
  
  .card-actions {
    justify-content: center;
  }
  
  .key-header {
    flex-direction: column;
    gap: var(--spacing-md);
    align-items: stretch;
    padding: var(--spacing-md);
  }

  .key-actions {
    justify-content: center;
    gap: var(--spacing-md);
  }
  
  .modern-modal {
    width: 95%;
    margin: var(--spacing-md);
  }
}

@media (max-width: 480px) {
  .breadcrumb {
    font-size: 0.75rem;
  }
  
  .header-text h2 {
    font-size: 1.5rem;
  }
  
  .card-header-content {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--spacing-sm);
  }
  
  .key-value {
    flex-direction: column;
    align-items: stretch;
    gap: var(--spacing-sm);
  }
  
  .copy-btn {
    align-self: flex-end;
  }
}
</style>