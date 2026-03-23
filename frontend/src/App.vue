<template>
  <div>
    <header class="header">
      <div style="display: flex; align-items: center; gap: 0.75rem;">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
        </svg>
        <h1 style="margin: 0;">Blink Dashboard</h1>
      </div>
      <button v-if="token" @click="logout" class="btn">Logout</button>
    </header>

    <div v-if="!token" class="login-container">
      <div class="card" style="max-width: 400px; margin: 4rem auto;">
        <h2>Authentication</h2>
        <p style="color: var(--text-muted); margin-bottom: 1.5rem; font-size: 0.875rem;">
          Enter your administrator password to access the secure vault.
        </p>
        <form @submit.prevent="login">
          <div style="margin-bottom: 1.5rem;">
            <label>Admin Password</label>
            <input type="password" v-model="password" required autofocus placeholder="Enter password..." />
          </div>
          <div v-if="error" style="color: var(--error); margin-bottom: 1rem; font-size: 0.875rem; background: rgba(239, 68, 68, 0.1); padding: 0.5rem; border-radius: 4px;">
            {{ error }}
          </div>
          <button type="submit" class="btn" style="width: 100%; padding: 0.75rem;">Authenticate</button>
        </form>
      </div>
    </div>

    <div v-else class="dashboard">
      <div class="stats-grid">
        <div class="card stat-card">
          <h3>Total Active Files</h3>
          <div class="stat-value">{{ stats.total_files }}</div>
        </div>
        <div class="card stat-card">
          <h3>Storage Consumed</h3>
          <div class="stat-value">{{ formatBytes(stats.storage_used) }}</div>
        </div>
        <div class="card stat-card">
          <h3>Total Bandwidth</h3>
          <div class="stat-value">{{ formatBytes(stats.bandwidth_consumed) }}</div>
        </div>
      </div>

      <div class="grid-2">
        <div class="card">
          <h2>Global Limitations</h2>
          <p style="color: var(--text-muted); font-size: 0.875rem; margin-bottom: 1.5rem;">
            Default values applied when a client upload request omits expiration headers.
          </p>
          <form @submit.prevent="saveSettings">
            <div style="margin-bottom: 1.25rem;">
              <label>Default Max Downloads</label>
              <input type="number" v-model.number="settings.max_downloads" min="1" />
            </div>
            <div style="margin-bottom: 1.5rem;">
              <label>Default Expiry (minutes)</label>
              <input type="number" v-model.number="settings.default_expiry_m" min="1" />
            </div>
            <h3 style="margin-top: 1.5rem; margin-bottom: 1rem; font-size: 1rem;">Maximum Allowed Limits</h3>
            <div style="margin-bottom: 1.25rem;">
              <label>Maximum Allowed Downloads</label>
              <input type="number" v-model.number="settings.max_allowed_downloads" min="1" />
            </div>
            <div style="margin-bottom: 1.5rem;">
              <label>Maximum Allowed Expiry (minutes)</label>
              <input type="number" v-model.number="settings.max_allowed_expiry_m" min="1" />
            </div>
            <div style="display: flex; align-items: center; gap: 1rem;">
              <button type="submit" class="btn">Update Limits</button>
              <span v-if="settingsSaved" style="color: var(--success); font-size: 0.875rem;">Configuration Saved!</span>
            </div>
          </form>
        </div>

        <div class="card" style="display: flex; flex-direction: column;">
          <h2>Vault Inventory</h2>
          <div style="overflow-x: auto; flex-grow: 1;">
            <table>
              <thead>
                <tr>
                  <th>UUID / NanoID</th>
                  <th>Payload Size</th>
                  <th>Downloads Left</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="file in files" :key="file.id">
                  <td style="font-family: monospace;">{{ file.id }}</td>
                  <td>{{ formatBytes(file.size) }}</td>
                  <td>
                    <span style="background: rgba(59, 130, 246, 0.1); color: var(--accent); padding: 0.2rem 0.5rem; border-radius: 9999px; font-size: 0.875rem; font-weight: 500;">
                      {{ file.downloads_left }}
                    </span>
                  </td>
                  <td>
                    <button @click="deleteFile(file.id)" class="btn btn-danger" style="padding: 0.35rem 0.75rem; font-size: 0.75rem;">Purge</button>
                  </td>
                </tr>
                <tr v-if="files.length === 0">
                  <td colspan="4" style="text-align: center; color: var(--text-muted); padding: 3rem 1rem;">
                    No secure files currently in the vault.
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const token = ref(localStorage.getItem('blink_token') || '')
const password = ref('')
const error = ref('')

const stats = ref({ total_files: 0, storage_used: 0, bandwidth_consumed: 0 })
const settings = ref({ max_downloads: 1, default_expiry_m: 10, max_allowed_downloads: 100, max_allowed_expiry_m: 1440 })
const settingsSaved = ref(false)
const files = ref([])

const fetchAuth = async (url, options = {}) => {
  const res = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'Authorization': `Bearer ${token.value}`
    }
  })
  if (res.status === 401) {
    logout()
    throw new Error('Unauthorized')
  }
  return res
}

const login = async () => {
  error.value = ''
  try {
    const res = await fetch('/admin/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: password.value })
    })
    
    if (!res.ok) {
      if (res.status === 401) throw new Error('Invalid authentication credentials')
      throw new Error('An error occurred. Make sure the backend is running.')
    }
    
    const data = await res.json()
    token.value = data.token
    localStorage.setItem('blink_token', data.token)
    loadDashboard()
  } catch (e) {
    error.value = e.message
  }
}

const logout = () => {
  token.value = ''
  localStorage.removeItem('blink_token')
}

const loadDashboard = async () => {
  if (!token.value) return
  
  try {
    const [statsRes, settingsRes, filesRes] = await Promise.all([
      fetchAuth('/admin/api/stats'),
      fetchAuth('/admin/api/settings'),
      fetchAuth('/admin/api/files')
    ])
    
    stats.value = await statsRes.json()
    settings.value = await settingsRes.json()
    const filesData = await filesRes.json()
    files.value = filesData || []
  } catch (e) {
    console.error(e)
  }
}

const saveSettings = async () => {
  try {
    await fetchAuth('/admin/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(settings.value)
    })
    settingsSaved.value = true
    setTimeout(() => settingsSaved.value = false, 3000)
  } catch (e) {
    console.error(e)
  }
}

const deleteFile = async (id) => {
  if (!confirm('Are you sure you want to permanently purge this file?')) return
  try {
    await fetchAuth(`/admin/api/files/${id}`, { method: 'DELETE' })
    await loadDashboard()
  } catch (e) {
    console.error(e)
  }
}

const formatBytes = (bytes) => {
  if (bytes === 0 || !bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onMounted(() => {
  if (token.value) loadDashboard()
})
</script>

<style scoped>
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.stat-card h3 {
  color: var(--text-muted);
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.5rem;
}

.stat-value {
  font-size: 2.25rem;
  font-weight: 600;
  color: var(--text-main);
}

.grid-2 {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 1.5rem;
}

@media (max-width: 768px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
}
</style>
