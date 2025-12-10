"use client"

import { Button } from "@/components/ui/button"
import { ShoppingCart } from "lucide-react"
import Link from "next/link"

type StickyCheckoutProps = {
  totalPrice: number
  totalTickets: number
}

export function StickyCheckout({ totalPrice, totalTickets }: StickyCheckoutProps) {
  return (
    <div className="fixed bottom-0 left-0 right-0 border-t border-[#1a1a1a] bg-[#0a0a0a]/95 backdrop-blur-lg z-50">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between max-w-4xl mx-auto">
          <div className="flex items-center gap-6">
            <div>
              <p className="text-sm text-muted-foreground">Total</p>
              <p className="text-3xl font-bold text-[#d4af37]">฿{totalPrice.toLocaleString()}</p>
            </div>
            {totalTickets > 0 && (
              <div className="hidden md:block text-sm text-muted-foreground">
                {totalTickets} {totalTickets === 1 ? "ticket" : "tickets"} selected
              </div>
            )}
          </div>

          <Link href={totalTickets > 0 ? "/queue" : "#"}>
            <Button
              size="lg"
              disabled={totalTickets === 0}
              className="bg-[#d4af37] hover:bg-[#d4af37]/90 text-black font-semibold px-8 h-12"
            >
              <ShoppingCart className="w-5 h-5 mr-2" />
              Reserve Now
            </Button>
          </Link>
        </div>
      </div>
    </div>
  )
}
