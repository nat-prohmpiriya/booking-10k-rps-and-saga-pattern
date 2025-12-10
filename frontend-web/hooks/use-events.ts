"use client"

import { useState, useEffect, useCallback } from "react"
import { eventsApi } from "@/lib/api/events"
import type { EventResponse, EventListFilter } from "@/lib/api/types"
import { EVENTS_DATA, type Event as MockEvent } from "@/lib/events-data"

interface UseEventsReturn {
  events: EventDisplay[]
  isLoading: boolean
  error: string | null
  refetch: () => Promise<void>
  total: number
}

export interface EventDisplay {
  id: string | number
  title: string
  subtitle?: string
  venue: string
  date: string
  fullDate?: string
  time?: string
  image: string
  heroImage?: string
  price: number
  status?: string
}

function mapApiEventToDisplay(event: EventResponse): EventDisplay {
  const startDate = new Date(event.start_time)
  return {
    id: event.id,
    title: event.name,
    venue: event.venue_id,
    date: startDate.toLocaleDateString("en-US", { month: "short", day: "numeric" }),
    fullDate: startDate.toLocaleDateString("en-US", { weekday: "long", year: "numeric", month: "long", day: "numeric" }),
    time: `${startDate.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" })} - ${new Date(event.end_time).toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" })}`,
    image: "/images/events/event-1.jpg",
    heroImage: "/images/events/event-1.jpg",
    price: 0,
    status: event.status,
  }
}

function mapMockEventToDisplay(event: MockEvent): EventDisplay {
  return {
    id: event.id,
    title: event.title,
    subtitle: event.subtitle,
    venue: event.venue,
    date: event.date,
    fullDate: event.fullDate,
    time: event.time,
    image: event.image,
    heroImage: event.heroImage,
    price: Math.min(...event.ticketZones.map((z) => z.price)),
  }
}

export function useEvents(filter?: EventListFilter): UseEventsReturn {
  const [events, setEvents] = useState<EventDisplay[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [total, setTotal] = useState(0)

  const fetchEvents = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await eventsApi.list(filter)
      const mappedEvents = response.events.map(mapApiEventToDisplay)
      setEvents(mappedEvents)
      setTotal(response.total)
    } catch (err) {
      console.warn("Failed to fetch events from API, using mock data:", err)
      const mockEvents = EVENTS_DATA.map(mapMockEventToDisplay)
      setEvents(mockEvents)
      setTotal(mockEvents.length)
      setError(null)
    } finally {
      setIsLoading(false)
    }
  }, [filter])

  useEffect(() => {
    fetchEvents()
  }, [fetchEvents])

  return {
    events,
    isLoading,
    error,
    refetch: fetchEvents,
    total,
  }
}

interface UseEventReturn {
  event: EventDisplay | null
  isLoading: boolean
  error: string | null
}

export function useEvent(id: string | number): UseEventReturn {
  const [event, setEvent] = useState<EventDisplay | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchEvent() {
      setIsLoading(true)
      setError(null)
      try {
        const response = await eventsApi.getById(String(id))
        setEvent(mapApiEventToDisplay(response))
      } catch (err) {
        console.warn("Failed to fetch event from API, using mock data:", err)
        const mockEvent = EVENTS_DATA.find((e) => e.id === Number(id))
        if (mockEvent) {
          setEvent(mapMockEventToDisplay(mockEvent))
        } else {
          setError("Event not found")
        }
      } finally {
        setIsLoading(false)
      }
    }

    fetchEvent()
  }, [id])

  return { event, isLoading, error }
}
