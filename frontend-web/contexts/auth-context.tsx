"use client"

import { createContext, useContext, useEffect, useState, useCallback } from "react"
import type { ReactNode } from "react"
import { authApi } from "@/lib/api/auth"
import type { UserResponse, LoginRequest, RegisterRequest } from "@/lib/api/types"
import { ApiRequestError } from "@/lib/api/client"

interface AuthContextType {
  user: UserResponse | null
  isLoading: boolean
  isAuthenticated: boolean
  login: (data: LoginRequest) => Promise<void>
  register: (data: RegisterRequest) => Promise<void>
  logout: () => void
  error: string | null
  clearError: () => void
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const storedUser = authApi.getStoredUser()
    if (storedUser && authApi.isAuthenticated()) {
      setUser(storedUser)
    }
    setIsLoading(false)

    // Listen for unauthorized events (handled by axios interceptor)
    const handleUnauthorized = () => {
      setUser(null)
      setError("Session expired. Please login again.")
    }

    window.addEventListener("auth:unauthorized", handleUnauthorized)
    return () => {
      window.removeEventListener("auth:unauthorized", handleUnauthorized)
    }
  }, [])

  const login = useCallback(async (data: LoginRequest) => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await authApi.login(data)
      setUser(response.user)
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message)
      } else if (err instanceof Error) {
        setError(err.message)
      } else {
        setError("An unexpected error occurred")
      }
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [])

  const register = useCallback(async (data: RegisterRequest) => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await authApi.register(data)
      setUser(response.user)
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message)
      } else if (err instanceof Error) {
        setError(err.message)
      } else {
        setError("An unexpected error occurred")
      }
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [])

  const logout = useCallback(() => {
    authApi.logout()
    setUser(null)
    setError(null)
  }, [])

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        login,
        register,
        logout,
        error,
        clearError,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return context
}
