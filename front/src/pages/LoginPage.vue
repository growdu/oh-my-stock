<template>
  <div class="login-container">
    <div class="login-box">
      <div class="login-logo">
        <img src="@/assets/logo.png" alt="logo" class="logo-img" />
        <h2 class="logo-title">oh-my-stock</h2>
      </div>

      <el-card class="login-card" shadow="hover">
        <el-tabs v-model="activeTab" type="card">
          <!-- 登录 Tab -->
          <el-tab-pane label="登录" name="login">
            <el-form
              :model="loginForm"
              :rules="rulesLogin"
              ref="loginFormRef"
              label-width="80px"
              class="login-form"
            >
              <el-form-item label="用户名" prop="username">
                <el-input v-model="loginForm.username" placeholder="请输入用户名" clearable style="width: 300px"/>
              </el-form-item>
              <el-form-item label="密码" prop="password">
                <el-input v-model="loginForm.password" type="password" show-password placeholder="请输入密码" clearable style="width: 300px"/>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" style="width: 300px" @click="handleLogin">登录</el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <!-- 注册 Tab -->
          <el-tab-pane label="注册" name="register">
            <el-form
              :model="registerForm"
              :rules="rulesRegister"
              ref="registerFormRef"
              label-width="80px"
              class="login-form"
            >
              <el-form-item label="用户名" prop="username">
                <el-input v-model="registerForm.username" placeholder="请输入用户名" clearable style="width: 300px"/>
              </el-form-item>
              <el-form-item label="密码" prop="password">
                <el-input v-model="registerForm.password" type="password" show-password placeholder="请输入密码" clearable style="width: 300px"/>
              </el-form-item>
              <el-form-item label="邮箱">
                <el-input v-model="registerForm.email" placeholder="请输入邮箱" clearable style="width: 300px"/>
              </el-form-item>
              <el-form-item label="手机号">
                <el-input v-model="registerForm.phone" placeholder="请输入手机号" clearable style="width: 300px"/>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" style="width: 300px" @click="handleRegister">注册</el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const router = useRouter()
const activeTab = ref('login')

// 登录表单
const loginFormRef = ref(null)
const loginForm = ref({ username: '', password: '' })
const rulesLogin = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

// 注册表单
const registerFormRef = ref(null)
const registerForm = ref({ username: '', password: '', email: '', phone: '' })
const rulesRegister = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

// 登录请求
const handleLogin = async () => {
  loginFormRef.value.validate(async (valid) => {
    if (!valid) return
    try {
      const res = await request.post('/user/login', loginForm.value)
      // 后端返回 token + user_id
      localStorage.setItem('token', res.data.token)
      localStorage.setItem('user_id', res.data.user_id)
      localStorage.setItem('username', loginForm.value.username)
      ElMessage.success(loginForm.value.username+'登录成功')
      router.push('/home')
    } catch (err) {
      ElMessage.error(err.response?.data?.error || '登录失败')
    }
  })
}

// 注册请求
const handleRegister = async () => {
  registerFormRef.value.validate(async (valid) => {
    if (!valid) return
    try {
      await request.post('/user/register', registerForm.value)
      ElMessage.success('注册成功，请登录')
      activeTab.value = 'login'
      registerForm.value = { username: '', password: '', email: '', phone: '' }
    } catch (err) {
      ElMessage.error(err.response?.data?.error || '注册失败')
    }
  })
}

// 页面加载时检查登录状态
onMounted(() => {
  const token = localStorage.getItem('token')
  if (token) {
    router.push('/home')
  }
})
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: linear-gradient(135deg, #74ebd5 0%, #9face6 100%);
}

.login-box {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.login-logo {
  text-align: center;
  margin-bottom: 20px;
}
.logo-img {
  width: 80px;
  height: 80px;
  border-radius: 50%;
}
.logo-title {
  margin-top: 10px;
  font-size: 24px;
  font-weight: bold;
  color: #fff;
}

.login-card {
  padding: 20px;
  width: 420px;
  border-radius: 16px;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
  background: #fff;
}

.login-form {
  display: flex;
  flex-direction: column;
}
</style>
