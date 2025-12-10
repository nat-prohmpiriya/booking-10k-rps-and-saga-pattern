export interface Event {
  id: number
  title: string
  subtitle: string
  venue: string
  date: string
  fullDate: string
  time: string
  image: string
  heroImage: string
  ticketZones: TicketZone[]
}

export interface TicketZone {
  id: string
  name: string
  price: number
  available: number
  soldOut: boolean
}

export const EVENTS_DATA: Event[] = [
  {
    id: 1,
    title: "Summer Music Festival 2025",
    subtitle: "The Ultimate Summer Experience",
    venue: "Impact Arena, Bangkok",
    date: "Jun 15",
    fullDate: "Sunday, June 15, 2025",
    time: "6:00 PM - 11:00 PM",
    image: "/images/events/event-1.jpg",
    heroImage: "/images/events/event-1.jpg",
    ticketZones: [
      { id: "vip", name: "VIP Premium", price: 4999, available: 12, soldOut: false },
      { id: "platinum", name: "Platinum", price: 3499, available: 28, soldOut: false },
      { id: "gold", name: "Gold", price: 2499, available: 50, soldOut: false },
      { id: "silver", name: "Silver", price: 1499, available: 100, soldOut: false },
    ],
  },
  {
    id: 2,
    title: "Jazz Night Under The Stars",
    subtitle: "An Evening of Smooth Jazz",
    venue: "Royal Paragon Hall",
    date: "May 28",
    fullDate: "Wednesday, May 28, 2025",
    time: "7:00 PM - 10:30 PM",
    image: "/images/events/event-2.jpg",
    heroImage: "/images/events/event-2.jpg",
    ticketZones: [
      { id: "vip", name: "VIP Table", price: 3500, available: 8, soldOut: false },
      { id: "platinum", name: "Premium Seating", price: 2500, available: 20, soldOut: false },
      { id: "gold", name: "Standard Seating", price: 1800, available: 40, soldOut: false },
      { id: "silver", name: "General Admission", price: 1200, available: 80, soldOut: false },
    ],
  },
  {
    id: 3,
    title: "Electronic Dreams Festival",
    subtitle: "A Journey Into Sound",
    venue: "Bitec Bangna",
    date: "Jul 10",
    fullDate: "Thursday, July 10, 2025",
    time: "8:00 PM - 4:00 AM",
    image: "/images/events/event-3.jpg",
    heroImage: "/images/events/event-3.jpg",
    ticketZones: [
      { id: "vip", name: "VIP Backstage", price: 5500, available: 5, soldOut: false },
      { id: "platinum", name: "Platinum Zone", price: 4200, available: 15, soldOut: false },
      { id: "gold", name: "Gold Zone", price: 3200, available: 0, soldOut: true },
      { id: "silver", name: "General Entry", price: 2200, available: 200, soldOut: false },
    ],
  },
  {
    id: 4,
    title: "Classical Symphony Orchestra",
    subtitle: "Timeless Masterpieces",
    venue: "Thailand Cultural Centre",
    date: "Jun 5",
    fullDate: "Thursday, June 5, 2025",
    time: "7:30 PM - 10:00 PM",
    image: "/images/events/event-4.jpg",
    heroImage: "/images/events/event-4.jpg",
    ticketZones: [
      { id: "vip", name: "Royal Box", price: 3000, available: 4, soldOut: false },
      { id: "platinum", name: "Orchestra Front", price: 2000, available: 30, soldOut: false },
      { id: "gold", name: "Orchestra Back", price: 1500, available: 60, soldOut: false },
      { id: "silver", name: "Balcony", price: 900, available: 120, soldOut: false },
    ],
  },
  {
    id: 5,
    title: "Rock Legends Live",
    subtitle: "One Night Only",
    venue: "Thunder Dome, Bangkok",
    date: "Aug 20",
    fullDate: "Wednesday, August 20, 2025",
    time: "7:00 PM - 11:30 PM",
    image: "/images/events/event-5.jpg",
    heroImage: "/images/events/event-5.jpg",
    ticketZones: [
      { id: "vip", name: "VIP Pit", price: 5000, available: 20, soldOut: false },
      { id: "platinum", name: "Front Stage", price: 3800, available: 50, soldOut: false },
      { id: "gold", name: "Mid Arena", price: 2800, available: 0, soldOut: true },
      { id: "silver", name: "Standing Zone", price: 1800, available: 300, soldOut: false },
    ],
  },
  {
    id: 6,
    title: "Hip Hop Block Party",
    subtitle: "Street Vibes Festival",
    venue: "CentralWorld Square",
    date: "Jul 30",
    fullDate: "Wednesday, July 30, 2025",
    time: "5:00 PM - 11:00 PM",
    image: "/images/events/event-6.jpg",
    heroImage: "/images/events/event-6.jpg",
    ticketZones: [
      { id: "vip", name: "VIP Lounge", price: 3500, available: 10, soldOut: false },
      { id: "platinum", name: "Premium Area", price: 2500, available: 30, soldOut: false },
      { id: "gold", name: "Main Stage", price: 2000, available: 100, soldOut: false },
      { id: "silver", name: "General Admission", price: 1200, available: 500, soldOut: false },
    ],
  },
]

export function getEventById(id: number): Event | undefined {
  return EVENTS_DATA.find((event) => event.id === id)
}
