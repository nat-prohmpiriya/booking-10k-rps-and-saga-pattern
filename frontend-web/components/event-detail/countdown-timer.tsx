"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { Clock } from "lucide-react"

interface CountdownTimerProps {
  targetDate?: Date
}

export function CountdownTimer({ targetDate }: CountdownTimerProps) {
  const [timeLeft, setTimeLeft] = useState({
    days: 0,
    hours: 0,
    minutes: 0,
    seconds: 0,
  })

  useEffect(() => {
    // Set sale opening date to 7 days from now if not provided
    const saleDate = targetDate || new Date(Date.now() + 7 * 24 * 60 * 60 * 1000)

    const timer = setInterval(() => {
      const now = new Date().getTime()
      const distance = saleDate.getTime() - now

      if (distance < 0) {
        clearInterval(timer)
        return
      }

      setTimeLeft({
        days: Math.floor(distance / (1000 * 60 * 60 * 24)),
        hours: Math.floor((distance % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60)),
        minutes: Math.floor((distance % (1000 * 60 * 60)) / (1000 * 60)),
        seconds: Math.floor((distance % (1000 * 60)) / 1000),
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [targetDate])

  return (
    <Card className="bg-[#d4af37]/10 border-[#d4af37]/30 p-6">
      <div className="flex items-center gap-4 mb-4">
        <Clock className="w-5 h-5 text-[#d4af37]" />
        <h3 className="text-lg font-semibold text-[#d4af37]">Sale Opens In</h3>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="text-center">
          <div className="text-3xl md:text-4xl font-bold text-white mb-1">{timeLeft.days}</div>
          <div className="text-sm text-muted-foreground">Days</div>
        </div>
        <div className="text-center">
          <div className="text-3xl md:text-4xl font-bold text-white mb-1">{timeLeft.hours}</div>
          <div className="text-sm text-muted-foreground">Hours</div>
        </div>
        <div className="text-center">
          <div className="text-3xl md:text-4xl font-bold text-white mb-1">{timeLeft.minutes}</div>
          <div className="text-sm text-muted-foreground">Minutes</div>
        </div>
        <div className="text-center">
          <div className="text-3xl md:text-4xl font-bold text-white mb-1">{timeLeft.seconds}</div>
          <div className="text-sm text-muted-foreground">Seconds</div>
        </div>
      </div>
    </Card>
  )
}
