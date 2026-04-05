<template>
  <div class="public-ips">
    <h1>Public IP Addresses</h1>
    <p class="description">
      This page shows all public IP addresses used by EC2 instances, load balancers, and NAT gateways 
      across all regions in all accessible AWS accounts.
    </p>

    <div v-if="loading" class="loading">
      <div class="spinner"></div>
      <p>Loading public IP addresses...</p>
    </div>

    <div v-else-if="error" class="error">
      <h3>Error loading public IPs</h3>
      <p>{{ error }}</p>
      <button @click="loadPublicIPs" class="retry-btn">Retry</button>
    </div>

    <div v-else class="content">
      <div class="summary">
        <div class="summary-stats">
          <div class="summary-item">
            <span class="label">Total IPs:</span>
            <span class="value">{{ publicIPs.length }}</span>
          </div>
          <div class="summary-item">
            <span class="label">Accounts:</span>
            <span class="value">{{ uniqueAccounts.length }}</span>
          </div>
          <div class="summary-item">
            <span class="label">Regions:</span>
            <span class="value">{{ uniqueRegions.length }}</span>
          </div>
        </div>
        <div class="summary-actions">
          <button @click="refreshData" class="btn btn-secondary" :disabled="loading">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
            </svg>
            {{ loading ? 'Refreshing...' : 'Refresh' }}
          </button>
          <button @click="downloadIPsJSON" class="btn btn-success" :disabled="loading || publicIPs.length === 0">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M14,2H6A2,2 0 0,0 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2M18,20H6V4H13V9H18V20Z"/>
            </svg>
            Download JSON
          </button>
        </div>
      </div>

      <div class="filters">
        <input 
          v-model="searchQuery" 
          type="text" 
          placeholder="Search by IP, account, region, or resource..."
          class="search-input"
        />
        <select v-model="filterAccount" class="filter-select">
          <option value="">All Accounts</option>
          <option v-for="account in uniqueAccounts" :key="account.id" :value="account.id">
            {{ account.name }} ({{ account.id }})
          </option>
        </select>
        <select v-model="filterResourceType" class="filter-select">
          <option value="">All Resource Types</option>
          <option value="EC2">EC2 Instances</option>
          <option value="application">Application Load Balancer</option>
          <option value="network">Network Load Balancer</option>
          <option value="gateway">Gateway Load Balancer</option>
          <option value="CLB">Classic Load Balancer</option>
          <option value="NAT">NAT Gateway</option>
        </select>
        <select v-model="filterRegion" class="filter-select">
          <option value="">All Regions</option>
          <option v-for="region in uniqueRegions" :key="region" :value="region">
            {{ region }}
          </option>
        </select>
      </div>

      <div class="table-container">
        <table class="ip-table">
          <thead>
            <tr>
              <th @click="sortBy('ip_address')" class="sortable">
                IP Address
                <span v-if="sortField === 'ip_address'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th @click="sortBy('account_name')" class="sortable">
                Account
                <span v-if="sortField === 'account_name'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th @click="sortBy('region')" class="sortable">
                Region
                <span v-if="sortField === 'region'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th @click="sortBy('resource_type')" class="sortable">
                Resource Type
                <span v-if="sortField === 'resource_type'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th>Resource Name</th>
              <th>Resource ID</th>
              <th @click="sortBy('state')" class="sortable state-column">
                State
                <span v-if="sortField === 'state'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="ip in filteredAndSortedIPs" :key="`${ip.ip_address}-${ip.resource_id}`" class="ip-row">
              <td class="ip-address">
                <code>{{ ip.ip_address }}</code>
              </td>
              <td class="account">
                <div class="account-info">
                  <div class="account-name">{{ ip.account_name }}</div>
                  <div class="account-id">{{ ip.account_id }}</div>
                </div>
              </td>
              <td class="region">{{ ip.region }}</td>
              <td class="resource-type">
                <span :class="`resource-badge resource-${ip.resource_type.toLowerCase()}`">
                  {{ ip.resource_type }}
                </span>
              </td>
              <td class="resource-name">
                {{ ip.resource_name || '-' }}
              </td>
              <td class="resource-id">
                <code>{{ ip.resource_id }}</code>
              </td>
              <td class="state state-column">
                <span :class="`state-badge state-${ip.state?.toLowerCase()}`">
                  {{ ip.state || '-' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="filteredAndSortedIPs.length === 0" class="no-results">
        <p>No public IP addresses found matching your filters.</p>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'PublicIPs',
  data() {
    return {
      publicIPs: [],
      loading: true,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterResourceType: '',
      filterRegion: '',
      sortField: 'ip_address',
      sortDirection: 'asc'
    }
  },
  computed: {
    uniqueAccounts() {
      const accountMap = new Map()
      this.publicIPs.forEach(ip => {
        if (!accountMap.has(ip.account_id)) {
          accountMap.set(ip.account_id, {
            id: ip.account_id,
            name: ip.account_name
          })
        }
      })
      return Array.from(accountMap.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    uniqueRegions() {
      const regions = [...new Set(this.publicIPs.map(ip => ip.region))]
      return regions.sort()
    },
    filteredAndSortedIPs() {
      let filtered = this.publicIPs

      // Apply search filter
      if (this.searchQuery.trim()) {
        const query = this.searchQuery.toLowerCase()
        filtered = filtered.filter(ip =>
          ip.ip_address.toLowerCase().includes(query) ||
          ip.account_name.toLowerCase().includes(query) ||
          ip.account_id.toLowerCase().includes(query) ||
          ip.region.toLowerCase().includes(query) ||
          ip.resource_type.toLowerCase().includes(query) ||
          (ip.resource_name && ip.resource_name.toLowerCase().includes(query)) ||
          ip.resource_id.toLowerCase().includes(query) ||
          (ip.state && ip.state.toLowerCase().includes(query))
        )
      }

      // Apply account filter
      if (this.filterAccount) {
        filtered = filtered.filter(ip => ip.account_id === this.filterAccount)
      }

      // Apply resource type filter
      if (this.filterResourceType) {
        filtered = filtered.filter(ip => ip.resource_type === this.filterResourceType)
      }

      // Apply region filter
      if (this.filterRegion) {
        filtered = filtered.filter(ip => ip.region === this.filterRegion)
      }

      // Apply sorting
      filtered.sort((a, b) => {
        let aVal = a[this.sortField] || ''
        let bVal = b[this.sortField] || ''
        
        // Special handling for IP addresses
        if (this.sortField === 'ip_address') {
          aVal = this.ipToNumber(aVal)
          bVal = this.ipToNumber(bVal)
        } else {
          aVal = aVal.toString().toLowerCase()
          bVal = bVal.toString().toLowerCase()
        }

        if (this.sortDirection === 'asc') {
          return aVal < bVal ? -1 : (aVal > bVal ? 1 : 0)
        } else {
          return aVal > bVal ? -1 : (aVal < bVal ? 1 : 0)
        }
      })

      return filtered
    }
  },
  methods: {
    async loadPublicIPs() {
      this.loading = true
      this.error = null
      
      try {
        const response = await fetch('/api/public-ips')
        
        if (!response.ok) {
          const errorData = await response.json()
          throw new Error(errorData.details || errorData.error || 'Failed to load public IPs')
        }
        
        this.publicIPs = await response.json()
      } catch (err) {
        console.error('Error loading public IPs:', err)
        this.error = err.message
      } finally {
        this.loading = false
      }
    },
    
    async refreshData() {
      if (this.canModify) {
        try {
          const response = await fetch('/api/cache/public-ips/invalidate', { method: 'POST', credentials: 'include' })
          if (!response.ok) {
            console.warn('Failed to invalidate cache')
          }
        } catch (error) {
          console.warn('Failed to invalidate public IPs cache:', error)
        }
      }
      await this.loadPublicIPs()
    },
    sortBy(field) {
      if (this.sortField === field) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortDirection = 'asc'
      }
    },
    ipToNumber(ip) {
      // Convert IP address to number for proper sorting
      return ip.split('.').reduce((acc, octet) => (acc << 8) + parseInt(octet), 0)
    },

    downloadIPsJSON() {
      try {
        const exportData = {
          exported_at: new Date().toISOString(),
          total_ips: this.filteredAndSortedIPs.length,
          unique_accounts: this.uniqueAccounts.length,
          unique_regions: this.uniqueRegions.length,
          filters: {
            search_query: this.searchQuery,
            account_filter: this.filterAccount,
            resource_type_filter: this.filterResourceType,
            region_filter: this.filterRegion
          },
          accounts: this.uniqueAccounts,
          public_ips: this.filteredAndSortedIPs
        }
        
        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr)
        
        const exportFileDefaultName = `aws-public-ips-${new Date().toISOString().split('T')[0]}.json`
        
        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (error) {
        console.error('Failed to download JSON:', error)
        alert('Failed to download JSON file')
      }
    }
  },
  async mounted() {
    await this.loadPublicIPs()
  }
}
</script>

<style scoped>
.public-ips {
  width: 100%;
  padding: 2rem;
}

h1 {
  color: var(--color-text-primary);
  margin-bottom: 0.5rem;
  font-size: 2rem;
}

.description {
  color: var(--color-text-secondary);
  margin-bottom: 2rem;
  font-size: 1.1rem;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem;
  text-align: center;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid var(--color-border);
  border-left-color: var(--color-btn-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.error {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  padding: 1.5rem;
  text-align: center;
}

.error h3 {
  color: var(--color-danger);
  margin-bottom: 0.5rem;
}

.error p {
  color: var(--color-danger);
  margin-bottom: 1rem;
}

.retry-btn {
  background: var(--color-btn-primary);
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  transition: all var(--transition-fast);
}

.retry-btn:hover {
  background: var(--color-btn-primary-hover);
}

.summary {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 2rem;
  margin-bottom: 2rem;
  padding: 1rem;
  background: var(--color-bg-secondary);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.summary-stats {
  display: flex;
  gap: 2rem;
}

.summary-actions {
  display: flex;
  gap: 1rem;
}

.summary-item {
  display: flex;
  flex-direction: column;
  text-align: center;
}

.summary-item .label {
  font-size: 0.9rem;
  color: var(--color-text-secondary);
  margin-bottom: 0.25rem;
}

.summary-item .value {
  font-size: 1.5rem;
  font-weight: bold;
  color: var(--color-text-primary);
}

.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.search-input,
.filter-select {
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 0.9rem;
  min-width: 200px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  transition: all var(--transition-fast);
}

.search-input:focus,
.filter-select:focus {
  outline: none;
  border-color: var(--color-btn-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-input {
  flex: 1;
  min-width: 300px;
}

.table-container {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.ip-table {
  width: 100%;
  min-width: 800px;
  border-collapse: collapse;
  background: var(--color-bg-primary);
}

.ip-table th {
  background: var(--color-bg-secondary);
  padding: 1rem;
  text-align: left;
  font-weight: 600;
  color: var(--color-text-primary);
  border-bottom: 2px solid var(--color-border);
  white-space: nowrap;
}

.ip-table th.sortable {
  cursor: pointer;
  user-select: none;
}

.ip-table th.sortable:hover {
  background: var(--color-bg-tertiary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-btn-primary);
}

.state-column {
  min-width: 120px;
  width: 120px;
}

.ip-table td {
  padding: 1rem;
  border-bottom: 1px solid var(--color-border);
  vertical-align: top;
  color: var(--color-text-primary);
}

.ip-row:hover {
  background: var(--color-bg-secondary);
}

.ip-address code,
.resource-id code {
  background: var(--color-bg-tertiary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.85rem;
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}

.account-info .account-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.account-info .account-id {
  font-size: 0.8rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.resource-badge {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 500;
  text-transform: uppercase;
}

.resource-ec2 {
  background: #dbeafe;
  color: #1e40af;
}

.resource-application {
  background: #dcfce7;
  color: #166534;
}

.resource-network {
  background: #fef3c7;
  color: #92400e;
}

.resource-gateway {
  background: #e0e7ff;
  color: #3730a3;
}

.resource-clb {
  background: #fef3c7;
  color: #92400e;
}

.resource-nat {
  background: #fce7f3;
  color: #be185d;
}

.state-badge {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 500;
  text-transform: lowercase;
}

.state-running,
.state-active {
  background: #dcfce7;
  color: #166534;
}

.state-stopped {
  background: #fecaca;
  color: #dc2626;
}

.state-pending {
  background: #fef3c7;
  color: #92400e;
}

.no-results {
  text-align: center;
  padding: 2rem;
  color: var(--color-text-secondary);
}

/* Button Styles */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  text-decoration: none;
  border: none;
  cursor: pointer;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
}

.btn-success {
  background: var(--color-btn-success);
  color: white;
  border: 1px solid var(--color-btn-success);
}

.btn-success:hover:not(:disabled) {
  background: var(--color-btn-success-hover);
  border-color: var(--color-btn-success-hover);
}

.btn-icon {
  width: 1rem;
  height: 1rem;
}

@media (max-width: 768px) {
  .public-ips {
    padding: 1rem;
  }
  
  .summary {
    flex-direction: column;
    gap: 1rem;
  }
  
  .filters {
    flex-direction: column;
  }
  
  .search-input,
  .filter-select {
    min-width: unset;
    width: 100%;
  }
  
  .ip-table {
    font-size: 0.8rem;
  }
  
  .ip-table th,
  .ip-table td {
    padding: 0.5rem;
  }
}
</style>