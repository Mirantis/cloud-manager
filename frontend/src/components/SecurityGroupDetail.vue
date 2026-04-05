<template>
  <div class="security-group-detail">
    <nav class="breadcrumb">
      <router-link to="/aws/security-groups">← Back to Security Groups</router-link>
    </nav>

    <h1>Security Group Detail</h1>

    <div class="params">
      <p><strong>Account ID:</strong> {{ accountId }}</p>
      <p><strong>Region:</strong> {{ region }}</p>
      <p><strong>Group ID:</strong> {{ groupId }}</p>
    </div>

    <div v-if="loading" class="loading">
      <div class="spinner"></div>
      <span>Loading security group details...</span>
    </div>

    <div v-else-if="error" class="error">
      <h3>Error loading security group</h3>
      <p>{{ error }}</p>
      <button @click="loadSecurityGroup" class="retry-btn">Retry</button>
    </div>

    <div v-else-if="securityGroup" class="content">
      <div class="info-section">
        <h2>{{ securityGroup.group_name }}</h2>
        <p><strong>Description:</strong> {{ securityGroup.description || 'No description' }}</p>
        <p><strong>VPC:</strong> {{ securityGroup.vpc_id || 'EC2-Classic' }}</p>
        <p><strong>Usage:</strong>
          <span :class="securityGroup.is_unused ? 'unused' : 'used'">
            {{ securityGroup.is_unused ? 'Unused' : 'Used' }}
          </span>
        </p>
        <p><strong>Internet Access:</strong>
          <span :class="securityGroup.has_open_ports ? 'open' : 'closed'">
            {{ securityGroup.has_open_ports ? 'Open' : 'Closed' }}
          </span>
        </p>
      </div>

      <!-- Show open ports if any -->
      <div v-if="securityGroup.has_open_ports && securityGroup.open_ports_info.length > 0" class="open-ports-section">
        <h3>⚠️ Ports Open to Internet</h3>
        <div class="open-ports">
          <div v-for="(port, index) in securityGroup.open_ports_info" :key="index" class="port-item">
            <strong>{{ port.port_range }}</strong> ({{ port.protocol.toUpperCase() }}) - {{ port.source }}
            <br><small>{{ port.description }}</small>
          </div>
        </div>
      </div>

      <!-- Inbound Rules -->
      <div class="rules-section">
        <h3>Inbound Rules ({{ securityGroup.ingress_rules.length }})</h3>
        <div v-if="securityGroup.ingress_rules.length === 0" class="no-rules">
          No inbound rules configured.
        </div>
        <table v-else class="rules-table">
          <thead>
            <tr>
              <th>Protocol</th>
              <th>Port Range</th>
              <th>Source</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(rule, index) in securityGroup.ingress_rules" :key="index">
              <td>{{ formatProtocol(rule.ip_protocol) }}</td>
              <td>{{ formatPortRange(rule) }}</td>
              <td>{{ formatSource(rule) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Outbound Rules -->
      <div class="rules-section">
        <h3>Outbound Rules ({{ securityGroup.egress_rules.length }})</h3>
        <div v-if="securityGroup.egress_rules.length === 0" class="no-rules">
          No outbound rules configured.
        </div>
        <table v-else class="rules-table">
          <thead>
            <tr>
              <th>Protocol</th>
              <th>Port Range</th>
              <th>Destination</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(rule, index) in securityGroup.egress_rules" :key="index">
              <td>{{ formatProtocol(rule.ip_protocol) }}</td>
              <td>{{ formatPortRange(rule) }}</td>
              <td>{{ formatSource(rule) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Usage Details -->
      <div v-if="!securityGroup.is_unused && securityGroup.usage_info" class="usage-section">
        <h3>Usage Details</h3>
        <div v-if="securityGroup.usage_info.attached_to_instances && securityGroup.usage_info.attached_to_instances.length > 0">
          <h4>EC2 Instances</h4>
          <ul>
            <li v-for="instance in securityGroup.usage_info.attached_to_instances" :key="instance">
              {{ instance }}
            </li>
          </ul>
        </div>
        <div v-if="securityGroup.usage_info.attached_to_network_interfaces && securityGroup.usage_info.attached_to_network_interfaces.length > 0">
          <h4>Network Interfaces</h4>
          <ul>
            <li v-for="eni in securityGroup.usage_info.attached_to_network_interfaces" :key="eni">
              {{ eni }}
            </li>
          </ul>
        </div>
        <div v-if="securityGroup.usage_info.referenced_by_security_groups && securityGroup.usage_info.referenced_by_security_groups.length > 0">
          <h4>Referenced by Security Groups</h4>
          <ul>
            <li v-for="sgRef in securityGroup.usage_info.referenced_by_security_groups" :key="sgRef">
              {{ sgRef }}
            </li>
          </ul>
        </div>
      </div>

      <!-- Actions -->
      <div class="actions">
        <button @click="refreshSecurityGroup" class="btn btn-secondary" :disabled="loading">
          {{ loading ? 'Refreshing...' : 'Refresh' }}
        </button>
        <button
          v-if="canModify && !securityGroup.is_default"
          @click="deleteSecurityGroup"
          class="btn btn-danger"
          :disabled="loading"
        >
          <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z"/>
          </svg>
          Delete Security Group
        </button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SecurityGroupDetail',
  props: ['accountId', 'region', 'groupId'],
  data() {
    return {
      securityGroup: null,
      loading: true,
      error: null
    }
  },
  async mounted() {
    console.log('SecurityGroupDetail mounted with props:', this.accountId, this.region, this.groupId)
    await this.loadSecurityGroup()
  },
  methods: {
    async loadSecurityGroup() {
      try {
        this.loading = true
        this.error = null

        console.log('Loading security group:', this.accountId, this.region, this.groupId)
        const response = await fetch(`/api/accounts/${this.accountId}/regions/${this.region}/security-groups/${this.groupId}`)

        if (!response.ok) {
          const errorData = await response.json()
          throw new Error(errorData.error || 'Failed to load security group')
        }

        this.securityGroup = await response.json()
        console.log('Loaded security group:', this.securityGroup)

      } catch (err) {
        console.error('Error loading security group:', err)
        this.error = err.message
      } finally {
        this.loading = false
      }
    },

    async refreshSecurityGroup() {
      if (this.canModify) {
        try {
          await fetch(`/api/cache/accounts/${this.accountId}/security-groups/invalidate`, { method: 'POST' })
        } catch (error) {
          console.warn('Failed to invalidate account security groups cache:', error)
          try {
            await fetch('/api/cache/security-groups/invalidate', { method: 'POST' })
          } catch (fallbackError) {
            console.warn('Failed to invalidate fallback cache:', fallbackError)
          }
        }
      }
      await this.loadSecurityGroup()
    },

    async deleteSecurityGroup() {
      const sg = this.securityGroup

      // Create detailed confirmation message
      let confirmMessage = `⚠️ DELETE SECURITY GROUP ⚠️\n\n`
      confirmMessage += `You are about to PERMANENTLY DELETE:\n`
      confirmMessage += `• Name: ${sg.group_name}\n`
      confirmMessage += `• ID: ${sg.group_id}\n`
      confirmMessage += `• Account: ${sg.account_name} (${sg.account_id})\n`
      confirmMessage += `• Region: ${sg.region}\n`
      confirmMessage += `• VPC: ${sg.vpc_id || 'EC2-Classic'}\n\n`

      if (!sg.is_unused) {
        confirmMessage += `🚨 WARNING: This security group is CURRENTLY IN USE!\n`
        if (sg.usage_info && sg.usage_info.total_attachments > 0) {
          confirmMessage += `Attached to ${sg.usage_info.total_attachments} resource(s).\n`
        }
        confirmMessage += `Deleting it may break existing resources.\n\n`
      } else {
        confirmMessage += `✅ This security group is unused and safe to delete.\n\n`
      }

      confirmMessage += `This action cannot be undone!\n\n`
      confirmMessage += `Do you want to proceed?`

      if (!confirm(confirmMessage)) {
        return
      }

      this.loading = true
      try {
        const response = await fetch(`/api/accounts/${this.accountId}/regions/${this.region}/security-groups/${this.groupId}`, {
          method: 'DELETE'
        })

        if (!response.ok) {
          const errorData = await response.json()
          throw new Error(errorData.error || 'Failed to delete security group')
        }

        const result = await response.json()
        alert(result.message || 'Security group deleted successfully')
        this.$router.push('/security-groups')
      } catch (error) {
        console.error('Failed to delete security group:', error)
        alert(`Failed to delete security group: ${error.message}`)
      } finally {
        this.loading = false
      }
    },

    formatProtocol(protocol) {
      if (protocol === '-1') return 'All'
      return protocol.toUpperCase()
    },

    formatPortRange(rule) {
      if (rule.ip_protocol === '-1') return 'All'
      if (rule.from_port === rule.to_port) return rule.from_port.toString()
      return `${rule.from_port}-${rule.to_port}`
    },

    formatSource(rule) {
      if (rule.cidr_ipv4) {
        if (rule.cidr_ipv4 === '0.0.0.0/0') return '0.0.0.0/0 (Anywhere IPv4)'
        return rule.cidr_ipv4
      }
      if (rule.cidr_ipv6) {
        if (rule.cidr_ipv6 === '::/0') return '::/0 (Anywhere IPv6)'
        return rule.cidr_ipv6
      }
      if (rule.group_id) {
        return `${rule.group_id} (Security Group)`
      }
      return 'Unknown'
    }
  }
}
</script>

<style scoped>
.security-group-detail {
  padding: 2rem;
  max-width: 1200px;
  margin: 0 auto;
}

.breadcrumb {
  margin-bottom: 2rem;
}

.breadcrumb a {
  color: var(--color-btn-primary);
  text-decoration: none;
}

.breadcrumb a:hover {
  text-decoration: underline;
}

.params {
  background: var(--color-bg-secondary);
  padding: 1rem;
  border-radius: 8px;
  margin-bottom: 2rem;
}

.params p {
  margin: 0.5rem 0;
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4rem;
  gap: 1rem;
}

.spinner {
  width: 32px;
  height: 32px;
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
  border: 1px solid rgba(239, 68, 68, 0.3);
  padding: 2rem;
  border-radius: 8px;
  text-align: center;
}

.retry-btn {
  background: var(--color-btn-danger);
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
}

.content {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.info-section {
  background: var(--color-bg-secondary);
  padding: 1.5rem;
  border-radius: 8px;
}

.info-section h2 {
  margin-top: 0;
  color: var(--color-text-primary);
}

.used {
  color: var(--color-success);
  font-weight: 600;
}

.unused {
  color: var(--color-warning);
  font-weight: 600;
}

.open {
  color: var(--color-danger);
  font-weight: 600;
}

.closed {
  color: var(--color-success);
  font-weight: 600;
}

.open-ports-section {
  background: rgba(249, 115, 22, 0.1);
  border: 1px solid rgba(249, 115, 22, 0.3);
  padding: 1.5rem;
  border-radius: 8px;
}

.open-ports-section h3 {
  color: var(--color-danger);
  margin-top: 0;
}

.port-item {
  background: var(--color-bg-primary);
  padding: 1rem;
  margin: 0.5rem 0;
  border-radius: 6px;
  border: 1px solid rgba(249, 115, 22, 0.2);
}

.rules-section {
  background: var(--color-bg-secondary);
  padding: 1.5rem;
  border-radius: 8px;
}

.rules-section h3 {
  margin-top: 0;
  color: var(--color-text-primary);
}

.rules-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 1rem;
}

.rules-table th {
  background: var(--color-bg-tertiary);
  padding: 0.75rem;
  text-align: left;
  border-bottom: 2px solid var(--color-border);
}

.rules-table td {
  padding: 0.75rem;
  border-bottom: 1px solid var(--color-border);
}

.rules-table tr:hover {
  background: var(--color-bg-tertiary);
}

.no-rules {
  color: var(--color-text-secondary);
  font-style: italic;
  padding: 1rem 0;
}

.usage-section {
  background: var(--color-bg-secondary);
  padding: 1.5rem;
  border-radius: 8px;
}

.usage-section h3 {
  margin-top: 0;
  color: var(--color-text-primary);
}

.usage-section h4 {
  color: var(--color-text-primary);
  margin-bottom: 0.5rem;
}

.usage-section ul {
  margin: 0.5rem 0 1.5rem 1rem;
}

.usage-section li {
  font-family: monospace;
  background: var(--color-bg-tertiary);
  padding: 0.25rem 0.5rem;
  margin: 0.25rem 0;
  border-radius: 4px;
  display: inline-block;
}

.actions {
  display: flex;
  gap: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
}

.btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-bg-secondary);
}

.btn-danger {
  background: var(--color-btn-danger);
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background: var(--color-btn-danger-hover);
}

@media (max-width: 768px) {
  .security-group-detail {
    padding: 1rem;
  }

  .actions {
    flex-direction: column;
  }

  .rules-table {
    font-size: 0.875rem;
  }
}
</style>