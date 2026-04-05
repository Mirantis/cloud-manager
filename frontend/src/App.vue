<template>
  <div id="app" :class="{ dark: isDarkTheme, 'sidebar-collapsed': sidebarCollapsed }">
    <!-- Mobile Overlay -->
    <div v-if="isMobile && sidebarCollapsed" class="sidebar-overlay" @click="toggleSidebar"></div>
    
    <!-- Sidebar Navigation -->
    <aside class="sidebar">
      <div class="sidebar-header">
        <router-link to="/aws/users" class="logo-link">
          <h1>Cloud Manager</h1>
        </router-link>
        <button @click="toggleSidebar" class="sidebar-toggle" v-if="isMobile">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M3,6H21V8H3V6M3,11H21V13H3V11M3,16H21V18H3V16Z"/>
          </svg>
        </button>
      </div>
      
      <!-- AWS Section -->
      <div class="sidebar-section">
        <div class="section-header" @click="toggleAWS">
          <div class="section-title">
            <svg class="section-icon aws-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4C16.41,4 20,7.59 20,12C20,16.41 16.41,20 12,20C7.59,20 4,16.41 4,12C4,7.59 7.59,4 12,4M12,6A6,6 0 0,0 6,12A6,6 0 0,0 12,18A6,6 0 0,0 18,12A6,6 0 0,0 12,6Z"/>
            </svg>
            <span>AWS</span>
          </div>
          <svg class="section-arrow" :class="{ 'expanded': awsExpanded }" viewBox="0 0 24 24" fill="currentColor">
            <path d="M7.41,8.58L12,13.17L16.59,8.58L18,10L12,16L6,10L7.41,8.58Z"/>
          </svg>
        </div>
        <div class="section-tabs" v-if="awsExpanded">
          <router-link to="/aws/users" class="sidebar-link">Users</router-link>
          <router-link to="/aws/public-ips" class="sidebar-link">Public IPs</router-link>
          <router-link to="/aws/security-groups" class="sidebar-link">Security Groups</router-link>
          <router-link to="/aws/ec2-instances" class="sidebar-link">EC2 Instances</router-link>
          <router-link to="/aws/ebs-volumes" class="sidebar-link">EBS Volumes</router-link>
          <router-link to="/aws/s3-buckets" class="sidebar-link">S3 Buckets</router-link>
          <router-link to="/aws/load-balancers" class="sidebar-link">Load Balancers</router-link>
          <router-link to="/aws/vpcs" class="sidebar-link">VPCs</router-link>
          <router-link to="/aws/nat-gateways" class="sidebar-link">NAT Gateways</router-link>
          <router-link to="/aws/roles" class="sidebar-link">IAM Roles</router-link>
          <router-link to="/aws/snapshots" class="sidebar-link">Snapshots</router-link>
          <router-link to="/aws/route53-domains" class="sidebar-link">Route53 Domains</router-link>
          <router-link to="/aws/sso/users" class="sidebar-link">SSO Users</router-link>
          <router-link to="/aws/sso/groups" class="sidebar-link">SSO Groups</router-link>
          <router-link to="/aws/sso/user-assignments" class="sidebar-link">User Assignments</router-link>
          <router-link to="/aws/sso/account-assignments" class="sidebar-link">Account Assignments</router-link>
        </div>
      </div>

      <!-- Admin Section -->
      <div class="sidebar-section" v-if="userRole === 'admin'">
        <div class="section-header" @click="toggleAdmin">
          <div class="section-title">
            <span>Admin</span>
          </div>
          <svg class="section-arrow" :class="{ 'expanded': adminExpanded }" viewBox="0 0 24 24" fill="currentColor">
            <path d="M7.41,8.58L12,13.17L16.59,8.58L18,10L12,16L6,10L7.41,8.58Z"/>
          </svg>
        </div>
        <div class="section-tabs" v-if="adminExpanded">
          <router-link to="/admin/users" class="sidebar-link">User Management</router-link>
        </div>
      </div>

      <!-- Azure Section -->
      <div class="sidebar-section">
        <div class="section-header" @click="toggleAzure">
          <div class="section-title">
            <svg class="section-icon azure-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M13.05,4.24L6.56,18.05L2,18L7.09,9.24L13.05,4.24M13.75,5.33L22,19.76H15.27L13.75,5.33Z"/>
            </svg>
            <span>Azure</span>
          </div>
          <svg class="section-arrow" :class="{ 'expanded': azureExpanded }" viewBox="0 0 24 24" fill="currentColor">
            <path d="M7.41,8.58L12,13.17L16.59,8.58L18,10L12,16L6,10L7.41,8.58Z"/>
          </svg>
        </div>
        <div class="section-tabs" v-if="azureExpanded">
          <router-link to="/azure/enterprise-apps" class="sidebar-link">Enterprise Apps</router-link>
          <router-link to="/azure/vms" class="sidebar-link">Virtual Machines</router-link>
          <router-link to="/azure/storage" class="sidebar-link">Storage Accounts</router-link>
        </div>
      </div>
    </aside>

    <!-- Main Content Area -->
    <div class="main-wrapper">
      <header>
        <div class="header-content">
          <div v-if="isMobile" class="mobile-menu-btn" @click="toggleSidebar">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M3,6H21V8H3V6M3,11H21V13H3V11M3,16H21V18H3V16Z"/>
            </svg>
          </div>
          <div class="header-spacer"></div>
          <div v-if="isAuthenticated" class="user-info">
            <span class="username">{{ username }}<span v-if="userRole" class="role-badge">{{ userRole }}</span></span>
            <button @click="handleLogout" class="logout-btn">Logout</button>
          </div>
          <button @click="toggleTheme" class="theme-btn">
            {{ isDarkTheme ? '☀️' : '🌙' }}
          </button>
        </div>
      </header>
      <main>
        <router-view v-slot="{ Component }">
          <keep-alive include="AllUsers">
            <component :is="Component" />
          </keep-alive>
        </router-view>
      </main>
    </div>
  </div>
</template>

<script>
import { authStore } from './authStore'

export default {
  name: 'App',
  data() {
    return {
      isDarkTheme: false,
      isAuthenticated: false,
      username: '',
      userRole: '',
      awsExpanded: true,
      azureExpanded: true,
      adminExpanded: true,
      sidebarCollapsed: false,
      isMobile: false
    }
  },
  methods: {
    toggleAdmin() {
      this.adminExpanded = !this.adminExpanded
      localStorage.setItem('adminExpanded', this.adminExpanded.toString())
    },
    toggleAWS() {
      this.awsExpanded = !this.awsExpanded
      localStorage.setItem('awsExpanded', this.awsExpanded.toString())
    },
    toggleAzure() {
      this.azureExpanded = !this.azureExpanded
      localStorage.setItem('azureExpanded', this.azureExpanded.toString())
    },
    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed
    },
    checkMobile() {
      this.isMobile = window.innerWidth < 768
      if (!this.isMobile) {
        this.sidebarCollapsed = false
      }
    },
    toggleTheme() {
      this.isDarkTheme = !this.isDarkTheme
      localStorage.setItem('theme', this.isDarkTheme ? 'dark' : 'light')
      this.updateTheme()
    },
    updateTheme() {
      if (this.isDarkTheme) {
        document.documentElement.setAttribute('data-theme', 'dark')
      } else {
        document.documentElement.setAttribute('data-theme', 'light')
      }
    },
    initTheme() {
      const savedTheme = localStorage.getItem('theme')
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      
      this.isDarkTheme = savedTheme === 'dark' || (!savedTheme && prefersDark)
      this.updateTheme()
    },
    async checkAuth() {
      try {
        const response = await fetch('/api/auth/check', {
          credentials: 'include'
        })
        const data = await response.json()
        this.isAuthenticated = data.authenticated || false
        this.username = data.username || ''
        this.userRole = data.role || ''
        authStore.authenticated = this.isAuthenticated
        authStore.username = this.username
        authStore.role = this.userRole
      } catch (err) {
        this.isAuthenticated = false
        this.username = ''
        this.userRole = ''
        authStore.authenticated = false
        authStore.username = ''
        authStore.role = ''
      }
    },
    async handleLogout() {
      try {
        await fetch('/api/auth/logout', {
          method: 'POST',
          credentials: 'include'
        })
        this.isAuthenticated = false
        this.username = ''
        this.userRole = ''
        authStore.authenticated = false
        authStore.username = ''
        authStore.role = ''
        this.$router.push('/login')
      } catch (err) {
        // Even if logout fails, redirect to login
        this.isAuthenticated = false
        this.username = ''
        this.userRole = ''
        authStore.authenticated = false
        authStore.username = ''
        authStore.role = ''
        this.$router.push('/login')
      }
    }
  },
  mounted() {
    this.initTheme()
    this.checkAuth()
    this.checkMobile()
    
    // Load sidebar state from localStorage
    const awsExpanded = localStorage.getItem('awsExpanded')
    const azureExpanded = localStorage.getItem('azureExpanded')
    if (awsExpanded !== null) {
      this.awsExpanded = awsExpanded === 'true'
    }
    if (azureExpanded !== null) {
      this.azureExpanded = azureExpanded === 'true'
    }
    const adminExpanded = localStorage.getItem('adminExpanded')
    if (adminExpanded !== null) {
      this.adminExpanded = adminExpanded === 'true'
    }
    
    // Handle window resize
    window.addEventListener('resize', this.checkMobile)
    
    // Check auth status periodically
    setInterval(() => {
      this.checkAuth()
    }, 60000) // Check every minute
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.checkMobile)
  },
  watch: {
    $route() {
      // Check auth when route changes
      this.checkAuth()
    }
  }
}
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

:root {
  /* Colors - Light Theme */
  --color-bg-primary: #ffffff;
  --color-bg-secondary: #f8f9fa;
  --color-bg-tertiary: #e9ecef;
  --color-bg-hover: #f3f4f6;
  --color-text-primary: #1f2937;
  --color-text-secondary: #6b7280;
  --color-text-tertiary: #9ca3af;
  --color-border: #e5e7eb;
  --color-border-light: #f3f4f6;
  --color-primary: #3b82f6;
  
  /* Button Colors */
  --color-btn-primary: #3b82f6;
  --color-btn-primary-hover: #2563eb;
  --color-btn-secondary: #6b7280;
  --color-btn-secondary-hover: #4b5563;
  --color-btn-success: #10b981;
  --color-btn-success-hover: #059669;
  --color-btn-warning: #f59e0b;
  --color-btn-warning-hover: #d97706;
  --color-btn-danger: #ef4444;
  --color-btn-danger-hover: #dc2626;
  
  /* Status Colors */
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --color-danger: #ef4444;
  
  /* Shadows */
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
  
  /* Border Radius */
  --radius-sm: 0.375rem;
  --radius: 0.5rem;
  --radius-md: 0.75rem;
  --radius-lg: 1rem;
  --radius-xl: 1.5rem;
  
  /* Spacing */
  --spacing-xs: 0.25rem;
  --spacing-sm: 0.5rem;
  --spacing: 1rem;
  --spacing-lg: 1.5rem;
  --spacing-xl: 2rem;
  --spacing-2xl: 3rem;
  
  /* Transitions */
  --transition-fast: 150ms ease-in-out;
  --transition: 200ms ease-in-out;
  --transition-slow: 300ms ease-in-out;
}

[data-theme="dark"] {
  /* Colors - Dark Theme */
  --color-bg-primary: #0d1117;
  --color-bg-secondary: #161b22;
  --color-bg-tertiary: #21262d;
  --color-bg-hover: #1c2128;
  --color-text-primary: #f0f6fc;
  --color-text-secondary: #8b949e;
  --color-text-tertiary: #6e7681;
  --color-border: #30363d;
  --color-border-light: #21262d;
  --color-primary: #58a6ff;
}

html, body {
  width: 100%;
  max-width: 100vw;
  overflow-x: hidden;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  line-height: 1.6;
  transition: background-color var(--transition), color var(--transition);
}

#app {
  min-height: 100vh;
  display: flex;
  flex-direction: row;
  width: 100%;
  max-width: 100vw;
  overflow-x: hidden;
}

/* Sidebar Styles */
.sidebar {
  width: 240px;
  background: var(--color-bg-primary);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  overflow-y: auto;
  transition: transform var(--transition);
  z-index: 1000;
}

.sidebar-header {
  padding: var(--spacing-lg);
  border-bottom: 1px solid var(--color-border);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sidebar-header .logo-link h1 {
  font-size: 1.25rem;
  margin: 0;
}

.sidebar-toggle {
  background: none;
  border: none;
  color: var(--color-text-primary);
  cursor: pointer;
  padding: var(--spacing-xs);
  display: none;
}

.sidebar-toggle svg {
  width: 1.5rem;
  height: 1.5rem;
}

.sidebar-section {
  border-bottom: 1px solid var(--color-border-light);
}

.section-header {
  padding: var(--spacing) var(--spacing-lg);
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  transition: background-color var(--transition-fast);
  user-select: none;
}

.section-header:hover {
  background: var(--color-bg-secondary);
}

.section-title {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-weight: 600;
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.section-icon {
  width: 1.25rem;
  height: 1.25rem;
}

.aws-icon {
  color: #ff9900;
}

.azure-icon {
  color: #0078d4;
}

.section-arrow {
  width: 1rem;
  height: 1rem;
  transition: transform var(--transition-fast);
  color: var(--color-text-secondary);
}

.section-arrow.expanded {
  transform: rotate(180deg);
}

.section-tabs {
  display: flex;
  flex-direction: column;
  animation: slideDown var(--transition);
}

@keyframes slideDown {
  from {
    opacity: 0;
    max-height: 0;
  }
  to {
    opacity: 1;
    max-height: 1000px;
  }
}

.sidebar-link {
  padding: var(--spacing-sm) var(--spacing-lg) var(--spacing-sm) 3rem;
  color: var(--color-text-secondary);
  text-decoration: none;
  font-size: 0.875rem;
  transition: all var(--transition-fast);
  border-left: 3px solid transparent;
}

.sidebar-link:hover {
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  border-left-color: var(--color-btn-primary);
}

.sidebar-link.router-link-active {
  background: var(--color-bg-secondary);
  color: var(--color-btn-primary);
  border-left-color: var(--color-btn-primary);
  font-weight: 500;
}

/* Main Wrapper */
.main-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin-left: 240px;
  min-height: 100vh;
  width: calc(100% - 240px);
  max-width: calc(100% - 240px);
  overflow-x: auto;
}

header {
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  box-shadow: var(--shadow);
  border-bottom: 1px solid var(--color-border);
}

.header-content {
  width: 100%;
  max-width: 100%;
  padding: 1rem 2rem;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: var(--spacing);
  overflow-x: auto;
}

.header-spacer {
  flex: 1;
}

.mobile-menu-btn {
  background: none;
  border: none;
  color: var(--color-text-primary);
  cursor: pointer;
  padding: var(--spacing-sm);
  display: none;
}

.mobile-menu-btn svg {
  width: 1.5rem;
  height: 1.5rem;
}

.logo-link {
  color: var(--color-text-primary);
  text-decoration: none;
  transition: opacity var(--transition-fast);
}

.logo-link:hover {
  opacity: 0.8;
}

.logo h1 {
  font-size: 1.5rem;
  font-weight: 600;
}

/* Removed old nav styles - now using sidebar */

.theme-btn {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  color: var(--color-text-primary);
  padding: var(--spacing-sm) var(--spacing);
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 1.2rem;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 2.5rem;
  height: 2.5rem;
}

.theme-btn:hover {
  background: var(--color-bg-tertiary);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.username {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}

.role-badge {
  margin-left: 0.5rem;
  font-size: 0.7rem;
  text-transform: uppercase;
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  background: var(--color-bg-tertiary);
  color: var(--color-text-tertiary);
  font-weight: 600;
}

.logout-btn {
  background: var(--color-btn-danger);
  color: white;
  border: none;
  padding: var(--spacing-sm) var(--spacing);
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all var(--transition-fast);
}

.logout-btn:hover {
  background: var(--color-btn-danger-hover);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

main {
  flex: 1;
  padding: var(--spacing-xl);
  width: 100%;
  max-width: 100%;
  background: var(--color-bg-secondary);
  min-height: calc(100vh - 4rem);
  overflow-x: auto;
}

.card {
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  padding: var(--spacing-xl);
  margin-bottom: var(--spacing);
  box-shadow: var(--shadow);
  border: 1px solid var(--color-border-light);
  transition: all var(--transition-fast);
}

.card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

/* Modern Button System */
.btn {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing);
  border-radius: var(--radius);
  border: 1px solid transparent;
  font-size: 0.875rem;
  font-weight: 500;
  line-height: 1.5;
  text-decoration: none;
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
  overflow: hidden;
  
  /* Primary Button */
  background: var(--color-btn-primary);
  color: white;
  box-shadow: var(--shadow-sm);
}

.btn:hover {
  background: var(--color-btn-primary-hover);
  transform: translateY(-1px);
  box-shadow: var(--shadow);
}

.btn:active {
  transform: translateY(0);
  box-shadow: var(--shadow-sm);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

/* Button Variants */
.btn-secondary {
  background: var(--color-btn-secondary);
  color: white;
}

.btn-secondary:hover {
  background: var(--color-btn-secondary-hover);
}

.btn-success {
  background: var(--color-btn-success);
  color: white;
}

.btn-success:hover {
  background: var(--color-btn-success-hover);
}

.btn-warning {
  background: var(--color-btn-warning);
  color: white;
}

.btn-warning:hover {
  background: var(--color-btn-warning-hover);
}

.btn-danger {
  background: var(--color-btn-danger);
  color: white;
}

.btn-danger:hover {
  background: var(--color-btn-danger-hover);
}

/* Button Sizes */
.btn-sm {
  padding: var(--spacing-xs) var(--spacing-sm);
  font-size: 0.75rem;
}

.btn-lg {
  padding: var(--spacing) var(--spacing-lg);
  font-size: 1rem;
}

/* Button with Icons */
.btn-icon {
  width: 1rem;
  height: 1rem;
  flex-shrink: 0;
}

.table {
  width: 100%;
  border-collapse: collapse;
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: var(--shadow);
  border: 1px solid var(--color-border-light);
}

.table th, .table td {
  padding: var(--spacing);
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.table th {
  background: var(--color-bg-secondary);
  font-weight: 600;
  color: var(--color-text-primary);
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.table tbody tr {
  transition: background-color var(--transition-fast);
}

.table tbody tr:hover {
  background: var(--color-bg-secondary);
}

.loading {
  text-align: center;
  padding: var(--spacing-2xl);
  color: var(--color-text-secondary);
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--spacing);
}

.loading::before {
  content: '';
  display: block;
  width: 2.5rem;
  height: 2.5rem;
  border: 3px solid var(--color-border);
  border-top: 3px solid var(--color-btn-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-danger);
  padding: var(--spacing);
  border-radius: var(--radius);
  margin: var(--spacing) 0;
  border: 1px solid rgba(239, 68, 68, 0.2);
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.error::before {
  content: '⚠️';
  font-size: 1.2rem;
}

/* Mobile Overlay */
.sidebar-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 999;
  animation: fadeIn var(--transition-fast);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* Responsive Styles */
@media (max-width: 768px) {
  #app {
    position: relative;
  }

  .sidebar {
    transform: translateX(-100%);
    box-shadow: var(--shadow-lg);
  }

  #app.sidebar-collapsed .sidebar {
    transform: translateX(0);
  }

  .main-wrapper {
    margin-left: 0;
    width: 100%;
    max-width: 100%;
  }

  .mobile-menu-btn {
    display: block;
  }

  .sidebar-toggle {
    display: block;
  }

  main {
    padding: 1rem;
  }
  
  .table {
    font-size: 0.875rem;
  }
  
  .table th, .table td {
    padding: 0.5rem;
  }
}
</style>