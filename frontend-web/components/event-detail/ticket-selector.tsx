"use client"

import { Minus, Plus } from "lucide-react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"

export type TicketZone = {
  id: string
  name: string
  price: number
  available: number
  soldOut: boolean
}

type TicketSelectorProps = {
  zones: TicketZone[]
  selectedTickets: Record<string, number>
  onTicketChange: (zoneId: string, quantity: number) => void
}

export function TicketSelector({ zones, selectedTickets, onTicketChange }: TicketSelectorProps) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-3xl font-bold mb-2">Select Tickets</h2>
        <p className="text-muted-foreground">Choose your preferred seating zone</p>
      </div>

      <div className="grid gap-4">
        {zones.map((zone) => (
          <Card
            key={zone.id}
            className={`p-6 transition-all ${
              zone.soldOut
                ? "bg-[#0f0f0f] border-[#1a1a1a] opacity-50"
                : "bg-[#0f0f0f] border-[#1a1a1a] hover:border-[#d4af37]/50"
            }`}
          >
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-2">
                  <h3 className="text-xl font-semibold">{zone.name}</h3>
                  {zone.soldOut && (
                    <Badge variant="secondary" className="bg-red-500/10 text-red-500 border-red-500/20">
                      Sold Out
                    </Badge>
                  )}
                </div>
                <div className="flex items-baseline gap-3">
                  <p className="text-2xl font-bold text-[#d4af37]">฿{zone.price.toLocaleString()}</p>
                  {!zone.soldOut && <p className="text-sm text-muted-foreground">{zone.available} tickets remaining</p>}
                </div>
              </div>

              {!zone.soldOut && (
                <div className="flex items-center gap-3">
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => onTicketChange(zone.id, Math.max(0, (selectedTickets[zone.id] || 0) - 1))}
                    disabled={(selectedTickets[zone.id] || 0) === 0}
                    className="h-10 w-10 rounded-full border-[#d4af37]/30 hover:bg-[#d4af37]/10 hover:border-[#d4af37]"
                  >
                    <Minus className="h-4 w-4 text-[#d4af37]" />
                  </Button>

                  <div className="w-12 text-center">
                    <span className="text-xl font-semibold">{selectedTickets[zone.id] || 0}</span>
                  </div>

                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => onTicketChange(zone.id, Math.min(4, (selectedTickets[zone.id] || 0) + 1))}
                    disabled={(selectedTickets[zone.id] || 0) >= 4}
                    className="h-10 w-10 rounded-full border-[#d4af37]/30 hover:bg-[#d4af37]/10 hover:border-[#d4af37]"
                  >
                    <Plus className="h-4 w-4 text-[#d4af37]" />
                  </Button>
                </div>
              )}
            </div>
          </Card>
        ))}
      </div>
    </div>
  )
}
