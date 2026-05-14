<template>
  <div class="eks-cluster-detail">
    <nav class="breadcrumb">
      <router-link to="/aws/eks-clusters">← Back to EKS Clusters</router-link>
    </nav>

    <div class="page-header">
      <div class="header-content">
        <div class="header-title">
          <div class="header-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12,2L1,21H23L12,2M12,6L18.9,21H5.1L12,6Z"/>
            </svg>
          </div>
          <div class="title-content">
            <h1>{{ clusterName }}</h1>
            <p v-if="cluster">
              <span :class="['status-badge', getStatusClass(cluster.status)]">{{ cluster.status }}</span>
              &nbsp;{{ cluster.account_name }} &bull; {{ cluster.region }}
            </p>
          </div>
        </div>
        <div class="header-actions">
          <button @click="downloadJSON" class="btn btn-success" :disabled="loading || !cluster">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M14,2H6A2,2 0 0,0 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2M18,20H6V4H13V9H18V20Z"/>
            </svg>
            Download JSON
          </button>
          <button @click="loadData" class="btn btn-secondary" :disabled="loading">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
            </svg>
            {{ loading ? 'Refreshing...' : 'Refresh' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading cluster details...</p>
    </div>

    <div v-else-if="error" class="error-container">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
        </svg>
      </div>
      <h3>Failed to Load Cluster</h3>
      <p>{{ error }}</p>
      <button @click="loadData" class="btn btn-primary">Try Again</button>
    </div>

    <div v-else-if="!cluster" class="error-container">
      <h3>Cluster Not Found</h3>
      <p>No cluster named <strong>{{ clusterName }}</strong> found in account {{ accountId }} / {{ region }}.</p>
      <router-link to="/aws/eks-clusters" class="btn btn-primary">Back to EKS Clusters</router-link>
    </div>

    <div v-else class="main-content">
      <!-- Cluster Info -->
      <div class="section-card">
        <h2 class="section-title">Cluster Details</h2>
        <div class="info-grid">
          <div class="info-row">
            <span class="info-label">Cluster Name</span>
            <span class="info-value mono">{{ cluster.cluster_name }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">ARN</span>
            <span class="info-value mono small">{{ cluster.cluster_arn }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Account</span>
            <span class="info-value">{{ cluster.account_name }} <span class="muted">({{ cluster.account_id }})</span></span>
          </div>
          <div class="info-row">
            <span class="info-label">Region</span>
            <span class="info-value mono">{{ cluster.region }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Status</span>
            <span class="info-value">
              <span :class="['status-badge', getStatusClass(cluster.status)]">{{ cluster.status }}</span>
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">Platform Version</span>
            <span class="info-value mono">{{ cluster.platform_version || '-' }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Auth Mode</span>
            <span class="info-value mono">{{ cluster.auth_mode || '-' }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">API Access</span>
            <span class="info-value">
              <span v-if="cluster.public_access" class="feature-badge feature-public">Public</span>
              <span v-else class="feature-badge feature-private">Private</span>
            </span>
          </div>
          <div class="info-row" v-if="cluster.endpoint">
            <span class="info-label">Endpoint</span>
            <span class="info-value mono small">{{ cluster.endpoint }}</span>
          </div>
          <div class="info-row" v-if="cluster.created_at">
            <span class="info-label">Created At</span>
            <span class="info-value">{{ formatDate(cluster.created_at) }}</span>
          </div>
        </div>
      </div>

      <!-- Tags -->
      <div class="section-card" v-if="cluster.tags && cluster.tags.length > 0">
        <h2 class="section-title">Tags <span class="count-badge">{{ cluster.tags.length }}</span></h2>
        <table class="detail-table">
          <thead>
            <tr>
              <th>Key</th>
              <th>Value</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="tag in cluster.tags" :key="tag.key">
              <td class="mono">{{ tag.key }}</td>
              <td class="mono">{{ tag.value }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Node Groups -->
      <div class="section-card" v-if="cluster.node_groups && cluster.node_groups.length > 0">
        <h2 class="section-title">Node Groups <span class="count-badge">{{ cluster.node_groups.length }}</span></h2>
        <div class="node-groups">
          <div v-for="ng in cluster.node_groups" :key="ng.node_group_name" class="node-group-card">
            <div class="ng-header">
              <div class="ng-name">{{ ng.node_group_name }}</div>
              <span :class="['status-badge', getStatusClass(ng.status)]">{{ ng.status }}</span>
            </div>
            <div class="ng-info-grid">
              <div class="ng-info-row">
                <span class="ng-info-label">ARN</span>
                <span class="ng-info-value mono small">{{ ng.node_group_arn }}</span>
              </div>
              <div class="ng-info-row">
                <span class="ng-info-label">Instance Types</span>
                <span class="ng-info-value">
                  <span v-for="it in ng.instance_types" :key="it" class="instance-badge">{{ it }}</span>
                  <span v-if="!ng.instance_types || ng.instance_types.length === 0" class="muted">-</span>
                </span>
              </div>
              <div class="ng-info-row" v-if="ng.scaling_config">
                <span class="ng-info-label">Scaling</span>
                <span class="ng-info-value">
                  <span class="scaling-config">
                    <span class="scaling-item" title="Desired">{{ ng.scaling_config.desired_size }} desired</span>
                    &bull;
                    <span class="scaling-item" title="Min">{{ ng.scaling_config.min_size }} min</span>
                    &bull;
                    <span class="scaling-item" title="Max">{{ ng.scaling_config.max_size }} max</span>
                  </span>
                </span>
              </div>
              <div class="ng-info-row" v-if="ng.release_version">
                <span class="ng-info-label">Release Version</span>
                <span class="ng-info-value mono">{{ ng.release_version }}</span>
              </div>
            </div>

            <!-- Node Group Labels -->
            <div v-if="ng.labels && Object.keys(ng.labels).length > 0" class="ng-subsection">
              <div class="ng-subsection-title">Labels</div>
              <table class="detail-table detail-table-sm">
                <thead>
                  <tr><th>Key</th><th>Value</th></tr>
                </thead>
                <tbody>
                  <tr v-for="(val, key) in ng.labels" :key="key">
                    <td class="mono">{{ key }}</td>
                    <td class="mono">{{ val }}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            <!-- Node Group Tags -->
            <div v-if="ng.tags && ng.tags.length > 0" class="ng-subsection">
              <div class="ng-subsection-title">Tags</div>
              <table class="detail-table detail-table-sm">
                <thead>
                  <tr><th>Key</th><th>Value</th></tr>
                </thead>
                <tbody>
                  <tr v-for="tag in ng.tags" :key="tag.key">
                    <td class="mono">{{ tag.key }}</td>
                    <td class="mono">{{ tag.value }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>

      <div class="section-card" v-else-if="cluster.node_groups && cluster.node_groups.length === 0">
        <h2 class="section-title">Node Groups</h2>
        <p class="muted" style="padding: 1rem 0;">No node groups found for this cluster.</p>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'EKSClusterDetail',
  props: {
    accountId: { type: String, required: true },
    region: { type: String, required: true },
    clusterName: { type: String, required: true }
  },
  data() {
    return {
      cluster: null,
      loading: true,
      error: null
    }
  },
  watch: {
    accountId: 'loadData',
    region: 'loadData',
    clusterName: 'loadData'
  },
  mounted() {
    this.loadData()
  },
  methods: {
    async loadData() {
      try {
        this.loading = true
        this.error = null
        const response = await axios.get(`/api/accounts/${this.accountId}/eks-clusters`)
        const clusters = response.data || []
        this.cluster = clusters.find(
          c => c.cluster_name === this.clusterName && c.region === this.region
        ) || null
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load cluster details'
      } finally {
        this.loading = false
      }
    },
    getStatusClass(status) {
      switch (status?.toUpperCase()) {
        case 'ACTIVE': return 'status-active'
        case 'CREATING': return 'status-creating'
        case 'DELETING': return 'status-deleting'
        case 'FAILED': return 'status-failed'
        case 'PAUSED': return 'status-paused'
        default: return 'status-default'
      }
    },
    formatDate(dt) {
      if (!dt) return '-'
      return new Date(dt).toLocaleString()
    },
    downloadJSON() {
      try {
        const data = JSON.stringify(this.cluster, null, 2)
        const blob = new Blob([data], { type: 'application/json' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `eks-cluster-${this.clusterName}-${new Date().toISOString().slice(0, 10)}.json`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)
      } catch (e) {
        console.error('Failed to download JSON:', e)
      }
    }
  }
}
</script>

<style scoped>
.eks-cluster-detail {
  min-height: 100vh;
  background: var(--color-bg-secondary);
  padding: 1.5rem;
}

.breadcrumb {
  margin-bottom: 1.5rem;
}

.breadcrumb a {
  color: var(--color-primary);
  text-decoration: none;
}

.breadcrumb a:hover {
  text-decoration: underline;
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
  background: linear-gradient(135deg, #f97316 0%, #ea580c 100%);
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
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--color-text-primary);
}

.title-content p {
  margin: 0.4rem 0 0 0;
  color: var(--color-text-secondary);
  font-size: 0.95rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
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
  text-decoration: none;
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
  border-top-color: #f97316;
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

.section-card {
  background: var(--color-bg-primary);
  border-radius: 12px;
  padding: 2rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.section-title {
  margin: 0 0 1.5rem 0;
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--color-text-primary);
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-tertiary);
  color: var(--color-text-secondary);
  font-size: 0.8rem;
  font-weight: 600;
  padding: 0.15rem 0.5rem;
  border-radius: 10px;
}

.info-grid {
  display: grid;
  gap: 0;
}

.info-row {
  display: grid;
  grid-template-columns: 180px 1fr;
  gap: 1rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--color-border);
  align-items: start;
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  font-weight: 500;
  color: var(--color-text-secondary);
  font-size: 0.9rem;
  padding-top: 0.1rem;
}

.info-value {
  color: var(--color-text-primary);
  word-break: break-all;
}

.info-value.mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.9rem;
}

.info-value.small {
  font-size: 0.8rem;
  color: var(--color-text-secondary);
}

.muted {
  color: var(--color-text-secondary);
}

.detail-table {
  width: 100%;
  border-collapse: collapse;
}

.detail-table th {
  text-align: left;
  padding: 0.6rem 1rem;
  background: var(--color-bg-secondary);
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.detail-table td {
  padding: 0.6rem 1rem;
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  font-size: 0.9rem;
}

.detail-table tr:last-child td {
  border-bottom: none;
}

.detail-table.detail-table-sm th,
.detail-table.detail-table-sm td {
  padding: 0.4rem 0.75rem;
  font-size: 0.85rem;
}

.mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.small {
  font-size: 0.8rem;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-active { background: rgba(16, 185, 129, 0.1); color: #10b981; }
.status-creating { background: rgba(59, 130, 246, 0.1); color: #3b82f6; }
.status-deleting { background: rgba(245, 158, 11, 0.1); color: #f59e0b; }
.status-failed { background: rgba(239, 68, 68, 0.1); color: #ef4444; }
.status-paused { background: rgba(107, 114, 128, 0.1); color: #6b7280; }
.status-default { background: rgba(107, 114, 128, 0.1); color: #6b7280; }

.feature-badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.feature-public { background: rgba(245, 158, 11, 0.1); color: #f59e0b; }
.feature-private { background: rgba(16, 185, 129, 0.1); color: #10b981; }

/* Node Groups */
.node-groups {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.node-group-card {
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
}

.ng-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.9rem 1.25rem;
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
}

.ng-name {
  font-weight: 600;
  color: #f97316;
  font-size: 1rem;
}

.ng-info-grid {
  padding: 0 1.25rem;
}

.ng-info-row {
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 1rem;
  padding: 0.6rem 0;
  border-bottom: 1px solid var(--color-border);
  align-items: start;
}

.ng-info-row:last-child {
  border-bottom: none;
}

.ng-info-label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.ng-info-value {
  font-size: 0.9rem;
  color: var(--color-text-primary);
  word-break: break-all;
}

.ng-info-value.mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.ng-info-value.small {
  font-size: 0.8rem;
  color: var(--color-text-secondary);
}

.instance-badge {
  display: inline-block;
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  margin-right: 0.35rem;
}

.scaling-config {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--color-text-primary);
  font-size: 0.9rem;
}

.ng-subsection {
  border-top: 1px solid var(--color-border);
  padding: 1rem 1.25rem;
}

.ng-subsection-title {
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--color-text-secondary);
  margin-bottom: 0.75rem;
}
</style>
