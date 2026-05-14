<template>
  <div class="eks-clusters-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12,2L1,21H23L12,2M12,6L18.9,21H5.1L12,6Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>EKS Clusters</h1>
            <p>{{ clusters.length }} cluster{{ clusters.length !== 1 ? 's' : '' }} across {{ uniqueAccounts.length }} accounts</p>
          </div>
        </div>
        <div class="header-actions">
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || clusters.length === 0">
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
      <p>Loading EKS clusters...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load EKS Clusters</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Summary Stats -->
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ clusters.length }}</div>
          <div class="stat-label">Total Clusters</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ runningClusters }}</div>
          <div class="stat-label">Active</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ totalNodeGroups }}</div>
          <div class="stat-label">Node Groups</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ publicClusters }}</div>
          <div class="stat-label">Public API</div>
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
            placeholder="Search by cluster name, ID, account, region..."
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
        <select v-model="filterStatus" class="filter-select">
          <option value="">All Statuses</option>
          <option value="ACTIVE">Active</option>
          <option value="CREATING">Creating</option>
          <option value="DELETING">Deleting</option>
          <option value="FAILED">Failed</option>
          <option value="PAUSED">Paused</option>
        </select>
      </div>

      <!-- Clusters Table -->
      <div class="table-container" v-if="filteredClusters.length > 0">
        <table class="data-table">
          <thead>
            <tr>
              <th @click="sortBy('cluster_name')" class="sortable">
                Cluster Name
                <span v-if="sortColumn === 'cluster_name'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('cluster_arn')" class="sortable">
                ARN
                <span v-if="sortColumn === 'cluster_arn'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('account_name')" class="sortable">
                Account
                <span v-if="sortColumn === 'account_name'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('region')" class="sortable">
                Region
                <span v-if="sortColumn === 'region'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('status')" class="sortable">
                Status
                <span v-if="sortColumn === 'status'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Platform Version</th>
              <th>Node Groups</th>
              <th>Auth Mode</th>
              <th>Public API</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="cluster in filteredClusters" :key="cluster.cluster_name + cluster.account_id + cluster.region">
              <td class="name-cell">
                <router-link
                  :to="{ name: 'EKSClusterDetail', params: { accountId: cluster.account_id, region: cluster.region, clusterName: cluster.cluster_name } }"
                  class="cluster-name cluster-link"
                >{{ cluster.cluster_name }}</router-link>
              </td>
              <td class="mono" :title="cluster.cluster_arn">{{ shortArn(cluster.cluster_arn) }}</td>
              <td>
                <span class="account-name">{{ cluster.account_name }}</span>
                <span class="account-id">{{ cluster.account_id }}</span>
              </td>
              <td class="mono">{{ cluster.region }}</td>
              <td>
                <span :class="['status-badge', getStatusClass(cluster.status)]">
                  {{ cluster.status }}
                </span>
              </td>
              <td class="mono">{{ cluster.platform_version || '-' }}</td>
              <td class="center">
                <span v-if="cluster.node_groups && cluster.node_groups.length > 0" class="ng-badge">
                  {{ cluster.node_groups.length }}
                </span>
                <span v-else class="ng-none">-</span>
              </td>
              <td>{{ cluster.auth_mode || '-' }}</td>
              <td class="center">
                <span v-if="cluster.public_access" class="feature-badge feature-public" title="Public API Access">
                  Public
                </span>
                <span v-else class="feature-badge feature-private" title="Private API Access">
                  Private
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Empty State -->
      <div v-else class="empty-state">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M12,2L1,21H23L12,2M12,6L18.9,21H5.1L12,6Z"/>
        </svg>
        <h3>No EKS Clusters Found</h3>
        <p v-if="searchQuery || filterAccount || filterRegion || filterStatus">
          No clusters match your current filters. Try adjusting your search criteria.
        </p>
        <p v-else>No EKS clusters are available in your accounts.</p>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'EKSClusters',
  data() {
    return {
      clusters: [],
      loading: false,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterRegion: '',
      filterStatus: '',
      sortColumn: 'cluster_name',
      sortDirection: 'asc'
    }
  },
  computed: {
    uniqueAccounts() {
      const accounts = new Map()
      this.clusters.forEach(cluster => {
        if (!accounts.has(cluster.account_id)) {
          accounts.set(cluster.account_id, {
            id: cluster.account_id,
            name: cluster.account_name
          })
        }
      })
      return Array.from(accounts.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    uniqueRegions() {
      return [...new Set(this.clusters.map(c => c.region))].sort()
    },
    runningClusters() {
      return this.clusters.filter(c => c.status === 'ACTIVE').length
    },
    totalNodeGroups() {
      return this.clusters.reduce((sum, c) => sum + (c.node_groups?.length || 0), 0)
    },
    publicClusters() {
      return this.clusters.filter(c => c.public_access).length
    },
    filteredClusters() {
      let result = [...this.clusters]

      // Search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        result = result.filter(cluster =>
          (cluster.cluster_name || '').toLowerCase().includes(query) ||
          (cluster.cluster_arn || '').toLowerCase().includes(query) ||
          (cluster.account_name || '').toLowerCase().includes(query) ||
          (cluster.account_id || '').toLowerCase().includes(query) ||
          (cluster.region || '').toLowerCase().includes(query) ||
          (cluster.platform_version || '').toLowerCase().includes(query)
        )
      }

      // Account filter
      if (this.filterAccount) {
        result = result.filter(c => c.account_id === this.filterAccount)
      }

      // Region filter
      if (this.filterRegion) {
        result = result.filter(c => c.region === this.filterRegion)
      }

      // Status filter
      if (this.filterStatus) {
        result = result.filter(c => c.status === this.filterStatus)
      }

      // Sorting
      result.sort((a, b) => {
        let aVal = a[this.sortColumn] ?? ''
        let bVal = b[this.sortColumn] ?? ''

        if (typeof aVal === 'string') aVal = aVal.toLowerCase()
        if (typeof bVal === 'string') bVal = bVal.toLowerCase()

        if (aVal < bVal) return this.sortDirection === 'asc' ? -1 : 1
        if (aVal > bVal) return this.sortDirection === 'asc' ? 1 : -1
        return 0
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
        const response = await axios.get('/api/eks-clusters')
        this.clusters = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load EKS clusters'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      try {
        await axios.post('/api/cache/eks-clusters/invalidate')
      } catch (error) {
        console.warn('Failed to invalidate cache:', error)
      }
      await this.loadData()
    },
    sortBy(column) {
      if (this.sortColumn === column) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortColumn = column
        this.sortDirection = 'asc'
      }
    },
    getStatusClass(status) {
      switch (status?.toUpperCase()) {
        case 'ACTIVE': return 'status-active'
        case 'CREATING': return 'status-creating'
        case 'DELETING': return 'status-deleting'
        case 'FAILED': return 'status-failed'
        case 'PAUSED': return 'status-paused'
        default: return 'status-default'
      }
    },
    shortArn(arn) {
      if (!arn) return ''
      return arn.length > 40 ? '...' + arn.slice(-37) : arn
    },
    downloadJSON() {
      const data = JSON.stringify(this.filteredClusters, null, 2)
      const blob = new Blob([data], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `eks-clusters-${new Date().toISOString().slice(0, 10)}.json`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    }
  }
}
</script>

<style scoped>
.eks-clusters-container {
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
  align-items: center;
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

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
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

.main-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.summary-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
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
  color: #f97316;
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

.data-table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
}

.data-table thead {
  background: var(--color-bg-secondary);
}

.data-table th,
.data-table td {
  padding: 1rem 1.5rem;
  text-align: left;
}

.data-table th {
  font-weight: 600;
  color: var(--color-text-secondary);
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  user-select: none;
}

.data-table th.sortable:hover {
  color: var(--color-primary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-primary);
}

.data-table tbody tr {
  border-top: 1px solid var(--color-border);
}

.data-table tbody tr:hover {
  background: var(--color-bg-hover);
}

.data-table td {
  color: var(--color-text-primary);
}

.mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.85rem;
  color: var(--color-text-secondary);
}

.center {
  text-align: center;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.cluster-name {
  font-weight: 600;
  color: #f97316;
}

.cluster-link {
  text-decoration: none;
  color: #f97316;
}

.cluster-link:hover {
  text-decoration: underline;
  color: #ea580c;
}

.account-name {
  display: block;
  font-weight: 500;
  color: var(--color-text-primary);
}

.account-id {
  display: block;
  font-size: 0.8rem;
  color: var(--color-text-secondary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-active {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.status-creating {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.status-deleting {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.status-failed {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.status-paused {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.status-default {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.ng-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-text-primary);
  background: var(--color-bg-tertiary);
}

.ng-none {
  color: var(--color-text-secondary);
}

.feature-badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.feature-public {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.feature-private {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.empty-state {
  padding: 4rem 2rem;
  text-align: center;
}

.empty-state svg {
  width: 64px;
  height: 64px;
  color: var(--color-text-secondary);
  margin-bottom: 1.5rem;
}

.empty-state h3 {
  margin: 0 0 0.5rem 0;
  color: var(--color-text-primary);
}

.empty-state p {
  color: var(--color-text-secondary);
  margin: 0;
}
</style>
