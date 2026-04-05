<template>
  <div class="load-balancers-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4C16.41,4 20,7.59 20,12C20,16.41 16.41,20 12,20C7.59,20 4,16.41 4,12C4,7.59 7.59,4 12,4M12,6A6,6 0 0,0 6,12A6,6 0 0,0 12,18A6,6 0 0,0 18,12A6,6 0 0,0 12,6M12,8A4,4 0 0,1 16,12A4,4 0 0,1 12,16A4,4 0 0,1 8,12A4,4 0 0,1 12,8Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>Load Balancers</h1>
            <p>{{ loadBalancers.length }} load balancers across {{ uniqueAccounts.length }} accounts</p>
          </div>
        </div>
        <div class="header-actions">
          <button 
            v-if="canModify"
            @click="deleteAllUnusedLoadBalancers" 
            class="btn btn-danger" 
            :disabled="loading || !filterAccount || unusedLoadBalancersCount === 0"
            :title="!filterAccount ? 'Select an account first to enable bulk delete' : (unusedLoadBalancersCount === 0 ? 'No unused load balancers in this account' : `Delete ${unusedLoadBalancersCount} unused load balancer(s) in ${selectedAccountName}`)"
          >
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
            </svg>
            Delete All Unused ({{ unusedLoadBalancersCount }})
          </button>
          <button 
            v-if="canModify"
            @click="deleteAllOldUnusedLoadBalancers" 
            class="btn btn-danger" 
            :disabled="loading || !filterAccount || oldUnusedLoadBalancersCount === 0"
            :title="!filterAccount ? 'Select an account first to enable bulk delete' : (oldUnusedLoadBalancersCount === 0 ? 'No unused load balancers older than 6 months in this account' : `Delete ${oldUnusedLoadBalancersCount} unused load balancer(s) older than 6 months in ${selectedAccountName}`)"
          >
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
            </svg>
            Delete Old Unused ({{ oldUnusedLoadBalancersCount }})
          </button>
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || loadBalancers.length === 0">
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
      <p>Loading load balancers...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load Load Balancers</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Summary Stats -->
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ loadBalancers.length }}</div>
          <div class="stat-label">Total Load Balancers</div>
        </div>
        <div class="stat-card stat-card-unused">
          <div class="stat-value">{{ unusedLoadBalancersCount }}</div>
          <div class="stat-label">Unused</div>
        </div>
        <div class="stat-card stat-card-old-unused">
          <div class="stat-value">{{ oldUnusedLoadBalancersCount }}</div>
          <div class="stat-label">Unused (6+ months)</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ uniqueAccounts.length }}</div>
          <div class="stat-label">Accounts</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ uniqueRegions.length }}</div>
          <div class="stat-label">Regions</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ applicationLBs }}</div>
          <div class="stat-label">Application (ALB)</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ networkLBs }}</div>
          <div class="stat-label">Network (NLB)</div>
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
            placeholder="Search by name, DNS, account, type, or region..."
            class="search-input"
          />
        </div>
        <select v-model="filterAccount" class="filter-select">
          <option value="">All Accounts</option>
          <option v-for="account in uniqueAccounts" :key="account.id" :value="account.id">
            {{ account.name }} ({{ account.id }})
          </option>
        </select>
        <select v-model="filterType" class="filter-select">
          <option value="">All Types</option>
          <option value="application">Application (ALB)</option>
          <option value="network">Network (NLB)</option>
          <option value="classic">Classic ELB</option>
        </select>
        <select v-model="filterRegion" class="filter-select">
          <option value="">All Regions</option>
          <option v-for="region in uniqueRegions" :key="region" :value="region">
            {{ region }}
          </option>
        </select>
        <select v-model="filterUnused" class="filter-select">
          <option value="">All Load Balancers</option>
          <option value="unused">Unused Only</option>
          <option value="used">In Use Only</option>
        </select>
      </div>

      <!-- Load Balancers Table -->
      <div class="table-container">
        <div v-if="filteredLoadBalancers.length === 0" class="empty-state">
          <div class="empty-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4C16.41,4 20,7.59 20,12C20,16.41 16.41,20 12,20C7.59,20 4,16.41 4,12C4,7.59 7.59,4 12,4M12,6A6,6 0 0,0 6,12A6,6 0 0,0 12,18A6,6 0 0,0 18,12A6,6 0 0,0 12,6M12,8A4,4 0 0,1 16,12A4,4 0 0,1 12,16A4,4 0 0,1 8,12A4,4 0 0,1 12,8Z"/>
            </svg>
          </div>
          <h4>No load balancers found</h4>
          <p>No load balancers match the current search and filter criteria.</p>
        </div>
        <table v-else class="load-balancers-table">
          <thead>
            <tr>
              <th @click="sortBy('load_balancer_name')" class="sortable">
                Name
                <span class="sort-indicator" v-if="sortField === 'load_balancer_name'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('account_name')" class="sortable">
                Account
                <span class="sort-indicator" v-if="sortField === 'account_name'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('type')" class="sortable">
                Type
                <span class="sort-indicator" v-if="sortField === 'type'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('scheme')" class="sortable">
                Scheme
                <span class="sort-indicator" v-if="sortField === 'scheme'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('region')" class="sortable">
                Region
                <span class="sort-indicator" v-if="sortField === 'region'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Targets</th>
              <th @click="sortBy('created_time')" class="sortable">
                Created
                <span class="sort-indicator" v-if="sortField === 'created_time'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="lb in filteredLoadBalancers" :key="lb.load_balancer_arn || lb.load_balancer_name + lb.account_id" class="data-row">
              <td>
                <code class="lb-name">{{ lb.load_balancer_name }}</code>
                <div v-if="lb.dns_name" class="dns-name">{{ lb.dns_name }}</div>
              </td>
              <td>
                <div class="account-info">
                  <div class="account-name">{{ lb.account_name }}</div>
                  <div class="account-id">{{ lb.account_id }}</div>
                </div>
              </td>
              <td>
                <span :class="['type-badge', `type-${lb.type}`]">
                  {{ lb.type === 'application' ? 'ALB' : lb.type === 'network' ? 'NLB' : 'Classic' }}
                </span>
              </td>
              <td>
                <span :class="['scheme-badge', lb.scheme === 'internet-facing' ? 'scheme-internet' : 'scheme-internal']">
                  {{ lb.scheme === 'internet-facing' ? 'Internet' : 'Internal' }}
                </span>
              </td>
              <td>{{ lb.region }}</td>
              <td>
                <div class="target-info">
                  <span v-if="lb.target_count > 0" class="target-count">
                    {{ lb.healthy_target_count }}/{{ lb.target_count }} healthy
                  </span>
                  <span v-else class="no-targets">No targets</span>
                  <div v-if="lb.listener_count > 0" class="listener-count">{{ lb.listener_count }} listener(s)</div>
                </div>
              </td>
              <td>
                <div v-if="lb.created_time" class="date-info">
                  <div class="date-value">{{ formatDate(lb.created_time) }}</div>
                  <div class="date-relative">{{ formatRelativeDate(lb.created_time) }}</div>
                </div>
                <span v-else>-</span>
              </td>
              <td>
                <span v-if="lb.is_unused" class="badge badge-warning">Unused</span>
                <span v-else class="badge badge-success">In Use</span>
              </td>
              <td>
                <div class="action-buttons">
                  <button
                    v-if="canModify && lb.is_unused"
                    @click="deleteLoadBalancer(lb)"
                    class="action-btn action-btn-delete"
                    :disabled="loading"
                    title="Delete unused load balancer"
                  >
                    <svg viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                    </svg>
                    Delete
                  </button>
                  <span v-else class="no-action">-</span>
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
  name: 'LoadBalancers',
  data() {
    return {
      loadBalancers: [],
      loading: true,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterType: '',
      filterRegion: '',
      filterUnused: '',
      sortField: 'created_time',
      sortDirection: 'desc'
    }
  },
  computed: {
    uniqueAccounts() {
      const accounts = new Map()
      this.loadBalancers.forEach(lb => {
        if (!accounts.has(lb.account_id)) {
          accounts.set(lb.account_id, { id: lb.account_id, name: lb.account_name })
        }
      })
      return Array.from(accounts.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    selectedAccountName() {
      if (!this.filterAccount) return ''
      const account = this.uniqueAccounts.find(a => a.id === this.filterAccount)
      return account ? account.name : this.filterAccount
    },
    unusedLoadBalancersCount() {
      // If account filter is set, only count LBs from that account
      let lbsToCheck = this.loadBalancers
      if (this.filterAccount && this.filterAccount.trim() !== '') {
        const filterAccountId = String(this.filterAccount).trim()
        lbsToCheck = lbsToCheck.filter(lb => {
          const lbAccountId = lb.account_id ? String(lb.account_id).trim() : ''
          return lbAccountId === filterAccountId
        })
      }
      return lbsToCheck.filter(lb => lb.is_unused).length
    },
    unusedLoadBalancers() {
      // Get all unused LBs, optionally filtered by account
      let lbsToCheck = this.loadBalancers
      if (this.filterAccount && this.filterAccount.trim() !== '') {
        const filterAccountId = String(this.filterAccount).trim()
        lbsToCheck = lbsToCheck.filter(lb => {
          const lbAccountId = lb.account_id ? String(lb.account_id).trim() : ''
          return lbAccountId === filterAccountId
        })
      }
      return lbsToCheck.filter(lb => lb.is_unused)
    },
    oldUnusedLoadBalancersCount() {
      // Count unused LBs older than 6 months, optionally filtered by account
      let lbsToCheck = this.loadBalancers
      if (this.filterAccount && this.filterAccount.trim() !== '') {
        const filterAccountId = String(this.filterAccount).trim()
        lbsToCheck = lbsToCheck.filter(lb => {
          const lbAccountId = lb.account_id ? String(lb.account_id).trim() : ''
          return lbAccountId === filterAccountId
        })
      }
      const sixMonthsAgo = new Date()
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
      return lbsToCheck.filter(lb => {
        if (!lb.is_unused) return false
        if (!lb.created_time) return false
        const createdDate = new Date(lb.created_time)
        return createdDate < sixMonthsAgo
      }).length
    },
    oldUnusedLoadBalancers() {
      // Get unused LBs older than 6 months, optionally filtered by account
      let lbsToCheck = this.loadBalancers
      if (this.filterAccount && this.filterAccount.trim() !== '') {
        const filterAccountId = String(this.filterAccount).trim()
        lbsToCheck = lbsToCheck.filter(lb => {
          const lbAccountId = lb.account_id ? String(lb.account_id).trim() : ''
          return lbAccountId === filterAccountId
        })
      }
      const sixMonthsAgo = new Date()
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
      return lbsToCheck.filter(lb => {
        if (!lb.is_unused) return false
        if (!lb.created_time) return false
        const createdDate = new Date(lb.created_time)
        return createdDate < sixMonthsAgo
      })
    },
    uniqueRegions() {
      return [...new Set(this.loadBalancers.map(lb => lb.region))].sort()
    },
    applicationLBs() {
      return this.loadBalancers.filter(lb => lb.type === 'application').length
    },
    networkLBs() {
      return this.loadBalancers.filter(lb => lb.type === 'network').length
    },
    classicLBs() {
      return this.loadBalancers.filter(lb => lb.type === 'classic').length
    },
    filteredLoadBalancers() {
      let result = this.loadBalancers

      // Apply search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        result = result.filter(lb =>
          (lb.load_balancer_name || '').toLowerCase().includes(query) ||
          (lb.dns_name && lb.dns_name.toLowerCase().includes(query)) ||
          (lb.account_name || '').toLowerCase().includes(query) ||
          (lb.account_id || '').toLowerCase().includes(query) ||
          (lb.type || '').toLowerCase().includes(query) ||
          (lb.region || '').toLowerCase().includes(query)
        )
      }

      // Apply account filter
      if (this.filterAccount && this.filterAccount.trim() !== '') {
        const filterAccountId = String(this.filterAccount).trim()
        result = result.filter(lb => {
          const lbAccountId = lb.account_id ? String(lb.account_id).trim() : ''
          return lbAccountId === filterAccountId
        })
      }

      // Apply type filter
      if (this.filterType) {
        result = result.filter(lb => lb.type === this.filterType)
      }

      // Apply region filter
      if (this.filterRegion) {
        result = result.filter(lb => lb.region === this.filterRegion)
      }

      // Apply unused filter
      if (this.filterUnused === 'unused') {
        result = result.filter(lb => lb.is_unused)
      } else if (this.filterUnused === 'used') {
        result = result.filter(lb => !lb.is_unused)
      }

      // Apply sorting
      result.sort((a, b) => {
        let aVal = a[this.sortField]
        let bVal = b[this.sortField]

        // Handle null/undefined values
        if (aVal == null && bVal == null) return 0
        if (aVal == null) return 1
        if (bVal == null) return -1

        // Handle date/time fields
        if (this.sortField === 'created_time') {
          aVal = new Date(aVal).getTime()
          bVal = new Date(bVal).getTime()
        }
        // Handle string fields - convert to lowercase for case-insensitive comparison
        else if (typeof aVal === 'string' || typeof bVal === 'string') {
          aVal = String(aVal || '').toLowerCase()
          bVal = String(bVal || '').toLowerCase()
        }
        // Handle numeric fields (like target_count, healthy_target_count, listener_count)
        else if (typeof aVal === 'number' || typeof bVal === 'number') {
          aVal = Number(aVal) || 0
          bVal = Number(bVal) || 0
        }
        // Handle boolean fields
        else if (typeof aVal === 'boolean' || typeof bVal === 'boolean') {
          aVal = Boolean(aVal)
          bVal = Boolean(bVal)
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
        const response = await axios.get('/api/load-balancers')
        this.loadBalancers = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load load balancers'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      if (this.canModify) {
        try {
          await axios.post('/api/cache/load-balancers/invalidate')
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
        this.sortDirection = field === 'created_time' ? 'desc' : 'asc'
      }
    },
    formatDate(dateString) {
      if (!dateString) return '-'
      return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      })
    },
    formatRelativeDate(dateString) {
      if (!dateString) return ''
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
          total_load_balancers: this.filteredLoadBalancers.length,
          unused_count: this.unusedLoadBalancersCount,
          load_balancers: this.filteredLoadBalancers
        }

        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)
        const exportFileDefaultName = `aws-load-balancers-${new Date().toISOString().split('T')[0]}.json`

        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (error) {
        console.error('Failed to download JSON:', error)
        alert('Failed to download JSON file')
      }
    },
    async deleteLoadBalancer(lb) {
      const confirmed = confirm(
        `Are you sure you want to delete load balancer "${lb.load_balancer_name}"?\n\n` +
        `Account: ${lb.account_name}\n` +
        `Type: ${lb.type === 'application' ? 'ALB' : lb.type === 'network' ? 'NLB' : 'Classic ELB'}\n` +
        `Region: ${lb.region}\n` +
        `DNS: ${lb.dns_name || 'N/A'}\n\n` +
        `WARNING: This action cannot be undone.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        const identifier = lb.load_balancer_arn || lb.load_balancer_name
        const url = `/api/accounts/${lb.account_id}/regions/${lb.region}/load-balancers?id=${encodeURIComponent(identifier)}&type=${lb.type}`
        await axios.delete(url)

        // Reload data
        await this.loadData()

        alert(`Load balancer ${lb.load_balancer_name} deleted successfully`)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete load balancer'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete load balancer:', err)
      } finally {
        this.loading = false
      }
    },
    async deleteAllUnusedLoadBalancers() {
      const unusedLBs = this.unusedLoadBalancers
      if (unusedLBs.length === 0) {
        alert('No unused load balancers to delete')
        return
      }

      const lbsList = unusedLBs.slice(0, 10).map(lb => `  - ${lb.load_balancer_name} (${lb.type}, ${lb.region})`).join('\n')
      const moreText = unusedLBs.length > 10 ? `\n  ... and ${unusedLBs.length - 10} more` : ''

      const confirmed = confirm(
        `Are you sure you want to delete ALL ${unusedLBs.length} unused load balancer(s)?\n\n` +
        `Account: ${this.selectedAccountName}\n\n` +
        `Load balancers to delete:\n${lbsList}${moreText}\n\n` +
        `WARNING: This action cannot be undone!`
      )
      if (!confirmed) return

      try {
        this.loading = true
        let successCount = 0
        let failCount = 0
        const errors = []

        // Delete load balancers one by one
        for (const lb of unusedLBs) {
          try {
            const identifier = lb.load_balancer_arn || lb.load_balancer_name
            const url = `/api/accounts/${lb.account_id}/regions/${lb.region}/load-balancers?id=${encodeURIComponent(identifier)}&type=${lb.type}`
            await axios.delete(url)
            successCount++
            
            // Small delay to avoid rate limiting
            await new Promise(resolve => setTimeout(resolve, 200))
          } catch (err) {
            failCount++
            const errorMsg = err.response?.data?.error || err.message || 'Unknown error'
            errors.push(`${lb.load_balancer_name}: ${errorMsg}`)
            console.error(`Failed to delete load balancer ${lb.load_balancer_name}:`, err)
          }
        }

        // Reload data
        await this.loadData()

        // Show results
        let message = `Deleted ${successCount} load balancer(s) successfully`
        if (failCount > 0) {
          message += `\n\nFailed to delete ${failCount} load balancer(s):\n${errors.slice(0, 5).join('\n')}`
          if (errors.length > 5) {
            message += `\n... and ${errors.length - 5} more errors (check console)`
          }
        }
        alert(message)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete load balancers'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete load balancers:', err)
      } finally {
        this.loading = false
      }
    },
    async deleteAllOldUnusedLoadBalancers() {
      const oldUnusedLBs = this.oldUnusedLoadBalancers
      if (oldUnusedLBs.length === 0) {
        alert('No unused load balancers older than 6 months to delete')
        return
      }

      const lbsList = oldUnusedLBs.slice(0, 10).map(lb => {
        const age = this.getAgeInMonths(lb.created_time)
        return `  - ${lb.load_balancer_name} (${lb.type}, ${lb.region}, ${age} months old)`
      }).join('\n')
      const moreText = oldUnusedLBs.length > 10 ? `\n  ... and ${oldUnusedLBs.length - 10} more` : ''

      const confirmed = confirm(
        `Are you sure you want to delete ALL ${oldUnusedLBs.length} unused load balancer(s) older than 6 months?\n\n` +
        `Account: ${this.selectedAccountName}\n\n` +
        `Load balancers to delete:\n${lbsList}${moreText}\n\n` +
        `WARNING: This action cannot be undone!`
      )
      if (!confirmed) return

      try {
        this.loading = true
        let successCount = 0
        let failCount = 0
        const errors = []

        // Delete load balancers one by one
        for (const lb of oldUnusedLBs) {
          try {
            const identifier = lb.load_balancer_arn || lb.load_balancer_name
            const url = `/api/accounts/${lb.account_id}/regions/${lb.region}/load-balancers?id=${encodeURIComponent(identifier)}&type=${lb.type}`
            await axios.delete(url)
            successCount++
            
            // Small delay to avoid rate limiting
            await new Promise(resolve => setTimeout(resolve, 200))
          } catch (err) {
            failCount++
            const errorMsg = err.response?.data?.error || err.message || 'Unknown error'
            errors.push(`${lb.load_balancer_name}: ${errorMsg}`)
            console.error(`Failed to delete load balancer ${lb.load_balancer_name}:`, err)
          }
        }

        // Reload data
        await this.loadData()

        // Show results
        let message = `Deleted ${successCount} load balancer(s) successfully`
        if (failCount > 0) {
          message += `\n\nFailed to delete ${failCount} load balancer(s):\n${errors.slice(0, 5).join('\n')}`
          if (errors.length > 5) {
            message += `\n... and ${errors.length - 5} more errors (check console)`
          }
        }
        alert(message)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete load balancers'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete load balancers:', err)
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
    }
  }
}
</script>

<style scoped>
.load-balancers-container {
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

.btn-primary:hover {
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

.stat-card-unused {
  border: 2px solid rgba(245, 158, 11, 0.3);
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.05) 0%, rgba(245, 158, 11, 0.02) 100%);
}

.stat-card-unused .stat-value {
  color: #f59e0b;
}

.stat-card-old-unused {
  border: 2px solid rgba(239, 68, 68, 0.3);
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.05) 0%, rgba(239, 68, 68, 0.02) 100%);
}

.stat-card-old-unused .stat-value {
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

.load-balancers-table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
}

.load-balancers-table thead {
  background: var(--color-bg-secondary);
}

.load-balancers-table th {
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

.load-balancers-table th.sortable:hover {
  color: var(--color-primary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-primary);
}

.load-balancers-table tbody tr {
  border-top: 1px solid var(--color-border);
}

.load-balancers-table tbody tr:hover {
  background: var(--color-bg-hover);
}

.load-balancers-table td {
  padding: 1rem 1.5rem;
  color: var(--color-text-primary);
}

.lb-name {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.9rem;
  background: var(--color-bg-secondary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  color: var(--color-primary);
  display: block;
  margin-bottom: 0.25rem;
}

.dns-name {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
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

.type-badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.type-application {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.type-network {
  background: rgba(139, 92, 246, 0.1);
  color: #8b5cf6;
}

.type-classic {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.scheme-badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.scheme-internet {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.scheme-internal {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.target-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.target-count {
  font-weight: 500;
}

.no-targets {
  color: var(--color-text-secondary);
  font-style: italic;
}

.listener-count {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
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

.badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.badge-success {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.badge-warning {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
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

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    align-items: flex-start;
  }

  .header-actions {
    width: 100%;
    flex-wrap: wrap;
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

  .load-balancers-table {
    font-size: 0.85rem;
  }

  .load-balancers-table th,
  .load-balancers-table td {
    padding: 0.75rem 1rem;
  }
}
</style>
