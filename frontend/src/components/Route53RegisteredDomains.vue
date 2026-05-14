<template>
  <div class="route53-registered-domains-container">
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M16.36,14C16.44,13.34 16.5,12.68 16.5,12C16.5,11.32 16.44,10.66 16.36,10H19.74C19.9,10.64 20,11.31 20,12C20,12.69 19.9,13.36 19.74,14M14.59,19.56C15.19,18.45 15.65,17.25 15.97,16H18.92C17.96,17.65 16.43,18.93 14.59,19.56M14.34,14H9.66C9.56,13.34 9.5,12.68 9.5,12C9.5,11.32 9.56,10.65 9.66,10H14.34C14.43,10.65 14.5,11.32 14.5,12C14.5,12.68 14.43,13.34 14.34,14M12,19.96C11.17,18.76 10.5,17.43 10.09,16H13.91C13.5,17.43 12.83,18.76 12,19.96M8,8H5.08C6.03,6.34 7.57,5.06 9.4,4.44C8.8,5.55 8.35,6.75 8,8M5.08,16H8C8.35,17.25 8.8,18.45 9.4,19.56C7.57,18.93 6.03,17.65 5.08,16M4.26,14C4.1,13.36 4,12.69 4,12C4,11.31 4.1,10.64 4.26,10H7.64C7.56,10.66 7.5,11.32 7.5,12C7.5,12.68 7.56,13.34 7.64,14M12,4.03C12.83,5.23 13.5,6.57 13.91,8H10.09C10.5,6.57 11.17,5.23 12,4.03M18.92,8H15.97C15.65,6.75 15.19,5.55 14.59,4.44C16.43,5.07 17.96,6.34 18.92,8M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>Route53 Registered Domains</h1>
            <p>{{ domains.length }} registered domains across {{ uniqueAccounts.length }} accounts</p>
          </div>
        </div>
        <div class="header-actions">
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || domains.length === 0">
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

    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading Route53 registered domains...</p>
    </div>

    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load Route53 Registered Domains</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <div v-else class="main-content">
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ domains.length }}</div>
          <div class="stat-label">Total Domains</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ autoRenewCount }}</div>
          <div class="stat-label">Auto-Renew On</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ transferLockCount }}</div>
          <div class="stat-label">Transfer Locked</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ expiringCount }}</div>
          <div class="stat-label">Expiring in 90 days</div>
        </div>
      </div>

      <div class="filters">
        <div class="search-box">
          <svg class="search-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M9.5,3A6.5,6.5 0 0,1 16,9.5C16,11.11 15.41,12.59 14.44,13.73L14.71,14H15.5L20.5,19L19,20.5L14,15.5V14.71L13.73,14.44C12.59,15.41 11.11,16 9.5,16A6.5,6.5 0 0,1 3,9.5A6.5,6.5 0 0,1 9.5,3M9.5,5C7,5 5,7 5,9.5C5,12 7,14 9.5,14C12,14 14,12 14,9.5C14,7 12,5 9.5,5Z"/>
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search by domain name or account..."
            class="search-input"
          />
        </div>
        <select v-model="filterAccount" class="filter-select">
          <option value="">All Accounts</option>
          <option v-for="account in uniqueAccounts" :key="account.id" :value="account.id">
            {{ account.name }} ({{ account.id }})
          </option>
        </select>
        <select v-model="filterAutoRenew" class="filter-select">
          <option value="">All Auto-Renew</option>
          <option value="on">Auto-Renew On</option>
          <option value="off">Auto-Renew Off</option>
        </select>
      </div>

      <div class="table-container">
        <div v-if="filteredDomains.length === 0" class="empty-state">
          <div class="empty-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z"/>
            </svg>
          </div>
          <h4>No domains found</h4>
          <p>No registered domains match the current search and filter criteria.</p>
        </div>
        <table v-else class="domains-table">
          <thead>
            <tr>
              <th @click="sortBy('domain_name')" class="sortable">
                Domain Name
                <span class="sort-indicator" v-if="sortField === 'domain_name'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('account_name')" class="sortable">
                Account
                <span class="sort-indicator" v-if="sortField === 'account_name'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('expiry')" class="sortable">
                Expiry
                <span class="sort-indicator" v-if="sortField === 'expiry'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Auto-Renew</th>
              <th>Transfer Lock</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="domain in filteredDomains" :key="domain.domain_name + domain.account_id" class="data-row" :class="{ 'expiring-soon': isExpiringSoon(domain.expiry) }">
              <td>
                <code class="domain-name">{{ domain.domain_name }}</code>
              </td>
              <td>
                <div class="account-info">
                  <div class="account-name">{{ domain.account_name }}</div>
                  <div class="account-id">{{ domain.account_id }}</div>
                </div>
              </td>
              <td>
                <span :class="['expiry-date', isExpiringSoon(domain.expiry) ? 'expiry-warning' : '']">
                  {{ domain.expiry || '-' }}
                </span>
              </td>
              <td>
                <span :class="['badge', domain.auto_renew ? 'badge-success' : 'badge-warning']">
                  {{ domain.auto_renew ? 'On' : 'Off' }}
                </span>
              </td>
              <td>
                <span :class="['badge', domain.transfer_lock ? 'badge-success' : 'badge-danger']">
                  {{ domain.transfer_lock ? 'Locked' : 'Unlocked' }}
                </span>
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
  name: 'Route53RegisteredDomains',
  data() {
    return {
      domains: [],
      loading: true,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterAutoRenew: '',
      sortField: 'domain_name',
      sortDirection: 'asc'
    }
  },
  computed: {
    uniqueAccounts() {
      const accounts = new Map()
      this.domains.forEach(d => {
        if (!accounts.has(d.account_id)) {
          accounts.set(d.account_id, { id: d.account_id, name: d.account_name })
        }
      })
      return Array.from(accounts.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    autoRenewCount() {
      return this.domains.filter(d => d.auto_renew).length
    },
    transferLockCount() {
      return this.domains.filter(d => d.transfer_lock).length
    },
    expiringCount() {
      return this.domains.filter(d => this.isExpiringSoon(d.expiry)).length
    },
    filteredDomains() {
      let result = this.domains

      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        result = result.filter(d =>
          d.domain_name.toLowerCase().includes(query) ||
          d.account_name.toLowerCase().includes(query) ||
          d.account_id.toLowerCase().includes(query)
        )
      }

      if (this.filterAccount) {
        result = result.filter(d => d.account_id === this.filterAccount)
      }

      if (this.filterAutoRenew === 'on') {
        result = result.filter(d => d.auto_renew)
      } else if (this.filterAutoRenew === 'off') {
        result = result.filter(d => !d.auto_renew)
      }

      result = [...result].sort((a, b) => {
        let aVal = a[this.sortField]
        let bVal = b[this.sortField]
        if (typeof aVal === 'string') {
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
        const response = await axios.get('/api/route53-registered-domains')
        this.domains = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load Route53 registered domains'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      if (this.canModify) {
        try {
          await axios.post('/api/cache/route53-registered-domains/invalidate')
        } catch (e) {
          console.warn('Failed to invalidate cache:', e)
        }
      }
      await this.loadData()
    },
    sortBy(field) {
      if (this.sortField === field) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortDirection = 'asc'
      }
    },
    isExpiringSoon(expiry) {
      if (!expiry) return false
      const expiryDate = new Date(expiry)
      const now = new Date()
      const diffDays = (expiryDate - now) / (1000 * 60 * 60 * 24)
      return diffDays >= 0 && diffDays <= 90
    },
    downloadJSON() {
      try {
        const exportData = {
          exported_at: new Date().toISOString(),
          total_domains: this.filteredDomains.length,
          domains: this.filteredDomains
        }
        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)
        const exportFileDefaultName = `aws-route53-registered-domains-${new Date().toISOString().split('T')[0]}.json`
        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (e) {
        console.error('Failed to download JSON:', e)
        alert('Failed to download JSON file')
      }
    }
  }
}
</script>

<style scoped>
.route53-registered-domains-container {
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
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
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

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.loading-container {
  background: var(--color-bg-primary);
  border-radius: 12px;
  padding: 4rem;
  text-align: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--color-border);
  border-top-color: #667eea;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 1rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-container {
  background: var(--color-bg-primary);
  border-radius: 12px;
  padding: 4rem;
  text-align: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.error-icon {
  width: 48px;
  height: 48px;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
}

.error-icon svg {
  width: 28px;
  height: 28px;
  color: #ef4444;
}

.summary-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.stat-card {
  background: var(--color-bg-primary);
  border-radius: 12px;
  padding: 1.5rem;
  text-align: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  color: var(--color-text-primary);
}

.stat-label {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.search-box {
  flex: 1;
  min-width: 200px;
  position: relative;
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  width: 18px;
  height: 18px;
  color: var(--color-text-secondary);
}

.search-input {
  width: 100%;
  padding: 0.75rem 0.75rem 0.75rem 2.5rem;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 0.95rem;
  box-sizing: border-box;
}

.search-input:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.15);
}

.filter-select {
  padding: 0.75rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 0.95rem;
  min-width: 160px;
}

.filter-select:focus {
  outline: none;
  border-color: #667eea;
}

.table-container {
  background: var(--color-bg-primary);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.empty-state {
  padding: 4rem;
  text-align: center;
  color: var(--color-text-secondary);
}

.empty-icon {
  width: 48px;
  height: 48px;
  background: var(--color-bg-tertiary);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
}

.empty-icon svg {
  width: 28px;
  height: 28px;
  color: var(--color-text-secondary);
}

.domains-table {
  width: 100%;
  border-collapse: collapse;
}

.domains-table th {
  padding: 1rem 1.25rem;
  text-align: left;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}

.domains-table th.sortable {
  cursor: pointer;
  user-select: none;
}

.domains-table th.sortable:hover {
  color: var(--color-text-primary);
}

.sort-indicator {
  margin-left: 0.25rem;
}

.domains-table td {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  font-size: 0.95rem;
}

.data-row:last-child td {
  border-bottom: none;
}

.data-row:hover td {
  background: var(--color-bg-hover);
}

.data-row.expiring-soon td {
  background: rgba(245, 158, 11, 0.05);
}

.domain-name {
  font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
  font-size: 0.9rem;
  background: var(--color-bg-secondary);
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  color: var(--color-text-primary);
}

.account-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.account-name {
  font-weight: 500;
}

.account-id {
  font-size: 0.8rem;
  color: var(--color-text-secondary);
  font-family: monospace;
}

.expiry-date {
  font-size: 0.9rem;
}

.expiry-warning {
  color: #f59e0b;
  font-weight: 600;
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 20px;
  font-size: 0.8rem;
  font-weight: 600;
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
</style>
