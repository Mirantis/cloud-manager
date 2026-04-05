<template>
  <div class="s3-buckets-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,15H15A3,3 0 0,1 12,12A3,3 0 0,1 15,9H19A3,3 0 0,1 22,12A3,3 0 0,1 19,15M5,15A3,3 0 0,1 2,12A3,3 0 0,1 5,9H9A3,3 0 0,1 12,12A3,3 0 0,1 9,15H5M19,10A2,2 0 0,0 17,12A2,2 0 0,0 19,14A2,2 0 0,0 21,12A2,2 0 0,0 19,10M5,10A2,2 0 0,0 3,12A2,2 0 0,0 5,14A2,2 0 0,0 7,12A2,2 0 0,0 5,10Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>S3 Buckets</h1>
            <p>{{ buckets.length }} buckets across {{ uniqueAccounts.length }} accounts</p>
          </div>
        </div>
        <div class="header-actions">
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || buckets.length === 0">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M14,2H6A2,2 0 0,0 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2M18,20H6V4H13V9H18V20Z"/>
            </svg>
            Download JSON
          </button>
          <button @click="refreshData" class="btn btn-secondary" :disabled="loading">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
            </svg>
            {{ loading ? 'Refreshing...' : 'Refresh' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading S3 buckets...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load S3 Buckets</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Summary Stats -->
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ buckets.length }}</div>
          <div class="stat-label">Total Buckets</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ encryptedBuckets }}</div>
          <div class="stat-label">Encrypted</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ publicBuckets }}</div>
          <div class="stat-label">Public Access</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ uniqueRegions.length }}</div>
          <div class="stat-label">Regions</div>
        </div>
      </div>

      <!-- Filters -->
      <div class="filters">
        <div class="search-box">
          <svg class="search-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M9.5,3A6.5,6.5 0 0,1 16,9.5C16,11.11 15.41,12.59 14.44,13.73L14.71,14H15.5L20.5,19L19,20.5L14,15.5V14.71L13.73,14.44C12.59,15.41 11.11,16 9.5,16A6.5,6.5 0 0,1 3,9.5A6.5,6.5 0 0,1 9.5,3M9.5,5C7,5 5,7 5,9.5C5,12 7,14 9.5,14C12,14 14,12 14,9.5C14,7 12,5 9.5,5Z"/>
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search by bucket name, account, or region..."
            class="search-input"
          />
        </div>
        <select v-model="filterAccount" class="filter-select">
          <option value="">All Accounts</option>
          <option v-for="account in uniqueAccounts" :key="account.id" :value="account.id">
            {{ account.name }} ({{ account.id }})
          </option>
        </select>
        <select v-model="filterRegion" class="filter-select">
          <option value="">All Regions</option>
          <option v-for="region in uniqueRegions" :key="region" :value="region">
            {{ region }}
          </option>
        </select>
        <select v-model="filterEncryption" class="filter-select">
          <option value="">All Buckets</option>
          <option value="encrypted">Encrypted Only</option>
          <option value="unencrypted">Unencrypted Only</option>
        </select>
        <select v-model="filterPublic" class="filter-select">
          <option value="">All Access Levels</option>
          <option value="public">Public Access</option>
          <option value="private">Private Only</option>
        </select>
      </div>

      <!-- Buckets Table -->
      <div class="table-container">
        <div v-if="filteredBuckets.length === 0" class="empty-state">
          <div class="empty-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,15H15A3,3 0 0,1 12,12A3,3 0 0,1 15,9H19A3,3 0 0,1 22,12A3,3 0 0,1 19,15M5,15A3,3 0 0,1 2,12A3,3 0 0,1 5,9H9A3,3 0 0,1 12,12A3,3 0 0,1 9,15H5M19,10A2,2 0 0,0 17,12A2,2 0 0,0 19,14A2,2 0 0,0 21,12A2,2 0 0,0 19,10M5,10A2,2 0 0,0 3,12A2,2 0 0,0 5,14A2,2 0 0,0 7,12A2,2 0 0,0 5,10Z"/>
            </svg>
          </div>
          <h4>No buckets found</h4>
          <p>No buckets match the current search and filter criteria.</p>
        </div>
        <table v-else class="buckets-table">
          <thead>
            <tr>
              <th @click="sortBy('name')" class="sortable">
                Bucket Name
                <span class="sort-indicator" v-if="sortField === 'name'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('account_name')" class="sortable">
                Account
                <span class="sort-indicator" v-if="sortField === 'account_name'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('region')" class="sortable">
                Region
                <span class="sort-indicator" v-if="sortField === 'region'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('creation_date')" class="sortable">
                Created
                <span class="sort-indicator" v-if="sortField === 'creation_date'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Security</th>
              <th>Features</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="bucket in filteredBuckets" :key="bucket.name" class="data-row">
              <td><code class="bucket-name">{{ bucket.name }}</code></td>
              <td>
                <div class="account-info">
                  <div class="account-name">{{ bucket.account_name }}</div>
                  <div class="account-id">{{ bucket.account_id }}</div>
                </div>
              </td>
              <td>{{ bucket.region }}</td>
              <td>
                <div class="date-info">
                  <div class="date-value">{{ formatDate(bucket.creation_date) }}</div>
                  <div class="date-relative">{{ formatRelativeDate(bucket.creation_date) }}</div>
                </div>
              </td>
              <td>
                <div class="security-badges">
                  <span :class="['badge', bucket.encrypted ? 'badge-success' : 'badge-warning']">
                    {{ bucket.encrypted ? 'Encrypted' : 'Not Encrypted' }}
                  </span>
                  <span :class="['badge', bucket.is_public ? 'badge-danger' : 'badge-success']">
                    {{ bucket.is_public ? 'Public' : 'Private' }}
                  </span>
                  <span v-if="bucket.versioning === 'Enabled'" class="badge badge-info">
                    Versioning
                  </span>
                </div>
              </td>
              <td>
                <div class="feature-badges">
                  <span v-if="bucket.has_lifecycle_policy" class="badge badge-secondary" title="Lifecycle Policy">
                    Lifecycle
                  </span>
                  <span v-if="bucket.has_logging" class="badge badge-secondary" title="Logging Enabled">
                    Logging
                  </span>
                </div>
              </td>
              <td>
                <div class="action-buttons">
                  <button
                    v-if="canModify"
                    @click="deleteBucket(bucket)"
                    class="action-btn action-btn-delete"
                    :disabled="loading"
                    title="Delete bucket (must be empty)"
                  >
                    <svg viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                    </svg>
                    Delete
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'S3Buckets',
  data() {
    return {
      buckets: [],
      loading: true,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterRegion: '',
      filterEncryption: '',
      filterPublic: '',
      sortField: 'creation_date',
      sortDirection: 'desc'
    }
  },
  computed: {
    uniqueAccounts() {
      const accounts = new Map()
      this.buckets.forEach(b => {
        if (!accounts.has(b.account_id)) {
          accounts.set(b.account_id, { id: b.account_id, name: b.account_name })
        }
      })
      return Array.from(accounts.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    uniqueRegions() {
      return [...new Set(this.buckets.map(b => b.region))].sort()
    },
    encryptedBuckets() {
      return this.buckets.filter(b => b.encrypted).length
    },
    publicBuckets() {
      return this.buckets.filter(b => b.is_public).length
    },
    filteredBuckets() {
      let result = this.buckets

      // Apply search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        result = result.filter(b =>
          b.name.toLowerCase().includes(query) ||
          b.account_name.toLowerCase().includes(query) ||
          b.account_id.toLowerCase().includes(query) ||
          b.region.toLowerCase().includes(query)
        )
      }

      // Apply filters
      if (this.filterAccount) {
        result = result.filter(b => b.account_id === this.filterAccount)
      }
      if (this.filterRegion) {
        result = result.filter(b => b.region === this.filterRegion)
      }
      if (this.filterEncryption === 'encrypted') {
        result = result.filter(b => b.encrypted)
      } else if (this.filterEncryption === 'unencrypted') {
        result = result.filter(b => !b.encrypted)
      }
      if (this.filterPublic === 'public') {
        result = result.filter(b => b.is_public)
      } else if (this.filterPublic === 'private') {
        result = result.filter(b => !b.is_public)
      }

      // Apply sorting
      result.sort((a, b) => {
        let aVal = a[this.sortField]
        let bVal = b[this.sortField]

        if (this.sortField === 'creation_date') {
          aVal = new Date(aVal)
          bVal = new Date(bVal)
        } else if (typeof aVal === 'string') {
          aVal = aVal?.toLowerCase() || ''
          bVal = bVal?.toLowerCase() || ''
        }

        if (this.sortDirection === 'asc') {
          return aVal < bVal ? -1 : (aVal > bVal ? 1 : 0)
        } else {
          return aVal > bVal ? -1 : (aVal < bVal ? 1 : 0)
        }
      })

      return result
    }
  },
  async mounted() {
    await this.loadData()
  },
  methods: {
    async loadData() {
      try {
        this.loading = true
        this.error = null
        const response = await axios.get('/api/s3-buckets')
        this.buckets = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load S3 buckets'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      if (this.canModify) {
        try {
          await axios.post('/api/cache/s3-buckets/invalidate')
        } catch (error) {
          console.warn('Failed to invalidate cache:', error)
        }
      }
      await this.loadData()
    },
    sortBy(field) {
      if (this.sortField === field) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortDirection = field === 'creation_date' ? 'desc' : 'asc'
      }
    },
    formatDate(dateString) {
      return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
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
    downloadJSON() {
      try {
        const exportData = {
          exported_at: new Date().toISOString(),
          total_buckets: this.filteredBuckets.length,
          encrypted_buckets: this.filteredBuckets.filter(b => b.encrypted).length,
          public_buckets: this.filteredBuckets.filter(b => b.is_public).length,
          buckets: this.filteredBuckets
        }

        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)
        const exportFileDefaultName = `aws-s3-buckets-${new Date().toISOString().split('T')[0]}.json`

        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (error) {
        console.error('Failed to download JSON:', error)
        alert('Failed to download JSON file')
      }
    },
    async deleteBucket(bucket) {
      const confirmed = confirm(
        `Are you sure you want to delete bucket "${bucket.name}"?\n\n` +
        `WARNING: The bucket must be empty to be deleted. This action cannot be undone.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        const url = `/api/accounts/${bucket.account_id}/regions/${bucket.region}/buckets/${bucket.name}`
        await axios.delete(url)

        // Reload data - the cache has been updated
        await this.loadData()

        alert(`Bucket ${bucket.name} deleted successfully`)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete bucket'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete bucket:', err)
      } finally {
        this.loading = false
      }
    }
  }
}
</script>

<style scoped>
.s3-buckets-container {
  min-height: 100vh;
  background: var(--color-bg-secondary);
  padding: 1.5rem;
}

.page-header {
  background: var(--color-bg-primary);
  border-radius: 12px;
  padding: 2rem;
  margin-bottom: 1.5rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 2rem;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.header-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #f97316 0%, #ea580c 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.header-icon svg {
  width: 28px;
  height: 28px;
  color: white;
}

.title-content h1 {
  margin: 0;
  font-size: 2rem;
  font-weight: 700;
  color: var(--color-text-primary);
}

.title-content p {
  margin: 0.25rem 0 0 0;
  color: var(--color-text-secondary);
  font-size: 0.95rem;
}

.header-actions {
  display: flex;
  gap: 1rem;
  flex-shrink: 0;
}

.btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  border: none;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-icon {
  width: 18px;
  height: 18px;
}

.btn-success {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  color: white;
}

.btn-success:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
}

.btn-secondary {
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-bg-hover);
}

.loading-container {
  background: var(--color-bg-primary);
  border-radius: 12px;
  padding: 4rem 2rem;
  text-align: center;
}

.loading-spinner {
  width: 48px;
  height: 48px;
  border: 4px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 1rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-container p {
  color: var(--color-text-secondary);
  margin: 0;
}

.error-container {
  background: var(--color-bg-primary);
  border-radius: 12px;
  padding: 4rem 2rem;
  text-align: center;
}

.error-icon {
  width: 64px;
  height: 64px;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1.5rem;
}

.error-icon svg {
  width: 32px;
  height: 32px;
  color: var(--color-danger);
}

.error-container h3 {
  margin: 0 0 0.5rem 0;
  color: var(--color-text-primary);
}

.error-container p {
  color: var(--color-text-secondary);
  margin: 0 0 1.5rem 0;
}

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.main-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.summary-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.stat-card {
  background: var(--color-bg-primary);
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  color: var(--color-primary);
  margin-bottom: 0.25rem;
}

.stat-label {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.filters {
  background: var(--color-bg-primary);
  border-radius: 12px;
  padding: 1.5rem;
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.search-box {
  flex: 1;
  min-width: 300px;
  position: relative;
}

.search-icon {
  position: absolute;
  left: 1rem;
  top: 50%;
  transform: translateY(-50%);
  width: 20px;
  height: 20px;
  color: var(--color-text-secondary);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 0.75rem 1rem 0.75rem 3rem;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  font-size: 0.95rem;
}

.search-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.filter-select {
  padding: 0.75rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  font-size: 0.95rem;
  cursor: pointer;
}

.filter-select:focus {
  outline: none;
  border-color: var(--color-primary);
}

.table-container {
  background: var(--color-bg-primary);
  border-radius: 12px;
  overflow-x: auto;
  overflow-y: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.empty-state {
  padding: 4rem 2rem;
  text-align: center;
}

.empty-icon {
  width: 64px;
  height: 64px;
  background: var(--color-bg-secondary);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1.5rem;
}

.empty-icon svg {
  width: 32px;
  height: 32px;
  color: var(--color-text-secondary);
}

.empty-state h4 {
  margin: 0 0 0.5rem 0;
  color: var(--color-text-primary);
}

.empty-state p {
  color: var(--color-text-secondary);
  margin: 0;
}

.buckets-table {
  width: 100%;
  min-width: 800px;
  border-collapse: collapse;
}

.buckets-table thead {
  background: var(--color-bg-secondary);
}

.buckets-table th {
  padding: 1rem 1.5rem;
  text-align: left;
  font-weight: 600;
  color: var(--color-text-secondary);
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  user-select: none;
}

.buckets-table th.sortable:hover {
  color: var(--color-primary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-primary);
}

.buckets-table tbody tr {
  border-top: 1px solid var(--color-border);
}

.buckets-table tbody tr:hover {
  background: var(--color-bg-hover);
}

.buckets-table td {
  padding: 1rem 1.5rem;
  color: var(--color-text-primary);
}

.bucket-name {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.9rem;
  background: var(--color-bg-secondary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  color: var(--color-primary);
}

.account-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.account-name {
  font-weight: 500;
}

.account-id {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.date-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.date-value {
  font-weight: 500;
}

.date-relative {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
}

.security-badges,
.feature-badges {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  width: fit-content;
}

.badge-success {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.badge-warning {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.badge-danger {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.badge-info {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.badge-secondary {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.5rem 0.75rem;
  border: none;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn svg {
  width: 14px;
  height: 14px;
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-btn-delete {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: white;
}

.action-btn-delete:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    align-items: flex-start;
  }

  .header-actions {
    width: 100%;
  }

  .btn {
    flex: 1;
  }

  .summary-stats {
    grid-template-columns: repeat(2, 1fr);
  }

  .filters {
    flex-direction: column;
  }

  .search-box {
    min-width: 100%;
  }

  .filter-select {
    width: 100%;
  }

  .buckets-table {
    font-size: 0.85rem;
  }

  .buckets-table th,
  .buckets-table td {
    padding: 0.75rem 1rem;
  }
}
</style>
