<template>
  <div class="snapshots-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M13,9V3.5L18.5,9M6,2C4.89,2 4,2.89 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2H6Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>EBS Snapshots</h1>
            <p v-if="filterAccount">
              {{ filteredSnapshots.length }} snapshot{{ filteredSnapshots.length !== 1 ? 's' : '' }} in {{ selectedAccountName }}
            </p>
            <p v-else>
              {{ snapshots.length }} snapshots across {{ uniqueAccounts.length }} accounts
            </p>
          </div>
        </div>
        <div class="header-actions">
          <button 
            v-if="filterAccount && canModify" 
            @click="deleteOldSnapshots" 
            class="btn btn-danger" 
            :disabled="loading || deletingOldSnapshots || oldSnapshotsCount === 0"
          >
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
            </svg>
            {{ deletingOldSnapshots ? 'Deleting...' : `Delete ${oldSnapshotsCount} Old Snapshots` }}
          </button>
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || snapshots.length === 0">
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
      <p>Loading snapshots...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load Snapshots</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Summary Stats -->
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ filterAccount ? filteredSnapshots.length : snapshots.length }}</div>
          <div class="stat-label">{{ filterAccount ? 'Snapshots' : 'Total Snapshots' }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ filterAccount ? filteredTotalSizeGB : totalSizeGB }} GB</div>
          <div class="stat-label">{{ filterAccount ? 'Total Size' : 'Total Size' }}</div>
        </div>
        <div class="stat-card" v-if="!filterAccount">
          <div class="stat-value">{{ uniqueAccounts.length }}</div>
          <div class="stat-label">Accounts</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ filterAccount ? filteredUniqueRegions.length : uniqueRegions.length }}</div>
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
            placeholder="Search by snapshot ID, volume ID, description, or account..."
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
          <option value="completed">Completed</option>
          <option value="pending">Pending</option>
          <option value="error">Error</option>
        </select>
      </div>

      <!-- Table -->
      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th @click="sortBy('snapshot_id')" class="sortable">
                Snapshot ID
                <span v-if="sortField === 'snapshot_id'" class="sort-indicator">
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
              <th @click="sortBy('volume_size')" class="sortable">
                Size
                <span v-if="sortField === 'volume_size'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th>Description</th>
              <th @click="sortBy('state')" class="sortable">
                State
                <span v-if="sortField === 'state'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th @click="sortBy('start_time')" class="sortable">
                Created
                <span v-if="sortField === 'start_time'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="snapshot in filteredSnapshots" :key="snapshot.snapshot_id" class="data-row">
              <td>
                <code class="snapshot-id">{{ snapshot.snapshot_id }}</code>
                <div class="volume-id">{{ snapshot.volume_id }}</div>
              </td>
              <td>
                <div class="account-info">
                  <div class="account-name">{{ snapshot.account_name }}</div>
                  <div class="account-id">{{ snapshot.account_id }}</div>
                </div>
              </td>
              <td>{{ snapshot.region }}</td>
              <td>{{ snapshot.volume_size }} GB</td>
              <td class="description-cell">
                <span class="description" :title="snapshot.description">
                  {{ truncateDescription(snapshot.description) }}
                </span>
              </td>
              <td>
                <span :class="['state-badge', `state-${snapshot.state}`]">
                  {{ snapshot.state }}
                </span>
              </td>
              <td>
                <div class="date-info">
                  <div class="date-value">{{ formatDate(snapshot.start_time) }}</div>
                  <div class="date-relative">{{ formatRelativeDate(snapshot.start_time) }}</div>
                </div>
              </td>
              <td>
                <button 
                  v-if="canModify"
                  @click="deleteSnapshot(snapshot)"
                  class="btn btn-sm btn-danger"
                  :disabled="actionLoading[snapshot.snapshot_id]"
                >
                  {{ actionLoading[snapshot.snapshot_id] ? 'Deleting...' : 'Delete' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-if="filteredSnapshots.length === 0" class="empty-results">
          <div class="empty-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M13,9V3.5L18.5,9M6,2C4.89,2 4,2.89 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2H6Z"/>
            </svg>
          </div>
          <h4>No snapshots found</h4>
          <p>No snapshots match the current search and filter criteria.</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'Snapshots',
  data() {
    return {
      snapshots: [],
      loading: true,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterRegion: '',
      filterState: '',
      sortField: 'start_time',
      sortDirection: 'desc',
      actionLoading: {},
      deletingOldSnapshots: false
    }
  },
  computed: {
    uniqueAccounts() {
      const accounts = new Map()
      this.snapshots.forEach(s => {
        if (!accounts.has(s.account_id)) {
          accounts.set(s.account_id, { id: s.account_id, name: s.account_name })
        }
      })
      return Array.from(accounts.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    uniqueRegions() {
      return [...new Set(this.snapshots.map(s => s.region))].sort()
    },
    totalSizeGB() {
      return this.snapshots.reduce((sum, s) => sum + (s.volume_size || 0), 0)
    },
    filteredTotalSizeGB() {
      return this.filteredSnapshots.reduce((sum, s) => sum + (s.volume_size || 0), 0)
    },
    filteredUniqueRegions() {
      return [...new Set(this.filteredSnapshots.map(s => s.region))].sort()
    },
    selectedAccountName() {
      if (!this.filterAccount) return ''
      const account = this.uniqueAccounts.find(a => a.id === this.filterAccount)
      return account ? `${account.name} (${account.id})` : this.filterAccount
    },
    oldSnapshotsCount() {
      if (!this.filterAccount) return 0
      const sixMonthsAgo = new Date()
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
      return this.filteredSnapshots.filter(s => {
        const snapshotDate = new Date(s.start_time)
        return snapshotDate < sixMonthsAgo && s.state === 'completed'
      }).length
    },
    filteredSnapshots() {
      let result = this.snapshots

      // Apply search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        result = result.filter(s =>
          s.snapshot_id.toLowerCase().includes(query) ||
          s.volume_id.toLowerCase().includes(query) ||
          s.description?.toLowerCase().includes(query) ||
          s.account_name.toLowerCase().includes(query) ||
          s.account_id.toLowerCase().includes(query)
        )
      }

      // Apply account filter
      if (this.filterAccount) {
        result = result.filter(s => s.account_id === this.filterAccount)
      }

      // Apply region filter
      if (this.filterRegion) {
        result = result.filter(s => s.region === this.filterRegion)
      }

      // Apply state filter
      if (this.filterState) {
        result = result.filter(s => s.state === this.filterState)
      }

      // Apply sorting
      result.sort((a, b) => {
        let aVal = a[this.sortField]
        let bVal = b[this.sortField]

        if (this.sortField === 'start_time') {
          aVal = new Date(aVal)
          bVal = new Date(bVal)
        } else if (this.sortField === 'volume_size') {
          aVal = aVal || 0
          bVal = bVal || 0
        } else if (typeof aVal === 'string') {
          aVal = aVal.toLowerCase()
          bVal = bVal.toLowerCase()
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
        const response = await axios.get('/api/snapshots')
        this.snapshots = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load snapshots'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      await this.loadData()
    },
    sortBy(field) {
      if (this.sortField === field) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortDirection = field === 'start_time' || field === 'volume_size' ? 'desc' : 'asc'
      }
    },
    formatDate(dateString) {
      return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
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
    truncateDescription(description) {
      if (!description) return '-'
      return description.length > 50 ? description.substring(0, 50) + '...' : description
    },
    async deleteSnapshot(snapshot) {
      const confirmMessage = `Are you sure you want to DELETE snapshot "${snapshot.snapshot_id}"?\n\nThis action cannot be undone!`
      if (!confirm(confirmMessage)) return

      this.actionLoading[snapshot.snapshot_id] = true

      try {
        await axios.delete(`/api/accounts/${snapshot.account_id}/regions/${snapshot.region}/snapshots/${snapshot.snapshot_id}`)
        alert(`Snapshot "${snapshot.snapshot_id}" has been successfully deleted.`)
        // Remove from local state
        this.snapshots = this.snapshots.filter(s => s.snapshot_id !== snapshot.snapshot_id)
      } catch (error) {
        console.error('Failed to delete snapshot:', error)
        alert(error.response?.data?.error || 'Failed to delete snapshot. Please try again.')
      } finally {
        delete this.actionLoading[snapshot.snapshot_id]
      }
    },
    downloadJSON() {
      try {
        const exportData = {
          exported_at: new Date().toISOString(),
          total_snapshots: this.filteredSnapshots.length,
          total_size_gb: this.totalSizeGB,
          snapshots: this.filteredSnapshots
        }

        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)

        const exportFileDefaultName = `aws-snapshots-${new Date().toISOString().split('T')[0]}.json`

        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (error) {
        console.error('Failed to download JSON:', error)
        alert('Failed to download JSON file')
      }
    },
    async deleteOldSnapshots() {
      if (!this.filterAccount) {
        alert('Please select an account first')
        return
      }

      const count = this.oldSnapshotsCount
      if (count === 0) {
        alert('No snapshots older than 6 months found in the selected account')
        return
      }

      const confirmMessage = `Are you sure you want to DELETE ${count} snapshot${count !== 1 ? 's' : ''} older than 6 months in ${this.selectedAccountName}?\n\nThis action cannot be undone!`
      if (!confirm(confirmMessage)) return

      this.deletingOldSnapshots = true

      try {
        const response = await axios.delete(`/api/accounts/${this.filterAccount}/snapshots/old`, {
          params: { older_than_months: 6 }
        })

        const deletedCount = response.data.count || 0
        alert(`Successfully deleted ${deletedCount} snapshot${deletedCount !== 1 ? 's' : ''} older than 6 months.`)
        
        // Refresh the data to update the list
        await this.loadData()
      } catch (error) {
        console.error('Failed to delete old snapshots:', error)
        const errorMsg = error.response?.data?.error || 'Failed to delete old snapshots. Please try again.'
        const deletedCount = error.response?.data?.count || 0
        
        if (deletedCount > 0) {
          alert(`Partially completed: ${deletedCount} snapshot${deletedCount !== 1 ? 's' : ''} deleted, but some errors occurred.\n\n${errorMsg}`)
          // Refresh the data to update the list
          await this.loadData()
        } else {
          alert(errorMsg)
        }
      } finally {
        this.deletingOldSnapshots = false
      }
    }
  }
}
</script>

<style scoped>
.snapshots-container {
  min-height: 100vh;
  background: var(--color-bg-secondary);
  padding: 1.5rem;
}

.page-header {
  margin-bottom: 2rem;
  padding: 1.5rem;
  background: var(--color-bg-primary);
  border-radius: 0.75rem;
  border: 1px solid var(--color-border);
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header-icon {
  width: 4rem;
  height: 4rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #8b5cf6, #a855f7);
  border-radius: 1rem;
  color: white;
  flex-shrink: 0;
}

.header-icon svg {
  width: 2rem;
  height: 2rem;
}

.title-content h1 {
  font-size: 2rem;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0 0 0.5rem 0;
}

.title-content p {
  font-size: 1rem;
  color: var(--color-text-secondary);
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 0.75rem;
  flex-shrink: 0;
}

.loading-container,
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  text-align: center;
  padding: 3rem;
}

.loading-spinner {
  width: 3rem;
  height: 3rem;
  border: 3px solid var(--color-border);
  border-top: 3px solid var(--color-btn-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error-icon {
  width: 4rem;
  height: 4rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 50%;
  color: #ef4444;
  margin-bottom: 1rem;
}

.error-icon svg {
  width: 2rem;
  height: 2rem;
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
  border-radius: 0.75rem;
  padding: 1.5rem;
  border: 1px solid var(--color-border);
  text-align: center;
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  color: var(--color-text-primary);
}

.stat-label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.filters {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
  padding: 1rem;
  background: var(--color-bg-primary);
  border-radius: 0.75rem;
  border: 1px solid var(--color-border);
}

.search-box {
  flex: 1;
  min-width: 250px;
  position: relative;
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  width: 1rem;
  height: 1rem;
  color: var(--color-text-secondary);
}

.search-input {
  width: 100%;
  padding: 0.75rem 0.75rem 0.75rem 2.5rem;
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  font-size: 0.875rem;
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
}

.search-input:focus {
  outline: none;
  border-color: var(--color-btn-primary);
}

.filter-select {
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  font-size: 0.875rem;
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  min-width: 150px;
}

.table-container {
  background: var(--color-bg-primary);
  border-radius: 0.75rem;
  border: 1px solid var(--color-border);
  overflow-x: auto;
  overflow-y: hidden;
}

.data-table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.data-table th {
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  font-weight: 600;
  text-align: left;
  padding: 1rem 0.75rem;
  border-bottom: 1px solid var(--color-border);
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  white-space: nowrap;
}

.data-table td {
  padding: 1rem 0.75rem;
  border-bottom: 1px solid var(--color-border);
  vertical-align: top;
}

.data-row {
  transition: all 0.2s ease;
}

.data-row:hover {
  background: var(--color-bg-secondary);
}

.sortable {
  cursor: pointer;
  user-select: none;
}

.sortable:hover {
  background: var(--color-bg-tertiary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-btn-primary);
}

.snapshot-id {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.75rem;
  background: var(--color-bg-tertiary);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  border: 1px solid var(--color-border);
}

.volume-id {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.7rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.account-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.account-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.account-id {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.description-cell {
  max-width: 200px;
}

.description {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.state-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: capitalize;
}

.state-completed {
  background: rgba(16, 185, 129, 0.1);
  color: var(--color-success);
}

.state-pending {
  background: rgba(245, 158, 11, 0.1);
  color: var(--color-warning);
}

.state-error {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-danger);
}

.date-info {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.date-value {
  font-weight: 500;
  color: var(--color-text-primary);
}

.date-relative {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--color-btn-primary);
  color: white;
}

.btn-secondary {
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
}

.btn-success {
  background: var(--color-btn-success);
  color: white;
}

.btn-danger {
  background: var(--color-btn-danger);
  color: white;
}

.btn-sm {
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
}

.btn-icon {
  width: 1rem;
  height: 1rem;
}

.empty-results {
  text-align: center;
  padding: 4rem 2rem;
  color: var(--color-text-secondary);
}

.empty-icon {
  width: 4rem;
  height: 4rem;
  margin: 0 auto 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-secondary);
  border-radius: 50%;
}

.empty-icon svg {
  width: 2rem;
  height: 2rem;
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
  }

  .header-actions {
    width: 100%;
    justify-content: center;
  }

  .filters {
    flex-direction: column;
  }

  .filter-select {
    width: 100%;
  }
}
</style>

