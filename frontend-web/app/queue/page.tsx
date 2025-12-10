"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { useRouter } from "next/navigation"

export default function QueueWaitingRoom() {
  const [position, setPosition] = useState(1247)
  const [dots, setDots] = useState("")
  const router = useRouter()

  // Simulate queue position updates
  useEffect(() => {
    const interval = setInterval(() => {
      setPosition((prev) => {
        const newPosition = Math.max(1, prev - Math.floor(Math.random() * 50 + 10))
        // Redirect to checkout when position reaches 1
        if (newPosition <= 1) {
          router.push("/checkout")
        }
        return newPosition
      })
    }, 2000)
    return () => clearInterval(interval)
  }, [router])

  // Animate loading dots
  useEffect(() => {
    const interval = setInterval(() => {
      setDots((prev) => (prev.length >= 3 ? "" : prev + "."))
    }, 500)
    return () => clearInterval(interval)
  }, [])

  const estimatedWait = Math.ceil(position / 250)

  return (
    <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center p-4">
      <div className="w-full max-w-2xl space-y-8">
        {/* Main Content */}
        <div className="text-center space-y-8">
          {/* Header */}
          <div className="space-y-4">
            <div className="inline-block">
              <div className="w-16 h-16 rounded-full border-2 border-[#d4af37] flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-[#d4af37]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </div>
            </div>

            <h1 className="text-4xl md:text-5xl font-bold text-balance">You're in the Queue</h1>

            <p className="text-gray-400 text-lg">Thank you for your patience</p>
          </div>

          {/* Position Display */}
          <div className="space-y-2">
            <p className="text-sm uppercase tracking-wider text-[#d4af37] font-medium">Your Position</p>
            <div className="relative">
              <div className="text-7xl md:text-8xl font-bold text-[#d4af37] animate-pulse">
                #{position.toLocaleString()}
              </div>
              {/* Glow effect */}
              <div className="absolute inset-0 blur-3xl opacity-30 bg-[#d4af37] -z-10" />
            </div>
          </div>

          {/* Estimated Wait Time */}
          <div className="space-y-2">
            <p className="text-sm uppercase tracking-wider text-gray-400 font-medium">Estimated Wait Time</p>
            <p className="text-3xl font-semibold text-white">
              ~{estimatedWait} {estimatedWait === 1 ? "minute" : "minutes"}
            </p>
          </div>

          {/* Animated Progress Indicator */}
          <div className="py-8">
            <div className="w-full h-1 bg-gray-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-linear-to-r from-[#d4af37] to-[#f4d03f]"
                style={{
                  width: "100%",
                  animation: "shimmer 2s ease-in-out infinite",
                }}
              />
            </div>
          </div>

          {/* Keep Page Open Notice */}
          <Card className="bg-[#1a1a1a] border-gray-800 p-6">
            <div className="flex items-start gap-4">
              <div className="shrink-0">
                <svg className="w-6 h-6 text-[#d4af37]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </div>
              <div className="text-left space-y-1">
                <p className="font-semibold text-white">Please keep this page open</p>
                <p className="text-sm text-gray-400">
                  Closing this page will lose your spot in the queue. You'll be automatically redirected when it's your
                  turn.
                </p>
              </div>
            </div>
          </Card>
        </div>

        {/* Event Info Card */}
        <Card className="bg-linear-to-br from-[#1a1a1a] to-[#0f0f0f] border-[#d4af37]/20 p-6 md:p-8">
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-[#d4af37] animate-pulse" />
              <p className="text-xs uppercase tracking-wider text-[#d4af37] font-medium">Waiting For</p>
            </div>

            <div>
              <h2 className="text-2xl font-bold mb-2">Exclusive Gala Night 2025</h2>
              <p className="text-gray-400 leading-relaxed">
                You're in line for early access to our exclusive event. Limited tickets available for this premium
                experience.
              </p>
            </div>

            <div className="grid grid-cols-2 gap-4 pt-4 border-t border-gray-800">
              <div>
                <p className="text-xs text-gray-500 uppercase tracking-wider mb-1">Event Date</p>
                <p className="font-medium">March 15, 2025</p>
              </div>
              <div>
                <p className="text-xs text-gray-500 uppercase tracking-wider mb-1">Access Type</p>
                <p className="font-medium text-[#d4af37]">VIP Early Access</p>
              </div>
            </div>
          </div>
        </Card>

        {/* Auto-refresh Indicator */}
        <div className="flex items-center justify-center gap-2 text-sm text-gray-500">
          <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
          <span>Auto-updating{dots}</span>
        </div>
      </div>

      {/* Custom animation styles */}
      <style jsx>{`
        @keyframes shimmer {
          0%, 100% {
            transform: translateX(-100%);
          }
          50% {
            transform: translateX(100%);
          }
        }
      `}</style>
    </div>
  )
}
