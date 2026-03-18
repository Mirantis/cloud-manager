<template>
  <div class="route53-domain-detail">
    <nav class="breadcrumb">
      <router-link to="/aws/route53-domains">← Back to Route53 Domains</router-link>
    </nav>

    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>{{ domainName || hostedZoneId }}</h1>
            <p>Hosted Zone: {{ hostedZoneId }} | Account: {{ accountId }}</p>
          </div>
        </div>
        <div class="header-actions">
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || records.length === 0">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M14,2H6A2,2 0 0,0 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2M18,20H6V4H13V9H18V20Z"/>
            </svg>
            Download JSON
          </button>
          <button @click="loadData" class="btn btn-secondary" :disabled="loading">
            {{ loading ? 'Refreshing...' : 'Refresh' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading DNS records...</p>
    </div>

    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load DNS Records</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <div v-else class="main-content">
      <div class="filters">
        <div class="search-box">
          <svg class="search-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M9.5,3A6.5,6.5 0 0,1 16,9.5C16,11.11 15.41,12.59 14.44,13.73L14.71,14H15.5L20.5,19L19,20.5L14,15.5V14.71L13.73,14.44C12.59,15.41 11.11,16 9.5,16A6.5,6.5 0 0,1 3,9.5A6.5,6.5 0 0,1 9.5,3M9.5,5C7,5 5,7 5,9.5C5,12 7,14 9.5,14C12,14 14,12 14,9.5C14,7 12,5 9.5,5Z"/>
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search by name, type, or value..."
            class="search-input"
          />
        </div>
        <select v-model="filterType" class="filter-select">
          <option value="">All Types</option>
          <option value="A">A</option>
          <option value="AAAA">AAAA</option>
          <option value="CNAME">CNAME</option>
          <option value="MX">MX</option>
          <option value="TXT">TXT</option>
          <option value="NS">NS</option>
          <option value="SOA">SOA</option>
          <option value="SRV">SRV</option>
          <option value="CAA">CAA</option>
        </select>
      </div>

      <div class="table-container">
        <div v-if="filteredRecords.length === 0" class="empty-state">
          <h4>No records found</h4>
          <p>No DNS records match the current search and filter criteria.</p>
        </div>
        <table v-else class="records-table">
          <thead>
            <tr>
              <th @click="sortBy('name')" class="sortable">
                Name
                <span class="sort-indicator" v-if="sortField === 'name'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('type')" class="sortable">
                Type
                <span class="sort-indicator" v-if="sortField === 'type'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('ttl')" class="sortable">
                TTL
                <span class="sort-indicator" v-if="sortField === 'ttl'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Values</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(record, idx) in filteredRecords" :key="idx" class="data-row">
              <td><code class="record-name">{{ record.name }}</code></td>
              <td><span class="record-type">{{ record.type }}</span></td>
              <td>{{ record.ttl }}</td>
              <td>
                <div class="values-cell">
                  <span v-for="(v, i) in record.values" :key="i" class="value-item">{{ v }}</span>
                  <span v-if="record.values.length === 0" class="muted">-</span>
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
  name: 'Route53DomainDetail',
  props: {
    accountId: { type: String, required: true },
    hostedZoneId: { type: String, required: true }
  },
  data() {
    return {
      records: [],
      domainName: '',
      loading: true,
      error: null,
      searchQuery: '',
      filterType: '',
      sortField: 'name',
      sortDirection: 'asc'
    }
  },
  computed: {
    filteredRecords() {
      let result = this.records

      if (this.searchQuery) {
        const q = this.searchQuery.toLowerCase()
        result = result.filter(r =>
          (r.name && r.name.toLowerCase().includes(q)) ||
          (r.type && r.type.toLowerCase().includes(q)) ||
          (r.values && r.values.some(v => v && v.toLowerCase().includes(q)))
        )
      }

      if (this.filterType) {
        result = result.filter(r => r.type === this.filterType)
      }

      result.sort((a, b) => {
        let aVal = a[this.sortField]
        let bVal = b[this.sortField]
        if (typeof aVal === 'number' && typeof bVal === 'number') {
          return this.sortDirection === 'asc' ? aVal - bVal : bVal - aVal
        }
        aVal = String(aVal || '').toLowerCase()
        bVal = String(bVal || '').toLowerCase()
        const cmp = aVal.localeCompare(bVal)
        return this.sortDirection === 'asc' ? cmp : -cmp
      })

      return result
    }
  },
  watch: {
    accountId: 'loadData',
    hostedZoneId: 'loadData'
  },
  mounted() {
    this.domainName = this.$route.query.name || ''
    this.loadData()
  },
  methods: {
    async loadData() {
      try {
        this.loading = true
        this.error = null
        const zoneId = decodeURIComponent(this.hostedZoneId)
        const url = `/api/accounts/${this.accountId}/route53-domains/${encodeURIComponent(zoneId)}/records`
        const response = await axios.get(url)
        this.records = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load DNS records'
      } finally {
        this.loading = false
      }
    },
    sortBy(field) {
      if (this.sortField === field) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortDirection = 'asc'
      }
    },
    downloadJSON() {
      try {
        const exportData = {
          exported_at: new Date().toISOString(),
          account_id: this.accountId,
          hosted_zone_id: decodeURIComponent(this.hostedZoneId),
          domain_name: this.domainName,
          total_records: this.filteredRecords.length,
          records: this.filteredRecords
        }
        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)
        const fname = `route53-records-${this.domainName || 'zone'}-${new Date().toISOString().split('T')[0]}.json`
        const link = document.createElement('a')
        link.setAttribute('href', dataUri)
        link.setAttribute('download', fname)
        link.click()
      } catch (e) {
        console.error('Failed to download JSON:', e)
        alert('Failed to download JSON file')
      }
    }
  }
}
</script>

<style scoped>
.route53-domain-detail {
  min-height: 100vh;
  background: var(--color-bg-secondary);
  padding: 1.5rem;
}

.breadcrumb {
  margin-bottom: 1.5rem;
}

.breadcrumb a {
  color: var(--color-primary);
  text-decoration: none;
}

.breadcrumb a:hover {
  text-decoration: underline;
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
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.header-icon svg {
  width: 28px;
  height: 28px;
  color: white;
}

.title-content h1 {
  margin: 0;
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--color-text-primary);
}

.title-content p {
  margin: 0.25rem 0 0 0;
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}

.header-actions {
  display: flex;
  gap: 1rem;
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

.btn-secondary {
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.loading-container,
.error-container {
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

.main-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
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
  min-width: 280px;
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

.filter-select {
  padding: 0.75rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  font-size: 0.95rem;
  cursor: pointer;
}

.table-container {
  background: var(--color-bg-primary);
  border-radius: 12px;
  overflow-x: auto;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.empty-state {
  padding: 4rem 2rem;
  text-align: center;
}

.empty-state h4 {
  margin: 0 0 0.5rem 0;
  color: var(--color-text-primary);
}

.empty-state p {
  color: var(--color-text-secondary);
  margin: 0;
}

.records-table {
  width: 100%;
  min-width: 600px;
  border-collapse: collapse;
}

.records-table thead {
  background: var(--color-bg-secondary);
}

.records-table th {
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

.records-table th.sortable:hover {
  color: var(--color-primary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-primary);
}

.records-table tbody tr {
  border-top: 1px solid var(--color-border);
}

.records-table tbody tr:hover {
  background: var(--color-bg-hover);
}

.records-table td {
  padding: 1rem 1.5rem;
  color: var(--color-text-primary);
}

.record-name {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.9rem;
  background: var(--color-bg-secondary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  color: var(--color-primary);
}

.record-type {
  font-weight: 500;
}

.values-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.value-item {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.85rem;
  word-break: break-all;
}

.muted {
  color: var(--color-text-secondary);
}
</style>
