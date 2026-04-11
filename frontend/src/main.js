import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Login from './components/Login.vue'
import AllUsers from './components/AllUsers.vue'
import UserDetail from './components/UserDetail.vue'
import PublicIPs from './components/PublicIPs.vue'
import SecurityGroups from './components/SecurityGroups.vue'
import SecurityGroupDetail from './components/SecurityGroupDetail.vue'
import Snapshots from './components/Snapshots.vue'
import EC2Instances from './components/EC2Instances.vue'
import EBSVolumes from './components/EBSVolumes.vue'
import S3Buckets from './components/S3Buckets.vue'
import Roles from './components/Roles.vue'
import LoadBalancers from './components/LoadBalancers.vue'
import VPCs from './components/VPCs.vue'
import NATGateways from './components/NATGateways.vue'
import Route53Domains from './components/Route53Domains.vue'
import Route53DomainDetail from './components/Route53DomainDetail.vue'
import AzureEnterpriseApps from './components/AzureEnterpriseApps.vue'
import AzureVMs from './components/AzureVMs.vue'
import AzureStorage from './components/AzureStorage.vue'
import SSOUsers from './components/SSOUsers.vue'
import SSOGroups from './components/SSOGroups.vue'
import SSOUserDetail from './components/SSOUserDetail.vue'
import SSOGroupDetail from './components/SSOGroupDetail.vue'
import SSOUserAssignments from './components/SSOUserAssignments.vue'
import SSOAccountAssignments from './components/SSOAccountAssignments.vue'
import SSOAccountDetail from './components/SSOAccountDetail.vue'
import AdminUsers from './components/AdminUsers.vue'
import APITokens from './components/APITokens.vue'
import { authStore } from './authStore'

const routes = [
  { path: '/login', name: 'Login', component: Login },
  { path: '/', redirect: '/aws/users' },
  // AWS Routes
  { path: '/aws/users', name: 'AllUsers', component: AllUsers, meta: { requiresAuth: true } },
  { path: '/aws/public-ips', name: 'PublicIPs', component: PublicIPs, meta: { requiresAuth: true } },
  { path: '/aws/security-groups', name: 'SecurityGroups', component: SecurityGroups, meta: { requiresAuth: true } },
  { path: '/aws/security-groups/:accountId/:region/:groupId', name: 'SecurityGroupDetail', component: SecurityGroupDetail, props: true, meta: { requiresAuth: true } },
  { path: '/aws/snapshots', name: 'Snapshots', component: Snapshots, meta: { requiresAuth: true } },
  { path: '/aws/ec2-instances', name: 'EC2Instances', component: EC2Instances, meta: { requiresAuth: true } },
  { path: '/aws/ebs-volumes', name: 'EBSVolumes', component: EBSVolumes, meta: { requiresAuth: true } },
  { path: '/aws/s3-buckets', name: 'S3Buckets', component: S3Buckets, meta: { requiresAuth: true } },
  { path: '/aws/roles', name: 'Roles', component: Roles, meta: { requiresAuth: true } },
  { path: '/aws/load-balancers', name: 'LoadBalancers', component: LoadBalancers, meta: { requiresAuth: true } },
  { path: '/aws/vpcs', name: 'VPCs', component: VPCs, meta: { requiresAuth: true } },
  { path: '/aws/nat-gateways', name: 'NATGateways', component: NATGateways, meta: { requiresAuth: true } },
  { path: '/aws/route53-domains', name: 'Route53Domains', component: Route53Domains, meta: { requiresAuth: true } },
  { path: '/aws/route53-domains/:accountId/:hostedZoneId', name: 'Route53DomainDetail', component: Route53DomainDetail, props: true, meta: { requiresAuth: true } },
  { path: '/aws/accounts/:accountId/users/:username', name: 'UserDetail', component: UserDetail, props: true, meta: { requiresAuth: true } },
  // SSO Routes
  { path: '/aws/sso/users', name: 'SSOUsers', component: SSOUsers, meta: { requiresAuth: true } },
  { path: '/aws/sso/users/:userId', name: 'SSOUserDetail', component: SSOUserDetail, props: true, meta: { requiresAuth: true } },
  { path: '/aws/sso/groups', name: 'SSOGroups', component: SSOGroups, meta: { requiresAuth: true } },
  { path: '/aws/sso/groups/:groupId', name: 'SSOGroupDetail', component: SSOGroupDetail, props: true, meta: { requiresAuth: true } },
  { path: '/aws/sso/user-assignments', name: 'SSOUserAssignments', component: SSOUserAssignments, meta: { requiresAuth: true } },
  { path: '/aws/sso/account-assignments', name: 'SSOAccountAssignments', component: SSOAccountAssignments, meta: { requiresAuth: true } },
  { path: '/aws/sso/accounts/:accountId/assignments', name: 'SSOAccountDetail', component: SSOAccountDetail, props: true, meta: { requiresAuth: true } },
  // Azure Routes
  { path: '/azure/enterprise-apps', name: 'AzureEnterpriseApps', component: AzureEnterpriseApps, meta: { requiresAuth: true } },
  { path: '/azure/vms', name: 'AzureVMs', component: AzureVMs, meta: { requiresAuth: true } },
  { path: '/azure/storage', name: 'AzureStorage', component: AzureStorage, meta: { requiresAuth: true } },
  { path: '/account/tokens', name: 'APITokens', component: APITokens, meta: { requiresAuth: true } },
  { path: '/admin/users', name: 'AdminUsers', component: AdminUsers, meta: { requiresAuth: true, requiresAdmin: true } },
  // Legacy route redirects for backward compatibility
  { path: '/public-ips', redirect: '/aws/public-ips' },
  { path: '/security-groups', redirect: '/aws/security-groups' },
  { path: '/snapshots', redirect: '/aws/snapshots' },
  { path: '/ec2-instances', redirect: '/aws/ec2-instances' },
  { path: '/ebs-volumes', redirect: '/aws/ebs-volumes' },
  { path: '/s3-buckets', redirect: '/aws/s3-buckets' },
  { path: '/roles', redirect: '/aws/roles' },
  { path: '/load-balancers', redirect: '/aws/load-balancers' },
  { path: '/vpcs', redirect: '/aws/vpcs' },
  { path: '/nat-gateways', redirect: '/aws/nat-gateways' },
  { path: '/route53-domains', redirect: '/aws/route53-domains' },
  { path: '/accounts/:accountId/users/:username', redirect: to => `/aws/accounts/${to.params.accountId}/users/${to.params.username}` }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard to check authentication
router.beforeEach(async (to, from, next) => {
  // Skip auth check for login page
  if (to.path === '/login') {
    // Check if already authenticated
    try {
      const response = await fetch('/api/auth/check', {
        credentials: 'include'
      })
      const data = await response.json()
      if (data.authenticated) {
        authStore.authenticated = true
        authStore.username = data.username || ''
        authStore.role = data.role || ''
        next('/aws/users')
        return
      }
    } catch (err) {
      // Ignore errors, allow login page
    }
    next()
    return
  }

  // Check authentication for protected routes
  if (to.meta.requiresAuth) {
    try {
      const response = await fetch('/api/auth/check', {
        credentials: 'include'
      })
      const data = await response.json()
      if (data.authenticated) {
        authStore.authenticated = true
        authStore.username = data.username || ''
        authStore.role = data.role || ''
        if (to.meta.requiresAdmin && data.role !== 'admin') {
          next('/aws/users')
          return
        }
        next()
      } else {
        authStore.authenticated = false
        authStore.username = ''
        authStore.role = ''
        next('/login')
      }
    } catch (err) {
      authStore.authenticated = false
      authStore.username = ''
      authStore.role = ''
      next('/login')
    }
  } else {
    next()
  }
})

const app = createApp(App)
app.use(router)
app.mixin({
  computed: {
    canModify() {
      const r = authStore.role
      return r === 'admin' || r === 'editor'
    }
  }
})
app.mount('#app')