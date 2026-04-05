<template>
  <div class="ec2-instances-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M4,6H20V16H4V6M18,8H6V14H18V8M20,18H4V20H20V18Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>EC2 Instances</h1>
            <p v-if="filterAccount">
              {{ filteredInstances.length }} instance{{ filteredInstances.length !== 1 ? 's' : '' }} in {{ selectedAccountName }}
            </p>
            <p v-else>
              {{ instances.length }} instances across {{ uniqueAccounts.length }} accounts
            </p>
          </div>
        </div>
        <div class="header-actions">
          <button 
            v-if="canModify"
            @click="terminateAllStoppedInstances" 
            class="btn btn-danger" 
            :disabled="loading || !filterAccount || stoppedInstancesCount === 0"
            :title="!filterAccount ? 'Select an account first to enable bulk terminate' : (stoppedInstancesCount === 0 ? 'No stopped instances in this account' : `Terminate ${stoppedInstancesCount} stopped instance(s) in ${selectedAccountName}`)"
          >
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
            </svg>
            Terminate All Stopped ({{ stoppedInstancesCount }})
          </button>
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || instances.length === 0">
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
      <p>Loading EC2 instances...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load EC2 Instances</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Summary Stats -->
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ filterAccount ? filteredInstances.length : instances.length }}</div>
          <div class="stat-label">{{ filterAccount ? 'Instances' : 'Total Instances' }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ filterAccount ? filteredRunningInstances : runningInstances }}</div>
          <div class="stat-label">Running</div>
        </div>
        <div class="stat-card stat-card-stopped">
          <div class="stat-value">{{ stoppedInstancesCount }}</div>
          <div class="stat-label">Stopped</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ formatCurrency(filterAccount ? filteredTotalMonthlyCost : totalMonthlyCost) }}</div>
          <div class="stat-label">Monthly Cost</div>
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
            placeholder="Search by instance ID, name, account, region, or type..."
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
          <option value="running">Running</option>
          <option value="stopped">Stopped</option>
          <option value="terminated">Terminated</option>
          <option value="pending">Pending</option>
          <option value="stopping">Stopping</option>
        </select>
      </div>

      <!-- Instances Table -->
      <div class="table-container">
        <div v-if="filteredInstances.length === 0" class="empty-state">
          <div class="empty-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M4,6H20V16H4V6M18,8H6V14H18V8M20,18H4V20H20V18Z"/>
            </svg>
          </div>
          <h4>No instances found</h4>
          <p>No instances match the current search and filter criteria.</p>
        </div>
        <table v-else class="instances-table">
          <thead>
            <tr>
              <th @click="sortBy('instance_id')" class="sortable">
                Instance ID
                <span class="sort-indicator" v-if="sortField === 'instance_id'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('name')" class="sortable">
                Name
                <span class="sort-indicator" v-if="sortField === 'name'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('account_name')" class="sortable">
                Account
                <span class="sort-indicator" v-if="sortField === 'account_name'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('region')" class="sortable">
                Region
                <span class="sort-indicator" v-if="sortField === 'region'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('instance_type')" class="sortable">
                Flavor
                <span class="sort-indicator" v-if="sortField === 'instance_type'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('monthly_cost')" class="sortable">
                Monthly Cost
                <span class="sort-indicator" v-if="sortField === 'monthly_cost'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('launch_time')" class="sortable">
                Launch Date
                <span class="sort-indicator" v-if="sortField === 'launch_time'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('state')" class="sortable">
                Status
                <span class="sort-indicator" v-if="sortField === 'state'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="instance in filteredInstances" :key="instance.instance_id" class="data-row">
              <td><code class="instance-id">{{ instance.instance_id }}</code></td>
              <td>{{ instance.name || '-' }}</td>
              <td>
                <div class="account-info">
                  <div class="account-name">{{ instance.account_name }}</div>
                  <div class="account-id">{{ instance.account_id }}</div>
                </div>
              </td>
              <td>{{ instance.region }}</td>
              <td>{{ instance.instance_type }}</td>
              <td>
                <span class="cost-value">{{ formatCurrency(instance.monthly_cost || 0) }}</span>
              </td>
              <td>
                <div class="date-info">
                  <div class="date-value">{{ formatDate(instance.launch_time) }}</div>
                  <div class="date-relative">{{ formatRelativeDate(instance.launch_time) }}</div>
                </div>
              </td>
              <td>
                <span :class="['state-badge', `state-${instance.state.toLowerCase()}`]">
                  {{ instance.state }}
                </span>
              </td>
              <td>
                <div class="action-buttons" v-if="canModify">
                  <button
                    v-if="instance.state === 'running'"
                    @click="stopInstance(instance)"
                    class="action-btn action-btn-stop"
                    :disabled="loading"
                    title="Stop instance"
                  >
                    <svg viewBox="0 0 24 24" fill="currentColor">
                      <path d="M6,6H18V18H6V6Z"/>
                    </svg>
                    Stop
                  </button>
                  <button
                    v-if="instance.state !== 'terminated' && instance.state !== 'terminating'"
                    @click="terminateInstance(instance)"
                    class="action-btn action-btn-terminate"
                    :disabled="loading"
                    title="Terminate instance"
                  >
                    <svg viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                    </svg>
                    Terminate
                  </button>
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
  name: 'EC2Instances',
  data() {
    return {
      instances: [],
      loading: true,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterRegion: '',
      filterState: '',
      sortField: 'launch_time',
      sortDirection: 'desc'
    }
  },
  computed: {
    uniqueAccounts() {
      const accounts = new Map()
      this.instances.forEach(i => {
        if (!accounts.has(i.account_id)) {
          accounts.set(i.account_id, { id: i.account_id, name: i.account_name })
        }
      })
      return Array.from(accounts.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    uniqueRegions() {
      return [...new Set(this.instances.map(i => i.region))].sort()
    },
    runningInstances() {
      return this.instances.filter(i => i.state === 'running').length
    },
    filteredRunningInstances() {
      return this.filteredInstances.filter(i => i.state === 'running').length
    },
    totalMonthlyCost() {
      return this.instances.reduce((sum, i) => sum + (i.monthly_cost || 0), 0)
    },
    filteredTotalMonthlyCost() {
      return this.filteredInstances.reduce((sum, i) => sum + (i.monthly_cost || 0), 0)
    },
    filteredUniqueRegions() {
      return [...new Set(this.filteredInstances.map(i => i.region))].sort()
    },
    stoppedInstancesCount() {
      // If account filter is set, only count instances from that account
      let instancesToCheck = this.instances
      if (this.filterAccount) {
        instancesToCheck = instancesToCheck.filter(i => i.account_id === this.filterAccount)
      }
      return instancesToCheck.filter(i => i.state === 'stopped').length
    },
    stoppedInstances() {
      // Get all stopped instances, optionally filtered by account
      let instancesToCheck = this.instances
      if (this.filterAccount) {
        instancesToCheck = instancesToCheck.filter(i => i.account_id === this.filterAccount)
      }
      return instancesToCheck.filter(i => i.state === 'stopped')
    },
    selectedAccountName() {
      if (!this.filterAccount) return ''
      const account = this.uniqueAccounts.find(a => a.id === this.filterAccount)
      return account ? account.name : this.filterAccount
    },
    filteredInstances() {
      let result = this.instances

      // Apply search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        result = result.filter(i =>
          i.instance_id.toLowerCase().includes(query) ||
          i.name?.toLowerCase().includes(query) ||
          i.account_name.toLowerCase().includes(query) ||
          i.account_id.toLowerCase().includes(query) ||
          i.region.toLowerCase().includes(query) ||
          i.instance_type.toLowerCase().includes(query)
        )
      }

      // Apply filters
      if (this.filterAccount) {
        result = result.filter(i => i.account_id === this.filterAccount)
      }
      if (this.filterRegion) {
        result = result.filter(i => i.region === this.filterRegion)
      }
      if (this.filterState) {
        result = result.filter(i => i.state === this.filterState)
      }

      // Apply sorting
      result.sort((a, b) => {
        let aVal = a[this.sortField]
        let bVal = b[this.sortField]

        if (this.sortField === 'launch_time') {
          aVal = new Date(aVal)
          bVal = new Date(bVal)
        } else if (this.sortField === 'monthly_cost') {
          aVal = aVal || 0
          bVal = bVal || 0
        } else if (typeof aVal === 'string') {
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
        const response = await axios.get('/api/ec2-instances')
        this.instances = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load EC2 instances'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      if (this.canModify) {
        try {
          await axios.post('/api/cache/ec2-instances/invalidate')
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
        this.sortDirection = field === 'launch_time' ? 'desc' : 'asc'
      }
    },
    formatDate(dateString) {
      return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
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
    formatCurrency(amount) {
      return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD',
        minimumFractionDigits: 2,
        maximumFractionDigits: 2
      }).format(amount)
    },
    downloadJSON() {
      try {
        const exportData = {
          exported_at: new Date().toISOString(),
          total_instances: this.filteredInstances.length,
          running_instances: this.runningInstances,
          instances: this.filteredInstances
        }

        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)
        const exportFileDefaultName = `aws-ec2-instances-${new Date().toISOString().split('T')[0]}.json`

        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (error) {
        console.error('Failed to download JSON:', error)
        alert('Failed to download JSON file')
      }
    },
    async stopInstance(instance) {
      const confirmed = confirm(`Are you sure you want to stop instance ${instance.instance_id}?`)
      if (!confirmed) return

      try {
        this.loading = true
        const url = `/api/accounts/${instance.account_id}/regions/${instance.region}/instances/${instance.instance_id}/stop`
        await axios.post(url)

        // Reload data - the cache has been updated with the new state
        await this.loadData()

        alert(`Instance ${instance.instance_id} stopped successfully`)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to stop instance'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to stop instance:', err)
      } finally {
        this.loading = false
      }
    },
    async terminateInstance(instance) {
      const confirmed = confirm(
        `Are you sure you want to TERMINATE instance ${instance.instance_id}?\n\n` +
        `WARNING: This action cannot be undone. The instance will be permanently deleted.`
      )
      if (!confirmed) return

      // Double confirmation for terminate
      const doubleConfirmed = confirm(
        `Last confirmation: Type the instance ID to confirm termination.\n\n` +
        `Instance ID: ${instance.instance_id}`
      )
      if (!doubleConfirmed) return

      try {
        this.loading = true
        const url = `/api/accounts/${instance.account_id}/regions/${instance.region}/instances/${instance.instance_id}/terminate`
        await axios.post(url)

        // Reload data - the cache has been updated with the new state
        await this.loadData()

        alert(`Instance ${instance.instance_id} terminated successfully`)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to terminate instance'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to terminate instance:', err)
      } finally {
        this.loading = false
      }
    },
    async terminateAllStoppedInstances() {
      const stoppedInsts = this.stoppedInstances
      if (stoppedInsts.length === 0) {
        alert('No stopped instances to terminate')
        return
      }

      const instancesList = stoppedInsts.slice(0, 10).map(i => `  - ${i.instance_id} (${i.name || 'no name'}, ${i.region})`).join('\n')
      const moreText = stoppedInsts.length > 10 ? `\n  ... and ${stoppedInsts.length - 10} more` : ''

      const confirmed = confirm(
        `Are you sure you want to TERMINATE ALL ${stoppedInsts.length} stopped instance(s)?\n\n` +
        `Account: ${this.selectedAccountName}\n\n` +
        `Instances to terminate:\n${instancesList}${moreText}\n\n` +
        `WARNING: This action cannot be undone!\n` +
        `All stopped instances in this account will be permanently deleted.`
      )
      if (!confirmed) return

      // Double confirmation for bulk terminate
      const doubleConfirmed = confirm(
        `FINAL CONFIRMATION\n\n` +
        `You are about to terminate ${stoppedInsts.length} stopped instance(s).\n\n` +
        `This will permanently delete these instances and all their data.\n\n` +
        `Click OK to proceed with termination.`
      )
      if (!doubleConfirmed) return

      try {
        this.loading = true
        let successCount = 0
        let failCount = 0
        const errors = []

        // Terminate instances one by one
        for (const instance of stoppedInsts) {
          try {
            const url = `/api/accounts/${instance.account_id}/regions/${instance.region}/instances/${instance.instance_id}/terminate`
            await axios.post(url)
            successCount++
            
            // Small delay to avoid rate limiting
            await new Promise(resolve => setTimeout(resolve, 200))
          } catch (err) {
            failCount++
            const errorMsg = err.response?.data?.error || err.message || 'Unknown error'
            errors.push(`${instance.instance_id}: ${errorMsg}`)
            console.error(`Failed to terminate instance ${instance.instance_id}:`, err)
          }
        }

        // Reload data
        await this.loadData()

        // Show results
        let message = `Terminated ${successCount} instance(s) successfully`
        if (failCount > 0) {
          message += `\n\nFailed to terminate ${failCount} instance(s):\n${errors.slice(0, 5).join('\n')}`
          if (errors.length > 5) {
            message += `\n... and ${errors.length - 5} more errors (check console)`
          }
        }
        alert(message)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to terminate instances'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to terminate instances:', err)
      } finally {
        this.loading = false
      }
    }
  }
}
</script>

<style scoped>
.ec2-instances-container {
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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
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

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.btn-danger {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: white;
}

.btn-danger:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
}

.main-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.summary-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
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

.stat-card-stopped {
  border: 2px solid rgba(239, 68, 68, 0.3);
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.05) 0%, rgba(239, 68, 68, 0.02) 100%);
}

.stat-card-stopped .stat-value {
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

.instances-table {
  width: 100%;
  min-width: 1000px;
  border-collapse: collapse;
}

.instances-table thead {
  background: var(--color-bg-secondary);
}

.instances-table th {
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

.instances-table th.sortable:hover {
  color: var(--color-primary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-primary);
}

.instances-table tbody tr {
  border-top: 1px solid var(--color-border);
}

.instances-table tbody tr:hover {
  background: var(--color-bg-hover);
}

.instances-table td {
  padding: 1rem 1.5rem;
  color: var(--color-text-primary);
}

.instance-id {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.9rem;
  background: var(--color-bg-secondary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  color: var(--color-primary);
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

.cost-value {
  font-weight: 600;
  color: var(--color-text-primary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875rem;
}

.state-badge {
  display: inline-block;
  padding: 0.35rem 0.75rem;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 500;
  text-transform: capitalize;
}

.state-running {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.state-stopped {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.state-pending {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.state-stopping {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.state-terminated {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
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

.action-btn-stop {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: white;
}

.action-btn-stop:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(245, 158, 11, 0.4);
}

.action-btn-terminate {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: white;
}

.action-btn-terminate:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    align-items: flex-start;
  }

  .header-actions {
    width: 100%;
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

  .instances-table {
    font-size: 0.85rem;
  }

  .instances-table th,
  .instances-table td {
    padding: 0.75rem 1rem;
  }
}
</style>
