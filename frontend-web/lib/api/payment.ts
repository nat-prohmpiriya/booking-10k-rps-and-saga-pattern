import { apiClient } from "./client"

export interface PortalSessionResponse {
  url: string
}

export const paymentApi = {
  /**
   * Create a Stripe Customer Portal session
   * Returns a URL to redirect the user to for managing payment methods
   */
  createPortalSession: async (returnUrl: string): Promise<PortalSessionResponse> => {
    return apiClient.post<PortalSessionResponse>("/payments/portal", {
      return_url: returnUrl,
    })
  },
}
