// API Response types
export interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
}

export interface ApiError {
  error: string
  code?: string
  message?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  page: number
  page_size: number
  total_items: number
  total_pages: number
}

// Auth types
export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  name: string
}

export interface RefreshTokenRequest {
  refresh_token: string
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: UserResponse
}

export interface UserResponse {
  id: string
  email: string
  name: string
  role: string
  created_at: string
}

// Event types
export interface EventResponse {
  id: string
  name: string
  slug: string
  description: string
  venue_id: string
  start_time: string
  end_time: string
  status: string
  tenant_id: string
  created_at: string
  updated_at: string
}

export interface EventListResponse {
  events: EventResponse[]
  total: number
  limit: number
  offset: number
}

export interface EventListFilter {
  status?: string
  venue_id?: string
  search?: string
  limit?: number
  offset?: number
}

// Show types
export interface ShowResponse {
  id: string
  event_id: string
  name: string
  start_time: string
  end_time: string
  status: string
  created_at: string
  updated_at: string
}

export interface ShowListResponse {
  shows: ShowResponse[]
  total: number
  limit: number
  offset: number
}

// Zone types
export interface ShowZoneResponse {
  id: string
  show_id: string
  name: string
  price: number
  total_seats: number
  available_seats: number
  description: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface ShowZoneListResponse {
  zones: ShowZoneResponse[]
  total: number
  limit: number
  offset: number
}

// Booking types
export interface ReserveSeatsRequest {
  show_id: string
  zone_id: string
  quantity: number
}

export interface ReservationResponse {
  id: string
  user_id: string
  show_id: string
  zone_id: string
  quantity: number
  status: string
  expires_at: string
  created_at: string
}

export interface BookingResponse {
  id: string
  user_id: string
  reservation_id: string
  status: string
  total_amount: number
  created_at: string
  updated_at: string
}

// Payment types
export interface CreatePaymentRequest {
  booking_id: string
  payment_method: string
  amount: number
}

export interface PaymentResponse {
  id: string
  booking_id: string
  amount: number
  status: string
  payment_method: string
  created_at: string
}
