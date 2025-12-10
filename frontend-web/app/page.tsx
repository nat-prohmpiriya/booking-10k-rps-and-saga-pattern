"use client"

import { Header } from "@/components/header"
import { Hero } from "@/components/hero"
import { EventCard } from "@/components/event-card"
import { useEvents } from "@/hooks/use-events"
import { Skeleton } from "@/components/ui/skeleton"

function EventCardSkeleton() {
  return (
    <div className="rounded-lg border border-border/50 overflow-hidden">
      <Skeleton className="h-48 lg:h-56 w-full" />
      <div className="p-5 space-y-4">
        <div className="space-y-2">
          <Skeleton className="h-6 w-3/4" />
          <Skeleton className="h-4 w-1/2" />
        </div>
        <div className="flex items-center justify-between pt-2 border-t border-border/50">
          <div className="space-y-1">
            <Skeleton className="h-3 w-12" />
            <Skeleton className="h-8 w-20" />
          </div>
          <Skeleton className="h-10 w-24" />
        </div>
      </div>
    </div>
  )
}

export default function Home() {
  const { events, isLoading } = useEvents()

  return (
    <main className="min-h-screen">
      <Header />
      <Hero />

      {/* Featured Events Section */}
      <section className="container mx-auto px-4 lg:px-8 py-16 lg:py-24">
        <div className="space-y-4 mb-12">
          <div className="inline-block glass px-4 py-2 rounded-full">
            <span className="text-primary text-sm font-medium">Featured Events</span>
          </div>
          <h2 className="text-3xl lg:text-5xl font-bold text-balance">Upcoming Events</h2>
          <p className="text-lg text-muted-foreground max-w-2xl text-pretty">
            Discover the hottest events happening now. Book your tickets before they sell out.
          </p>
        </div>

        {/* Event Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 lg:gap-8">
          {isLoading ? (
            <>
              <EventCardSkeleton />
              <EventCardSkeleton />
              <EventCardSkeleton />
              <EventCardSkeleton />
              <EventCardSkeleton />
              <EventCardSkeleton />
            </>
          ) : (
            events.map((event) => (
              <EventCard
                key={event.id}
                id={event.id}
                title={event.title}
                venue={event.venue}
                date={event.date}
                price={event.price}
                image={event.image}
              />
            ))
          )}
        </div>
      </section>
    </main>
  )
}
