"use client"

import { useState, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Separator } from "@/components/ui/separator"
import { CreditCard, Smartphone, Clock, Shield, Lock, Calendar, MapPin, Ticket } from "lucide-react"
import Link from "next/link"

export default function CheckoutPage() {
  const [timeLeft, setTimeLeft] = useState(585) // 9:45 in seconds
  const [paymentMethod, setPaymentMethod] = useState("card")

  // Mock booking data
  const booking = {
    eventName: "SYMPHONY UNDER THE STARS",
    eventDate: "Saturday, March 15, 2025",
    eventTime: "7:30 PM",
    venue: "Royal Bangkok Symphony Hall",
    zone: "VIP Premium Zone A",
    quantity: 2,
    pricePerTicket: 4500,
    serviceFee: 450,
    processingFee: 225,
  }

  const subtotal = booking.pricePerTicket * booking.quantity
  const totalFees = booking.serviceFee + booking.processingFee
  const total = subtotal + totalFees

  // Countdown timer
  useEffect(() => {
    const timer = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 0) {
          clearInterval(timer)
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [])

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`
  }

  const isUrgent = timeLeft < 300 // Less than 5 minutes

  return (
    <div className="min-h-screen" style={{ backgroundColor: "#0a0a0a" }}>
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-white">Checkout</h1>
          <p className="mt-2 text-gray-400">Complete your booking securely</p>
        </div>

        {/* Two Column Layout */}
        <div className="grid gap-8 lg:grid-cols-2">
          {/* Left Column - Order Summary */}
          <div className="lg:order-1">
            <Card className="overflow-hidden border-0" style={{ backgroundColor: "#141414" }}>
              <div className="p-6">
                <h2 className="mb-6 text-xl font-semibold text-white">Order Summary</h2>

                {/* Event Image */}
                <div className="mb-6 overflow-hidden rounded-lg">
                  <img src="/images/event-checkout.jpg" alt="Event" className="h-48 w-full object-cover" />
                </div>

                {/* Event Details */}
                <div className="space-y-4">
                  <div>
                    <h3 className="text-lg font-semibold text-white">{booking.eventName}</h3>
                  </div>

                  <div className="flex items-start gap-3 text-sm text-gray-300">
                    <Calendar className="mt-0.5 h-4 w-4 shrink-0" style={{ color: "#d4af37" }} />
                    <div>
                      <div>{booking.eventDate}</div>
                      <div className="text-gray-400">{booking.eventTime}</div>
                    </div>
                  </div>

                  <div className="flex items-start gap-3 text-sm text-gray-300">
                    <MapPin className="mt-0.5 h-4 w-4 shrink-0" style={{ color: "#d4af37" }} />
                    <div>{booking.venue}</div>
                  </div>

                  <div className="flex items-start gap-3 text-sm text-gray-300">
                    <Ticket className="mt-0.5 h-4 w-4 shrink-0" style={{ color: "#d4af37" }} />
                    <div>
                      <div>{booking.zone}</div>
                      <div className="text-gray-400">Quantity: {booking.quantity}</div>
                    </div>
                  </div>
                </div>

                <Separator className="my-6 bg-gray-700" />

                {/* Price Breakdown */}
                <div className="space-y-3">
                  <div className="flex justify-between text-sm text-gray-300">
                    <span>
                      Subtotal ({booking.quantity} {booking.quantity > 1 ? "tickets" : "ticket"})
                    </span>
                    <span>฿{subtotal.toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-sm text-gray-300">
                    <span>Service Fee</span>
                    <span>฿{booking.serviceFee.toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-sm text-gray-300">
                    <span>Processing Fee</span>
                    <span>฿{booking.processingFee.toLocaleString()}</span>
                  </div>

                  <Separator className="bg-gray-700" />

                  <div className="flex justify-between text-lg font-bold text-white">
                    <span>Total</span>
                    <span style={{ color: "#d4af37" }}>฿{total.toLocaleString()}</span>
                  </div>
                </div>
              </div>
            </Card>
          </div>

          {/* Right Column - Payment Section */}
          <div className="lg:order-2">
            <Card className="border-0" style={{ backgroundColor: "#141414" }}>
              <div className="p-6">
                {/* Countdown Timer */}
                <div
                  className={`mb-6 flex items-center justify-center gap-2 rounded-lg p-4 ${
                    isUrgent ? "bg-red-950/50" : "bg-gray-800/50"
                  }`}
                >
                  <Clock className={`h-5 w-5 ${isUrgent ? "text-red-400" : "text-gray-400"}`} />
                  <span className={`font-mono text-lg font-semibold ${isUrgent ? "text-red-400" : "text-gray-300"}`}>
                    Complete in {formatTime(timeLeft)}
                  </span>
                </div>

                <h2 className="mb-6 text-xl font-semibold text-white">Payment Details</h2>

                {/* Payment Method Selector */}
                <div className="mb-6">
                  <Label className="mb-3 block text-sm font-medium text-gray-300">Payment Method</Label>
                  <RadioGroup value={paymentMethod} onValueChange={setPaymentMethod}>
                    <div
                      className="flex items-center space-x-3 rounded-lg border p-4"
                      style={{
                        borderColor: paymentMethod === "card" ? "#d4af37" : "#2a2a2a",
                        backgroundColor: paymentMethod === "card" ? "#1a1a1a" : "transparent",
                      }}
                    >
                      <RadioGroupItem value="card" id="card" />
                      <CreditCard className="h-5 w-5 text-gray-400" />
                      <Label htmlFor="card" className="flex-1 cursor-pointer text-white">
                        Credit / Debit Card
                      </Label>
                    </div>

                    <div
                      className="mt-3 flex items-center space-x-3 rounded-lg border p-4"
                      style={{
                        borderColor: paymentMethod === "promptpay" ? "#d4af37" : "#2a2a2a",
                        backgroundColor: paymentMethod === "promptpay" ? "#1a1a1a" : "transparent",
                      }}
                    >
                      <RadioGroupItem value="promptpay" id="promptpay" />
                      <Smartphone className="h-5 w-5 text-gray-400" />
                      <Label htmlFor="promptpay" className="flex-1 cursor-pointer text-white">
                        PromptPay
                      </Label>
                    </div>
                  </RadioGroup>
                </div>

                {/* Payment Form */}
                {paymentMethod === "card" && (
                  <div className="space-y-4">
                    <div>
                      <Label htmlFor="cardNumber" className="text-gray-300">
                        Card Number
                      </Label>
                      <Input
                        id="cardNumber"
                        placeholder="1234 5678 9012 3456"
                        className="mt-1.5 border-gray-700 bg-black/30 text-white placeholder:text-gray-500"
                        maxLength={19}
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label htmlFor="expiry" className="text-gray-300">
                          Expiry Date
                        </Label>
                        <Input
                          id="expiry"
                          placeholder="MM / YY"
                          className="mt-1.5 border-gray-700 bg-black/30 text-white placeholder:text-gray-500"
                          maxLength={7}
                        />
                      </div>
                      <div>
                        <Label htmlFor="cvv" className="text-gray-300">
                          CVV
                        </Label>
                        <Input
                          id="cvv"
                          placeholder="123"
                          type="password"
                          className="mt-1.5 border-gray-700 bg-black/30 text-white placeholder:text-gray-500"
                          maxLength={4}
                        />
                      </div>
                    </div>

                    <div>
                      <Label htmlFor="cardName" className="text-gray-300">
                        Cardholder Name
                      </Label>
                      <Input
                        id="cardName"
                        placeholder="JOHN DOE"
                        className="mt-1.5 border-gray-700 bg-black/30 text-white placeholder:text-gray-500"
                      />
                    </div>
                  </div>
                )}

                {paymentMethod === "promptpay" && (
                  <div className="rounded-lg bg-gray-800/50 p-6 text-center">
                    <Smartphone className="mx-auto mb-3 h-12 w-12 text-gray-400" />
                    <p className="text-sm text-gray-300">
                      You will receive a PromptPay QR code after clicking the pay button
                    </p>
                  </div>
                )}

                {/* Pay Button */}
                <Link href="/booking/confirmation">
                  <Button
                    className="mt-6 w-full py-6 text-lg font-semibold"
                    style={{
                      backgroundColor: "#d4af37",
                      color: "#0a0a0a",
                    }}
                  >
                    Pay ฿{total.toLocaleString()}
                  </Button>
                </Link>

                {/* Trust Badges */}
                <div className="mt-6 flex flex-wrap items-center justify-center gap-4 border-t border-gray-700 pt-6">
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <Shield className="h-4 w-4" style={{ color: "#d4af37" }} />
                    <span>SSL Encrypted</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <Lock className="h-4 w-4" style={{ color: "#d4af37" }} />
                    <span>Secure Payment</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <CreditCard className="h-4 w-4" style={{ color: "#d4af37" }} />
                    <span>PCI Compliant</span>
                  </div>
                </div>

                <p className="mt-4 text-center text-xs text-gray-500">
                  Your payment information is processed securely. We do not store credit card details.
                </p>
              </div>
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}
