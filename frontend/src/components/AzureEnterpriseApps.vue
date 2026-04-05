<template>
  <div class="azure-apps-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon" style="color: #0078d4;">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M3.33 7L10.82 9.65L9 3L3.33 7M14 15.76V17.75C14 18.37 13.5 18.86 12.88 18.86H11.09C10.5 18.86 10 18.37 10 17.75V15.76C9.68 15.56 9.41 15.29 9.21 14.97C8.93 14.53 8.78 14 8.78 13.45V9.65L1 7L10 3.74L19 7V13.45C19 14 18.85 14.53 18.57 14.97C18.37 15.29 18.1 15.56 17.78 15.76V17.75C17.78 19.3 16.5 20.55 14.91 20.55H13.09C11.5 20.55 10.22 19.3 10.22 17.75V15.76C9.68 15.56 9.21 15.17 8.85 14.68C8.41 14.07 8.17 13.32 8.17 12.54V8.82L1.67 7L10 4L18.33 7V12.54C18.33 13.32 18.09 14.07 17.65 14.68C17.29 15.17 16.82 15.56 16.28 15.76V17.75C16.28 18.42 15.67 18.97 14.91 18.97H13.09C12.33 18.97 11.72 18.42 11.72 17.75V15.76H16.28V12.54C16.28 11.65 15.55 10.92 14.66 10.92H13.34C12.45 10.92 11.72 11.65 11.72 12.54V15.76H10.22V12.54C10.22 10.82 11.62 9.42 13.34 9.42H14.66C16.38 9.42 17.78 10.82 17.78 12.54V15.76H14V17.75C14 18.37 13.5 18.86 12.88 18.86H11.09C10.5 18.86 10 18.37 10 17.75V15.76H14Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>Azure Enterprise Applications</h1>
            <p>{{ filteredApps.length }} applications</p>
          </div>
        </div>
        <div class="header-actions">
          <button @click="downloadAppsJSON" class="btn btn-success" :disabled="loading || apps.length === 0">
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

    <!-- Info Message -->
    <div v-if="!azureConfigured" class="info-message">
      <svg viewBox="0 0 24 24" fill="currentColor">
        <path d="M13,9H11V7H13M13,17H11V11H13M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z"/>
      </svg>
      <div>
        <strong>Azure not configured</strong>
        <p>Set AZURE_TENANT_ID, AZURE_CLIENT_ID, and AZURE_CLIENT_SECRET environment variables to enable Azure features.</p>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading enterprise applications...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load Applications</h3>
      <p>{{ error }}</p>
      <button @click="refreshData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Search and Filter -->
      <div class="app-filters">
        <div class="search-box">
          <svg class="search-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M9.5,3A6.5,6.5 0 0,1 16,9.5C16,11.11 15.41,12.59 14.44,13.73L14.71,14H15.5L20.5,19L19,20.5L14,15.5V14.71L13.73,14.44C12.59,15.41 11.11,16 9.5,16A6.5,6.5 0 0,1 3,9.5A6.5,6.5 0 0,1 9.5,3M9.5,5C7,5 5,7 5,9.5C5,12 7,14 9.5,14C12,14 14,12 14,9.5C14,7 12,5 9.5,5Z"/>
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search by name, app ID, or type..."
            class="search-input"
          />
        </div>
        <div class="filter-buttons">
          <button
            @click="filter = 'all'"
            :class="['filter-btn', { active: filter === 'all' }]"
          >
            All ({{ apps.length }})
          </button>
          <button
            @click="filter = 'enabled'"
            :class="['filter-btn', { active: filter === 'enabled' }]"
          >
            Enabled ({{ enabledCount }})
          </button>
          <button
            @click="filter = 'disabled'"
            :class="['filter-btn', { active: filter === 'disabled' }]"
          >
            Disabled ({{ disabledCount }})
          </button>
        </div>
      </div>

      <!-- Apps Table -->
      <div class="apps-table-container">
        <table class="apps-table">
          <thead>
            <tr>
              <th>Display Name</th>
              <th>Application ID</th>
              <th>Type</th>
              <th>Status</th>
              <th @click="sortBy('created_datetime')" class="sortable">
                Created
                <span class="sort-indicator" v-if="sortField === 'created_datetime'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="app in filteredApps" :key="app.id">
              <td>
                <div class="app-name">
                  <strong>{{ app.display_name }}</strong>
                  <span v-if="app.tags && app.tags.length > 0" class="app-tags">
                    <span v-for="tag in app.tags" :key="tag" class="tag">{{ tag }}</span>
                  </span>
                </div>
              </td>
              <td><code class="code-text">{{ app.app_id }}</code></td>
              <td>
                <span class="type-badge">{{ app.service_principal_type }}</span>
              </td>
              <td>
                <span :class="['status-badge', app.account_enabled ? 'status-enabled' : 'status-disabled']">
                  {{ app.account_enabled ? 'Enabled' : 'Disabled' }}
                </span>
              </td>
              <td>{{ formatDate(app.created_datetime) }}</td>
              <td>
                <div class="action-buttons">
                  <button @click="viewAppDetails(app)" class="btn btn-sm" title="View Details">
                    <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M12,9A3,3 0 0,0 9,12A3,3 0 0,0 12,15A3,3 0 0,0 15,12A3,3 0 0,0 12,9M12,17A5,5 0 0,1 7,12A5,5 0 0,1 12,7A5,5 0 0,1 17,12A5,5 0 0,1 12,17M12,4.5C7,4.5 2.73,7.61 1,12C2.73,16.39 7,19.5 12,19.5C17,19.5 21.27,16.39 23,12C21.27,7.61 17,4.5 12,4.5Z"/>
                    </svg>
                  </button>
                  <button v-if="canModify" @click="confirmDelete(app)" class="btn btn-sm btn-danger" title="Delete Application">
                    <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-if="filteredApps.length === 0 && !loading && !error" class="empty-state">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M19,13H13V19H11V13H5V11H11V5H13V11H19V13Z"/>
          </svg>
          <h3>No applications found</h3>
          <p v-if="searchQuery">Try adjusting your search criteria</p>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click="showDeleteModal = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>Confirm Deletion</h2>
          <button @click="showDeleteModal = false" class="modal-close">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"/>
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <p>Are you sure you want to delete this enterprise application?</p>
          <div class="app-details-box">
            <p><strong>Display Name:</strong> {{ appToDelete?.display_name }}</p>
            <p><strong>Application ID:</strong> {{ appToDelete?.app_id }}</p>
            <p><strong>Type:</strong> {{ appToDelete?.service_principal_type }}</p>
          </div>
          <p class="warning-text">This action cannot be undone.</p>
        </div>
        <div class="modal-footer">
          <button @click="showDeleteModal = false" class="btn btn-secondary">Cancel</button>
          <button @click="deleteApp" class="btn btn-danger" :disabled="deleting">
            {{ deleting ? 'Deleting...' : 'Delete Application' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Details Modal -->
    <div v-if="showDetailsModal" class="modal-overlay" @click="showDetailsModal = false">
      <div class="modal-content large" @click.stop>
        <div class="modal-header">
          <h2>Application Details</h2>
          <button @click="showDetailsModal = false" class="modal-close">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"/>
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <div v-if="selectedApp" class="details-grid">
            <div class="detail-item">
              <label>Display Name</label>
              <p>{{ selectedApp.display_name }}</p>
            </div>
            <div class="detail-item">
              <label>Application ID</label>
              <p><code class="code-text">{{ selectedApp.app_id }}</code></p>
            </div>
            <div class="detail-item">
              <label>Object ID</label>
              <p><code class="code-text">{{ selectedApp.id }}</code></p>
            </div>
            <div class="detail-item">
              <label>Type</label>
              <p>{{ selectedApp.service_principal_type }}</p>
            </div>
            <div class="detail-item">
              <label>Status</label>
              <p>
                <span :class="['status-badge', selectedApp.account_enabled ? 'status-enabled' : 'status-disabled']">
                  {{ selectedApp.account_enabled ? 'Enabled' : 'Disabled' }}
                </span>
              </p>
            </div>
            <div class="detail-item">
              <label>Created</label>
              <p>{{ formatDate(selectedApp.created_datetime) }}</p>
            </div>
            <div class="detail-item" v-if="selectedApp.app_owner_org_id">
              <label>Owner Organization ID</label>
              <p><code class="code-text">{{ selectedApp.app_owner_org_id }}</code></p>
            </div>
            <div class="detail-item">
              <label>App Role Assignment Required</label>
              <p>{{ selectedApp.app_role_assignment_required ? 'Yes' : 'No' }}</p>
            </div>
            <div class="detail-item full-width" v-if="selectedApp.homepage">
              <label>Homepage</label>
              <p><a :href="selectedApp.homepage" target="_blank" class="link">{{ selectedApp.homepage }}</a></p>
            </div>
            <div class="detail-item full-width" v-if="selectedApp.reply_urls && selectedApp.reply_urls.length > 0">
              <label>Reply URLs</label>
              <ul class="url-list">
                <li v-for="(url, index) in selectedApp.reply_urls" :key="index">{{ url }}</li>
              </ul>
            </div>
            <div class="detail-item full-width" v-if="selectedApp.tags && selectedApp.tags.length > 0">
              <label>Tags</label>
              <div class="tags-list">
                <span v-for="tag in selectedApp.tags" :key="tag" class="tag">{{ tag }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showDetailsModal = false" class="btn btn-secondary">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'AzureEnterpriseApps',
  data() {
    return {
      apps: [],
      loading: false,
      error: null,
      searchQuery: '',
      filter: 'all',
      showDeleteModal: false,
      showDetailsModal: false,
      appToDelete: null,
      selectedApp: null,
      deleting: false,
      azureConfigured: true,
      sortField: 'created_datetime',
      sortDirection: 'desc'
    }
  },
  computed: {
    filteredApps() {
      let filtered = this.apps

      // Apply status filter
      if (this.filter === 'enabled') {
        filtered = filtered.filter(app => app.account_enabled)
      } else if (this.filter === 'disabled') {
        filtered = filtered.filter(app => !app.account_enabled)
      }

      // Apply search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        filtered = filtered.filter(app =>
          app.display_name.toLowerCase().includes(query) ||
          app.app_id.toLowerCase().includes(query) ||
          app.service_principal_type.toLowerCase().includes(query)
        )
      }

      // Apply sorting
      filtered.sort((a, b) => {
        let aVal = a[this.sortField]
        let bVal = b[this.sortField]

        // Handle null/undefined values
        if (aVal == null && bVal == null) return 0
        if (aVal == null) return 1
        if (bVal == null) return -1

        // Handle date/time fields
        if (this.sortField === 'created_datetime') {
          aVal = new Date(aVal).getTime()
          bVal = new Date(bVal).getTime()
        }
        // Handle string fields - convert to lowercase for case-insensitive comparison
        else if (typeof aVal === 'string' || typeof bVal === 'string') {
          aVal = String(aVal || '').toLowerCase()
          bVal = String(bVal || '').toLowerCase()
        }

        // Perform comparison
        let comparison = 0
        if (aVal < bVal) {
          comparison = -1
        } else if (aVal > bVal) {
          comparison = 1
        }

        // Reverse if descending
        return this.sortDirection === 'asc' ? comparison : -comparison
      })

      return filtered
    },
    enabledCount() {
      return this.apps.filter(app => app.account_enabled).length
    },
    disabledCount() {
      return this.apps.filter(app => !app.account_enabled).length
    }
  },
  methods: {
    async fetchApps() {
      this.loading = true
      this.error = null
      try {
        const response = await fetch('/api/azure/enterprise-applications')
        if (!response.ok) {
          if (response.status === 404) {
            this.azureConfigured = false
            this.apps = []
            return
          }
          throw new Error(`HTTP error! status: ${response.status}`)
        }
        this.apps = await response.json()
        this.azureConfigured = true
      } catch (err) {
        this.error = err.message
        console.error('Error fetching Azure enterprise applications:', err)
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      await this.fetchApps()
    },
    confirmDelete(app) {
      this.appToDelete = app
      this.showDeleteModal = true
    },
    async deleteApp() {
      if (!this.appToDelete) return

      this.deleting = true
      try {
        const response = await fetch(`/api/azure/enterprise-applications/${this.appToDelete.id}`, {
          method: 'DELETE'
        })

        if (!response.ok) {
          const errorData = await response.json()
          throw new Error(errorData.error || 'Failed to delete application')
        }

        // Remove from local array
        this.apps = this.apps.filter(app => app.id !== this.appToDelete.id)
        this.showDeleteModal = false
        this.appToDelete = null
      } catch (err) {
        alert(`Failed to delete application: ${err.message}`)
      } finally {
        this.deleting = false
      }
    },
    viewAppDetails(app) {
      this.selectedApp = app
      this.showDetailsModal = true
    },
    downloadAppsJSON() {
      const dataStr = JSON.stringify(this.apps, null, 2)
      const dataBlob = new Blob([dataStr], { type: 'application/json' })
      const url = URL.createObjectURL(dataBlob)
      const link = document.createElement('a')
      link.href = url
      link.download = `azure-enterprise-apps-${new Date().toISOString()}.json`
      link.click()
      URL.revokeObjectURL(url)
    },
    sortBy(field) {
      if (this.sortField === field) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortDirection = field === 'created_datetime' ? 'desc' : 'asc'
      }
    },
    formatDate(dateString) {
      if (!dateString) {
        return 'N/A'
      }
      const date = new Date(dateString)
      if (isNaN(date.getTime())) {
        return 'N/A'
      }
      return date.toLocaleString()
    }
  },
  mounted() {
    this.fetchApps()
  }
}
</script>

<style scoped>
.azure-apps-container {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  padding: var(--spacing-xl);
  margin-bottom: var(--spacing-xl);
  box-shadow: var(--shadow);
  border: 1px solid var(--color-border-light);
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--spacing);
}

.header-title {
  display: flex;
  align-items: center;
  gap: var(--spacing);
}

.header-icon {
  width: 3rem;
  height: 3rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-secondary);
  border-radius: var(--radius);
}

.header-icon svg {
  width: 1.75rem;
  height: 1.75rem;
}

.title-content h1 {
  font-size: 1.75rem;
  font-weight: 600;
  margin-bottom: var(--spacing-xs);
}

.title-content p {
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

.header-actions {
  display: flex;
  gap: var(--spacing-sm);
}

.info-message {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing);
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.2);
  color: var(--color-btn-primary);
  padding: var(--spacing);
  border-radius: var(--radius);
  margin-bottom: var(--spacing-xl);
}

.info-message svg {
  width: 1.5rem;
  height: 1.5rem;
  flex-shrink: 0;
}

.info-message strong {
  display: block;
  margin-bottom: var(--spacing-xs);
}

.info-message p {
  font-size: 0.875rem;
  margin: 0;
}

.loading-container, .error-container {
  text-align: center;
  padding: var(--spacing-2xl);
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow);
}

.loading-spinner {
  width: 2.5rem;
  height: 2.5rem;
  border: 3px solid var(--color-border);
  border-top: 3px solid var(--color-btn-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto var(--spacing);
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-icon {
  width: 3rem;
  height: 3rem;
  margin: 0 auto var(--spacing);
  color: var(--color-danger);
}

.error-icon svg {
  width: 100%;
  height: 100%;
}

.app-filters {
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  padding: var(--spacing);
  margin-bottom: var(--spacing);
  box-shadow: var(--shadow);
  border: 1px solid var(--color-border-light);
}

.search-box {
  position: relative;
  margin-bottom: var(--spacing);
}

.search-icon {
  position: absolute;
  left: var(--spacing);
  top: 50%;
  transform: translateY(-50%);
  width: 1.25rem;
  height: 1.25rem;
  color: var(--color-text-tertiary);
}

.search-input {
  width: 100%;
  padding: var(--spacing-sm) var(--spacing) var(--spacing-sm) 3rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  font-size: 0.875rem;
  transition: all var(--transition-fast);
}

.search-input:focus {
  outline: none;
  border-color: var(--color-btn-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.filter-buttons {
  display: flex;
  gap: var(--spacing-sm);
  flex-wrap: wrap;
}

.filter-btn {
  padding: var(--spacing-sm) var(--spacing);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all var(--transition-fast);
}

.filter-btn:hover {
  background: var(--color-bg-tertiary);
  border-color: var(--color-btn-primary);
}

.filter-btn.active {
  background: var(--color-btn-primary);
  border-color: var(--color-btn-primary);
  color: white;
}

.apps-table-container {
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  overflow-x: auto;
  overflow-y: hidden;
  box-shadow: var(--shadow);
  border: 1px solid var(--color-border-light);
}

.apps-table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
}

.apps-table th {
  background: var(--color-bg-secondary);
  padding: var(--spacing);
  text-align: left;
  font-weight: 600;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-secondary);
  border-bottom: 1px solid var(--color-border);
}

.apps-table th.sortable {
  cursor: pointer;
  user-select: none;
  transition: color var(--transition-fast);
}

.apps-table th.sortable:hover {
  color: var(--color-btn-primary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-btn-primary);
}

.apps-table td {
  padding: var(--spacing);
  border-bottom: 1px solid var(--color-border);
  font-size: 0.875rem;
}

.apps-table tbody tr {
  transition: background-color var(--transition-fast);
}

.apps-table tbody tr:hover {
  background: var(--color-bg-secondary);
}

.app-name {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.app-tags {
  display: flex;
  gap: var(--spacing-xs);
  flex-wrap: wrap;
}

.tag {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.code-text {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
  background: var(--color-bg-secondary);
  padding: 0.125rem 0.375rem;
  border-radius: var(--radius-sm);
  color: var(--color-text-primary);
}

.type-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-primary);
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 600;
}

.status-enabled {
  background: rgba(16, 185, 129, 0.1);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.status-disabled {
  background: rgba(107, 114, 128, 0.1);
  color: var(--color-btn-secondary);
  border: 1px solid rgba(107, 114, 128, 0.2);
}

.action-buttons {
  display: flex;
  gap: var(--spacing-xs);
}

.empty-state {
  text-align: center;
  padding: var(--spacing-2xl);
  color: var(--color-text-secondary);
}

.empty-state svg {
  width: 3rem;
  height: 3rem;
  margin-bottom: var(--spacing);
  opacity: 0.5;
}

.empty-state h3 {
  margin-bottom: var(--spacing-sm);
  color: var(--color-text-primary);
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: var(--spacing);
}

.modal-content {
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  max-width: 32rem;
  width: 100%;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: var(--shadow-lg);
}

.modal-content.large {
  max-width: 48rem;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-lg);
  border-bottom: 1px solid var(--color-border);
}

.modal-header h2 {
  font-size: 1.25rem;
  font-weight: 600;
}

.modal-close {
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  padding: var(--spacing-xs);
  border-radius: var(--radius);
  transition: all var(--transition-fast);
}

.modal-close:hover {
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
}

.modal-close svg {
  width: 1.5rem;
  height: 1.5rem;
}

.modal-body {
  padding: var(--spacing-lg);
}

.app-details-box {
  background: var(--color-bg-secondary);
  padding: var(--spacing);
  border-radius: var(--radius);
  margin: var(--spacing) 0;
}

.app-details-box p {
  margin: var(--spacing-xs) 0;
  font-size: 0.875rem;
}

.warning-text {
  color: var(--color-danger);
  font-weight: 500;
  margin-top: var(--spacing);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-sm);
  padding: var(--spacing-lg);
  border-top: 1px solid var(--color-border);
}

.details-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--spacing-lg);
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.detail-item.full-width {
  grid-column: 1 / -1;
}

.detail-item label {
  font-weight: 600;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-secondary);
}

.detail-item p {
  font-size: 0.875rem;
  color: var(--color-text-primary);
}

.link {
  color: var(--color-btn-primary);
  text-decoration: none;
  transition: opacity var(--transition-fast);
}

.link:hover {
  opacity: 0.8;
  text-decoration: underline;
}

.url-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.url-list li {
  padding: var(--spacing-xs) 0;
  font-size: 0.875rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.tags-list {
  display: flex;
  gap: var(--spacing-xs);
  flex-wrap: wrap;
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    align-items: flex-start;
  }

  .apps-table {
    font-size: 0.75rem;
  }

  .apps-table th,
  .apps-table td {
    padding: var(--spacing-sm);
  }

  .details-grid {
    grid-template-columns: 1fr;
  }
}
</style>
