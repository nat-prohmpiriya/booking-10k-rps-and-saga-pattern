import { apiClient } from "./client"
import type {
  ReserveSeatsRequest,
  ReservationResponse,
  BookingResponse,
  CreatePaymentRequest,
  PaymentResponse,
} from "./types"

export const bookingApi = {
  async reserveSeats(data: ReserveSeatsRequest): Promise<ReservationResponse> {
    return apiClient.post<ReservationResponse>("/bookings/reserve", data, { requireAuth: true })
  },

  async confirmBooking(bookingId: string): Promise<BookingResponse> {
    return apiClient.post<BookingResponse>(`/bookings/${bookingId}/confirm`, {}, { requireAuth: true })
  },

  async getBooking(bookingId: string): Promise<BookingResponse> {
    return apiClient.get<BookingResponse>(`/bookings/${bookingId}`, { requireAuth: true })
  },

  async listUserBookings(): Promise<BookingResponse[]> {
    return apiClient.get<BookingResponse[]>("/bookings", { requireAuth: true })
  },
}

export const paymentApi = {
  async createPayment(data: CreatePaymentRequest): Promise<PaymentResponse> {
    return apiClient.post<PaymentResponse>("/payments", data, { requireAuth: true })
  },

  async getPayment(paymentId: string): Promise<PaymentResponse> {
    return apiClient.get<PaymentResponse>(`/payments/${paymentId}`, { requireAuth: true })
  },
}
