export const getUserId = () => localStorage.getItem('user_id') || ''
export const getToken   = () => localStorage.getItem('token')  || ''
export const isLoggedIn = () => !!getToken()
