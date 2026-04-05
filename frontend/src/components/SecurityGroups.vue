<template>
  <div class="security-groups">
    <h1>Security Groups</h1>
    <p class="description">
      This page shows all security groups across all regions in all accessible AWS accounts,
      highlighting which security groups have ports open to the internet (0.0.0.0/0 or ::/0).
    </p>

    <div v-if="loading" class="loading">
      <div class="spinner"></div>
      <p>Loading security groups...</p>
      <div v-if="accountLoadingStatus.length > 0" class="account-loading-status">
        <div v-for="status in accountLoadingStatus" :key="status.accountId" class="account-status">
          <span class="account-name">{{ status.accountName }}</span>
          <span :class="['status', status.status]">
            <span v-if="status.status === 'loading'" class="mini-spinner"></span>
            {{ status.message }}
          </span>
        </div>
      </div>
    </div>

    <div v-else-if="error" class="error">
      <h3>Error loading security groups</h3>
      <p>{{ error }}</p>
      <button @click="loadSecurityGroups" class="retry-btn">Retry</button>
    </div>

    <div v-else class="content">
      <div class="summary">
        <div class="summary-stats">
          <div class="summary-item">
            <span class="label">Total Security Groups:</span>
            <span class="value">{{ securityGroups.length }}</span>
          </div>
          <div class="summary-item">
            <span class="label">Open to Internet:</span>
            <span class="value open-ports">{{ openToInternetCount }}</span>
          </div>
          <div class="summary-item">
            <span class="label">Unused:</span>
            <span class="value unused">{{ unusedCount }}</span>
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
          <button @click="downloadSecurityGroupsJSON" class="btn btn-success" :disabled="loading || securityGroups.length === 0">
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
          placeholder="Search by group ID, name, account, region, or description..."
          class="search-input"
        />
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
        <select v-model="filterOpenPorts" class="filter-select">
          <option value="">All Security Groups</option>
          <option value="open">Open to Internet</option>
          <option value="closed">Not Open to Internet</option>
        </select>
        <select v-model="filterUsage" class="filter-select">
          <option value="">All Security Groups</option>
          <option value="used">Used</option>
          <option value="unused">Unused</option>
        </select>
      </div>

      <div class="table-container">
        <table class="sg-table">
          <thead>
            <tr>
              <th @click="sortBy('group_name')" class="sortable">
                Group Name
                <span v-if="sortField === 'group_name'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th>Group ID</th>
              <th>Description</th>
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
              <th>VPC ID</th>
              <th class="usage-column">Usage</th>
              <th class="internet-access-column">Internet Access</th>
              <th class="actions-column">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="sg in filteredAndSortedSGs" :key="`${sg.group_id}-${sg.region}`" class="sg-row" @click="navigateToSecurityGroup(sg)">
              <td class="group-name">
                <div class="group-info">
                  <div class="name">{{ sg.group_name }}</div>
                  <div v-if="sg.is_default" class="default-badge">Default</div>
                </div>
              </td>
              <td class="group-id">
                <code>{{ sg.group_id }}</code>
              </td>
              <td class="description">
                <div class="description-text" :title="sg.description">
                  {{ sg.description || '-' }}
                </div>
              </td>
              <td class="account">
                <div class="account-info">
                  <div class="account-name">{{ sg.account_name }}</div>
                  <div class="account-id">{{ sg.account_id }}</div>
                </div>
              </td>
              <td class="region">{{ sg.region }}</td>
              <td class="vpc-id">
                <code v-if="sg.vpc_id">{{ sg.vpc_id }}</code>
                <span v-else>EC2-Classic</span>
              </td>
              <td class="usage usage-column">
                <div class="usage-info">
                  <span :class="`usage-badge ${sg.is_unused ? 'unused' : 'used'}`">
                    {{ sg.is_unused ? 'Unused' : 'Used' }}
                  </span>
                  <div v-if="!sg.is_unused && sg.usage_info" class="usage-details">
                    <span v-if="sg.usage_info.total_attachments > 0" class="attachment-count">
                      {{ sg.usage_info.total_attachments }} attachment{{ sg.usage_info.total_attachments > 1 ? 's' : '' }}
                    </span>
                  </div>
                </div>
              </td>
              <td class="internet-access internet-access-column">
                <span :class="`internet-badge ${sg.has_open_ports ? 'open' : 'closed'}`">
                  {{ sg.has_open_ports ? 'Open' : 'Closed' }}
                </span>
              </td>
              <td class="actions actions-column" @click.stop>
                <div class="action-buttons">
                  <router-link :to="`/aws/security-groups/${sg.account_id}/${sg.region}/${sg.group_id}`" class="btn btn-primary btn-sm">
                    <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M12,9A3,3 0 0,0 9,12A3,3 0 0,0 12,15A3,3 0 0,0 15,12A3,3 0 0,0 12,9M12,17A5,5 0 0,1 7,12A5,5 0 0,1 12,7A5,5 0 0,1 17,12A5,5 0 0,1 12,17M12,4.5C7,4.5 2.73,7.61 1,12C2.73,16.39 7,19.5 12,19.5C17,19.5 21.27,16.39 23,12C21.27,7.61 17,4.5 12,4.5Z"/>
                    </svg>
                    View
                  </router-link>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="filteredAndSortedSGs.length === 0" class="no-results">
        <p>No security groups found matching your filters.</p>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SecurityGroups',
  data() {
    return {
      securityGroups: [],
      loading: true,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterRegion: '',
      filterOpenPorts: '',
      filterUsage: '',
      sortField: 'group_name',
      sortDirection: 'asc',
      accountLoadingStatus: []
    }
  },
  computed: {
    uniqueAccounts() {
      const accountMap = new Map()
      this.securityGroups.forEach(sg => {
        if (!accountMap.has(sg.account_id)) {
          accountMap.set(sg.account_id, {
            id: sg.account_id,
            name: sg.account_name
          })
        }
      })
      return Array.from(accountMap.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    uniqueRegions() {
      const regions = [...new Set(this.securityGroups.map(sg => sg.region))]
      return regions.sort()
    },
    openToInternetCount() {
      return this.securityGroups.filter(sg => sg.has_open_ports).length
    },
    unusedCount() {
      return this.securityGroups.filter(sg => sg.is_unused).length
    },
    filteredAndSortedSGs() {
      let filtered = this.securityGroups

      // Apply search filter
      if (this.searchQuery.trim()) {
        const query = this.searchQuery.toLowerCase()
        filtered = filtered.filter(sg =>
          sg.group_name.toLowerCase().includes(query) ||
          sg.group_id.toLowerCase().includes(query) ||
          sg.account_name.toLowerCase().includes(query) ||
          sg.account_id.toLowerCase().includes(query) ||
          sg.region.toLowerCase().includes(query) ||
          (sg.description && sg.description.toLowerCase().includes(query)) ||
          (sg.vpc_id && sg.vpc_id.toLowerCase().includes(query))
        )
      }

      // Apply account filter
      if (this.filterAccount) {
        filtered = filtered.filter(sg => sg.account_id === this.filterAccount)
      }

      // Apply region filter
      if (this.filterRegion) {
        filtered = filtered.filter(sg => sg.region === this.filterRegion)
      }

      // Apply open ports filter
      if (this.filterOpenPorts === 'open') {
        filtered = filtered.filter(sg => sg.has_open_ports)
      } else if (this.filterOpenPorts === 'closed') {
        filtered = filtered.filter(sg => !sg.has_open_ports)
      }

      // Apply usage filter
      if (this.filterUsage === 'used') {
        filtered = filtered.filter(sg => !sg.is_unused)
      } else if (this.filterUsage === 'unused') {
        filtered = filtered.filter(sg => sg.is_unused)
      }

      // Apply sorting
      filtered.sort((a, b) => {
        let aVal = a[this.sortField] || ''
        let bVal = b[this.sortField] || ''

        aVal = aVal.toString().toLowerCase()
        bVal = bVal.toString().toLowerCase()

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
    async loadSecurityGroups() {
      this.loading = true
      this.error = null
      this.accountLoadingStatus = []
      this.securityGroups = []

      try {
        // First, load accounts
        console.log('Loading accounts...')
        const accountsResponse = await fetch('/api/accounts')
        if (!accountsResponse.ok) {
          const errorData = await accountsResponse.json()
          throw new Error(errorData.details || errorData.error || 'Failed to load accounts')
        }

        const accounts = await accountsResponse.json()
        const accessibleAccounts = accounts.filter(account => account.accessible)

        if (accessibleAccounts.length === 0) {
          this.loading = false
          return
        }

        // Initialize loading status for each account
        this.accountLoadingStatus = accessibleAccounts.map(account => ({
          accountId: account.id,
          accountName: account.name,
          status: 'loading',
          message: 'Loading...'
        }))

        // Load security groups for all accounts in parallel
        console.log(`Loading security groups for ${accessibleAccounts.length} accounts in parallel...`)
        const startTime = performance.now()

        const promises = accessibleAccounts.map(async (account) => {
          try {
            const response = await fetch(`/api/accounts/${account.id}/security-groups`)

            if (!response.ok) {
              const errorData = await response.json()
              this.updateAccountStatus(account.id, 'error', `Error: ${errorData.error || 'Failed to load'}`)
              return []
            }

            const sgs = await response.json()
            this.updateAccountStatus(account.id, 'completed', `Loaded ${sgs.length} security groups`)
            return sgs
          } catch (error) {
            console.error(`Error loading security groups for account ${account.id}:`, error)
            this.updateAccountStatus(account.id, 'error', `Error: ${error.message}`)
            return []
          }
        })

        // Wait for all requests to complete
        const results = await Promise.all(promises)

        // Combine all results
        this.securityGroups = results.flat()

        const endTime = performance.now()
        console.log(`Loaded ${this.securityGroups.length} security groups from ${accessibleAccounts.length} accounts in ${Math.round(endTime - startTime)}ms`)

      } catch (err) {
        console.error('Error loading security groups:', err)
        this.error = err.message
      } finally {
        this.loading = false
      }
    },

    updateAccountStatus(accountId, status, message) {
      const statusIndex = this.accountLoadingStatus.findIndex(s => s.accountId === accountId)
      if (statusIndex >= 0) {
        this.accountLoadingStatus[statusIndex].status = status
        this.accountLoadingStatus[statusIndex].message = message
      }
    },

    async refreshData() {
      if (this.canModify) {
        try {
          const response = await fetch('/api/cache/security-groups/invalidate', { method: 'POST', credentials: 'include' })
          if (!response.ok) {
            console.warn('Failed to invalidate cache')
          }
        } catch (error) {
          console.warn('Failed to invalidate security groups cache:', error)
        }
      }
      await this.loadSecurityGroups()
    },
    sortBy(field) {
      if (this.sortField === field) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortDirection = 'asc'
      }
    },

    downloadSecurityGroupsJSON() {
      try {
        const exportData = {
          exported_at: new Date().toISOString(),
          total_security_groups: this.filteredAndSortedSGs.length,
          open_to_internet_count: this.openToInternetCount,
          unused_count: this.unusedCount,
          unique_accounts: this.uniqueAccounts.length,
          unique_regions: this.uniqueRegions.length,
          filters: {
            search_query: this.searchQuery,
            account_filter: this.filterAccount,
            region_filter: this.filterRegion,
            open_ports_filter: this.filterOpenPorts,
            usage_filter: this.filterUsage
          },
          accounts: this.uniqueAccounts,
          security_groups: this.filteredAndSortedSGs
        }

        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr)

        const exportFileDefaultName = `aws-security-groups-${new Date().toISOString().split('T')[0]}.json`

        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (error) {
        console.error('Failed to download JSON:', error)
        alert('Failed to download JSON file')
      }
    },

    async deleteSecurityGroup(sg) {
      const confirmMessage = `Are you sure you want to DELETE the security group "${sg.group_name}" (${sg.group_id})?

This will:
1. Permanently delete the security group
2. This action cannot be undone

Security Group Details:
- Account: ${sg.account_name} (${sg.account_id})
- Region: ${sg.region}
- VPC: ${sg.vpc_id || 'EC2-Classic'}
- Status: ${sg.is_unused ? 'Unused' : 'In Use'}`

      if (!confirm(confirmMessage)) return

      this.loading = true

      try {
        const response = await fetch(`/api/accounts/${sg.account_id}/regions/${sg.region}/security-groups/${sg.group_id}`, {
          method: 'DELETE'
        })

        if (!response.ok) {
          const errorData = await response.json()
          throw new Error(errorData.error || 'Failed to delete security group')
        }

        const result = await response.json()
        alert(result.message || `Security group ${sg.group_id} deleted successfully`)

        // Refresh the security groups list
        await this.loadSecurityGroups()
      } catch (error) {
        console.error('Failed to delete security group:', error)
        alert(`Failed to delete security group: ${error.message}`)
      } finally {
        this.loading = false
      }
    },

    navigateToSecurityGroup(sg) {
      console.log('Navigating to security group:', sg.account_id, sg.region, sg.group_id)
      const path = `/security-groups/${sg.account_id}/${sg.region}/${sg.group_id}`
      console.log('Navigation path:', path)
      this.$router.push(path)
    }
  },
  async mounted() {
    await this.loadSecurityGroups()
  }
}
</script>

<style scoped>
.security-groups {
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

.account-loading-status {
  margin-top: 1.5rem;
  max-width: 600px;
  margin-left: auto;
  margin-right: auto;
}

.account-status {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 1rem;
  margin: 0.25rem 0;
  background: var(--color-bg-secondary);
  border-radius: 6px;
  border: 1px solid var(--color-border);
}

.account-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
}

.status.loading {
  color: var(--color-btn-primary);
}

.status.completed {
  color: var(--color-btn-success);
}

.status.error {
  color: var(--color-danger);
}

.mini-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top: 2px solid var(--color-btn-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
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

.summary-item .value.open-ports {
  color: var(--color-danger);
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

.sg-table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
  background: var(--color-bg-primary);
}

.sg-table th {
  background: var(--color-bg-secondary);
  padding: 1rem;
  text-align: left;
  font-weight: 600;
  color: var(--color-text-primary);
  border-bottom: 2px solid var(--color-border);
  white-space: nowrap;
}

.sg-table th.sortable {
  cursor: pointer;
  user-select: none;
}

.sg-table th.sortable:hover {
  background: var(--color-bg-tertiary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-btn-primary);
}

.internet-access-column {
  min-width: 120px;
  width: 120px;
}

.usage-column {
  min-width: 120px;
  width: 120px;
}


.actions-column {
  min-width: 80px;
  width: 80px;
  text-align: center;
}

.sg-table td {
  padding: 1rem;
  border-bottom: 1px solid var(--color-border);
  vertical-align: top;
  color: var(--color-text-primary);
}

.sg-row {
  cursor: pointer;
  transition: background-color var(--transition-fast);
}

.sg-row:hover {
  background: var(--color-bg-tertiary) !important;
}

.group-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.group-info .name {
  font-weight: 500;
}

.default-badge {
  background: var(--color-btn-warning);
  color: white;
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-size: 0.7rem;
  font-weight: 500;
  text-transform: uppercase;
}

.group-id code {
  background: var(--color-bg-tertiary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.85rem;
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}

.description-text {
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.internet-badge {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 500;
  text-transform: uppercase;
  border: 1px solid transparent;
}

.internet-badge.open {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-danger);
  border: 1px solid rgba(239, 68, 68, 0.2);
}

.internet-badge.closed {
  background: rgba(16, 185, 129, 0.1);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.usage-badge {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 500;
  text-transform: uppercase;
  border: 1px solid transparent;
}

.usage-badge.used {
  background: rgba(16, 185, 129, 0.1);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.usage-badge.unused {
  background: rgba(245, 158, 11, 0.1);
  color: var(--color-warning);
  border: 1px solid rgba(245, 158, 11, 0.2);
}

.usage-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.usage-details {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.attachment-count {
  font-style: italic;
}

.no-action {
  font-size: 0.8rem;
  color: var(--color-text-secondary);
  font-style: italic;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.value.unused {
  color: var(--color-warning);
  font-weight: 600;
}


.action-buttons {
  display: flex;
  gap: 0.5rem;
  justify-content: center;
  align-items: center;
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
  .security-groups {
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

  .sg-table {
    font-size: 0.8rem;
  }

  .sg-table th,
  .sg-table td {
    padding: 0.5rem;
  }

  .description-text {
    max-width: 150px;
  }
}
</style>