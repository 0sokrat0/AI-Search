export interface User {
  id: string
  email: string
  name: string
  roles: Role[]
  tenantID: string
  createdAt: string
}

export type Role = 'super_admin' | 'employee'

export interface BackendUser {
  id: string
  tenant_id: string
  email: string
  name: string
  roles: Role[]
  is_active: boolean
  created_at: string
  updated_at: string
  last_login?: string | null
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  accessToken: string
  refreshToken: string
  user: User
}

export interface RefreshTokenRequest {
  refreshToken: string
}

export interface RefreshTokenResponse {
  accessToken: string
  refreshToken: string
}
