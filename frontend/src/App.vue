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
            <div style="margin-bottom: 1.5rem;">
              <label>Maximum File Size (MB)</label>
              <input type="number" v-model.number="settings.max_allowed_file_size_mb" min="1" />
            </div>
            <div style="display: flex; align-items: center; gap: 1rem;">
              <button type="submit" class="btn">Update Limits</button>
              <span v-if="settingsSaved" style="color: var(--success); font-size: 0.875rem;">Configuration Saved!</span>
            </div>
          </form>
        </div>

        <div class="card" style="display: flex; flex-direction: column;">
          <h2>Vault Inventory</h2>

          <div style="background: var(--bg-body); padding: 1rem; border-radius: 8px; margin-bottom: 1.5rem; border: 1px solid var(--border);">
            <h3 style="margin-top: 0; margin-bottom: 1rem; font-size: 1rem;">Admin Upload</h3>
            <form @submit.prevent="uploadAdminFile" style="display: flex; gap: 1rem; align-items: flex-end; flex-wrap: wrap;">
               <div>
                  <label style="font-size: 0.75rem;">File</label><br>
                  <input type="file" ref="fileInput" required style="font-size: 0.875rem;" />
               </div>
               <div>
                  <label style="font-size: 0.75rem;">Downloads</label><br>
                  <input type="number" v-model.number="uploadDownloads" min="1" style="width: 80px;" />
               </div>
               <div>
                  <label style="font-size: 0.75rem;">Expiry (mins)</label><br>
                  <input type="number" v-model.number="uploadExpiry" min="1" style="width: 100px;" />
               </div>
               <button type="submit" class="btn" :disabled="isUploading">
                 {{ isUploading ? 'Uploading...' : 'Upload' }}
               </button>
               <button type="button" class="btn btn-danger" :disabled="!isUploading" @click="cancelUpload">
                 Cancel
               </button>
            </form>
            <div v-if="isUploading || uploadStatus === 'success' || uploadStatus === 'canceled'" class="upload-progress-wrap">
              <div class="upload-progress-head">
                <span>{{ uploadProgress }}%</span>
                <span>{{ formatBytes(uploadLoaded) }} / {{ formatBytes(uploadTotal) }}</span>
              </div>
              <div class="upload-progress-bar">
                <div class="upload-progress-fill" :style="{ width: `${uploadProgress}%` }"></div>
              </div>
            </div>
            <div v-if="uploadError" style="color: var(--error); margin-top: 0.5rem; font-size: 0.875rem;">{{ uploadError }}</div>
            <div v-else-if="uploadStatus === 'success'" style="color: var(--success); margin-top: 0.5rem; font-size: 0.875rem;">Upload completed successfully.</div>
          </div>

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
                    <div style="display: flex; gap: 0.5rem;">
                      <button v-if="file.uploaded_by_admin" @click="reconfigureFile(file)" class="btn" style="padding: 0.35rem 0.75rem; font-size: 0.75rem;">Reconfigure</button>
                      <button @click="deleteFile(file.id)" class="btn btn-danger" style="padding: 0.35rem 0.75rem; font-size: 0.75rem;">Purge</button>
                    </div>
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
const settings = ref({ max_downloads: 1, default_expiry_m: 10, max_allowed_downloads: 100, max_allowed_expiry_m: 1440, max_allowed_file_size_mb: 6144 })
const settingsSaved = ref(false)
const files = ref([])

const fileInput = ref(null)
const uploadDownloads = ref(10)
const uploadExpiry = ref(1440)
const isUploading = ref(false)
const uploadError = ref('')
const uploadStatus = ref('idle')
const uploadProgress = ref(0)
const uploadLoaded = ref(0)
const uploadTotal = ref(0)
const activeUploadXhr = ref(null)

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

const parseUploadError = (status, body) => {
  const text = (body || '').trim()
  if (status === 413) {
    return `File is larger than the configured limit (${settings.value.max_allowed_file_size_mb} MB).`
  }
  if (status === 401) {
    return 'Unauthorized. Please log in again.'
  }
  if (status === 408 || status === 504) {
    return 'Upload timed out before completion.'
  }
  if (text) {
    return text
  }
  return `Upload failed with status ${status}.`
}

const cancelUpload = () => {
  if (!activeUploadXhr.value) return
  activeUploadXhr.value.abort()
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

const reconfigureFile = async (file) => {
  const newDownloads = prompt(`New Downloads Left for ${file.id}?`, file.downloads_left)
  if (!newDownloads) return
  
  const currentExpiryTime = new Date(file.expiry_time).getTime()
  const now = new Date().getTime()
  const minsLeft = Math.max(1, Math.floor((currentExpiryTime - now) / 60000))
  
  const newExpiry = prompt(`New Expiry (minutes from now) for ${file.id}?`, minsLeft)
  if (!newExpiry) return
  
  const dl = parseInt(newDownloads, 10)
  const exp = parseInt(newExpiry, 10)
  if (isNaN(dl) || isNaN(exp)) return
  
  try {
    const res = await fetchAuth(`/admin/api/files/${file.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ downloads_left: dl, expiry_minutes: exp })
    })
    
    if (!res.ok) {
        alert(await res.text())
    }
    await loadDashboard()
  } catch (e) {
    console.error(e)
    alert(e.message)
  }
}

const uploadAdminFile = async () => {
  if (!fileInput.value.files[0]) return
  isUploading.value = true
  uploadError.value = ''
  uploadStatus.value = 'uploading'
  uploadProgress.value = 0
  uploadLoaded.value = 0
  uploadTotal.value = 0
  
  const file = fileInput.value.files[0]
  const targetUrl = `/admin/api/upload/${encodeURIComponent(file.name)}`
  
  try {
    await new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      activeUploadXhr.value = xhr

      xhr.open('POST', targetUrl)
      xhr.setRequestHeader('Authorization', `Bearer ${token.value}`)
      xhr.setRequestHeader('Max-Downloads', String(uploadDownloads.value))
      xhr.setRequestHeader('Expiry', String(uploadExpiry.value))

      xhr.upload.onprogress = (event) => {
        if (!event.lengthComputable) return
        uploadLoaded.value = event.loaded
        uploadTotal.value = event.total
        uploadProgress.value = Math.min(100, Math.round((event.loaded / event.total) * 100))
      }

      xhr.onload = () => {
        activeUploadXhr.value = null
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve()
          return
        }
        reject(new Error(parseUploadError(xhr.status, xhr.responseText)))
      }

      xhr.onerror = () => {
        activeUploadXhr.value = null
        reject(new Error('Network error during upload.'))
      }

      xhr.onabort = () => {
        activeUploadXhr.value = null
        const abortError = new Error('Upload canceled by user.')
        abortError.name = 'AbortError'
        reject(abortError)
      }

      xhr.send(file)
    })

    fileInput.value.value = ''
    uploadStatus.value = 'success'
    uploadProgress.value = 100
    await loadDashboard()
  } catch (e) {
    if (e.name === 'AbortError') {
      uploadStatus.value = 'canceled'
      uploadError.value = e.message
    } else {
      uploadStatus.value = 'error'
      uploadError.value = e.message
    }
  } finally {
    isUploading.value = false
    activeUploadXhr.value = null
  }
}

const formatBytes = (bytes) => {
  if (bytes === 0 || !bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
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

.upload-progress-wrap {
  margin-top: 0.75rem;
}

.upload-progress-head {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-bottom: 0.4rem;
}

.upload-progress-bar {
  width: 100%;
  height: 10px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.2);
  overflow: hidden;
}

.upload-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #22c55e, #3b82f6);
  transition: width 0.2s ease-out;
}

@media (max-width: 768px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
}
</style>
