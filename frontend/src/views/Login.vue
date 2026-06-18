<template>
  <div class="auth-container">
    <router-link to="/" class="home-link">← 返回首页</router-link>
    <div class="auth-card">
      <div class="logo">
        <h1>文豪写作</h1>
        <p>AI 驱动的智能写作助手</p>
      </div>

      <div class="tabs">
        <button :class="{ active: mode === 'login' }" @click="switchMode('login')">登录</button>
        <button :class="{ active: mode === 'register' }" @click="switchMode('register')">注册</button>
      </div>

      <form v-if="mode === 'login'" @submit.prevent="handleLogin">
        <div class="form-group">
          <label>邮箱</label>
          <input v-model="loginForm.email" type="email" placeholder="请输入邮箱" required />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="loginForm.password" type="password" placeholder="请输入密码" required />
        </div>
        <p v-if="loginError" class="error-msg">{{ loginError }}</p>
        <button type="submit" class="submit-btn" :disabled="loginLoading">
          {{ loginLoading ? '登录中...' : '登 录' }}
        </button>
      </form>

      <form v-else @submit.prevent="handleRegister">
        <div class="form-group">
          <label>邮箱</label>
          <div class="code-row">
            <input v-model="registerForm.email" type="email" placeholder="请输入邮箱" required />
            <button type="button" class="code-btn" :disabled="countdown > 0" @click="sendCode">
              {{ countdown > 0 ? countdown + 's' : '发送验证码' }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>验证码</label>
          <input v-model="registerForm.code" type="text" placeholder="请输入6位验证码" maxlength="6" required />
        </div>
        <div class="form-group">
          <label>用户名</label>
          <input v-model="registerForm.username" type="text" placeholder="2-16个字符，支持中英文" required />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="registerForm.password" type="password" placeholder="至少8位，包含大小写字母和数字" required />
          <p class="hint">至少8位，需包含大写字母、小写字母和数字</p>
        </div>
        <p v-if="registerError" class="error-msg">{{ registerError }}</p>
        <p v-if="registerSuccess" class="success-msg">{{ registerSuccess }}</p>
        <button type="submit" class="submit-btn" :disabled="registerLoading">
          {{ registerLoading ? '注册中...' : '注 册' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const router = useRouter()
const route = useRoute()
const mode = ref(route.query.mode === 'register' ? 'register' : 'login')
const loginError = ref('')
const registerError = ref('')
const registerSuccess = ref('')
const loginLoading = ref(false)
const registerLoading = ref(false)
const countdown = ref(0)

const loginForm = reactive({ email: '', password: '' })
const registerForm = reactive({ email: '', code: '', username: '', password: '' })

function switchMode(m) {
  mode.value = m
  loginError.value = ''
  registerError.value = ''
  registerSuccess.value = ''
}

watch(() => route.query.mode, value => {
  switchMode(value === 'register' ? 'register' : 'login')
})

async function api(path, method, body) {
  const headers = { 'Content-Type': 'application/json' }
  const token = localStorage.getItem('token')
  if (token) headers['Authorization'] = 'Bearer ' + token
  const res = await fetch('/api/v1' + path, { method, headers, body: body ? JSON.stringify(body) : undefined })
  const data = await res.json()
  return { status: res.status, data }
}

async function sendCode() {
  if (!registerForm.email) {
    registerError.value = '请输入邮箱'
    return
  }
  const { status, data } = await api('/auth/send-code', 'POST', { email: registerForm.email })
  if (status === 200) {
    registerSuccess.value = '验证码已发送'
    registerError.value = ''
    countdown.value = 60
    const timer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) clearInterval(timer)
    }, 1000)
  } else {
    registerError.value = data.error || '发送失败'
  }
}

async function handleLogin() {
  loginLoading.value = true
  loginError.value = ''
  const { status, data } = await api('/auth/login', 'POST', loginForm)
  loginLoading.value = false
  if (status === 200) {
    localStorage.setItem('token', data.token)
    localStorage.setItem('role', data.role)
    router.push(data.role === 'admin' ? '/admin' : '/write')
  } else {
    loginError.value = data.error || '登录失败'
  }
}

async function handleRegister() {
  registerLoading.value = true
  registerError.value = ''
  registerSuccess.value = ''

  const nameLen = [...registerForm.username].length
  if (nameLen < 2 || nameLen > 16) {
    registerError.value = '用户名需为2-16个字符'
    registerLoading.value = false
    return
  }

  const { status, data } = await api('/auth/register', 'POST', registerForm)
  registerLoading.value = false
  if (status === 201) {
    localStorage.setItem('token', data.token)
    localStorage.setItem('role', data.role)
    router.push(data.role === 'admin' ? '/admin' : '/write')
  } else {
    registerError.value = data.error || '注册失败'
  }
}
</script>

<style scoped>
.auth-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #eff6ff, #dbeafe, #e0e7ff);
  position: relative;
}
.home-link {
  position: fixed;
  top: 28px;
  left: 32px;
  color: #475569;
  text-decoration: none;
  font-size: 14px;
  font-weight: 600;
}
.auth-card {
  width: 420px;
  padding: 40px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 4px 24px rgba(0,0,0,0.06);
}
.logo {
  text-align: center;
  margin-bottom: 36px;
}
.logo h1 {
  font-size: 36px;
  font-weight: 700;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: 4px;
}
.logo p {
  color: #64748b;
  margin-top: 8px;
  font-size: 14px;
  letter-spacing: 2px;
}
.tabs {
  display: flex;
  margin-bottom: 28px;
  border-bottom: 1px solid #e2e8f0;
}
.tabs button {
  flex: 1;
  padding: 12px 0;
  border: none;
  background: none;
  color: #94a3b8;
  font-size: 16px;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.3s;
}
.tabs button.active {
  color: #2563eb;
  border-bottom-color: #2563eb;
}
.form-group {
  margin-bottom: 20px;
}
.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  color: #475569;
}
.form-group input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
  color: #1e293b;
  font-size: 15px;
  outline: none;
  transition: border-color 0.3s;
  box-sizing: border-box;
}
.form-group input:focus {
  border-color: #2563eb;
}
.code-row {
  display: flex;
  gap: 12px;
}
.code-row input {
  flex: 1;
}
.code-btn {
  white-space: nowrap;
  padding: 12px 16px;
  border: 1px solid #2563eb;
  border-radius: 8px;
  background: transparent;
  color: #2563eb;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.3s;
}
.code-btn:hover { background: rgba(37,99,235,0.08); }
.code-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.submit-btn {
  width: 100%;
  padding: 14px;
  border: none;
  border-radius: 8px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  letter-spacing: 2px;
  transition: opacity 0.3s;
  margin-top: 8px;
}
.submit-btn:hover { opacity: 0.9; }
.submit-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.error-msg { color: #dc2626; font-size: 13px; margin-top: 4px; }
.success-msg { color: #16a34a; font-size: 13px; margin-top: 4px; }
.hint { font-size: 12px; color: #94a3b8; margin-top: 4px; }

@media (max-width: 640px) {
  .auth-container {
    padding: 24px 16px;
    align-items: flex-start;
  }
  .home-link {
    position: static;
    display: inline-block;
    margin-bottom: 16px;
  }
  .auth-card {
    width: 100%;
    padding: 28px 20px;
    border-radius: 14px;
  }
  .logo h1 {
    font-size: 30px;
    letter-spacing: 2px;
  }
  .logo p {
    letter-spacing: 1px;
  }
  .code-row {
    flex-direction: column;
  }
  .code-btn {
    width: 100%;
  }
}
</style>
