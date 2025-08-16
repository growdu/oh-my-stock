<template>
    <div class="login-container">
      <div class="login-box">
        <!-- Logo 区域 -->
        <div class="login-logo">
          <img src="@/assets/logo.png" alt="logo" class="logo-img" />
          <h2 class="logo-title">oh-my-stock</h2>
        </div>
  
        <!-- 登录卡片 -->
        <el-card class="login-card" shadow="hover">
          <el-form
            :model="loginForm"
            :rules="rules"
            ref="loginFormRef"
            label-width="80px"
            class="login-form"
          >
            <el-form-item label="用户名" prop="username">
              <el-input
                v-model="loginForm.username"
                placeholder="请输入用户名"
                clearable
                style="width: 300px"
              />
            </el-form-item>
            <el-form-item label="密  码" prop="password">
              <el-input
                v-model="loginForm.password"
                type="password"
                placeholder="请输入密码"
                show-password
                clearable
                style="width: 300px"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" style="width: 300px" @click="handleLogin">
                登录
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </div>
    </div>
  </template>
  
  <script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const router = useRouter()

const loginFormRef = ref(null)
const loginForm = ref({
  username: '',
  password: ''
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

// 模拟登录 API
const mockLogin = ({ username, password }) => {
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (username === 'admin' && password === '123456') {
        resolve({ token: 'mock-token' })
      } else {
        reject(new Error('用户名或密码错误'))
      }
    }, 500)
  })
}

const handleLogin = async () => {
  loginFormRef.value.validate(async (valid) => {
    if (!valid) return
    try {
      const res = await mockLogin(loginForm.value)
      localStorage.setItem('token', res.token)
      ElMessage.success('登录成功')
      router.push('/home') // 登录成功跳转到 home 页面
    } catch (err) {
      ElMessage.error(err.message)
    }
  })
}
</script>

  
  <style scoped>
  /* 整体背景渐变 */
  .login-container {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100vh;
    background: linear-gradient(135deg, #74ebd5 0%, #9face6 100%);
  }
  
  /* 登录区域盒子 */
  .login-box {
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  
  /* logo 区域 */
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
  
  /* 卡片样式 */
  .login-card {
    padding: 20px;
    width: 400px;
    border-radius: 16px;
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
    background: #fff;
  }
  
  .login-title {
    text-align: center;
    margin-bottom: 20px;
    font-size: 22px;
    font-weight: bold;
    color: #333;
  }
  
  </style>
  