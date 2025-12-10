import axios from "axios"
import MockAdapter from "axios-mock-adapter"
import { apiClient, ApiRequestError } from "../client"

describe("API Client", () => {
  let mock: MockAdapter

  beforeEach(() => {
    // Create a mock adapter for axios
    mock = new MockAdapter(apiClient.getAxiosInstance())
    
    // Clear localStorage
    localStorage.clear()
  })

  afterEach(() => {
    mock.restore()
  })

  describe("JWT Token Injection", () => {
    it("should inject JWT token in request headers", async () => {
      const token = "test-access-token"
      apiClient.setAccessToken(token)

      mock.onGet("/test").reply((config) => {
        expect(config.headers?.Authorization).toBe(`Bearer ${token}`)
        return [200, { success: true, data: { message: "success" } }]
      })

      await apiClient.get("/test")
    })

    it("should not inject token if not set", async () => {
      apiClient.setAccessToken(null)

      mock.onGet("/test").reply((config) => {
        expect(config.headers?.Authorization).toBeUndefined()
        return [200, { success: true, data: { message: "success" } }]
      })

      await apiClient.get("/test")
    })
  })

  describe("Error Handling", () => {
    it("should throw ApiRequestError on 400 error", async () => {
      mock.onGet("/test").reply(400, {
        error: "Bad Request",
        message: "Invalid parameters",
        code: "INVALID_PARAMS",
      })

      await expect(apiClient.get("/test")).rejects.toThrow(ApiRequestError)
      await expect(apiClient.get("/test")).rejects.toMatchObject({
        status: 400,
        code: "INVALID_PARAMS",
      })
    })

    it("should handle network errors", async () => {
      mock.onGet("/test").networkError()

      await expect(apiClient.get("/test")).rejects.toThrow()
    })
  })

  describe("Token Refresh on 401", () => {
    it("should refresh token and retry request on 401", async () => {
      const oldToken = "old-access-token"
      const newToken = "new-access-token"
      const refreshToken = "refresh-token"

      apiClient.setAccessToken(oldToken)
      localStorage.setItem("refresh_token", refreshToken)

      // First request fails with 401
      mock.onGet("/protected").replyOnce(401, {
        error: "Unauthorized",
        message: "Token expired",
      })

      // Refresh token endpoint
      mock.onPost("/auth/refresh").replyOnce(200, {
        success: true,
        data: {
          access_token: newToken,
          refresh_token: "new-refresh-token",
          user: { id: 1, email: "test@example.com" },
        },
      })

      // Retry original request with new token
      mock.onGet("/protected").replyOnce((config) => {
        expect(config.headers?.Authorization).toBe(`Bearer ${newToken}`)
        return [200, { success: true, data: { message: "success" } }]
      })

      const result = await apiClient.get<{ message: string }>("/protected")
      expect(result).toEqual({ message: "success" })
    })

    it("should clear tokens if refresh fails", async () => {
      const oldToken = "old-access-token"
      apiClient.setAccessToken(oldToken)
      localStorage.setItem("refresh_token", "invalid-refresh-token")

      // First request fails with 401
      mock.onGet("/protected").reply(401)

      // Refresh token fails
      mock.onPost("/auth/refresh").reply(401, {
        error: "Unauthorized",
        message: "Invalid refresh token",
      })

      await expect(apiClient.get("/protected")).rejects.toThrow()
      expect(apiClient.getAccessToken()).toBeNull()
      expect(localStorage.getItem("refresh_token")).toBeNull()
    })

    it("should queue multiple requests during token refresh", async () => {
      const oldToken = "old-access-token"
      const newToken = "new-access-token"
      
      apiClient.setAccessToken(oldToken)
      localStorage.setItem("refresh_token", "refresh-token")

      // All requests fail with 401
      mock.onGet("/protected1").replyOnce(401)
      mock.onGet("/protected2").replyOnce(401)
      
      // Refresh token endpoint (called once)
      mock.onPost("/auth/refresh").replyOnce(200, {
        success: true,
        data: {
          access_token: newToken,
          refresh_token: "new-refresh-token",
          user: { id: 1, email: "test@example.com" },
        },
      })

      // Retry with new token
      mock.onGet("/protected1").replyOnce(200, { success: true, data: { result: 1 } })
      mock.onGet("/protected2").replyOnce(200, { success: true, data: { result: 2 } })

      const [result1, result2] = await Promise.all([
        apiClient.get<{ result: number }>("/protected1"),
        apiClient.get<{ result: number }>("/protected2"),
      ])

      expect(result1).toEqual({ result: 1 })
      expect(result2).toEqual({ result: 2 })
    })
  })

  describe("Response Unwrapping", () => {
    it("should unwrap { success: true, data: T } responses", async () => {
      mock.onGet("/test").reply(200, {
        success: true,
        data: { id: 1, name: "Test" },
      })

      const result = await apiClient.get<{ id: number; name: string }>("/test")
      expect(result).toEqual({ id: 1, name: "Test" })
    })

    it("should handle paginated responses", async () => {
      mock.onGet("/events").reply(200, {
        success: true,
        data: [{ id: 1 }, { id: 2 }],
        meta: { page: 1, per_page: 10, total: 2 },
      })

      const result = await apiClient.get<{ data: Array<{ id: number }>; meta: any }>("/events")
      expect(result).toEqual({
        data: [{ id: 1 }, { id: 2 }],
        meta: { page: 1, per_page: 10, total: 2 },
      })
    })
  })

  describe("HTTP Methods", () => {
    it("should support GET requests", async () => {
      mock.onGet("/test").reply(200, { success: true, data: { message: "get" } })
      const result = await apiClient.get<{ message: string }>("/test")
      expect(result).toEqual({ message: "get" })
    })

    it("should support POST requests with body", async () => {
      mock.onPost("/test", { name: "John" }).reply(200, { success: true, data: { id: 1 } })
      const result = await apiClient.post<{ id: number }>("/test", { name: "John" })
      expect(result).toEqual({ id: 1 })
    })

    it("should support PUT requests", async () => {
      mock.onPut("/test/1", { name: "Jane" }).reply(200, { success: true, data: { updated: true } })
      const result = await apiClient.put<{ updated: boolean }>("/test/1", { name: "Jane" })
      expect(result).toEqual({ updated: true })
    })

    it("should support PATCH requests", async () => {
      mock.onPatch("/test/1", { status: "active" }).reply(200, { success: true, data: { patched: true } })
      const result = await apiClient.patch<{ patched: boolean }>("/test/1", { status: "active" })
      expect(result).toEqual({ patched: true })
    })

    it("should support DELETE requests", async () => {
      mock.onDelete("/test/1").reply(200, { success: true, data: { deleted: true } })
      const result = await apiClient.delete<{ deleted: boolean }>("/test/1")
      expect(result).toEqual({ deleted: true })
    })
  })
})
