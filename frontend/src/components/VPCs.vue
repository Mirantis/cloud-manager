<template>
  <div class="vpcs-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M10,4H4C2.89,4 2,4.89 2,6V18A2,2 0 0,0 4,20H20A2,2 0 0,0 22,18V8C22,6.89 21.1,6 20,6H12L10,4Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>VPCs</h1>
            <p>{{ vpcs.length }} VPCs across {{ uniqueAccounts.length }} accounts</p>
          </div>
        </div>
        <div class="header-actions">
          <button 
            v-if="canModify"
            @click="deleteAllEmptyVPCs" 
            class="btn btn-danger" 
            :disabled="loading || !filterAccount || emptyVPCsCount === 0"
            :title="!filterAccount ? 'Select an account first to enable bulk delete' : (emptyVPCsCount === 0 ? 'No empty VPCs in this account' : `Delete ${emptyVPCsCount} empty VPC(s) in ${selectedAccountName}`)"
          >
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
            </svg>
            Delete All Empty ({{ emptyVPCsCount }})
          </button>
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || vpcs.length === 0">
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
      <p>Loading VPCs...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load VPCs</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Summary Stats -->
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ vpcs.length }}</div>
          <div class="stat-label">Total VPCs</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ defaultVPCs }}</div>
          <div class="stat-label">Default VPCs</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ vpcsWithIGW }}</div>
          <div class="stat-label">With Internet Gateway</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ totalSubnets }}</div>
          <div class="stat-label">Total Subnets</div>
        </div>
        <div class="stat-card stat-card-empty">
          <div class="stat-value">{{ emptyVPCsCount }}</div>
          <div class="stat-label">Empty VPCs</div>
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
            placeholder="Search by VPC ID, name, account, region, or CIDR..."
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
        <select v-model="filterType" class="filter-select">
          <option value="">All Types</option>
          <option value="default">Default VPCs</option>
          <option value="custom">Custom VPCs</option>
        </select>
      </div>

      <!-- VPCs Table -->
      <div class="table-container" v-if="filteredVPCs.length > 0">
        <table class="data-table">
          <thead>
            <tr>
              <th @click="sortBy('name')" class="sortable">
                Name
                <span v-if="sortColumn === 'name'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('vpc_id')" class="sortable">
                VPC ID
                <span v-if="sortColumn === 'vpc_id'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('account_name')" class="sortable">
                Account
                <span v-if="sortColumn === 'account_name'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('region')" class="sortable">
                Region
                <span v-if="sortColumn === 'region'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('cidr_block')" class="sortable">
                CIDR Block
                <span v-if="sortColumn === 'cidr_block'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('state')" class="sortable">
                State
                <span v-if="sortColumn === 'state'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('subnet_count')" class="sortable">
                Subnets
                <span v-if="sortColumn === 'subnet_count'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('nat_gateway_count')" class="sortable">
                NAT GWs
                <span v-if="sortColumn === 'nat_gateway_count'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Features</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="vpc in filteredVPCs" :key="vpc.vpc_id + vpc.account_id">
              <td class="name-cell">
                <span class="vpc-name">{{ vpc.name || '-' }}</span>
                <span v-if="vpc.is_default" class="badge badge-default">Default</span>
              </td>
              <td class="mono">{{ vpc.vpc_id }}</td>
              <td>
                <span class="account-name">{{ vpc.account_name }}</span>
                <span class="account-id">{{ vpc.account_id }}</span>
              </td>
              <td class="mono">{{ vpc.region }}</td>
              <td class="mono">{{ vpc.cidr_block }}</td>
              <td>
                <span :class="['status-badge', getStateClass(vpc.state)]">
                  {{ vpc.state }}
                </span>
              </td>
              <td class="center">{{ vpc.subnet_count }}</td>
              <td class="center">{{ vpc.nat_gateway_count }}</td>
              <td>
                <div class="features">
                  <span v-if="vpc.internet_gateway" class="feature-badge feature-igw" title="Has Internet Gateway">
                    IGW
                  </span>
                  <span v-if="vpc.has_flow_logs" class="feature-badge feature-logs" title="Flow Logs Enabled">
                    Logs
                  </span>
                  <span v-if="!vpc.internet_gateway && !vpc.has_flow_logs">-</span>
                </div>
              </td>
              <td>
                <div class="action-buttons">
                  <button
                    v-if="canModify && !vpc.is_default && vpc.subnet_count === 0 && vpc.nat_gateway_count === 0"
                    @click="deleteVPC(vpc)"
                    class="action-btn action-btn-delete"
                    :disabled="loading"
                    title="Delete empty VPC"
                  >
                    <svg viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                    </svg>
                    Delete
                  </button>
                  <span v-else class="no-action" :title="getNoDeleteReason(vpc)">-</span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Empty State -->
      <div v-else class="empty-state">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M10,4H4C2.89,4 2,4.89 2,6V18A2,2 0 0,0 4,20H20A2,2 0 0,0 22,18V8C22,6.89 21.1,6 20,6H12L10,4Z"/>
        </svg>
        <h3>No VPCs Found</h3>
        <p v-if="searchQuery || filterAccount || filterRegion || filterType">
          No VPCs match your current filters. Try adjusting your search criteria.
        </p>
        <p v-else>No VPCs are available in your accounts.</p>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'VPCs',
  data() {
    return {
      vpcs: [],
      loading: false,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterRegion: '',
      filterType: '',
      sortColumn: 'name',
      sortDirection: 'asc'
    }
  },
  computed: {
    uniqueAccounts() {
      const accounts = new Map()
      this.vpcs.forEach(vpc => {
        if (!accounts.has(vpc.account_id)) {
          accounts.set(vpc.account_id, {
            id: vpc.account_id,
            name: vpc.account_name
          })
        }
      })
      return Array.from(accounts.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    uniqueRegions() {
      return [...new Set(this.vpcs.map(v => v.region))].sort()
    },
    selectedAccountName() {
      if (!this.filterAccount) return ''
      const account = this.uniqueAccounts.find(a => a.id === this.filterAccount)
      return account ? account.name : this.filterAccount
    },
    defaultVPCs() {
      return this.vpcs.filter(v => v.is_default).length
    },
    emptyVPCsCount() {
      // Count empty VPCs (not default, no subnets, no NAT gateways), optionally filtered by account
      let vpcsToCheck = this.vpcs
      if (this.filterAccount && this.filterAccount.trim() !== '') {
        const filterAccountId = String(this.filterAccount).trim()
        vpcsToCheck = vpcsToCheck.filter(vpc => {
          const vpcAccountId = vpc.account_id ? String(vpc.account_id).trim() : ''
          return vpcAccountId === filterAccountId
        })
      }
      return vpcsToCheck.filter(vpc => 
        !vpc.is_default && 
        (vpc.subnet_count || 0) === 0 && 
        (vpc.nat_gateway_count || 0) === 0
      ).length
    },
    emptyVPCs() {
      // Get empty VPCs (not default, no subnets, no NAT gateways), optionally filtered by account
      let vpcsToCheck = this.vpcs
      if (this.filterAccount && this.filterAccount.trim() !== '') {
        const filterAccountId = String(this.filterAccount).trim()
        vpcsToCheck = vpcsToCheck.filter(vpc => {
          const vpcAccountId = vpc.account_id ? String(vpc.account_id).trim() : ''
          return vpcAccountId === filterAccountId
        })
      }
      return vpcsToCheck.filter(vpc => 
        !vpc.is_default && 
        (vpc.subnet_count || 0) === 0 && 
        (vpc.nat_gateway_count || 0) === 0
      )
    },
    vpcsWithIGW() {
      return this.vpcs.filter(v => v.internet_gateway).length
    },
    totalSubnets() {
      return this.vpcs.reduce((sum, v) => sum + (v.subnet_count || 0), 0)
    },
    filteredVPCs() {
      let result = [...this.vpcs]

      // Search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        result = result.filter(vpc =>
          (vpc.vpc_id || '').toLowerCase().includes(query) ||
          (vpc.name || '').toLowerCase().includes(query) ||
          (vpc.account_name || '').toLowerCase().includes(query) ||
          (vpc.account_id || '').toLowerCase().includes(query) ||
          (vpc.region || '').toLowerCase().includes(query) ||
          (vpc.cidr_block || '').toLowerCase().includes(query)
        )
      }

      // Account filter
      if (this.filterAccount) {
        result = result.filter(vpc => vpc.account_id === this.filterAccount)
      }

      // Region filter
      if (this.filterRegion) {
        result = result.filter(vpc => vpc.region === this.filterRegion)
      }

      // Type filter
      if (this.filterType === 'default') {
        result = result.filter(vpc => vpc.is_default)
      } else if (this.filterType === 'custom') {
        result = result.filter(vpc => !vpc.is_default)
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
        const response = await axios.get('/api/vpcs')
        this.vpcs = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load VPCs'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      if (this.canModify) {
        try {
          await axios.post('/api/cache/vpcs/invalidate')
        } catch (error) {
          console.warn('Failed to invalidate cache:', error)
        }
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
    getStateClass(state) {
      switch (state?.toLowerCase()) {
        case 'available': return 'status-available'
        case 'pending': return 'status-pending'
        default: return 'status-default'
      }
    },
    getNoDeleteReason(vpc) {
      if (vpc.is_default) return 'Cannot delete default VPC'
      if (vpc.subnet_count > 0) return `VPC has ${vpc.subnet_count} subnet(s)`
      if (vpc.nat_gateway_count > 0) return `VPC has ${vpc.nat_gateway_count} NAT Gateway(s)`
      return 'VPC has dependencies'
    },
    async deleteVPC(vpc) {
      const confirmed = confirm(
        `Are you sure you want to delete VPC "${vpc.name || vpc.vpc_id}"?\n\n` +
        `Account: ${vpc.account_name}\n` +
        `Region: ${vpc.region}\n` +
        `CIDR: ${vpc.cidr_block}\n\n` +
        `WARNING: This action cannot be undone.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        await axios.delete(`/api/accounts/${vpc.account_id}/regions/${vpc.region}/vpcs/${vpc.vpc_id}`)
        await this.loadData()
        alert(`VPC ${vpc.vpc_id} deleted successfully`)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete VPC'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete VPC:', err)
      } finally {
        this.loading = false
      }
    },
    async deleteAllEmptyVPCs() {
      const emptyVPCs = this.emptyVPCs
      if (emptyVPCs.length === 0) {
        alert('No empty VPCs to delete')
        return
      }

      const accountFilter = this.filterAccount
      const accountName = accountFilter 
        ? this.vpcs.find(v => v.account_id === accountFilter)?.account_name || accountFilter
        : 'all accounts'

      const vpcsList = emptyVPCs.slice(0, 10).map(vpc => 
        `  - ${vpc.name || vpc.vpc_id} (${vpc.region}, CIDR: ${vpc.cidr_block})`
      ).join('\n')
      const moreText = emptyVPCs.length > 10 ? `\n  ... and ${emptyVPCs.length - 10} more` : ''

      const confirmed = confirm(
        `Are you sure you want to delete ALL ${emptyVPCs.length} empty VPC(s)?\n\n` +
        `Account: ${accountName}\n\n` +
        `VPCs to delete:\n${vpcsList}${moreText}\n\n` +
        `WARNING: This action cannot be undone!\n` +
        `This will permanently delete all empty VPCs${accountFilter ? ' in this account' : ' across all accounts'}.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        let successCount = 0
        let failCount = 0
        const errors = []

        // Delete VPCs one by one
        for (const vpc of emptyVPCs) {
          try {
            const url = `/api/accounts/${vpc.account_id}/regions/${vpc.region}/vpcs/${vpc.vpc_id}`
            await axios.delete(url)
            successCount++
            
            // Small delay to avoid rate limiting
            await new Promise(resolve => setTimeout(resolve, 200))
          } catch (err) {
            failCount++
            const errorMsg = err.response?.data?.error || err.message || 'Unknown error'
            errors.push(`${vpc.vpc_id}: ${errorMsg}`)
            console.error(`Failed to delete VPC ${vpc.vpc_id}:`, err)
          }
        }

        // Reload data
        await this.loadData()

        // Show results
        let message = `Deleted ${successCount} VPC(s) successfully`
        if (failCount > 0) {
          message += `\n\nFailed to delete ${failCount} VPC(s):\n${errors.slice(0, 5).join('\n')}`
          if (errors.length > 5) {
            message += `\n... and ${errors.length - 5} more errors (check console)`
          }
        }
        alert(message)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete VPCs'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete VPCs:', err)
      } finally {
        this.loading = false
      }
    },
    downloadJSON() {
      const data = JSON.stringify(this.filteredVPCs, null, 2)
      const blob = new Blob([data], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `vpcs-${new Date().toISOString().slice(0, 10)}.json`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    }
  }
}
</script>

<style scoped>
.vpcs-container {
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
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
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

.btn-danger {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: white;
}

.btn-danger:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
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
  color: var(--color-primary);
  margin-bottom: 0.25rem;
}

.stat-label {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.stat-card-empty {
  border: 2px solid rgba(239, 68, 68, 0.3);
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.05) 0%, rgba(239, 68, 68, 0.02) 100%);
}

.stat-card-empty .stat-value {
  color: #ef4444;
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
  min-width: 800px;
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
  font-size: 0.9rem;
}

.center {
  text-align: center;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.vpc-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.badge-default {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.account-name {
  display: block;
  font-weight: 500;
  color: var(--color-text-primary);
}

.account-id {
  display: block;
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.status-available {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.status-pending {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.status-default {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.features {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.feature-badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.feature-igw {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.feature-logs {
  background: rgba(168, 85, 247, 0.1);
  color: #a855f7;
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
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

.no-action {
  color: var(--color-text-secondary);
  font-size: 0.85rem;
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

.empty-state h3 {
  margin: 0 0 0.5rem 0;
  color: var(--color-text-primary);
}

.empty-state p {
  color: var(--color-text-secondary);
  margin: 0;
}
</style>
