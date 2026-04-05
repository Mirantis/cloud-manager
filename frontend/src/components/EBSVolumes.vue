<template>
  <div class="volumes-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M20 8H4V6H20M20 18H4V12H20V18Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>EBS Volumes</h1>
            <p v-if="filterAccount">
              {{ filteredVolumes.length }} volume{{ filteredVolumes.length !== 1 ? 's' : '' }} in {{ selectedAccountName }}
            </p>
            <p v-else>
              {{ volumes.length }} volumes across {{ uniqueAccounts.length }} accounts
            </p>
          </div>
        </div>
        <div class="header-actions">
          <button 
            v-if="canModify"
            @click="deleteAllAvailableVolumes" 
            class="btn btn-danger" 
            :disabled="loading || !filterAccount || availableVolumesCount === 0"
            :title="!filterAccount ? 'Select an account first to enable bulk delete' : (availableVolumesCount === 0 ? 'No available volumes in this account' : `Delete ${availableVolumesCount} available (unused) volume(s) in ${selectedAccountName}`)"
          >
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
            </svg>
            Delete All Available ({{ availableVolumesCount }})
          </button>
          <button 
            v-if="canModify"
            @click="deleteAllOldAvailableVolumes" 
            class="btn btn-danger" 
            :disabled="loading || !filterAccount || oldAvailableVolumesCount === 0"
            :title="!filterAccount ? 'Select an account first to enable bulk delete' : (oldAvailableVolumesCount === 0 ? 'No available volumes older than 6 months in this account' : `Delete ${oldAvailableVolumesCount} available volume(s) older than 6 months in ${selectedAccountName}`)"
          >
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
            </svg>
            Delete Old Available ({{ oldAvailableVolumesCount }})
          </button>
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || volumes.length === 0">
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
      <p>Loading EBS volumes...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load EBS Volumes</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Summary Stats -->
      <div class="summary-stats">
        <div class="stat-card">
          <div class="stat-value">{{ volumes.length }}</div>
          <div class="stat-label">Total Volumes</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ inUseVolumes }}</div>
          <div class="stat-label">In Use</div>
        </div>
        <div class="stat-card stat-card-available">
          <div class="stat-value">{{ availableVolumesCount }}</div>
          <div class="stat-label">Available (Unused)</div>
        </div>
        <div class="stat-card stat-card-old-available">
          <div class="stat-value">{{ oldAvailableVolumesCount }}</div>
          <div class="stat-label">Available (6+ months)</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ filterAccount ? filteredTotalSize : totalSize }} GiB</div>
          <div class="stat-label">{{ filterAccount ? 'Total Size' : 'Total Size' }}</div>
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
            placeholder="Search by volume ID, name, account, region, or type..."
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
          <option value="in-use">In Use</option>
          <option value="creating">Creating</option>
          <option value="deleting">Deleting</option>
        </select>
        <select v-model="filterVolumeType" class="filter-select">
          <option value="">All Types</option>
          <option value="gp3">gp3</option>
          <option value="gp2">gp2</option>
          <option value="io2">io2</option>
          <option value="io1">io1</option>
          <option value="st1">st1</option>
          <option value="sc1">sc1</option>
          <option value="standard">standard</option>
        </select>
      </div>

      <!-- Volumes Table -->
      <div class="table-container">
        <div v-if="filteredVolumes.length === 0" class="empty-state">
          <div class="empty-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M20 8H4V6H20M20 18H4V12H20V18Z"/>
            </svg>
          </div>
          <h4>No volumes found</h4>
          <p>No volumes match the current search and filter criteria.</p>
        </div>
        <table v-else class="volumes-table">
          <thead>
            <tr>
              <th @click="sortBy('volume_id')" class="sortable">
                Volume ID
                <span class="sort-indicator" v-if="sortField === 'volume_id'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
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
              <th @click="sortBy('size')" class="sortable">
                Size (GiB)
                <span class="sort-indicator" v-if="sortField === 'size'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('volume_type')" class="sortable">
                Type
                <span class="sort-indicator" v-if="sortField === 'volume_type'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('monthly_cost')" class="sortable">
                Monthly Cost
                <span class="sort-indicator" v-if="sortField === 'monthly_cost'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('state')" class="sortable">
                Status
                <span class="sort-indicator" v-if="sortField === 'state'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th @click="sortBy('create_time')" class="sortable">
                Created
                <span class="sort-indicator" v-if="sortField === 'create_time'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
              </th>
              <th>Attached To</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="volume in filteredVolumes" :key="volume.volume_id" class="data-row">
              <td><code class="volume-id">{{ volume.volume_id }}</code></td>
              <td>{{ volume.name || '-' }}</td>
              <td>
                <div class="account-info">
                  <div class="account-name">{{ volume.account_name }}</div>
                  <div class="account-id">{{ volume.account_id }}</div>
                </div>
              </td>
              <td>{{ volume.region }}</td>
              <td>{{ volume.size }}</td>
              <td>
                <span class="type-badge">{{ volume.volume_type }}</span>
              </td>
              <td>
                <span class="cost-value">{{ formatCurrency(volume.monthly_cost || 0) }}</span>
              </td>
              <td>
                <span :class="['state-badge', `state-${volume.state}`]">
                  {{ volume.state }}
                </span>
              </td>
              <td>
                <span class="date-text">{{ formatDate(volume.create_time) }}</span>
              </td>
              <td>
                <div v-if="volume.attachments && volume.attachments.length > 0" class="attachments">
                  <div v-for="(att, idx) in volume.attachments" :key="idx" class="attachment">
                    <code>{{ att.instance_id }}</code>
                    <span class="device">{{ att.device }}</span>
                  </div>
                </div>
                <span v-else class="not-attached">Not attached</span>
              </td>
              <td>
                <div class="action-buttons" v-if="canModify">
                  <!-- Available volumes: Delete button -->
                  <button
                    v-if="volume.state === 'available'"
                    @click="deleteVolume(volume)"
                    class="action-btn action-btn-delete"
                    :disabled="loading"
                    title="Delete volume"
                  >
                    <svg viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                    </svg>
                    Delete
                  </button>
                  <!-- In-use volumes: Terminate instance(s) and Delete volume buttons -->
                  <template v-else-if="volume.state === 'in-use'">
                    <button
                      v-for="attachment in (volume.attachments || [])"
                      :key="attachment.instance_id"
                      @click="terminateInstance(volume, attachment.instance_id)"
                      class="action-btn action-btn-terminate"
                      :disabled="loading"
                      :title="`Terminate instance ${attachment.instance_id}`"
                    >
                      <svg viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4C16.41,4 20,7.59 20,12C20,16.41 16.41,20 12,20C7.59,20 4,16.41 4,12C4,7.59 7.59,4 12,4M11,17V16H9V14H13V13H10A1,1 0 0,1 9,12V9A1,1 0 0,1 10,8H14V10H12V11H15A1,1 0 0,1 16,12V16A1,1 0 0,1 15,17H11Z"/>
                      </svg>
                      Terminate {{ attachment.instance_id.substring(0, 8) }}...
                    </button>
                    <button
                      @click="deleteVolume(volume)"
                      class="action-btn action-btn-delete"
                      :disabled="loading"
                      title="Delete volume (will detach from instances first)"
                    >
                      <svg viewBox="0 0 24 24" fill="currentColor">
                        <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                      </svg>
                      Delete Volume
                    </button>
                  </template>
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
  name: 'EBSVolumes',
  data() {
    return {
      volumes: [],
      loading: true,
      error: null,
      searchQuery: '',
      filterAccount: '',
      filterRegion: '',
      filterState: '',
      filterVolumeType: '',
      sortField: 'create_time',
      sortDirection: 'desc'
    }
  },
  computed: {
    uniqueAccounts() {
      const accounts = new Map()
      this.volumes.forEach(v => {
        if (!accounts.has(v.account_id)) {
          accounts.set(v.account_id, { id: v.account_id, name: v.account_name })
        }
      })
      return Array.from(accounts.values()).sort((a, b) => a.name.localeCompare(b.name))
    },
    uniqueRegions() {
      return [...new Set(this.volumes.map(v => v.region))].sort()
    },
    inUseVolumes() {
      return this.volumes.filter(v => v.state === 'in-use').length
    },
    availableVolumesCount() {
      // If account filter is set, only count volumes from that account
      let volumesToCheck = this.volumes
      if (this.filterAccount) {
        volumesToCheck = volumesToCheck.filter(v => v.account_id === this.filterAccount)
      }
      return volumesToCheck.filter(v => v.state === 'available').length
    },
    availableVolumes() {
      // Get all available volumes, optionally filtered by account
      let volumesToCheck = this.volumes
      if (this.filterAccount) {
        volumesToCheck = volumesToCheck.filter(v => v.account_id === this.filterAccount)
      }
      return volumesToCheck.filter(v => v.state === 'available')
    },
    oldAvailableVolumesCount() {
      // Count available volumes older than 6 months, optionally filtered by account
      let volumesToCheck = this.volumes
      if (this.filterAccount) {
        volumesToCheck = volumesToCheck.filter(v => v.account_id === this.filterAccount)
      }
      const sixMonthsAgo = new Date()
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
      return volumesToCheck.filter(v => {
        if (v.state !== 'available') return false
        if (!v.create_time) return false
        const createdDate = new Date(v.create_time)
        return createdDate < sixMonthsAgo
      }).length
    },
    oldAvailableVolumes() {
      // Get available volumes older than 6 months, optionally filtered by account
      let volumesToCheck = this.volumes
      if (this.filterAccount) {
        volumesToCheck = volumesToCheck.filter(v => v.account_id === this.filterAccount)
      }
      const sixMonthsAgo = new Date()
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
      return volumesToCheck.filter(v => {
        if (v.state !== 'available') return false
        if (!v.create_time) return false
        const createdDate = new Date(v.create_time)
        return createdDate < sixMonthsAgo
      })
    },
    totalSize() {
      return this.volumes.reduce((sum, v) => sum + v.size, 0)
    },
    filteredTotalSize() {
      return this.filteredVolumes.reduce((sum, v) => sum + v.size, 0)
    },
    totalMonthlyCost() {
      return this.volumes.reduce((sum, v) => sum + (v.monthly_cost || 0), 0)
    },
    filteredTotalMonthlyCost() {
      return this.filteredVolumes.reduce((sum, v) => sum + (v.monthly_cost || 0), 0)
    },
    filteredUniqueRegions() {
      return [...new Set(this.filteredVolumes.map(v => v.region))].sort()
    },
    selectedAccountName() {
      if (!this.filterAccount) return ''
      const account = this.uniqueAccounts.find(a => a.id === this.filterAccount)
      return account ? `${account.name} (${account.id})` : this.filterAccount
    },
    filteredVolumes() {
      let result = this.volumes

      // Apply search filter
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        result = result.filter(v =>
          v.volume_id.toLowerCase().includes(query) ||
          v.name?.toLowerCase().includes(query) ||
          v.account_name.toLowerCase().includes(query) ||
          v.account_id.toLowerCase().includes(query) ||
          v.region.toLowerCase().includes(query) ||
          v.volume_type.toLowerCase().includes(query)
        )
      }

      // Apply filters
      if (this.filterAccount) {
        result = result.filter(v => v.account_id === this.filterAccount)
      }
      if (this.filterRegion) {
        result = result.filter(v => v.region === this.filterRegion)
      }
      if (this.filterState) {
        result = result.filter(v => v.state === this.filterState)
      }
      if (this.filterVolumeType) {
        result = result.filter(v => v.volume_type === this.filterVolumeType)
      }

      // Apply sorting
      result.sort((a, b) => {
        let aVal = a[this.sortField]
        let bVal = b[this.sortField]

        if (this.sortField === 'create_time') {
          aVal = new Date(aVal)
          bVal = new Date(bVal)
        } else if (this.sortField === 'monthly_cost' || this.sortField === 'size') {
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
        const response = await axios.get('/api/ebs-volumes')
        this.volumes = response.data || []
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load EBS volumes'
      } finally {
        this.loading = false
      }
    },
    async refreshData() {
      if (this.canModify) {
        try {
          await axios.post('/api/cache/ebs-volumes/invalidate')
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
        this.sortDirection = field === 'create_time' ? 'desc' : 'asc'
      }
    },
    async detachVolume(volume) {
      const attachmentInfo = volume.attachments && volume.attachments.length > 0
        ? volume.attachments.map(att => `  - Instance: ${att.instance_id} (${att.device})`).join('\n')
        : 'Unknown instances'

      const confirmed = confirm(
        `Are you sure you want to detach volume ${volume.volume_id}?\n\n` +
        `Attached to:\n${attachmentInfo}\n\n` +
        `Size: ${volume.size} GiB\n` +
        `Type: ${volume.volume_type}\n\n` +
        `After detaching, you will be able to delete the volume.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        const url = `/api/accounts/${volume.account_id}/regions/${volume.region}/volumes/${volume.volume_id}/detach`
        await axios.post(url)

        // Reload data - the cache has been updated
        await this.loadData()

        alert(`Volume ${volume.volume_id} detached successfully`)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to detach volume'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to detach volume:', err)
      } finally {
        this.loading = false
      }
    },
    async deleteVolume(volume) {
      let confirmMessage = `Are you sure you want to delete volume ${volume.volume_id}?\n\n` +
        `Size: ${volume.size} GiB\n` +
        `Type: ${volume.volume_type}\n\n`
      
      if (volume.state === 'in-use' && volume.attachments && volume.attachments.length > 0) {
        confirmMessage += `WARNING: This volume is attached to:\n` +
          volume.attachments.map(att => `  - Instance: ${att.instance_id} (${att.device})`).join('\n') +
          `\n\nThe volume will be detached first, then deleted.\n\n`
      }
      
      confirmMessage += `WARNING: This action cannot be undone.`
      
      const confirmed = confirm(confirmMessage)
      if (!confirmed) return

      try {
        this.loading = true
        const url = `/api/accounts/${volume.account_id}/regions/${volume.region}/volumes/${volume.volume_id}`
        await axios.delete(url)

        // Reload data - the cache has been updated
        await this.loadData()

        alert(`Volume ${volume.volume_id} deleted successfully`)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete volume'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete volume:', err)
      } finally {
        this.loading = false
      }
    },
    async terminateInstance(volume, instanceId) {
      const confirmed = confirm(
        `Are you sure you want to terminate instance ${instanceId}?\n\n` +
        `This instance is attached to volume ${volume.volume_id}.\n\n` +
        `WARNING: Terminating the instance will:\n` +
        `- Stop and terminate the EC2 instance\n` +
        `- Detach the volume automatically\n` +
        `- This action cannot be undone\n\n` +
        `After termination, you can delete the volume separately.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        const url = `/api/accounts/${volume.account_id}/regions/${volume.region}/instances/${instanceId}/terminate`
        await axios.post(url)

        // Reload data - the cache has been updated
        await this.loadData()

        alert(`Instance ${instanceId} terminated successfully`)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to terminate instance'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to terminate instance:', err)
      } finally {
        this.loading = false
      }
    },
    async deleteAllAvailableVolumes() {
      const availableVols = this.availableVolumes
      if (availableVols.length === 0) {
        alert('No available volumes to delete')
        return
      }

      const accountFilter = this.filterAccount
      const accountName = accountFilter 
        ? this.volumes.find(v => v.account_id === accountFilter)?.account_name || accountFilter
        : 'all accounts'

      const totalSize = availableVols.reduce((sum, v) => sum + v.size, 0)
      const volumesList = availableVols.slice(0, 10).map(v => `  - ${v.volume_id} (${v.size} GiB, ${v.region})`).join('\n')
      const moreText = availableVols.length > 10 ? `\n  ... and ${availableVols.length - 10} more` : ''

      const confirmed = confirm(
        `Are you sure you want to delete ALL ${availableVols.length} available volume(s)?\n\n` +
        `Account: ${accountName}\n` +
        `Total size: ${totalSize} GiB\n\n` +
        `Volumes to delete:\n${volumesList}${moreText}\n\n` +
        `WARNING: This action cannot be undone!\n` +
        `This will permanently delete all available volumes${accountFilter ? ' in this account' : ' across all accounts'}.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        let successCount = 0
        let failCount = 0
        const errors = []

        // Delete volumes one by one
        for (const volume of availableVols) {
          try {
            const url = `/api/accounts/${volume.account_id}/regions/${volume.region}/volumes/${volume.volume_id}`
            await axios.delete(url)
            successCount++
            
            // Small delay to avoid rate limiting
            await new Promise(resolve => setTimeout(resolve, 100))
          } catch (err) {
            failCount++
            const errorMsg = err.response?.data?.error || err.message || 'Unknown error'
            errors.push(`${volume.volume_id}: ${errorMsg}`)
            console.error(`Failed to delete volume ${volume.volume_id}:`, err)
          }
        }

        // Reload data
        await this.loadData()

        // Show results
        let message = `Deleted ${successCount} volume(s) successfully`
        if (failCount > 0) {
          message += `\n\nFailed to delete ${failCount} volume(s):\n${errors.slice(0, 5).join('\n')}`
          if (errors.length > 5) {
            message += `\n... and ${errors.length - 5} more errors (check console)`
          }
        }
        alert(message)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete volumes'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete volumes:', err)
      } finally {
        this.loading = false
      }
    },
    async deleteAllOldAvailableVolumes() {
      const oldAvailableVols = this.oldAvailableVolumes
      if (oldAvailableVols.length === 0) {
        alert('No available volumes older than 6 months to delete')
        return
      }

      const accountFilter = this.filterAccount
      const accountName = accountFilter 
        ? this.volumes.find(v => v.account_id === accountFilter)?.account_name || accountFilter
        : 'all accounts'

      const totalSize = oldAvailableVols.reduce((sum, v) => sum + v.size, 0)
      const volumesList = oldAvailableVols.slice(0, 10).map(v => {
        const age = this.getAgeInMonths(v.create_time)
        return `  - ${v.volume_id} (${v.size} GiB, ${v.region}, ${age} months old)`
      }).join('\n')
      const moreText = oldAvailableVols.length > 10 ? `\n  ... and ${oldAvailableVols.length - 10} more` : ''

      const confirmed = confirm(
        `Are you sure you want to delete ALL ${oldAvailableVols.length} available volume(s) older than 6 months?\n\n` +
        `Account: ${accountName}\n` +
        `Total size: ${totalSize} GiB\n\n` +
        `Volumes to delete:\n${volumesList}${moreText}\n\n` +
        `WARNING: This action cannot be undone!\n` +
        `This will permanently delete all available volumes older than 6 months${accountFilter ? ' in this account' : ' across all accounts'}.`
      )
      if (!confirmed) return

      try {
        this.loading = true
        let successCount = 0
        let failCount = 0
        const errors = []

        // Delete volumes one by one
        for (const volume of oldAvailableVols) {
          try {
            const url = `/api/accounts/${volume.account_id}/regions/${volume.region}/volumes/${volume.volume_id}`
            await axios.delete(url)
            successCount++
            
            // Small delay to avoid rate limiting
            await new Promise(resolve => setTimeout(resolve, 100))
          } catch (err) {
            failCount++
            const errorMsg = err.response?.data?.error || err.message || 'Unknown error'
            errors.push(`${volume.volume_id}: ${errorMsg}`)
            console.error(`Failed to delete volume ${volume.volume_id}:`, err)
          }
        }

        // Reload data
        await this.loadData()

        // Show results
        let message = `Deleted ${successCount} volume(s) successfully`
        if (failCount > 0) {
          message += `\n\nFailed to delete ${failCount} volume(s):\n${errors.slice(0, 5).join('\n')}`
          if (errors.length > 5) {
            message += `\n... and ${errors.length - 5} more errors (check console)`
          }
        }
        alert(message)
      } catch (err) {
        const errorMsg = err.response?.data?.error || 'Failed to delete volumes'
        alert(`Error: ${errorMsg}`)
        console.error('Failed to delete volumes:', err)
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
      try {
        const exportData = {
          exported_at: new Date().toISOString(),
          total_volumes: this.filteredVolumes.length,
          total_size_gib: this.totalSize,
          volumes: this.filteredVolumes
        }

        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)
        const exportFileDefaultName = `aws-ebs-volumes-${new Date().toISOString().split('T')[0]}.json`

        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (error) {
        console.error('Failed to download JSON:', error)
        alert('Failed to download JSON file')
      }
    },
    formatDate(dateString) {
      if (!dateString) return '-'
      const date = new Date(dateString)
      const now = new Date()
      const diffMs = now - date
      const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

      // Show relative time for recent dates
      if (diffDays === 0) {
        const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
        if (diffHours === 0) {
          const diffMins = Math.floor(diffMs / (1000 * 60))
          return diffMins <= 1 ? 'Just now' : `${diffMins} mins ago`
        }
        return diffHours === 1 ? '1 hour ago' : `${diffHours} hours ago`
      } else if (diffDays === 1) {
        return 'Yesterday'
      } else if (diffDays < 7) {
        return `${diffDays} days ago`
      }

      // Show absolute date for older dates
      return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
      })
    },
    formatCurrency(amount) {
      return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD',
        minimumFractionDigits: 2,
        maximumFractionDigits: 2
      }).format(amount)
    }
  }
}
</script>

<style scoped>
.volumes-container {
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

.btn-danger {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: white;
}

.btn-danger:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
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

.stat-card-available {
  border: 2px solid rgba(16, 185, 129, 0.3);
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.05) 0%, rgba(16, 185, 129, 0.02) 100%);
}

.stat-card-available .stat-value {
  color: #10b981;
}

.stat-card-old-available {
  border: 2px solid rgba(239, 68, 68, 0.3);
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.05) 0%, rgba(239, 68, 68, 0.02) 100%);
}

.stat-card-old-available .stat-value {
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

.volumes-table {
  width: 100%;
  min-width: 1000px;
  border-collapse: collapse;
}

.volumes-table thead {
  background: var(--color-bg-secondary);
}

.volumes-table th {
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

.volumes-table th.sortable:hover {
  color: var(--color-primary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-primary);
}

.volumes-table tbody tr {
  border-top: 1px solid var(--color-border);
}

.volumes-table tbody tr:hover {
  background: var(--color-bg-hover);
}

.volumes-table td {
  padding: 1rem 1.5rem;
  color: var(--color-text-primary);
}

.volume-id {
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

.type-badge {
  display: inline-block;
  padding: 0.35rem 0.75rem;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 500;
  background: rgba(99, 102, 241, 0.1);
  color: #6366f1;
}

.state-badge {
  display: inline-block;
  padding: 0.35rem 0.75rem;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 500;
  text-transform: capitalize;
}

.state-available {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.state-in-use {
  background: rgba(99, 102, 241, 0.1);
  color: #6366f1;
}

.state-creating {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.state-deleting {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.attachments {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.attachment {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.attachment code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.85rem;
  background: var(--color-bg-secondary);
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
}

.device {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
}

.not-attached {
  color: var(--color-text-secondary);
  font-style: italic;
}

.cost-value {
  font-weight: 600;
  color: var(--color-text-primary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875rem;
}

.date-text {
  font-size: 0.9rem;
  color: var(--color-text-secondary);
  white-space: nowrap;
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

.action-btn-detach {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: white;
}

.action-btn-detach:hover:not(:disabled) {
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

  .volumes-table {
    font-size: 0.85rem;
  }

  .volumes-table th,
  .volumes-table td {
    padding: 0.75rem 1rem;
  }
}
</style>
