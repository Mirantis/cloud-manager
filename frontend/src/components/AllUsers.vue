<template>
  <div class="all-users-container">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2C13.1 2 14 2.9 14 4C14 5.1 13.1 6 12 6C10.9 6 10 5.1 10 4C10 2.9 10.9 2 12 2Z"/>
              <path d="M21 9V7L15 1H5C3.89 1 3 1.89 3 3V19C3 20.11 3.89 21 5 21H11V19H5V3H13V9H21Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>All Organization Users</h1>
            <p>{{ allUsers.length }} users across {{ accounts.length }} accounts</p>
          </div>
        </div>
        <div class="header-actions">
          <button @click="downloadUsersJSON" class="btn btn-success" :disabled="loading || allUsers.length === 0">
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
      <p>Loading organization users...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load Users</h3>
      <p>{{ error }}</p>
      <button @click="refreshData" class="btn btn-primary">Try Again</button>
    </div>

    <!-- Main Content -->
    <div v-else class="main-content">
      <!-- Search and Filter -->
      <div class="user-filters">
        <div class="search-box">
          <svg class="search-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M9.5,3A6.5,6.5 0 0,1 16,9.5C16,11.11 15.41,12.59 14.44,13.73L14.71,14H15.5L20.5,19L19,20.5L14,15.5V14.71L13.73,14.44C12.59,15.41 11.11,16 9.5,16A6.5,6.5 0 0,1 3,9.5A6.5,6.5 0 0,1 9.5,3M9.5,5C7,5 5,7 5,9.5C5,12 7,14 9.5,14C12,14 14,12 14,9.5C14,7 12,5 9.5,5Z"/>
          </svg>
          <input
            v-model="userSearchQuery"
            type="text"
            placeholder="Search users, accounts, or access key IDs..."
            class="search-input"
          />
        </div>
        <div class="filter-buttons">
          <button 
            @click="userFilter = 'all'" 
            :class="['filter-btn', { active: userFilter === 'all' }]"
          >
            All ({{ allUsers.length }})
          </button>
          <button 
            @click="userFilter = 'withKeys'" 
            :class="['filter-btn', { active: userFilter === 'withKeys' }]"
          >
            With Keys ({{ usersWithKeys }})
          </button>
          <button 
            @click="userFilter = 'oldKeys'" 
            :class="['filter-btn', { active: userFilter === 'oldKeys' }]"
          >
            Old Keys ({{ usersWithOldKeys }})
          </button>
          <button 
            @click="userFilter = 'withPasswords'" 
            :class="['filter-btn', { active: userFilter === 'withPasswords' }]"
          >
            With Passwords ({{ usersWithPasswords }})
          </button>
          <button 
            @click="userFilter = 'inactive'" 
            :class="['filter-btn', { active: userFilter === 'inactive' }]"
          >
            Inactive (6+ months) ({{ inactiveUsersCount }})
          </button>
        </div>
        <div class="account-actions">
          <select v-model="selectedAccount" class="account-select">
            <option value="">All Accounts</option>
            <option v-for="account in accounts" :key="account.id" :value="account.id">
              {{ account.name }} ({{ account.id }})
            </option>
          </select>
          <button 
            v-if="canModify"
            @click="deleteInactiveUsers" 
            class="btn btn-danger" 
            :disabled="!selectedAccount || deletingInactive"
            :title="selectedAccount ? `Delete inactive users from ${getAccountName(selectedAccount)}` : 'Select an account first'"
          >
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
            </svg>
            {{ deletingInactive ? 'Deleting...' : 'Delete Inactive Users' }}
          </button>
        </div>
      </div>

      <!-- Users Table -->
      <div class="users-table-container">
        <table class="users-table">
          <thead>
            <tr>
              <th @click="sortBy('username')" class="sortable">
                User
                <span v-if="sortField === 'username'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th @click="sortBy('accountName')" class="sortable">
                Account
                <span v-if="sortField === 'accountName'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th>Password</th>
              <th>Keys</th>
              <th>Old Keys</th>
              <th @click="sortBy('last_used')" class="sortable">
                Last Key Use
                <span v-if="sortField === 'last_used'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th @click="sortBy('create_date')" class="sortable">
                Created
                <span v-if="sortField === 'create_date'" class="sort-indicator">
                  {{ sortDirection === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in filteredUsers" :key="`${user.accountId}-${user.username}`" class="user-row clickable" @click="viewUser(user.accountId, user.username)">
              <td>
                <div class="user-info">
                  <div class="user-name">{{ user.username }}</div>
                  <div class="user-id">{{ user.user_id }}</div>
                </div>
              </td>
              <td>
                <div class="account-info">
                  <div class="account-name">{{ user.accountName }}</div>
                  <div class="account-id">{{ user.accountId }}</div>
                </div>
              </td>
              <td>
                <div class="password-cell">
                  <span :class="['password-status', user.password_set ? 'has-password' : 'no-password']">
                    <svg class="status-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path v-if="user.password_set" d="M21,7L9,19L3.5,13.5L4.91,12.09L9,16.17L19.59,5.59L21,7Z"/>
                      <path v-else d="M19,13H5V11H19V13Z"/>
                    </svg>
                    {{ user.password_set ? 'Yes' : 'No' }}
                  </span>
                  <div class="password-actions" v-if="user.password_set && canModify">
                    <button
                      @click.stop="rotateUserPassword(user.accountId, user.username)"
                      class="btn btn-primary btn-xs password-rotate-btn"
                      :disabled="actionLoading[`${user.accountId}-${user.username}-rotate-password`]"
                      title="Rotate console password"
                    >
                      <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
                      </svg>
                    </button>
                    <button
                      @click.stop="removeUserPassword(user.accountId, user.username)"
                      class="btn btn-warning btn-xs password-remove-btn"
                      :disabled="actionLoading[`${user.accountId}-${user.username}-password`]"
                      title="Remove console password"
                    >
                      <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                      </svg>
                    </button>
                  </div>
                </div>
              </td>
              <td>
                <span class="key-count">{{ user.access_keys?.length || 0 }}</span>
              </td>
              <td>
                <span :class="['old-key-count', { warning: getOldKeyCount(user) > 0 }]">
                  {{ getOldKeyCount(user) }}
                </span>
              </td>
              <td>
                <div class="last-use-info">
                  <div v-if="getMostRecentKeyUse(user)" class="last-use-data">
                    <div class="date-value">{{ formatDate(getMostRecentKeyUse(user)) }}</div>
                    <div class="date-relative">{{ formatRelativeDate(getMostRecentKeyUse(user)) }}</div>
                  </div>
                  <div v-else-if="hasActiveKeys(user)" class="never-used">
                    <span class="never-used-text">Never used</span>
                  </div>
                  <div v-else class="no-keys">
                    <span class="no-keys-text">No keys</span>
                  </div>
                </div>
              </td>
              <td>
                <div class="date-info">
                  <div class="date-value">{{ formatDate(user.create_date) }}</div>
                  <div class="date-relative">{{ formatRelativeDate(user.create_date) }}</div>
                </div>
              </td>
              <td>
                <div class="user-actions">
                  <button 
                    v-if="canModify"
                    @click.stop="deleteUser(user.accountId, user.username)"
                    class="btn btn-sm btn-danger"
                    :disabled="actionLoading[`${user.accountId}-${user.username}`]"
                  >
                    <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
                    </svg>
                    {{ actionLoading[`${user.accountId}-${user.username}`] ? 'Deleting...' : 'Delete' }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
        
        <div v-if="filteredUsers.length === 0" class="empty-results">
          <div class="empty-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2C13.1 2 14 2.9 14 4C14 5.1 13.1 6 12 6C10.9 6 10 5.1 10 4C10 2.9 10.9 2 12 2Z"/>
            </svg>
          </div>
          <h4>No users found</h4>
          <p>No users match the current search and filter criteria.</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'AllUsers',
  data() {
    return {
      accounts: [],
      allUsers: [],
      loading: true,
      error: null,
      userSearchQuery: '',
      userFilter: 'all',
      actionLoading: {},
      sortField: 'username',
      sortDirection: 'asc',
      selectedAccount: '',
      deletingInactive: false
    }
  },
  computed: {
    usersWithKeys() {
      return this.allUsers?.filter(user => (user.access_keys?.length || 0) > 0)?.length || 0
    },
    usersWithOldKeys() {
      const thirtyDaysAgo = new Date()
      thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)
      
      return this.allUsers?.filter(user => {
        return user.access_keys?.some(key => {
          const keyDate = new Date(key.create_date)
          return keyDate < thirtyDaysAgo
        })
      })?.length || 0
    },
    usersWithPasswords() {
      return this.allUsers?.filter(user => user.password_set)?.length || 0
    },
    inactiveUsersCount() {
      const sixMonthsAgo = new Date()
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
      
      return this.allUsers?.filter(user => {
        // Check password last used
        if (user.password_set && user.password_last_used) {
          const passwordLastUsed = new Date(user.password_last_used)
          if (passwordLastUsed > sixMonthsAgo) {
            return false
          }
        }
        
        // Check access keys last used
        if (user.access_keys && user.access_keys.length > 0) {
          for (const key of user.access_keys) {
            if (key.status === 'Active' && key.last_used_date) {
              const lastUsed = new Date(key.last_used_date)
              if (lastUsed > sixMonthsAgo) {
                return false
              }
            }
          }
        }
        
        // User is inactive if no password or password not used in 6 months,
        // and no active keys or all keys unused for 6+ months
        return true
      })?.length || 0
    },
    inactiveUsers() {
      const sixMonthsAgo = new Date()
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
      
      return this.allUsers?.filter(user => {
        // Check password last used
        if (user.password_set && user.password_last_used) {
          const passwordLastUsed = new Date(user.password_last_used)
          if (passwordLastUsed > sixMonthsAgo) {
            return false
          }
        }
        
        // Check access keys last used
        if (user.access_keys && user.access_keys.length > 0) {
          for (const key of user.access_keys) {
            if (key.status === 'Active' && key.last_used_date) {
              const lastUsed = new Date(key.last_used_date)
              if (lastUsed > sixMonthsAgo) {
                return false
              }
            }
          }
        }
        
        return true
      }) || []
    },
    enrichedUsers() {
      return this.allUsers?.map(user => {
        const account = this.accounts?.find(acc => acc.id === user.accountId)
        return {
          ...user,
          accountName: account?.name || 'Unknown Account'
        }
      }) || []
    },
    filteredUsers() {
      let users = this.enrichedUsers

      // Apply account filter first
      if (this.selectedAccount) {
        users = users.filter(user => user.accountId === this.selectedAccount)
      }

      // Apply search filter
      if (this.userSearchQuery) {
        const query = this.userSearchQuery.toLowerCase()
        users = users.filter(user =>
          user.username.toLowerCase().includes(query) ||
          user.user_id.toLowerCase().includes(query) ||
          user.accountName.toLowerCase().includes(query) ||
          user.accountId.toLowerCase().includes(query) ||
          (user.access_keys && user.access_keys.some(key =>
            key.access_key_id.toLowerCase().includes(query)
          ))
        )
      }

      // Apply category filter
      const thirtyDaysAgo = new Date()
      thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)

      switch (this.userFilter) {
        case 'withKeys':
          users = users.filter(user => (user.access_keys?.length || 0) > 0)
          break
        case 'oldKeys':
          users = users.filter(user =>
            user.access_keys?.some(key => {
              const keyDate = new Date(key.create_date)
              return keyDate < thirtyDaysAgo
            })
          )
          break
        case 'withPasswords':
          users = users.filter(user => user.password_set)
          break
        case 'inactive':
          users = users.filter(user => {
            // Apply inactive filter only to current filtered users
            const sixMonthsAgo = new Date()
            sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6)
            
            // Check password last used
            if (user.password_set && user.password_last_used) {
              const passwordLastUsed = new Date(user.password_last_used)
              if (passwordLastUsed > sixMonthsAgo) {
                return false
              }
            }
            
            // Check access keys last used
            if (user.access_keys && user.access_keys.length > 0) {
              for (const key of user.access_keys) {
                if (key.status === 'Active' && key.last_used_date) {
                  const lastUsed = new Date(key.last_used_date)
                  if (lastUsed > sixMonthsAgo) {
                    return false
                  }
                }
              }
            }
            
            return true
          })
          break
        default:
          // users remains as enrichedUsers
      }

      // Apply sorting
      users.sort((a, b) => {
        let aVal, bVal

        switch (this.sortField) {
          case 'username':
            aVal = a.username.toLowerCase()
            bVal = b.username.toLowerCase()
            break
          case 'accountName':
            aVal = a.accountName.toLowerCase()
            bVal = b.accountName.toLowerCase()
            break
          case 'create_date':
            aVal = new Date(a.create_date)
            bVal = new Date(b.create_date)
            break
          case 'last_used':
            aVal = this.getMostRecentKeyUse(a) ? new Date(this.getMostRecentKeyUse(a)) : new Date(0)
            bVal = this.getMostRecentKeyUse(b) ? new Date(this.getMostRecentKeyUse(b)) : new Date(0)
            break
          default:
            return 0
        }

        if (this.sortDirection === 'asc') {
          return aVal < bVal ? -1 : (aVal > bVal ? 1 : 0)
        } else {
          return aVal > bVal ? -1 : (aVal < bVal ? 1 : 0)
        }
      })

      return users
    }
  },
  async mounted() {
    // Initial load when component is first created
    await this.loadData()
  },
  // activated is called when a kept-alive component is re-activated
  activated() {
    // Check if we're returning from a user deletion (query params from UserDetail)
    if (this.$route.query.deletedUser && this.$route.query.deletedFromAccount) {
      const deletedUser = this.$route.query.deletedUser
      const deletedFromAccount = this.$route.query.deletedFromAccount
      console.log(`User ${deletedUser} was deleted from account ${deletedFromAccount}, updating local cache`)
      
      // Remove the deleted user from local state - no need to reload from server
      this.allUsers = this.allUsers.filter(
        user => !(user.accountId === deletedFromAccount && user.username === deletedUser)
      )
      
      // Clear the query params from URL
      this.$router.replace({ path: '/', query: {} })
    }
    // Otherwise, keep using cached data - no reload needed
  },
  methods: {
    async loadData() {
      try {
        this.loading = true
        this.error = null
        
        // Load accounts and all users in parallel
        const [accountsResponse, usersResponse] = await Promise.all([
          axios.get('/api/accounts'),
          axios.get('/api/users') // Single endpoint that loads all users in parallel on server
        ])
        
        this.accounts = accountsResponse.data
        this.allUsers = usersResponse.data || []
        
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load data'
      } finally {
        this.loading = false
      }
    },
    
    async refreshData() {
      if (this.canModify) {
        try {
          await axios.post('/api/cache/clear')
        } catch (error) {
          console.warn('Failed to clear cache:', error)
        }
      }
      
      this.allUsers = []
      await this.loadData()
    },

    async backgroundRefresh() {
      // If we don't have any data, show loading and do full refresh
      if (!this.allUsers || this.allUsers.length === 0) {
        console.log('No cached data available, doing full refresh')
        await this.refreshData()
        return
      }

      try {
        console.log('Refreshing data in background while showing cached data')
        
        if (this.canModify) {
          try {
            await axios.post('/api/cache/clear')
          } catch (e) {
            console.warn('Failed to clear cache:', e)
          }
        }
        
        // Load all data in parallel
        const [accountsResponse, usersResponse] = await Promise.all([
          axios.get('/api/accounts'),
          axios.get('/api/users')
        ])
        this.accounts = accountsResponse.data
        this.allUsers = usersResponse.data || []
        
        console.log('Background refresh completed - data updated')
        
      } catch (err) {
        console.warn('Background refresh failed:', err)
        // Keep showing the cached data even if refresh fails
      }
    },
    
    getOldKeyCount(user) {
      if (!user.access_keys || user.access_keys.length === 0) return 0

      const thirtyDaysAgo = new Date()
      thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)

      return user.access_keys.filter(key => {
        // Consider a key old if it was created more than 30 days ago AND either:
        // 1. It has never been used, OR
        // 2. It was last used more than 30 days ago
        const keyDate = new Date(key.create_date)
        const isOldByCreate = keyDate < thirtyDaysAgo

        if (!isOldByCreate) return false

        // If key has never been used, consider it old if created > 30 days ago
        if (!key.last_used_date) return true

        // If key has been used, check if last use was > 30 days ago
        const lastUsedDate = new Date(key.last_used_date)
        return lastUsedDate < thirtyDaysAgo
      }).length
    },

    getMostRecentKeyUse(user) {
      if (!user.access_keys || user.access_keys.length === 0) return null

      let mostRecent = null
      for (const key of user.access_keys) {
        if (key.last_used_date) {
          const lastUsed = new Date(key.last_used_date)
          if (!mostRecent || lastUsed > mostRecent) {
            mostRecent = lastUsed
          }
        }
      }
      return mostRecent
    },

    hasActiveKeys(user) {
      return user.access_keys && user.access_keys.length > 0 &&
             user.access_keys.some(key => key.status === 'Active')
    },

    sortBy(field) {
      if (this.sortField === field) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortDirection = field === 'create_date' || field === 'last_used' ? 'desc' : 'asc'
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
    
    viewUser(accountId, username) {
      this.$router.push(`/accounts/${accountId}/users/${username}`)
    },
    
    getAccountName(accountId) {
      const account = this.accounts?.find(acc => acc.id === accountId)
      return account?.name || accountId
    },
    
    async deleteInactiveUsers() {
      if (!this.selectedAccount) {
        alert('Please select an account first')
        return
      }
      
      const accountName = this.getAccountName(this.selectedAccount)
      const inactiveCount = this.inactiveUsers.filter(u => u.accountId === this.selectedAccount).length
      
      if (inactiveCount === 0) {
        alert(`No inactive users found in account ${accountName}`)
        return
      }
      
      const confirmMessage = `Are you sure you want to DELETE ${inactiveCount} inactive user(s) from account "${accountName}"?\n\nThis will permanently delete users that have been inactive for 6+ months.\n\nThis action cannot be undone!`
      
      if (!confirm(confirmMessage)) return
      
      this.deletingInactive = true
      
      try {
        const response = await axios.post(`/api/accounts/${this.selectedAccount}/users/inactive/delete`)
        const deletedUsers = response.data.deleted_users || []
        const failedCount = response.data.failed_users?.length || 0
        
        // Remove deleted users from local state without full refresh
        if (deletedUsers.length > 0) {
          const deletedSet = new Set(deletedUsers)
          this.allUsers = this.allUsers.filter(
            user => !(user.accountId === this.selectedAccount && deletedSet.has(user.username))
          )
        }
        
        let message = `Successfully deleted ${deletedUsers.length} inactive user(s) from ${accountName}`
        if (failedCount > 0) {
          message += `\n\nFailed to delete ${failedCount} user(s):\n${response.data.failed_users.join('\n')}`
        }
        alert(message)
        
        // Invalidate cache for the account to ensure fresh data on next load
        try {
          await axios.post(`/api/cache/accounts/${this.selectedAccount}/invalidate`)
        } catch (cacheErr) {
          console.warn('Failed to invalidate account cache:', cacheErr)
        }
      } catch (error) {
        console.error('Failed to delete inactive users:', error)
        alert(error.response?.data?.error || error.response?.data?.details || 'Failed to delete inactive users. Please try again.')
      } finally {
        this.deletingInactive = false
      }
    },

    async deleteUser(accountId, username) {
      const confirmMessage = `Are you sure you want to DELETE the user "${username}"?\n\nThis will:\n1. Delete all access keys for this user\n2. Delete the user's login profile (if exists)\n3. Permanently delete the user\n\nThis action cannot be undone!`
      
      if (!confirm(confirmMessage)) return
      
      const loadingKey = `${accountId}-${username}`
      this.actionLoading[loadingKey] = true
      
      try {
        await axios.delete(`/api/accounts/${accountId}/users/${username}`)
        alert(`User "${username}" has been successfully deleted.`)
        // Remove the user from local state - backend already invalidated its cache
        this.allUsers = this.allUsers.filter(
          user => !(user.accountId === accountId && user.username === username)
        )
      } catch (error) {
        console.error('Failed to delete user:', error)
        alert(error.response?.data?.error || 'Failed to delete user. Please try again.')
      } finally {
        delete this.actionLoading[loadingKey]
      }
    },

    async removeUserPassword(accountId, username) {
      const confirmMessage = `Are you sure you want to remove the console password for user "${username}"?\n\nThis will:\n1. Delete the user's console login password\n2. Prevent the user from logging into the AWS Console\n3. Not affect programmatic access (access keys)\n\nThis action cannot be undone!`

      if (!confirm(confirmMessage)) return

      const loadingKey = `${accountId}-${username}-password`
      this.actionLoading[loadingKey] = true

      try {
        await axios.delete(`/api/accounts/${accountId}/users/${username}/password`)

        // Explicitly invalidate cache to ensure fresh data
        try {
          await axios.post(`/api/cache/accounts/${accountId}/invalidate`)
        } catch (cacheErr) {
          console.warn('Failed to invalidate account cache:', cacheErr)
        }

        alert(`Console password for user "${username}" has been successfully removed.`)
        // Refresh the data to update the list
        await this.refreshData()
      } catch (error) {
        console.error('Failed to remove user password:', error)
        alert(error.response?.data?.error || 'Failed to remove user password. Please try again.')
      } finally {
        delete this.actionLoading[loadingKey]
      }
    },

    async rotateUserPassword(accountId, username) {
      const confirmMessage = `Are you sure you want to rotate the console password for user "${username}"?\n\nThis will:\n1. Generate a new random console password\n2. ${this.allUsers.find(u => u.accountId === accountId && u.username === username)?.password_set ? 'Replace the existing password' : 'Create a new password'}\n3. Display the new password once (save it immediately)\n\nContinue?`

      if (!confirm(confirmMessage)) return

      const loadingKey = `${accountId}-${username}-rotate-password`
      this.actionLoading[loadingKey] = true

      try {
        const response = await axios.post(`/api/accounts/${accountId}/users/${username}/password/rotate`)

        // Explicitly invalidate cache to ensure fresh data
        try {
          await axios.post(`/api/cache/accounts/${accountId}/invalidate`)
        } catch (cacheErr) {
          console.warn('Failed to invalidate account cache:', cacheErr)
        }

        // Show the new password in an alert
        const newPassword = response.data.new_password
        alert(`New password for user "${username}":\n\n${newPassword}\n\nSave this password now! This is the only time it will be displayed.`)

        // Refresh the data to update the list
        await this.refreshData()
      } catch (error) {
        console.error('Failed to rotate user password:', error)
        alert(error.response?.data?.error || 'Failed to rotate user password. Please try again.')
      } finally {
        delete this.actionLoading[loadingKey]
      }
    },

    downloadUsersJSON() {
      try {
        const exportData = {
          exported_at: new Date().toISOString(),
          total_users: this.filteredUsers.length,
          total_accounts: this.accounts.length,
          accounts: this.accounts.map(account => ({
            id: account.id,
            name: account.name,
            accessible: account.accessible
          })),
          users: this.filteredUsers
        }
        
        const dataStr = JSON.stringify(exportData, null, 2)
        const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr)
        
        const exportFileDefaultName = `aws-iam-users-${new Date().toISOString().split('T')[0]}.json`
        
        const linkElement = document.createElement('a')
        linkElement.setAttribute('href', dataUri)
        linkElement.setAttribute('download', exportFileDefaultName)
        linkElement.click()
      } catch (error) {
        console.error('Failed to download JSON:', error)
        alert('Failed to download JSON file')
      }
    }
  }
}
</script>

<style scoped>
/* Container */
.all-users-container {
  min-height: 100vh;
  background: var(--color-bg-secondary);
  padding: 1.5rem;
}

/* Page Header */
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
  background: linear-gradient(135deg, #3b82f6, #6366f1);
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

/* Loading & Error States */
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

.error-container h3 {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 0.5rem;
}

.error-container p {
  color: var(--color-text-secondary);
  margin-bottom: 1.5rem;
}

/* Main Content */
.main-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* User Filters */
.user-filters {
  padding: 1.5rem;
  background: var(--color-bg-primary);
  border-radius: 0.75rem;
  border: 1px solid var(--color-border);
}

.search-box {
  position: relative;
  margin-bottom: 1rem;
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  width: 1rem;
  height: 1rem;
  color: var(--color-text-secondary);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 0.75rem 0.75rem 0.75rem 2.5rem;
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  font-size: 0.875rem;
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.search-input:focus {
  outline: none;
  border-color: var(--color-btn-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.filter-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.filter-btn {
  padding: 0.5rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  font-size: 0.75rem;
  font-weight: 500;
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.filter-btn:hover {
  background: var(--color-bg-tertiary);
}

.filter-btn.active {
  background: var(--color-btn-primary);
  color: white;
  border-color: transparent;
}

.account-actions {
  display: flex;
  gap: 1rem;
  align-items: center;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
}

.account-select {
  padding: 0.5rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 0.95rem;
  min-width: 250px;
  cursor: pointer;
}

.account-select:focus {
  outline: none;
  border-color: var(--color-primary);
}

/* Users Table */
.users-table-container {
  background: var(--color-bg-primary);
  border-radius: 0.75rem;
  border: 1px solid var(--color-border);
  overflow-x: auto;
  overflow-y: hidden;
}

.users-table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.users-table th {
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

.users-table td {
  padding: 1rem 0.75rem;
  border-bottom: 1px solid var(--color-border);
  vertical-align: top;
}

.user-row.clickable {
  cursor: pointer;
  transition: all 0.2s ease;
}

.user-row.clickable:hover {
  background: var(--color-bg-secondary);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.user-info,
.account-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.user-name,
.account-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.user-id,
.account-id {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  background: var(--color-bg-tertiary);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  border: 1px solid var(--color-border);
  width: fit-content;
}

.password-status {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  font-weight: 500;
}

.password-status.has-password {
  color: var(--color-success);
}

.password-status.no-password {
  color: var(--color-danger);
}

.password-cell {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  justify-content: space-between;
}

.password-actions {
  display: flex;
  gap: var(--spacing-xs);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.user-row:hover .password-actions {
  opacity: 1;
}

.password-rotate-btn,
.password-remove-btn {
  transition: all 0.2s ease;
}

.status-icon {
  width: 0.875rem;
  height: 0.875rem;
}

.key-count {
  font-weight: 600;
  color: var(--color-text-primary);
}

.old-key-count {
  font-weight: 600;
  color: var(--color-text-primary);
}

.old-key-count.warning {
  color: var(--color-warning);
}

.last-use-info {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.last-use-data {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.never-used {
  text-align: center;
}

.never-used-text {
  color: var(--color-warning);
  font-style: italic;
  font-size: 0.8rem;
}

.no-keys {
  text-align: center;
}

.no-keys-text {
  color: var(--color-text-tertiary);
  font-size: 0.8rem;
}

.sortable {
  cursor: pointer;
  user-select: none;
}

.sortable:hover {
  background: var(--color-bg-secondary);
}

.sort-indicator {
  margin-left: 0.5rem;
  color: var(--color-btn-primary);
  font-weight: bold;
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

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  text-decoration: none;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.2s ease;
  justify-content: center;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--color-btn-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-btn-primary-hover);
  transform: translateY(-1px);
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

.btn-sm {
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
  gap: 0.375rem;
}

.btn-xs {
  padding: 0.25rem 0.5rem;
  font-size: 0.625rem;
  gap: 0.25rem;
}

.btn-warning {
  background: var(--color-btn-warning);
  color: white;
  border: 1px solid var(--color-btn-warning);
}

.btn-warning:hover:not(:disabled) {
  background: var(--color-btn-warning-hover);
  border-color: var(--color-btn-warning-hover);
}

.btn-danger {
  background: var(--color-btn-danger);
  color: white;
  border: 1px solid var(--color-btn-danger);
}

.btn-danger:hover:not(:disabled) {
  background: var(--color-btn-danger-hover);
  border-color: var(--color-btn-danger-hover);
}

.btn-icon {
  width: 1rem;
  height: 1rem;
}

/* Empty Results */
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
  color: var(--color-text-secondary);
}

.empty-icon svg {
  width: 2rem;
  height: 2rem;
}

.empty-results h4 {
  font-size: 1.125rem;
  color: var(--color-text-primary);
  margin-bottom: 0.5rem;
}

.empty-results p {
  color: var(--color-text-secondary);
}

/* Responsive Design */
@media (max-width: 1024px) {
  .users-table th:nth-child(3),
  .users-table td:nth-child(3),
  .users-table th:nth-child(6),
  .users-table td:nth-child(6) {
    display: none;
  }
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }
  
  .header-actions {
    justify-content: center;
  }
  
  .filter-buttons {
    flex-direction: column;
  }
  
  .filter-btn {
    text-align: center;
  }
  
  .users-table th:nth-child(2),
  .users-table td:nth-child(2) {
    display: none;
  }
}

@media (max-width: 480px) {
  .all-users-container {
    padding: 1rem;
  }
  
  .header-title {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }
  
  .header-icon {
    width: 3rem;
    height: 3rem;
  }
  
  .header-icon svg {
    width: 1.5rem;
    height: 1.5rem;
  }
  
  .title-content h1 {
    font-size: 1.5rem;
  }
  
  .users-table th:nth-child(4),
  .users-table td:nth-child(4),
  .users-table th:nth-child(5),
  .users-table td:nth-child(5) {
    display: none;
  }
}


.user-actions {
  display: flex;
  gap: var(--spacing-xs);
  align-items: center;
}
</style>