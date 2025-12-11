import { loadStripe, Stripe } from "@stripe/stripe-js"

// Stripe publishable key - should be from environment variable
const stripePublishableKey = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY || ""

let stripePromise: Promise<Stripe | null> | null = null

export const getStripe = (): Promise<Stripe | null> => {
  if (!stripePromise && stripePublishableKey) {
    stripePromise = loadStripe(stripePublishableKey)
  }
  return stripePromise || Promise.resolve(null)
}

export const isStripeConfigured = (): boolean => {
  return !!stripePublishableKey
}
