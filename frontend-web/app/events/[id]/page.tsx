"use client"

import { useState } from "react"
import { useParams, notFound } from "next/navigation"
import { EventHero } from "@/components/event-detail/event-hero"
import { EventInfo } from "@/components/event-detail/event-info"
import { TicketSelector } from "@/components/event-detail/ticket-selector"
import { StickyCheckout } from "@/components/event-detail/sticky-checkout"
import { CountdownTimer } from "@/components/event-detail/countdown-timer"
import { Header } from "@/components/header"
import { getEventById } from "@/lib/events-data"

export default function EventDetailPage() {
  const params = useParams()
  const eventId = Number(params.id)
  const event = getEventById(eventId)

  const [selectedTickets, setSelectedTickets] = useState<Record<string, number>>({})

  if (!event) {
    notFound()
  }

  const handleTicketChange = (zoneId: string, quantity: number) => {
    setSelectedTickets((prev) => {
      if (quantity === 0) {
        const { [zoneId]: _, ...rest } = prev
        return rest
      }
      return { ...prev, [zoneId]: quantity }
    })
  }

  const getTotalPrice = () => {
    return Object.entries(selectedTickets).reduce((total, [zoneId, quantity]) => {
      const zone = event.ticketZones.find((z) => z.id === zoneId)
      return total + (zone?.price || 0) * quantity
    }, 0)
  }

  const getTotalTickets = () => {
    return Object.values(selectedTickets).reduce((sum, qty) => sum + qty, 0)
  }

  return (
    <div className="min-h-screen bg-[#0a0a0a] text-white">
      <Header />
      <EventHero image={event.heroImage} />

      <div className="container mx-auto px-4 pb-32">
        <div className="relative -mt-32 z-10">
          <div className="max-w-4xl mx-auto space-y-8">
            <div>
              <h1 className="text-5xl md:text-6xl font-bold mb-4 text-balance">{event.title}</h1>
              <p className="text-xl text-[#d4af37] text-pretty">{event.subtitle}</p>
            </div>

            <CountdownTimer />

            <EventInfo
              date={event.fullDate.split(",")[0] + "," + event.fullDate.split(",")[1]}
              year="2025"
              time={event.time.split(" - ")[0]}
              doorsOpen={(() => {
                const startTime = event.time.split(" - ")[0]
                const [time, period] = startTime.split(" ")
                const [hours, minutes] = time.split(":")
                const hour = parseInt(hours)
                const newHour = hour === 1 ? 12 : hour - 1
                return `${newHour}:${minutes} ${period}`
              })()}
              venue={event.venue.split(",")[0]}
              location={event.venue.includes(",") ? event.venue.split(",")[1].trim() : event.venue}
            />

            <TicketSelector zones={event.ticketZones} selectedTickets={selectedTickets} onTicketChange={handleTicketChange} />
          </div>
        </div>
      </div>

      <StickyCheckout totalPrice={getTotalPrice()} totalTickets={getTotalTickets()} />
    </div>
  )
}
