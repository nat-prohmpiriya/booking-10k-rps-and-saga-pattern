"use client"

import { useEffect, useState } from "react"
import { Check, Download, Wallet, Calendar, MapPin, Ticket } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import Link from "next/link"

export default function BookingConfirmationPage() {
  const [showSuccess, setShowSuccess] = useState(false)

  useEffect(() => {
    // Trigger success animation on mount
    setShowSuccess(true)
  }, [])

  // Sample booking data
  const booking = {
    reference: "BK-2025-7X9M4",
    eventName: "Exclusive Gala Night 2025",
    date: "Saturday, March 15, 2025",
    time: "7:30 PM",
    venue: "Grand Ballroom",
    address: "Crystal Palace Hotel, Bangkok",
    zone: "VIP Premium Zone A",
    seat: "Row 5, Seat 12-13",
  }

  return (
    <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center p-4">
      <div className="max-w-2xl w-full space-y-8 py-12">
        {/* Success Animation */}
        <div className="flex flex-col items-center space-y-6">
          <div
            className={`relative transition-all duration-700 ${
              showSuccess ? "scale-100 opacity-100" : "scale-50 opacity-0"
            }`}
          >
            <div className="w-24 h-24 rounded-full bg-[#d4af37]/10 flex items-center justify-center relative">
              {/* Gold glow effect */}
              <div className="absolute inset-0 rounded-full bg-[#d4af37] opacity-20 blur-xl animate-pulse" />
              <div className="relative w-16 h-16 rounded-full bg-[#d4af37] flex items-center justify-center">
                <Check className="w-10 h-10 text-[#0a0a0a] stroke-[3]" />
              </div>
            </div>
          </div>

          <div className="text-center space-y-2">
            <h1 className="text-4xl md:text-5xl font-bold text-balance">Booking Confirmed!</h1>
            <p className="text-lg text-zinc-400">{"Your tickets have been sent to your email"}</p>
          </div>
        </div>

        {/* E-Ticket Card */}
        <Card className="bg-zinc-900/50 border-zinc-800 backdrop-blur-sm overflow-hidden">
          <div className="p-6 md:p-8 space-y-6">
            {/* QR Code Section */}
            <div className="flex flex-col md:flex-row gap-6 items-start md:items-center">
              <div className="shrink-0">
                <div className="w-32 h-32 bg-white rounded-lg p-2 shadow-lg shadow-[#d4af37]/20">
                  {/* QR Code placeholder */}
                  <svg viewBox="0 0 100 100" className="w-full h-full" xmlns="http://www.w3.org/2000/svg">
                    <rect width="100" height="100" fill="white" />
                    <g fill="black">
                      {/* QR code pattern simulation */}
                      <rect x="10" y="10" width="30" height="30" />
                      <rect x="60" y="10" width="30" height="30" />
                      <rect x="10" y="60" width="30" height="30" />
                      <rect x="15" y="15" width="20" height="20" fill="white" />
                      <rect x="65" y="15" width="20" height="20" fill="white" />
                      <rect x="15" y="65" width="20" height="20" fill="white" />
                      <rect x="20" y="20" width="10" height="10" />
                      <rect x="70" y="20" width="10" height="10" />
                      <rect x="20" y="70" width="10" height="10" />
                      {/* Random pattern squares */}
                      <rect x="50" y="20" width="5" height="5" />
                      <rect x="45" y="25" width="5" height="5" />
                      <rect x="55" y="30" width="5" height="5" />
                      <rect x="50" y="35" width="5" height="5" />
                      <rect x="60" y="45" width="5" height="5" />
                      <rect x="65" y="50" width="5" height="5" />
                      <rect x="70" y="55" width="5" height="5" />
                      <rect x="50" y="60" width="5" height="5" />
                      <rect x="55" y="65" width="5" height="5" />
                      <rect x="45" y="70" width="5" height="5" />
                      <rect x="25" y="50" width="5" height="5" />
                      <rect x="30" y="45" width="5" height="5" />
                      <rect x="20" y="55" width="5" height="5" />
                    </g>
                  </svg>
                </div>
              </div>

              <div className="flex-1 space-y-4">
                {/* Booking Reference */}
                <div>
                  <p className="text-sm text-zinc-500 uppercase tracking-wider">Booking Reference</p>
                  <p className="text-2xl font-bold text-[#d4af37] font-mono tracking-wide">{booking.reference}</p>
                </div>

                {/* Event Name */}
                <div>
                  <h2 className="text-2xl font-semibold text-balance">{booking.eventName}</h2>
                </div>
              </div>
            </div>

            {/* Divider */}
            <div className="border-t border-zinc-800" />

            {/* Event Details */}
            <div className="grid md:grid-cols-2 gap-6">
              {/* Date & Time */}
              <div className="flex gap-3">
                <div className="shrink-0">
                  <div className="w-10 h-10 rounded-lg bg-[#d4af37]/10 flex items-center justify-center">
                    <Calendar className="w-5 h-5 text-[#d4af37]" />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Date & Time</p>
                  <p className="font-medium">{booking.date}</p>
                  <p className="text-sm text-zinc-400">{booking.time}</p>
                </div>
              </div>

              {/* Venue */}
              <div className="flex gap-3">
                <div className="shrink-0">
                  <div className="w-10 h-10 rounded-lg bg-[#d4af37]/10 flex items-center justify-center">
                    <MapPin className="w-5 h-5 text-[#d4af37]" />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Venue</p>
                  <p className="font-medium">{booking.venue}</p>
                  <p className="text-sm text-zinc-400">{booking.address}</p>
                </div>
              </div>

              {/* Zone */}
              <div className="flex gap-3">
                <div className="shrink-0">
                  <div className="w-10 h-10 rounded-lg bg-[#d4af37]/10 flex items-center justify-center">
                    <Ticket className="w-5 h-5 text-[#d4af37]" />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Zone</p>
                  <p className="font-medium">{booking.zone}</p>
                </div>
              </div>

              {/* Seat */}
              <div className="flex gap-3">
                <div className="shrink-0">
                  <div className="w-10 h-10 rounded-lg bg-[#d4af37]/10 flex items-center justify-center">
                    <span className="text-[#d4af37] font-bold text-lg">#</span>
                  </div>
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Seat</p>
                  <p className="font-medium">{booking.seat}</p>
                </div>
              </div>
            </div>

            {/* Divider */}
            <div className="border-t border-zinc-800" />

            {/* Barcode */}
            <div className="space-y-2">
              <p className="text-sm text-zinc-500 text-center">Scan at entry</p>
              <div className="flex justify-center">
                <svg
                  width="280"
                  height="60"
                  viewBox="0 0 280 60"
                  className="w-full max-w-sm"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <rect width="280" height="60" fill="white" rx="4" />
                  {/* Barcode pattern */}
                  <g fill="black">
                    <rect x="10" y="10" width="3" height="40" />
                    <rect x="16" y="10" width="2" height="40" />
                    <rect x="22" y="10" width="4" height="40" />
                    <rect x="29" y="10" width="2" height="40" />
                    <rect x="34" y="10" width="5" height="40" />
                    <rect x="42" y="10" width="2" height="40" />
                    <rect x="47" y="10" width="3" height="40" />
                    <rect x="53" y="10" width="2" height="40" />
                    <rect x="58" y="10" width="4" height="40" />
                    <rect x="65" y="10" width="3" height="40" />
                    <rect x="71" y="10" width="2" height="40" />
                    <rect x="76" y="10" width="5" height="40" />
                    <rect x="84" y="10" width="2" height="40" />
                    <rect x="89" y="10" width="3" height="40" />
                    <rect x="95" y="10" width="4" height="40" />
                    <rect x="102" y="10" width="2" height="40" />
                    <rect x="107" y="10" width="3" height="40" />
                    <rect x="113" y="10" width="5" height="40" />
                    <rect x="121" y="10" width="2" height="40" />
                    <rect x="126" y="10" width="4" height="40" />
                    <rect x="133" y="10" width="2" height="40" />
                    <rect x="138" y="10" width="3" height="40" />
                    <rect x="144" y="10" width="2" height="40" />
                    <rect x="149" y="10" width="5" height="40" />
                    <rect x="157" y="10" width="3" height="40" />
                    <rect x="163" y="10" width="2" height="40" />
                    <rect x="168" y="10" width="4" height="40" />
                    <rect x="175" y="10" width="2" height="40" />
                    <rect x="180" y="10" width="5" height="40" />
                    <rect x="188" y="10" width="2" height="40" />
                    <rect x="193" y="10" width="3" height="40" />
                    <rect x="199" y="10" width="4" height="40" />
                    <rect x="206" y="10" width="2" height="40" />
                    <rect x="211" y="10" width="3" height="40" />
                    <rect x="217" y="10" width="2" height="40" />
                    <rect x="222" y="10" width="5" height="40" />
                    <rect x="230" y="10" width="3" height="40" />
                    <rect x="236" y="10" width="2" height="40" />
                    <rect x="241" y="10" width="4" height="40" />
                    <rect x="248" y="10" width="2" height="40" />
                    <rect x="253" y="10" width="5" height="40" />
                    <rect x="261" y="10" width="3" height="40" />
                    <rect x="267" y="10" width="2" height="40" />
                  </g>
                </svg>
              </div>
              <p className="text-xs text-zinc-500 text-center font-mono">{booking.reference}</p>
            </div>
          </div>
        </Card>

        {/* Action Buttons */}
        <div className="flex flex-col sm:flex-row gap-4">
          <Button className="flex-1 h-12 bg-[#d4af37] hover:bg-[#c19d2f] text-[#0a0a0a] font-semibold" size="lg">
            <Download className="w-5 h-5 mr-2" />
            Download E-Ticket
          </Button>
          <Button
            variant="outline"
            className="flex-1 h-12 border-zinc-700 hover:bg-zinc-800 hover:text-white bg-transparent"
            size="lg"
          >
            <Wallet className="w-5 h-5 mr-2" />
            Add to Wallet
          </Button>
        </div>

        {/* View My Bookings Link */}
        <div className="text-center">
          <Link
            href="/"
            className="text-[#d4af37] hover:text-[#c19d2f] font-medium inline-flex items-center gap-2 transition-colors"
          >
            Back to Home
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </Link>
        </div>
      </div>
    </div>
  )
}
