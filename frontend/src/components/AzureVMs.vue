<template>
  <div class="azure-vms-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon" style="color: #0078d4;">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M4,6H20V16H4V6M18,8H6V14H18V8M20,18H4V20H20V18Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>Azure Virtual Machines</h1>
            <p>{{ filteredVMs.length }} virtual machines</p>
          </div>
        </div>
        <div class="header-actions">
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || vms.length === 0">
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

    <!-- Info Message -->
    <div v-if="!azureConfigured" class="info-message">
      <svg viewBox="0 0 24 24" fill="currentColor">
        <path d="M13,9H11V7H13M13,17H11V11H13M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z"/>
      </svg>
      <div>
        <strong>Azure Resource Manager not configured</strong>
        <p>Set AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, and AZURE_SUBSCRIPTION_ID environment variables to enable Azure VM management.</p>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading virtual machines...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load Virtual Machines</h3>
      <p>{{ error }}</p>
      <button @click="refreshData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Summary Stats -->
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ vms.length }}</div>
          <div class="stat-label">Total VMs</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ runningCount }}</div>
          <div class="stat-label">Running</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ stoppedCount }}</div>
          <div class="stat-label">Stopped</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ uniqueResourceGroups.length }}</div>
          <div class="stat-label">Resource Groups</div>
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
            placeholder="Search by name, resource group, location, or size..."
            class="search-input"
          />
        </div>
        <div class="filter-group">
          <select v-model="filterResourceGroup" class="filter-select">
            <option value="">All Resource Groups</option>
            <option v-for="rg in uniqueResourceGroups" :key="rg" :value="rg">{{ rg }}</option>
          </select>
          <select v-model="filterStatus" class="filter-select">
            <option value="">All Statuses</option>
            <option value="running">Running</option>
            <option value="stopped">Stopped</option>
            <option value="deallocated">Deallocated</option>
          </select>
        </div>
      </div>

      <!-- VMs Table -->
      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th @click="sortBy('name')" class="sortable">
                Name
                <span v-if="sortField === 'name'" class="sort-indicator">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('resource_group')" class="sortable">
                Resource Group
                <span v-if="sortField === 'resource_group'" class="sort-indicator">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('location')" class="sortable">
                Location
                <span v-if="sortField === 'location'" class="sort-indicator">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('vm_size')" class="sortable">
                Size
                <span v-if="sortField === 'vm_size'" class="sort-indicator">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('status')" class="sortable">
                Status
                <span v-if="sortField === 'status'" class="sort-indicator">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('os_type')" class="sortable">
                OS Type
                <span v-if="sortField === 'os_type'" class="sort-indicator">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="vm in paginatedVMs" :key="vm.id" class="table-row">
              <td><strong>{{ vm.name }}</strong></td>
              <td>{{ vm.resource_group }}</td>
              <td>{{ vm.location }}</td>
              <td>{{ vm.vm_size }}</td>
              <td>
                <span :class="['status-badge', getStatusClass(vm.status)]">
                  {{ vm.status || 'Unknown' }}
                </span>
              </td>
              <td>{{ vm.os_type || 'N/A' }}</td>
              <td>
                <div class="action-buttons" v-if="canModify">
                  <button
                    v-if="vm.status && vm.status.toLowerCase() !== 'running'"
                    @click="startVM(vm)"
                    class="btn btn-sm btn-success"
                    :disabled="vmActionLoading === vm.id"
                    title="Start VM"
                  >
                    <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M8,5.14V19.14L19,12.14L8,5.14Z"/>
                    </svg>
                  </button>
                  <button
                    v-if="vm.status && vm.status.toLowerCase() === 'running'"
                    @click="stopVM(vm)"
                    class="btn btn-sm btn-warning"
                    :disabled="vmActionLoading === vm.id"
                    title="Stop VM"
                  >
                    <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M18,18H6V6H18V18Z"/>
                    </svg>
                  </button>
                  <button
                    @click="deleteVM(vm)"
                    class="btn btn-sm btn-danger"
                    :disabled="vmActionLoading === vm.id"
                    title="Delete VM"
                  >
                    <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="pagination">
        <button @click="currentPage = 1" :disabled="currentPage === 1" class="btn btn-sm">First</button>
        <button @click="currentPage--" :disabled="currentPage === 1" class="btn btn-sm">Previous</button>
        <span>Page {{ currentPage }} of {{ totalPages }}</span>
        <button @click="currentPage++" :disabled="currentPage === totalPages" class="btn btn-sm">Next</button>
        <button @click="currentPage = totalPages" :disabled="currentPage === totalPages" class="btn btn-sm">Last</button>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'AzureVMs',
  data() {
    return {
      vms: [],
      loading: false,
      error: null,
      searchQuery: '',
      filterResourceGroup: '',
      filterStatus: '',
      sortField: 'name',
      sortOrder: 'asc',
      currentPage: 1,
      itemsPerPage: 50,
      vmActionLoading: null
    }
  },
  computed: {
    azureConfigured() {
      return this.vms.length > 0 || !this.error || !this.error.includes('not configured')
    },
    filteredVMs() {
      let filtered = this.vms

      // Search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        filtered = filtered.filter(vm =>
          vm.name?.toLowerCase().includes(query) ||
          vm.resource_group?.toLowerCase().includes(query) ||
          vm.location?.toLowerCase().includes(query) ||
          vm.vm_size?.toLowerCase().includes(query)
        )
      }

      // Resource group filter
      if (this.filterResourceGroup) {
        filtered = filtered.filter(vm => vm.resource_group === this.filterResourceGroup)
      }

      // Status filter
      if (this.filterStatus) {
        filtered = filtered.filter(vm =>
          vm.status?.toLowerCase() === this.filterStatus.toLowerCase()
        )
      }

      // Sort
      filtered.sort((a, b) => {
        const aVal = a[this.sortField] || ''
        const bVal = b[this.sortField] || ''
        const comparison = String(aVal).localeCompare(String(bVal), undefined, { numeric: true, sensitivity: 'base' })
        return this.sortOrder === 'asc' ? comparison : -comparison
      })

      return filtered
    },
    paginatedVMs() {
      const start = (this.currentPage - 1) * this.itemsPerPage
      return this.filteredVMs.slice(start, start + this.itemsPerPage)
    },
    totalPages() {
      return Math.ceil(this.filteredVMs.length / this.itemsPerPage)
    },
    uniqueResourceGroups() {
      return [...new Set(this.vms.map(vm => vm.resource_group).filter(Boolean))].sort()
    },
    runningCount() {
      return this.vms.filter(vm => vm.status?.toLowerCase() === 'running').length
    },
    stoppedCount() {
      return this.vms.filter(vm =>
        vm.status?.toLowerCase() === 'stopped' ||
        vm.status?.toLowerCase() === 'deallocated'
      ).length
    }
  },
  methods: {
    async loadData() {
      this.loading = true
      this.error = null
      try {
        const response = await axios.get('/api/azure/vms')
        this.vms = response.data || []
        this.currentPage = 1
      } catch (err) {
        console.error('Failed to load VMs:', err)
        this.error = err.response?.data?.error || err.response?.data?.details || err.message || 'Failed to load virtual machines'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      await this.loadData()
    },
    sortBy(field) {
      if (this.sortField === field) {
        this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortOrder = 'asc'
      }
    },
    getStatusClass(status) {
      if (!status) return 'status-unknown'
      const s = status.toLowerCase()
      if (s === 'running') return 'status-enabled'
      if (s === 'stopped' || s === 'deallocated') return 'status-disabled'
      return 'status-unknown'
    },
    async startVM(vm) {
      if (!confirm(`Are you sure you want to start VM "${vm.name}"?`)) return

      this.vmActionLoading = vm.id
      try {
        await axios.post(`/api/azure/vms/${vm.resource_group}/${vm.name}/start`)
        alert(`VM "${vm.name}" started successfully`)
        await this.refreshData()
      } catch (err) {
        alert(err.response?.data?.error || err.response?.data?.details || 'Failed to start VM')
      } finally {
        this.vmActionLoading = null
      }
    },
    async stopVM(vm) {
      if (!confirm(`Are you sure you want to stop VM "${vm.name}"?`)) return

      this.vmActionLoading = vm.id
      try {
        await axios.post(`/api/azure/vms/${vm.resource_group}/${vm.name}/stop`)
        alert(`VM "${vm.name}" stopped successfully`)
        await this.refreshData()
      } catch (err) {
        alert(err.response?.data?.error || err.response?.data?.details || 'Failed to stop VM')
      } finally {
        this.vmActionLoading = null
      }
    },
    async deleteVM(vm) {
      if (!confirm(`Are you sure you want to DELETE VM "${vm.name}"?\n\nThis action cannot be undone!`)) return

      this.vmActionLoading = vm.id
      try {
        await axios.delete(`/api/azure/vms/${vm.resource_group}/${vm.name}`)
        alert(`VM "${vm.name}" deleted successfully`)
        await this.refreshData()
      } catch (err) {
        alert(err.response?.data?.error || err.response?.data?.details || 'Failed to delete VM')
      } finally {
        this.vmActionLoading = null
      }
    },
    downloadJSON() {
      const dataStr = JSON.stringify(this.vms, null, 2)
      const dataBlob = new Blob([dataStr], { type: 'application/json' })
      const url = URL.createObjectURL(dataBlob)
      const link = document.createElement('a')
      link.href = url
      link.download = `azure-vms-${new Date().toISOString().split('T')[0]}.json`
      link.click()
      URL.revokeObjectURL(url)
    }
  },
  mounted() {
    this.loadData()
  }
}
</script>

<style scoped>
.azure-vms-container {
  padding: 2rem;
  max-width: 1600px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 2rem;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header-icon {
  width: 48px;
  height: 48px;
}

.title-content h1 {
  margin: 0;
  font-size: 2rem;
}

.title-content p {
  margin: 0.25rem 0 0 0;
  color: var(--color-text-secondary);
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

.info-message {
  background: var(--color-warning-bg);
  border: 1px solid var(--color-warning-border);
  border-radius: 8px;
  padding: 1rem;
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
}

.info-message svg {
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  color: var(--color-warning);
}

.loading-container,
.error-container {
  text-align: center;
  padding: 4rem 2rem;
}

.loading-spinner {
  border: 4px solid var(--color-border);
  border-top: 4px solid var(--color-primary);
  border-radius: 50%;
  width: 48px;
  height: 48px;
  animation: spin 1s linear infinite;
  margin: 0 auto 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.summary-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.stat-card {
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 1.5rem;
  text-align: center;
}

.stat-value {
  font-size: 2rem;
  font-weight: bold;
  color: var(--color-primary);
}

.stat-label {
  color: var(--color-text-secondary);
  margin-top: 0.5rem;
}

.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.search-box {
  flex: 1;
  min-width: 300px;
  position: relative;
}

.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  width: 20px;
  height: 20px;
  color: var(--color-text-secondary);
}

.search-input {
  width: 100%;
  padding: 0.75rem 1rem 0.75rem 2.5rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-input-bg);
  color: var(--color-text);
}

.filter-group {
  display: flex;
  gap: 0.5rem;
}

.filter-select {
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-input-bg);
  color: var(--color-text);
}

.table-container {
  overflow-x: auto;
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.data-table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
}

.data-table th {
  padding: 1rem;
  text-align: left;
  background: var(--color-table-header-bg);
  border-bottom: 2px solid var(--color-border);
  font-weight: 600;
}

.sortable {
  cursor: pointer;
  user-select: none;
}

.sortable:hover {
  background: var(--color-table-header-hover);
}

.sort-indicator {
  margin-left: 0.5rem;
}

.data-table td {
  padding: 1rem;
  border-bottom: 1px solid var(--color-border);
}

.table-row:hover {
  background: var(--color-table-row-hover);
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
  font-weight: 500;
}

.status-enabled {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.status-disabled {
  background: var(--color-error-bg);
  color: var(--color-error);
}

.status-unknown {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.875rem;
}

.btn-success {
  background: var(--color-success);
  color: white;
}

.btn-warning {
  background: var(--color-warning);
  color: white;
}

.btn-danger {
  background: var(--color-error);
  color: white;
}

.btn-secondary {
  background: var(--color-secondary);
  color: white;
}

.btn-icon {
  width: 16px;
  height: 16px;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
  margin-top: 2rem;
}
</style>
