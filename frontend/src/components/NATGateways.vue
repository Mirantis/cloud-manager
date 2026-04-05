<template>
  <div class="nat-gateways-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4M12,6A6,6 0 0,0 6,12A6,6 0 0,0 12,18A6,6 0 0,0 18,12A6,6 0 0,0 12,6M12,8A4,4 0 0,1 16,12A4,4 0 0,1 12,16A4,4 0 0,1 8,12A4,4 0 0,1 12,8Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>NAT Gateways</h1>
            <p>{{ natGateways.length }} NAT Gateways across {{ uniqueAccounts.length }} accounts</p>
          </div>
        </div>
        <div class="header-actions">
          <button 
            v-if="canModify"
            @click="deleteAllOldNATGateways" 
            class="btn btn-danger" 
            :disabled="loading || !filterAccount || oldNATGatewaysCount === 0"
            :title="!filterAccount ? 'Select an account first to enable bulk delete' : (oldNATGatewaysCount === 0 ? 'No NAT Gateways older than 6 months in this account' : `Delete ${oldNATGatewaysCount} NAT Gateway(s) older than 6 months in ${selectedAccountName}`)"
          >
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
            </svg>
            Delete Old ({{ oldNATGatewaysCount }})
          </button>
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || natGateways.length === 0">
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
      <p>Loading NAT Gateways...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load NAT Gateways</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Summary Stats -->
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ natGateways.length }}</div>
          <div class="stat-label">Total NAT Gateways</div>
        </div>
        <div class="stat-card stat-card-available">
          <div class="stat-value">{{ availableNATs }}</div>
          <div class="stat-label">Available</div>
        </div>
        <div class="stat-card stat-card-old">
          <div class="stat-value">{{ oldNATGatewaysCount }}</div>
          <div class="stat-label">Old (6+ months)</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ publicNATs }}</div>
          <div class="stat-label">Public</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ privateNATs }}</div>
          <div class="stat-label">Private</div>
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
            placeholder="Search by NAT Gateway ID, name, account, region, or IP..."
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
        <select v-model="filterState" class="filter-select">
          <option value="">All States</option>
          <option value="available">Available</option>
          <option value="pending">Pending</option>
          <option value="failed">Failed</option>
          <option value="deleting">Deleting</option>
        </select>
        <select v-model="filterType" class="filter-select">
          <option value="">All Types</option>
          <option value="public">Public</option>
          <option value="private">Private</option>
        </select>
      </div>

      <!-- NAT Gateways Table -->
      <div class="table-container" v-if="filteredNATGateways.length > 0">
        <table class="data-table">
          <thead>
            <tr>
              <th @click="sortBy('name')" class="sortable">
                Name
                <span v-if="sortColumn === 'name'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('nat_gateway_id')" class="sortable">
                NAT Gateway ID
                <span v-if="sortColumn === 'nat_gateway_id'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('account_name')" class="sortable">
                Account
                <span v-if="sortColumn === 'account_name'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('region')" class="sortable">
                Region
                <span v-if="sortColumn === 'region'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('state')" class="sortable">
                State
                <span v-if="sortColumn === 'state'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('connectivity_type')" class="sortable">
                Type
                <span v-if="sortColumn === 'connectivity_type'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Public IP</th>
              <th>Private IP</th>
              <th>VPC ID</th>
              <th @click="sortBy('create_time')" class="sortable">
                Created
                <span v-if="sortColumn === 'create_time'" class="sort-indicator">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="nat in filteredNATGateways" :key="nat.nat_gateway_id + nat.account_id">
              <td class="name-cell">
                <span class="nat-name">{{ nat.name || '-' }}</span>
              </td>
              <td class="mono">{{ nat.nat_gateway_id }}</td>
              <td>
                <span class="account-name">{{ nat.account_name }}</span>
                <span class="account-id">{{ nat.account_id }}</span>
              </td>
              <td class="mono">{{ nat.region }}</td>
              <td>
                <span :class="['status-badge', getStateClass(nat.state)]">
                  {{ nat.state }}
                </span>
              </td>
              <td>
                <span :class="['type-badge', nat.connectivity_type === 'public' ? 'type-public' : 'type-private']">
                  {{ nat.connectivity_type || 'public' }}
                </span>
              </td>
              <td class="mono">{{ nat.public_ip || '-' }}</td>
              <td class="mono">{{ nat.private_ip || '-' }}</td>
              <td class="mono">{{ nat.vpc_id }}</td>
              <td>{{ formatDate(nat.create_time) }}</td>
              <td>
                <div class="action-buttons">
                  <button
                    v-if="canModify"
                    @click="deleteNATGateway(nat)"
                    class="action-btn action-btn-delete"
                    :disabled="loading || nat.state === 'deleting'"
                    title="Delete NAT Gateway"
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

      <!-- Empty State -->
      <div v-else class="empty-state">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4Z"/>
        </svg>
        <h3>No NAT Gateways Found</h3>
        <p v-if="searchQuery || filterAccount || filterRegion || filterState || filterType">
          No NAT Gateways match your current filters. Try adjusting your search criteria.
        </p>
        <p v-else>No NAT Gateways are available in your accounts.</p>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'NATGateways',
  data() {
    return {
      natGateways: [],
      loading: false,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterRegion: '',
      filterState: '',
      filterType: '',
      sortColumn: 'name',
      sortDirection: 'asc'
    }
  },
  computed: {
    uniqueAccounts() {
      const accounts = new Map()
      this.natGateways.forEach(nat => {
        if (!accounts.has(nat.account_id)) {
          accounts.set(nat.account_id, {
            id: nat.account_id,
            name: nat.account_name
          })
        }
      })
      return Array.from(accounts.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    uniqueRegions() {
      return [...new Set(this.natGateways.map(n => n.region))].sort()
    },
    availableNATs() {
      return this.natGateways.filter(n => n.state === 'available').length
    },
    publicNATs() {
      return this.natGateways.filter(n => n.connectivity_type === 'public' || !n.connectivity_type).length
    },
    privateNATs() {
      return this.natGateways.filter(n => n.connectivity_type === 'private').length
    },
    selectedAccountName() {
      if (!this.filterAccount) return ''
      const account = this.uniqueAccounts.find(a => a.id === this.filterAccount)
      return account ? account.name : this.filterAccount
    },
    oldNATGatewaysCount() {
      // Count NAT Gateways older than 6 months, optionally filtered by account
      let natsToCheck = this.natGateways
      if (this.filterAccount && this.filterAccount.trim() !== '') {
        const filterAccountId = String(this.filterAccount).trim()
        natsToCheck = natsToCheck.filter(nat => {
          const natAccountId = nat.account_id ? String(nat.account_id).trim() : ''
          return natAccountId === filterAccountId
        })
      }
      const sixMonthsAgo = new Date()
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
      return natsToCheck.filter(nat => {
        if (!nat.create_time) return false
        const createdDate = new Date(nat.create_time)
        return createdDate < sixMonthsAgo
      }).length
    },
    oldNATGateways() {
      // Get NAT Gateways older than 6 months, optionally filtered by account
      let natsToCheck = this.natGateways
      if (this.filterAccount && this.filterAccount.trim() !== '') {
        const filterAccountId = String(this.filterAccount).trim()
        natsToCheck = natsToCheck.filter(nat => {
          const natAccountId = nat.account_id ? String(nat.account_id).trim() : ''
          return natAccountId === filterAccountId
        })
      }
      const sixMonthsAgo = new Date()
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
      return natsToCheck.filter(nat => {
        if (!nat.create_time) return false
        const createdDate = new Date(nat.create_time)
        return createdDate < sixMonthsAgo
      })
    },
    filteredNATGateways() {
      let result = [...this.natGateways]

      // Search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        result = result.filter(nat =>
          (nat.nat_gateway_id || '').toLowerCase().includes(query) ||
          (nat.name || '').toLowerCase().includes(query) ||
          (nat.account_name || '').toLowerCase().includes(query) ||
          (nat.account_id || '').toLowerCase().includes(query) ||
          (nat.region || '').toLowerCase().includes(query) ||
          (nat.public_ip || '').toLowerCase().includes(query) ||
          (nat.private_ip || '').toLowerCase().includes(query) ||
          (nat.vpc_id || '').toLowerCase().includes(query)
        )
      }

      // Account filter
      if (this.filterAccount) {
        result = result.filter(nat => nat.account_id === this.filterAccount)
      }

      // Region filter
      if (this.filterRegion) {
        result = result.filter(nat => nat.region === this.filterRegion)
      }

      // State filter
      if (this.filterState) {
        result = result.filter(nat => nat.state === this.filterState)
      }

      // Type filter
      if (this.filterType) {
        result = result.filter(nat => {
          const type = nat.connectivity_type || 'public'
          return type === this.filterType
        })
      }

      // Sorting
      result.sort((a, b) => {
        let aVal = a[this.sortColumn] ?? ''
        let bVal = b[this.sortColumn] ?? ''

        // Handle date sorting
        if (this.sortColumn === 'create_time') {
          aVal = aVal ? new Date(aVal).getTime() : 0
          bVal = bVal ? new Date(bVal).getTime() : 0
        } else if (typeof aVal === 'string') {
          aVal = aVal.toLowerCase()
          bVal = bVal.toLowerCase()
        }

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
        const response = await axios.get('/api/nat-gateways')
        this.natGateways = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load NAT Gateways'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      if (this.canModify) {
        try {
          await axios.post('/api/cache/nat-gateways/invalidate')
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
        case 'failed': return 'status-failed'
        case 'deleting': return 'status-deleting'
        case 'deleted': return 'status-deleted'
        default: return 'status-default'
      }
    },
    formatDate(dateString) {
      if (!dateString) return '-'
      const date = new Date(dateString)
      return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
      })
    },
    async deleteNATGateway(nat) {
      const confirmed = confirm(
        `Are you sure you want to delete NAT Gateway "${nat.name || nat.nat_gateway_id}"?\n\n` +
        `Account: ${nat.account_name}\n` +
        `Region: ${nat.region}\n` +
        `Public IP: ${nat.public_ip || 'N/A'}\n` +
        `VPC: ${nat.vpc_id}\n\n` +
        `WARNING: This action cannot be undone. Resources using this NAT Gateway will lose internet access.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        await axios.delete(`/api/accounts/${nat.account_id}/regions/${nat.region}/nat-gateways/${nat.nat_gateway_id}`)
        await this.loadData()
        alert(`NAT Gateway ${nat.nat_gateway_id} deletion initiated successfully`)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete NAT Gateway'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete NAT Gateway:', err)
      } finally {
        this.loading = false
      }
    },
    async deleteAllOldNATGateways() {
      const oldNATs = this.oldNATGateways
      if (oldNATs.length === 0) {
        alert('No NAT Gateways older than 6 months to delete')
        return
      }

      const accountFilter = this.filterAccount
      const accountName = accountFilter 
        ? this.natGateways.find(n => n.account_id === accountFilter)?.account_name || accountFilter
        : 'all accounts'

      const natsList = oldNATs.slice(0, 10).map(nat => {
        const age = this.getAgeInMonths(nat.create_time)
        return `  - ${nat.name || nat.nat_gateway_id} (${nat.region}, ${nat.connectivity_type || 'public'}, ${age} months old)`
      }).join('\n')
      const moreText = oldNATs.length > 10 ? `\n  ... and ${oldNATs.length - 10} more` : ''

      const confirmed = confirm(
        `Are you sure you want to delete ALL ${oldNATs.length} NAT Gateway(s) older than 6 months?\n\n` +
        `Account: ${accountName}\n\n` +
        `NAT Gateways to delete:\n${natsList}${moreText}\n\n` +
        `WARNING: This action cannot be undone!\n` +
        `Resources using these NAT Gateways will lose internet access.\n` +
        `This will permanently delete all NAT Gateways older than 6 months${accountFilter ? ' in this account' : ' across all accounts'}.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        let successCount = 0
        let failCount = 0
        const errors = []

        // Delete NAT Gateways one by one
        for (const nat of oldNATs) {
          try {
            const url = `/api/accounts/${nat.account_id}/regions/${nat.region}/nat-gateways/${nat.nat_gateway_id}`
            await axios.delete(url)
            successCount++
            
            // Small delay to avoid rate limiting
            await new Promise(resolve => setTimeout(resolve, 200))
          } catch (err) {
            failCount++
            const errorMsg = err.response?.data?.error || err.message || 'Unknown error'
            errors.push(`${nat.nat_gateway_id}: ${errorMsg}`)
            console.error(`Failed to delete NAT Gateway ${nat.nat_gateway_id}:`, err)
          }
        }

        // Reload data
        await this.loadData()

        // Show results
        let message = `Deleted ${successCount} NAT Gateway(s) successfully`
        if (failCount > 0) {
          message += `\n\nFailed to delete ${failCount} NAT Gateway(s):\n${errors.slice(0, 5).join('\n')}`
          if (errors.length > 5) {
            message += `\n... and ${errors.length - 5} more errors (check console)`
          }
        }
        alert(message)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete NAT Gateways'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete NAT Gateways:', err)
      } finally {
        this.loading = false
      }
    },
    getAgeInMonths(createdTime) {
      if (!createdTime) return 0
      const now = new Date()
      const created = new Date(createdTime)
      const diffTime = Math.abs(now - created)
      const diffMonths = Math.floor(diffTime / (1000 * 60 * 60 * 24 * 30))
      return diffMonths
    },
    downloadJSON() {
      const data = JSON.stringify(this.filteredNATGateways, null, 2)
      const blob = new Blob([data], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `nat-gateways-${new Date().toISOString().slice(0, 10)}.json`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    }
  }
}
</script>

<style scoped>
.nat-gateways-container {
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

.stat-card-available {
  border: 2px solid rgba(16, 185, 129, 0.3);
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.05) 0%, rgba(16, 185, 129, 0.02) 100%);
}

.stat-card-available .stat-value {
  color: #10b981;
}

.stat-card-old {
  border: 2px solid rgba(239, 68, 68, 0.3);
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.05) 0%, rgba(239, 68, 68, 0.02) 100%);
}

.stat-card-old .stat-value {
  color: #ef4444;
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

.name-cell {
  font-weight: 500;
  color: var(--color-text-primary);
}

.nat-name {
  font-weight: 500;
  color: var(--color-text-primary);
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

.status-failed {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.status-deleting {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.status-deleted {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.status-default {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.type-badge {
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 600;
}

.type-public {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.type-private {
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
